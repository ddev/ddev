package ddevapp_test

import (
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/drud/ddev/pkg/testcommon"
	"github.com/stretchr/testify/require"
	"os"
	"testing"

	asrt "github.com/stretchr/testify/assert"
)

// TestComposer does trivial tests of the ddev composer command
// More tests are found in the cmd package
func TestComposer(t *testing.T) {
	assert := asrt.New(t)
	app := &ddevapp.DdevApp{}

	// Use drupal8 only for this test, just need a little composer action
	site := FullTestSites[8]
	// If running this with GOTEST_SHORT we have to create the directory, tarball etc.
	if site.Dir == "" || !fileutil.FileExists(site.Dir) {
		app := &ddevapp.DdevApp{Name: site.Name}
		_ = app.Stop(true, false)
		_ = globalconfig.RemoveProjectInfo(site.Name)

		err := site.Prepare()
		require.NoError(t, err)
		// nolint: errcheck
		defer os.RemoveAll(site.Dir)
	}

	testDir, _ := os.Getwd()
	// nolint: errcheck
	defer os.Chdir(testDir)
	_ = os.Chdir(site.Dir)

	testcommon.ClearDockerEnv()
	err := app.Init(site.Dir)
	app.ComposerVersion = "2"
	assert.NoError(err)
	app.Hooks = map[string][]ddevapp.YAMLTask{"post-composer": {{"exec-host": "touch hello-post-composer-" + app.Name}}, "pre-composer": {{"exec-host": "touch hello-pre-composer-" + app.Name}}}
	// Make sure we get rid of this for other uses
	defer func() {
		app.Hooks = nil
		app.ComposerVersion = ""
		_ = app.WriteConfig()
		_ = app.Stop(true, false)
	}()
	err = app.Start()
	require.NoError(t, err)
	_, _, err = app.Composer([]string{"install"})
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

	pwd, _ := os.Getwd()
	err := os.Chdir(testDir)
	assert.NoError(err)

	app, err := ddevapp.NewApp(testDir, false)
	assert.NoError(err)

	t.Cleanup(func() {
		err = app.Stop(true, false)
		assert.NoError(err)
		err = os.Chdir(pwd)
		assert.NoError(err)
		err = os.RemoveAll(testDir)
		assert.NoError(err)
	})

	// Make sure base version (default) is composer v1
	err = app.Start()
	require.NoError(t, err)
	stdout, _, err := app.Exec(&ddevapp.ExecOpts{Cmd: "composer --version"})
	assert.NoError(err)
	assert.Contains(stdout, "Composer version 2")

	// Make sure it does the right thing with latest 2.x
	app.ComposerVersion = "1"
	err = app.Start()
	require.NoError(t, err)
	stdout, _, err = app.Exec(&ddevapp.ExecOpts{Cmd: "composer --version"})
	assert.NoError(err)
	assert.Contains(stdout, "Composer version 1")

	// With version "2" we should be back to latest v2
	app.ComposerVersion = "2"
	err = app.Start()
	require.NoError(t, err)
	stdout, _, err = app.Exec(&ddevapp.ExecOpts{Cmd: "composer --version"})
	assert.NoError(err)
	assert.Contains(stdout, "Composer version 2")

	// With explicit version, we should get that version
	app.ComposerVersion = "2.0.1"
	err = app.Start()
	require.NoError(t, err)
	stdout, _, err = app.Exec(&ddevapp.ExecOpts{Cmd: "composer --version"})
	assert.NoError(err)
	assert.Contains(stdout, "Composer version "+app.ComposerVersion)
}
