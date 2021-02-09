package cmd

import (
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/fileutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/testcommon"
	asrt "github.com/stretchr/testify/assert"
)

// TestLogsNoConfig tests what happens with when running "ddev logs" when
// the directory has not been configured (and no project name is given)
func TestLogsNoConfig(t *testing.T) {
	assert := asrt.New(t)
	testDir := testcommon.CreateTmpDir("no-valid-ddev-config")
	defer testcommon.CleanupDir(testDir)
	defer testcommon.Chdir(testDir)()

	args := []string{"logs"}
	out, err := exec.RunCommand(DdevBin, args)
	assert.Error(err)
	assert.Contains(string(out), "Please specify a project name or change directories")
}

// TestLogs tests that the ddev logs functionality is working.
func TestLogs(t *testing.T) {
	assert := asrt.New(t)

	v := TestSites[0]
	// Copy our fatal error php into the docroot of testsite.
	pwd, err := os.Getwd()
	assert.NoError(err)
	err = fileutil.CopyFile(filepath.Join(pwd, "testdata", "logtest.php"), filepath.Join(v.Dir, v.Docroot, "logtest.php"))
	assert.NoError(err)
	cleanup := v.Chdir()

	app, err := ddevapp.NewApp(v.Dir, true)
	assert.NoError(err)

	url := "http://" + v.Name + "." + app.ProjectTLD + "/logtest.php"
	_, err = testcommon.EnsureLocalHTTPContent(t, url, "Notice to demonstrate logging", 5)
	assert.NoError(err)

	args := []string{"logs"}
	out, err := exec.RunCommand(DdevBin, args)

	assert.NoError(err)
	assert.Contains(string(out), "Server started")
	assert.Contains(string(out), "Notice to demonstrate logging", "PHP notice not found for project %s output='%s", v.Name, string(out))
	cleanup()
}
