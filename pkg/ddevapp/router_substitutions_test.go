package ddevapp

import (
	"testing"

	"github.com/moby/moby/api/types/container"
	"github.com/stretchr/testify/require"
)

// TestParseFormatRouterPortSubstitutions checks the round-trip and
// malformed-input behavior of the RouterPortSubstitutionsLabel serialization.
func TestParseFormatRouterPortSubstitutions(t *testing.T) {
	require.Equal(t, map[string]string{}, parseRouterPortSubstitutions(""))
	require.Equal(t, map[string]string{"80": "33000"}, parseRouterPortSubstitutions("80=33000"))
	require.Equal(t, map[string]string{"80": "33000", "443": "33001"}, parseRouterPortSubstitutions("80=33000,443=33001"))
	// Malformed pairs are skipped rather than breaking the whole label
	require.Equal(t, map[string]string{"80": "33000"}, parseRouterPortSubstitutions("80=33000,junk,=1,2="))

	require.Equal(t, "", formatRouterPortSubstitutions(map[string]string{}))
	// Output is sorted for determinism
	require.Equal(t, "443=33001,80=33000", formatRouterPortSubstitutions(map[string]string{"80": "33000", "443": "33001"}))

	roundTrip := parseRouterPortSubstitutions(formatRouterPortSubstitutions(map[string]string{"80": "33000", "443": "33001", "8025": "33002"}))
	require.Equal(t, map[string]string{"80": "33000", "443": "33001", "8025": "33002"}, roundTrip)
}

// TestKnownRouterPortSubstitutions checks that the router label and the
// in-process map are merged, with the in-process map winning on conflict.
func TestKnownRouterPortSubstitutions(t *testing.T) {
	origSubstitutions := routerPortEphemeralSubstitutions
	t.Cleanup(func() {
		routerPortEphemeralSubstitutions = origSubstitutions
	})

	router := &container.Summary{
		Labels: map[string]string{
			RouterPortSubstitutionsLabel: "80=33000,443=33001",
		},
	}

	routerPortEphemeralSubstitutions = map[string]string{}
	require.Equal(t, map[string]string{"80": "33000", "443": "33001"}, knownRouterPortSubstitutions(router))

	// nil router yields only the in-process entries
	routerPortEphemeralSubstitutions = map[string]string{"8025": "33002"}
	require.Equal(t, map[string]string{"8025": "33002"}, knownRouterPortSubstitutions(nil))

	// In-process entries overlay the label and win on conflict
	routerPortEphemeralSubstitutions = map[string]string{"80": "33005", "8025": "33002"}
	require.Equal(t, map[string]string{"80": "33005", "443": "33001", "8025": "33002"}, knownRouterPortSubstitutions(router))

	// Router without the label contributes nothing
	routerPortEphemeralSubstitutions = map[string]string{}
	require.Equal(t, map[string]string{}, knownRouterPortSubstitutions(&container.Summary{Labels: map[string]string{}}))
}
