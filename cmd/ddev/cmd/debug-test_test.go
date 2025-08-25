package cmd

import (
	"os"
	"runtime"
	"testing"

	"github.com/ddev/ddev/pkg/exec"
	"github.com/stretchr/testify/require"
)

// TestCmdDebugTest ensures that `ddev debug test` has basic functionality
func TestCmdDebugTest(t *testing.T) {
	origDir, _ := os.Getwd()
	site := TestSites[0]
	_ = os.Chdir(site.Dir)
	t.Cleanup(func() {
		_ = os.Chdir(origDir)
	})

	_ = os.Setenv("DDEV_NONINTERACTIVE", "true")
	_ = os.Setenv("DDEV_DEBUG", "true")

	out, err := exec.RunHostCommand(DdevBin, "debug", "test")
	require.NoError(t, err, "out=%s", out)
	require.Contains(t, out, "OS Information")
	require.Contains(t, out, "webserver_type:")
	if runtime.GOOS != "windows" {
		require.Contains(t, out, "PING dkdkd.ddev.site")
	}
}
