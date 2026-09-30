package ddevapp

import (
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/stretchr/testify/require"
)

// TestSSHAgentUpstreamMount checks which bind mount each upstream socket gets.
func TestSSHAgentUpstreamMount(t *testing.T) {
	mount, name := sshAgentUpstreamMount(hostServicesSSHAuthSock)
	require.Equal(t, "/run/host-services/ssh-auth.sock:/upstream/agent.sock", mount)
	require.Equal(t, "agent.sock", name)

	mount, name = sshAgentUpstreamMount("/run/user/1000/gcr/ssh")
	require.Equal(t, "/run/user/1000/gcr:/upstream", mount)
	require.Equal(t, "ssh", name)

	mount, name = sshAgentUpstreamMount("/tmp/ssh-abc123/agent.4567")
	require.Equal(t, "/tmp/ssh-abc123:/upstream", mount)
	require.Equal(t, "agent.4567", name)
}

// TestCheckSSHAgentUpstreamListening checks the warning for a missing or dead agent socket.
func TestCheckSSHAgentUpstreamListening(t *testing.T) {
	origUpstream := globalconfig.DdevGlobalConfig.SSHAgentUpstream
	t.Cleanup(func() {
		globalconfig.DdevGlobalConfig.SSHAgentUpstream = origUpstream
	})
	// macOS limits socket paths to 104 bytes, which t.TempDir() can exceed.
	home, err := os.UserHomeDir()
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Join(home, "tmp"), 0755))
	dir, err := os.MkdirTemp(filepath.Join(home, "tmp"), "sock")
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.RemoveAll(dir)
	})
	sock := filepath.Join(dir, "agent.sock")
	globalconfig.DdevGlobalConfig.SSHAgentUpstream = sock

	require.Error(t, checkSSHAgentUpstreamListening(sock))

	listener, err := net.Listen("unix", sock)
	require.NoError(t, err)
	require.NoError(t, checkSSHAgentUpstreamListening(sock))
	require.NoError(t, listener.Close())
	require.Error(t, checkSSHAgentUpstreamListening(sock))
}

// TestSSHAgentUpstreamSocketPaths checks the values that need no Docker provider.
func TestSSHAgentUpstreamSocketPaths(t *testing.T) {
	origUpstream := globalconfig.DdevGlobalConfig.SSHAgentUpstream
	t.Cleanup(func() {
		globalconfig.DdevGlobalConfig.SSHAgentUpstream = origUpstream
	})
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	if nodeps.IsWindows() {
		for _, value := range []string{"host", "/path/to/agent.sock", "~/.ssh/agent.sock", `C:\Users\me\agent.sock`} {
			globalconfig.DdevGlobalConfig.SSHAgentUpstream = value
			_, err := SSHAgentUpstreamSocket()
			require.ErrorContains(t, err, "doesn't work on Windows", value)
		}
		return
	}

	require.NoError(t, os.MkdirAll(filepath.Join(home, "tmp"), 0755))
	dir, err := os.MkdirTemp(filepath.Join(home, "tmp"), "sock")
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.RemoveAll(dir)
	})
	for value, expected := range map[string]string{
		"":                               "",
		filepath.Join(dir, "agent.sock"): filepath.Join(dir, "agent.sock"),
		"~/tmp/" + filepath.Base(dir) + "/ssh.sock": filepath.Join(dir, "ssh.sock"),
	} {
		globalconfig.DdevGlobalConfig.SSHAgentUpstream = value
		sock, err := SSHAgentUpstreamSocket()
		require.NoError(t, err, value)
		require.Equal(t, expected, sock, value)
	}

	globalconfig.DdevGlobalConfig.SSHAgentUpstream = filepath.Join(dir, "missing", "agent.sock")
	_, err = SSHAgentUpstreamSocket()
	require.ErrorContains(t, err, "doesn't exist")

	for _, value := range []string{"relative/agent.sock", "agent.sock", "./agent.sock"} {
		globalconfig.DdevGlobalConfig.SSHAgentUpstream = value
		_, err := SSHAgentUpstreamSocket()
		require.Error(t, err, value)
	}
}
