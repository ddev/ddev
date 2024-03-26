package cmd

import (
	"os"
	"runtime"
	"testing"

	"github.com/ddev/ddev/pkg/exec"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCmdDebugTest ensures that `ddev debug test` has basic functionality
func TestCmdDebugTest(t *testing.T) {
	assert := asrt.New(t)

	origDir, _ := os.Getwd()
	site := TestSites[0]
	err := os.Chdir(site.Dir)
	require.NoError(t, err)

	require.NoError(t, err)
	t.Cleanup(func() {
		err = os.Chdir(origDir)
		assert.NoError(err)
	})

	_ = os.Setenv("DDEV_NONINTERACTIVE", "true")
	_ = os.Setenv("DDEV_DEBUG", "true")

	out, err := exec.RunHostCommand(DdevBin, "debug", "test")
	// This is just a casual look at the output, not intended to look for all details.
	require.Contains(t, out, "OS Information")
	require.Contains(t, out, "webserver_type:")
	if runtime.GOOS != "windows" {
		require.Contains(t, out, "PING dkdkd.ddev.site")
	}
}
