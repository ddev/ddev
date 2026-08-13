package exec_test

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunHostCommandWithOptions(t *testing.T) {
	bashPath := util.FindBashPath()

	// Test basic command execution without options
	t.Run("no options", func(t *testing.T) {
		output, err := exec.RunHostCommandWithOptions(bashPath, []exec.CmdOption{}, "-c", "echo hello")
		require.NoError(t, err)
		assert.Equal(t, "hello", strings.TrimSpace(output))
	})

	// Test WithStdin option
	t.Run("with stdin", func(t *testing.T) {
		stdin := strings.NewReader("test input")

		output, err := exec.RunHostCommandWithOptions(bashPath, []exec.CmdOption{
			exec.WithStdin(stdin),
		}, "-c", "cat")
		require.NoError(t, err)
		assert.Equal(t, "test input", strings.TrimSpace(output))
	})

	// Test WithEnv option
	t.Run("with env", func(t *testing.T) {
		output, err := exec.RunHostCommandWithOptions(bashPath, []exec.CmdOption{
			exec.WithEnv([]string{"TEST_VAR=test_value"}),
		}, "-c", "echo $TEST_VAR")
		require.NoError(t, err)
		assert.Equal(t, "test_value", strings.TrimSpace(output))
	})

	// Test multiple options combined
	t.Run("with stdin and env", func(t *testing.T) {
		stdin := strings.NewReader("hello world")
		output, err := exec.RunHostCommandWithOptions(bashPath, []exec.CmdOption{
			exec.WithStdin(stdin),
			exec.WithEnv(append(os.Environ(), "TEST_PREFIX=prefix:")),
		}, "-c", "echo $TEST_PREFIX$(cat)")
		require.NoError(t, err)
		assert.Equal(t, "prefix:hello world", strings.TrimSpace(output))
	})

	// Test empty options slice
	t.Run("empty options", func(t *testing.T) {
		output, err := exec.RunHostCommandWithOptions(bashPath, []exec.CmdOption{}, "-c", "echo test")
		require.NoError(t, err)
		assert.Equal(t, "test", strings.TrimSpace(output))
	})

	// Test nil options slice
	t.Run("nil options", func(t *testing.T) {
		output, err := exec.RunHostCommandWithOptions(bashPath, nil, "-c", "echo test")
		require.NoError(t, err)
		assert.Equal(t, "test", strings.TrimSpace(output))
	})

	// Test command with error
	t.Run("command error", func(t *testing.T) {
		_, err := exec.RunHostCommandWithOptions("nonexistent-ddev-command", []exec.CmdOption{})
		require.Error(t, err, "expected error for nonexistent-ddev-command")
	})

	// Test WithEnv with multiple environment variables
	t.Run("with multiple env vars", func(t *testing.T) {
		output, err := exec.RunHostCommandWithOptions(bashPath, []exec.CmdOption{
			exec.WithEnv([]string{"VAR1=hello", "VAR2=world"}),
		}, "-c", "echo $VAR1-$VAR2")
		require.NoError(t, err)
		assert.Equal(t, "hello-world", strings.TrimSpace(output))
	})

	// Test WithEnv overriding system environment
	t.Run("with env override", func(t *testing.T) {
		// Set a system env var
		t.Setenv("TEST_OVERRIDE", "system_value")

		output, err := exec.RunHostCommandWithOptions(bashPath, []exec.CmdOption{
			exec.WithEnv([]string{"TEST_OVERRIDE=custom_value"}),
		}, "-c", "echo $TEST_OVERRIDE")
		require.NoError(t, err)
		assert.Equal(t, "custom_value", strings.TrimSpace(output))
	})

	// Test WithStdin with empty reader
	t.Run("with empty stdin", func(t *testing.T) {
		stdin := strings.NewReader("")

		output, err := exec.RunHostCommandWithOptions(bashPath, []exec.CmdOption{
			exec.WithStdin(stdin),
		}, "-c", "cat")
		require.NoError(t, err)
		assert.Equal(t, "", output)
	})
}

// captureStdErr is CaptureStdOut's os.Stderr counterpart; pkg/util has no
// public equivalent since nothing else needs one.
func captureStdErr() func() string {
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	return func() string {
		outC := make(chan string)
		go func() {
			var buf bytes.Buffer
			_, err := io.Copy(&buf, r)
			util.CheckErr(err)
			outC <- buf.String()
		}()
		util.CheckClose(w)
		os.Stderr = old
		return <-outC
	}
}

// TestRunInteractiveCommandWithCapture verifies that the command's output both
// reaches the terminal and comes back to the caller, unmodified.
func TestRunInteractiveCommandWithCapture(t *testing.T) {
	bashPath := util.FindBashPath()

	t.Run("captures stdout and stderr, keeping them separate on the terminal", func(t *testing.T) {
		restoreStdout := util.CaptureStdOut()
		restoreStderr := captureStdErr()
		captured, err := exec.RunInteractiveCommandWithCapture(bashPath, []string{"-c", `echo to-stdout; echo to-stderr >&2`})
		printedStdout := restoreStdout()
		printedStderr := restoreStderr()
		require.NoError(t, err)
		require.Contains(t, captured, "to-stdout")
		require.Contains(t, captured, "to-stderr")
		// The user still sees it happen, on the same stream as always;
		// capturing is not a substitute for that, and must not merge streams.
		require.Contains(t, printedStdout, "to-stdout")
		require.NotContains(t, printedStdout, "to-stderr")
		require.Contains(t, printedStderr, "to-stderr")
	})

	t.Run("returns output along with the error", func(t *testing.T) {
		restoreStdout := util.CaptureStdOut()
		restoreStderr := captureStdErr()
		captured, err := exec.RunInteractiveCommandWithCapture(bashPath, []string{"-c", `echo why-it-failed >&2; exit 3`})
		_ = restoreStdout()
		_ = restoreStderr()
		require.Error(t, err)
		require.Contains(t, captured, "why-it-failed")
	})

	t.Run("passes bytes through unchanged", func(t *testing.T) {
		restoreStdout := util.CaptureStdOut()
		restoreStderr := captureStdErr()
		captured, err := exec.RunInteractiveCommandWithCapture(bashPath, []string{"-c", `printf '\033[31mred\033[0m\rprogress'`})
		_ = restoreStdout()
		_ = restoreStderr()
		require.NoError(t, err)
		// Unlike RunInteractiveCommandWithOutput(), colors and carriage returns survive.
		// stdout-only command: captured is "<stdout>\n<empty stderr>".
		require.Equal(t, "\x1b[31mred\x1b[0m\rprogress\n", captured)
	})
}
