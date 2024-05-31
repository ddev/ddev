package cmd

import (
	"github.com/stretchr/testify/require"
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

	// Confirm all sites are running
	for _, app := range apps {
		status, statusDesc := app.SiteStatus()
		assert.Equal(ddevapp.SiteRunning, status, "All sites should be running, but project=%s status=%s statusDesc=%s", app.GetName(), status, statusDesc)
		assert.Equal(ddevapp.SiteRunning, statusDesc, `The status description should be "running", but project %s status  is: %s`, app.GetName(), statusDesc)
	}
}
