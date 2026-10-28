package ddevapp_test

import (
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/stretchr/testify/require"
)

func TestPopulateGlobalCustomCommandFiles(t *testing.T) {
	origDir, _ := os.Getwd()
	site := TestSites[0]
	app, err := ddevapp.NewApp(site.Dir, false)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.Chdir(origDir)
		err = app.Stop(true, false)
		require.NoError(t, err)
	})
	_ = site.Chdir()
	err = app.Start()
	require.NoError(t, err)

	// Start copies the bundled global commands.
	_, _, err = app.Exec(&ddevapp.ExecOpts{Cmd: "test -x " + path.Join(ddevapp.GlobalCommandsContainerDir, "web", "xdebug")})
	require.NoError(t, err)

	xdgDir := testcommon.CreateTmpDir(t.Name())
	t.Cleanup(func() {
		_ = os.RemoveAll(xdgDir)
	})
	t.Setenv("DDEV_XDG_CONFIG_HOME", xdgDir)
	commandsDir := filepath.Join(globalconfig.GetGlobalDdevDir(), "commands")
	// db is a symlink to a directory elsewhere, as when commands live in a
	// dotfiles repository.
	dbDir := filepath.Join(commandsDir, "db")
	if !nodeps.IsWindows() {
		dbDir = filepath.Join(xdgDir, "dotfiles-db")
		require.NoError(t, os.MkdirAll(commandsDir, 0755))
		require.NoError(t, os.Symlink(dbDir, filepath.Join(commandsDir, "db")))
	}
	files := map[string]string{
		filepath.Join(commandsDir, "web", "test-web-cmd"):     "#!/usr/bin/env bash\necho web\n",
		filepath.Join(commandsDir, "web", "test-removed-cmd"): "#!/usr/bin/env bash\necho removed\n",
		filepath.Join(dbDir, "test-db-cmd"):                   "#!/usr/bin/env bash\necho db\n",
		filepath.Join(commandsDir, "host", "test-host-cmd"):   "#!/usr/bin/env bash\necho host\n",
		filepath.Join(commandsDir, ".git", "HEAD"):            "ref: refs/heads/main\n",
	}
	for p, content := range files {
		require.NoError(t, os.MkdirAll(filepath.Dir(p), 0755))
		require.NoError(t, os.WriteFile(p, []byte(content), 0644))
	}
	if !nodeps.IsWindows() {
		target := filepath.Join(xdgDir, "linked-target")
		require.NoError(t, os.WriteFile(target, []byte("#!/usr/bin/env bash\necho linked\n"), 0644))
		require.NoError(t, os.Symlink(target, filepath.Join(commandsDir, "web", "test-linked-cmd")))
		require.NoError(t, os.Symlink("..", filepath.Join(commandsDir, "web", "loop")))
	}

	services, err := ddevapp.PopulateGlobalCustomCommandFiles(app)
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"web", "db"}, services)

	out, _, err := app.Exec(&ddevapp.ExecOpts{Cmd: path.Join(ddevapp.GlobalCommandsContainerDir, "web", "test-web-cmd")})
	require.NoError(t, err)
	require.Equal(t, "web\n", out)
	out, _, err = app.Exec(&ddevapp.ExecOpts{Service: "db", Cmd: path.Join(ddevapp.GlobalCommandsContainerDir, "db", "test-db-cmd")})
	require.NoError(t, err)
	require.Equal(t, "db\n", out)
	for _, dir := range []string{"host", ".git"} {
		_, _, err = app.Exec(&ddevapp.ExecOpts{Cmd: "test ! -e " + path.Join(ddevapp.GlobalCommandsContainerDir, dir)})
		require.NoError(t, err, "%s must not be copied into containers", dir)
	}
	info, err := os.Stat(filepath.Join(commandsDir, "web", "test-web-cmd"))
	require.NoError(t, err)
	out, _, err = app.Exec(&ddevapp.ExecOpts{Cmd: "stat -c %Y " + path.Join(ddevapp.GlobalCommandsContainerDir, "web", "test-web-cmd")})
	require.NoError(t, err)
	require.Equal(t, strconv.FormatInt(info.ModTime().Unix(), 10), strings.TrimSpace(out))
	if !nodeps.IsWindows() {
		out, _, err = app.Exec(&ddevapp.ExecOpts{Cmd: path.Join(ddevapp.GlobalCommandsContainerDir, "web", "test-linked-cmd")})
		require.NoError(t, err)
		require.Equal(t, "linked\n", out)
	}

	require.NoError(t, os.Remove(filepath.Join(commandsDir, "web", "test-removed-cmd")))
	_, err = ddevapp.PopulateGlobalCustomCommandFiles(app)
	require.NoError(t, err)
	_, _, err = app.Exec(&ddevapp.ExecOpts{Cmd: "test ! -e " + path.Join(ddevapp.GlobalCommandsContainerDir, "web", "test-removed-cmd")})
	require.NoError(t, err, "a removed command must be removed from the container")

	err = app.RecreateServices([]string{"db"})
	require.NoError(t, err)
	out, _, err = app.Exec(&ddevapp.ExecOpts{Service: "db", Cmd: path.Join(ddevapp.GlobalCommandsContainerDir, "db", "test-db-cmd")})
	require.NoError(t, err)
	require.Equal(t, "db\n", out)
}
