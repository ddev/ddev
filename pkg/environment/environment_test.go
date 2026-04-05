package environment

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsWSL2Environment(t *testing.T) {
	t.Parallel()

	for _, envType := range []string{
		DDEVEnvironmentWSL2,
		DDEVEnvironmentWSL2Mirrored,
		DDEVEnvironmentWSL2VirtioProxy,
		DDEVEnvironmentWSL2None,
		DDEVEnvironmentWSL2Bridged,
	} {
		require.True(t, IsWSL2Environment(envType), envType)
	}

	for _, envType := range []string{
		DDEVEnvironmentLinux,
		DDEVEnvironmentDarwin,
		DDEVEnvironmentWindows,
		DDEVEnvironmentCodespaces,
		DDEVEnvironmentDevcontainer,
		"remote",
	} {
		require.False(t, IsWSL2Environment(envType), envType)
	}
}
