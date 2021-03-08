package cmd

import (
	"github.com/stretchr/testify/require"
	"os"
	"runtime"
	"testing"

	"path/filepath"

	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/testcommon"
	"github.com/drud/ddev/pkg/util"
	asrt "github.com/stretchr/testify/assert"
)

// TestCmdPauseContainers runs `ddev pause` on the test apps
func TestCmdPauseContainers(t *testing.T) {
	assert := asrt.New(t)

	// Make sure we have running sites.
	err := addSites()
	require.NoError(t, err)

	site := TestSites[0]
	cleanup := site.Chdir()

	out, err := exec.RunCommand(DdevBin, []string{"pause"})
	assert.NoError(err, "ddev pause should succeed but failed, err: %v, output: %s", err, out)
	assert.Contains(out, "has been paused")

	apps := ddevapp.GetActiveProjects()

	for _, app := range apps {
		if app.GetName() != site.Name {
			continue
		}

		assert.True(app.SiteStatus() == ddevapp.SitePaused)
	}

	cleanup()

	// Re-create running sites.
	err = addSites()
	require.NoError(t, err)
	out, err = exec.RunCommand(DdevBin, []string{"pause", "--all"})
	assert.NoError(err, "ddev pause --all should succeed but failed, err: %v, output: %s", err, out)

	// Confirm all sites are stopped.
	apps = ddevapp.GetActiveProjects()
	for _, app := range apps {
		assert.True(app.SiteStatus() == ddevapp.SitePaused, "All sites should be stopped, but %s status: %s", app.GetName(), app.SiteStatus())
	}

	// Now put the sites back together so other tests can use them.
	err = addSites()
	require.NoError(t, err)
}

// TestCmdPauseContainersMissingProjectDirectory ensures the `ddev pause` command returns the expected help text when
// a project's directory no longer exists.
func TestCmdPauseContainersMissingProjectDirectory(t *testing.T) {
	var err error
	var out string

	if runtime.GOOS == "windows" {
		t.Skip("Skipping because unreliable on Windows")
	}

	assert := asrt.New(t)

	projDir, _ := os.Getwd()

	projectName := util.RandString(6)

	tmpDir := testcommon.CreateTmpDir(t.Name())
	defer testcommon.CleanupDir(tmpDir)
	defer testcommon.Chdir(tmpDir)()

	_, err = exec.RunCommand(DdevBin, []string{"config", "--project-type", "php", "--project-name", projectName})
	assert.NoError(err)

	_, err = exec.RunCommand(DdevBin, []string{"start", "-y"})
	assert.NoError(err)

	//nolint: errcheck
	defer exec.RunCommand(DdevBin, []string{"stop", "-RO", projectName})

	_, err = exec.RunCommand(DdevBin, []string{"pause", projectName})
	assert.NoError(err)

	err = os.Chdir(projDir)
	assert.NoError(err)

	copyDir := filepath.Join(testcommon.CreateTmpDir(t.Name()), util.RandString(4))
	err = os.Rename(tmpDir, copyDir)
	assert.NoError(err)

	out, err = exec.RunCommand(DdevBin, []string{"pause", projectName})
	assert.Error(err, "Expected an error when pausing project with no project directory")
	assert.Contains(out, "ddev can no longer find your project files")
}
