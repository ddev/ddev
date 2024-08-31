package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/stretchr/testify/require"

	asrt "github.com/stretchr/testify/assert"
)

// TestCmdAddon tests various `ddev add-on` commands.
func TestCmdAddon(t *testing.T) {
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
		_, err = exec.RunHostCommand(DdevBin, "add-on", "remove", "memcached")
		assert.NoError(err)
		_, err = exec.RunHostCommand(DdevBin, "add-on", "remove", "example")
		assert.NoError(err)

		err = os.Chdir(origDir)
		assert.NoError(err)
		_ = os.RemoveAll(filepath.Join(globalconfig.GetGlobalDdevDir(), "commands/web/global-touched"))
	})

	// Make sure 'ddev add-on list' works first
	out, err := exec.RunHostCommand(DdevBin, "add-on", "list")
	assert.NoError(err, "failed ddev add-on list: %v (%s)", err, out)
	assert.Contains(out, "ddev/ddev-memcached")

	tarballFile := filepath.Join(origDir, "testdata", t.Name(), "ddev-memcached.tar.gz")

	// Test with many input styles
	for _, arg := range []string{
		"ddev/ddev-memcached",
		"https://github.com/ddev/ddev-memcached/archive/refs/tags/v1.1.1.tar.gz",
		tarballFile} {
		out, err := exec.RunHostCommand(DdevBin, "add-on", "get", arg)
		assert.NoError(err, "failed ddev add-on get %s", arg)
		assert.Contains(out, "Installed DDEV add-on")
		assert.FileExists(app.GetConfigPath("docker-compose.memcached.yaml"))
	}

	// Test with a directory-path input
	exampleDir := filepath.Join(origDir, "testdata", t.Name(), "example-repo")
	err = fileutil.TemplateStringToFile("no signature here", nil, app.GetConfigPath("file-with-no-ddev-generated.txt"))
	require.NoError(t, err)
	err = fileutil.TemplateStringToFile("no signature here", nil, filepath.Join(globalconfig.GetGlobalDdevDir(), "file-with-no-ddev-generated.txt"))
	require.NoError(t, err)

	out, err = exec.RunHostCommand(DdevBin, "add-on", "get", exampleDir)
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

// TestCmdAddonInstalled tests `ddev add-on list --installed` and `ddev add-on remove`
func TestCmdAddonInstalled(t *testing.T) {
	if os.Getenv("DDEV_RUN_GET_TESTS") != "true" {
		t.Skip("Skipping because DDEV_RUN_GET_TESTS is not set")
	}
	origDdevDebug := os.Getenv("DDEV_DEBUG")
	_ = os.Unsetenv("DDEV_DEBUG")
	assert := asrt.New(t)

	origDir, _ := os.Getwd()
	site := TestSites[0]
	err := os.Chdir(site.Dir)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, err = exec.RunHostCommand(DdevBin, "add-on", "remove", "memcached")
		assert.NoError(err)
		_, err = exec.RunHostCommand(DdevBin, "add-on", "remove", "redis")
		assert.NoError(err)

		err = os.Chdir(origDir)
		assert.NoError(err)
		_ = os.RemoveAll(filepath.Join(globalconfig.GetGlobalDdevDir(), "commands/web/global-touched"))
		_ = os.Setenv("DDEV_DEBUG", origDdevDebug)
	})

	out, err := exec.RunHostCommand(DdevBin, "add-on", "get", "ddev/ddev-memcached", "--json-output")
	require.NoError(t, err, "failed ddev add-on get ddev/ddev-memcached: %v (output='%s')", err, out)

	memcachedManifest := getManifestFromLogs(t, out)
	require.NoError(t, err)

	out, err = exec.RunHostCommand(DdevBin, "add-on", "get", "ddev/ddev-redis", "--json-output")
	require.NoError(t, err, "failed ddev add-on get ddev/ddev-redis: %v (output='%s')", err, out)

	redisManifest := getManifestFromLogs(t, out)
	require.NoError(t, err)

	installedOutput, err := exec.RunHostCommand(DdevBin, "add-on", "list", "--installed", "--json-output")
	require.NoError(t, err, "failed ddev add-on list --installed --json-output: %v (output='%s')", err, installedOutput)
	installedManifests := getManifestMapFromLogs(t, installedOutput)

	require.NotEmptyf(t, memcachedManifest["Version"], "memcached manifest is empty: %v", memcachedManifest)
	require.NotEmptyf(t, redisManifest["Version"], "redis manifest is empty: %v", redisManifest)

	assert.Equal(memcachedManifest["Version"], installedManifests["memcached"]["Version"])
	assert.Equal(redisManifest["Version"], installedManifests["redis"]["Version"])

	// Now try the remove using other techniques (full repo name, partial repo name)
	for _, n := range []string{"ddev/ddev-redis", "ddev-redis", "redis"} {
		out, err = exec.RunHostCommand(DdevBin, "add-on", "get", "ddev/ddev-redis", "--json-output")
		require.NoError(t, err, "failed ddev add-on get %s: %v (output='%s')", n, err, out)
		out, err = exec.RunHostCommand(DdevBin, "add-on", "remove", n)
		require.NoError(t, err, "unable to ddev add-on remove %s: %v, output='%s'", n, err, out)
	}
	// Now make sure we put it back so it can be removed in cleanu
	out, err = exec.RunHostCommand(DdevBin, "add-on", "get", "ddev/ddev-redis")
	assert.NoError(err, "unable to ddev add-on get redis: %v, output='%s'", err, out)
}

// TestCmdAddonProjectFlag tests the `--project` flag in `ddev add-on` subcommands
func TestCmdAddonProjectFlag(t *testing.T) {
	if os.Getenv("DDEV_RUN_GET_TESTS") != "true" {
		t.Skip("Skipping because DDEV_RUN_GET_TESTS is not set")
	}
	origDdevDebug := os.Getenv("DDEV_DEBUG")
	_ = os.Unsetenv("DDEV_DEBUG")
	assert := asrt.New(t)

	site := TestSites[0]
	// Explicitly don't chdir to the project

	t.Cleanup(func() {
		_, err := exec.RunHostCommand(DdevBin, "add-on", "remove", "redis", "--project", site.Name)
		assert.NoError(err)
		_ = os.RemoveAll(filepath.Join(globalconfig.GetGlobalDdevDir(), "commands/web/global-touched"))
		_ = os.Setenv("DDEV_DEBUG", origDdevDebug)
	})

	// Install the add-on using the `--project` flag
	out, err := exec.RunHostCommand(DdevBin, "add-on", "get", "ddev/ddev-redis", "--project", site.Name, "--json-output")
	require.NoError(t, err, "failed ddev add-on get ddev/ddev-redis --project %s --json-output: %v (output='%s')", site.Name, err, out)

	redisManifest := getManifestFromLogs(t, out)
	require.NoError(t, err)

	installedOutput, err := exec.RunHostCommand(DdevBin, "add-on", "list", "--installed", "--project", site.Name, "--json-output")
	require.NoError(t, err, "failed ddev add-on list --installed --project %s --json-output: %v (output='%s')", site.Name, err, installedOutput)
	installedManifests := getManifestMapFromLogs(t, installedOutput)

	require.NotEmptyf(t, redisManifest["Version"], "redis manifest is empty: %v", redisManifest)
	assert.Equal(redisManifest["Version"], installedManifests["redis"]["Version"])

	// Remove the add-on using the `--project` flag
	out, err = exec.RunHostCommand(DdevBin, "add-on", "remove", "ddev/ddev-redis", "--project", site.Name)
	require.NoError(t, err, "unable to ddev add-on remove ddev/ddev-redis --project %s: %v, output='%s'", site.Name, err, out)

	// Now make sure we put it back so it can be removed in cleanup
	out, err = exec.RunHostCommand(DdevBin, "add-on", "get", "ddev/ddev-redis", "--project", site.Name)
	assert.NoError(err, "unable to ddev add-on get ddev/ddev-redis --project %s: %v, output='%s'", site.Name, err, out)
}

// getManifestFromLogs returns the manifest built from 'raw' section of
// ddev add-on get <project> -j output
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
// ddev add-on list --installed -j output
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
