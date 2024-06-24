package cmd

import (
	"fmt"
	"os"
	osexec "os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/ddev/ddev/pkg/util"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCustomCommands does basic checks to make sure custom commands work OK.
func TestCustomCommands(t *testing.T) {
	assert := asrt.New(t)
	runTime := util.TimeTrackC(t.Name())

	origDir, _ := os.Getwd()

	site := TestSites[0]
	err := os.Chdir(site.Dir)
	require.NoError(t, err)

	app, err := ddevapp.NewApp("", false)
	assert.NoError(err)

	tmpXdgConfigHomeDir := testcommon.CopyGlobalDdevDir(t)

	testdataCustomCommandsDir := filepath.Join(origDir, "testdata", t.Name())

	origType := app.Type
	t.Cleanup(func() {
		err = os.Chdir(origDir)
		assert.NoError(err)
		err = app.Stop(true, false)
		assert.NoError(err)
		testcommon.ResetGlobalDdevDir(t, tmpXdgConfigHomeDir)
		runTime()
		app.Type = origType
		err = app.WriteConfig()
		assert.NoError(err)
		_ = fileutil.PurgeDirectory(filepath.Join(site.Dir, ".ddev", "commands"))
	})
	// We must start the app before copying commands, so they don't get copied over
	err = app.Start()
	require.NoError(t, err)

	tmpHomeGlobalCommandsDir := filepath.Join(globalconfig.GetGlobalDdevDir(), "commands")
	err = os.RemoveAll(tmpHomeGlobalCommandsDir)
	assert.NoError(err)

	projectCommandsDir := app.GetConfigPath("commands")
	err = fileutil.CopyDir(filepath.Join(testdataCustomCommandsDir, "global_commands"), tmpHomeGlobalCommandsDir)
	require.NoError(t, err)

	// Must sync our added commands before using them.
	err = app.MutagenSyncFlush()
	assert.NoError(err)

	// We need to run some assertions outside of the context of a project first
	err = os.Chdir(tmpXdgConfigHomeDir)
	require.NoError(t, err)

	// Check that only global host commands with the XXX annotation display here
	out, err := exec.RunHostCommand(DdevBin)
	assert.NoError(err)
	assert.Contains(out, "testhostglobal-noproject global (global shell host container command)")
	assert.NotContains(out, "testhostcmd project (shell host container command)")
	assert.NotContains(out, "testwebcmd project (shell web container command)")
	assert.NotContains(out, "testhostglobal global (global shell host container command)")
	assert.NotContains(out, "testwebglobal global (global shell web container command)")
	assert.NotContains(out, "testhostcmd global")
	assert.NotContains(out, "testwebcmd global")
	assert.NotContains(out, "not-a-command")

	out, err = exec.RunHostCommand(DdevBin, "testhostglobal-noproject", "hostarg1", "hostarg2", "--hostflag1")
	assert.NoError(err)
	expectedHost, _ := os.Hostname()
	assert.Contains(out, fmt.Sprintf("%s was executed with args=hostarg1 hostarg2 --hostflag1 on host %s", "testhostglobal-noproject", expectedHost))

	// The remaining assertions will be performed inside the project dir
	err = os.Chdir(site.Dir)
	require.NoError(t, err)

	out, err = exec.RunHostCommand(DdevBin, "debug", "fix-commands")
	require.NoError(t, err, "failed to run ddev debug fix-commands, out='%s', err=%v", out, err)
	out, err = exec.RunHostCommand(DdevBin)
	require.NoError(t, err, "failed to run DDEV command, output='%s', err=%v", out, err)
	assert.Contains(out, "mysql client in db container")

	// Test the `ddev mysql` command with stdin
	inputFile := filepath.Join(testdataCustomCommandsDir, "select99.sql")
	f, err := os.Open(inputFile)
	require.NoError(t, err)
	t.Cleanup(func() {
		err = f.Close()
		assert.NoError(err)
	})
	command := osexec.Command(DdevBin, "mysql")
	command.Stdin = f
	byteOut, err := command.CombinedOutput()
	require.NoError(t, err, "Failed ddev mysql; output=%v", string(byteOut))
	assert.Contains(string(byteOut), "99\n99\n")

	err = os.RemoveAll(projectCommandsDir)
	assert.NoError(err)

	// Now copy a project commands and global commands and make sure they show up and execute properly
	err = fileutil.CopyDir(filepath.Join(testdataCustomCommandsDir, "project_commands"), projectCommandsDir)
	assert.NoError(err)
	_, _ = exec.RunHostCommand(DdevBin, "debug", "fix-commands")

	// Must sync our added in-container commands before using them.
	err = app.MutagenSyncFlush()
	assert.NoError(err)

	out, err = exec.RunHostCommand(DdevBin)
	assert.NoError(err)
	assert.Contains(out, "testhostglobal-noproject global (global shell host container command)")
	assert.Contains(out, "testhostcmd project (shell host container command)")
	assert.Contains(out, "testwebcmd project (shell web container command)")
	assert.Contains(out, "testhostglobal global (global shell host container command)")
	assert.Contains(out, "testwebglobal global (global shell web container command)")
	assert.NotContains(out, "testhostcmd global") //the global testhostcmd should have been overridden by the project one
	assert.NotContains(out, "testwebcmd global")  //the global testwebcmd should have been overridden by the project one
	assert.NotContains(out, "not-a-command")

	// Have to do app.Start() because commands are copied into containers on start
	err = app.Start()
	require.NoError(t, err)
	for _, c := range []string{"testhostcmd", "testhostglobal", "testhostglobal-noproject", "testwebcmd", "testwebglobal"} {
		out, err = exec.RunHostCommand(DdevBin, c, "hostarg1", "hostarg2", "--hostflag1")
		if err != nil {
			userHome, err := os.UserHomeDir()
			assert.NoError(err)
			globalDdevDir := globalconfig.GetGlobalDdevDir()
			homeEnv := os.Getenv("HOME")
			t.Errorf("userHome=%s, globalDdevDir=%s, homeEnv=%s", userHome, globalDdevDir, homeEnv)
			t.Errorf("Failed to run ddev %s: %v, home=%s output=%s", c, err, userHome, out)
			out, err = exec.RunHostCommand("ls", "-lR", globalDdevDir, "commands")
			assert.NoError(err)
			t.Errorf("Commands dir: %s", out)
		}
		expectedHost, _ := os.Hostname()
		if !strings.Contains(c, "host") {
			expectedHost = site.Name + "-web"
		}
		assert.Contains(out, fmt.Sprintf("%s was executed with args=hostarg1 hostarg2 --hostflag1 on host %s", c, expectedHost))
	}

	// Test line breaks in examples
	c := "testhostcmd"
	out, err = exec.RunHostCommand(DdevBin, "help", c)
	assert.NoError(err, "Failed to run ddev %s %s", "help", c)
	assert.Contains(out, "Examples:\n  ddev testhostcmd\n  ddev testhostcmd -h")

	// Test flags are imported from comments
	c = "testhostcmdflags"
	out, err = exec.RunHostCommand(DdevBin, c, "--test")
	expectedHost, _ = os.Hostname()
	assert.NoError(err, "Failed to run ddev %s %v", c, "--test")
	assert.Contains(out, fmt.Sprintf("%s was executed with args=--test on host %s", c, expectedHost))

	out, err = exec.RunHostCommand(DdevBin, "help", c)
	assert.NoError(err, "Failed to run ddev %s %v", "help", c)
	assert.Contains(out, "  -t, --test   Usage of test")

	// Tests with app type PHP
	app.Type = nodeps.AppTypePHP
	err = app.WriteConfig()
	assert.NoError(err)

	// Make sure that all the official ddev-provided custom commands are usable by checking help
	for _, c := range []string{"launch", "phpmyadmin", "xdebug"} {
		_, err = exec.RunHostCommand(DdevBin, c, "-h")
		assert.NoError(err, "Failed to run ddev %s -h", c)
	}

	for _, c := range []string{"mysql", "npm", "php", "yarn"} {
		_, err = exec.RunHostCommand(DdevBin, c, "--version")
		assert.NoError(err, "Failed to run ddev %s --version", c)
	}

	// See if `ddev python` works for Python app types
	origAppType := app.Type
	for _, appType := range []string{nodeps.AppTypeDjango4, nodeps.AppTypePython} {
		app.Type = appType
		err = app.WriteConfig()
		require.NoError(t, err)
		for _, c := range []string{"python"} {
			out, err = exec.RunHostCommand(DdevBin, c, "--version")
			assert.NoError(err, "Expected ddev python --version to work with apptype=%s but it didn't, output=%s", c, app.Type, out)
		}
	}
	app.Type = origAppType

	// The various CMS commands should not be available here
	for _, c := range []string{"artisan", "cake", "drush", "magento", "typo3", "wp"} {
		_, err = exec.RunHostCommand(DdevBin, c, "-h")
		assert.Error(err, "found command %s when it should not have been there (no error) app.Type=%s", c, app.Type)
	}

	// TYPO3 commands should only be available for type typo3
	app.Type = nodeps.AppTypeTYPO3
	_ = app.WriteConfig()

	_, _ = exec.RunHostCommand(DdevBin, "debug", "fix-commands")
	err = app.MutagenSyncFlush()
	assert.NoError(err)
	for _, c := range []string{"typo3"} {
		_, err = exec.RunHostCommand(DdevBin, "help", c)
		assert.NoError(err)
	}

	// Drupal commands should only be available for type drupal
	app.Type = nodeps.AppTypeDrupal9
	_ = app.WriteConfig()
	_, _ = exec.RunHostCommand(DdevBin)
	err = app.MutagenSyncFlush()
	assert.NoError(err)

	for _, c := range []string{"drush"} {
		_, err = exec.RunHostCommand(DdevBin, "help", c)
		assert.NoError(err)
	}

	// Laravel commands should only be available for type laravel
	app.Type = nodeps.AppTypeLaravel
	_ = app.WriteConfig()
	_, _ = exec.RunHostCommand(DdevBin)
	err = app.MutagenSyncFlush()
	assert.NoError(err)
	for _, c := range []string{"artisan", "pint"} {
		_, err = exec.RunHostCommand(DdevBin, "help", c)
		assert.NoError(err)
	}

	// WordPress commands should only be available for type wordpress
	app.Type = nodeps.AppTypeWordPress
	_ = app.WriteConfig()
	_, _ = exec.RunHostCommand(DdevBin)
	err = app.MutagenSyncFlush()
	assert.NoError(err)
	for _, c := range []string{"wp"} {
		_, err = exec.RunHostCommand(DdevBin, "help", c)
		assert.NoError(err, "expected to find command %s for app.Type=%s", c, app.Type)
	}

	// Craft CMS commands should only be available for type craftcms
	app.Type = nodeps.AppTypeCraftCms
	_ = app.WriteConfig()
	_, _ = exec.RunHostCommand(DdevBin)
	err = app.MutagenSyncFlush()
	assert.NoError(err)
	for _, c := range []string{"craft"} {
		_, err = exec.RunHostCommand(DdevBin, "help", c)
		assert.NoError(err)
	}

	// CakePHP commands should only be available for type cakephp
	app.Type = nodeps.AppTypeCakePHP
	_ = app.WriteConfig()
	_, _ = exec.RunHostCommand(DdevBin)
	err = app.MutagenSyncFlush()
	assert.NoError(err)
	for _, c := range []string{"cake"} {
		_, err = exec.RunHostCommand(DdevBin, "help", c)
		assert.NoError(err)
	}

	// Make sure that the non-command stuff we installed has been copied into /mnt/ddev-global-cache
	commandDirInVolume := "/mnt/ddev-global-cache/global-commands/"
	for _, f := range []string{".gitattributes", "db/mysqldump.example", "db/README.txt", "web/README.txt"} {
		filePathInVolume := path.Join(commandDirInVolume, f)
		out, err = exec.RunHostCommand(DdevBin, "exec", "[ -f "+filePathInVolume+" ] && exit 0 || exit 1")
		assert.NoError(err, filePathInVolume+" does not exist, output=%s", out)
	}

	// Make sure that the non-command stuff we installed is in project commands dir
	for _, f := range []string{".gitattributes", "db/README.txt", "host/README.txt", "host/solrtail.example", "solr/README.txt", "solr/solrtail.example", "web/README.txt"} {
		assert.FileExists(filepath.Join(projectCommandsDir, f))
	}

	// Make sure that the old launch, mysql, and xdebug commands aren't in the project directory
	for _, command := range []string{"db/mysql", "host/launch", "web/xdebug"} {
		cmdPath := app.GetConfigPath(filepath.Join("commands", command))
		assert.False(fileutil.FileExists(cmdPath), "file %s exists but it should not", cmdPath)
	}
}

// TestLaunchCommand tests that the launch command behaves all the ways it should behave
func TestLaunchCommand(t *testing.T) {
	assert := asrt.New(t)

	origDir, _ := os.Getwd()
	// Create a temporary directory and switch to it.
	testDir := testcommon.CreateTmpDir(t.Name())
	err := os.Chdir(testDir)
	assert.NoError(err)

	t.Setenv("DDEV_DEBUG", "true")
	app, err := ddevapp.NewApp(testDir, false)
	require.NoError(t, err)
	err = app.WriteConfig()
	require.NoError(t, err)
	t.Cleanup(func() {
		err = os.Chdir(origDir)
		assert.NoError(err)
		err = app.Stop(true, false)
		assert.NoError(err)
		_ = os.RemoveAll(testDir)
	})

	primaryURLWithoutPort := app.GetPrimaryURL()
	app.RouterHTTPPort = "8080"
	app.RouterHTTPSPort = "8443"
	app.MailpitHTTPPort = "18025"
	app.MailpitHTTPSPort = "18026"
	err = app.WriteConfig()
	assert.NoError(err)
	err = app.Start()
	require.NoError(t, err)

	desc, err := app.Describe(false)
	require.NoError(t, err)
	cases := map[string]string{
		"":                                 app.GetPrimaryURL(),
		"-m":                               desc["mailpit_https_url"].(string),
		"path":                             app.GetPrimaryURL() + "/path",
		"nested/path":                      app.GetPrimaryURL() + "/nested/path",
		"/path-with-slash":                 app.GetPrimaryURL() + "/path-with-slash",
		app.GetPrimaryURL() + "/full-path": app.GetPrimaryURL() + "/full-path",
		"http://example.com":               "http://example.com",
		"https://example.com:443/test":     "https://example.com:443/test",
		":8080":                            desc["httpurl"].(string),
		":8080/http-port-path":             desc["httpurl"].(string) + "/http-port-path",
		":8443":                            desc["httpsurl"].(string),
		":8443/https-port-path":            desc["httpsurl"].(string) + "/https-port-path",
		":18025":                           "http://" + app.GetHostname() + ":18025",
		":18026":                           "https://" + app.GetHostname() + ":18026",
		// if it is impossible to determine the http/https scheme, the default site protocol should be used
		":3000":                     primaryURLWithoutPort + ":3000",
		":3000/with-default-scheme": "https://" + app.GetHostname() + ":3000/with-default-scheme",
	}
	if runtime.GOOS == "windows" {
		// Git-Bash converts single forward slashes to a Windows path
		// Escape it with a second slash, see https://stackoverflow.com/q/58677021
		cases["//path-with-slash"] = app.GetPrimaryURL() + "/path-with-slash"
		delete(cases, "/path-with-slash")
	}
	if app.CanUseHTTPOnly() {
		cases["-m"] = desc["mailpit_url"].(string)
		cases[":3000/with-default-scheme"] = "http://" + app.GetHostname() + ":3000/with-default-scheme"
	}
	for partialCommand, expect := range cases {
		// Try with the base URL, simplest case
		c := DdevBin + `  launch ` + partialCommand + ` | awk '/FULLURL/ {print $2}'`
		out, err := exec.RunHostCommand("bash", "-c", c)
		out = strings.Trim(out, "\r\n")
		assert.NoError(err, `couldn't run "%s"", output=%s`, c, out)
		assert.Equal(out, expect, "output of %s is incorrect with app.RouterHTTPSPort=%s: %s", c, app.RouterHTTPSPort, out)
	}
}

// TestMysqlCommand tests `ddev mysql“
func TestMysqlCommand(t *testing.T) {
	assert := asrt.New(t)

	origDir, _ := os.Getwd()

	// Create a temporary directory and switch to it.
	tmpDir := testcommon.CreateTmpDir(t.Name())
	err := os.Chdir(tmpDir)
	require.NoError(t, err)

	app, err := ddevapp.NewApp(tmpDir, false)
	require.NoError(t, err)

	t.Cleanup(func() {
		err = app.Stop(true, false)
		assert.NoError(err)
		err = os.Chdir(origDir)
		assert.NoError(err)
		err = os.RemoveAll(tmpDir)
		assert.NoError(err)
	})

	err = app.WriteConfig()
	require.NoError(t, err)

	// This populates the project's
	// /mnt/ddev-global-cache/global-commands/ which otherwise doesn't get done until ddev start
	// This matters when --no-bind-mount=true
	_, err = exec.RunHostCommand("ddev")
	assert.NoError(err)

	err = app.Start()
	require.NoError(t, err)

	// Test ddev mysql -uroot -proot mysql
	command := osexec.Command("bash", "-c", "echo 'SHOW TABLES;' | "+DdevBin+" mysql --user=root --password=root --database=mysql")
	byteOut, err := command.CombinedOutput()
	assert.NoError(err, "byteOut=%v", string(byteOut))
	assert.Contains(string(byteOut), `Tables_in_mysql
column_stats
columns_priv`)
}

// TestPsqlCommand tests `ddev psql“
func TestPsqlCommand(t *testing.T) {
	assert := asrt.New(t)

	origDir, _ := os.Getwd()

	// Create a temporary directory and switch to it.
	tmpDir := testcommon.CreateTmpDir(t.Name())
	err := os.Chdir(tmpDir)
	require.NoError(t, err)

	app, err := ddevapp.NewApp(tmpDir, false)
	require.NoError(t, err)
	app.Database = ddevapp.DatabaseDesc{Type: nodeps.Postgres, Version: nodeps.PostgresDefaultVersion}
	t.Cleanup(func() {
		err = app.Stop(true, false)
		assert.NoError(err)
		err = os.Chdir(origDir)
		assert.NoError(err)
		_ = os.RemoveAll(tmpDir)
	})

	err = app.WriteConfig()
	require.NoError(t, err)

	// This populates the project's
	// /mnt/ddev-global-cache/global-commands/ which otherwise doesn't get done until ddev start
	// This matters when --no-bind-mount=true
	_, err = exec.RunHostCommand("ddev")
	assert.NoError(err)

	err = app.Start()
	require.NoError(t, err)

	// Test ddev psql with \l
	out, err := exec.RunHostCommand("bash", "-c", fmt.Sprintf(`echo '\l' | %s psql -t`, DdevBin))
	assert.NoError(err, "out=%s", out)
	assert.Contains(out, `db        | db    | UTF8`)
}

// TestNpmYarnCommands tests that the `ddev npm` and yarn commands behaves and install run in
// expected relative directory.
func TestNpmYarnCommands(t *testing.T) {
	assert := asrt.New(t)

	origDir, _ := os.Getwd()

	site := TestSites[0]

	packageJSON := `
	{
	  "name": "junk",
	  "version": "1.0.0",
	  "description": "",
	  "main": "index.js",
	  "devDependencies": {},
	  "scripts": {
		"test": "echo \"Error: no test specified\" && exit 1"
	  },
	  "author": "",
	  "license": "ISC"
	}
	`

	app, err := ddevapp.NewApp(site.Dir, false)
	require.NoError(t, err)

	t.Cleanup(func() {
		err = app.Stop(true, false)
		assert.NoError(err)
		err = os.Chdir(origDir)
		assert.NoError(err)
		_ = os.RemoveAll(filepath.Join(app.AppRoot, "one"))
		_ = os.RemoveAll(filepath.Join(app.AppRoot, "package.json"))
		_ = os.RemoveAll(filepath.Join(app.AppRoot, "package-lock.json"))
	})

	err = app.Start()
	require.NoError(t, err)

	testDirs := []string{"", "one", "one/two"}
	for _, d := range testDirs {
		workDir := filepath.Join(app.AppRoot, d)
		err = os.MkdirAll(workDir, 0755)
		require.NoError(t, err)
		err = os.Chdir(workDir)
		require.NoError(t, err)
		packageJSONFile := filepath.Join(workDir, "package.json")
		err = os.WriteFile(packageJSONFile, []byte(packageJSON), 0755)
		require.NoError(t, err)
		err = app.MutagenSyncFlush()
		require.NoError(t, err)
		out, err := exec.RunHostCommand(DdevBin, "npm", "install", "--no-audit")
		assert.NoError(err)
		assert.Contains(out, "up to date in", "d='%s', npm install has wrong output; output='%s'", d, out)
		out, err = exec.RunHostCommand(DdevBin, "yarn", "install")
		assert.NoError(err)
		assert.Contains(out, "success Saved lockfile")

		err = os.RemoveAll(packageJSONFile)
		assert.NoError(err)
		err = os.RemoveAll(filepath.Join(workDir, "package-lock.json"))
		assert.NoError(err)
		err = app.MutagenSyncFlush()
		require.NoError(t, err)
	}
}
