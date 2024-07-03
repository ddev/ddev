package ddevapp_test

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/ddev/ddev/pkg/util"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProcessHooks tests execution of commands defined in config.yaml
func TestProcessHooks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows, as it always hangs")
	}
	assert := asrt.New(t)

	site := TestSites[0]
	origDir, _ := os.Getwd()
	runTime := util.TimeTrackC(t.Name())

	testcommon.ClearDockerEnv()
	app, err := ddevapp.NewApp(site.Dir, true)
	require.NoError(t, err)

	// We don't get the expected task debug output without DDEV_DEBUG
	origDdevDebug := os.Getenv("DDEV_DEBUG")
	_ = os.Setenv(`DDEV_DEBUG`, `true`) // test requires DDEV_DEBUG to see task output

	t.Cleanup(func() {
		runTime()
		err = os.Chdir(origDir)
		assert.NoError(err)
		err = app.Stop(true, false)
		assert.NoError(err)
		_ = os.RemoveAll(app.GetConfigPath("config.hooks.yaml"))
		_ = os.RemoveAll(filepath.Join(app.AppRoot, "composer.json"))
		_ = os.Setenv(`DDEV_DEBUG`, origDdevDebug)
	})
	err = app.Start()
	require.NoError(t, err)

	// Create a composer.json so we can do actions on it.
	fName := filepath.Join(app.AppRoot, "composer.json")
	err = os.WriteFile(fName, []byte("{}"), 0644)
	require.NoError(t, err)

	type taskExpectation struct {
		task             string
		stdoutExpect     string
		fulloutputExpect string
	}
	// ExecHost commands must be able to run on Windows.
	// echo and pwd are things that work pretty much the same in both places.

	// 2022-02-16: I'm unable to get the Composer examples to work here. Intermittent results
	// Half the time they work and get expected Composer output, the other half they come up with empty string.
	tasks := []taskExpectation{
		//{"composer: install", "", "Running task: Composer command '[install]' in web container"},
		//{"composer: licenses --format=json", "no-version-set", "Running task: Composer command 'licenses --format=json' in web container"},
		//{"composer:\n    exec_raw: [licenses, --format=json]", "no-version-set", "Running task: Composer command '[licenses --format=json]' in web container"},
		{"exec: ls /usr/local/bin", "acli\ncomposer", "Running task: Exec command 'ls /usr/local/bin'"},
		{"exec-host: \"echo something\"", "something\n", "Running task: Exec command 'echo something' on the host"},
		{"exec: echo MYSQL_PWD=${MYSQL_PWD:-}\n    service: db", "MYSQL_PWD=db\n", "Running task: Exec command 'echo MYSQL_PWD=${MYSQL_PWD:-}' in container/service 'db'"},
		{"exec: \"echo TestProcessHooks > /var/www/html/TestProcessHooks${DDEV_ROUTER_HTTPS_PORT}.txt\"", "", "Running task: Exec command 'echo TestProcessHooks > /var/www/html/TestProcessHooks${DDEV_ROUTER_HTTPS_PORT}.txt'"},
		{"exec: \"touch /var/tmp/TestProcessHooks && touch /var/www/html/touch_works_after_and.txt\"", "", "Running task: Exec command 'touch /var/tmp/TestProcessHooks && touch /var/www/html/touch_works_after_and.txt'"},
		{"exec:\n    exec_raw: [ls, /usr/local]", "bin\netc\ngames\n", "Exec command '[ls /usr/local] (raw)'"},
	}
	for _, task := range tasks {
		fName := app.GetConfigPath("config.hooks.yaml")
		fullTask := []byte("hooks:\n  post-start:\n  - " + task.task + "\n")
		err = os.WriteFile(fName, fullTask, 0644)
		require.NoError(t, err)

		app, err = ddevapp.NewApp(site.Dir, true)
		require.NoError(t, err)

		captureOutputFunc, err := util.CaptureOutputToFile()
		require.NoError(t, err, `failed to capture output to file for taxk='%v' err=%v`, task, err)
		userOutFunc := util.CaptureUserOut()

		err = app.Start()
		require.NoError(t, err, `failed to app.Start() for task '%v' err='%v'`, task, err)

		out := captureOutputFunc()
		userOut := userOutFunc()
		require.Contains(t, out, task.stdoutExpect, "task: '%v'", task.task)
		require.Contains(t, userOut, task.fulloutputExpect, "task: %v", task.task)
		require.NotContains(t, userOut, "Task failed")

		err = app.Stop(true, false)
		require.NoError(t, err)
	}

	err = app.Restart()
	require.NoError(t, err)

	require.FileExists(t, filepath.Join(app.AppRoot, fmt.Sprintf("TestProcessHooks%s.txt", app.GetRouterHTTPSPort())))
	require.FileExists(t, filepath.Join(app.AppRoot, "touch_works_after_and.txt"))

	// Make sure skip hooks work
	ddevapp.SkipHooks = true
	app.Hooks = map[string][]ddevapp.YAMLTask{
		"hook-test-skip-hooks": {
			{"exec": "\"echo TestProcessHooks > /var/www/html/TestProcessHooksSkipHooks${DDEV_ROUTER_HTTPS_PORT}.txt\""},
		},
	}
	err = app.ProcessHooks("hook-test")
	require.NoError(t, err)
	require.NoFileExists(t, filepath.Join(app.AppRoot, fmt.Sprintf("TestProcessHooksSkipHooks%s.txt", app.GetRouterHTTPSPort())))
	ddevapp.SkipHooks = false

	// Attempt processing hooks with a guaranteed failure
	app.Hooks = map[string][]ddevapp.YAMLTask{
		"hook-test": {
			{"exec": "ls /does-not-exist"},
		},
	}
	// With default setting, ProcessHooks should succeed
	err = app.ProcessHooks("hook-test")
	require.NoError(t, err)

	// With FailOnHookFail or FailOnHookFailGlobal or both, it should fail.
	app.FailOnHookFail = true
	err = app.ProcessHooks("hook-test")
	require.Error(t, err)
	app.FailOnHookFail = false
	app.FailOnHookFailGlobal = true
	err = app.ProcessHooks("hook-test")
	require.Error(t, err)
	app.FailOnHookFail = true
	err = app.ProcessHooks("hook-test")
	require.Error(t, err)
}
