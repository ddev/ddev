package ddevapp_test

import (
	"fmt"
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/testcommon"
	"github.com/drud/ddev/pkg/util"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestProcessHooks tests execution of commands defined in config.yaml
func TestProcessHooks(t *testing.T) {
	assert := asrt.New(t)

	site := TestSites[0]
	origDir, _ := os.Getwd()
	oldDdevDebug := os.Getenv("DDEV_DEBUG")
	// We don't get the expected task debug output without DDEV_DEBUG
	_ = os.Setenv("DDEV_DEBUG", "true")
	runTime := util.TimeTrack(time.Now(), t.Name())

	testcommon.ClearDockerEnv()
	app, err := ddevapp.NewApp(site.Dir, true)
	assert.NoError(err)
	t.Cleanup(func() {
		runTime()
		err = os.Chdir(origDir)
		assert.NoError(err)
		err = app.Stop(true, false)
		assert.NoError(err)
		err = os.RemoveAll(app.GetConfigPath("config.hooks.yaml"))
		assert.NoError(err)
		err = os.RemoveAll(filepath.Join(app.AppRoot, "composer.json"))
		assert.NoError(err)
		_ = os.Setenv("DDEV_DEBUG", oldDdevDebug)
	})
	err = app.Start()
	assert.NoError(err)

	// create a composer.json just so we can do actions on it.
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

	// 2022-02-16: I'm unable to get the composer examples to work here. Intermittent results
	// Half the time they work and get expected composer output, the other half they come up with empty string.
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
		assert.NoError(err)

		app, err = ddevapp.NewApp(site.Dir, true)
		assert.NoError(err)

		captureOutputFunc, err := util.CaptureOutputToFile()
		assert.NoError(err)
		userOutFunc := util.CaptureUserOut()

		err = app.Start()
		assert.NoError(err)

		out := captureOutputFunc()
		userOut := userOutFunc()
		assert.Contains(out, task.stdoutExpect, "task: %v", task.task)
		assert.Contains(userOut, task.fulloutputExpect, "task: %v", task.task)
		assert.NotContains(userOut, "Task failed")
	}

	err = app.MutagenSyncFlush()
	assert.NoError(err)

	assert.FileExists(filepath.Join(app.AppRoot, fmt.Sprintf("TestProcessHooks%s.txt", app.RouterHTTPSPort)))
	assert.FileExists(filepath.Join(app.AppRoot, "touch_works_after_and.txt"))

	// Attempt processing hooks with a guaranteed failure
	app.Hooks = map[string][]ddevapp.YAMLTask{
		"hook-test": {
			{"exec": "ls /does-not-exist"},
		},
	}
	// With default setting, ProcessHooks should succeed
	err = app.ProcessHooks("hook-test")
	assert.NoError(err)

	// With FailOnHookFail or FailOnHookFailGlobal or both, it should fail.
	app.FailOnHookFail = true
	err = app.ProcessHooks("hook-test")
	assert.Error(err)
	app.FailOnHookFail = false
	app.FailOnHookFailGlobal = true
	err = app.ProcessHooks("hook-test")
	assert.Error(err)
	app.FailOnHookFail = true
	err = app.ProcessHooks("hook-test")
	assert.Error(err)
}
