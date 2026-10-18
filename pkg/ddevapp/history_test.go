package ddevapp_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/stretchr/testify/require"
)

// TestHistoryBindMount checks that history lands in .ddev/.history and that
// anything root writes there is removable by the host user after a restart.
func TestHistoryBindMount(t *testing.T) {
	if globalconfig.DdevGlobalConfig.NoBindMounts {
		t.Skip("History isn't kept with no_bind_mounts")
	}
	origDir, _ := os.Getwd()
	site := TestSites[0]

	testcommon.ClearDockerEnv()
	app, err := ddevapp.NewApp(site.Dir, true)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.Chdir(origDir)
		require.NoError(t, app.Stop(true, false))
		require.NoError(t, os.RemoveAll(app.GetHistoryDir()))
	})
	require.NoError(t, app.Restart())

	historyDir := app.GetHistoryDir()
	require.FileExists(t, filepath.Join(historyDir, ".gitignore"))

	out, _, err := app.Exec(&ddevapp.ExecOpts{Cmd: `echo "history-test" >> "$HISTFILE" && echo "$HISTFILE $MYSQL_HISTFILE"`})
	require.NoError(t, err)
	require.Equal(t, "/mnt/ddev-history/bash_history /mnt/ddev-history/mysql_history", strings.TrimSpace(out))
	history, err := fileutil.ReadFileIntoString(filepath.Join(historyDir, "web", "bash_history"))
	require.NoError(t, err)
	require.Contains(t, history, "history-test")

	out, _, err = app.Exec(&ddevapp.ExecOpts{Service: "db", Cmd: `echo "$HISTFILE"`})
	require.NoError(t, err)
	require.Equal(t, "/mnt/ddev-history/bash_history", strings.TrimSpace(out))

	// A root-owned directory would leave its contents undeletable by the host user
	_, _, err = app.Exec(&ddevapp.ExecOpts{User: "root", Cmd: "mkdir -p /mnt/ddev-history/rootdir && touch /mnt/ddev-history/rootdir/f"})
	require.NoError(t, err)
	require.NoError(t, app.Restart())
	require.NoError(t, os.RemoveAll(filepath.Join(historyDir, "web", "rootdir")))
}
