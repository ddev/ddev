package ddevapp_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/moby/moby/api/types/mount"
	"github.com/stretchr/testify/require"
)

// TestWebCAROOTCopied checks that web gets the host's root CA without
// mounting ddev-global-cache, and signs its certificate with it.
func TestWebCAROOTCopied(t *testing.T) {
	caRoot := globalconfig.GetCAROOT()
	if caRoot == "" {
		t.Skip("mkcert is not set up on the host")
	}
	hostCA, err := os.ReadFile(filepath.Join(caRoot, "rootCA.pem"))
	require.NoError(t, err)

	origDir, _ := os.Getwd()
	testcommon.ClearDockerEnv()
	app, err := ddevapp.NewApp(TestSites[0].Dir, true)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.Chdir(origDir)
		require.NoError(t, app.Stop(true, false))
	})
	require.NoError(t, app.Stop(true, false))
	require.NoError(t, app.Start())

	for _, service := range []string{"web", "db"} {
		c, err := dockerutil.InspectContainer(ddevapp.GetContainerName(app, service))
		require.NoError(t, err)
		for _, m := range c.Mounts {
			require.False(t, m.Type == mount.TypeVolume && m.Name == "ddev-global-cache", "%s should not mount ddev-global-cache", service)
		}
	}

	out, _, err := app.Exec(&ddevapp.ExecOpts{Cmd: `cat "$CAROOT/rootCA.pem"`})
	require.NoError(t, err)
	require.Equal(t, strings.TrimSpace(string(hostCA)), strings.TrimSpace(out))

	uid, _, _ := dockerutil.GetContainerUser()
	out, _, err = app.Exec(&ddevapp.ExecOpts{Cmd: `stat -c "%u %a" "$CAROOT/rootCA-key.pem"`})
	require.NoError(t, err)
	require.Equal(t, uid+" 600", strings.TrimSpace(out))

	out, _, err = app.Exec(&ddevapp.ExecOpts{Cmd: `openssl verify -CAfile "$CAROOT/rootCA.pem" /etc/ssl/certs/master.crt`})
	require.NoError(t, err, "web should sign its certificate with the host's root CA, output=%s", out)

	// Restarting only web recreates its container without a full start
	require.NoError(t, app.RestartServices([]string{"web"}, ddevapp.StartOptions{}))
	out, _, err = app.Exec(&ddevapp.ExecOpts{Cmd: `openssl verify -CAfile "$CAROOT/rootCA.pem" /etc/ssl/certs/master.crt`})
	require.NoError(t, err, "recreated web should sign its certificate with the host's root CA, output=%s", out)
}
