package cmd

import (
	"path/filepath"
	"testing"

	"os"

	"io/ioutil"

	"github.com/drud/ddev/pkg/testcommon"
	"github.com/drud/drud-go/utils/network"
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

		confByte := []byte("<?php trigger_error(\"Fatal error\", E_USER_ERROR);")
		err := ioutil.WriteFile(filepath.Join(v.Dir, "docroot", "index.php"), confByte, 0644)
		assert.NoError(err)

		o := network.NewHTTPOptions("http://127.0.0.1/index.php")
		o.ExpectedStatus = 500
		o.Timeout = 30
		o.Headers["Host"] = v.Name + ".ddev.local"
		err = network.EnsureHTTPStatus(o)
		assert.NoError(err)

		args := []string{"logs"}
		out, err := system.RunCommand(DdevBin, args)

		assert.NoError(err)
		assert.Contains(string(out), "Server started")
		assert.Contains(string(out), "PHP message: PHP Stack trace:")

		cleanup()
	}
}
