package ddevapp_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/ddev/ddev/pkg/util"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNodeJSVersions whether we can configure nodejs versions
func TestNodeJSVersions(t *testing.T) {
	if os.Getenv("DDEV_SKIP_NODEJS_TEST") == "true" {
		t.Skip("Skipping TestNodeJSVersions because DDEV_SKIP_NODEJS_TEST is true")
	}
	assert := asrt.New(t)

	site := TestSites[0]
	origDir, _ := os.Getwd()
	_ = os.Chdir(site.Dir)

	runTime := util.TimeTrackC(t.Name())

	testcommon.ClearDockerEnv()
	app, err := ddevapp.NewApp(site.Dir, true)
	assert.NoError(err)
	t.Cleanup(func() {
		runTime()
		err = os.Chdir(origDir)
		assert.NoError(err)
		err = app.Stop(true, false)
		assert.NoError(err)
	})

	nvmrcFile := filepath.Join(app.AppRoot, ".nvmrc")
	err = fileutil.CopyFile(filepath.Join(origDir, "testdata", t.Name(), ".nvmrc"), nvmrcFile)
	require.NoError(t, err)
	nvmrcFileContents, err := os.ReadFile(nvmrcFile)
	require.NoError(t, err, "Unable to read %s: %v", nvmrcFile, err)
	nvmrcVersion := strings.TrimSpace(string(nvmrcFileContents))

	packageJSONFile := filepath.Join(app.AppRoot, "package.json")
	err = fileutil.CopyFile(filepath.Join(origDir, "testdata", t.Name(), "package.json"), packageJSONFile)
	require.NoError(t, err)
	packageJSONFileContents, err := os.ReadFile(packageJSONFile)
	require.NoError(t, err, "Unable to read %s: %v", packageJSONFile, err)
	var packageJSON map[string]interface{}
	err = json.Unmarshal(packageJSONFileContents, &packageJSON)
	require.NoError(t, err, "Unable to unmarshal %s: %v", packageJSONFile, err)
	engines := packageJSON["engines"].(map[string]interface{})
	packageJSONVersion := engines["node"].(string)

	err = app.Start()
	require.NoError(t, err)

	// Testing some random versions, complete, incomplete, and labels
	for _, v := range []string{"6", "auto", "engine", "16.0.0", "20"} {
		app.NodeJSVersion = v
		if app.NodeJSVersion == "auto" {
			v = nvmrcVersion
		} else if app.NodeJSVersion == "engine" {
			v = packageJSONVersion
		}
		err = app.Restart()
		assert.NoError(err)
		out, _, err := app.Exec(&ddevapp.ExecOpts{
			Cmd: "node --version",
		})
		assert.NoError(err)
		assert.True(strings.HasPrefix(out, "v"+v), "Expected node version to start with '%s', but got %s", v, out)
	}

	out, _, err := app.Exec(&ddevapp.ExecOpts{
		Cmd: `bash -ic "nvm install 6 && node --version"`,
	})
	require.NoError(t, err)
	assert.Contains(out, "Now using node v6")

	out, err = exec.RunHostCommand(DdevBin, "nvm", "install", "8")
	require.NoError(t, err, "output=%v", out)
	assert.Contains(out, "Now using node v8")
	out, err = exec.RunHostCommand(DdevBin, "nvm", "use", "8")
	require.NoError(t, err, "output=%v", out)
	out, err = exec.RunHostCommand(DdevBin, "nvm", "alias", "default", "8")
	require.NoError(t, err, "output=%v", out)

	out, err = exec.RunHostCommand(DdevBin, "exec", "node", "--version")
	require.NoError(t, err)

	assert.Contains(out, "v8.17")

}

// TestCorepackEnable tests behavior of corepack_enable
func TestCorepackEnable(t *testing.T) {
	assert := asrt.New(t)

	site := TestSites[0]
	origDir, _ := os.Getwd()
	_ = os.Chdir(site.Dir)

	runTime := util.TimeTrackC(t.Name())

	testcommon.ClearDockerEnv()
	app, err := ddevapp.NewApp(site.Dir, true)
	assert.NoError(err)
	t.Cleanup(func() {
		runTime()
		_ = os.Chdir(origDir)
		_ = app.Restart()
	})

	err = app.Start()
	require.NoError(t, err)

	app.CorepackEnable = false
	err = app.Start()
	require.NoError(t, err)
	out, _, err := app.Exec(&ddevapp.ExecOpts{
		Cmd: `ls -l /usr/local/bin/yarn`,
	})
	require.NoError(t, err)
	require.NotContains(t, out, "corepack")
	out, _, err = app.Exec(&ddevapp.ExecOpts{
		Cmd: `yarn --version`,
	})
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(out, "1."))

	app.CorepackEnable = true
	err = app.Start()
	require.NoError(t, err)
	out, _, err = app.Exec(&ddevapp.ExecOpts{
		Cmd: `ls -l /usr/local/bin/yarn`,
	})
	require.NoError(t, err)
	require.Contains(t, out, "corepack")
	_, _, err = app.Exec(&ddevapp.ExecOpts{
		Cmd: `yarn set version stable`,
	})
	require.NoError(t, err)

	out, _, err = app.Exec(&ddevapp.ExecOpts{
		Cmd: `yarn --version`,
	})
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(out, "4."))
}
