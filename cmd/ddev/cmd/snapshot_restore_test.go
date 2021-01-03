package cmd

import (
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/exec"
	asrt "github.com/stretchr/testify/assert"
	"testing"
)

// TestCmdSnapshotRestore runs `ddev snapshot restore` on the test apps
func TestCmdSnapshotRestore(t *testing.T) {
	assert := asrt.New(t)

	site := TestSites[0]
	cleanup := site.Chdir()

	app, err := ddevapp.NewApp(site.Dir, false, "")
	assert.NoError(err)

	defer func() {
		// Make sure all databases are back to default empty
		_ = app.Stop(true, false)
		_ = app.Start()
		cleanup()
	}()

	// Ensure that a snapshot is created
	args := []string{"snapshot", "--name", "test-snapshot"}
	out, err := exec.RunCommand(DdevBin, args)
	assert.NoError(err)
	assert.Contains(out, "Created snapshot test-snapshot")

	// Ensure that a snapshot can be restored
	args = []string{"snapshot", "restore", "test-snapshot"}
	out, err = exec.RunCommand(DdevBin, args)
	assert.NoError(err)
	assert.Contains(out, "Restored database snapshot")

	// Ensure that latest snapshot can be restored
	args = []string{"snapshot", "restore", "--latest"}
	out, err = exec.RunCommand(DdevBin, args)
	assert.NoError(err)
	assert.Contains(out, "Restored database snapshot")
}
