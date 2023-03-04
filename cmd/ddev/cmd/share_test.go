package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	exec2 "github.com/ddev/ddev/pkg/exec"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestShareCmd tests `ddev share`
func TestShareCmd(t *testing.T) {
	if os.Getenv("DDEV_TEST_SHARE_CMD") != "true" {
		t.Skip("Skipping because DDEV_TEST_SHARE_CMD != true")
	}
	if runtime.GOOS == "windows" {
		t.Skip("Skipping because unreliable on Windows due to DNS lookup failure")
	}
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		t.Skip("Skipping on GitHub actions because no auth can be provided")
	}
	assert := asrt.New(t)

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
	logData := make(map[string]string)

	scanDone := make(chan bool, 1)
	defer close(scanDone)

	// Read through the ngrok json output until we get the url it has opened
	go func() {
		for scanner.Scan() {
			logLine := scanner.Text()

			err := json.Unmarshal([]byte(logLine), &logData)
			if err != nil {
				switch err.(type) {
				case *json.SyntaxError:
					continue
				default:
					t.Errorf("failed unmarshalling %v: %v", logLine, err)
					break
				}
			}
			if logErr, ok := logData["err"]; ok && logErr != "<nil>" {
				if strings.Contains(logErr, "Your account is limited to 1 simultaneous") {
					t.Errorf("Failed because ngrok account in use elsewhere: %s", logErr)
					break
				}
			}
			if _, ok := logData["url"]; ok {
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
		t.Fatal("Timed out waiting for reads\n", time.Now())
	}
	// If URL is provided, try to hit it and look for expected response
	if url, ok := logData["url"]; ok {
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
		t.Errorf("no URL found: %v", logData)
	}
}

// pKill kills a started cmd; If windows, it shells out to the
// taskkill command.
func pKill(cmd *exec.Cmd) error {
	var err error
	if cmd == nil {
		return fmt.Errorf("pKill: cmd is nill")
	}
	if runtime.GOOS == "windows" {
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
