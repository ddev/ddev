package ddevapp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ddev/ddev/pkg/globalconfig"
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

// TestSSHAgentUpstreamSocketPaths checks the values that need no Docker provider.
func TestSSHAgentUpstreamSocketPaths(t *testing.T) {
	origUpstream := globalconfig.DdevGlobalConfig.SSHAgentUpstream
	t.Cleanup(func() {
		globalconfig.DdevGlobalConfig.SSHAgentUpstream = origUpstream
	})
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	for value, expected := range map[string]string{
		"":                     "",
		"/path/to/agent.sock":  "/path/to/agent.sock",
		"~/.ssh/agent.sock":    filepath.Join(home, ".ssh", "agent.sock"),
		"~/tmp/agent/ssh.sock": filepath.Join(home, "tmp", "agent", "ssh.sock"),
	} {
		globalconfig.DdevGlobalConfig.SSHAgentUpstream = value
		sock, err := SSHAgentUpstreamSocket()
		require.NoError(t, err, value)
		require.Equal(t, expected, sock, value)
	}

	for _, value := range []string{"relative/agent.sock", "agent.sock", "./agent.sock"} {
		globalconfig.DdevGlobalConfig.SSHAgentUpstream = value
		_, err := SSHAgentUpstreamSocket()
		require.Error(t, err, value)
	}
}
