package cmd

import (
	"fmt"
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
	app, err := ddevapp.GetActiveApp("")
	require.NoError(t, err)

	t.Cleanup(func() {
		_, err = exec.RunHostCommand(DdevBin, "service", "disable", "memcached")
		assert.NoError(err)
		_, err = exec.RunHostCommand(DdevBin, "service", "disable", "example")
		assert.NoError(err)

		_ = os.RemoveAll(app.GetConfigPath(fmt.Sprintf("docker-compose.%s.yaml", "memcached")))
		_ = os.RemoveAll(app.GetConfigPath(fmt.Sprintf("docker-compose.%s.yaml", "example")))

		err = os.Chdir(origDir)
		assert.NoError(err)
		err = os.RemoveAll(filepath.Join(globalconfig.GetGlobalDdevDir(), "commands/web/global-touched"))
		assert.NoError(err)
	})

	// Make sure get --list works first
	out, err := exec.RunHostCommand(DdevBin, "get", "--list")
	assert.NoError(err, "failed ddev get --list: %v (%s)", err, out)
	assert.Contains(out, "drud/ddev-memcached")

	tarballFile := filepath.Join(origDir, "testdata", t.Name(), "ddev-memcached.tar.gz")

	// Test with many input styles
	for _, arg := range []string{
		"drud/ddev-memcached",
		"https://github.com/drud/ddev-memcached/archive/refs/tags/v1.1.1.tar.gz",
		tarballFile} {
		out, err := exec.RunHostCommand(DdevBin, "get", arg)
		assert.NoError(err, "failed ddev get %s", arg)
		assert.Contains(out, "Downloaded add-on")
		assert.FileExists(app.GetConfigPath("docker-compose.memcached.yaml"))
	}

	// Test with a directory-path input
	exampleDir := filepath.Join(origDir, "testdata", t.Name(), "example-repo")
	_, err = exec.RunHostCommand(DdevBin, "get", exampleDir)
	assert.NoError(err)
	assert.FileExists(app.GetConfigPath("i-have-been-touched"))
	assert.FileExists(app.GetConfigPath("docker-compose.example.yaml"))
	assert.FileExists(filepath.Join(globalconfig.GetGlobalDdevDir(), "commands/web/global-touched"))
}
