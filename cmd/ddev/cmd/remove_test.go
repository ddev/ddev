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

// TestCmdRemove runs `ddev rm` on the test apps
func TestCmdRemove(t *testing.T) {
	assert := asrt.New(t)

	// Make sure we have running sites.
	err := addSites()
	require.NoError(t, err)
	for _, site := range DevTestSites {
		cleanup := site.Chdir()

		out, err := exec.RunCommand(DdevBin, []string{"remove"})
		assert.NoError(err, "ddev remove should succeed but failed, err: %v, output: %s", err, out)
		assert.Contains(out, "has been removed")

		// Ensure the site that was just stopped does not appear in the list of sites
		apps := ddevapp.GetApps()
		for _, app := range apps {
			assert.True(app.GetName() != site.Name)
		}

		cleanup()
	}

	// Re-create running sites.
	err = addSites()
	require.NoError(t, err)

	// Ensure the --all option can remove all active apps
	out, err := exec.RunCommand(DdevBin, []string{"remove", "--all"})
	assert.NoError(err, "ddev remove --all should succeed but failed, err: %v, output: %s", err, out)
	out, err = exec.RunCommand(DdevBin, []string{"list"})
	assert.NoError(err)
	assert.Contains(out, "no active ddev projects")
	assert.Equal(0, len(ddevapp.GetApps()), "Not all apps were removed after ddev remove --all")

	// Now put the sites back together so other tests can use them.
	err = addSites()
	require.NoError(t, err)
}

// TestCmdRemoveMissingProjectDirectory ensures the `ddev remove` command can operate on a project when the
// project's directory has been removed.
func TestCmdRemoveMissingProjectDirectory(t *testing.T) {
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
	assert.NoError(err)

	err = os.Chdir(projDir)
	assert.NoError(err)

	copyDir := filepath.Join(testcommon.CreateTmpDir(t.Name()), util.RandString(4))
	err = os.Rename(tmpDir, copyDir)
	assert.NoError(err)

	out, err = exec.RunCommand(DdevBin, []string{"remove", projectName})
	assert.NoError(err)
	assert.Contains(out, "has been removed")

	err = os.Rename(copyDir, tmpDir)
	assert.NoError(err)
}
