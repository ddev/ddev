package ddevapp_test

import (
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/drud/ddev/pkg/testcommon"
	"github.com/stretchr/testify/require"
	"os"
	"strings"
	"testing"

	asrt "github.com/stretchr/testify/assert"
)

// TestComposer does trivial tests of the ddev composer command
// More tests are found in the cmd package
func TestComposer(t *testing.T) {
	assert := asrt.New(t)
	app := &ddevapp.DdevApp{}
	origDir, _ := os.Getwd()

	// Use drupal9 only for this test, just need a little composer action
	site := FullTestSites[8]
	// If running this with GOTEST_SHORT we have to create the directory, tarball etc.
	if site.Dir == "" || !fileutil.FileExists(site.Dir) {
		app := &ddevapp.DdevApp{Name: site.Name}
		_ = app.Stop(true, false)
		_ = globalconfig.RemoveProjectInfo(site.Name)

		err := site.Prepare()
		require.NoError(t, err)
		t.Cleanup(func() {
			err = app.Stop(true, false)
			assert.NoError(err)
			err := os.Chdir(origDir)
			assert.NoError(err)
			err = os.RemoveAll(app.AppRoot)
			assert.NoError(err)
		})
	}

	_ = os.Chdir(site.Dir)

	testcommon.ClearDockerEnv()
	err := app.Init(site.Dir)
	assert.NoError(err)
	app.Hooks = map[string][]ddevapp.YAMLTask{"post-composer": {{"exec-host": "touch hello-post-composer-" + app.Name}}, "pre-composer": {{"exec-host": "touch hello-pre-composer-" + app.Name}}}
	// Make sure we get rid of this for other uses

	t.Cleanup(func() {
		err = os.Chdir(origDir)
		assert.NoError(err)
		app.Hooks = nil
		app.ComposerVersion = ""
		_ = app.WriteConfig()
		_ = app.Stop(true, false)
	})

	err = app.Start()
	require.NoError(t, err)

	// Make sure to remove the var-dump-server to start; composer install should replace it.
	_ = os.RemoveAll("vendor/bin/var-dump-server")

	err = app.MutagenSyncFlush()
	assert.NoError(err)

	_, _, err = app.Composer([]string{"install", "--no-progress", "--no-interaction"})
	assert.NoError(err)
	err = app.MutagenSyncFlush()
	assert.NoError(err)

	out, _, err := app.Exec(&ddevapp.ExecOpts{
		Cmd: "ls -l vendor/bin/var-dump-server | awk '{print $1}'",
	})
	assert.NoError(err)
	assert.True(strings.HasPrefix(out, "lrwx") || strings.HasPrefix(out, "-rwx"), "perms of var-dump-server should be 'lrwx' or '-rwx', got '%s' instead", out)

	_, _, err = app.Exec(&ddevapp.ExecOpts{
		Cmd: "vendor/bin/var-dump-server -h",
	})
	assert.NoError(err)

	assert.FileExists("hello-pre-composer-" + app.Name)
	assert.FileExists("hello-post-composer-" + app.Name)
	err = os.Remove("hello-pre-composer-" + app.Name)
	assert.NoError(err)
	err = os.Remove("hello-post-composer-" + app.Name)
	assert.NoError(err)
}

// TestComposerVersion tests to make sure that composer_version setting
// works correctly
func TestComposerVersion(t *testing.T) {
	assert := asrt.New(t)

	testDir := testcommon.CreateTmpDir(t.Name())

	origDir, _ := os.Getwd()
	err := os.Chdir(testDir)
	assert.NoError(err)

	app, err := ddevapp.NewApp(testDir, false)
	assert.NoError(err)
	app.Name = t.Name()
	err = app.WriteConfig()
	require.NoError(t, err)

	t.Cleanup(func() {
		err = app.Stop(true, false)
		assert.NoError(err)
		err = os.Chdir(origDir)
		assert.NoError(err)
		// Mutagen can compete with removal, so go ahead and ignore result
		_ = os.RemoveAll(testDir)
	})

	// Make sure base version (default) is composer v2
	err = app.Start()
	require.NoError(t, err)
	stdout, _, err := app.Exec(&ddevapp.ExecOpts{Cmd: "composer --version"})
	assert.NoError(err)
	assert.True(strings.HasPrefix(stdout, "Composer 2") || strings.HasPrefix(stdout, "Composer version 2"), "composer version not the expected composer 2: %v", stdout)

	// Make sure it does the right thing with 1.x
	app.ComposerVersion = "1"
	err = app.Restart()
	require.NoError(t, err)
	stdout, _, err = app.Exec(&ddevapp.ExecOpts{Cmd: "composer --version"})
	assert.NoError(err)
	assert.Contains(stdout, "Composer version 1")

	// With version "2" we should be back to latest v2
	app.ComposerVersion = "2"
	err = app.Restart()
	require.NoError(t, err)
	stdout, _, err = app.Exec(&ddevapp.ExecOpts{Cmd: "composer --version"})
	assert.NoError(err)
	assert.True(strings.HasPrefix(stdout, "Composer 2") || strings.HasPrefix(stdout, "Composer version 2"), "composer version doesn't start with the expected value: %v", stdout)

	// With explicit version, we should get that version
	app.ComposerVersion = "2.0.1"
	err = app.Restart()
	require.NoError(t, err)
	stdout, _, err = app.Exec(&ddevapp.ExecOpts{Cmd: "composer --version"})
	assert.NoError(err)
	assert.Contains(stdout, "Composer version "+app.ComposerVersion)
}
