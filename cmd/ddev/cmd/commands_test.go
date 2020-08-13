package cmd

import (
	"fmt"
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/nodeps"
	"github.com/drud/ddev/pkg/testcommon"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	osexec "os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestCustomCommands does basic checks to make sure custom commands work OK.
func TestCustomCommands(t *testing.T) {
	assert := asrt.New(t)

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
	app, _ := ddevapp.NewApp(TestSites[0].Dir, false, "")
	t.Cleanup(func() {
		_ = os.RemoveAll(tmpHome)
		_ = os.Setenv("HOME", origHome)
		_ = os.Setenv("DDEV_DEBUG", origDebug)
		_ = fileutil.PurgeDirectory(filepath.Join(site.Dir, ".ddev", "commands"))
		_ = fileutil.PurgeDirectory(filepath.Join(site.Dir, ".ddev", ".global_commands"))
		switchDir()
	})

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
	assert.FileExists(filepath.Join(projectCommandsDir, "host", "mysqlworkbench.example"))
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
	assert.NotContains(out, "testwebcmd global") //the global testwebcmd should have been overridden by the projct one
	assert.Contains(out, "testhostglobal")

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

	// Make sure that all the official ddev-provided custom commands are usable by just checking help
	for _, c := range []string{"launch", "live", "mysql", "xdebug"} {
		_, err = exec.RunCommand(DdevBin, []string{c, "-h"})
		assert.NoError(err, "Failed to run ddev %s -h", c)
	}

	// The TYPO3 and Drupal commands should not be available here (currently WordPress)
	for _, c := range []string{"typo3", "typo3cms", "drush", "artisan", "magento"} {
		_, err = exec.RunCommand(DdevBin, []string{c, "-h"})
		assert.Error(err)
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

	// Make sure that the non-command stuff we installed is there
	for _, f := range []string{"db/mysqldump.example", "db/README.txt", "web/README.txt", "host/README.txt", "host/phpstorm.example"} {
		assert.FileExists(filepath.Join(projectCommandsDir, f))
		assert.FileExists(filepath.Join(globalCommandsDir, f))
	}

	// Make sure we haven't accidentally created anything inappropriate in ~/.ddev
	assert.False(fileutil.FileExists(filepath.Join(tmpHome, ".ddev", ".globalcommands")))
	assert.False(fileutil.FileExists(filepath.Join(origHome, ".ddev", ".globalcommands")))
}

// TestLaunchCommand tests that the launch command behaves all the ways it should behave
func TestLaunchCommand(t *testing.T) {
	assert := asrt.New(t)

	// Create a temporary directory and switch to it.
	tmpdir := testcommon.CreateTmpDir(t.Name())
	defer testcommon.CleanupDir(tmpdir)
	defer testcommon.Chdir(tmpdir)()

	_ = os.Setenv("DDEV_DEBUG", "true")
	app, err := ddevapp.NewApp(tmpdir, false, "")
	require.NoError(t, err)
	err = app.WriteConfig()
	require.NoError(t, err)
	err = app.Start()
	require.NoError(t, err)
	defer func() {
		_ = app.Stop(true, false)
	}()

	// This only tests the https port changes, but that might be enough
	for _, routerPort := range []string{nodeps.DdevDefaultRouterHTTPSPort, "8443"} {
		app.RouterHTTPSPort = routerPort
		_ = app.WriteConfig()
		err = app.Start()
		assert.NoError(err)

		desc, _ := app.Describe(false)
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
			assert.Equal(expect, out, "ouptput of %s is incorrect with app.RouterHTTPSPort=%s: %s", c, app.RouterHTTPSPort, out)
		}
	}
}

// TestMysqlCommand tests `ddev mysql``
func TestMysqlCommand(t *testing.T) {
	assert := asrt.New(t)

	// Create a temporary directory and switch to it.
	tmpdir := testcommon.CreateTmpDir(t.Name())
	defer testcommon.CleanupDir(tmpdir)
	defer testcommon.Chdir(tmpdir)()

	app, err := ddevapp.NewApp(tmpdir, false, "")
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
