package ddevapp_test

import (
	"os"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/ddev/ddev/pkg/util"
	"github.com/ddev/ddev/pkg/versionconstants"
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

	// Seed history where older DDEV versions kept it, to check it's copied into a new volume
	err = app.Stop(false, false)
	require.NoError(t, err)
	if dockerutil.VolumeExists(webVolume) {
		require.NoError(t, dockerutil.RemoveVolume(webVolume))
	}
	seedCmd := `mkdir -p /mnt/ddev-global-cache/bashhistory/` + app.Name + `-web && echo "seeded-history" > /mnt/ddev-global-cache/bashhistory/` + app.Name + `-web/bash_history`
	_, out, err := dockerutil.RunSimpleContainer(versionconstants.UtilitiesImage, "seed-history-"+util.RandString(6), []string{"sh", "-c", seedCmd}, nil, nil, []string{"ddev-global-cache:/mnt/ddev-global-cache"}, "0", true, false, nil, nil, nil)
	require.NoError(t, err, "output=%s", out)

	require.NoError(t, app.Start())
	require.True(t, dockerutil.VolumeExists(webVolume))
	require.True(t, dockerutil.VolumeExists(dbVolume))

	out, _, err = app.Exec(&ddevapp.ExecOpts{Cmd: `echo "$COMPOSER_CACHE_DIR $npm_config_cache $MYSQL_HISTFILE" && cat /mnt/ddev-project-cache/history/bash_history`})
	require.NoError(t, err)
	require.Contains(t, out, "/mnt/ddev-project-cache/composer /mnt/ddev-project-cache/npm /mnt/ddev-project-cache/history/mysql_history")
	require.Contains(t, out, "seeded-history")

	_, _, err = app.Exec(&ddevapp.ExecOpts{Service: "db", Cmd: `echo "db-history" >> "$HISTFILE"`})
	require.NoError(t, err)
	_, _, err = app.Exec(&ddevapp.ExecOpts{Service: "db", Cmd: "grep -q ' /mnt/ddev-global-cache ' /proc/mounts"})
	require.Error(t, err, "db shouldn't mount ddev-global-cache")
	_, _, err = app.Exec(&ddevapp.ExecOpts{Service: "db", Cmd: "test -x /mnt/ddev-project-cache/global-commands/mysql"})
	require.NoError(t, err, "db's global commands should be copied into its project cache")

	require.NoError(t, app.Restart())
	out, _, err = app.Exec(&ddevapp.ExecOpts{Service: "db", Cmd: `cat "$HISTFILE"`})
	require.NoError(t, err)
	require.Contains(t, out, "db-history")
}
