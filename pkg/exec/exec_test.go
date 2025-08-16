package exec_test

import (
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
