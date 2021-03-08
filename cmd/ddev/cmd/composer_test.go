package cmd

import (
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/testcommon"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestComposerCmd(t *testing.T) {
	assert := asrt.New(t)

	// This often fails on Windows with NFS
	// It appears to be something about composer itself, or could be stale nfs
	// Might be fixable with `ls` in the directory before taking action
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows because it always seems to fail on windows nfs")
	}

	oldDir, err := os.Getwd()
	assert.NoError(err)

	tmpDir := testcommon.CreateTmpDir(t.Name())
	err = os.Chdir(tmpDir)
	assert.NoError(err)

	// Basic config
	args := []string{"config", "--project-type", "php"}
	_, err = exec.RunCommand(DdevBin, args)
	assert.NoError(err)

	// Test trivial command
	args = []string{"composer"}
	out, err := exec.RunCommand(DdevBin, args)
	assert.NoError(err)
	assert.Contains(out, "Available commands:")

	// Get an app just so we can do waits
	app, err := ddevapp.NewApp(tmpDir, true)
	assert.NoError(err)

	t.Cleanup(func() {
		err = os.Chdir(oldDir)
		assert.NoError(err)
		err = app.Stop(true, false)
		assert.NoError(err)
		err = os.RemoveAll(tmpDir)
		assert.NoError(err)
	})

	// Test create-project
	// These two often fail on Windows with NFS
	// It appears to be something about composer itself?

	// ddev composer create --prefer-dist --no-interaction --no-dev psr/log:1.1.0
	args = []string{"composer", "create", "--prefer-dist", "--no-interaction", "--no-dev", "psr/log:1.1.0"}
	out, err = exec.RunCommand(DdevBin, args)
	assert.NoError(err, "failed to run %v: err=%v, output=\n=====\n%s\n=====\n", args, err, out)
	assert.Contains(out, "Created project in ")
	assert.FileExists(filepath.Join(tmpDir, "Psr/Log/LogLevel.php"))

	err = app.StartAndWait(5)
	assert.NoError(err)
	// ddev composer create --prefer-dist--no-dev --no-install psr/log:1.1.0
	args = []string{"composer", "create", "--prefer-dist", "--no-dev", "--no-install", "psr/log:1.1.0"}
	out, err = exec.RunCommand(DdevBin, args)
	assert.NoError(err, "failed to run %v: err=%v, output=\n=====\n%s\n=====\n", args, err, out)
	assert.Contains(out, "Created project in ")
	assert.FileExists(filepath.Join(tmpDir, "Psr/Log/LogLevel.php"))

	// Test a composer require, with passthrough args
	args = []string{"composer", "require", "sebastian/version", "--no-plugins", "--ansi"}
	out, err = exec.RunCommand(DdevBin, args)
	assert.NoError(err, "failed to run %v: err=%v, output=\n=====\n%s\n=====\n", args, err, out)
	assert.Contains(out, "Generating autoload files")
	assert.FileExists(filepath.Join(tmpDir, "vendor/sebastian/version/composer.json"))
	// Test a composer remove
	args = []string{"composer", "remove", "sebastian/version"}
	out, err = exec.RunCommand(DdevBin, args)
	assert.NoError(err, "failed to run %v: err=%v, output=\n=====\n%s\n=====\n", args, err, out)
	assert.Contains(out, "Generating autoload files")
	assert.False(fileutil.FileExists(filepath.Join(tmpDir, "vendor/sebastian")))

	// Verify that we can target a composer.json in another directory
	cDir := filepath.Join(app.AppRoot, "composerdir")
	err = os.MkdirAll(cDir, 0755)
	assert.NoError(err)
	args = []string{"composer", "-d", "composerdir", "init", "-q", "--name=j/j"}
	_, err = exec.RunCommand(DdevBin, args)
	require.NoError(t, err)
	args = []string{"composer", "-d", "composerdir", "require", "sebastian/version", "--no-plugins", "--ansi"}
	_, err = exec.RunCommand(DdevBin, args)
	require.NoError(t, err)
	rv, err := fileutil.FgrepStringInFile(filepath.Join(cDir, "composer.json"), "sebastian/version")
	assert.NoError(err)
	assert.True(rv)
	assert.True(fileutil.FileExists(filepath.Join(cDir, "vendor/sebastian")))
}
