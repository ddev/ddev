package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/composer"
	"github.com/ddev/ddev/pkg/config/types"
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/testcommon"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComposerCreateCmd(t *testing.T) {
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

	validAppTypes := ddevapp.GetValidAppTypesWithoutAliases()
	if os.Getenv("GOTEST_SHORT") != "" {
		validAppTypes = []string{nodeps.AppTypePHP, nodeps.AppTypeDrupal}
	}

	for _, docRoot := range []string{"", "doc-root"} {
		for _, projectType := range validAppTypes {
			if projectType == nodeps.AppTypeDjango4 || projectType == nodeps.AppTypePython {
				// Skip as an empty django4/python do not start nicely right away
				// https://github.com/ddev/ddev/issues/5171
				t.Logf("== SKIP TestComposerCreateCmd for project of type '%s' with docroot '%s'\n", projectType, docRoot)
				t.Logf("== SKIP python projects are not starting up nicely and composer create is very unlikely to be used")
				continue
			}
			if projectType == nodeps.AppTypeDrupal6 {
				t.Logf("== SKIP TestComposerCreateCmd for project of type '%s' with docroot '%s'\n", projectType, docRoot)
				t.Logf("== SKIP drupal6 projects uses a very old php version and composer create is very unlikely to be used")
				continue
			}
			t.Logf("== BEGIN TestComposerCreateCmd for project of type '%s' with docroot  '%s'\n", projectType, docRoot)
			tmpDir := testcommon.CreateTmpDir(t.Name() + projectType)
			err = os.Chdir(tmpDir)
			assert.NoError(err)

			// Prepare arguments
			arguments := []string{"config", "--project-type", projectType}

			composerRoot := tmpDir
			if docRoot != "" {
				arguments = append(arguments, "--docroot", docRoot)
				// For Drupal, we test that the composer root is the same as the created root
				if projectType == nodeps.AppTypeDrupal {
					arguments = append(arguments, "--composer-root", docRoot)
					composerRoot = filepath.Join(tmpDir, docRoot)
				}
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
				err = os.Chdir(origDir)
				assert.NoError(err)
				_ = os.RemoveAll(tmpDir)
			})

			err = app.StartAndWait(5)
			require.NoError(t, err)

			// This is a local package that we can use to test composer create
			repository := `{"type": "path", "url": ".ddev/test-ddev-composer-create", "options": {"symlink": false}}`
			repositoryPath := filepath.Join(origDir, "testdata", t.Name(), ".ddev/test-ddev-composer-create")
			err = fileutil.CopyDir(repositoryPath, filepath.Join(app.AppRoot, ".ddev/test-ddev-composer-create"))
			require.NoError(t, err)
			err = app.MutagenSyncFlush()
			require.NoError(t, err)

			// ddev composer create --repository '{"type": "path", "url": ".ddev/test-ddev-composer-create", "options": {"symlink": false}}' --no-interaction --no-dev -av --fake-flag test/ddev-composer-create
			// Rules:
			// combined short options will be split into single options, i.e. -av here will become -a -v
			// all invalid options will be ignored without any error: --fake-flag
			// composer commands will use only the flags they are supposed to use, i.e. -a (short form of --classmap-authoritative) will not be used by create-project
			// Validation:
			// valid options for create-project: --repository --no-interaction --no-dev -v
			// valid options for run-script: --no-interaction --no-dev -v
			// valid options for install: --no-interaction --no-dev -a -v
			args := []string{"composer", "create", "--repository", repository, "--no-interaction", "--no-dev", "-av", "--fake-flag", "test/ddev-composer-create"}

			// If --no-install is provided, the 'install' and 'run-script post-create-project-cmd' scripts will not run, but 'run-script post-root-package-install' will
			if docRoot != "" {
				// ddev composer create --repository '{"type": "path", "url": ".ddev/test-ddev-composer-create", "options": {"symlink": false}}' --no-interaction --no-install test/ddev-composer-create
				// Validation:
				// valid options for create-project: --repository --no-interaction --no-install
				// valid options for run-script: --no-interaction
				// valid options for install: doesn't run
				args = []string{"composer", "create", "--repository", repository, "--no-interaction", "--no-install", "test/ddev-composer-create"}
			}

			// If --preserve-flags is provided, the --no-plugins and --no-scripts flags will not be added
			if docRoot != "" && projectType == nodeps.AppTypePHP {
				// ddev composer create --repository '{"type": "path", "url": ".ddev/test-ddev-composer-create", "options": {"symlink": false}}' --preserve-flags --no-interaction test/ddev-composer-create
				// Validation:
				// valid options for create-project: --repository --no-interaction --no-install
				// valid options for run-script: none
				// valid options for install: doesn't run
				args = []string{"composer", "create", "--repository", repository, "--preserve-flags", "--no-interaction", "test/ddev-composer-create"}
			}

			// Test failure
			file, err := os.Create(filepath.Join(composerRoot, "touch1.txt"))
			out, err = exec.RunHostCommand(DdevBin, args...)
			assert.Error(err)
			assert.Contains(out, "touch1.txt")
			_ = file.Close()
			_ = os.Remove(filepath.Join(composerRoot, "touch1.txt"))

			// Test success
			out, err = exec.RunHostCommand(DdevBin, args...)
			assert.NoError(err, "failed to run %v: err=%v, output=\n=====\n%s\n=====\n", args, err, out)
			assert.Contains(out, "Created project in ")
			assert.FileExists(filepath.Join(composerRoot, "composer.json"))

			// Check that all composer commands were run, these include the specified flags
			if docRoot == "" {
				// When there is no --preserve-flags, the --no-plugins and --no-scripts flags are appended
				// --no-install is always added to 'composer create-project' to avoid problems with installation
				assert.Contains(out, `Executing Composer command: [composer create-project --repository {"type": "path", "url": ".ddev/test-ddev-composer-create", "options": {"symlink": false}} --no-interaction --no-dev -v test/ddev-composer-create --no-plugins --no-scripts --no-install`)
				assert.Contains(out, "Executing composer command: [composer run-script post-root-package-install --no-interaction --no-dev -v]")
				// -av is expanded to -a -v
				assert.Contains(out, "Executing Composer command: [composer install --no-interaction --no-dev -a -v]")
				assert.Contains(out, "Executing composer command: [composer run-script post-create-project-cmd --no-interaction --no-dev -v]")

				// Check if the files created by composer scripts from the test/composer-create-run-script package are here
				assert.FileExists(filepath.Join(composerRoot, "created-by-post-root-package-install"))
				assert.FileExists(filepath.Join(composerRoot, "created-by-post-create-project-cmd"))
				// Check if vendor/autoload.php is created
				assert.FileExists(filepath.Join(composerRoot, "vendor", "autoload.php"))
				// psr/log is installed by test/ddev-composer-create package
				assert.DirExists(filepath.Join(composerRoot, "vendor", "psr", "log"))
			} else {
				// If --preserve-flags is provided, the --no-plugins and --no-scripts flags will not be added
				if projectType == nodeps.AppTypePHP {
					assert.Contains(out, `Executing Composer command: [composer create-project --repository {"type": "path", "url": ".ddev/test-ddev-composer-create", "options": {"symlink": false}} --no-interaction test/ddev-composer-create --no-install`)
					assert.Contains(out, "Executing Composer command: [composer install --no-interaction]")
					// With --preserve-flags scripts will run, but without DDEV help
					assert.NotContains(out, "Executing composer command: [composer run-script post-root-package-install")
					assert.NotContains(out, "Executing composer command: [composer run-script post-create-project-cmd")
					// Check if the files created by composer scripts from the test/composer-create-run-script package are here
					assert.FileExists(filepath.Join(composerRoot, "created-by-post-root-package-install"))
					assert.FileExists(filepath.Join(composerRoot, "created-by-post-create-project-cmd"))
					// Check if vendor/autoload.php is created
					assert.FileExists(filepath.Join(composerRoot, "vendor", "autoload.php"))
					// psr/log is installed by test/ddev-composer-create package
					assert.DirExists(filepath.Join(composerRoot, "vendor", "psr", "log"))
				} else {
					// If --no-install is provided, the 'install' and 'run-script post-create-project-cmd' flags will not run, but 'run-script post-root-package-install' will
					assert.Contains(out, `Executing Composer command: [composer create-project --repository {"type": "path", "url": ".ddev/test-ddev-composer-create", "options": {"symlink": false}} --no-interaction --no-install test/ddev-composer-create --no-plugins --no-scripts`)
					assert.Contains(out, "Executing composer command: [composer run-script post-root-package-install --no-interaction]")
					assert.NotContains(out, "Executing Composer command: [composer install")
					assert.NotContains(out, "Executing composer command: [composer run-script post-create-project-cmd")
					// Check if the files created by composer scripts from the test/composer-create-run-script package are here
					assert.FileExists(filepath.Join(composerRoot, "created-by-post-root-package-install"))
					// The post-create-project-cmd script doesn't run, so there should be no file
					assert.NoFileExists(filepath.Join(composerRoot, "created-by-post-create-project-cmd"))
					// If --no-install is provided, no vendor should be created
					assert.NoDirExists(filepath.Join(composerRoot, "vendor"))
				}
			}

			// Check that resulting composer.json (copied from testdata) has post-root-package-install and post-create-project-cmd scripts
			composerManifest, err := composer.NewManifest(filepath.Join(composerRoot, "composer.json"))
			assert.NoError(err)
			assert.True(composerManifest != nil)
			assert.True(composerManifest.HasPostRootPackageInstallScript())
			assert.True(composerManifest.HasPostCreateProjectCmdScript())

			err = app.Stop(true, false)
			require.NoError(t, err)
		}
	}
}

func TestComposerCreateAutocomplete(t *testing.T) {
	assert := asrt.New(t)

	// Change to the directory for the project to test.
	// We don't really care what the project is, they should
	// all have composer installed in the web container.
	origDir, err := os.Getwd()
	assert.NoError(err)
	err = os.Chdir(TestSites[0].Dir)
	assert.NoError(err)

	t.Cleanup(func() {
		err = os.Chdir(origDir)
		assert.NoError(err)
	})

	// Make sure the sites exist and are running
	err = addSites()
	require.NoError(t, err)

	// Pressing tab after `composer completion` should result in the completion "bash"
	out, err := exec.RunHostCommand(DdevBin, "__complete", "composer", "create", "--")
	assert.NoError(err)
	// Completions are terminated with ":4", so just grab the stuff before that
	completions, _, found := strings.Cut(out, ":")
	assert.True(found)
	// We don't need to check all of the possible options - just check that
	// we're getting some completion suggestions that make sense and not just garbage
	assert.Contains(completions, "--no-install")
	assert.Contains(completions, "--no-scripts")
	assert.Contains(completions, "--keep-vcs")
}
