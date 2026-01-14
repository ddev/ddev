package ddevapp_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/ddev/ddev/pkg/util"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProcessHooks tests execution of commands defined in config.yaml
func TestProcessHooks(t *testing.T) {
	// Don't run this unless GOTEST_SHORT is unset; it doesn't need to be run everywhere.
	if os.Getenv("GOTEST_SHORT") != "" {
		t.Skip("Skip because GOTEST_SHORT is set")
	}

	if nodeps.IsWindows() {
		t.Skip("Skipping on traditional Windows, as it always hangs")
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
	err = app.Restart()
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
	tasks := map[string]taskExpectation{
		//"composer install":                     {"composer: install", "", "Running task: Composer command '[install]' in web container"},
		//"composer licenses":                    {"composer: licenses --format=json", "no-version-set", "Running task: Composer command 'licenses --format=json' in web container"},
		//"composer with exec_raw":               {"composer:\n    exec_raw: [licenses, --format=json]", "no-version-set", "Running task: Composer command '[licenses --format=json]' in web container"},
		"exec ls in web":                       {"exec: ls /usr/local/bin", "acli\nbuild_php_extension.sh\ncomposer", "Running task: Exec command 'ls /usr/local/bin'"},
		"exec-host echo":                       {"exec-host: \"echo something\"", "something\n", "Running task: Exec command 'echo something' on the host"},
		"exec with service db":                 {"exec: echo MYSQL_HISTFILE=${MYSQL_HISTFILE:-}\n    service: db", "MYSQL_HISTFILE=/mnt/ddev-global-cache/mysqlhistory", "Running task: Exec command 'echo MYSQL_HISTFILE=${MYSQL_HISTFILE:-}' in container/service 'db'"},
		"exec with user root string":           {"exec: ls -la /root\n    service: db\n    user: root", "total ", "Running task: Exec command 'ls -la /root' in container/service 'db'"},
		"exec with user 0 integer":             {"exec: ls -la /root\n    service: db\n    user: 0", "total ", "Running task: Exec command 'ls -la /root' in container/service 'db'"},
		"exec with environment variable":       {"exec: \"echo TestProcessHooks > /var/www/html/TestProcessHooks-php-version-${DDEV_PHP_VERSION}.txt\"", "", "Running task: Exec command 'echo TestProcessHooks > /var/www/html/TestProcessHooks-php-version-${DDEV_PHP_VERSION}.txt'"},
		"exec with multiple commands using &&": {"exec: \"touch /var/tmp/TestProcessHooks && touch /var/www/html/touch_works_after_and.txt\"", "", "Running task: Exec command 'touch /var/tmp/TestProcessHooks && touch /var/www/html/touch_works_after_and.txt'"},
		"exec with exec_raw array":             {"exec:\n    exec_raw: [ls, /usr/local]", "bin\netc\ngames\n", "Exec command '[ls /usr/local] (raw)'"},
	}
	for name, task := range tasks {
		t.Run(name, func(t *testing.T) {
			fName := app.GetConfigPath("config.hooks.yaml")
			fullTask := []byte("hooks:\n  post-start:\n  - " + task.task + "\n")
			err = os.WriteFile(fName, fullTask, 0644)
			require.NoError(t, err)

			app, err = ddevapp.NewApp(site.Dir, true)
			require.NoError(t, err)

			captureOutputFunc, err := util.CaptureOutputToFile()
			require.NoError(t, err, `failed to capture output to file for task='%v' err=%v`, task, err)
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
		})
	}

	err = app.Restart()
	require.NoError(t, err)

	t.Run("verify file creation from hooks", func(t *testing.T) {
		require.FileExists(t, filepath.Join(app.AppRoot, "TestProcessHooks-php-version-"+app.PHPVersion+".txt"))
		require.FileExists(t, filepath.Join(app.AppRoot, "touch_works_after_and.txt"))
	})

	t.Run("skip hooks when SkipHooks is true", func(t *testing.T) {
		ddevapp.SkipHooks = true
		defer func() { ddevapp.SkipHooks = false }()

		app.Hooks = map[string][]ddevapp.YAMLTask{
			"hook-test-skip-hooks": {
				{"exec": "\"echo TestProcessHooks > /var/www/html/TestProcessHooksSkipHooks-php-version-${DDEV_PHP_VERSION}.txt\""},
			},
		}
		err = app.ProcessHooks("hook-test")
		require.NoError(t, err)
		require.NoFileExists(t, filepath.Join(app.AppRoot, "TestProcessHooksSkipHooks-php-version-"+app.PHPVersion+".txt"))
	})

	t.Run("hook failure handling", func(t *testing.T) {
		app.Hooks = map[string][]ddevapp.YAMLTask{
			"hook-test": {
				{"exec": "ls /does-not-exist"},
			},
		}

		t.Run("default setting allows hook failure", func(t *testing.T) {
			err = app.ProcessHooks("hook-test")
			require.NoError(t, err)
		})

		t.Run("FailOnHookFail causes failure", func(t *testing.T) {
			app.FailOnHookFail = true
			defer func() { app.FailOnHookFail = false }()

			err = app.ProcessHooks("hook-test")
			require.Error(t, err)
		})

		t.Run("FailOnHookFailGlobal causes failure", func(t *testing.T) {
			app.FailOnHookFailGlobal = true
			defer func() { app.FailOnHookFailGlobal = false }()

			err = app.ProcessHooks("hook-test")
			require.Error(t, err)
		})

		t.Run("both FailOnHookFail and FailOnHookFailGlobal cause failure", func(t *testing.T) {
			app.FailOnHookFail = true
			app.FailOnHookFailGlobal = true
			defer func() {
				app.FailOnHookFail = false
				app.FailOnHookFailGlobal = false
			}()

			err = app.ProcessHooks("hook-test")
			require.Error(t, err)
		})
	})

	t.Run("pre-share and post-share hooks", func(t *testing.T) {
		app.Hooks = map[string][]ddevapp.YAMLTask{
			"pre-share": {
				{"exec-host": "touch " + filepath.Join(app.AppRoot, "pre-share-hook-ran.txt")},
			},
			"post-share": {
				{"exec-host": "touch " + filepath.Join(app.AppRoot, "post-share-hook-ran.txt")},
			},
		}

		t.Run("pre-share hook executes", func(t *testing.T) {
			err = app.ProcessHooks("pre-share")
			require.NoError(t, err)
			require.FileExists(t, filepath.Join(app.AppRoot, "pre-share-hook-ran.txt"))
		})

		t.Run("post-share hook executes", func(t *testing.T) {
			err = app.ProcessHooks("post-share")
			require.NoError(t, err)
			require.FileExists(t, filepath.Join(app.AppRoot, "post-share-hook-ran.txt"))
		})
	})
}
