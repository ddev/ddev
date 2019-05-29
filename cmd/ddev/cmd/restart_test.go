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
	site := DevTestSites[0]
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

// TestDevRestartJSON runs `ddev restart -j` on the test apps and harvests and checks the output
func TestDevRestartJSON(t *testing.T) {
	assert := asrt.New(t)
	site := DevTestSites[0]
	cleanup := site.Chdir()
	defer cleanup()

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

	var item map[string]interface{}
	for _, item = range logItems {
		if item["level"] == "info" && item["msg"] != nil && strings.Contains(item["msg"].(string), "Your project can be reached at "+strings.Join(app.GetAllURLs(), ", ")) {
			break
		}
	}
	assert.Contains(item["msg"], "Your project can be reached at "+strings.Join(app.GetAllURLs(), ", "))
}
