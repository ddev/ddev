package cmd

import (
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/stretchr/testify/require"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/exec"
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

		// Ensure that the stopped site does not appear in the list of sites
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
	assert.Contains(out, "No DDEV projects were found.")

	_, err = exec.RunHostCommand(DdevBin, "stop", "--all", "--stop-ssh-agent")
	assert.NoError(err)
	sshAgent, err := dockerutil.FindContainerByName("ddev-ssh-agent")
	assert.NoError(err)
	// ssh-agent should be gone
	assert.Nil(sshAgent)
}
