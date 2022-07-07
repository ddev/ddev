package cmd

import (
	"fmt"
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/globalconfig"
	copy2 "github.com/otiai10/copy"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"runtime"
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
	err = fileutil.TemplateStringToFile("no signature here", nil, app.GetConfigPath("file-with-no-ddev-generated.txt"))
	require.NoError(t, err)
	err = fileutil.TemplateStringToFile("no signature here", nil, filepath.Join(globalconfig.GetGlobalDdevDir(), "file-with-no-ddev-generated.txt"))
	require.NoError(t, err)

	out, err = exec.RunHostCommand(DdevBin, "get", exampleDir)
	assert.NoError(err, "output=%s", out)
	assert.FileExists(app.GetConfigPath("i-have-been-touched"))
	assert.FileExists(app.GetConfigPath("docker-compose.example.yaml"))
	exists, err := fileutil.FgrepStringInFile(app.GetConfigPath("file-with-no-ddev-generated.txt"), "install should result in a warning")
	require.NoError(t, err)
	assert.False(exists, "the file with no ddev-generated.txt should not have been replaced")

	assert.FileExists(filepath.Join(globalconfig.GetGlobalDdevDir(), "commands/web/global-touched"))
	assert.FileExists(filepath.Join(globalconfig.GetGlobalDdevDir(), "globalextras/okfile.txt"))

	exists, err = fileutil.FgrepStringInFile(filepath.Join(globalconfig.GetGlobalDdevDir(), "file-with-no-ddev-generated.txt"), "install should result in a warning")
	require.NoError(t, err)
	assert.False(exists, "the file with no ddev-generated.txt should not have been replaced")

	assert.Contains(out, fmt.Sprintf("NOT overwriting file/directory %s", app.GetConfigPath("file-with-no-ddev-generated.txt")))
	assert.Contains(out, fmt.Sprintf("NOT overwriting file/directory %s", filepath.Join(globalconfig.GetGlobalDdevDir(), "file-with-no-ddev-generated.txt")))

}

// TestCmdGetComplex tests advanced usages
func TestCmdGetComplex(t *testing.T) {
	assert := asrt.New(t)

	origDir, _ := os.Getwd()
	site := TestSites[0]
	err := os.Chdir(site.Dir)
	require.NoError(t, err)
	app, err := ddevapp.GetActiveApp("")
	require.NoError(t, err)

	err = copy2.Copy(filepath.Join(origDir, "testdata", t.Name(), "project"), app.GetAppRoot())
	require.NoError(t, err)

	t.Cleanup(func() {
		err = os.Chdir(origDir)
		assert.NoError(err)
		for _, f := range []string{".platform", ".platform.app.yaml"} {
			err = os.RemoveAll(filepath.Join(app.GetAppRoot(), f))
		}
		for _, f := range []string{fmt.Sprintf("junk_%s_%s.txt", runtime.GOOS, runtime.GOARCH), "config.platformsh.yaml"} {
			err = os.RemoveAll(app.GetConfigPath(f))
			assert.NoError(err)
		}
		// We have to completely kill off app because the install.yaml + config.platformsh.yaml got us a completely different
		// database.
		err = app.Stop(true, false)
		assert.NoError(err)
		app, err = ddevapp.NewApp(app.AppRoot, true)
		assert.NoError(err)
		err = app.Start()
		assert.NoError(err)
	})

	out, err := exec.RunHostCommand(DdevBin, "get", filepath.Join(origDir, "testdata", t.Name(), "recipe"))
	require.NoError(t, err, "out=%s", out)

	app, err = ddevapp.GetActiveApp("")
	require.NoError(t, err)

	// Make sure that all the interpolations we wrote via go templates got in there
	assert.Equal("web99", app.Docroot)
	assert.Equal("mariadb", app.Database.Type)
	assert.Equal("10.7", app.Database.Version)
	assert.Equal("8.1", app.PHPVersion)

	// Make sure that environment variable interpolation happened. If it did, we'll have the one file
	// we're looking for.
	assert.FileExists(app.GetConfigPath(fmt.Sprintf("junk_%s_%s.txt", runtime.GOOS, runtime.GOARCH)))
}
