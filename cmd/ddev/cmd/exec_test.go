package cmd

import (
	"testing"

	"github.com/drud/ddev/pkg/exec"
	asrt "github.com/stretchr/testify/assert"
)

// TestDevExecBadArgs run `ddev exec` without the proper args
func TestDevExecBadArgs(t *testing.T) {
	// Change to the first DevTestSite for the duration of this test.
	defer DevTestSites[0].Chdir()()
	assert := asrt.New(t)

	args := []string{"exec"}
	out, err := exec.RunCommand(DdevBin, args)
	assert.Error(err)
	assert.Contains(string(out), "Usage:")

	args = []string{"exec", "--root"}
	out, err = exec.RunCommand(DdevBin, args)
	assert.Error(err)
	assert.Contains(string(out), "Usage:")
}

// TestDevExec run `ddev exec pwd` with proper args
func TestDevExec(t *testing.T) {

	assert := asrt.New(t)
	for _, v := range DevTestSites {
		cleanup := v.Chdir()

		args := []string{"exec", "pwd"}
		out, err := exec.RunCommand(DdevBin, args)
		assert.NoError(err)
		assert.Contains(string(out), "/var/www/html")

		args = []string{"-s", "db", "exec", "pwd"}
		out, err = exec.RunCommand(DdevBin, args)
		assert.NoError(err)
		assert.Contains(string(out), "/")

		args = []string{"exec", "--root", "whoami"}
		out, err = exec.RunCommand(DdevBin, args)
		assert.NoError(err)
		assert.Contains(string(out), "root")

		args = []string{"exec", "-R", "whoami"}
		out, err = exec.RunCommand(DdevBin, args)
		assert.NoError(err)
		assert.Contains(string(out), "root")

		cleanup()
	}
}
