package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
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
			t.Logf("== BEGIN TestComposerCreateCmd for project of type '%s' with docroot '%s'\n", projectType, docRoot)
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
				if slices.Contains([]string{nodeps.AppTypeDrupal11, nodeps.AppTypeDrupal10, nodeps.AppTypeDrupal9, nodeps.AppTypeDrupal8}, projectType) {
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

			cmd := ""
			// These are different conditions to test different composer flag combinations
			// Conditions for docRoot and projectType are not important here, they are only needed to make the test act different each time
			if docRoot == "" {
				cmd = "ddev composer create --no-plugins --no-scripts ddev/ddev-test-composer-create"
				if projectType == nodeps.AppTypePHP {
					cmd = "ddev composer create --no-install ddev/ddev-test-composer-create custom_directory"
					composerDirOnHost = filepath.Join(composerDirOnHost, "custom_directory")
					err = os.MkdirAll(composerDirOnHost, 0755)
					require.NoError(t, err)
				}
			} else {
				if projectType != nodeps.AppTypePHP {
					cmd = "ddev composer create ddev/ddev-test-composer-create --prefer-install auto another_directory --no-dev v1.0.0"
					composerDirOnHost = filepath.Join(composerDirOnHost, "another_directory")
					err = os.MkdirAll(composerDirOnHost, 0755)
					require.NoError(t, err)
				} else {
					cmd = "ddev composer create -vvv ddev/ddev-test-composer-create --prefer-install=auto --fake-flag ."
				}
			}

			t.Logf("Attempting cmd='%s' with docroot='%s' composer_root='%s' type='%s'", cmd, docRoot, docRoot, projectType)
			args := strings.Split(strings.TrimPrefix(cmd, "ddev "), " ")

			// If a file exists in the composer root then composer create should fail
			file, err := os.Create(filepath.Join(composerDirOnHost, "touch1.txt"))
			require.NoError(t, err)
			out, err = exec.RunHostCommand(DdevBin, args...)
			require.Error(t, err)
			require.Contains(t, out, "touch1.txt")
			_ = file.Close()
			_ = os.Remove(filepath.Join(composerDirOnHost, "touch1.txt"))

			// At this point, custom_directory and another_directory are empty
			// Remove custom_directory to see if it will be created by Composer
			// And do not remove another_directory to see if Composer will write to it
			if strings.Contains(cmd, "custom_directory") {
				_ = os.RemoveAll(composerDirOnHost)
			}

			// Test success
			out, err = exec.RunHostCommand(DdevBin, args...)
			require.NoError(t, err, "['%s'] failed to run %v: err=%v, output=\n=====\n%s\n=====\n", cmd, args, err, out)
			require.Contains(t, out, "Created project in ")
			require.FileExists(t, filepath.Join(composerDirOnHost, "composer.json"))

			if cmd == "ddev composer create --no-plugins --no-scripts ddev/ddev-test-composer-create" {
				// Check what was executed or not
				require.Contains(t, out, "Executing Composer command: [composer create-project --no-plugins --no-scripts --no-install ddev/ddev-test-composer-create /tmp/")
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

			if cmd == "ddev composer create -vvv ddev/ddev-test-composer-create --prefer-install=auto --fake-flag ." {
				// Check what was executed or not
				require.Contains(t, out, "Executing Composer command: [composer create-project -vvv --prefer-install=auto --no-plugins --no-scripts --no-install ddev/ddev-test-composer-create /tmp/")
				require.Contains(t, out, "Executing Composer command: [composer run-script post-root-package-install -vvv]")
				require.Contains(t, out, "Executing Composer command: [composer install -vvv --prefer-install=auto]")
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

			if cmd == "ddev composer create --no-install ddev/ddev-test-composer-create custom_directory" {
				// Check what was executed or not
				require.Contains(t, out, "Executing Composer command: [composer create-project --no-install --no-plugins --no-scripts ddev/ddev-test-composer-create /tmp/")
				require.Contains(t, out, "custom_directory")
				require.Contains(t, out, "Executing Composer command: [composer run-script post-root-package-install]")
				require.NotContains(t, out, "Executing Composer command: [composer install")
				require.Contains(t, out, "Executing Composer command: [composer run-script post-create-project-cmd]")
				// Check the actual result of executing composer scripts
				require.FileExists(t, filepath.Join(composerDirOnHost, "created-by-post-root-package-install"))
				require.FileExists(t, filepath.Join(composerDirOnHost, "created-by-post-create-project-cmd"))
				// Check vendor directory
				require.NoDirExists(t, filepath.Join(composerDirOnHost, "vendor"))
			}

			if cmd == "ddev composer create ddev/ddev-test-composer-create --prefer-install auto another_directory --no-dev v1.0.0" {
				// Check what was executed or not
				require.Contains(t, out, "Executing Composer command: [composer create-project --prefer-install auto --no-dev --no-plugins --no-scripts --no-install ddev/ddev-test-composer-create /tmp/")
				require.Contains(t, out, "another_directory v1.0.0")
				require.Contains(t, out, "Executing Composer command: [composer run-script post-root-package-install --no-dev]")
				require.Contains(t, out, "Executing Composer command: [composer install --prefer-install auto --no-dev]")
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
