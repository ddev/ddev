package cmd

import (
	"fmt"
	"github.com/ddev/ddev/pkg/globalconfig"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/util"
	"github.com/stretchr/testify/require"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/exec"
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

	app, err := ddevapp.GetActiveApp("")
	if err != nil {
		assert.Fail("Could not find an active DDEV configuration: %v", err)
	}

	assert.Contains(string(out), "Your project can be reached at")
	switch slices.Contains(globalconfig.DdevGlobalConfig.OmitContainersGlobal, "ddev-router") {
	case true:
		assert.Contains(string(out), "127.0.0.1")
	case false:
		assert.NotContains(string(out), "127.0.0.1")
		assert.Contains(string(out), app.GetPrimaryURL())
	}
	cleanup()
}

// TestCmdRestartJSON runs `ddev restart -j` on the test apps and harvests and checks the output
func TestCmdRestartJSON(t *testing.T) {
	assert := asrt.New(t)
	site := TestSites[0]
	origDdevDebug := os.Getenv("DDEV_DEBUG")
	_ = os.Unsetenv("DDEV_DEBUG")
	origDir, _ := os.Getwd()
	err := os.Chdir(site.Dir)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.Chdir(origDir)
		_ = os.Setenv("DDEV_DEBUG", origDdevDebug)
	})

	_, err = ddevapp.GetActiveApp("")
	if err != nil {
		assert.Fail("Could not find an active DDEV configuration: %v", err)
	}

	bash := util.FindBashPath()
	out, err := exec.RunHostCommand(bash, "-c", fmt.Sprintf("%s restart -j 2>/dev/null", DdevBin))
	require.NoError(t, err)

	logItems, err := unmarshalJSONLogs(out)
	require.NoError(t, err)

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
