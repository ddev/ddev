package cmd

import (
	"testing"

	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/stretchr/testify/require"
)

// TestUtilityDockercheckCmd tests basic functionality of ddev utility dockercheck
func TestUtilityDockercheckCmd(t *testing.T) {
	// Basic execution test
	out, err := exec.RunHostCommand(DdevBin, "utility", "dockercheck")
	require.NoError(t, err)
	require.Contains(t, out, "Docker platform:")
	require.Contains(t, out, "Using Docker context:")
	require.Contains(t, out, "Using Docker host:")
	// Check that TLS configuration is shown (either configured or not)
	require.Regexp(t, "TLS (not configured|enabled)", out)
	require.Contains(t, out, "docker-compose:")
	// Check for buildx (Docker) or buildx skip message (Podman)
	require.Regexp(t, "(docker buildx version|buildx check skipped)", out)
	require.Contains(t, out, "Docker version:")
	require.Contains(t, out, "Docker API version:")
	// Check for build test success on Docker (skipped on Podman)
	if !dockerutil.IsPodman() {
		require.Contains(t, out, "docker buildx is working correctly")
	}
	require.Contains(t, out, "Docker authentication is configured correctly")
}
