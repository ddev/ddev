package cmd

import (
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"

	asrt "github.com/stretchr/testify/assert"
)

// TestCmdGet tests various `ddev get` commands .
func TestCmdGet(t *testing.T) {
	assert := asrt.New(t)

	origDir, _ := os.Getwd()
	site := TestSites[0]
	err := os.Chdir(site.Dir)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, err = exec.RunHostCommand(DdevBin, "service", "disable", "memcached")
		assert.NoError(err)
		err = os.Chdir(origDir)
		assert.NoError(err)
		err = os.RemoveAll(filepath.Join(globalconfig.GetGlobalDdevDir(), "commands/web/global-touched"))
		assert.NoError(err)
	})

	app, err := ddevapp.GetActiveApp("")
	require.NoError(t, err)

	tarballFile := filepath.Join(origDir, "testdata", t.Name(), "ddev-memcached.tar.gz")
	for _, arg := range []string{
		"drud/ddev-memcached",
		"https://github.com/drud/ddev-memcached/archive/refs/tags/v1.1.1.tar.gz",
		tarballFile} {
		out, err := exec.RunHostCommand(DdevBin, "get", arg)
		assert.NoError(err, "failed ddev get %s", arg)
		assert.Contains(out, "Downloaded add-on")
		assert.FileExists(app.GetConfigPath("docker-compose.memcached.yaml"))
	}

	exampleDir := filepath.Join(origDir, "testdata", t.Name(), "example-repo")
	_, err = exec.RunHostCommand(DdevBin, "get", exampleDir)
	assert.NoError(err)
	assert.FileExists(app.GetConfigPath("i-have-been-touched"))
	assert.FileExists(app.GetConfigPath("docker-compose.example.yaml"))
	assert.FileExists(filepath.Join(globalconfig.GetGlobalDdevDir(), "commands/web/global-touched"))
}
