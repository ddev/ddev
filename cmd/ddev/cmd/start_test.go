package cmd

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"os"

	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/testcommon"
	"github.com/drud/ddev/pkg/util"
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

// TestDdevStartMissingProjectDirectory ensures the `ddev start` command returns the expected help text when
// a project's directory no longer exists.
func TestDdevStartMissingProjectDirectory(t *testing.T) {
	var err error
	var out string
	assert := asrt.New(t)

	projectName := util.RandString(6)

	tmpDir := testcommon.CreateTmpDir(t.Name())
	defer testcommon.CleanupDir(tmpDir)
	defer testcommon.Chdir(tmpDir)()

	_, err = exec.RunCommand(DdevBin, []string{"config", "--project-type", "php", "--project-name", projectName})
	assert.NoError(err)

	_, err = exec.RunCommand(DdevBin, []string{"start"})

	//nolint: errcheck
	defer exec.RunCommand(DdevBin, []string{"remove", "-RO", projectName})
	assert.NoError(err)

	copyDir := filepath.Join(testcommon.CreateTmpDir(t.Name()), util.RandString(4))
	err = os.Rename(tmpDir, copyDir)
	defer testcommon.CleanupDir(copyDir)
	defer testcommon.Chdir(copyDir)()
	assert.NoError(err)

	out, err = exec.RunCommand(DdevBin, []string{"start", projectName})
	assert.Error(err, "Expected an error when starting project with no project directory")
	assert.Contains(out, "ddev can no longer find your project files")
}
