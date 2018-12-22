package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"

	"os"

	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/testcommon"
	"github.com/drud/ddev/pkg/util"
	asrt "github.com/stretchr/testify/assert"
)

// TestDevRemove runs `ddev rm` on the test apps
func TestDevRemove(t *testing.T) {
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
	// Ensure a user can't accidentally wipe out everything.
	appsBefore := len(ddevapp.GetApps())
	out, err := exec.RunCommand(DdevBin, []string{"remove", "--remove-data", "--all"})
	assert.Error(err, "ddev remove --all --remove-data should error, but succeeded")
	assert.Contains(out, "Illegal option")
	assert.EqualValues(appsBefore, len(ddevapp.GetApps()), "No apps should be removed or added after ddev remove --all --remove-data")

	// Ensure the --all option can remove all active apps
	out, err = exec.RunCommand(DdevBin, []string{"remove", "--all"})
	assert.NoError(err, "ddev remove --all should succeed but failed, err: %v, output: %s", err, out)
	out, err = exec.RunCommand(DdevBin, []string{"list"})
	assert.NoError(err)
	assert.Contains(out, "no active ddev projects")
	assert.Equal(0, len(ddevapp.GetApps()), "Not all apps were removed after ddev remove --all")

	// Now put the sites back together so other tests can use them.
	err = addSites()
	require.NoError(t, err)
}

// TestDdevRemoveMissingProjectDirectory ensures the `ddev remove` command can operate on a project when the
// project's directory has been removed.
func TestDdevRemoveMissingProjectDirectory(t *testing.T) {
	var err error
	var out string
	assert := asrt.New(t)

	projectName := util.RandString(6)

	tmpDir := testcommon.CreateTmpDir(t.Name())
	defer testcommon.Chdir(tmpDir)()

	_, err = exec.RunCommand(DdevBin, []string{"config", "--project-type", "php", "--project-name", projectName})
	assert.NoError(err)

	_, err = exec.RunCommand(DdevBin, []string{"start"})
	assert.NoError(err)

	err = os.RemoveAll(tmpDir)
	assert.NoError(err)

	out, err = exec.RunCommand(DdevBin, []string{"remove", projectName})
	assert.NoError(err)
	assert.Contains(out, "has been removed")
}
