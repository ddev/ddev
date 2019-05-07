package cmd

import (
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

// TestCmdExec runs `ddev exec pwd` with proper args
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

		cleanup()
	}
}
