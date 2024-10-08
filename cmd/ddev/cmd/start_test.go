package cmd

import (
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/stretchr/testify/require"
	"slices"
	"testing"

	"os"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/testcommon"
	asrt "github.com/stretchr/testify/assert"
)

// TestCmdStart runs `ddev start` on the test apps
func TestCmdStart(t *testing.T) {
	assert := asrt.New(t)

	// Gather reporting about goroutines at exit
	_ = os.Setenv("DDEV_GOROUTINES", "true")
	// Make sure we have running sites.
	err := addSites()
	require.NoError(t, err)

	// Stop all sites.
	out, err := exec.RunCommand(DdevBin, []string{"stop", "--all"})
	assert.NoError(err)
	testcommon.CheckGoroutineOutput(t, out)

	apps := []*ddevapp.DdevApp{}
	for _, testSite := range TestSites {
		app, err := ddevapp.NewApp(testSite.Dir, false)
		require.NoError(t, err)
		apps = append(apps, app)
	}

	// Build start command startMultipleArgs
	startMultipleArgs := []string{"start", "-y"}
	for _, app := range apps {
		startMultipleArgs = append(startMultipleArgs, app.GetName())
	}

	// Start multiple projects in one command
	out, err = exec.RunCommand(DdevBin, startMultipleArgs)
	assert.NoError(err, "ddev start with multiple project names should have succeeded, but failed, err: %v, output %s", err, out)
	testcommon.CheckGoroutineOutput(t, out)
	// If we omit the router, we should see the 127.0.0.1 URL.
	// Whether we have the router or not is not up to us, since it is omitted on gitpod and codespaces.
	if slices.Contains(globalconfig.DdevGlobalConfig.OmitContainersGlobal, "ddev-router") {
		// Assert that the output contains the 127.0.0.1 URL
		assert.Contains(out, "127.0.0.1", "The output should contain the 127.0.0.1 URL, but it does not: %s", out)
	}
	// Confirm all sites are running
	for _, app := range apps {
		status, statusDesc := app.SiteStatus()
		assert.Equal(ddevapp.SiteRunning, status, "All sites should be running, but project=%s status=%s statusDesc=%s", app.GetName(), status, statusDesc)
		assert.Equal(ddevapp.SiteRunning, statusDesc, `The status description should be "running", but project %s status  is: %s`, app.GetName(), statusDesc)
		if len(globalconfig.DdevGlobalConfig.OmitContainersGlobal) == 0 {
			assert.Contains(out, app.GetPrimaryURL(), "The output should contain the primary URL, but it does not: %s", out)
		}
	}
}
