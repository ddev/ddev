package cmd

import (
	"fmt"
	"os"
	osexec "os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/nodeps"
	"github.com/drud/ddev/pkg/testcommon"
	"github.com/drud/ddev/pkg/util"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCustomCommands does basic checks to make sure custom commands work OK.
func TestCustomCommands(t *testing.T) {
	assert := asrt.New(t)
	runTime := util.TimeTrack(time.Now(), t.Name())

	tmpHome := testcommon.CreateTmpDir(t.Name() + "tempHome")
	origHome := os.Getenv("HOME")
	origDebug := os.Getenv("DDEV_DEBUG")
	// Change the homedir temporarily
	err := os.Setenv("HOME", tmpHome)
	require.NoError(t, err)
	_ = os.Setenv("DDEV_DEBUG", "")

	pwd, _ := os.Getwd()
	testCustomCommandsDir := filepath.Join(pwd, "testdata", t.Name())

	site := TestSites[0]
	switchDir := TestSites[0].Chdir()
	app, _ := ddevapp.NewApp(TestSites[0].Dir, false)
	origType := app.Type
	t.Cleanup(func() {
		runTime()
		app.Type = origType
		_ = app.WriteConfig()
		_ = os.RemoveAll(tmpHome)
		_ = os.Setenv("HOME", origHome)
		_ = os.Setenv("DDEV_DEBUG", origDebug)
		_ = fileutil.PurgeDirectory(filepath.Join(site.Dir, ".ddev", "commands"))
		_ = fileutil.PurgeDirectory(filepath.Join(site.Dir, ".ddev", ".global_commands"))
		switchDir()
	})
	err = app.Start()
	require.NoError(t, err)

	// We can't use the standard getGlobalDDevDir here because *our* global hasn't changed.
	// It's changed via $HOME for the ddev subprocess
	err = os.MkdirAll(filepath.Join(tmpHome, ".ddev"), 0755)
	assert.NoError(err)
	tmpHomeGlobalCommandsDir := filepath.Join(tmpHome, ".ddev", "commands")
	err = os.RemoveAll(tmpHomeGlobalCommandsDir)
	assert.NoError(err)

	projectCommandsDir := app.GetConfigPath("commands")
	globalCommandsDir := app.GetConfigPath(".global_commands")
	_ = os.RemoveAll(globalCommandsDir)
	err = fileutil.CopyDir(filepath.Join(testCustomCommandsDir, "global_commands"), tmpHomeGlobalCommandsDir)
	require.NoError(t, err)

	assert.FileExists(filepath.Join(projectCommandsDir, "db", "mysql"))
	out, err := exec.RunCommand(DdevBin, []string{})
	assert.NoError(err)
	assert.Contains(out, "mysql client in db container")

	// Test the `ddev mysql` command with stdin
	inputFile := filepath.Join(testCustomCommandsDir, "select99.sql")
	f, err := os.Open(inputFile)
	require.NoError(t, err)
	// nolint: errcheck
	defer f.Close()
	command := osexec.Command(DdevBin, "mysql")
	command.Stdin = f
	byteOut, err := command.CombinedOutput()
	require.NoError(t, err, "Failed ddev mysql; output=%v", string(byteOut))
	assert.Contains(string(byteOut), "99\n99\n")

	_ = os.RemoveAll(projectCommandsDir)
	_ = os.RemoveAll(globalCommandsDir)

	// Now copy a project commands and global commands and make sure they show up and execute properly
	err = fileutil.CopyDir(filepath.Join(testCustomCommandsDir, "project_commands"), projectCommandsDir)
	assert.NoError(err)

	out, err = exec.RunCommand(DdevBin, []string{})
	assert.NoError(err)
	assert.Contains(out, "testhostcmd project (shell host container command)")
	assert.Contains(out, "testwebcmd project (shell web container command)")
	assert.Contains(out, "testhostglobal global (global shell host container command)")
	assert.Contains(out, "testwebglobal global (global shell web container command)")
	assert.NotContains(out, "testhostcmd global") //the global testhostcmd should have been overridden by the projct one
	assert.NotContains(out, "testwebcmd global")  //the global testwebcmd should have been overridden by the projct one

	for _, c := range []string{"testhostcmd", "testhostglobal", "testwebcmd", "testwebglobal"} {
		args := []string{c, "hostarg1", "hostarg2", "--hostflag1"}
		out, err = exec.RunCommand(DdevBin, args)
		assert.NoError(err, "Failed to run ddev %s %v", c, args)
		expectedHost, _ := os.Hostname()
		if !strings.Contains(c, "host") {
			expectedHost = site.Name + "-web"
		}
		assert.Contains(out, fmt.Sprintf("%s was executed with args=hostarg1 hostarg2 --hostflag1 on host %s", c, expectedHost))
	}

	// Test line breaks in examples
	c := "testhostcmd"
	args := []string{c, "-h"}
	out, err = exec.RunCommand(DdevBin, args)
	assert.NoError(err, "Failed to run ddev %s %v", c, args)
	assert.Contains(out, "Examples:\n  ddev testhostcmd\n  ddev testhostcmd -h")

	// Test flags are imported from comments
	c = "testhostcmdflags"
	args = []string{c, "--test"}
	out, err = exec.RunCommand(DdevBin, args)
	expectedHost, _ := os.Hostname()
	assert.NoError(err, "Failed to run ddev %s %v", c, args)
	assert.Contains(out, fmt.Sprintf("%s was executed with args=--test on host %s", c, expectedHost))

	args = []string{c, "-h"}
	out, err = exec.RunCommand(DdevBin, args)
	assert.NoError(err, "Failed to run ddev %s %v", c, args)
	assert.Contains(out, "  -t, --test   Usage of test")

	// Tests with app type PHP
	app.Type = nodeps.AppTypePHP
	err = app.WriteConfig()
	assert.NoError(err)

	// Make sure that all the official ddev-provided custom commands are usable by just checking help
	for _, c := range []string{"launch", "mysql", "xdebug"} {
		_, err = exec.RunCommand(DdevBin, []string{c, "-h"})
		assert.NoError(err, "Failed to run ddev %s -h", c)
	}

	// The various CMS commands should not be available here
	for _, c := range []string{"artisan", "drush", "magento", "typo3", "typo3cms", "wp"} {
		_, err = exec.RunCommand(DdevBin, []string{c, "-h"})
		assert.Error(err, "found command %s when it should not have been there (no error) app.Type=%s", c, app.Type)
	}

	// TYPO3 commands should only be available for type typo3
	app.Type = nodeps.AppTypeTYPO3
	_ = app.WriteConfig()
	_, _ = exec.RunCommand(DdevBin, nil)
	for _, c := range []string{"typo3", "typo3cms"} {
		_, err = exec.RunCommand(DdevBin, []string{c, "-h"})
		assert.NoError(err)
	}

	// Drupal types should only be available for type drupal*
	app.Type = nodeps.AppTypeDrupal9
	_ = app.WriteConfig()
	_, _ = exec.RunCommand(DdevBin, nil)
	for _, c := range []string{"drush"} {
		_, err = exec.RunCommand(DdevBin, []string{c, "-h"})
		assert.NoError(err)
	}

	// Laravel types should only be available for type laravel
	app.Type = nodeps.AppTypeLaravel
	_ = app.WriteConfig()
	_, _ = exec.RunCommand(DdevBin, nil)
	for _, c := range []string{"artisan"} {
		_, err = exec.RunCommand(DdevBin, []string{c, "-h"})
		assert.NoError(err)
	}

	// Wordpress types should only be available for type drupal*
	app.Type = nodeps.AppTypeWordPress
	_ = app.WriteConfig()
	_, _ = exec.RunCommand(DdevBin, nil)
	for _, c := range []string{"wp"} {
		_, err = exec.RunCommand(DdevBin, []string{c, "-h"})
		assert.NoError(err, "expected to find command %s for app.Type=%s", c, app.Type)
	}

	// Make sure that the non-command stuff we installed is in globalCommandsDir
	for _, f := range []string{"db/mysqldump.example", "db/README.txt", "host/heidisql", "host/mysqlworkbench.example", "host/phpstorm.example", "host/README.txt", "host/sequelace", "host/sequelpro", "host/tableplus", "web/README.txt"} {
		assert.FileExists(filepath.Join(globalCommandsDir, f))
	}
	// Make sure that the non-command stuff we installed is in project commands dir
	for _, f := range []string{"db/mysql", "db/README.txt", "host/launch", "host/README.txt", "host/solrtail.example", "solr/README.txt", "solr/solrtail.example", "web/README.txt", "web/xdebug"} {
		assert.FileExists(filepath.Join(projectCommandsDir, f))
	}

	// Make sure we haven't accidentally created anything inappropriate in ~/.ddev
	assert.False(fileutil.FileExists(filepath.Join(tmpHome, ".ddev", ".globalcommands")))
	assert.False(fileutil.FileExists(filepath.Join(origHome, ".ddev", ".globalcommands")))
}

// TestLaunchCommand tests that the launch command behaves all the ways it should behave
func TestLaunchCommand(t *testing.T) {
	assert := asrt.New(t)

	pwd, _ := os.Getwd()
	// Create a temporary directory and switch to it.
	tmpdir := testcommon.CreateTmpDir(t.Name())
	err := os.Chdir(tmpdir)
	assert.NoError(err)

	_ = os.Setenv("DDEV_DEBUG", "true")
	app, err := ddevapp.NewApp(tmpdir, false)
	require.NoError(t, err)
	err = app.WriteConfig()
	require.NoError(t, err)
	t.Cleanup(func() {
		err = app.Stop(true, false)
		assert.NoError(err)
		err = os.Chdir(pwd)
		assert.NoError(err)
		_ = os.RemoveAll(tmpdir)
	})

	// This only tests the https port changes, but that might be enough
	app.RouterHTTPSPort = "8443"
	_ = app.WriteConfig()
	err = app.Start()
	require.NoError(t, err)

	desc, err := app.Describe(false)
	require.NoError(t, err)
	cases := map[string]string{
		"":   app.GetPrimaryURL(),
		"-p": desc["phpmyadmin_https_url"].(string),
		"-m": desc["mailhog_https_url"].(string),
	}
	for partialCommand, expect := range cases {
		// Try with the base URL, simplest case
		c := DdevBin + `  launch ` + partialCommand + ` | awk '/FULLURL/ {print $2}'`
		out, err := exec.RunCommand("bash", []string{"-c", c})
		out = strings.Trim(out, "\n")
		assert.NoError(err, `couldn't run "%s"", output=%s`, c, out)
		assert.Contains(out, expect, "output of %s is incorrect with app.RouterHTTPSPort=%s: %s", c, app.RouterHTTPSPort, out)
	}
}

// TestMysqlCommand tests `ddev mysql``
func TestMysqlCommand(t *testing.T) {
	assert := asrt.New(t)

	// Create a temporary directory and switch to it.
	tmpdir := testcommon.CreateTmpDir(t.Name())
	defer testcommon.CleanupDir(tmpdir)
	defer testcommon.Chdir(tmpdir)()

	app, err := ddevapp.NewApp(tmpdir, false)
	require.NoError(t, err)
	err = app.WriteConfig()
	require.NoError(t, err)
	err = app.Start()
	require.NoError(t, err)
	defer func() {
		_ = app.Stop(true, false)
	}()

	// Test ddev mysql -uroot -proot mysql
	command := osexec.Command("bash", "-c", "echo 'SHOW TABLES;' | "+DdevBin+" mysql --user=root --password=root --database=mysql")
	byteOut, err := command.CombinedOutput()
	assert.NoError(err, "byteOut=%v", string(byteOut))
	assert.Contains(string(byteOut), `Tables_in_mysql
column_stats
columns_priv`)

}
