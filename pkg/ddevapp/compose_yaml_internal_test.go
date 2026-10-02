package ddevapp

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHasUnspecifiedHostPort(t *testing.T) {
	require.True(t, hasUnspecifiedHostPort("0"))
	require.True(t, hasUnspecifiedHostPort(""))
	require.False(t, hasUnspecifiedHostPort("12345"))
}
