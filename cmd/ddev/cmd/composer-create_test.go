package cmd

import (
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"

	"github.com/ddev/ddev/pkg/config/types"
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/testcommon"
	asrt "github.com/stretchr/testify/assert"
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

	types := ddevapp.GetValidAppTypes()
	if os.Getenv("GOTEST_SHORT") != "" {
		types = []string{nodeps.AppTypePHP, nodeps.AppTypeDrupal10}
	}

	for _, docRoot := range []string{"", "doc-root"} {
		for _, projectType := range types {
			if projectType == nodeps.AppTypeDjango4 {
				// Skip as an empty django4 do not start nicely
				// https://github.com/ddev/ddev/issues/5171
				t.Logf("== SKIP TestComposerCreateCmd for project of type '%s' with docroot  '%s'\n", projectType, docRoot)
				continue
			}
			t.Logf("== BEGIN TestComposerCreateCmd for project of type '%s' with docroot  '%s'\n", projectType, docRoot)
			tmpDir := testcommon.CreateTmpDir(t.Name() + projectType)
			err = os.Chdir(tmpDir)
			assert.NoError(err)

			// Prepare arguments
			arguments := []string{"config", "--project-type", projectType}

			if docRoot != "" {
				arguments = append(arguments, "--docroot", docRoot, "--create-docroot")
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
			// This often fail on Windows with NFS, also Colima
			// It appears to be something about Composer itself?

			err = app.StartAndWait(5)
			require.NoError(t, err)
			// ddev composer create --prefer-dist --no-interaction --no-dev psr/log:1.1.0
			args := []string{"composer", "create", "--prefer-dist", "--no-interaction", "--no-dev", "--no-install", "psr/log:1.1.0"}
			out, err = exec.RunHostCommand(DdevBin, args...)
			assert.NoError(err, "failed to run %v: err=%v, output=\n=====\n%s\n=====\n", args, err, out)
			assert.Contains(out, "Created project in ")
			assert.FileExists(filepath.Join(tmpDir, "composer.json"))
		}
	}
}
