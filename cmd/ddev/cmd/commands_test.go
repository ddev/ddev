package cmd

import (
	"fmt"
	"github.com/drud/ddev/pkg/globalconfig"
	"os"
	osexec "os/exec"
	"path/filepath"
	"runtime"
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

	origDir, _ := os.Getwd()

	origHome, err := os.UserHomeDir()
	require.NoError(t, err)

	if runtime.GOOS == "windows" {
		origHome = os.Getenv("USERPROFILE")
	}
	origDebug := os.Getenv("DDEV_DEBUG")

	site := TestSites[0]
	err = os.Chdir(site.Dir)
	require.NoError(t, err)

	app, err := ddevapp.NewApp("", false)
	assert.NoError(err)

	// Must be stopped before changing homedir or mutagen will lose track
	// of sessions which are also tracked in the homedir.
	err = app.Stop(true, false)
	require.NoError(t, err)

	tmpHome := testcommon.CreateTmpDir(t.Name() + "-tempHome")

	// Change the homedir temporarily
	_ = os.Setenv("HOME", tmpHome)
	_ = os.Setenv("USERPROFILE", tmpHome)
	_ = os.Setenv("DDEV_DEBUG", "")

	// Make sure we have the .ddev/bin dir we need
	err = fileutil.CopyDir(filepath.Join(origHome, ".ddev/bin"), filepath.Join(tmpHome, ".ddev/bin"))
	require.NoError(t, err)

	testdataCustomCommandsDir := filepath.Join(origDir, "testdata", t.Name())

	origType := app.Type
	t.Cleanup(func() {
		// Stop the mutagen daemon runnning in the bogus homedir
		ddevapp.StopMutagenDaemon()
		err = os.Chdir(origDir)
		assert.NoError(err)
		runTime()
		app.Type = origType
		err = app.WriteConfig()
		assert.NoError(err)
		_ = os.RemoveAll(tmpHome)
		_ = os.Setenv("HOME", origHome)
		_ = os.Setenv("USERPROFILE", origHome)
		_ = os.Setenv("DDEV_DEBUG", origDebug)
		err = fileutil.PurgeDirectory(filepath.Join(site.Dir, ".ddev", "commands"))
		assert.NoError(err)
		err = fileutil.PurgeDirectory(filepath.Join(site.Dir, ".ddev", ".global_commands"))
		assert.NoError(err)
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
	projectGlobalCommandsCopy := app.GetConfigPath(".global_commands")
	_ = os.RemoveAll(projectGlobalCommandsCopy)
	err = fileutil.CopyDir(filepath.Join(testdataCustomCommandsDir, "global_commands"), tmpHomeGlobalCommandsDir)
	require.NoError(t, err)

	// Must sync our added commands before using them.
	err = app.MutagenSyncFlush()
	assert.NoError(err)

	out, err := exec.RunHostCommand(DdevBin)
	assert.NoError(err)
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
	err = os.RemoveAll(projectGlobalCommandsCopy)
	assert.NoError(err)

	// Now copy a project commands and global commands and make sure they show up and execute properly
	err = fileutil.CopyDir(filepath.Join(testdataCustomCommandsDir, "project_commands"), projectCommandsDir)
	assert.NoError(err)

	// Must sync our added in-container commands before using them.
	err = app.MutagenSyncFlush()
	assert.NoError(err)

	out, err = exec.RunHostCommand(DdevBin)
	assert.NoError(err)
	assert.Contains(out, "testhostcmd project (shell host container command)")
	assert.Contains(out, "testwebcmd project (shell web container command)")
	assert.Contains(out, "testhostglobal global (global shell host container command)")
	assert.Contains(out, "testwebglobal global (global shell web container command)")
	assert.NotContains(out, "testhostcmd global") //the global testhostcmd should have been overridden by the project one
	assert.NotContains(out, "testwebcmd global")  //the global testwebcmd should have been overridden by the project one

	// Have to do app.Start() because commands are copied into containers on start
	err = app.Start()
	require.NoError(t, err)
	for _, c := range []string{"testhostcmd", "testhostglobal", "testwebcmd", "testwebglobal"} {
		out, err = exec.RunHostCommand(DdevBin, c, "hostarg1", "hostarg2", "--hostflag1")
		if err != nil {
			userHome, err := os.UserHomeDir()
			assert.NoError(err)
			globalDdevDir := globalconfig.GetGlobalDdevDir()
			homeEnv := os.Getenv("HOME")
			t.Errorf("userHome=%s, globalDdevDir=%s, homeEnv=%s", userHome, globalDdevDir, homeEnv)
			t.Errorf("Failed to run ddev %s: %v, home=%s output=%s", c, err, userHome, out)
			out, err = exec.RunHostCommand("ls", "-lR", globalDdevDir, "comamnds")
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
	out, err = exec.RunHostCommand(DdevBin, c, "-h")
	assert.NoError(err, "Failed to run ddev %s %s", c, "-h")
	assert.Contains(out, "Examples:\n  ddev testhostcmd\n  ddev testhostcmd -h")

	// Test flags are imported from comments
	c = "testhostcmdflags"
	out, err = exec.RunHostCommand(DdevBin, c, "--test")
	expectedHost, _ := os.Hostname()
	assert.NoError(err, "Failed to run ddev %s %v", c, "--test")
	assert.Contains(out, fmt.Sprintf("%s was executed with args=--test on host %s", c, expectedHost))

	out, err = exec.RunHostCommand(DdevBin, c, "-h")
	assert.NoError(err, "Failed to run ddev %s %v", c, "-h")
	assert.Contains(out, "  -t, --test   Usage of test")

	// Tests with app type PHP
	app.Type = nodeps.AppTypePHP
	err = app.WriteConfig()
	assert.NoError(err)

	// Make sure that all the official ddev-provided custom commands are usable by just checking help
	for _, c := range []string{"launch", "mysql", "php", "xdebug"} {
		_, err = exec.RunHostCommand(DdevBin, c, "-h")
		assert.NoError(err, "Failed to run ddev %s -h", c)
	}

	// The various CMS commands should not be available here
	for _, c := range []string{"artisan", "drush", "magento", "typo3", "typo3cms", "wp"} {
		_, err = exec.RunHostCommand(DdevBin, c, "-h")
		assert.Error(err, "found command %s when it should not have been there (no error) app.Type=%s", c, app.Type)
	}

	// TYPO3 commands should only be available for type typo3
	app.Type = nodeps.AppTypeTYPO3
	_ = app.WriteConfig()

	_, _ = exec.RunHostCommand(DdevBin)
	err = app.MutagenSyncFlush()
	assert.NoError(err)
	for _, c := range []string{"typo3", "typo3cms"} {
		_, err = exec.RunHostCommand(DdevBin, c, "-h")
		assert.NoError(err)
	}

	// Drupal types should only be available for type drupal*
	app.Type = nodeps.AppTypeDrupal9
	_ = app.WriteConfig()
	_, _ = exec.RunHostCommand(DdevBin)
	err = app.MutagenSyncFlush()
	assert.NoError(err)

	for _, c := range []string{"drush"} {
		_, err = exec.RunHostCommand(DdevBin, c, "-h")
		assert.NoError(err)
	}

	// Laravel types should only be available for type laravel
	app.Type = nodeps.AppTypeLaravel
	_ = app.WriteConfig()
	_, _ = exec.RunHostCommand(DdevBin)
	err = app.MutagenSyncFlush()
	assert.NoError(err)
	for _, c := range []string{"artisan"} {
		_, err = exec.RunHostCommand(DdevBin, c, "-h")
		assert.NoError(err)
	}

	// WordPress types should only be available for type drupal*
	app.Type = nodeps.AppTypeWordPress
	_ = app.WriteConfig()
	_, _ = exec.RunHostCommand(DdevBin)
	err = app.MutagenSyncFlush()
	assert.NoError(err)
	for _, c := range []string{"wp"} {
		_, err = exec.RunHostCommand(DdevBin, c, "-h")
		assert.NoError(err, "expected to find command %s for app.Type=%s", c, app.Type)
	}

	// Make sure that the non-command stuff we installed has been copied into projectGlobalCommandsCopy
	for _, f := range []string{".gitattributes", "db/mysqldump.example", "db/README.txt", "host/heidisql", "host/mysqlworkbench.example", "host/phpstorm.example", "host/README.txt", "host/sequelace", "host/sequelpro", "host/tableplus", "web/README.txt"} {
		assert.FileExists(filepath.Join(projectGlobalCommandsCopy, f))
	}
	// Make sure that the non-command stuff we installed is in project commands dir
	for _, f := range []string{".gitattributes", "db/README.txt", "host/README.txt", "host/solrtail.example", "solr/README.txt", "solr/solrtail.example", "web/README.txt"} {
		assert.FileExists(filepath.Join(projectCommandsDir, f))
	}

	// Make sure we haven't accidentally created anything inappropriate in ~/.ddev
	assert.False(fileutil.FileExists(filepath.Join(tmpHome, ".ddev", ".globalcommands")))
	assert.False(fileutil.FileExists(filepath.Join(origHome, ".ddev", ".globalcommands")))

	// Make sure that the old launch, mysql, and xdebug commands aren't in the project directory
	for _, command := range []string{"db/mysql", "host/launch", "web/xdebug"} {
		cmdPath := app.GetConfigPath(filepath.Join("commands", command))
		assert.False(fileutil.FileExists(cmdPath), "file %s exists but it should not", cmdPath)
	}

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
		err = os.RemoveAll(tmpdir)
		assert.NoError(err)
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
	if globalconfig.DdevGlobalConfig.MkcertCARoot == "" {
		cases["-p"] = desc["phpmyadmin_url"].(string)
		cases["-m"] = desc["mailhog_url"].(string)
	}
	for partialCommand, expect := range cases {
		// Try with the base URL, simplest case
		c := DdevBin + `  launch ` + partialCommand + ` | awk '/FULLURL/ {print $2}'`
		out, err := exec.RunHostCommand("bash", "-c", c)
		out = strings.Trim(out, "\r\n")
		assert.NoError(err, `couldn't run "%s"", output=%s`, c, out)
		assert.Contains(out, expect, "output of %s is incorrect with app.RouterHTTPSPort=%s: %s", c, app.RouterHTTPSPort, out)
	}
}

// TestMysqlCommand tests `ddev mysql``
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
	// .ddev/.global_commands which otherwise doesn't get done until ddev start
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

// TestPsqlCommand tests `ddev psql``
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
		err = os.RemoveAll(tmpDir)
		assert.NoError(err)
	})

	err = app.WriteConfig()
	require.NoError(t, err)

	// This populates the project's
	// .ddev/.global_commands which otherwise doesn't get done until ddev start
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
