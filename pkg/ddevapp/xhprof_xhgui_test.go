package ddevapp_test

import (
	"fmt"
	"github.com/ddev/ddev/pkg/config/types"
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/ddev/ddev/pkg/util"
	assert2 "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// TestDdevXhprofPrependEnabled tests running with xhprof_enabled = true and xhprof_mode=prepend
func TestDdevXhprofPrependEnabled(t *testing.T) {
	if runtime.GOOS == "darwin" && os.Getenv("DDEV_RUN_TEST_ANYWAY") != "true" {
		// TODO: Return to this when working on xhprof xhgui etc.
		t.Skip("Skipping on darwin to ignore problems with 'connection reset by peer'")
	}

	origDir, _ := os.Getwd()

	testcommon.ClearDockerEnv()

	projDir := testcommon.CreateTmpDir(t.Name())
	app, err := ddevapp.NewApp(projDir, false)
	require.NoError(t, err)

	if app.GetWebserverType() == nodeps.AppTypeGeneric {
		t.Skip("Xhprof is not tested on generic webserver")
	}

	app.Type = nodeps.AppTypePHP
	app.Name = t.Name()
	err = app.WriteConfig()
	require.NoError(t, err)

	// Create the simplest possible php file
	err = fileutil.TemplateStringToFile("<?php\nphpinfo();\n", nil, filepath.Join(app.AppRoot, "index.php"))
	require.NoError(t, err)

	runTime := util.TimeTrackC(fmt.Sprintf("%s %s", app.Name, t.Name()))

	// Does not work with php5.6 anyway (SEGV), for resource conservation
	// skip older unsupported versions
	phpKeys := nodeps.GetValidPHPVersions()
	exclusions := []string{nodeps.PHP56}
	phpKeys = util.SubtractSlices(phpKeys, exclusions)
	sort.Strings(phpKeys)

	// If GOTESt_SHORT is set, we'll just use the default version instead
	if os.Getenv("GOTEST_SHORT") != "" {
		phpKeys = []string{nodeps.PHPDefault}
	}

	err = app.Init(app.AppRoot)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, err = exec.RunHostCommand(DdevBin, "config", "global", "--xhprof-mode-reset")
		require.NoError(t, err)

		_ = app.Stop(true, false)
		_ = os.Chdir(origDir)
		_ = os.RemoveAll(projDir)
	})

	globalconfig.DdevGlobalConfig.XHProfMode = "prepend"
	err = globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig)
	require.NoError(t, err)

	webserverKeys := nodeps.GetPHPWebserverTypes()
	// Most of the time we can just test with the default webserver_type
	if os.Getenv("GOTEST_SHORT") != "" {
		webserverKeys = []string{nodeps.WebserverDefault}
	}

	for _, webserverKey := range webserverKeys {
		app.WebserverType = webserverKey

		for _, v := range phpKeys {
			t.Logf("Beginning XHProf checks with XHProf webserver_type=%s php%s\n", webserverKey, v)
			fmt.Printf("Attempting XHProf checks with XHProf PHP%s\n", v)
			app.PHPVersion = v

			err = app.Restart()
			require.NoError(t, err)

			stdout, _, err := app.Exec(&ddevapp.ExecOpts{
				Service: "web",
				Cmd:     "php --ri xhprof",
			})
			require.Error(t, err)
			require.Contains(t, stdout, "Extension 'xhprof' not present")

			// Run with Xhprof enabled
			_, _, err = app.Exec(&ddevapp.ExecOpts{
				Cmd: "enable_xhprof",
			})
			require.NoError(t, err)

			stdout, _, err = app.Exec(&ddevapp.ExecOpts{
				Service: "web",
				Cmd:     "php --ri xhprof",
			})
			require.NoError(t, err)
			require.Contains(t, stdout, "xhprof.output_dir", "xhprof should be enabled but is not enabled for %s", v)

			out, _, err := testcommon.GetLocalHTTPResponse(t, app.GetPrimaryURL(), 2)
			require.NoError(t, err, "Failed to get base URL webserver_type=%s, php_version=%s", webserverKey, v)
			require.Contains(t, out, "module_xhprof")

			out, _, err = testcommon.GetLocalHTTPResponse(t, app.GetPrimaryURL()+"/xhprof/", 2)
			require.NoError(t, err)
			// Output should contain at least one run
			require.Contains(t, out, ".ddev.xhprof</a><small>")

			// Disable all to avoid confusion
			_, _, err = app.Exec(&ddevapp.ExecOpts{
				Cmd: "disable_xhprof && rm -rf /tmp/xhprof/*",
			})
			require.NoError(t, err)
		}
	}
	runTime()
}

// TestDdevXhprofXhguiEnabled tests running with xhprof_enabled = true and xhprof_mode=xhgui
func TestDdevXhprofXhguiEnabled(t *testing.T) {
	if runtime.GOOS == "darwin" && os.Getenv("DDEV_RUN_TEST_ANYWAY") != "true" {
		// TODO: Return to this when working on xhprof xhgui etc.
		t.Skip("Skipping on darwin to ignore problems with 'connection reset by peer'")
	}

	assert := assert2.New(t)
	origDir, _ := os.Getwd()

	testcommon.ClearDockerEnv()

	projDir := testcommon.CreateTmpDir(t.Name())
	app, err := ddevapp.NewApp(projDir, false)
	require.NoError(t, err)

	if app.GetWebserverType() == nodeps.AppTypeGeneric {
		t.Skip("Xhprof is not tested on generic webserver")
	}

	app.Type = nodeps.AppTypePHP
	app.XHProfMode = types.XHProfModeXHGui
	app.Name = t.Name()

	err = app.WriteConfig()
	require.NoError(t, err)

	// Create the simplest possible php file
	err = fileutil.TemplateStringToFile("<?php\nphpinfo();\n", nil, filepath.Join(app.AppRoot, "index.php"))
	require.NoError(t, err)

	runTime := util.TimeTrackC(fmt.Sprintf("%s %s", app.Name, t.Name()))

	// Does not work with php5.6 anyway (SEGV), for resource conservation
	// skip older unsupported versions
	phpKeys := nodeps.GetValidPHPVersions()
	exclusions := []string{nodeps.PHP56}
	phpKeys = util.SubtractSlices(phpKeys, exclusions)
	sort.Strings(phpKeys)

	// If GOTESt_SHORT is set, we'll just use the default version instead
	if os.Getenv("GOTEST_SHORT") != "" {
		phpKeys = []string{nodeps.PHPDefault}
	}

	err = app.Init(app.AppRoot)
	require.NoError(t, err)

	t.Cleanup(func() {
		err = app.Stop(true, false)
		assert.NoError(err)
		err = os.Chdir(origDir)
		assert.NoError(err)
		_ = os.RemoveAll(projDir)
	})

	webserverKeys := nodeps.GetPHPWebserverTypes()
	// Most of the time we can just test with the default webserver_type
	if os.Getenv("GOTEST_SHORT") != "" {
		webserverKeys = []string{nodeps.WebserverDefault}
	}

	for _, webserverKey := range webserverKeys {
		app.WebserverType = webserverKey

		for _, v := range phpKeys {
			t.Logf("Beginning XHProf checks with XHProf webserver_type=%s php%s\n", webserverKey, v)
			fmt.Printf("Attempting XHProf checks with XHProf PHP%s\n", v)
			app.PHPVersion = v
			app.XHProfMode = types.XHProfModeXHGui

			err = app.Restart()
			require.NoError(t, err)

			stdout, _, err := app.Exec(&ddevapp.ExecOpts{
				Service: "web",
				Cmd:     "php --ri xhprof",
			})
			require.Error(t, err)
			assert.Contains(stdout, "Extension 'xhprof' not present")

			err = ddevapp.XHGuiSetup(app)
			require.NoError(t, err)

			// Start optional xhgui profile
			err = app.StartOptionalProfiles([]string{"xhgui"})
			assert.NoError(err)
			_, _, err = app.Exec(&ddevapp.ExecOpts{
				Cmd: "enable_xhprof",
			})
			require.NoError(t, err)

			stdout, _, err = app.Exec(&ddevapp.ExecOpts{
				Service: "web",
				Cmd:     "php --ri xhprof",
			})
			require.NoError(t, err)
			assert.Contains(stdout, "xhprof.output_dir", "xhprof should be enabled but is not enabled for %s", v)

			out, _, err := testcommon.GetLocalHTTPResponse(t, app.GetPrimaryURL(), 2)
			require.NoError(t, err, "Failed to get base URL webserver_type=%s, php_version=%s", webserverKey, v)
			require.Contains(t, out, "module_xhprof")

			// NOW try using xhgui

			xhguiURL := app.GetXHGuiURL()
			out, _, err = testcommon.GetLocalHTTPResponse(t, xhguiURL, 2)
			require.NoError(t, err)
			// Output should contain at least one run
			require.Contains(t, out, strings.ToLower(t.Name()+"."+nodeps.DdevDefaultTLD))
			require.Contains(t, out, "Recent runs")

			// Disable all to avoid confusion
			_, _, err = app.Exec(&ddevapp.ExecOpts{
				Cmd: "disable_xhprof && rm -rf /tmp/xhprof/*",
			})
			require.NoError(t, err)
		}
	}
	runTime()
}
