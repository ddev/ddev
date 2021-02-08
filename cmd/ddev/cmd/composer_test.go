package cmd

import (
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/testcommon"
	asrt "github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestComposerCmd(t *testing.T) {
	assert := asrt.New(t)

	oldDir, err := os.Getwd()
	assert.NoError(err)
	// nolint: errcheck
	defer os.Chdir(oldDir)

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
	//nolint: errcheck
	defer app.Stop(true, false)

	// Test create-project
	// These two often fail on Windows with NFS
	// It appears to be something about composer itself?

	if !(runtime.GOOS == "windows" && (app.NFSMountEnabled || app.NFSMountEnabledGlobal)) {
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
	}

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
}
