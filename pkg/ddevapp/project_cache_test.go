package ddevapp_test

import (
	"os"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/stretchr/testify/require"
)

func TestProjectCacheVolumes(t *testing.T) {
	origDir, _ := os.Getwd()
	site := TestSites[0]

	testcommon.ClearDockerEnv()
	app, err := ddevapp.NewApp(site.Dir, true)
	require.NoError(t, err)
	webVolume := app.GetProjectCacheVolumeName("web")
	dbVolume := app.GetProjectCacheVolumeName("db")
	t.Cleanup(func() {
		_ = os.Chdir(origDir)
		require.NoError(t, app.Stop(true, false))
		require.False(t, dockerutil.VolumeExists(webVolume))
		require.False(t, dockerutil.VolumeExists(dbVolume))
	})

	err = app.Stop(false, false)
	require.NoError(t, err)
	if dockerutil.VolumeExists(webVolume) {
		require.NoError(t, dockerutil.RemoveVolume(webVolume))
	}

	require.NoError(t, app.Start())
	require.True(t, dockerutil.VolumeExists(webVolume))
	require.True(t, dockerutil.VolumeExists(dbVolume))

	out, _, err := app.Exec(&ddevapp.ExecOpts{Cmd: `echo "$COMPOSER_CACHE_DIR $npm_config_cache $MYSQL_HISTFILE" && test -d /mnt/ddev-project-cache/history`})
	require.NoError(t, err)
	require.Contains(t, out, "/mnt/ddev-project-cache/composer /mnt/ddev-project-cache/npm /mnt/ddev-project-cache/history/mysql_history")

	_, _, err = app.Exec(&ddevapp.ExecOpts{Service: "db", Cmd: `echo "db-history" >> "$HISTFILE"`})
	require.NoError(t, err)
	_, _, err = app.Exec(&ddevapp.ExecOpts{Service: "db", Cmd: "touch /mnt/ddev-global-cache/should-fail"})
	require.Error(t, err, "db should mount ddev-global-cache read-only")

	require.NoError(t, app.Restart())
	out, _, err = app.Exec(&ddevapp.ExecOpts{Service: "db", Cmd: `cat "$HISTFILE"`})
	require.NoError(t, err)
	require.Contains(t, out, "db-history")
}
