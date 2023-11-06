package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ddev/ddev/pkg/config/types"
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/testcommon"
	asrt "github.com/stretchr/testify/assert"
)

func TestComposerCmd1(t *testing.T) {
	// 2022-05-24: I've spent lots of time debugging intermittent `composer create` failures when NFS
	// is enabled, both on macOS and Windows. As far as I can tell, it only happens in this test, I've
	// never recreated manually. I do see https://github.com/composer/composer/issues/9627 which seemed
	// to deal with similar issues in vagrant context, and has a hack now embedded into Composer.
	if nodeps.PerformanceModeDefault == types.PerformanceModeNFS {
		t.Skip("Composer has strange behavior in NFS context, so skipping")
	}
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
			err = os.RemoveAll(tmpDir)
			assert.NoError(err)
		})

		// Test create-project
		// These two often fail on Windows with NFS, also Colima
		// It appears to be something about Composer itself?

		// ddev composer create --prefer-dist --no-interaction --no-dev psr/log:1.1.0
		args := []string{"composer", "create", "--prefer-dist", "--no-interaction", "--no-dev", "--no-install", "psr/log:1.1.0"}
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

func TestComposerCmd2(t *testing.T) {
	// 2022-05-24: I've spent lots of time debugging intermittent `composer create` failures when NFS
	// is enabled, both on macOS and Windows. As far as I can tell, it only happens in this test, I've
	// never recreated manually. I do see https://github.com/composer/composer/issues/9627 which seemed
	// to deal with similar issues in vagrant context, and has a hack now embedded into Composer.
	if nodeps.PerformanceModeDefault == types.PerformanceModeNFS {
		t.Skip("Composer has strange behavior in NFS context, so skipping")
	}
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
			err = os.RemoveAll(tmpDir)
			assert.NoError(err)
		})

		// Test create-project
		// These two often fail on Windows with NFS, also Colima
		// It appears to be something about Composer itself?

		// ddev composer create --prefer-dist --no-dev --no-install psr/log:1.1.0
		args := []string{"composer", "create", "--prefer-dist", "--no-dev", "--no-install", "psr/log:1.1.0"}
		out, err = exec.RunHostCommand(DdevBin, args...)
		assert.NoError(err, "failed to run %v: err=%v, output=\n=====\n%s\n=====\n", args, err, out)
		assert.Contains(out, "Created project in ")
		assert.FileExists(filepath.Join(tmpDir, composerRoot, "Psr/Log/LogLevel.php"))

		// Test a composer require, with passthrough args
		args = []string{"composer", "require", "sebastian/version", "--no-plugins", "--ansi"}
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
	}
}
