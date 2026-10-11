package ddevapp_test

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/ddev/ddev/pkg/config/types"
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/ddev/ddev/pkg/util"
	"github.com/stretchr/testify/require"
)

// requireMutagenStatus fails the test unless the project's Mutagen sync has the wanted status.
func requireMutagenStatus(t *testing.T, app *ddevapp.DdevApp, want string) {
	t.Helper()
	status, short, long, err := app.MutagenStatus()
	require.NoError(t, err, "could not run Mutagen sync list: status=%s short=%s long=%v", status, short, long)
	require.Equal(t, want, status, "wrong Mutagen status: status=%s short=%s long=%v", status, short, long)
}

// logMutagenDiagnostics logs Mutagen's own view of the world, for use when a test has failed.
func logMutagenDiagnostics(t *testing.T) {
	t.Helper()
	for _, c := range [][]string{
		{globalconfig.GetMutagenPath(), "sync", "list", "-l"},
		{DdevBin, "utility", "mutagen-diagnose", "--all"},
		{"bash", "-c", "ps -ef | grep [m]utagen"},
	} {
		out, err := exec.RunHostCommand(c[0], c[1:]...)
		t.Logf("MUTAGEN DIAGNOSTIC %v (MUTAGEN_DATA_DIRECTORY=%s) err=%v:\n%s", c, globalconfig.GetMutagenDataDirectory(), err, out)
	}
}

// TestMutagenSimple tests basic Mutagen functionality
func TestMutagenSimple(t *testing.T) {
	if nodeps.IsWindows() {
		t.Skip("TestMutagenSimple takes way too long on Windows, skipping")
	}
	if nodeps.IsWSL2MirroredMode() {
		t.Skip("Skipping TestMutagenSimple in WSL2 mirrored mode; It can fail randomly on the 'composer install' step due to network issues, unable to download random dependencies from either dist or source.")
	}
	testcommon.SkipUnlessDefaultEnvironment(t)

	// Make sure there's not an existing Mutagen running, perhaps in wrong directory
	_, _ = exec.RunHostCommand("pkill", "mutagen")

	// Make sure this leaves us in the original test directory
	origDir, _ := os.Getwd()

	// CodeIgniter has a small Composer tree, so network trouble is unlikely to fail the install
	site := FullTestSites[slices.IndexFunc(FullTestSites, func(s testcommon.TestSite) bool { return s.Name == "TestPkgCodeigniter4" })]
	// We will create directory from scratch, as we'll be removing files and changing it.
	app := &ddevapp.DdevApp{Name: site.Name}
	_ = app.Stop(true, false)
	_ = globalconfig.RemoveProjectInfo(site.Name)

	err := site.Prepare()
	require.NoError(t, err)
	runTime := util.TimeTrackC(fmt.Sprintf("%s %s", site.Name, t.Name()))

	err = app.Init(site.Dir)
	require.NoError(t, err)
	app.SetPerformanceMode(types.PerformanceModeMutagen)
	err = app.WriteConfig()
	require.NoError(t, err)

	t.Cleanup(func() {
		runTime()
		err = os.Chdir(origDir)
		require.NoError(t, err)
		err = app.Stop(true, false)
		require.NoError(t, err)
		require.False(t, dockerutil.VolumeExists(ddevapp.GetMutagenVolumeName(app)))
	})
	// Registered after the Stop cleanup, so it runs first while the project still exists
	t.Cleanup(func() {
		if t.Failed() {
			logMutagenDiagnostics(t)
		}
	})

	start := time.Now()
	step := func(name string) {
		t.Helper()
		t.Logf("STEP (+%s): %s", time.Since(start).Round(time.Second), name)
	}

	step("first Start()")
	err = app.Start()
	require.NoError(t, err, "first Start()")

	require.True(t, dockerutil.VolumeExists(ddevapp.GetMutagenVolumeName(app)))

	// Make sure that we've added the proper config into the mutagen.yml file
	// These should *not* be added if NoBindMounts is true
	if !globalconfig.DdevGlobalConfig.NoBindMounts {
		mutagenFile := ddevapp.GetMutagenConfigFilePath(app)
		for _, d := range app.GetUploadDirs() {
			searchDir := path.Join(app.Docroot, d)
			exists, err := fileutil.FgrepStringInFile(mutagenFile, `- "/`+searchDir+`"`)
			require.NoError(t, err)
			require.True(t, exists, `upload_dir "/%s" not found in Mutagen config`, d)
		}
	}

	desc, err := app.Describe(false)
	require.True(t, desc["mutagen_enabled"].(bool))

	// Make sure the sync is there
	step("check sync is ok")
	requireMutagenStatus(t, app, "ok")

	// Remove the vendor directory and sync
	step("remove vendor and flush")
	err = os.RemoveAll(filepath.Join(app.AppRoot, "vendor"))
	require.NoError(t, err)
	err = app.MutagenSyncFlush()
	require.NoError(t, err)
	_, _, err = app.Exec(&ddevapp.ExecOpts{
		Cmd: "test -d /var/www/html/vendor",
	})
	require.Error(t, err, "/var/www/html/vendor exists and it should not")

	_, _, err = app.Composer([]string{"config", "--no-plugins", "allow-plugins", "true"})
	require.NoError(t, err)

	// Now composer install again and make sure all the stuff comes back
	step("composer install")
	stdout, stderr, err := app.Composer([]string{"install", "--no-progress", "--no-interaction"})
	require.NoError(t, err, "stdout=%s, stderr=%s, err=%v", stdout, stderr, err)
	_, _, err = app.Exec(&ddevapp.ExecOpts{
		Cmd: "test -f vendor/bin/phpunit >/dev/null",
	})
	require.NoError(t, err)

	err = app.MutagenSyncFlush()
	require.NoError(t, err)
	require.FileExists(t, filepath.Join(app.AppRoot, "vendor/bin/phpunit"))

	// Stop app, should result in paused sync
	step("Stop() should pause sync")
	err = app.Stop(false, false)
	require.NoError(t, err, "Stop() before daemon stop")
	requireMutagenStatus(t, app, "paused")

	// Make sure we can stop the daemon
	step("StopMutagenDaemon")
	ddevapp.StopMutagenDaemon("")
	if nodeps.IsMacOS() || nodeps.IsLinux() {
		// Verify that the Mutagen daemon stopped/died
		sleepWait := time.Second * 1
		if nodeps.IsLinux() {
			sleepWait = time.Second * 5
		}
		time.Sleep(sleepWait)
		_, err := exec.RunHostCommand("pkill", "-HUP", "mutagen")
		require.Error(t, err)
	}

	out, err := exec.RunHostCommand(globalconfig.GetMutagenPath(), "sync", "list")
	require.NoError(t, err, "Mutagen sync list failed with MUTAGEN_DATA_DIRECTORY=%s: out=%s: %v", globalconfig.GetMutagenDataDirectory(), out, err)
	require.Contains(t, out, "Started Mutagen daemon in background")

	step("second Start() after daemon stop")
	err = app.Start()
	require.NoError(t, err, "second Start() after StopMutagenDaemon")

	// Make sure sync is down on pause also
	step("Pause() should pause sync")
	err = app.Pause()
	require.NoError(t, err, "Pause()")
	requireMutagenStatus(t, app, "paused")

	// And that it's re-established when we start again
	step("third Start() after Pause()")
	err = app.Start()
	require.NoError(t, err, "third Start() after Pause()")
	requireMutagenStatus(t, app, "ok")

	err = app.Stop(false, false)
	require.NoError(t, err, "Stop() before mutagen-agents.tar.gz removal")

	// Remove the mutagen-agents.tar.gz file if it exists and start the app
	if fileutil.FileExists(filepath.Join(globalconfig.GetDDEVBinDir(), "mutagen-agents.tar.gz")) {
		err = os.Remove(filepath.Join(globalconfig.GetDDEVBinDir(), "mutagen-agents.tar.gz"))
		require.NoError(t, err)
	}
	step("Start() without mutagen-agents.tar.gz")
	err = app.Start()
	require.NoError(t, err, "Start() without mutagen-agents.tar.gz")
	// Verify that the mutagen-agents.tar.gz file was downloaded
	require.FileExists(t, filepath.Join(globalconfig.GetDDEVBinDir(), "mutagen-agents.tar.gz"))
	// Remove the mutagen-agents.tar.gz file again and restart the app
	err = os.Remove(filepath.Join(globalconfig.GetDDEVBinDir(), "mutagen-agents.tar.gz"))
	require.NoError(t, err)
	step("Restart() without mutagen-agents.tar.gz")
	err = app.Restart()
	require.NoError(t, err, "Restart() without mutagen-agents.tar.gz")
	// Verify again that the mutagen-agents.tar.gz file was downloaded
	require.FileExists(t, filepath.Join(globalconfig.GetDDEVBinDir(), "mutagen-agents.tar.gz"))

	// Make sure Mutagen daemon gets stopped on poweroff
	step("PowerOff")
	ddevapp.PowerOff()
	if nodeps.IsMacOS() || nodeps.IsLinux() {
		// Verify that the Mutagen daemon stopped/died
		sleepWait := time.Second * 5
		time.Sleep(sleepWait)
		// pkill -HUP checks for process existence.
		_, err := exec.RunHostCommand("pkill", "-HUP", "mutagen")
		require.Error(t, err)
	}
}

// TestMutagenConfigChange tests Mutagen new session creation on mutagen.yml change
func TestMutagenConfigChange(t *testing.T) {
	if !nodeps.IsMacOS() {
		t.Skip("TestMutagenConfigChange runs only on macOS (without Colima), skipping")
	}

	// Make sure there's not an existing Mutagen running, perhaps in wrong directory
	_, _ = exec.RunHostCommand("pkill", "mutagen")

	// Make sure this leaves us in the original test directory
	origDir, _ := os.Getwd()

	// Use Drupal11 as it is a good target for Composer failures
	site := FullTestSites[16]
	// We will create directory from scratch, as we'll be removing files and changing it.
	app := &ddevapp.DdevApp{Name: site.Name}
	_ = app.Stop(true, false)
	_ = globalconfig.RemoveProjectInfo(site.Name)

	err := site.Prepare()
	require.NoError(t, err)
	runTime := util.TimeTrackC(fmt.Sprintf("%s %s", site.Name, t.Name()))

	err = app.Init(site.Dir)
	require.NoError(t, err)
	app.SetPerformanceMode(types.PerformanceModeMutagen)
	err = app.WriteConfig()
	require.NoError(t, err)

	t.Cleanup(func() {
		runTime()
		err = os.Chdir(origDir)
		require.NoError(t, err)
		err = app.Stop(true, false)
		require.NoError(t, err)
		// We can remove mutagen.yml and it will be recreated.
		_ = os.RemoveAll(app.GetConfigPath("mutagen/mutagen.yml"))
		require.False(t, dockerutil.VolumeExists(ddevapp.GetMutagenVolumeName(app)))
	})
	err = app.Start()
	require.NoError(t, err)

	origSyncID, err := app.GetMutagenSyncID()
	require.NoError(t, err)

	// On first start, we have the standard hash from the ddev-provided mutagen.yml
	firstStartHash, err := ddevapp.GetMutagenConfigFileHashLabel(app)
	require.NoError(t, err)

	err = app.Restart()
	require.NoError(t, err)

	// After a restart, nothing should have changed, so we should have same label
	secondStartHash, err := ddevapp.GetMutagenConfigFileHashLabel(app)
	require.NoError(t, err)

	afterRestartSyncID, err := app.GetMutagenSyncID()
	require.NoError(t, err)
	require.Equal(t, origSyncID, afterRestartSyncID)

	require.Equal(t, firstStartHash, secondStartHash)

	err = app.Stop(false, false)
	require.NoError(t, err)

	// Now change the mutagen.yml
	err = fileutil.CopyFile(filepath.Join(origDir, "testdata", t.Name(), "mutagen.yml"), app.GetConfigPath(filepath.Join("mutagen", "mutagen.yml")))
	require.NoError(t, err)

	err = app.Start()
	require.NoError(t, err)

	thirdStartHash, err := ddevapp.GetMutagenConfigFileHashLabel(app)
	require.NoError(t, err)

	afterChangeSyncID, err := app.GetMutagenSyncID()
	require.NoError(t, err)
	require.NotEqual(t, origSyncID, afterChangeSyncID)

	require.NotEqual(t, firstStartHash, thirdStartHash)

	util.Debug("origSyncID=%s afterRestartSyncID=%s afterChangeSyncID=%s", origSyncID, afterRestartSyncID, afterChangeSyncID)
}

// TestMutagenDiagnose tests the MutagenDiagnose function
func TestMutagenDiagnose(t *testing.T) {
	if nodeps.IsWindows() {
		t.Skip("TestMutagenDiagnose skipped on Windows")
	}

	// Make sure there's not an existing Mutagen running
	_, _ = exec.RunHostCommand("pkill", "mutagen")

	// Use a simple test site
	site := TestSites[0]
	app := &ddevapp.DdevApp{Name: site.Name}
	_ = app.Stop(true, false)
	_ = globalconfig.RemoveProjectInfo(site.Name)

	err := site.Prepare()
	require.NoError(t, err)

	err = app.Init(site.Dir)
	require.NoError(t, err)
	app.SetPerformanceMode(types.PerformanceModeMutagen)
	err = app.WriteConfig()
	require.NoError(t, err)

	t.Cleanup(func() {
		err = app.Stop(true, false)
		require.NoError(t, err)
	})

	err = app.Start()
	require.NoError(t, err)

	// Run DiagnoseMutagenConfiguration
	result := ddevapp.DiagnoseMutagenConfiguration(app)
	require.NotNil(t, result)

	// Basic sanity checks on the diagnostic result
	require.True(t, result.SessionExists, "Mutagen session should exist")
	require.NotEmpty(t, result.SyncStatus, "Sync status should not be empty")
	require.GreaterOrEqual(t, result.VolumeSize, int64(0), "Volume size should be >= 0")
	require.NotEmpty(t, result.VolumeSizeHuman, "Volume size human should not be empty")

	// Check that volume was detected
	volumeName := ddevapp.GetMutagenVolumeName(app)
	require.True(t, dockerutil.VolumeExists(volumeName), "Mutagen volume should exist")

	// Verify issue counting logic:
	// When HasProblems is true, IssueCount should include those problems
	// When HasProblems is false, Problems slice should be empty
	if result.HasProblems {
		// If HasProblems is set, IssueCount should reflect at least 1 issue from sync problems
		require.GreaterOrEqual(t, result.IssueCount, 1, "IssueCount should be >= 1 when HasProblems is true")
	} else {
		// When there are no sync problems, Problems slice should be empty
		require.Empty(t, result.Problems, "Problems slice should be empty when HasProblems is false")
	}

	// For a healthy sync (status "ok"), HasProblems should be false
	if result.SyncStatus == "ok" {
		require.False(t, result.HasProblems, "HasProblems should be false when sync status is 'ok'")
	}
}
