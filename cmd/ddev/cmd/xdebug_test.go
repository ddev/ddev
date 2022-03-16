package cmd

import (
	"github.com/drud/ddev/pkg/globalconfig"
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

	globalconfig.DdevVerbose = true

	// TestDdevXdebugEnabled has already tested enough versions, so limit it here.
	// and this is a pretty limited test, doesn't do much but turn on and off
	phpVersions := []string{nodeps.PHP80, nodeps.PHP81}

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
		_ = os.Setenv("DDEV_VERBOSE", "")
		globalconfig.DdevVerbose = false
	})

	// An odd bug in v1.16.2 popped up only when composer version was set, might as well set it here
	_, err = exec.RunHostCommand(DdevBin, "config", "--composer-version=2")
	assert.NoError(err)

	for _, phpVersion := range phpVersions {
		t.Logf("Testing xdebug command in php%s", phpVersion)
		_, err := exec.RunHostCommand(DdevBin, "config", "--php-version", phpVersion)
		require.NoError(t, err)

		_, err = exec.RunHostCommand(DdevBin, "start", "-y")
		assert.NoError(err, "failed ddev start with php=%v: %v", phpVersion, err)

		out, err := exec.RunHostCommand(DdevBin, "xdebug", "status")
		assert.NoError(err, "failed ddev xdebug status with php=%v: %v", phpVersion, err)
		assert.Contains(string(out), "xdebug disabled")

		out, err = exec.RunHostCommand(DdevBin, "xdebug", "on")
		assert.NoError(err)
		assert.Contains(string(out), "Enabled xdebug")

		out, err = exec.RunHostCommand(DdevBin, "xdebug", "status")
		assert.NoError(err)
		assert.Contains(string(out), "xdebug enabled")

		out, err = exec.RunHostCommand(DdevBin, "xdebug", "off")
		assert.NoError(err)
		assert.Contains(string(out), "Disabled xdebug")

		out, err = exec.RunHostCommand(DdevBin, "xdebug", "status")
		assert.NoError(err)
		assert.Contains(string(out), "xdebug disabled")

		_, err = exec.RunHostCommand(DdevBin, "stop")
		assert.NoError(err, "failed ddev stop with php=%v: %v", phpVersion, err)
	}
}
