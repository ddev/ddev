package ddevapp_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/ddev/ddev/pkg/util"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNodeJSVersions whether we can configure nodejs versions
func TestNodeJSVersions(t *testing.T) {
	if nodeps.IsEnvTrue("DDEV_SKIP_NODEJS_TEST") {
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
	app.CorepackEnable = false
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
	var packageJSON map[string]any
	err = json.Unmarshal(packageJSONFileContents, &packageJSON)
	require.NoError(t, err, "Unable to unmarshal %s: %v", packageJSONFile, err)
	engines := packageJSON["engines"].(map[string]any)
	packageJSONVersion := engines["node"].(string)

	err = app.Start()
	require.NoError(t, err)

	// Testing some random versions, complete, incomplete, and labels.
	// An empty value must fall back to the default Node.js version DDEV provides.
	for _, v := range []string{"6", "auto", "engine", "16.0.0", "20", ""} {
		app.NodeJSVersion = v
		if app.NodeJSVersion == "auto" {
			v = nvmrcVersion
		} else if app.NodeJSVersion == "engine" {
			v = packageJSONVersion
		} else if app.NodeJSVersion == "" {
			v = nodeps.NodeJSDefault
		}
		err = app.Restart()
		assert.NoError(err)
		out, _, err := app.Exec(&ddevapp.ExecOpts{
			Cmd: "node --version",
		})
		assert.NoError(err)
		assert.True(strings.HasPrefix(out, "v"+v), "Expected node version to start with '%s', but got %s", v, out)

		// Verify node is reachable via sudo (secure_path must include /usr/local/n/bin)
		out, _, err = app.Exec(&ddevapp.ExecOpts{
			Cmd:  `sudo -u "$DDEV_USER" node --version`,
			User: "root",
		})
		assert.NoError(err)
		assert.True(strings.HasPrefix(out, "v"+v), "Expected sudo node version to start with '%s', but got %s", v, out)

		// Verify that globally installed npm packages are present and working
		_, _, err = app.Exec(&ddevapp.ExecOpts{
			Cmd: `gulp --version && yarn --version`,
		})
		assert.NoError(err)
	}
}

// TestNodeJSRoot tests that nodejs_root reads the version file from a
// subdirectory of the project root
func TestNodeJSRoot(t *testing.T) {
	if nodeps.IsEnvTrue("DDEV_SKIP_NODEJS_TEST") {
		t.Skip("Skipping TestNodeJSRoot because DDEV_SKIP_NODEJS_TEST is true")
	}

	site := TestSites[0]
	origDir, _ := os.Getwd()
	_ = os.Chdir(site.Dir)

	runTime := util.TimeTrackC(t.Name())

	testcommon.ClearDockerEnv()
	app, err := ddevapp.NewApp(site.Dir, true)
	require.NoError(t, err)
	app.CorepackEnable = false
	app.NodeJSRoot = "frontend"
	app.NodeJSVersion = "auto"

	nvmrcFile := filepath.Join(app.AppRoot, ".nvmrc")
	nodeJSRootDir := filepath.Join(app.AppRoot, app.NodeJSRoot)
	t.Cleanup(func() {
		runTime()
		_ = os.Chdir(origDir)
		_ = os.Remove(nvmrcFile)
		_ = os.RemoveAll(nodeJSRootDir)
		_ = app.Stop(true, false)
	})

	// The project root gets a .nvmrc of its own, with a different version, so a
	// pass can't come from reading the wrong directory.
	testdataDir := filepath.Join(origDir, "testdata", t.Name())
	require.NoError(t, fileutil.CopyFile(filepath.Join(testdataDir, ".nvmrc"), nvmrcFile))
	_ = os.RemoveAll(nodeJSRootDir)
	require.NoError(t, fileutil.CopyDir(filepath.Join(testdataDir, app.NodeJSRoot), nodeJSRootDir))
	subdirVersion, err := fileutil.ReadFileIntoString(filepath.Join(nodeJSRootDir, ".nvmrc"))
	require.NoError(t, err)
	subdirVersion = strings.TrimSpace(subdirVersion)

	err = app.Restart()
	require.NoError(t, err)
	out, _, err := app.Exec(&ddevapp.ExecOpts{
		Cmd: "node --version",
	})
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(out, "v"+subdirVersion), "Expected node version from nodejs_root to start with '%s', but got %s", subdirVersion, out)
}

// TestCorepackEnable tests behavior of corepack_enable, both where Node.js
// bundles corepack (<=24) and where DDEV must install it from npm (>=25)
func TestCorepackEnable(t *testing.T) {
	site := TestSites[0]
	origDir, _ := os.Getwd()
	_ = os.Chdir(site.Dir)

	runTime := util.TimeTrackC(t.Name())

	testcommon.ClearDockerEnv()
	app, err := ddevapp.NewApp(site.Dir, true)
	require.NoError(t, err)
	origCorepackEnable := app.CorepackEnable
	origNodeJSVersion := app.NodeJSVersion
	preDockerfile := app.GetConfigPath("web-build/pre.Dockerfile.corepack-test")
	t.Cleanup(func() {
		runTime()
		_ = os.Chdir(origDir)
		_ = os.Remove(preDockerfile)
		app.CorepackEnable = origCorepackEnable
		app.NodeJSVersion = origNodeJSVersion
		_ = app.Restart()
	})

	for _, tc := range []struct {
		nodeJSVersion string
		// Simulates a base image built with Node.js >=25, which has no
		// corepack; otherwise the one Node.js 24 left behind survives `n install`
		removeBaseCorepack bool
	}{
		{nodeJSVersion: "24"},
		{nodeJSVersion: "26", removeBaseCorepack: true},
	} {
		t.Run("node"+tc.nodeJSVersion, func(t *testing.T) {
			_ = os.Remove(preDockerfile)
			if tc.removeBaseCorepack {
				// Not `npm uninstall -g corepack`, which also removes the yarn 1 bin links
				err := os.WriteFile(preDockerfile, []byte("RUN rm -rf /usr/local/n/lib/node_modules/corepack /usr/local/n/bin/corepack\n"), 0644)
				require.NoError(t, err)
			}
			app.NodeJSVersion = tc.nodeJSVersion

			app.CorepackEnable = false
			err := app.Start()
			require.NoError(t, err)
			out, _, err := app.Exec(&ddevapp.ExecOpts{
				Cmd: `node --version`,
			})
			require.NoError(t, err)
			require.True(t, strings.HasPrefix(out, "v"+tc.nodeJSVersion+"."), "Expected node version to start with 'v%s.', but got %s", tc.nodeJSVersion, out)
			out, _, err = app.Exec(&ddevapp.ExecOpts{
				Cmd: `command -v yarn && ls -l "$(command -v yarn)"`,
			})
			require.NoError(t, err)
			require.Contains(t, out, "/usr/local/n/bin/yarn")
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
				Cmd: `command -v yarn && ls -l "$(command -v yarn)"`,
			})
			require.NoError(t, err)
			require.Contains(t, out, "/usr/local/n/bin/yarn")
			require.Contains(t, out, "corepack")

			// Outside the project, so the packageManager field yarn writes
			// can't leak into the next subtest or later tests
			out, _, err = app.Exec(&ddevapp.ExecOpts{
				Cmd: `mkdir -p /tmp/corepack-test && cd /tmp/corepack-test && echo '{}' >package.json && yarn set version stable >/dev/null && yarn --version`,
			})
			require.NoError(t, err)
			require.True(t, strings.HasPrefix(out, "4."), "Expected yarn 4.x, but got %s", out)
		})
	}
}

// TestNpmGlobalInstall tests that `npm install -g` works for the project
// user, who is not the file owner of /usr/local/n but a member of its
// group (0/root) - regression test for
// https://github.com/ddev/ddev/issues/8640
func TestNpmGlobalInstall(t *testing.T) {
	site := TestSites[0]
	origDir, _ := os.Getwd()
	_ = os.Chdir(site.Dir)

	runTime := util.TimeTrackC(t.Name())

	testcommon.ClearDockerEnv()
	app, err := ddevapp.NewApp(site.Dir, true)
	require.NoError(t, err)
	t.Cleanup(func() {
		runTime()
		_ = os.Chdir(origDir)
		_ = app.Stop(true, false)
	})

	err = app.Start()
	require.NoError(t, err)

	out, stderr, err := app.Exec(&ddevapp.ExecOpts{
		Cmd: `npm install -g npm@latest`,
		// Retry registry fetches that fail on transient network errors.
		Env: []string{"npm_config_fetch_retries=5", "npm_config_fetch_retry_mintimeout=2000", "npm_config_fetch_retry_maxtimeout=20000"},
	})
	if err != nil {
		// npm's stderr only points at its debug log, which has the real cause.
		debugLog, _, _ := app.Exec(&ddevapp.ExecOpts{
			Cmd: `node --version; npm --version; f=$(ls -t "$(npm config get cache)"/_logs/*-debug-*.log 2>/dev/null | head -1); [ -n "$f" ] && tail -n 100 "$f"`,
		})
		require.NoError(t, err, "npm install -g failed\nstdout: %s\nstderr: %s\nnpm debug log:\n%s", out, stderr, debugLog)
	}
	require.NotContains(t, out+stderr, "EACCES")

	out, _, err = app.Exec(&ddevapp.ExecOpts{
		Cmd: `npm --version`,
	})
	require.NoError(t, err)
	require.NotEmpty(t, strings.TrimSpace(out))
}
