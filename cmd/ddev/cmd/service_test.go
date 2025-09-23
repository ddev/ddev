package cmd

import (
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"

	asrt "github.com/stretchr/testify/assert"
)

// TestCmdService tests ddev service enable/disable .
func TestCmdService(t *testing.T) {
	assert := asrt.New(t)

	origDir, _ := os.Getwd()
	site := TestSites[0]
	err := os.Chdir(site.Dir)
	require.NoError(t, err)
	app, err := ddevapp.GetActiveApp("")
	require.NoError(t, err)
	t.Cleanup(func() {
		_, err = exec.RunHostCommand(DdevBin, "service", "disable", "memcached")
		assert.NoError(err)
		err = os.Chdir(origDir)
		assert.NoError(err)
	})

	tarballFile := filepath.Join(origDir, "testdata", t.Name(), "ddev-memcached.tar.gz")
	_, err = exec.RunHostCommand(DdevBin, "get", tarballFile)
	require.NoError(t, err, "Failed to get memcached tarball %s: %v", tarballFile, err)
	assert.FileExists(app.GetConfigPath("docker-compose.memcached.yaml"))
	_, err = exec.RunHostCommand(DdevBin, "service", "disable", "memcached")
	assert.NoFileExists(app.GetConfigPath("docker-compose.memcached.yaml"))
	_, err = exec.RunHostCommand(DdevBin, "service", "enable", "memcached")
	assert.FileExists(app.GetConfigPath("docker-compose.memcached.yaml"))
}
