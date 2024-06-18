package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/config/types"
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/exec"
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

	types := ddevapp.GetValidAppTypesWithoutAliases()
	if os.Getenv("GOTEST_SHORT") != "" {
		types = []string{nodeps.AppTypePHP, nodeps.AppTypeDrupal}
	}

	for _, docRoot := range []string{"", "doc-root"} {
		for _, projectType := range types {
			if projectType == nodeps.AppTypeDjango4 || projectType == nodeps.AppTypePython {
				// Skip as an empty django4/python do not start nicely right away
				// https://github.com/ddev/ddev/issues/5171
				t.Logf("== SKIP TestComposerCreateCmd for project of type '%s' with docroot  '%s'\n", projectType, docRoot)
				t.Logf("== SKIP python projects are not starting up nicely and composer create is very unlikely to be used")
				continue
			}
			if projectType == nodeps.AppTypeDrupal6 {
				// Skip as an empty django4/python do not start nicely right away
				// https://github.com/ddev/ddev/issues/5171
				t.Logf("== SKIP TestComposerCreateCmd for project of type '%s' with docroot  '%s'\n", projectType, docRoot)
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
				// For Drupal we test that the composer root is the same as the created root
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
			// ddev composer create --prefer-dist --no-interaction --no-dev psr/log:1.1.0
			args := []string{"composer", "create", "--prefer-dist", "--no-interaction", "--no-dev", "--no-install", "psr/log:1.1.0"}

			// Test failure
			file, err := os.Create(filepath.Join(composerRoot, "touch1.txt"))
			out, err = exec.RunHostCommand(DdevBin, args...)
			assert.Error(err)
			assert.Contains(out, "touch1.txt")
			_ = file.Close()
			os.Remove(filepath.Join(composerRoot, "touch1.txt"))

			// Test success
			out, err = exec.RunHostCommand(DdevBin, args...)
			assert.NoError(err, "failed to run %v: err=%v, output=\n=====\n%s\n=====\n", args, err, out)
			assert.Contains(out, "Created project in ")
			assert.FileExists(filepath.Join(composerRoot, "composer.json"))

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
