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
	"runtime"
	"strings"
	"testing"
	"time"
)

// TestMutagenSimple tests basic mutagen functionality
func TestMutagenSimple(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("TestMutagenSimple one takes way too long on Windows, skipping")
	}
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
	err = app.WriteConfig()
	require.NoError(t, err)

	t.Cleanup(func() {
		err = os.Chdir(origDir)
		assert.NoError(err)
		err = app.Stop(true, false)
		assert.NoError(err)
		assert.False(dockerutil.VolumeExists(ddevapp.GetMutagenVolumeName(app)))
	})
	err = app.Start()
	require.NoError(t, err)

	assert.True(dockerutil.VolumeExists(ddevapp.GetMutagenVolumeName(app)))

	desc, err := app.Describe(false)
	assert.True(desc["mutagen_enabled"].(bool))

	// Make sure the sync is there
	out, err := exec.RunHostCommand(mutagenPath, "sync", "list", ddevapp.MutagenSyncName(app.Name))
	assert.NoError(err, "output=%s", out)

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
	stdout, stderr, err := app.Composer([]string{"install", "--no-progress", "--no-interaction"})
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
	out, err = exec.RunHostCommand(mutagenPath, "sync", "list", ddevapp.MutagenSyncName(app.Name))
	assert.Error(err, "output=%s", out)

	// Make sure we can stop the daemon
	ddevapp.StopMutagenDaemon()
	if runtime.GOOS == "darwin" || runtime.GOOS == "linux" {
		out, err := exec.RunHostCommand("bash", "-c", "ps -ef | grep mutagen")
		assert.NoError(err)
		t.Logf("mutagen processes after StopMutagenDaemon: \n=====\n%s====\n", out)
	}

	out, err = exec.RunHostCommand(globalconfig.GetMutagenPath(), "sync", "list")
	assert.NoError(err)
	assert.Contains(out, "Started Mutagen daemon in background")
	if !strings.Contains(out, "Started Mutagen daemon in background") && (runtime.GOOS == "darwin" || runtime.GOOS == "linux") {
		out, err := exec.RunHostCommand("bash", "-c", "ps -ef | grep mutagen")
		assert.NoError(err)
		t.Logf("current mutagen processes: \n=====\n%s====\n", out)
	}

	err = app.Start()
	assert.NoError(err)

	// Make sure sync is down on pause also
	err = app.Pause()
	out, err = exec.RunHostCommand(mutagenPath, "sync", "list", ddevapp.MutagenSyncName(app.Name))
	assert.Error(err, "output=%s", out)

	// And that it's re-established when we start again
	err = app.Start()
	out, err = exec.RunHostCommand(mutagenPath, "sync", "list", ddevapp.MutagenSyncName(app.Name))
	assert.NoError(err, "could not run mutagen sync list: output=%s", out)

	runTime()
}
