package cmd

import (
	"github.com/stretchr/testify/require"
	"testing"

	"strings"

	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/exec"
	asrt "github.com/stretchr/testify/assert"
)

// TestCmdRestart runs `ddev restart` on the test apps
func TestCmdRestart(t *testing.T) {
	assert := asrt.New(t)
	site := TestSites[0]
	cleanup := site.Chdir()

	args := []string{"restart"}
	out, err := exec.RunCommand(DdevBin, args)
	assert.NoError(err)

	_, err = ddevapp.GetActiveApp("")
	if err != nil {
		assert.Fail("Could not find an active ddev configuration: %v", err)
	}

	assert.Contains(string(out), "Your project can be reached at")
	cleanup()
}

// TestCmdRestartJSON runs `ddev restart -j` on the test apps and harvests and checks the output
func TestCmdRestartJSON(t *testing.T) {
	assert := asrt.New(t)
	site := TestSites[0]
	cleanup := site.Chdir()
	defer cleanup()

	_, err := ddevapp.GetActiveApp("")
	if err != nil {
		assert.Fail("Could not find an active ddev configuration: %v", err)
	}

	args := []string{"restart", "-j"}
	out, err := exec.RunCommand(DdevBin, args)
	require.NoError(t, err)

	logItems, err := unmarshalJSONLogs(out)
	assert.NoError(err)

	// The key item should be the last item; there may be a warning
	// or other info before that.

	var item map[string]interface{}
	for _, item = range logItems {
		if item["level"] == "info" && item["msg"] != nil && strings.Contains(item["msg"].(string), "Your project can be reached at") {
			break
		}
	}
	assert.Contains(item["msg"], "Your project can be reached at")
}
