package ddevapp_test

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/ddev/ddev/pkg/util"
	"github.com/stretchr/testify/require"
)

// TestApacheRunsAsRoot verifies which user the apache2 process runs as with
// webserver-type=apache-fpm. Under Docker rootless with bind mounts the web
// container runs as 0:0 and the ddev-keep-root shim keeps apache2 at UID 0
// (Apache otherwise refuses to run as root). In every other case the shim is
// removed from the image (see WriteBuildDockerfile) and apache2 runs as the
// container user.
func TestApacheRunsAsRoot(t *testing.T) {
	origDir, _ := os.Getwd()

	testDir := testcommon.CreateTmpDir(t.Name())
	projName := t.Name()

	t.Cleanup(func() {
		_ = os.Chdir(origDir)
		app, err := ddevapp.GetActiveApp(projName)
		if err == nil {
			_ = app.Stop(true, false)
		}
		_ = os.RemoveAll(testDir)
		testcommon.ClearDockerEnv()
	})

	// Clean up any existing name conflicts
	app, err := ddevapp.GetActiveApp(projName)
	if err == nil {
		err = app.Stop(true, false)
		require.NoError(t, err)
	}

	app, err = ddevapp.NewApp(testDir, false)
	require.NoError(t, err)
	app.Type = nodeps.AppTypePHP
	app.Name = projName
	app.WebserverType = nodeps.WebserverApacheFPM
	err = app.WriteConfig()
	require.NoError(t, err)

	defer util.TimeTrackC(fmt.Sprintf("%s %s", projName, t.Name()))()

	// If Apache could not run as root this Restart would fail, because the web
	// container would never become healthy.
	err = app.Restart()
	require.NoError(t, err)

	// Apache runs as root only on Docker rootless with bind mounts (where the
	// web container runs as 0:0); otherwise it runs as the container user.
	uid, _, _ := dockerutil.GetContainerUser()
	expectedUID := uid
	keepRoot := dockerutil.IsDockerRootless() && !globalconfig.DdevGlobalConfig.NoBindMounts
	if keepRoot {
		expectedUID = "0"
	}

	// List the UID of every apache2 process. Empty output means apache2 is not
	// running (for example because it refused to run as root).
	out, _, err := app.Exec(&ddevapp.ExecOpts{
		Service: "web",
		Cmd:     "ps -o uid= -C apache2",
	})
	require.NoError(t, err)
	uids := strings.Fields(out)
	require.NotEmpty(t, uids, "no apache2 processes found; apache2 may have failed to start")
	for _, u := range uids {
		require.Equal(t, expectedUID, u, "apache2 running as unexpected uid; ps output: %q", out)
	}

	// The ddev-keep-root shim must exist in the image only when it is needed.
	out, _, err = app.Exec(&ddevapp.ExecOpts{
		Service: "web",
		Cmd:     "test -f /usr/local/lib/ddev/ddev-keep-root.so && echo present || echo absent",
	})
	require.NoError(t, err)
	if keepRoot {
		require.Equal(t, "present", strings.TrimSpace(out))
	} else {
		require.Equal(t, "absent", strings.TrimSpace(out))
	}
}
