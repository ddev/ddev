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
)

// TestDevLogsNoConfig tests what happens with when running "ddev logs" when
// the directory has not been configured (and no project name is given)
func TestDevLogsNoConfig(t *testing.T) {
	assert := asrt.New(t)
	testDir := testcommon.CreateTmpDir("no-valid-ddev-config")
	defer testcommon.CleanupDir(testDir)
	defer testcommon.Chdir(testDir)()

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
		err = fileutil.CopyFile(filepath.Join(pwd, "testdata", "logtest.php"), filepath.Join(v.Dir, v.Docroot, "logtest.php"))
		assert.NoError(err)
		cleanup := v.Chdir()

		url := "http://" + v.Name + "." + version.DDevTLD + "/logtest.php"
		out, err := testcommon.GetLocalHTTPResponse(t, url)
		assert.NoError(err)

		args := []string{"logs"}
		out, err = exec.RunCommand(DdevBin, args)

		assert.NoError(err)
		assert.Contains(string(out), "Server started")
		assert.Contains(string(out), "Notice to demonstrate logging", "PHP notice not found for project %s output='%s", v.Name, string(out))
		cleanup()
	}
}
