package cmd

import (
	"fmt"
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/util"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/exec"
	asrt "github.com/stretchr/testify/assert"
)

// TestCmdExecBadArgs run `ddev exec` without the proper args
func TestCmdExecBadArgs(t *testing.T) {
	// Change to the first DevTestSite for the duration of this test.
	defer TestSites[0].Chdir()()
	assert := asrt.New(t)

	args := []string{"exec"}
	out, err := exec.RunCommand(DdevBin, args)
	assert.Error(err)
	assert.Contains(string(out), "Usage:")
}

// TestCmdExec runs a number of exec commands to verify behavior
func TestCmdExec(t *testing.T) {
	assert := asrt.New(t)
	origDir, err := os.Getwd()
	require.NoError(t, err)

	site := TestSites[0]
	err = os.Chdir(site.Dir)
	require.NoError(t, err)

	t.Cleanup(func() {
		err = os.Chdir(origDir)
		assert.NoError(err)
	})
	app, err := ddevapp.GetActiveApp(site.Name)
	assert.NoError(err)

	// Test default invocation
	out, err := exec.RunHostCommand(DdevBin, "exec", "pwd")
	assert.NoError(err)
	assert.Contains(out, "/var/www/html")

	// Test specifying service
	out, err = exec.RunHostCommand(DdevBin, "-s", "db", "exec", "pwd")
	assert.NoError(err)
	assert.Contains(out, "/")

	// Test specifying working directory
	out, err = exec.RunHostCommand(DdevBin, "exec", "-d", "/bin", "pwd")
	assert.NoError(err)
	assert.Contains(out, "/bin")

	// Test specifying service and working directory
	out, err = exec.RunHostCommand(DdevBin, "exec", "-s", "db", "-d", "/var", "pwd")
	assert.NoError(err)
	assert.Contains(out, "/var")

	// Test with raw cmd
	out, err = exec.RunHostCommand(DdevBin, "exec", "--raw", "--", "ls", "/usr/local")
	assert.NoError(err)

	assert.True(strings.HasPrefix(out, "bin\netc\ngames"), "expected '%v' to start with 'bin\\netc\\games\\include'", out)

	// Test sudo
	out, err = exec.RunHostCommand(DdevBin, "exec", "sudo", "whoami")
	assert.NoError(err)
	assert.Contains(out, "root")

	// Test that an nonexistent working directory generates an error
	out, err = exec.RunHostCommand(DdevBin, "exec", "-d", "/does/not/exist", "pwd")
	assert.Error(err)
	assert.Contains(out, "no such file or directory")

	_, err = exec.RunHostCommand(DdevBin, "exec", "ls >/var/www/html/TestCmdExec-${OSTYPE}.txt")
	assert.NoError(err)
	err = app.MutagenSyncFlush()
	assert.NoError(err)
	assert.FileExists(filepath.Join(site.Dir, "TestCmdExec-linux-gnu.txt"))

	_, err = exec.RunHostCommand(DdevBin, "exec", "ls >/dev/null && touch /var/www/html/TestCmdExec-touch-all-in-one.txt")
	assert.NoError(err)

	// We just created in the container, must flush before looking for it on host
	err = app.MutagenSyncFlush()
	assert.NoError(err)

	assert.FileExists(filepath.Join(site.Dir, "TestCmdExec-touch-all-in-one.txt"))

	_, err = exec.RunHostCommand(DdevBin, "exec", "true", "&&", "touch", "/var/www/html/TestCmdExec-touch-separate-args.txt")
	assert.NoError(err)

	err = app.MutagenSyncFlush()
	assert.NoError(err)

	assert.FileExists(filepath.Join(site.Dir, "TestCmdExec-touch-separate-args.txt"))

	bashPath := util.FindBashPath()
	// Make sure we can pipe things into ddev exec and have them work in stdin inside container
	filename := t.Name() + "_junk.txt"
	ddevBinForBash := dockerutil.MassageWindowsHostMountpoint(DdevBin)
	_, err = exec.RunHostCommand(bashPath, "-c", fmt.Sprintf(`printf "This file was piped into ddev exec" | %s exec "cat >/var/www/html/%s"`, ddevBinForBash, filename))
	assert.NoError(err)
	err = app.MutagenSyncFlush()
	assert.NoError(err)
	require.FileExists(t, filepath.Join(site.Dir, filename))

	content, err := os.ReadFile(filepath.Join(site.Dir, filename))
	assert.NoError(err)
	assert.Equal("This file was piped into ddev exec", string(content))

	// Make sure that redirection of output from ddev exec works
	f, err := os.CreateTemp("", "")
	err = f.Close()
	assert.NoError(err)
	defer os.Remove(f.Name()) // nolint: errcheck

	bashTempName := dockerutil.MassageWindowsHostMountpoint(f.Name())
	_, err = exec.RunHostCommand(bashPath, "-c", fmt.Sprintf("%s exec ls -l //usr/local/bin/composer >%s", ddevBinForBash, bashTempName))

	out, err = fileutil.ReadFileIntoString(f.Name())
	assert.NoError(err)
	assert.Contains(out, "/usr/local/bin/composer")
}
