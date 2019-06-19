package cmd

import (
	"bufio"
	"encoding/json"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"net/http"
	"os/exec"
	"syscall"
	"testing"
)

// TestShareCmd tests `ddev share`
func TestShareCmd(t *testing.T) {
	assert := asrt.New(t)
	urlRead := false

	site := TestSites[0]
	defer site.Chdir()()

	// Configure ddev/ngrok to use json output to stdout
	cmd := exec.Command(DdevBin, "config", "--ngrok-args", "-log stdout -log-format=json")
	err := cmd.Start()
	require.NoError(t, err)
	err = cmd.Wait()
	require.NoError(t, err)

	cmd = exec.Command(DdevBin, "share", "--use-http")
	cmdReader, err := cmd.StdoutPipe()
	require.NoError(t, err)
	scanner := bufio.NewScanner(cmdReader)

	// Read through the ngrok json output until we get the url it has opened
	go func() {
		for scanner.Scan() {
			logLine := scanner.Text()
			logData := make(map[string]string)

			err := json.Unmarshal([]byte(logLine), &logData)
			if err != nil {
				switch err.(type) {
				case *json.SyntaxError:
					continue
				default:
					t.Logf("failed unmarshalling %v: %v", logLine, err)
					require.NoError(t, err)
				}
			}
			// If URL is provided, try to hit it and look for expected response
			if url, ok := logData["url"]; ok {
				resp, err := http.Get(url + site.Safe200URIWithExpectation.URI)
				assert.NoError(err)
				defer resp.Body.Close()
				body, err := ioutil.ReadAll(resp.Body)
				assert.Contains(string(body), site.Safe200URIWithExpectation.Expect)
				err = cmd.Process.Signal(syscall.SIGTERM)
				assert.NoError(err)
				urlRead = true
				break
			}
		}
	}()
	err = cmd.Start()
	require.NoError(t, err)
	_ = cmd.Wait()
	assert.True(urlRead)
}
