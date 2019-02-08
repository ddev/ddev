package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"

	"os"

	"path/filepath"

	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/testcommon"
	"github.com/drud/ddev/pkg/util"
	asrt "github.com/stretchr/testify/assert"
)

// TestCmdStop runs `ddev stop` on the test apps
func TestCmdStop(t *testing.T) {
	assert := asrt.New(t)

	// Make sure we have running sites.
	err := addSites()
	require.NoError(t, err)

	for _, site := range DevTestSites {
		cleanup := site.Chdir()

		out, err := exec.RunCommand(DdevBin, []string{"stop"})
		assert.NoError(err, "ddev stop should succeed but failed, err: %v, output: %s", err, out)
		assert.Contains(out, "has been stopped")

		apps := ddevapp.GetApps()
		for _, app := range apps {
			if app.GetName() != site.Name {
				continue
			}

			assert.True(app.SiteStatus() == ddevapp.SiteStopped)
		}

		cleanup()
	}

	// Re-create running sites.
	err = addSites()
	require.NoError(t, err)
	out, err := exec.RunCommand(DdevBin, []string{"stop", "--all"})
	assert.NoError(err, "ddev stop --all should succeed but failed, err: %v, output: %s", err, out)

	// Confirm all sites are stopped.
	apps := ddevapp.GetApps()
	for _, app := range apps {
		assert.True(app.SiteStatus() == ddevapp.SiteStopped, "All sites should be stopped, but %s status: %s", app.GetName(), app.SiteStatus())
	}

	// Now put the sites back together so other tests can use them.
	err = addSites()
	require.NoError(t, err)
}

// TestCmdStopMissingProjectDirectory ensures the `ddev stop` command returns the expected help text when
// a project's directory no longer exists.
func TestCmdStopMissingProjectDirectory(t *testing.T) {
	var err error
	var out string
	assert := asrt.New(t)

	projDir, _ := os.Getwd()

	projectName := util.RandString(6)

	tmpDir := testcommon.CreateTmpDir(t.Name())
	defer testcommon.CleanupDir(tmpDir)
	defer testcommon.Chdir(tmpDir)()

	_, err = exec.RunCommand(DdevBin, []string{"config", "--project-type", "php", "--project-name", projectName})
	assert.NoError(err)

	_, err = exec.RunCommand(DdevBin, []string{"start"})
	//nolint: errcheck
	defer exec.RunCommand(DdevBin, []string{"remove", projectName})
	assert.NoError(err)

	err = os.Chdir(projDir)
	assert.NoError(err)

	copyDir := filepath.Join(testcommon.CreateTmpDir(t.Name()), util.RandString(4))
	err = os.Rename(tmpDir, copyDir)
	assert.NoError(err)

	out, err = exec.RunCommand(DdevBin, []string{"stop", projectName})
	assert.Error(err, "Expected an error when stopping project with no project directory")
	assert.Contains(out, "ddev can no longer find your project files")
}
