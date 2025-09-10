package dockerutil_test

import (
	"testing"

	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/stretchr/testify/require"
)

// TestDockerIP tries out a number of DockerHost permutations
// to verify that GetDockerIP does them right
func TestGetDockerIP(t *testing.T) {
	expectations := map[string]string{
		"":                            "127.0.0.1",
		"unix:///var/run/docker.sock": "127.0.0.1",
		"unix:///Users/rfay/.docker/run/docker.sock": "127.0.0.1",
		"unix:///Users/rfay/.colima/docker.sock":     "127.0.0.1",
		"tcp://185.199.110.153:2375":                 "185.199.110.153",
	}

	// Save original DockerHost to restore it later
	_, origDockerHost, err := dockerutil.GetDockerContextNameAndHost()
	require.NoError(t, err)
	t.Cleanup(func() {
		err = dockerutil.ResetDockerHost(origDockerHost)
		require.NoError(t, err)
	})

	for k, v := range expectations {
		// DockerIP is cached, so we have to reset it to check
		err = dockerutil.ResetDockerHost(k)
		require.NoError(t, err)
		result, err := dockerutil.GetDockerIP()
		require.NoError(t, err)
		require.Equal(t, v, result, "for %s expected %s, got %s", k, v, result)
	}
}
