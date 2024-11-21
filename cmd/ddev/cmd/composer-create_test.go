package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/composer"
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/stretchr/testify/require"
)

func TestComposerCreateCmd(t *testing.T) {
	composerVersionForThisTest := nodeps.ComposerDefault
	//composerVersionForThisTest := "2.8.0"

	origDir, err := os.Getwd()
	require.NoError(t, err)

	validAppTypes := ddevapp.GetValidAppTypes()
	if os.Getenv("GOTEST_SHORT") != "" {
		validAppTypes = []string{nodeps.AppTypePHP, nodeps.AppTypeDrupal11}
	}

	for _, docRoot := range []string{"", "doc-root"} {
		for _, projectType := range validAppTypes {
			if projectType == nodeps.AppTypeDrupal6 {
				t.Logf("== SKIP TestComposerCreateCmd for project of type '%s' with docroot '%s'\n", projectType, docRoot)
				t.Logf("== SKIP drupal6 projects uses a very old php version and composer create is very unlikely to be used")
				continue
			}
			t.Logf("== BEGIN TestComposerCreateCmd for project of type '%s' with docroot  '%s'\n", projectType, docRoot)
			tmpDir := testcommon.CreateTmpDir(t.Name() + projectType)
			err = os.Chdir(tmpDir)
			require.NoError(t, err)

			// Prepare arguments
			arguments := []string{"config", "--project-type", projectType, "--composer-version", composerVersionForThisTest}

			composerDirOnHost := tmpDir
			if docRoot != "" {
				arguments = append(arguments, "--docroot", docRoot)
				// For Drupal, we arbitrarily place the composer root to be the docroot
				// Normally for Drupal the docroot would be web, and the composer root would be the
				// project root (default). But here we're making sure we can use the docroot
				// as the composer_root. Acquia sites often do this...
				if projectType == nodeps.AppTypeDrupal11 || projectType == nodeps.AppTypeDrupal10 || projectType == nodeps.AppTypeDrupal9 || projectType == nodeps.AppTypeDrupal8 {
					arguments = append(arguments, "--composer-root", docRoot)
					composerDirOnHost = filepath.Join(tmpDir, docRoot)
				}
			}

			// Basic config
			_, err = exec.RunHostCommand(DdevBin, arguments...)
			require.NoError(t, err)

			// Test trivial command
			out, err := exec.RunHostCommand(DdevBin, "composer", "--version")
			require.NoError(t, err)
			require.Contains(t, out, "Composer version")

			// Get an app so we can do waits
			app, err := ddevapp.NewApp(tmpDir, true)
			require.NoError(t, err)

			t.Cleanup(func() {
				err = app.Stop(true, false)
				require.NoError(t, err)
				err = os.Chdir(origDir)
				require.NoError(t, err)
				_ = os.RemoveAll(tmpDir)
			})

			err = app.Start()
			require.NoError(t, err)

			out, err = exec.RunHostCommand(DdevBin, "composer", "--version")
			require.NoError(t, err)
			require.Contains(t, out, fmt.Sprintf("Composer version %s", composerVersionForThisTest))

			composerCommandTypeCheck := ""
			args := []string{}

			// These are different conditions to test different composer flag combinations
			// Conditions for docRoot and projectType are not important here, they are only needed to make the test act different each time
			if docRoot == "" {
				composerCommandTypeCheck = "installation with --no-plugins --no-scripts"
				if projectType == nodeps.AppTypePHP {
					composerCommandTypeCheck = "installation with -vvv --fake-flag"
				}
			} else {
				composerCommandTypeCheck = "installation with --no-install --prefer-install auto"
				if projectType == nodeps.AppTypePHP {
					composerCommandTypeCheck = "installation with --no-dev --prefer-install=auto"
				}
			}

			t.Logf("Attempting composerCommandTypeCheck='%s' with docroot='%s' projectType=%s", composerCommandTypeCheck, docRoot, projectType)
			// ddev composer create --no-plugins --no-scripts ddev/ddev-test-composer-create
			if composerCommandTypeCheck == "installation with --no-plugins --no-scripts" {
				args = []string{"composer", "create", "--no-plugins", "--no-scripts", "ddev/ddev-test-composer-create"}
			}

			// ddev composer create -vvv --fake-flag ddev/ddev-test-composer-create
			if composerCommandTypeCheck == "installation with -vvv --fake-flag" {
				args = []string{"composer", "create", "-vvv", "--fake-flag", "ddev/ddev-test-composer-create"}
			}

			// ddev composer create --no-install --prefer-install auto ddev/ddev-test-composer-create
			if composerCommandTypeCheck == "installation with --no-install --prefer-install auto" {
				args = []string{"composer", "create", "--no-install", "--prefer-install", "auto", "ddev/ddev-test-composer-create"}
			}

			// ddev composer create --no-dev --prefer-install=auto ddev/ddev-test-composer-create
			if composerCommandTypeCheck == "installation with --no-dev --prefer-install=auto" {
				args = []string{"composer", "create", "--no-dev", "--prefer-install=auto", "ddev/ddev-test-composer-create"}
			}

			// If a file exists in the composer root then composer create should fail
			file, err := os.Create(filepath.Join(composerDirOnHost, "touch1.txt"))
			out, err = exec.RunHostCommand(DdevBin, args...)
			require.Error(t, err)
			require.Contains(t, out, "touch1.txt")
			_ = file.Close()
			_ = os.Remove(filepath.Join(composerDirOnHost, "touch1.txt"))

			// Test success
			out, err = exec.RunHostCommand(DdevBin, args...)
			require.NoError(t, err, "['%s'] failed to run %v: err=%v, output=\n=====\n%s\n=====\n", composerCommandTypeCheck, args, err, out)
			require.Contains(t, out, "Created project in ")
			require.FileExists(t, filepath.Join(composerDirOnHost, "composer.json"))

			// ddev composer create --no-plugins --no-scripts ddev/ddev-test-composer-create
			if composerCommandTypeCheck == "installation with --no-plugins --no-scripts" {
				// Check what was executed or not
				require.Contains(t, out, "Executing Composer command: [composer create-project --no-plugins --no-scripts ddev/ddev-test-composer-create --no-install")
				require.NotContains(t, out, "Executing Composer command: [composer run-script post-root-package-install")
				require.Contains(t, out, "Executing Composer command: [composer install --no-plugins --no-scripts]")
				require.NotContains(t, out, "Executing Composer command: [composer run-script post-create-project-cmd")
				// Check the actual result of executing composer scripts
				require.NoFileExists(t, filepath.Join(composerDirOnHost, "created-by-post-root-package-install"))
				require.NoFileExists(t, filepath.Join(composerDirOnHost, "created-by-post-create-project-cmd"))
				// Check vendor directory
				require.FileExists(t, filepath.Join(composerDirOnHost, "vendor", "autoload.php"))
				require.FileExists(t, filepath.Join(composerDirOnHost, "vendor", "ddev", "ddev-test-composer-require", "composer.json"))
				require.FileExists(t, filepath.Join(composerDirOnHost, "vendor", "ddev", "ddev-test-composer-require-dev", "composer.json"))
			}

			// ddev composer create -vvv --fake-flag ddev/ddev-test-composer-create
			if composerCommandTypeCheck == "installation with -vvv --fake-flag" {
				// Check what was executed or not
				require.Contains(t, out, "Executing Composer command: [composer create-project -vvv ddev/ddev-test-composer-create --no-plugins --no-scripts --no-install")
				require.Contains(t, out, "Executing Composer command: [composer run-script post-root-package-install -vvv]")
				require.Contains(t, out, "Executing Composer command: [composer install -vvv]")
				require.Contains(t, out, "Executing Composer command: [composer run-script post-create-project-cmd -vvv]")
				require.NotContains(t, out, "--fake-flag")
				// Check the actual result of executing composer scripts
				require.FileExists(t, filepath.Join(composerDirOnHost, "created-by-post-root-package-install"))
				require.FileExists(t, filepath.Join(composerDirOnHost, "created-by-post-create-project-cmd"))
				// Check vendor directory
				require.FileExists(t, filepath.Join(composerDirOnHost, "vendor", "autoload.php"))
				require.FileExists(t, filepath.Join(composerDirOnHost, "vendor", "ddev", "ddev-test-composer-require", "composer.json"))
				require.FileExists(t, filepath.Join(composerDirOnHost, "vendor", "ddev", "ddev-test-composer-require-dev", "composer.json"))
			}

			// ddev composer create --no-install --prefer-install auto ddev/ddev-test-composer-create
			if composerCommandTypeCheck == "installation with --no-install --prefer-install auto" {
				// Check what was executed or not
				require.Contains(t, out, "Executing Composer command: [composer create-project --no-install --prefer-install auto ddev/ddev-test-composer-create --no-plugins --no-scripts")
				require.Contains(t, out, "Executing Composer command: [composer run-script post-root-package-install]")
				require.NotContains(t, out, "Executing Composer command: [composer install")
				require.Contains(t, out, "Executing Composer command: [composer run-script post-create-project-cmd]")
				// Check the actual result of executing composer scripts
				require.FileExists(t, filepath.Join(composerDirOnHost, "created-by-post-root-package-install"))
				require.FileExists(t, filepath.Join(composerDirOnHost, "created-by-post-create-project-cmd"))
				// Check vendor directory
				require.NoDirExists(t, filepath.Join(composerDirOnHost, "vendor"))
			}

			// ddev composer create --no-dev --prefer-install=auto ddev/ddev-test-composer-create
			if composerCommandTypeCheck == "installation with --no-dev --prefer-install=auto" {
				// Check what was executed or not
				require.Contains(t, out, "Executing Composer command: [composer create-project --no-dev --prefer-install=auto ddev/ddev-test-composer-create --no-plugins --no-scripts --no-install")
				require.Contains(t, out, "Executing Composer command: [composer run-script post-root-package-install --no-dev]")
				require.Contains(t, out, "Executing Composer command: [composer install --no-dev --prefer-install=auto]")
				require.Contains(t, out, "Executing Composer command: [composer run-script post-create-project-cmd --no-dev]")
				// Check the actual result of executing composer scripts
				require.FileExists(t, filepath.Join(composerDirOnHost, "created-by-post-root-package-install"))
				require.FileExists(t, filepath.Join(composerDirOnHost, "created-by-post-create-project-cmd"))
				// Check vendor directory
				require.FileExists(t, filepath.Join(composerDirOnHost, "vendor", "autoload.php"))
				require.FileExists(t, filepath.Join(composerDirOnHost, "vendor", "ddev", "ddev-test-composer-require", "composer.json"))
				require.NoFileExists(t, filepath.Join(composerDirOnHost, "vendor", "ddev", "ddev-test-composer-require-dev", "composer.json"))
			}

			require.Contains(t, out, "Moving install to Composer root")
			require.Contains(t, out, "ddev composer create was successful")

			// Check that resulting composer.json (copied from testdata) has post-root-package-install and post-create-project-cmd scripts
			composerManifest, err := composer.NewManifest(filepath.Join(composerDirOnHost, "composer.json"))
			require.NoError(t, err)
			require.True(t, composerManifest != nil)
			require.True(t, composerManifest.HasPostRootPackageInstallScript())
			require.True(t, composerManifest.HasPostCreateProjectCmdScript())

			err = app.Stop(true, false)
			require.NoError(t, err)
		}
	}
}

func TestComposerCreateAutocomplete(t *testing.T) {
	// DDEV_DEBUG may result in extra output that we don't want
	origDdevDebug := os.Getenv("DDEV_DEBUG")
	_ = os.Unsetenv("DDEV_DEBUG")
	// Change to the directory for the project to test.
	// We don't really care what the project is, they should
	// all have composer installed in the web container.
	origDir, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(TestSites[0].Dir)
	require.NoError(t, err)

	t.Cleanup(func() {
		err = os.Chdir(origDir)
		require.NoError(t, err)
		_ = os.Setenv("DDEV_DEBUG", origDdevDebug)
	})

	// Make sure the sites exist and are running
	err = addSites()
	require.NoError(t, err)

	// Pressing tab after `composer completion` should result in the completion "bash"
	out, err := exec.RunHostCommand(DdevBin, "__complete", "composer", "create", "--")
	require.NoError(t, err)
	// Completions are terminated with ":4", so just grab the stuff before that
	completions, _, found := strings.Cut(out, ":")
	require.True(t, found)
	// We don't need to check all of the possible options - just check that
	// we're getting some completion suggestions that make sense and not just garbage
	require.Contains(t, completions, "--no-install")
	require.Contains(t, completions, "--no-scripts")
	require.Contains(t, completions, "--keep-vcs")
}
