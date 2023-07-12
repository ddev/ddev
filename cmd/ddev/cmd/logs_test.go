package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/stretchr/testify/require"

	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/testcommon"
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

// TestCmdLogs tests that the ddev logs functionality is working.
func TestCmdLogs(t *testing.T) {
	//if nodeps.IsAppleSilicon() {
	//	t.Skip("Skipping on mac M1 to ignore problems with 'connection reset by peer'")
	//}
	assert := asrt.New(t)

	origDir, _ := os.Getwd()
	site := TestSites[0]
	// Copy our fatal error php into the docroot of testsite.
	pwd, err := os.Getwd()
	assert.NoError(err)

	err = os.Chdir(site.Dir)
	assert.NoError(err)

	logtestFilePath := filepath.Join(site.Dir, site.Docroot, "logtest.php")
	err = fileutil.CopyFile(filepath.Join(pwd, "testdata", t.Name(), "logtest.php"), logtestFilePath)
	assert.NoError(err)

	app, err := ddevapp.NewApp(site.Dir, true)
	assert.NoError(err)

	t.Cleanup(func() {
		err = os.Chdir(origDir)
		assert.NoError(err)
		err = os.Remove(logtestFilePath)
		assert.NoError(err)
	})
	// We have to sync or our logtest.php may not yet be available inside container
	err = app.MutagenSyncFlush()
	assert.NoError(err)

	url := app.GetPrimaryURL() + "/logtest.php"
	_, err = testcommon.EnsureLocalHTTPContent(t, url, "Notice to demonstrate logging", 10)
	assert.NoError(err)

	out, err := exec.RunHostCommand(DdevBin, "logs")
	require.NoError(t, err)
	assert.Contains(string(out), "Server started")
	assert.Contains(string(out), "Notice to demonstrate logging", "PHP notice not found for project %s output='%s", site.Name, string(out))
}
