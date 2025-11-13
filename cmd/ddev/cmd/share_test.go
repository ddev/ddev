package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	exec2 "github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/nodeps"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestShareCmd tests `ddev share`
func TestShareCmd(t *testing.T) {
	if os.Getenv("DDEV_TEST_SHARE_CMD") != "true" {
		t.Skip("Skipping because DDEV_TEST_SHARE_CMD != true")
	}
	if nodeps.IsWindows() {
		t.Skip("Skipping because unreliable on Windows due to DNS lookup failure")
	}
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		t.Skip("Skipping on GitHub actions because no auth can be provided")
	}
	assert := asrt.New(t)

	// Disable DDEV_DEBUG to prevent non-JSON output in ngrok logs
	t.Setenv("DDEV_DEBUG", "false")

	site := TestSites[0]
	defer site.Chdir()()

	// Configure ddev/ngrok to use json output to stdout
	cmd := exec.Command(DdevBin, "config", "--ngrok-args", "--log stdout --log-format=json")
	err := cmd.Start()
	require.NoError(t, err)
	err = cmd.Wait()
	require.NoError(t, err)

	cmd = exec.Command(DdevBin, "share")
	cmdReader, err := cmd.StdoutPipe()
	require.NoError(t, err)
	scanner := bufio.NewScanner(cmdReader)

	// Make absolutely sure the ngrok process gets killed off, because otherwise
	// the testbot (windows) can remain occupied forever.
	// nolint: errcheck
	t.Cleanup(func() {
		err = pKill(cmd)
		assert.NoError(err)
		_ = cmd.Wait()
		_ = cmdReader.Close()
		_, err = exec.LookPath("killall")
		// Try to kill ngrok any way we can, avoid having two run at same time.
		if err == nil {
			_, _ = exec2.RunHostCommand("killall", "-9", "ngrok")
		}

		if err != nil && !strings.Contains(err.Error(), "process already finished") {
			assert.NoError(err)
		}
	})

	// Use map[string]any to tolerate non-string log fields (e.g., enabled:false)
	var logData map[string]any
	var logLines []string

	scanDone := make(chan bool, 1)
	defer close(scanDone)

	// Read through the ngrok json output until we get the url it has opened
	go func() {
		for scanner.Scan() {
			logLine := scanner.Text()
			logLines = append(logLines, logLine)

			var m map[string]any
			if err := json.Unmarshal([]byte(logLine), &m); err != nil {
				// Continue scanning; some lines may not be JSON or may not match expected schema
				t.Logf("Ignoring ngrok log line (unmarshal error):\n  Line: %s\n  Error: %v", logLine, err)
				continue
			}

			// Assign to the shared logData only after successful unmarshal
			logData = m

			// If ngrok emitted an error, try to surface it
			if rawErr, ok := m["err"]; ok && rawErr != nil {
				switch e := rawErr.(type) {
				case string:
					if e != "" && e != "<nil>" {
						if strings.Contains(e, "Your account is limited to 1 simultaneous") {
							t.Errorf("Failed because ngrok account in use elsewhere: %s", e)
							break
						}
						t.Logf("ngrok error: %s", e)
					}
				default:
					if b, _ := json.Marshal(e); len(b) > 0 {
						t.Logf("ngrok error payload: %s", string(b))
					} else {
						t.Logf("ngrok error payload (non-JSON-marshalable): %#v", e)
					}
				}
			}

			// Stop reading once ngrok announces a URL
			if _, ok := m["url"]; ok {
				break
			}
		}
		scanDone <- true
	}()

	err = cmd.Start()
	require.NoError(t, err)
	select {
	case <-scanDone:
		fmt.Printf("Scanning all done at %v\n", time.Now())
	case <-time.After(20 * time.Second):
		// On timeout, print recent ngrok logs to help debugging
		t.Logf("Timed out waiting for ngrok; last %d log lines follow:", len(logLines))
		for i := max(0, len(logLines)-20); i < len(logLines); i++ {
			t.Logf("ngrok[%d]: %s", i, logLines[i])
		}
		t.Fatal("Timed out waiting for reads\n", time.Now())
	}
	// If URL is provided, try to hit it and look for expected response
	if rawURL, ok := logData["url"]; ok {
		url, ok := rawURL.(string)
		if !ok || url == "" {
			t.Errorf("url present but not a string: %#v (full logData=%#v)", rawURL, logData)
			return
		}
		resp, err := http.Get(url + site.Safe200URIWithExpectation.URI)
		if err != nil {
			t.Logf("http.Get on url=%s failed, err=%v", url+site.Safe200URIWithExpectation.URI, err)
			err = pKill(cmd)
			assert.NoError(err)
			return
		}
		//nolint: errcheck
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		assert.NoError(err)
		assert.Contains(string(body), site.Safe200URIWithExpectation.Expect)
	} else {
		// No URL found; dump recent logs for clarity
		t.Logf("No URL found in ngrok output; last %d log lines follow:", len(logLines))
		for i := max(0, len(logLines)-20); i < len(logLines); i++ {
			t.Logf("ngrok[%d]: %s", i, logLines[i])
		}
		t.Errorf("no URL found; last parsed logData=%#v", logData)
	}
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

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
