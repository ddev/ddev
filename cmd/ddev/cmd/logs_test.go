package cmd

import (
	"io/ioutil"
	"log"
	"testing"

	"os"

	"github.com/drud/drud-go/utils/system"
	"github.com/stretchr/testify/assert"
)

func TestDevLogsBadArgs(t *testing.T) {
	assert := assert.New(t)

	testDir, err := ioutil.TempDir("", "no-valid-ddev-config")
	if err != nil {
		log.Fatalf("Could not create temporary directory %s ", testDir)
	}

	err = os.Chdir(testDir)
	if err != nil {
		t.Skip("Could not change to temporary directory %s: %v", testDir, err)
	}

	args := []string{"logs"}
	out, err := system.RunCommand(DdevBin, args)
	assert.Error(err)
	assert.Contains(string(out), "Unable to determine the application for this command")
}

// TestDevLogs tests that the Dev logs functionality is working.
func TestDevLogs(t *testing.T) {
	if skipComposeTests {
		t.Skip("Compose tests being skipped.")
	}
	assert := assert.New(t)

	for _, v := range DevTestSites {
		cleanup := v.Chdir()

		args := []string{"logs"}
		out, err := system.RunCommand(DdevBin, args)
		assert.NoError(err)
		assert.Contains(string(out), "Server started")
		assert.Contains(string(out), "GET")

		cleanup()
	}
}
