package cmd

import (
	"github.com/drud/ddev/pkg/util"
	"os"
	"testing"

	"github.com/drud/ddev/pkg/testcommon"

	"github.com/drud/ddev/pkg/exec"
	asrt "github.com/stretchr/testify/assert"
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

	// Test create-project
	// ddev composer create cweagans/composer-patches --prefer-dist --no-interaction
	args = []string{"composer", "create", "--prefer-dist", "--no-interaction", "cweagans/composer-patches", "1.6.5"}
	out, err = exec.RunCommand(DdevBin, args)
	assert.NoError(err, "failed to run %v: err=%v, output=\n=====\n%s\n=====\n", args, out)
	assert.Contains(out, "Created project in ")

	// Test a composer require
	args = []string{"composer", "require", "twitter/bootstrap"}
	out, err = exec.RunCommand(DdevBin, args)
	assert.NoError(err, "failed to run %v: err=%v, output=\n=====\n%s\n=====\n", args, out)
	assert.Contains(out, "Generating autoload files")

	// Test a command with flags
	args = []string{"composer", "depends", "twitter/bootstrap", "--tree"}
	out, err = exec.RunCommand(DdevBin, args)
	assert.NoError(err, "failed to run %v: err=%v, output=\n=====\n%s\n=====\n", args, out)
	assert.Contains(out, "--twitter/bootstrap")

	// Test a composer remove
	if util.IsDockerToolbox() {
		// On docker toolbox, git objects are read-only, causing the composer remove to fail.
		_, err = exec.RunCommand(DdevBin, []string{"exec", "bash", "-c", "chmod -R u+w /var/www/html/"})
		assert.NoError(err)
	}
	args = []string{"composer", "remove", "twitter/bootstrap"}
	out, err = exec.RunCommand(DdevBin, args)
	assert.NoError(err, "failed to run %v: err=%v, output=\n=====\n%s\n=====\n", args, out)
	assert.Contains(out, "Generating autoload files")
}
