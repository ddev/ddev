package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/testcommon"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComposerCmdCreateConfigInstall(t *testing.T) {
	assert := asrt.New(t)

	origDir, err := os.Getwd()
	assert.NoError(err)

	for _, composerRoot := range []string{"", "composer-root"} {
		tmpDir := testcommon.CreateTmpDir(t.Name())
		err = os.Chdir(tmpDir)
		assert.NoError(err)

		// Prepare arguments
		arguments := []string{"config", "--project-type", "php"}

		if composerRoot != "" {
			arguments = append(arguments, "--composer-root", composerRoot)
			err = os.Mkdir(composerRoot, 0777)
			assert.NoError(err)
		}

		// Basic config
		_, err = exec.RunHostCommand(DdevBin, arguments...)
		assert.NoError(err)

		// Test trivial command
		out, err := exec.RunHostCommand(DdevBin, "composer")
		assert.NoError(err)
		assert.Contains(out, "Available commands:")

		// Get an app so we can do waits
		app, err := ddevapp.NewApp(tmpDir, true)
		assert.NoError(err)

		t.Cleanup(func() {
			//nolint: errcheck
			err = app.Stop(true, false)
			assert.NoError(err)

			err = os.Chdir(origDir)
			assert.NoError(err)
			_ = os.RemoveAll(tmpDir)
		})

		// Test create-project
		// It appears to be something about Composer itself?

		// ddev composer create-project --prefer-dist --no-interaction --no-dev psr/log:1.1.0
		args := []string{"composer", "create-project", "--prefer-dist", "--no-interaction", "--no-dev", "--no-install", "psr/log:1.1.0"}
		out, err = exec.RunHostCommand(DdevBin, args...)
		assert.NoError(err, "failed to run %v: err=%v, output=\n=====\n%s\n=====\n", args, err, out)
		assert.Contains(out, "Created project in ")
		assert.FileExists(filepath.Join(tmpDir, composerRoot, "composer.json"))

		args = []string{"composer", "config", "--append", "--", "allow-plugins", "true"}
		_, err = exec.RunHostCommand(DdevBin, args...)
		assert.NoError(err)

		args = []string{"composer", "install"}
		_, err = exec.RunHostCommand(DdevBin, args...)
		assert.NoError(err)

		assert.FileExists(filepath.Join(tmpDir, composerRoot, "Psr/Log/LogLevel.php"))
	}
}

func TestComposerCmdCreateRequireRemoveConfigVersion(t *testing.T) {
	assert := asrt.New(t)

	origDir, err := os.Getwd()
	assert.NoError(err)

	for _, composerRoot := range []string{"", "composer-root"} {
		tmpDir := testcommon.CreateTmpDir(t.Name())
		err = os.Chdir(tmpDir)
		assert.NoError(err)

		// Prepare arguments
		arguments := []string{"config", "--project-type", "php"}

		if composerRoot != "" {
			arguments = append(arguments, "--composer-root", composerRoot)
			err = os.Mkdir(composerRoot, 0777)
			assert.NoError(err)
		}

		// Basic config
		_, err = exec.RunHostCommand(DdevBin, arguments...)
		assert.NoError(err)

		// Test trivial command
		out, err := exec.RunHostCommand(DdevBin, "composer")
		assert.NoError(err)
		assert.Contains(out, "Available commands:")

		// Get an app so we can do waits
		app, err := ddevapp.NewApp(tmpDir, true)
		assert.NoError(err)

		t.Cleanup(func() {
			//nolint: errcheck
			err = app.Stop(true, false)
			assert.NoError(err)

			err = os.Chdir(origDir)
			assert.NoError(err)
			_ = os.RemoveAll(tmpDir)
		})

		// ddev composer create-project --prefer-dist --no-dev --no-install psr/log:1.1.0 .
		args := []string{"composer", "create-project", "--prefer-dist", "--no-dev", "--no-install", "psr/log:1.1.0", "."}
		out, err = exec.RunHostCommand(DdevBin, args...)
		assert.NoError(err, "failed to run %v: err=%v, output=\n=====\n%s\n=====\n", args, err, out)
		assert.Contains(out, "Created project in ")
		assert.FileExists(filepath.Join(tmpDir, composerRoot, "composer.json"))
		assert.FileExists(filepath.Join(tmpDir, composerRoot, "Psr/Log/LogLevel.php"))

		// Test a composer require, with passthrough args
		args = []string{"composer", "require", "sebastian/version:5.0.1 as 5.0.0", "--no-plugins", "--ansi"}
		out, err = exec.RunHostCommand(DdevBin, args...)
		assert.NoError(err, "failed to run %v: err=%v, output=\n=====\n%s\n=====\n", args, err, out)
		assert.Contains(out, "Generating autoload files")
		assert.FileExists(filepath.Join(tmpDir, composerRoot, "vendor/sebastian/version/composer.json"))
		// Test a composer remove
		args = []string{"composer", "remove", "sebastian/version"}
		out, err = exec.RunHostCommand(DdevBin, args...)
		assert.NoError(err, "failed to run %v: err=%v, output=\n=====\n%s\n=====\n", args, err, out)
		assert.Contains(out, "Generating autoload files")
		assert.False(fileutil.FileExists(filepath.Join(tmpDir, composerRoot, "vendor/sebastian")))
		// Test a composer config, with passthrough args
		args = []string{"composer", "config", "repositories.packagist", `{"type": "composer", "url": "https://packagist.org"}`}
		out, err = exec.RunHostCommand(DdevBin, args...)
		assert.NoError(err, "failed to run %v: err=%v, output=\n=====\n%s\n=====\n", args, err, out)
		composerJSON, err := fileutil.ReadFileIntoString(filepath.Join(tmpDir, composerRoot, "composer.json"))
		assert.NoError(err, "failed to read %v: err=%v", filepath.Join(tmpDir, composerRoot, "composer.json"), err)
		assert.Contains(composerJSON, "https://packagist.org")
	}
}

func TestComposerAutocomplete(t *testing.T) {
	// Change to the directory for the project to test.
	// We don't really care what the project is, they should
	// all have composer installed in the web container.
	origDir, _ := os.Getwd()
	origDdevDebug := os.Getenv("DDEV_DEBUG")
	_ = os.Unsetenv("DDEV_DEBUG")
	err := os.Chdir(TestSites[0].Dir)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = os.Chdir(origDir)
		_ = os.Setenv("DDEV_DEBUG", origDdevDebug)
	})

	// Make sure the sites exist and are running
	err = addSites()
	require.NoError(t, err)

	// Pressing tab after `composer completion` should result in the completion "bash"
	out, err := exec.RunHostCommand(DdevBin, "__complete", "composer", "completion", "")
	require.NoError(t, err)
	// Completions are terminated with ":4", so just grab the stuff before that
	completions, _, found := strings.Cut(out, ":")
	require.True(t, found)
	require.Equal(t, "bash", strings.TrimSpace(completions))
}
