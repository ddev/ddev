package cmd

import (
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/exec"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

// TestCmdSnapshotRestore runs `ddev snapshot restore` on the test apps
func TestCmdSnapshotRestore(t *testing.T) {
	assert := asrt.New(t)

	site := TestSites[0]
	cleanup := site.Chdir()

	app, err := ddevapp.NewApp(site.Dir, false)
	assert.NoError(err)

	t.Cleanup(func() {
		// Make sure all databases are back to default empty
		_ = app.Stop(true, false)
		cleanup()
	})

	err = app.Start()
	require.NoError(t, err)

	// Ensure that a snapshot is created
	args := []string{"snapshot", "--name", "test-snapshot"}
	out, err := exec.RunCommand(DdevBin, args)
	assert.NoError(err)
	assert.Contains(out, "Created database snapshot test-snapshot")

	// Try interactive command
	// Doesn't seem to work without pty, 2021-12-14
	//if runtime.GOOS != "windows" {
	//	out, err = exec.RunCommand("bash", []string{"-c", "echo -nq '\n' | " + DdevBin + " snapshot restore"})
	//	assert.NoError(err)
	//	assert.Contains(out, "Restored database snapshot")
	//}

	// Ensure that latest snapshot can be restored
	out, err = exec.RunHostCommand(DdevBin, "snapshot", "restore", "--latest")
	assert.NoError(err)
	assert.Contains(out, "Restored database snapshot")
}
