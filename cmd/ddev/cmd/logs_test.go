package cmd

import (
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/version"
	"path/filepath"
	"testing"

	"os"

	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/testcommon"
	asrt "github.com/stretchr/testify/assert"
	"time"
)

// TestDevLogsNoConfig tests what happens with when running "ddev logs" when
// the directory has not been configured (and no project name is given)
func TestDevLogsNoConfig(t *testing.T) {
	assert := asrt.New(t)

	testDir := testcommon.CreateTmpDir("no-valid-ddev-config")

	err := os.Chdir(testDir)
	if err != nil {
		t.Skipf("Could not change to temporary directory %s: %v", testDir, err)
	}

	args := []string{"logs"}
	out, err := exec.RunCommand(DdevBin, args)
	assert.Error(err)
	assert.Contains(string(out), "Please specify a project name or change directories")
}

// TestDevLogs tests that the Dev logs functionality is working.
func TestDevLogs(t *testing.T) {
	assert := asrt.New(t)

	for _, v := range DevTestSites {
		// Copy our fatal error php into the docroot of testsite.
		pwd, err := os.Getwd()
		assert.NoError(err)
		err = fileutil.CopyFile(filepath.Join(pwd, "testdata", "fatal.php"), filepath.Join(v.Dir, v.Docroot, "fatal.php"))
		assert.NoError(err)
		cleanup := v.Chdir()

		url := "http://" + v.Name + "." + version.DDevTLD + "/fatal.php"
		out, err := testcommon.GetLocalHTTPResponse(t, url)
		_ = out
		assert.NoError(err)
		// Because php display_errors = On the error results in a 200 anyway.

		// logs may not respond exactly right away, wait a tiny bit.
		time.Sleep(2 * time.Second)
		args := []string{"logs"}
		out, err = exec.RunCommand(DdevBin, args)

		assert.NoError(err)
		assert.Contains(string(out), "Server started")
		assert.Contains(string(out), "PHP Fatal error:", "PHP Fatal error not found for project %s output='%s", v.Name, string(out))
		cleanup()
	}
}
