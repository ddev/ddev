package cmd

import (
	"os"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/testcommon"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCmdSnapshotRestore runs `ddev snapshot restore` on the test apps
func TestCmdSnapshotRestore(t *testing.T) {
	assert := asrt.New(t)
	// Gather reporting about goroutines at exit
	_ = os.Setenv("DDEV_GOROUTINES", "true")

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
		_ = os.RemoveAll(app.GetConfigPath("db_snapshots"))
	})

	err = app.Restart()
	require.NoError(t, err)

	// Ensure that a snapshot is created
	args := []string{"snapshot", "--name", "test-snapshot"}
	out, err := exec.RunCommand(DdevBin, args)
	assert.NoError(err)
	assert.Contains(out, "Created database snapshot test-snapshot")
	testcommon.CheckGoroutineOutput(t, out)

	// Ensure that latest snapshot can be restored
	out, err = exec.RunHostCommand(DdevBin, "snapshot", "restore", "--latest")
	assert.NoError(err)
	assert.Contains(out, "Database snapshot test-snapshot was restored")
	testcommon.CheckGoroutineOutput(t, out)
}
