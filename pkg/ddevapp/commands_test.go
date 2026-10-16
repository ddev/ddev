package ddevapp_test

import (
	"os"
	"path"
	"path/filepath"
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
	files := map[string]string{
		"web/test-web-cmd":   "#!/usr/bin/env bash\necho web\n",
		"db/test-db-cmd":     "#!/usr/bin/env bash\necho db\n",
		"host/test-host-cmd": "#!/usr/bin/env bash\necho host\n",
	}
	for name, content := range files {
		p := filepath.Join(commandsDir, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(p), 0755))
		require.NoError(t, os.WriteFile(p, []byte(content), 0644))
	}
	if !nodeps.IsWindows() {
		target := filepath.Join(xdgDir, "linked-target")
		require.NoError(t, os.WriteFile(target, []byte("#!/usr/bin/env bash\necho linked\n"), 0644))
		require.NoError(t, os.Symlink(target, filepath.Join(commandsDir, "web", "test-linked-cmd")))
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
	_, _, err = app.Exec(&ddevapp.ExecOpts{Cmd: "test ! -e " + path.Join(ddevapp.GlobalCommandsContainerDir, "host")})
	require.NoError(t, err, "host commands must not be copied into containers")
	if !nodeps.IsWindows() {
		out, _, err = app.Exec(&ddevapp.ExecOpts{Cmd: path.Join(ddevapp.GlobalCommandsContainerDir, "web", "test-linked-cmd")})
		require.NoError(t, err)
		require.Equal(t, "linked\n", out)
	}
}
