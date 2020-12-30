package cmd

import (
	"fmt"
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/exec"
	asrt "github.com/stretchr/testify/assert"
	"os"
	"testing"
)

// TestCmdSnapshot runs `ddev snapshot` on the test apps
func TestCmdSnapshot(t *testing.T) {
	assert := asrt.New(t)

	testDir, _ := os.Getwd()
	fmt.Println(testDir)
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

	// Ensure that a snapshot can be created
	args := []string{"snapshot", "--name", "test-snapshot"}
	out, err := exec.RunCommand(DdevBin, args)
	assert.NoError(err)
	assert.Contains(string(out), "Created snapshot test-snapshot")

	// Try to delete a not existing snapshot
	args = []string{"snapshot", "--name", "not-existing-snapshot", "--cleanup", "--yes"}
	out, err = exec.RunCommand(DdevBin, args)
	assert.Error(err)
	assert.Contains(string(out), "Failed to delete snapshot")

	// Ensure that an existing snapshot can be deleted
	args = []string{"snapshot", "--name", "test-snapshot", "--cleanup"}
	out, err = exec.RunCommand(DdevBin, args)
	assert.NoError(err)
	assert.Contains(string(out), "Deleted database snapshot test-snapshot")
}
