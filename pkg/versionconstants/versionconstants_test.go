package versionconstants

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestIsUnreleasedDdevVersion verifies detection of DDEV versions built from
// an untagged commit, as opposed to a clean release tag.
func TestIsUnreleasedDdevVersion(t *testing.T) {
	unreleased := []string{
		"v1.25.3-111-geadf098e9", // git describe, commits past a tag
		"v1.25.3-dirty",          // exact tag, but uncommitted changes
		"v1.25.3-111-geadf098e9-dirty",
		"geadf098", // debug.BuildInfo short-hash fallback
		"geadf098-dirty",
	}
	for _, v := range unreleased {
		require.True(t, IsUnreleasedDdevVersion(v), "expected %q to be detected as unreleased", v)
	}

	released := []string{
		"v1.25.3",
		"v1.25.3-beta1",
		"v0.0.0-overridden-by-make",
	}
	for _, v := range released {
		require.False(t, IsUnreleasedDdevVersion(v), "expected %q to be detected as a release", v)
	}
}
