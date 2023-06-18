package cmd

import (
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/stretchr/testify/require"
	"runtime"
	"testing"

	"os"

	"path/filepath"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/ddev/ddev/pkg/util"
	asrt "github.com/stretchr/testify/assert"
)

// TestCmdStop runs `ddev stop` on the test apps
func TestCmdStop(t *testing.T) {
	assert := asrt.New(t)

	t.Cleanup(func() {
		err := addSites()
		assert.NoError(err)
	})
	// Make sure we have running sites.
	err := addSites()
	require.NoError(t, err)
	for _, site := range TestSites {
		cleanup := site.Chdir()

		out, err := exec.RunHostCommand(DdevBin, "stop")
		assert.NoError(err, "ddev stop should succeed but failed, err: %v, output: %s", err, out)
		assert.Contains(out, "has been stopped")

		// Ensure the site that was just stopped does not appear in the list of sites
		apps := ddevapp.GetActiveProjects()
		for _, app := range apps {
			assert.True(app.GetName() != site.Name)
		}

		cleanup()
	}

	// Re-create running sites.
	err = addSites()
	require.NoError(t, err)

	// Ensure the --all option can remove all active apps
	out, err := exec.RunHostCommand(DdevBin, "stop", "--all")
	assert.NoError(err, "ddev stop --all should succeed but failed, err: %v, output: %s", err, out)

	out, err = exec.RunHostCommand(DdevBin, "list", "--active-only")
	assert.NoError(err)
	assert.Contains(out, "No ddev projects were found.")

	_, err = exec.RunHostCommand(DdevBin, "stop", "--all", "--stop-ssh-agent")
	assert.NoError(err)
	sshAgent, err := dockerutil.FindContainerByName("ddev-ssh-agent")
	assert.NoError(err)
	// ssh-agent should be gone
	assert.Nil(sshAgent)
}

// TestCmdStopMissingProjectDirectory ensures the `ddev stop` command can operate on a project when the
// project's directory has been removed.
func TestCmdStopMissingProjectDirectory(t *testing.T) {
	var err error
	var out string

	if runtime.GOOS == "windows" {
		t.Skip("Skipping because unreliable on Windows")
	}

	assert := asrt.New(t)
	origDir, _ := os.Getwd()

	projectName := util.RandString(6)
	tmpDir := testcommon.CreateTmpDir(t.Name())
	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, err = exec.RunHostCommand(DdevBin, "stop", "-RO", projectName)
		assert.NoError(err)
		err = os.Chdir(origDir)
		assert.NoError(err)

		err = os.RemoveAll(tmpDir)
		assert.NoError(err)
	})

	_, err = exec.RunHostCommand(DdevBin, "config", "--project-type", "php", "--project-name", projectName)
	assert.NoError(err)

	_, err = exec.RunHostCommand(DdevBin, "start", "-y")
	assert.NoError(err)

	_, err = exec.RunHostCommand(DdevBin, "stop", projectName)
	assert.NoError(err)

	err = os.Chdir(origDir)
	assert.NoError(err)

	copyDir := filepath.Join(testcommon.CreateTmpDir(t.Name()), util.RandString(4))
	err = os.Rename(tmpDir, copyDir)
	assert.NoError(err)
	//nolint: errcheck
	defer os.Rename(copyDir, tmpDir)

	out, err = exec.RunHostCommand(DdevBin, "stop", projectName)
	assert.NoError(err)
	assert.Contains(out, "has been stopped")

}
