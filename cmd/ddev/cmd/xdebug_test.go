package cmd

import (
	"github.com/drud/ddev/pkg/nodeps"
	"github.com/stretchr/testify/require"
	"os"
	"testing"

	"github.com/drud/ddev/pkg/exec"
	asrt "github.com/stretchr/testify/assert"
)

// TestCmdXdebug tests the `ddev xdebug` command
func TestCmdXdebug(t *testing.T) {
	assert := asrt.New(t)
	pwd, _ := os.Getwd()
	v := TestSites[0]

	err := os.Chdir(v.Dir)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, err = exec.RunCommand(DdevBin, []string{"config", "--php-version", nodeps.PHPDefault, "--composer-version=\"\""})
		assert.NoError(err)
		_, err = exec.RunCommand(DdevBin, []string{"xdebug", "off"})
		assert.NoError(err)
		err := os.Chdir(pwd)
		assert.NoError(err)
	})

	// An odd bug in v1.16.2 popped up only when composer version was set, might as well set it here
	_, err = exec.RunCommand(DdevBin, []string{"config", "--composer-version=2"})
	assert.NoError(err)

	for phpVersion := range nodeps.ValidPHPVersions {
		t.Logf("Testing xdebug command in php%s", phpVersion)
		_, err := exec.RunCommand(DdevBin, []string{"config", "--php-version", phpVersion})
		require.NoError(t, err)

		_, err = exec.RunCommand(DdevBin, []string{"start", "-y"})
		assert.NoError(err)

		out, err := exec.RunCommand(DdevBin, []string{"xdebug", "status"})
		assert.NoError(err)
		assert.Contains(string(out), "xdebug disabled")

		out, err = exec.RunCommand(DdevBin, []string{"xdebug", "on"})
		assert.NoError(err)
		assert.Contains(string(out), "Enabled xdebug")

		out, err = exec.RunCommand(DdevBin, []string{"xdebug", "status"})
		assert.NoError(err)
		assert.Contains(string(out), "xdebug enabled")

		out, err = exec.RunCommand(DdevBin, []string{"xdebug", "off"})
		assert.NoError(err)
		assert.Contains(string(out), "Disabled xdebug")

		out, err = exec.RunCommand(DdevBin, []string{"xdebug", "status"})
		assert.NoError(err)
		assert.Contains(string(out), "xdebug disabled")
	}
}
