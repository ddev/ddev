package cmd

import (
	"testing"

	"strings"

	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/exec"
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

		app, err := ddevapp.GetActiveApp("")
		if err != nil {
			assert.Fail("Could not find an active ddev configuration: %v", err)
		}

		assert.Contains(string(out), "Your project can be reached at")
		assert.Contains(string(out), strings.Join(app.GetAllURLs(), ", "))
		cleanup()
	}
}

// TestDevRestartJSON runs `ddev restart -j` on the test apps and harvests and checks the output
func TestDevRestartJSON(t *testing.T) {
	assert := asrt.New(t)
	for _, site := range DevTestSites {
		cleanup := site.Chdir()

		app, err := ddevapp.GetActiveApp("")
		if err != nil {
			assert.Fail("Could not find an active ddev configuration: %v", err)
		}

		args := []string{"restart", "-j"}
		out, err := exec.RunCommand(DdevBin, args)
		assert.NoError(err)

		logItems, err := unmarshalJSONLogs(out)
		assert.NoError(err)

		// The key item should be the last item; there may be a warning
		// or other info before that.
		data := logItems[len(logItems)-1]
		assert.EqualValues(data["level"], "info")
		assert.Contains(data["msg"], "Your project can be reached at "+strings.Join(app.GetAllURLs(), ", "))

		cleanup()
	}
}
