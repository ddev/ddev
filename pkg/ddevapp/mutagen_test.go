package ddevapp_test

import (
	"fmt"
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/drud/ddev/pkg/util"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestMutagenSimple tests basic mutagen functionality
func TestMutagenSimple(t *testing.T) {
	assert := asrt.New(t)

	mutagenPath := globalconfig.GetMutagenPath()

	// Make sure this leaves us in the original test directory
	origDir, _ := os.Getwd()

	// Use Drupal9 as it is a good target for composer failures
	site := FullTestSites[8]
	// We will create directory from scratch, as we'll be removing files and changing it.
	app := &ddevapp.DdevApp{Name: site.Name}
	_ = app.Stop(true, false)
	_ = globalconfig.RemoveProjectInfo(site.Name)

	err := site.Prepare()
	require.NoError(t, err)
	runTime := util.TimeTrack(time.Now(), fmt.Sprintf("%s %s", site.Name, t.Name()))

	err = app.Init(site.Dir)
	assert.NoError(err)
	app.MutagenEnabled = true

	t.Cleanup(func() {
		err = os.Chdir(origDir)
		assert.NoError(err)
		err = app.Stop(true, false)
		assert.NoError(err)
		assert.False(dockerutil.VolumeExists(app.Name + "_project_mutagen"))
	})
	err = app.Start()
	assert.NoError(err)

	assert.True(dockerutil.VolumeExists(app.Name + "_project_mutagen"))

	desc, err := app.Describe(false)
	assert.True(desc["mutagen_enabled"].(bool))

	// Make sure the sync is there
	_, err = exec.RunCommand(mutagenPath, []string{"sync", "list", ddevapp.MutagenSyncName(app.Name)})
	assert.NoError(err)

	// Remove the vendor directory and sync
	err = os.RemoveAll(filepath.Join(app.AppRoot, "vendor"))
	assert.NoError(err)
	err = app.MutagenSyncFlush()
	assert.NoError(err)
	_, stderr, err := app.Exec(&ddevapp.ExecOpts{
		Cmd: "ls -l /var/www/html/vendor",
	})
	assert.Contains(stderr, "cannot access '/var/www/html/vendor'")

	// Now composer install again and make sure all the stuff comes back
	stdout, stderr, err := app.Composer([]string{"install"})
	assert.NoError(err)
	t.Logf("composer install output: \nstdout: %s\n\nstderr: %s\n", stdout, stderr)
	_, _, err = app.Exec(&ddevapp.ExecOpts{
		Cmd: "ls -l vendor/bin/var-dump-server >/dev/null",
	})
	assert.NoError(err)

	err = app.MutagenSyncFlush()
	assert.NoError(err)
	assert.FileExists(filepath.Join(app.AppRoot, "vendor/bin/var-dump-server"))

	// Stop app, should result in no more mutagen sync
	err = app.Stop(false, false)
	_, err = exec.RunCommand(mutagenPath, []string{"sync", "list", ddevapp.MutagenSyncName(app.Name)})
	assert.Error(err)

	err = app.Start()
	assert.NoError(err)

	// Make sure sync is down on pause also
	err = app.Pause()
	_, err = exec.RunCommand(mutagenPath, []string{"sync", "list", ddevapp.MutagenSyncName(app.Name)})
	assert.Error(err)

	// And that it's re-established when we start again
	err = app.Start()
	_, err = exec.RunCommand(mutagenPath, []string{"sync", "list", ddevapp.MutagenSyncName(app.Name)})
	assert.NoError(err)

	runTime()
}
