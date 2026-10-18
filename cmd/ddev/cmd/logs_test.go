package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/testcommon"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	// Gather reporting about goroutines at exit
	t.Setenv("DDEV_GOROUTINES", "true")

	site := TestSites[0]
	pwd, err := os.Getwd()
	require.NoError(t, err)

	err = os.Chdir(site.Dir)
	require.NoError(t, err)

	logtestFilePath := filepath.Join(site.Dir, site.Docroot, "logtest.php")
	err = fileutil.CopyFile(filepath.Join(pwd, "testdata", t.Name(), "logtest.php"), logtestFilePath)
	require.NoError(t, err)

	app, err := ddevapp.NewApp(site.Dir, true)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = os.Chdir(pwd)
		_ = os.Remove(logtestFilePath)
	})
	// We have to sync or our logtest.php may not yet be available inside container
	err = app.MutagenSyncFlush()
	require.NoError(t, err)

	// The project may have just been restarted by an earlier test, and the
	// router can briefly reset connections while it picks up the new routes.
	url := app.GetPrimaryURL() + "/logtest.php"
	testcommon.RequireLocalHTTPContent(t, url, "Notice to demonstrate logging",
		testcommon.WithMessagef("logtest.php should emit the PHP notice that `ddev logs` surfaces"),
		testcommon.WithTimeout(10*time.Second),
		testcommon.WithMaxAttempts(5),
	)

	out, err := exec.RunHostCommand(DdevBin, "logs")
	require.NoError(t, err)
	testcommon.CheckGoroutineOutput(t, out)
	for _, want := range []string{"Server started", "Notice to demonstrate logging"} {
		if !strings.Contains(out, want) {
			t.Errorf("`ddev logs` for project %s does not contain %q; last lines:\n%s", site.Name, want, lastLines(out, 20))
		}
	}
}

// lastLines returns at most n trailing lines of s.
func lastLines(s string, n int) string {
	lines := strings.Split(strings.TrimRight(s, "\r\n"), "\n")
	return strings.Join(lines[max(0, len(lines)-n):], "\n")
}
