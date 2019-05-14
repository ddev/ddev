package cmd

import (
	"github.com/drud/ddev/pkg/dockerutil"
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

		// Ensure the site that was just stopped does not appear in the list of sites
		apps := ddevapp.GetDockerProjects()
		for _, app := range apps {
			assert.True(app.GetName() != site.Name)
		}

		cleanup()
	}

	// Re-create running sites.
	err = addSites()
	require.NoError(t, err)

	// Ensure the --all option can remove all active apps
	out, err := exec.RunCommand(DdevBin, []string{"stop", "--all", "-RO"})
	assert.NoError(err, "ddev stop --all should succeed but failed, err: %v, output: %s", err, out)
	out, err = exec.RunCommand(DdevBin, []string{"list"})
	assert.NoError(err)
	assert.Contains(out, "no active ddev projects")
	containers, err := dockerutil.GetDockerContainers(true)
	assert.NoError(err)
	// Just the ddev-ssh-agent should remain running (1 container)
	assert.Equal(1, len(containers), "Not all projects were removed after ddev stop --all")
	_, err = exec.RunCommand(DdevBin, []string{"stop", "--all", "-RO", "--stop-ssh-agent"})
	assert.NoError(err)
	containers, err = dockerutil.GetDockerContainers(true)
	assert.NoError(err)
	// All containers should now be gone
	assert.Equal(0, len(containers))
}

// TestCmdStopMissingProjectDirectory ensures the `ddev stop` command can operate on a project when the
// project's directory has been removed.
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

	//nolint: errcheck
	defer exec.RunCommand(DdevBin, []string{"stop", "-RO", projectName})

	_, err = exec.RunCommand(DdevBin, []string{"start"})
	assert.NoError(err)

	err = os.Chdir(projDir)
	assert.NoError(err)

	copyDir := filepath.Join(testcommon.CreateTmpDir(t.Name()), util.RandString(4))
	err = os.Rename(tmpDir, copyDir)
	assert.NoError(err)

	out, err = exec.RunCommand(DdevBin, []string{"stop", projectName})
	assert.NoError(err)
	assert.Contains(out, "has been stopped")

	err = os.Rename(copyDir, tmpDir)
	assert.NoError(err)
}
