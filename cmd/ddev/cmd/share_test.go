package cmd

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/stretchr/testify/require"
)

// TestShareCmdNgrok tests `ddev share` with ngrok provider
func TestShareCmdNgrok(t *testing.T) {
	if nodeps.IsEnvFalse("DDEV_TEST_SHARE_CMD") {
		t.Skip("Skipping because DDEV_TEST_SHARE_CMD != true")
	}
	if nodeps.IsEnvFalse("DDEV_RUN_TEST_ANYWAY") && nodeps.IsWindows() {
		t.Skip("Skipping because unreliable on Windows due to DNS lookup failure")
	}
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		t.Skip("Skipping on GitHub actions because no auth can be provided")
	}
	t.Setenv("DDEV_GOROUTINES", "")

	// Check if ngrok is installed
	_, err := exec.LookPath("ngrok")
	if err != nil {
		t.Skip("Skipping because ngrok is not installed")
	}

	// Disable DDEV_DEBUG to prevent non-JSON output in ngrok logs
	t.Setenv("DDEV_DEBUG", "")

	site := TestSites[0]
	defer site.Chdir()()

	// Start the project first, so the tunnel wait below does not include image builds
	err = exec.Command(DdevBin, "start").Run()
	require.NoError(t, err)

	cmd := exec.Command(DdevBin, "share", "--provider=ngrok")
	// Enable debug output to get verbose ngrok.sh logging
	cmd.Env = append(os.Environ(), "DDEV_DEBUG=true")
	var stdoutBuf, stderrBuf syncBuffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	t.Log("Starting ngrok share command...")
	exited := startCmd(t, cmd)
	t.Cleanup(func() {
		_ = pKill(cmd)
		<-exited
	})

	// Poll for output with intermediate logging (ngrok can take several seconds)
	t.Log("Waiting for ngrok tunnel to establish...")
	maxWait := 30 * time.Second
	pollInterval := 2 * time.Second
	elapsed := time.Duration(0)
	lastStderrLen := 0

poll:
	for elapsed < maxWait {
		time.Sleep(pollInterval)
		elapsed += pollInterval

		stdoutOutput := stdoutBuf.String()
		stderrOutput := stderrBuf.String()

		// Log new stderr content if there's been progress (helps see what ngrok.sh is doing)
		if len(stderrOutput) > lastStderrLen {
			newContent := stderrOutput[lastStderrLen:]
			// Only log if there's substantial new content (avoid spam)
			if len(strings.TrimSpace(newContent)) > 0 {
				t.Logf("New output:\n%s", newContent)
			}
			lastStderrLen = len(stderrOutput)
		}

		// Check for URL success
		if strings.Contains(stdoutOutput, "Tunnel URL:") {
			t.Logf("Ngrok tunnel established after %v", elapsed)
			break
		}

		// Check for account limit error (non-fatal for test purposes)
		if strings.Contains(stderrOutput, "Your account is limited to 1 simultaneous") {
			t.Logf("Ngrok account in use elsewhere (expected in development): %v", elapsed)
			break
		}

		// Diagnostic: Check if ngrok API is reachable at 6 seconds
		if elapsed == 6*time.Second {
			resp, err := http.Get("http://localhost:4040/api/tunnels")
			if err != nil {
				t.Logf("Diagnostic: ngrok API not reachable at localhost:4040: %v", err)
			} else {
				body, _ := io.ReadAll(resp.Body)
				_ = resp.Body.Close()
				t.Logf("Diagnostic: ngrok API response: status=%d, body_length=%d", resp.StatusCode, len(body))
			}
		}

		select {
		case <-exited:
			t.Logf("ddev share exited after %v without a tunnel URL", elapsed)
			break poll
		default:
			t.Logf("Still waiting for tunnel... (%v/%v)", elapsed, maxWait)
		}
	}

	// ddev share might already have exited if the account limit was hit
	_ = pKill(cmd)
	<-exited

	// Check captured output
	stdoutOutput := stdoutBuf.String()
	stderrOutput := stderrBuf.String()
	t.Logf("Stdout output:\n%s", stdoutOutput)
	t.Logf("Stderr output:\n%s", stderrOutput)

	// Verify ngrok provider successfully established tunnel
	// The test should only pass if ngrok actually worked
	hasURL := strings.Contains(stdoutOutput, "Tunnel URL:")
	require.True(t, hasURL,
		"Should show Tunnel URL (ngrok provider successfully established tunnel)")

	// If we got a URL, verify it looks like ngrok
	if hasURL {
		require.Contains(t, stdoutOutput, "ngrok")
	}
	if !nodeps.IsWindows() {
		require.Contains(t, stdoutOutput, "Stopping tunnel", "ddev share should handle SIGTERM by stopping the tunnel")
	}
}

// TestShareCmdCloudflared tests `ddev share` with cloudflared
func TestShareCmdCloudflared(t *testing.T) {
	if nodeps.IsEnvFalse("DDEV_TEST_SHARE_CMD") {
		t.Skip("Skipping because DDEV_TEST_SHARE_CMD != true")
	}
	if nodeps.IsEnvFalse("DDEV_RUN_TEST_ANYWAY") && nodeps.IsWindows() {
		t.Skip("Skipping because unreliable on Windows")
	}
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		t.Skip("Skipping on GitHub actions")
	}
	t.Setenv("DDEV_GOROUTINES", "")

	// Check if cloudflared is installed
	_, err := exec.LookPath("cloudflared")
	if err != nil {
		t.Skip("Skipping because cloudflared is not installed")
	}

	site := TestSites[0]
	defer site.Chdir()()

	err = exec.Command(DdevBin, "start").Run()
	require.NoError(t, err)

	cmd := exec.Command(DdevBin, "share", "--provider=cloudflared")
	var stdoutBuf, stderrBuf syncBuffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	t.Log("Starting cloudflared share command...")
	exited := startCmd(t, cmd)
	t.Cleanup(func() {
		_ = pKill(cmd)
		<-exited
	})

	// Poll for output with intermediate logging (cloudflared can take 10+ seconds)
	t.Log("Waiting for cloudflared tunnel to establish...")
	maxWait := 20 * time.Second
	pollInterval := 2 * time.Second
	elapsed := time.Duration(0)

poll:
	for elapsed < maxWait {
		time.Sleep(pollInterval)
		elapsed += pollInterval

		stdoutOutput := stdoutBuf.String()
		stderrOutput := stderrBuf.String()

		if strings.Contains(stdoutOutput, "Tunnel URL:") && strings.Contains(stdoutOutput, "trycloudflare.com") {
			t.Logf("Cloudflared tunnel established after %v", elapsed)
			break
		}
		// Detect cloudflare quick-tunnel API errors early so we can log and fail with context.
		if strings.Contains(stderrOutput, "Error unmarshaling QuickTunnel") || strings.Contains(stderrOutput, "failed to unmarshal quick Tunnel") {
			t.Logf("Stdout so far:\n%s", stdoutOutput)
			t.Fatalf("cloudflare quick-tunnel API returned an unmarshal error (possible transient 500):\n%s", stderrOutput)
		}
		select {
		case <-exited:
			t.Logf("ddev share exited after %v without a tunnel URL", elapsed)
			break poll
		default:
			t.Logf("Still waiting for tunnel... (%v/%v)", elapsed, maxWait)
		}
	}

	// Stop the share command
	_ = pKill(cmd)
	<-exited

	// Check captured output
	stdoutOutput := stdoutBuf.String()
	stderrOutput := stderrBuf.String()
	t.Logf("Stdout output:\n%s", stdoutOutput)
	t.Logf("Stderr output:\n%s", stderrOutput)

	// Verify URL was displayed
	require.Contains(t, stdoutOutput, "Tunnel URL:",
		"cloudflared did not output a tunnel URL; stderr output:\n%s", stderrOutput)
	require.Contains(t, stdoutOutput, "trycloudflare.com",
		"tunnel URL does not contain trycloudflare.com; stderr output:\n%s", stderrOutput)
	if !nodeps.IsWindows() {
		require.Contains(t, stdoutOutput, "Stopping tunnel", "ddev share should handle SIGTERM by stopping the tunnel")
	}
}

// TestShareCmdProviderSystem tests the script-based provider system
func TestShareCmdProviderSystem(t *testing.T) {
	if nodeps.IsEnvFalse("DDEV_RUN_TEST_ANYWAY") && nodeps.IsWindows() {
		t.Skip("Skipping: Test cannot work yet on traditional windows (pkill, etc)")
	}
	t.Setenv("DDEV_GOROUTINES", "")

	expectedQRCode, err := os.ReadFile(filepath.Join("testdata", t.Name(), "qrcode.txt"))
	require.NoError(t, err)
	expectedNoColorQRCode, err := os.ReadFile(filepath.Join("testdata", t.Name(), "qrcode_no_color.txt"))
	require.NoError(t, err)

	site := TestSites[0]
	defer site.Chdir()()

	// Ensure project is started
	cmd := exec.Command(DdevBin, "start")
	err = cmd.Run()
	require.NoError(t, err)

	// Test 1: Create a mock provider and verify URL capture
	t.Run("MockProviderURLCapture", func(t *testing.T) {
		mockScript := `#!/usr/bin/env bash
echo "Starting mock tunnel..." >&2
sleep 1
echo "https://mock-test-tunnel.example.com"
sleep 2
`
		mockPath := site.Dir + "/.ddev/share-providers/mock-test.sh"
		err := os.WriteFile(mockPath, []byte(mockScript), 0755)
		require.NoError(t, err)
		t.Cleanup(func() {
			_ = os.Remove(mockPath)
		})

		expectedQR := strings.ReplaceAll(string(expectedQRCode), "\r\n", "\n")
		expectedNoColorQR := strings.ReplaceAll(string(expectedNoColorQRCode), "\r\n", "\n")

		runShare := func(noColor string, wantContent string) string {
			t.Setenv("NO_COLOR", noColor)
			cmd := exec.Command(DdevBin, "share", "--provider=mock-test")
			var stdoutBuf, stderrBuf syncBuffer
			cmd.Stdout = &stdoutBuf
			cmd.Stderr = &stderrBuf

			exited := startCmd(t, cmd)

			t.Cleanup(func() {
				_ = pKill(cmd)
				<-exited
			})

			// ddev share can take several seconds to launch the provider on slow hosts
			waitErr := waitForContent(&stdoutBuf, wantContent, exited, 30*time.Second)

			// ddev share exits on its own once the mock provider does, so the kill may find it gone
			_ = pKill(cmd)
			<-exited

			t.Logf("Stdout output with NO_COLOR=%s:\n%s", noColor, stdoutBuf.String())
			t.Logf("Stderr output with NO_COLOR=%s:\n%s", noColor, stderrBuf.String())
			require.NoError(t, waitErr)
			return strings.ReplaceAll(stdoutBuf.String(), "\r\n", "\n")
		}

		stdoutOutput := runShare("", expectedQR)
		// util.Success() writes to stdout, not stderr
		require.Contains(t, stdoutOutput, "Tunnel URL:")
		require.Contains(t, stdoutOutput, "mock-test-tunnel")
		require.Contains(t, stdoutOutput, expectedQR)
		require.Contains(t, runShare("1", expectedNoColorQR), expectedNoColorQR)
	})

	// Test 2: Verify hooks have access to DDEV_SHARE_URL
	t.Run("HookURLAccess", func(t *testing.T) {
		mockScript := `#!/usr/bin/env bash
echo "https://hook-test-tunnel.example.com"
sleep 30
`
		mockPath := site.Dir + "/.ddev/share-providers/hook-test.sh"
		err := os.WriteFile(mockPath, []byte(mockScript), 0755)
		require.NoError(t, err)
		t.Cleanup(func() {
			_ = os.Remove(mockPath)
		})

		// Create config.hooks.yaml with pre-share hook that checks DDEV_SHARE_URL
		hooksConfig := `hooks:
  pre-share:
    - exec-host: |
        if [ -n "$DDEV_SHARE_URL" ]; then
          echo "HOOK_SUCCESS: DDEV_SHARE_URL=$DDEV_SHARE_URL" >&2
        else
          echo "HOOK_FAILURE: DDEV_SHARE_URL not set" >&2
        fi
`
		hooksPath := site.Dir + "/.ddev/config.hooks.yaml"
		err = os.WriteFile(hooksPath, []byte(hooksConfig), 0644)
		require.NoError(t, err)
		t.Cleanup(func() {
			_ = os.Remove(hooksPath)
		})

		cmd := exec.Command(DdevBin, "share", "--provider=hook-test")
		stderrReader, err := cmd.StderrPipe()
		require.NoError(t, err)

		t.Cleanup(func() {
			_ = pKill(cmd)
			_ = cmd.Wait()
			_ = stderrReader.Close()
		})

		err = cmd.Start()
		require.NoError(t, err)

		// Check stderr for hook output
		var hookSuccess atomic.Bool
		scanner := bufio.NewScanner(stderrReader)
		go func() {
			for scanner.Scan() {
				line := scanner.Text()
				if strings.Contains(line, "HOOK_SUCCESS") && strings.Contains(line, "hook-test-tunnel") {
					hookSuccess.Store(true)
					break
				}
			}
		}()

		require.Eventually(t, hookSuccess.Load, 30*time.Second, 250*time.Millisecond, "Pre-share hook should have access to DDEV_SHARE_URL")
	})

	// Test 3: Provider priority (flag > config > default)
	t.Run("ProviderPriority", func(t *testing.T) {
		// Set config default provider
		cmd := exec.Command(DdevBin, "config", "--share-default-provider=config-provider")
		err := cmd.Run()
		require.NoError(t, err)

		// Create mock providers and collect paths for cleanup
		var mockPaths []string
		for _, name := range []string{"config-provider", "flag-provider"} {
			mockScript := fmt.Sprintf(`#!/usr/bin/env bash
echo "https://%s-tunnel.example.com"
sleep 2
`, name)
			mockPath := site.Dir + "/.ddev/share-providers/" + name + ".sh"
			err = os.WriteFile(mockPath, []byte(mockScript), 0755)
			require.NoError(t, err)
			mockPaths = append(mockPaths, mockPath)
		}

		// Cleanup mock provider files
		t.Cleanup(func() {
			for _, path := range mockPaths {
				_ = os.Remove(path)
			}
		})

		// Test flag overrides config
		cmd = exec.Command(DdevBin, "share", "--provider=flag-provider")
		var stdoutBuf, stderrBuf syncBuffer
		cmd.Stdout = &stdoutBuf
		cmd.Stderr = &stderrBuf

		exited := startCmd(t, cmd)

		t.Cleanup(func() {
			_ = pKill(cmd)
			<-exited
			// Reset config
			_ = exec.Command(DdevBin, "config", "--share-default-provider=").Run()
		})

		waitErr := waitForContent(&stdoutBuf, "Tunnel URL:", exited, 30*time.Second)

		_ = pKill(cmd)
		<-exited

		// Check captured output
		stdoutOutput := stdoutBuf.String()
		stderrOutput := stderrBuf.String()
		t.Logf("Stdout output:\n%s", stdoutOutput)
		t.Logf("Stderr output:\n%s", stderrOutput)
		require.NoError(t, waitErr)
		// util.Success() writes to stdout, not stderr
		require.Contains(t, stdoutOutput, "Tunnel URL:")
		require.Contains(t, stdoutOutput, "flag-provider-tunnel")
	})

	// Test 4: Provider not found error handling
	t.Run("ProviderNotFound", func(t *testing.T) {
		cmd := exec.Command(DdevBin, "share", "--provider=nonexistent")
		output, err := cmd.CombinedOutput()
		require.Error(t, err)
		require.Contains(t, string(output), "Failed to find share provider 'nonexistent'")
	})

	// Test 5: --provider-args flag passes DDEV_SHARE_ARGS to provider
	t.Run("ProviderArgsFlag", func(t *testing.T) {
		// Create a mock provider that echoes DDEV_SHARE_ARGS to stderr
		mockScript := `#!/usr/bin/env bash
echo "ARGS_RECEIVED: DDEV_SHARE_ARGS=${DDEV_SHARE_ARGS}" >&2
echo "https://args-test.example.com"
sleep 2
`
		mockPath := site.Dir + "/.ddev/share-providers/args-test.sh"
		err := os.WriteFile(mockPath, []byte(mockScript), 0755)
		require.NoError(t, err)
		t.Cleanup(func() {
			_ = os.Remove(mockPath)
		})

		cmd := exec.Command(DdevBin, "share", "--provider=args-test", "--provider-args=--custom-flag value123")
		var stdoutBuf, stderrBuf syncBuffer
		cmd.Stdout = &stdoutBuf
		cmd.Stderr = &stderrBuf

		exited := startCmd(t, cmd)

		t.Cleanup(func() {
			_ = pKill(cmd)
			<-exited
		})

		waitErr := waitForContent(&stdoutBuf, "Tunnel URL:", exited, 30*time.Second)

		_ = pKill(cmd)
		<-exited

		stderrOutput := stderrBuf.String()
		stdoutOutput := stdoutBuf.String()
		t.Logf("Stdout: %s", stdoutOutput)
		t.Logf("Stderr: %s", stderrOutput)
		require.NoError(t, waitErr)

		// Verify DDEV_SHARE_ARGS was passed to the provider (shown in ddev's output message)
		require.Contains(t, stdoutOutput, "with args: --custom-flag value123",
			"Provider should receive DDEV_SHARE_ARGS from --provider-args flag")
		require.Contains(t, stdoutOutput, "Tunnel URL:")
	})
}

// syncBuffer is a strings.Builder that is safe to read while a running
// command is still writing to it.
type syncBuffer struct {
	mu sync.Mutex
	b  strings.Builder
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

// startCmd starts cmd and returns a channel that closes once it has exited.
// Waiting on the channel instead of calling cmd.Wait() again keeps Wait to a
// single caller. WaitDelay bounds Wait if a leftover child keeps the output
// pipes open after cmd exits.
func startCmd(t *testing.T, cmd *exec.Cmd) <-chan struct{} {
	t.Helper()
	cmd.WaitDelay = 5 * time.Second
	require.NoError(t, cmd.Start())
	exited := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		close(exited)
	}()
	return exited
}

// waitForContent polls buf until it contains want, ignoring CRLF differences.
// It gives up early if the command writing to buf exits first.
func waitForContent(buf *syncBuffer, want string, exited <-chan struct{}, timeout time.Duration) error {
	contains := func() bool {
		return strings.Contains(strings.ReplaceAll(buf.String(), "\r\n", "\n"), want)
	}
	deadline := time.After(timeout)
	for !contains() {
		select {
		case <-exited:
			// Wait has finished copying output, so this check sees all of it
			if contains() {
				return nil
			}
			return fmt.Errorf("command exited before its output contained the expected content")
		case <-deadline:
			return fmt.Errorf("output did not contain expected content after %v", timeout)
		case <-time.After(250 * time.Millisecond):
		}
	}
	return nil
}

// pKill stops a started cmd. Elsewhere than Windows it sends SIGTERM, so
// ddev share can kill its provider's process group the way Ctrl-C does;
// SIGKILL would orphan the tunnel, which keeps cmd's output pipes open.
func pKill(cmd *exec.Cmd) error {
	var err error
	if cmd == nil {
		return fmt.Errorf("pKill: cmd is nill")
	}
	if nodeps.IsWindows() {
		// Windows has a completely different process model, no SIGCHLD,
		// no killing of subprocesses. I wasn't successful in finding a way
		// to properly kill a process set using golang; rfay 20190622
		kill := exec.Command("TASKKILL", "/T", "/F", "/PID", strconv.Itoa(cmd.Process.Pid))
		kill.Stderr = os.Stderr
		kill.Stdout = os.Stdout
		err = kill.Run()
	} else {
		err = cmd.Process.Signal(syscall.SIGTERM)
	}
	return err
}
