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
		args = []string{"composer", "require", "sebastian/version:5.0.1 as 5.0.0", "--no-plugins", "--ansi", "--no-security-blocking"}
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

func TestComposerWithUseWorkingDir(t *testing.T) {
	origDir, err := os.Getwd()
	require.NoError(t, err)

	for _, composerRoot := range []string{"", "application/composer-root"} {
		tmpDir := testcommon.CreateTmpDir(t.Name())
		err = os.Chdir(tmpDir)
		require.NoError(t, err)

		beforeComposerRootDir := ""

		// Prepare arguments
		arguments := []string{"config", "--project-type", "php"}

		if composerRoot != "" {
			arguments = append(arguments, "--composer-root", composerRoot)
			err = os.MkdirAll(filepath.Join(tmpDir, composerRoot), 0777)
			require.NoError(t, err)

			// Create a different composer.json file a level above the root directory.
			// Because the "composer_root" has been declared, then running `ddev composer`
			// against this working directory should never work because it's outside the
			// scope of the configured root.
			beforeComposerRootDir = filepath.Join(tmpDir, "before-composer-root")
			err = os.MkdirAll(beforeComposerRootDir, 0777)
			require.NoError(t, err)
			subComposer := `{
				"name": "ddev-test/before-root",
				"type": "project",
				"license": "The Unlicense"
			}`
			err = os.WriteFile(filepath.Join(beforeComposerRootDir, "composer.json"), []byte(subComposer), 0644)
			require.NoError(t, err)
		}

		// Basic config for a php project
		_, err = exec.RunHostCommand(DdevBin, arguments...)
		require.NoError(t, err)

		// Create the default index.php file
		err = os.WriteFile(filepath.Join(tmpDir, composerRoot, "index.php"), []byte("<?php\necho 'Hello world';"), 0644)
		require.NoError(t, err)

		// Create composer.json at root with license MIT
		rootComposer := `{
			"name": "ddev-test/root",
			"type": "project",
			"license": "MIT"
		}`
		err = os.WriteFile(filepath.Join(tmpDir, composerRoot, "composer.json"), []byte(rootComposer), 0644)
		require.NoError(t, err)

		// Create a subdirectory with its own composer.json and a different license
		subdir := filepath.Join(tmpDir, composerRoot, "sub")
		err = os.MkdirAll(subdir, 0777)
		require.NoError(t, err)
		subComposer := `{
			"name": "ddev-test/sub",
			"type": "project",
			"license": "BSD-3-Clause"
		}`
		err = os.WriteFile(filepath.Join(subdir, "composer.json"), []byte(subComposer), 0644)
		require.NoError(t, err)

		// Get an app so we can do waits
		app, err := ddevapp.NewApp(tmpDir, true)
		require.NoError(t, err)

		t.Cleanup(func() {
			//nolint: errcheck
			err = app.Stop(true, false)
			require.NoError(t, err)

			err = os.Chdir(origDir)
			require.NoError(t, err)
			_ = os.RemoveAll(tmpDir)
		})

		// Change into subdirectory and run composer license.
		// The root directory should still be used before `composer_use_working_dir` is set.
		err = os.Chdir(subdir)
		require.NoError(t, err)

		out, err := exec.RunHostCommand(DdevBin, "composer", "license")
		require.NoError(t, err, "composer license before enabling use-working-dir failed: %v, out=\n%s", err, out)
		// Expect root license (MIT)
		require.Contains(t, out, "MIT", "expected root license (MIT) when not using working dir; output=\n%s", out)
		require.NotContains(t, out, "BSD-3-Clause", "unexpected subdir license before enabling working dir; output=\n%s", out)

		// Enable Composer working dir
		_, err = exec.RunHostCommand(DdevBin, "config", "--composer-use-working-dir")
		require.NoError(t, err)

		// Rerun composer license; now should pick up the subdirectory's composer.json
		out, err = exec.RunHostCommand(DdevBin, "composer", "license")
		require.NoError(t, err, "composer license after enabling use-working-dir failed: %v, out=\n%s", err, out)
		require.Contains(t, out, "BSD-3-Clause", "expected subdir license (BSD-3-Clause) when using working dir; output=\n%s", out)

		if beforeComposerRootDir != "" {
			err = os.Chdir(beforeComposerRootDir)
			require.NoError(t, err)
			// Running composer on any directory above the Composer root should default to using
			// the root composer.json.
			out, err = exec.RunHostCommand(DdevBin, "composer", "license")
			require.NoError(t, err, "composer license after enabling use-working-dir failed: %v, out=\n%s", err, out)
			require.Contains(t, out, "MIT", "expected root license (MIT) when using working dir before the composer root; output=\n%s", out)
			require.NotContains(t, out, "The Unlicense", "unexpected before-composer-root license; output=\n%s", out)
		}
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
