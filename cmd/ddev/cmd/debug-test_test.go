package cmd

import (
	"github.com/ddev/ddev/pkg/ddevapp"
	"os"
	"testing"

	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/testcommon"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDebugTestCmd ensures that `ddev debug test` has basic functionality
func TestDebugTestCmd(t *testing.T) {
	assert := asrt.New(t)

	origDir, _ := os.Getwd()

	debugAppName := t.Name()

	testDir := testcommon.CreateTmpDir(debugAppName)
	err := os.Chdir(testDir)
	require.NoError(t, err)

	out, err := exec.RunHostCommand(DdevBin, "config", "--project-name", debugAppName)
	require.NoError(t, err, "Failed to run ddev config: %s", out)

	app, err := ddevapp.NewApp(testDir, true)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = app.Stop(true, false)
		err = os.Chdir(origDir)
		assert.NoError(err)
		_ = os.RemoveAll(testDir)
	})

	_ = os.Setenv("DDEV_NONINTERACTIVE", "true")
	_ = os.Setenv("DDEV_DEBUG", "true")

	out, err = exec.RunHostCommand(DdevBin, "debug", "test")
	require.Contains(t, out, "OS Information")
	require.Contains(t, out, "PING dkdkd.ddev.site")
}
