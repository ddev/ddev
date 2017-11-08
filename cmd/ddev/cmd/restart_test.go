package cmd

import (
	"testing"

	"encoding/json"

	"strings"

	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/plugins/platform"
	log "github.com/sirupsen/logrus"
	asrt "github.com/stretchr/testify/assert"
)

// TestDevRestart runs `ddev restart` on the test apps
func TestDevRestart(t *testing.T) {
	assert := asrt.New(t)
	for _, site := range DevTestSites {
		cleanup := site.Chdir()

		args := []string{"restart"}
		out, err := exec.RunCommand(DdevBin, args)
		assert.NoError(err)

		app, err := platform.GetActiveApp("")
		if err != nil {
			assert.Fail("Could not find an active ddev configuration: %v", err)
		}

		assert.Contains(string(out), "Your application can be reached at")
		assert.Contains(string(out), app.URL())

		cleanup()
	}
}

// TestDevRestartJSON runs `ddev restart -j` on the test apps and harvests and checks the output
func TestDevRestartJSON(t *testing.T) {
	assert := asrt.New(t)
	for _, site := range DevTestSites {
		cleanup := site.Chdir()

		app, err := platform.GetActiveApp("")
		if err != nil {
			assert.Fail("Could not find an active ddev configuration: %v", err)
		}

		args := []string{"restart", "-j"}
		out, err := exec.RunCommand(DdevBin, args)
		assert.NoError(err)
		logStrings := strings.Split(out, "\n")
		// We expect 4 lines of json in result, and a blank (Restarting,
		// need-add-hosts, successfully restarted, can be reached at,
		// blank at end)
		assert.True(len(logStrings) >= 3)

		// Wander through the json output lines making sure they're reasonable json.
		for _, entry := range logStrings {
			if entry != "" { // Ignore empty line.
				// Unmarshall the json results. Normal log entries have 3 fields
				data := make(log.Fields, 3)
				err = json.Unmarshal([]byte(entry), &data)
				assert.NoError(err)
				if !strings.Contains(data["msg"].(string), "You must manually add the following") {
					assert.EqualValues(data["level"], "info")
				}
				assert.NotEmpty(data["msg"])
			}
		}

		// Go ahead and look for normal strings within the json output.
		assert.Contains(string(out), "Your application can be reached at")
		assert.Contains(string(out), app.URL())

		cleanup()
	}
}
