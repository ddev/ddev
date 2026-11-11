package cmd

import (
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/util"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	t.Setenv("DDEV_DEBUG", "")
	origDir, _ := os.Getwd()
	err := os.Chdir(site.Dir)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.Chdir(origDir)
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

	var item map[string]any
	for _, item = range logItems {
		if item["level"] == "info" && item["msg"] != nil && strings.Contains(item["msg"].(string), "Your project can be reached at") {
			break
		}
	}
	assert.Contains(item["msg"], "Your project can be reached at")
}

// TestCmdRestartProfilesAndServices checks `ddev restart --profiles` and
// `ddev restart --service` with several services, one in a profile.
// See https://github.com/ddev/ddev/issues/7904
func TestCmdRestartProfilesAndServices(t *testing.T) {
	site := TestSites[0]
	origDir, _ := os.Getwd()
	err := os.Chdir(site.Dir)
	require.NoError(t, err)

	app, err := ddevapp.NewApp(site.Dir, false)
	require.NoError(t, err)
	composePath := app.GetConfigPath("docker-compose.restart-profiles.yaml")

	t.Cleanup(func() {
		_ = os.Chdir(origDir)
		_ = app.Stop(true, false)
		_ = os.RemoveAll(composePath)
		_ = app.Start()
	})

	err = os.WriteFile(composePath, []byte(`services:
  busybox1:
    image: busybox:stable
    init: true
    command: tail -f /dev/null
    profiles:
      - busybox
    container_name: ddev-${DDEV_SITENAME}-busybox1
`), 0644)
	require.NoError(t, err)

	out, err := exec.RunHostCommand(DdevBin, "restart", "--profiles=busybox")
	require.NoError(t, err, "output='%s'", out)
	require.Contains(t, out, "Started optional compose profiles 'busybox'")
	busybox, err := app.FindContainerByType("busybox1")
	require.NoError(t, err)
	require.NotNil(t, busybox)

	out, err = exec.RunHostCommand(DdevBin, "restart", "-s", "db,busybox1")
	require.NoError(t, err, "output='%s'", out)
	require.Contains(t, out, "Restarted db, busybox1 in "+site.Name)
	restartedBusybox, err := app.FindContainerByType("busybox1")
	require.NoError(t, err)
	require.NotNil(t, restartedBusybox)
	require.NotEqual(t, busybox.ID, restartedBusybox.ID)

	out, err = exec.RunHostCommand(DdevBin, "restart", "-s", "db", "--profiles=busybox")
	require.Error(t, err)
	require.Contains(t, out, "[profiles service] were all set")

	out, err = exec.RunHostCommand(DdevBin, "__complete", "restart", "-s", "db,")
	require.NoError(t, err)
	require.Contains(t, out, "db,busybox1")
	require.Contains(t, out, "db,web")
	require.NotContains(t, out, "db,db")
}
