package cmd

import (
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/testcommon"
	"github.com/drud/ddev/pkg/util"
	asrt "github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
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

	// Get an app just so we can do waits and check webcacheenabled etc.
	app, err := ddevapp.NewApp(tmpDir, "")
	assert.NoError(err)

	// Test create-project
	// ddev composer create --prefer-dist --no-interaction --no-dev psr/log 1.1.0
	args = []string{"composer", "create", "--prefer-dist", "--no-interaction", "--no-dev", "psr/log", "1.1.0"}
	out, err = exec.RunCommand(DdevBin, args)
	assert.NoError(err, "failed to run %v: err=%v, output=\n=====\n%s\n=====\n", args, out)
	assert.Contains(out, "Created project in ")
	ddevapp.WaitForSync(app, 2)
	assert.FileExists(filepath.Join(tmpDir, "Psr/Log/LogLevel.php"))

	// Test a composer require, with passthrough args
	args = []string{"composer", "require", "sebastian/version", "--no-plugins", "--ansi"}
	out, err = exec.RunCommand(DdevBin, args)
	assert.NoError(err, "failed to run %v: err=%v, output=\n=====\n%s\n=====\n", args, out)
	assert.Contains(out, "Generating autoload files")
	ddevapp.WaitForSync(app, 2)
	assert.FileExists(filepath.Join(tmpDir, "vendor/sebastian/version/composer.json"))

	// Test a composer remove
	if util.IsDockerToolbox() {
		// On docker toolbox, git objects are read-only, causing the composer remove to fail.
		_, err = exec.RunCommand(DdevBin, []string{"exec", "bash", "-c", "chmod -R u+w /var/www/html/"})
		assert.NoError(err)
	}
	args = []string{"composer", "remove", "sebastian/version"}
	out, err = exec.RunCommand(DdevBin, args)
	assert.NoError(err, "failed to run %v: err=%v, output=\n=====\n%s\n=====\n", args, out)
	assert.Contains(out, "Generating autoload files")
	ddevapp.WaitForSync(app, 2)
	assert.False(fileutil.FileExists(filepath.Join(tmpDir, "vendor/sebastian")))
}
