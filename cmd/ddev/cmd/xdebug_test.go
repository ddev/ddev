package cmd

import (
	"os"
	"testing"

	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/stretchr/testify/require"

	"github.com/ddev/ddev/pkg/exec"
)

// TestCmdXdebug tests the `ddev xdebug` command
func TestCmdXdebug(t *testing.T) {
	globalconfig.DdevVerbose = true

	// TestDdevXdebugEnabled has already tested enough versions, so limit it here.
	// and this is a pretty limited test, doesn't do much but turn on and off
	phpVersions := []string{nodeps.PHP83, nodeps.PHP84}

	pwd, _ := os.Getwd()
	v := TestSites[0]

	err := os.Chdir(v.Dir)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, err = exec.RunCommand(DdevBin, []string{"config", "--php-version", nodeps.PHPDefault, "--composer-version=\"\""})
		require.NoError(t, err)
		_, err = exec.RunCommand(DdevBin, []string{"xdebug", "off"})
		require.NoError(t, err)
		err := os.Chdir(pwd)
		require.NoError(t, err)
		_ = os.Setenv("DDEV_VERBOSE", "")
		globalconfig.DdevVerbose = false
	})

	// An odd bug in v1.16.2 popped up only when Composer version was set, might as well set it here
	_, err = exec.RunHostCommand(DdevBin, "config", "--composer-version=2")
	require.NoError(t, err)

	for _, phpVersion := range phpVersions {
		t.Logf("Testing Xdebug command in php%s", phpVersion)
		_, err := exec.RunHostCommand(DdevBin, "config", "--php-version", phpVersion)
		require.NoError(t, err)

		_, err = exec.RunHostCommand(DdevBin, "restart")
		require.NoError(t, err, "Failed ddev start with php=%v: %v", phpVersion, err)

		out, err := exec.RunHostCommand(DdevBin, "xdebug", "status")
		require.NoError(t, err, "Failed ddev xdebug status with php=%v: %v", phpVersion, err)
		require.Contains(t, string(out), "xdebug disabled")

		out, err = exec.RunHostCommand(DdevBin, "xdebug", "on")
		require.NoError(t, err)
		require.Contains(t, string(out), "Enabled xdebug")

		out, err = exec.RunHostCommand(DdevBin, "xdebug", "status")
		require.NoError(t, err)
		require.Contains(t, string(out), "xdebug enabled")

		out, err = exec.RunHostCommand(DdevBin, "xdebug", "info")
		require.NoError(t, err)
		require.Contains(t, string(out), "Step Debugger => ✔ enabled")

		out, err = exec.RunHostCommand(DdevBin, "xdebug", "off")
		require.NoError(t, err)
		require.Contains(t, string(out), "Disabled xdebug")

		out, err = exec.RunHostCommand(DdevBin, "xdebug", "status")
		require.NoError(t, err)
		require.Contains(t, string(out), "xdebug disabled")

		out, err = exec.RunHostCommand(DdevBin, "xdebug", "toggle")
		require.NoError(t, err)
		require.Contains(t, string(out), "Enabled xdebug")

		out, err = exec.RunHostCommand(DdevBin, "xdebug", "status")
		require.NoError(t, err)
		require.Contains(t, string(out), "xdebug enabled")

		out, err = exec.RunHostCommand(DdevBin, "xdebug", "toggle")
		require.NoError(t, err)
		require.Contains(t, string(out), "Disabled xdebug")

		out, err = exec.RunHostCommand(DdevBin, "xdebug", "status")
		require.NoError(t, err)
		require.Contains(t, string(out), "xdebug disabled")

		_, err = exec.RunHostCommand(DdevBin, "stop")
		require.NoError(t, err, "Failed ddev stop with php=%v: %v", phpVersion, err)
	}
}
