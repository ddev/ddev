package cmd

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	exec2 "github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/stretchr/testify/require"
)

// TestShareCmdNgrok tests `ddev share` with ngrok provider
func TestShareCmdNgrok(t *testing.T) {
	if os.Getenv("DDEV_TEST_SHARE_CMD") != "true" {
		t.Skip("Skipping because DDEV_TEST_SHARE_CMD != true")
	}
	if nodeps.IsWindows() {
		t.Skip("Skipping because unreliable on Windows due to DNS lookup failure")
	}
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		t.Skip("Skipping on GitHub actions because no auth can be provided")
	}
	t.Setenv(`DDEV_GOROUTINES`, "")

	// Check if ngrok is installed
	_, err := exec.LookPath("ngrok")
	if err != nil {
		t.Skip("Skipping because ngrok is not installed")
	}

	// Disable DDEV_DEBUG to prevent non-JSON output in ngrok logs
	t.Setenv("DDEV_DEBUG", "false")

	site := TestSites[0]
	defer site.Chdir()()

	cmd := exec.Command(DdevBin, "share", "--provider=ngrok")
	// Enable debug output to get verbose ngrok.sh logging
	cmd.Env = append(os.Environ(), "DDEV_DEBUG=true")
	var stdoutBuf, stderrBuf strings.Builder
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	t.Cleanup(func() {
		_ = pKill(cmd)
		_ = cmd.Wait()
		_, err := exec.LookPath("killall")
		// Try to kill ngrok any way we can, avoid having two run at same time.
		if err == nil {
			_, _ = exec2.RunHostCommand("killall", "-9", "ngrok")
		}
	})

	t.Log("Starting ngrok share command...")
	err = cmd.Start()
	require.NoError(t, err)

	// Poll for output with intermediate logging (ngrok can take several seconds)
	t.Log("Waiting for ngrok tunnel to establish...")
	maxWait := 30 * time.Second
	pollInterval := 2 * time.Second
	elapsed := time.Duration(0)
	lastStderrLen := 0

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

		t.Logf("Still waiting for tunnel... (%v/%v)", elapsed, maxWait)
	}

	// Kill the share command (might already be dead if account limit hit)
	_ = pKill(cmd)

	// Wait for command with timeout (pipes might take time to close)
	waitDone := make(chan bool)
	go func() {
		_ = cmd.Wait()
		waitDone <- true
	}()
	select {
	case <-waitDone:
		// Command exited cleanly
	case <-time.After(5 * time.Second):
		t.Log("Wait timed out after kill, continuing...")
	}

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
}

// TestShareCmdCloudflared tests `ddev share` with cloudflared
func TestShareCmdCloudflared(t *testing.T) {
	if os.Getenv("DDEV_TEST_SHARE_CMD") != "true" {
		t.Skip("Skipping because DDEV_TEST_SHARE_CMD != true")
	}
	if nodeps.IsWindows() {
		t.Skip("Skipping because unreliable on Windows")
	}
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		t.Skip("Skipping on GitHub actions")
	}
	t.Setenv(`DDEV_GOROUTINES`, "")

	// Check if cloudflared is installed
	_, err := exec.LookPath("cloudflared")
	if err != nil {
		t.Skip("Skipping because cloudflared is not installed")
	}

	site := TestSites[0]
	defer site.Chdir()()

	cmd := exec.Command(DdevBin, "share", "--provider=cloudflared")
	var stdoutBuf, stderrBuf strings.Builder
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	t.Cleanup(func() {
		_ = pKill(cmd)
		_ = cmd.Wait()
	})

	t.Log("Starting cloudflared share command...")
	err = cmd.Start()
	require.NoError(t, err)

	// Poll for output with intermediate logging (cloudflared can take 10+ seconds)
	t.Log("Waiting for cloudflared tunnel to establish...")
	maxWait := 20 * time.Second
	pollInterval := 2 * time.Second
	elapsed := time.Duration(0)

	for elapsed < maxWait {
		time.Sleep(pollInterval)
		elapsed += pollInterval

		stdoutOutput := stdoutBuf.String()
		if strings.Contains(stdoutOutput, "Tunnel URL:") && strings.Contains(stdoutOutput, "trycloudflare.com") {
			t.Logf("Cloudflared tunnel established after %v", elapsed)
			break
		}
		t.Logf("Still waiting for tunnel... (%v/%v)", elapsed, maxWait)
	}

	// Kill the share command
	_ = pKill(cmd)

	// Wait for command with timeout (pipes might take time to close)
	waitDone := make(chan bool)
	go func() {
		_ = cmd.Wait()
		waitDone <- true
	}()
	select {
	case <-waitDone:
		// Command exited cleanly
	case <-time.After(5 * time.Second):
		t.Log("Wait timed out after kill, continuing...")
	}

	// Check captured output
	stdoutOutput := stdoutBuf.String()
	stderrOutput := stderrBuf.String()
	t.Logf("Stdout output:\n%s", stdoutOutput)
	t.Logf("Stderr output:\n%s", stderrOutput)

	// Verify URL was displayed
	require.Contains(t, stdoutOutput, "Tunnel URL:")
	require.Contains(t, stdoutOutput, "trycloudflare.com")
}

// TestShareCmdProviderSystem tests the script-based provider system
func TestShareCmdProviderSystem(t *testing.T) {
	if nodeps.IsWindows() {
		t.Skip("Skipping: Test cannot work on traditional windows (pkill, etc). ddev share may not work on traditional windows at all")
	}
	t.Setenv(`DDEV_GOROUTINES`, "")

	site := TestSites[0]
	defer site.Chdir()()

	// Ensure project is started
	cmd := exec.Command(DdevBin, "start")
	err := cmd.Run()
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

		cmd := exec.Command(DdevBin, "share", "--provider=mock-test")
		var stdoutBuf, stderrBuf strings.Builder
		cmd.Stdout = &stdoutBuf
		cmd.Stderr = &stderrBuf

		err = cmd.Start()
		require.NoError(t, err)

		t.Cleanup(func() {
			_ = pKill(cmd)
			_ = cmd.Wait()
		})

		// Wait for provider to output URL and ddev share to capture/display it
		time.Sleep(3 * time.Second)

		// Kill the share command to end the test
		err = pKill(cmd)
		require.NoError(t, err)
		_ = cmd.Wait()

		// Check captured output
		stdoutOutput := stdoutBuf.String()
		stderrOutput := stderrBuf.String()
		t.Logf("Stdout output:\n%s", stdoutOutput)
		t.Logf("Stderr output:\n%s", stderrOutput)
		// util.Success() writes to stdout, not stderr
		require.Contains(t, stdoutOutput, "Tunnel URL:")
		require.Contains(t, stdoutOutput, "mock-test-tunnel")
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

		// Wait for hook execution
		time.Sleep(3 * time.Second)

		require.True(t, hookSuccess.Load(), "Pre-share hook should have access to DDEV_SHARE_URL")
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
		var stdoutBuf, stderrBuf strings.Builder
		cmd.Stdout = &stdoutBuf
		cmd.Stderr = &stderrBuf

		err = cmd.Start()
		require.NoError(t, err)

		t.Cleanup(func() {
			_ = pKill(cmd)
			_ = cmd.Wait()
			// Reset config
			_ = exec.Command(DdevBin, "config", "--share-default-provider=").Run()
		})

		// Wait for provider to output URL and ddev share to capture/display it
		time.Sleep(3 * time.Second)

		// Kill the share command to end the test
		err = pKill(cmd)
		require.NoError(t, err)
		_ = cmd.Wait()

		// Check captured output
		stdoutOutput := stdoutBuf.String()
		stderrOutput := stderrBuf.String()
		t.Logf("Stdout output:\n%s", stdoutOutput)
		t.Logf("Stderr output:\n%s", stderrOutput)
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
		var stdoutBuf, stderrBuf strings.Builder
		cmd.Stdout = &stdoutBuf
		cmd.Stderr = &stderrBuf

		err = cmd.Start()
		require.NoError(t, err)

		t.Cleanup(func() {
			_ = pKill(cmd)
			_ = cmd.Wait()
		})

		// Wait for provider to execute
		time.Sleep(3 * time.Second)

		_ = pKill(cmd)
		_ = cmd.Wait()

		stderrOutput := stderrBuf.String()
		stdoutOutput := stdoutBuf.String()
		t.Logf("Stdout: %s", stdoutOutput)
		t.Logf("Stderr: %s", stderrOutput)

		// Verify DDEV_SHARE_ARGS was passed to the provider (shown in ddev's output message)
		require.Contains(t, stdoutOutput, "with args: --custom-flag value123",
			"Provider should receive DDEV_SHARE_ARGS from --provider-args flag")
		require.Contains(t, stdoutOutput, "Tunnel URL:")
	})
}

// pKill kills a started cmd; If windows, it shells out to the
// taskkill command.
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
		err = cmd.Process.Kill()
	}
	return err
}
