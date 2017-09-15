package cmd

import (
	"path/filepath"
	"testing"

	"os"

	"io/ioutil"

	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/testcommon"
	"github.com/drud/ddev/pkg/util"
	asrt "github.com/stretchr/testify/assert"
)

func TestDevLogsBadArgs(t *testing.T) {
	assert := asrt.New(t)

	testDir := testcommon.CreateTmpDir("no-valid-ddev-config")

	err := os.Chdir(testDir)
	if err != nil {
		t.Skip("Could not change to temporary directory %s: %v", testDir, err)
	}

	args := []string{"logs"}
	out, err := exec.RunCommand(DdevBin, args)
	assert.Error(err)
	assert.Contains(string(out), "could not find containers which matched search criteria")
}

// TestDevLogs tests that the Dev logs functionality is working.
func TestDevLogs(t *testing.T) {
	assert := asrt.New(t)

	for _, v := range DevTestSites {
		cleanup := v.Chdir()

		confByte := []byte("<?php trigger_error(\"Fatal error\", E_USER_ERROR);")
		err := ioutil.WriteFile(filepath.Join(v.Dir, v.DocrootBase, "index.php"), confByte, 0644)
		assert.NoError(err)

		o := util.NewHTTPOptions("http://127.0.0.1/index.php")
		o.ExpectedStatus = 500
		o.Timeout = 30
		o.Headers["Host"] = v.Name + ".ddev.local"
		err = util.EnsureHTTPStatus(o)
		assert.NoError(err)

		args := []string{"logs"}
		out, err := exec.RunCommand(DdevBin, args)

		assert.NoError(err)
		assert.Contains(string(out), "Server started")
		assert.Contains(string(out), "PHP message: PHP Stack trace:")

		cleanup()
	}
}
