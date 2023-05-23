package cmd

import (
	"fmt"
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	copy2 "github.com/otiai10/copy"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"

	asrt "github.com/stretchr/testify/assert"
)

// TestCmdGet tests various `ddev get` commands .
func TestCmdGet(t *testing.T) {
	if os.Getenv("DDEV_RUN_GET_TESTS") != "true" {
		t.Skip("Skipping because DDEV_RUN_GET_TESTS is not set")
	}
	assert := asrt.New(t)

	origDir, _ := os.Getwd()
	site := TestSites[0]
	err := os.Chdir(site.Dir)
	require.NoError(t, err)
	app, err := ddevapp.GetActiveApp("")
	require.NoError(t, err)

	t.Cleanup(func() {
		_, err = exec.RunHostCommand(DdevBin, "get", "--remove", "memcached")
		assert.NoError(err)
		_, err = exec.RunHostCommand(DdevBin, "get", "--remove", "example")
		assert.NoError(err)

		err = os.Chdir(origDir)
		assert.NoError(err)
		err = os.RemoveAll(filepath.Join(globalconfig.GetGlobalDdevDir(), "commands/web/global-touched"))
		assert.NoError(err)
	})

	// Make sure get --list works first
	out, err := exec.RunHostCommand(DdevBin, "get", "--list")
	assert.NoError(err, "failed ddev get --list: %v (%s)", err, out)
	assert.Contains(out, "ddev/ddev-memcached")

	tarballFile := filepath.Join(origDir, "testdata", t.Name(), "ddev-memcached.tar.gz")

	// Test with many input styles
	for _, arg := range []string{
		"ddev/ddev-memcached",
		"https://github.com/ddev/ddev-memcached/archive/refs/tags/v1.1.1.tar.gz",
		tarballFile} {
		out, err := exec.RunHostCommand(DdevBin, "get", arg)
		assert.NoError(err, "failed ddev get %s", arg)
		assert.Contains(out, "Installed DDEV add-on")
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

	assert.Contains(out, fmt.Sprintf("NOT overwriting %s", app.GetConfigPath("file-with-no-ddev-generated.txt")))
	assert.Contains(out, fmt.Sprintf("NOT overwriting %s", filepath.Join(globalconfig.GetGlobalDdevDir(), "file-with-no-ddev-generated.txt")))
}

// TestCmdGetComplex tests advanced usages
func TestCmdGetComplex(t *testing.T) {
	if os.Getenv("DDEV_RUN_GET_TESTS") != "true" {
		t.Skip("Skipping because DDEV_RUN_GET_TESTS is not set")
	}

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

	// create no-ddev-generated.txt so we make sure we get warning about it.
	_ = os.MkdirAll(app.GetConfigPath("extra"), 0755)
	_, err = os.Create(app.GetConfigPath("extra/no-ddev-generated.txt"))
	require.NoError(t, err)

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
	info, err := os.Stat(app.GetConfigPath("extra/no-ddev-generated.txt"))
	require.NoError(t, err, "stat of no-ddev-generated.txt failed")
	assert.True(info.Size() == 0)

	assert.Contains(out, "👍 extra/has-ddev-generated.txt")
	assert.NotContains(out, "👍 extra/no-ddev-generated.txt")
	assert.Regexp(regexp.MustCompile(`NOT overwriting [^ ]*`+"extra/no-ddev-generated.txt"), out)
}

// TestCmdGetInstalled tests `ddev get --installed` and `ddev get --remove`
func TestCmdGetInstalled(t *testing.T) {
	if os.Getenv("DDEV_RUN_GET_TESTS") != "true" {
		t.Skip("Skipping because DDEV_RUN_GET_TESTS is not set")
	}
	assert := asrt.New(t)

	origDir, _ := os.Getwd()
	site := TestSites[0]
	err := os.Chdir(site.Dir)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, err = exec.RunHostCommand(DdevBin, "get", "--remove", "memcached")
		assert.NoError(err)
		_, err = exec.RunHostCommand(DdevBin, "get", "--remove", "redis")
		assert.NoError(err)

		err = os.Chdir(origDir)
		assert.NoError(err)
		err = os.RemoveAll(filepath.Join(globalconfig.GetGlobalDdevDir(), "commands/web/global-touched"))
		assert.NoError(err)
	})

	out, err := exec.RunHostCommand(DdevBin, "get", "ddev/ddev-memcached", "--json-output")
	require.NoError(t, err, "failed ddev get ddev/ddev-memcached: %v (output='%s')", err, out)

	memcachedManifest := getManifestFromLogs(t, out)
	require.NoError(t, err)

	out, err = exec.RunHostCommand(DdevBin, "get", "ddev/ddev-redis", "--json-output")
	require.NoError(t, err, "failed ddev get ddev/ddev-redis: %v (output='%s')", err, out)

	redisManifest := getManifestFromLogs(t, out)
	require.NoError(t, err)

	installedOutput, err := exec.RunHostCommand(DdevBin, "get", "--installed", "--json-output")
	require.NoError(t, err, "failed ddev get --installed --json-output: %v (output='%s')", err, installedOutput)
	installedManifests := getManifestMapFromLogs(t, installedOutput)

	require.NotEmptyf(t, memcachedManifest["Version"], "memcached manifest is empty: %v", memcachedManifest)
	require.NotEmptyf(t, redisManifest["Version"], "redis manifest is empty: %v", redisManifest)

	assert.Equal(memcachedManifest["Version"], installedManifests["memcached"]["Version"])
	assert.Equal(redisManifest["Version"], installedManifests["redis"]["Version"])

	// Now try the remove using other techniques (full repo name, partial repo name)
	for _, n := range []string{"ddev/ddev-redis", "ddev-redis", "redis"} {
		out, err = exec.RunHostCommand(DdevBin, "get", "ddev/ddev-redis", "--json-output")
		require.NoError(t, err, "failed ddev get %s: %v (output='%s')", n, err, out)
		out, err = exec.RunHostCommand(DdevBin, "get", "--remove", n)
		require.NoError(t, err, "unable to ddev get --remove %s: %v, output='%s'", n, err, out)
	}
	// Now make sure we put it back so it can be removed in cleanu
	out, err = exec.RunHostCommand(DdevBin, "get", "ddev/ddev-redis")
	assert.NoError(err, "unable to ddev get redis: %v, output='%s'", err, out)
}

// TestCmdGetDependencies tests the dependency behavior is correct
func TestCmdGetDependencies(t *testing.T) {
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
		out, err := exec.RunHostCommand(DdevBin, "get", "--remove", "dependency_recipe")
		assert.NoError(err, "output='%s'", out)
		out, err = exec.RunHostCommand(DdevBin, "get", "--remove", "depender_recipe")
		assert.NoError(err, "output='%s'", out)
		err = os.Chdir(origDir)
		assert.NoError(err)
	})

	// First try of depender_recipe should fail without dependency
	out, err := exec.RunHostCommand(DdevBin, "get", filepath.Join(origDir, "testdata", t.Name(), "depender_recipe"))
	require.Error(t, err, "out=%s", out)

	// Now add the dependency and try again
	out, err = exec.RunHostCommand(DdevBin, "get", filepath.Join(origDir, "testdata", t.Name(), "dependency_recipe"))
	require.NoError(t, err, "out=%s", out)

	// Now depender_recipe should succeed
	out, err = exec.RunHostCommand(DdevBin, "get", filepath.Join(origDir, "testdata", t.Name(), "depender_recipe"))
	require.NoError(t, err, "out=%s", out)
}

// getManifestFromLogs returns the manifest built from 'raw' section of
// ddev get <project> -j output
func getManifestFromLogs(t *testing.T, jsonOut string) map[string]interface{} {
	assert := asrt.New(t)

	logItems, err := unmarshalJSONLogs(jsonOut)
	require.NoError(t, err)
	data := logItems[len(logItems)-1]
	assert.EqualValues(data["level"], "info")

	m, ok := data["raw"].(map[string]interface{})
	require.True(t, ok)
	return m
}

// getManifestMapFromLogs returns the manifest array built from 'raw' section of
// ddev get --installed -j output
func getManifestMapFromLogs(t *testing.T, jsonOut string) map[string]map[string]interface{} {
	assert := asrt.New(t)

	logItems, err := unmarshalJSONLogs(jsonOut)
	require.NoError(t, err)
	data := logItems[len(logItems)-1]
	assert.EqualValues(data["level"], "info")

	m, ok := data["raw"].([]interface{})
	require.True(t, ok)
	masterMap := map[string]map[string]interface{}{}
	for _, item := range m {
		itemMap := item.(map[string]interface{})
		masterMap[itemMap["Name"].(string)] = itemMap
	}
	return masterMap
}
