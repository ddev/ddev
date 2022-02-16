package ddevapp_test

import (
	"fmt"
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/testcommon"
	"github.com/drud/ddev/pkg/util"
	asrt "github.com/stretchr/testify/assert"
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
	})
	err = app.Start()
	assert.NoError(err)

	type taskExpectation struct {
		task             string
		stdoutExpect     string
		fulloutputExpect string
	}
	// ExecHost commands must be able to run on Windows.
	// echo and pwd are things that work pretty much the same in both places.
	tasks := []taskExpectation{
		{"exec: ls /usr/local/bin", "acli\ncomposer", "Running task: Exec command 'ls /usr/local/bin'"},
		{"exec-host: \"echo something\"", "something\n", "Running task: Exec command 'echo something' on the host"},
		{"exec: echo MYSQL_USER=${MYSQL_USER}\n    service: db", "MYSQL_USER=db\n", "Running task: Exec command 'echo MYSQL_USER=${MYSQL_USER}' in container/service 'db'"},
		{"exec: \"echo TestProcessHooks > /var/www/html/TestProcessHooks${DDEV_ROUTER_HTTPS_PORT}.txt\"", "", "Running task: Exec command 'echo TestProcessHooks > /var/www/html/TestProcessHooks${DDEV_ROUTER_HTTPS_PORT}.txt'"},
		{"exec: \"touch /var/tmp/TestProcessHooks && touch /var/www/html/touch_works_after_and.txt\"", "", "Running task: Exec command 'touch /var/tmp/TestProcessHooks && touch /var/www/html/touch_works_after_and.txt'"},
		{"exec:\n    exec_raw: [ls, /usr/local]", "bin\netc\ngames\ninclude\nlib\nman\nsbin\nshare\nsrc\n", "Exec command '[ls /usr/local] (raw)'"},
	}
	for _, task := range tasks {
		fName := app.GetConfigPath("config.hooks.yaml")
		fullTask := []byte("hooks:\n  post-start:\n  - " + task.task + "\n")
		err = os.WriteFile(fName, fullTask, 0644)
		assert.NoError(err)

		_, err = app.ReadConfig(true)
		assert.NoError(err)

		captureOutputFunc, err := util.CaptureOutputToFile()
		assert.NoError(err)
		userOutFunc := util.CaptureUserOut()

		err = app.Start()
		assert.NoError(err)

		out := captureOutputFunc()
		userOut := userOutFunc()
		assert.Contains(out, task.stdoutExpect)
		assert.Contains(userOut, task.fulloutputExpect)
		assert.NotContains(userOut, "Task failed")
	}

	err = app.MutagenSyncFlush()
	assert.NoError(err)

	assert.FileExists(filepath.Join(app.AppRoot, fmt.Sprintf("TestProcessHooks%s.txt", app.RouterHTTPSPort)))
	assert.FileExists(filepath.Join(app.AppRoot, "touch_works_after_and.txt"))

	//// Attempt processing hooks with a guaranteed failure
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
