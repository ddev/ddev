package cmd

import (
	"github.com/stretchr/testify/require"
	"testing"

	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/exec"
	asrt "github.com/stretchr/testify/assert"
)

// TestDdevStart runs `ddev start` on the test apps
func TestDdevStart(t *testing.T) {
	assert := asrt.New(t)

	// Make sure we have running sites.
	err := addSites()
	require.NoError(t, err)

	// Stop all sites.
	_, err = exec.RunCommand(DdevBin, []string{"stop", "--all"})
	assert.NoError(err)

	// Ensure all sites are started after ddev start --all.
	out, err := exec.RunCommand(DdevBin, []string{"start", "--all"})
	assert.NoError(err, "ddev start --all should succeed but failed, err: %v, output: %s", err, out)

	// Confirm all sites are running.
	apps := ddevapp.GetApps()
	for _, app := range apps {
		assert.True(app.SiteStatus() == ddevapp.SiteRunning, "All sites should be running, but %s status: %s", app.GetName(), app.SiteStatus())
	}

	// Stop all sites.
	_, err = exec.RunCommand(DdevBin, []string{"stop", "--all"})
	assert.NoError(err)

	// Build start command startMultipleArgs
	startMultipleArgs := []string{"start"}
	for _, app := range apps {
		startMultipleArgs = append(startMultipleArgs, app.GetName())
	}

	// Start multiple projects in one command
	out, err = exec.RunCommand(DdevBin, startMultipleArgs)
	assert.NoError(err, "ddev start with multiple project names should have succeeded, but failed, err: %v, output %s", err, out)

	// Confirm all sites are running
	for _, app := range apps {
		assert.True(app.SiteStatus() == ddevapp.SiteRunning, "All sites should be running, but %s status: %s", app.GetName(), app.SiteStatus())
	}
}
