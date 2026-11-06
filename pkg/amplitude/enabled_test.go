package amplitude

import (
	"testing"

	"github.com/ddev/ddev/third_party/ampli"
	"github.com/stretchr/testify/require"
)

func TestIsEnabled(t *testing.T) {
	origClient, origDisabled := ampli.Instance.Client, ampli.Instance.Disabled
	t.Cleanup(func() {
		ampli.Instance.Client, ampli.Instance.Disabled = origClient, origDisabled
	})

	ampli.Instance.Client = nil
	ampli.Instance.Disabled = false
	require.False(t, IsEnabled())

	ampli.Instance.Disabled = true
	require.False(t, IsEnabled())
}
