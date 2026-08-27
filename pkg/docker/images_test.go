package docker

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveImageTag(t *testing.T) {
	require.Equal(t, "v1.25.4", resolveImageTag("c202e92108", "v1.25.4"))
	require.Equal(t, "c202e92108", resolveImageTag("c202e92108", "20260814_rfay_docker_update_phase_2"))
	require.Equal(t, "c202e92108", resolveImageTag("c202e92108", ""))
	require.Equal(t, "abc123", resolveImageTag("abc123", "v1.25.4-15-gabcdef1"))
}
