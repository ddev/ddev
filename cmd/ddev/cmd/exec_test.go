package cmd

import (
	"path/filepath"
	"testing"

	"github.com/drud/ddev/pkg/exec"
	asrt "github.com/stretchr/testify/assert"
)

// TestCmdExecBadArgs run `ddev exec` without the proper args
func TestCmdExecBadArgs(t *testing.T) {
	// Change to the first DevTestSite for the duration of this test.
	defer DevTestSites[0].Chdir()()
	assert := asrt.New(t)

	args := []string{"exec"}
	out, err := exec.RunCommand(DdevBin, args)
	assert.Error(err)
	assert.Contains(string(out), "Usage:")
}

// TestCmdExec runs a number of exec commands to verify behavior
func TestCmdExec(t *testing.T) {

	assert := asrt.New(t)
	for _, v := range DevTestSites {
		cleanup := v.Chdir()

		// Test default invocation
		args := []string{"exec", "pwd"}
		out, err := exec.RunCommand(DdevBin, args)
		assert.NoError(err)
		assert.Contains(out, "/var/www/html")

		// Test specifying service
		args = []string{"-s", "db", "exec", "pwd"}
		out, err = exec.RunCommand(DdevBin, args)
		assert.NoError(err)
		assert.Contains(out, "/")

		// Test specifying working directory
		args = []string{"exec", "-d", "/bin", "pwd"}
		out, err = exec.RunCommand(DdevBin, args)
		assert.NoError(err)
		assert.Contains(out, "/bin")

		// Test specifying service and working directory
		args = []string{"exec", "-s", "db", "-d", "/var", "pwd"}
		out, err = exec.RunCommand(DdevBin, args)
		assert.NoError(err)
		assert.Contains(out, "/var")

		// Test sudo
		args = []string{"exec", "sudo", "whoami"}
		out, err = exec.RunCommand(DdevBin, args)
		assert.NoError(err)
		assert.Contains(out, "root")

		// Test that an nonexistant working directory generates an error
		args = []string{"exec", "-d", "/does/not/exist", "pwd"}
		out, err = exec.RunCommand(DdevBin, args)
		assert.Error(err)
		assert.Contains(out, "no such file or directory")

		args = []string{"exec", "ls >/var/www/html/TestCmdExec-${OSTYPE}.txt"}
		out, err = exec.RunCommand(DdevBin, args)
		assert.NoError(err)
		assert.FileExists(filepath.Join(v.Dir, "TestCmdExec-linux-gnu.txt"))

		args = []string{"exec", "ls >/dev/null && touch /var/www/html/TestCmdExec-touch-all-in-one.txt"}
		out, err = exec.RunCommand(DdevBin, args)
		assert.NoError(err)
		assert.FileExists(filepath.Join(v.Dir, "TestCmdExec-touch-all-in-one.txt"))

		args = []string{"exec", "true", "&&", "touch", "/var/www/html/TestCmdExec-touch-separate-args.txt"}
		out, err = exec.RunCommand(DdevBin, args)
		assert.NoError(err)
		assert.FileExists(filepath.Join(v.Dir, "TestCmdExec-touch-separate-args.txt"))

		cleanup()
	}
}
