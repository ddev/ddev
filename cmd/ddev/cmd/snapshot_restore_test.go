package cmd

import (
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/exec"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

// TestCmdSnapshotRestore runs `ddev snapshot restore` on the test apps
func TestCmdSnapshotRestore(t *testing.T) {
	assert := asrt.New(t)

	origDir, _ := os.Getwd()
	site := TestSites[0]
	err := os.Chdir(site.Dir)
	assert.NoError(err)
	app, err := ddevapp.NewApp(site.Dir, false)
	require.NoError(t, err)

	t.Cleanup(func() {
		// Make sure all databases are back to default empty
		err = app.Stop(true, false)
		assert.NoError(err)
		err = os.Chdir(origDir)
		assert.NoError(err)
		err = os.RemoveAll(app.GetConfigPath("db_snapshots"))
		assert.NoError(err)
	})

	err = app.Restart()
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
	assert.Contains(out, "Database snapshot test-snapshot was restored")
}
