package cmd

import (
	"testing"

	"os"

	"github.com/drud/ddev/pkg/testcommon"
	"github.com/drud/drud-go/utils/system"
	"github.com/stretchr/testify/assert"
)

func TestDevLogsBadArgs(t *testing.T) {
	assert := assert.New(t)

	testDir := testcommon.CreateTmpDir("no-valid-ddev-config")

	err := os.Chdir(testDir)
	if err != nil {
		t.Skip("Could not change to temporary directory %s: %v", testDir, err)
	}

	args := []string{"logs"}
	out, err := system.RunCommand(DdevBin, args)
	assert.Error(err)
	assert.Contains(string(out), "unable to determine the application for this command")
}

// TestDevLogs tests that the Dev logs functionality is working.
func TestDevLogs(t *testing.T) {
	assert := assert.New(t)

	for _, v := range DevTestSites {
		cleanup := v.Chdir()

		args := []string{"logs"}
		out, err := system.RunCommand(DdevBin, args)
		assert.NoError(err)
		assert.Contains(string(out), "Server started")

		cleanup()
	}
}
