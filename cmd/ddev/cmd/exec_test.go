package cmd

import (
	"fmt"
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"

	"github.com/drud/ddev/pkg/exec"
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
	site := TestSites[0]
	cleanup := site.Chdir()
	defer cleanup()

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

	// Make sure we can pipe things into ddev exec and have them work in stdin inside container
	filename := t.Name() + "_junk.txt"
	_, err = exec.RunHostCommand("sh", "-c", fmt.Sprintf("printf 'This file was piped into ddev exec' | %s exec 'cat >/var/www/html/%s'", DdevBin, filename))
	assert.NoError(err)
	require.FileExists(t, filepath.Join(site.Dir, filename))

	content, err := os.ReadFile(filepath.Join(site.Dir, filename))
	assert.NoError(err)
	assert.Equal("This file was piped into ddev exec", string(content))
}
