package cmd

import (
	"github.com/stretchr/testify/require"
	"os"
	"testing"

	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/testcommon"
	asrt "github.com/stretchr/testify/assert"
)

// TestDebugDownloadImages tests ddev debug download-images
func TestDebugDownloadImages(t *testing.T) {
	assert := asrt.New(t)

	// Create a temporary directory and switch to it.
	origDir, _ := os.Getwd()

	testDir := testcommon.CreateTmpDir(t.Name())
	err := os.Chdir(testDir)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, err = exec.RunHostCommand(DdevBin, "delete", "-Oy", t.Name())
		assert.NoError(err)

		err = os.Chdir(origDir)
		assert.NoError(err)
		err = os.RemoveAll(testDir)
		assert.NoError(err)
	})

	out, err := exec.RunHostCommand(DdevBin, "config", "--project-name", t.Name())
	require.NoError(t, err, "Failed to run ddev config: %s", out)

	_ = os.Setenv("DDEV_DEBUG", "true")
	out, err = exec.RunHostCommand(DdevBin, "debug", "download-images")
	require.NoError(t, err, "Failed to run ddev debug download-images: %s", out)
	assert.Contains(out, "ddev-webserver")
	assert.Contains(out, "ddev-router")
	assert.Contains(out, "Successfully downloaded ddev images")
}
