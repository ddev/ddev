package cmd

import (
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/config/remoteconfig/types"
	"github.com/stretchr/testify/require"
)

// sampleAddons returns a small slice of Addons for use in rendering tests.
func sampleAddons() []types.Addon {
	return []types.Addon{
		{
			Title:       "ddev-redis",
			GitHubURL:   "https://github.com/ddev/ddev-redis",
			Description: "Redis integration for DDEV",
			Type:        "official",
		},
		{
			Title:       "ddev-memcached",
			GitHubURL:   "https://github.com/ddev/ddev-memcached",
			Description: "Memcached integration for DDEV",
			Type:        "contrib",
		},
	}
}

// TestRenderRepositoryListContent verifies that addon titles and descriptions
// appear in the rendered output, that official addons get a "*" suffix on their
// description, and that the footer contains the correct count.
func TestRenderRepositoryListContent(t *testing.T) {
	out := renderRepositoryList(sampleAddons(), false)

	require.Contains(t, out, "ddev-redis")
	require.Contains(t, out, "ddev-memcached")
	// Official addons get "*" appended to description
	require.Contains(t, out, "Redis integration for DDEV*")
	// Non-official addons do not
	require.NotContains(t, out, "Memcached integration for DDEV*")
	// Footer shows count and legend
	require.Contains(t, out, "2 add-ons found")
	require.Contains(t, out, "officially maintained by DDEV")
}

// TestRenderRepositoryListSorted verifies addons are rendered in alphabetical order.
func TestRenderRepositoryListSorted(t *testing.T) {
	// Supply addons in reverse alphabetical order; output must be sorted.
	addons := []types.Addon{
		{Title: "zzz-addon", Description: "Last"},
		{Title: "aaa-addon", Description: "First"},
	}
	out := renderRepositoryList(addons, false)

	posFirst := strings.Index(out, "aaa-addon")
	posLast := strings.Index(out, "zzz-addon")
	require.Greater(t, posFirst, -1, "aaa-addon should appear in output")
	require.Greater(t, posLast, -1, "zzz-addon should appear in output")
	require.Less(t, posFirst, posLast, "aaa-addon should appear before zzz-addon")
}

// TestRenderRepositoryListSnipsLongTitle verifies that a title exceeding the
// calculated column width is truncated with "…" rather than overflowing.
// In the test process GetTerminalWidthHeight returns 0, so the code falls back
// to 80 columns → addonWidth = max(30, (80-7)*3/10) = 30.
func TestRenderRepositoryListSnipsLongTitle(t *testing.T) {
	longTitle := strings.Repeat("x", 50)
	addons := []types.Addon{
		{Title: longTitle, Description: "Short description"},
	}
	out := renderRepositoryList(addons, false)

	require.NotContains(t, out, longTitle, "full long title should not appear untruncated")
	require.Contains(t, out, "…", "truncated title should end with ellipsis")
}

// TestRenderRepositoryListWrapTable verifies that --wrap-table shows the full title
// without truncation.
func TestRenderRepositoryListWrapTable(t *testing.T) {
	longTitle := strings.Repeat("x", 50)
	addons := []types.Addon{
		{Title: longTitle, Description: "Short description"},
	}
	out := renderRepositoryList(addons, true)

	require.Contains(t, out, longTitle, "full title should appear when wrap-table is set")
	require.NotContains(t, out, "…")
}

// TestRenderRepositoryListEmpty verifies the footer still renders for a single addon.
func TestRenderRepositoryListEmpty(t *testing.T) {
	out := renderRepositoryList([]types.Addon{
		{Title: "only-addon", Description: "Sole entry"},
	}, false)
	require.Contains(t, out, "1 add-ons found")
}

// TestRenderSearchResultsContent verifies that matching addon titles, descriptions,
// and the search term appear in the rendered search output.
func TestRenderSearchResultsContent(t *testing.T) {
	out := renderSearchResults(sampleAddons(), "redis", false)

	require.Contains(t, out, "ddev-redis")
	require.Contains(t, out, "ddev-memcached")
	require.Contains(t, out, "Redis integration for DDEV*")
	require.NotContains(t, out, "Memcached integration for DDEV*")
	// Footer includes search term and count
	require.Contains(t, out, "2 add-ons found matching 'redis'")
	require.Contains(t, out, "officially maintained by DDEV")
}

// TestRenderSearchResultsSorted verifies search results are rendered in alphabetical order.
func TestRenderSearchResultsSorted(t *testing.T) {
	addons := []types.Addon{
		{Title: "zzz-addon", Description: "Last"},
		{Title: "aaa-addon", Description: "First"},
	}
	out := renderSearchResults(addons, "addon", false)

	posFirst := strings.Index(out, "aaa-addon")
	posLast := strings.Index(out, "zzz-addon")
	require.Greater(t, posFirst, -1)
	require.Greater(t, posLast, -1)
	require.Less(t, posFirst, posLast)
}

// TestRenderSearchResultsSnipsLongTitle mirrors the addon-list snip test for
// the search renderer, which uses the same column-width logic.
func TestRenderSearchResultsSnipsLongTitle(t *testing.T) {
	longTitle := strings.Repeat("y", 50)
	addons := []types.Addon{
		{Title: longTitle, Description: "Short description"},
	}
	out := renderSearchResults(addons, "y", false)

	require.NotContains(t, out, longTitle)
	require.Contains(t, out, "…")
}

// TestRenderSearchResultsWrapTable verifies that --wrap-table shows the full title
// without truncation.
func TestRenderSearchResultsWrapTable(t *testing.T) {
	longTitle := strings.Repeat("y", 50)
	addons := []types.Addon{
		{Title: longTitle, Description: "Short description"},
	}
	out := renderSearchResults(addons, "y", true)

	require.Contains(t, out, longTitle, "full title should appear when wrap-table is set")
	require.NotContains(t, out, "…")
}
