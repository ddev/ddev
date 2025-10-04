package cmd

import (
	"testing"

	"github.com/ddev/ddev/pkg/exec"
	"github.com/stretchr/testify/require"
)

// TestUtilityDockcheckCmd tests basic functionality of ddev utility dockercheck
func TestUtilityDockcheckCmd(t *testing.T) {
	// Basic execution test
	out, err := exec.RunHostCommand(DdevBin, "utility", "dockercheck")
	require.NoError(t, err)
	require.Contains(t, out, "Docker platform:")
	require.Contains(t, out, "docker buildx version")
	require.Contains(t, out, "Docker version:")
	require.Contains(t, out, "Docker authentication is configured correctly")
}
