package github

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestParseArtifactURLFromHTML verifies that the nightly.link download URL is
// extracted from the pr-artifacts-comment bot's rendered link (and its raw
// Markdown form), and that the last match wins.
func TestParseArtifactURLFromHTML(t *testing.T) {
	const linuxURL = "https://nightly.link/ddev/ddev/actions/artifacts/123456789.zip"
	const macosURL = "https://nightly.link/ddev/ddev/actions/artifacts/999.zip"

	// Rendered anchor form (what GitHub serves for the bot's Markdown link).
	anchorHTML := `<ul>
<li><a href="` + linuxURL + `" rel="nofollow">ddev-linux-amd64.zip</a></li>
<li><a href="` + macosURL + `" rel="nofollow">ddev-macos-arm64.zip</a></li>
</ul>`
	require.Equal(t, linuxURL, parseArtifactURLFromHTML(anchorHTML, "ddev-linux-amd64"))
	require.Equal(t, macosURL, parseArtifactURLFromHTML(anchorHTML, "ddev-macos-arm64"))

	// Raw Markdown form (as embedded in the page's hydration payload).
	mdHTML := `[ddev-windows-amd64.zip](https://nightly.link/ddev/ddev/actions/artifacts/555.zip)`
	require.Equal(t, "https://nightly.link/ddev/ddev/actions/artifacts/555.zip",
		parseArtifactURLFromHTML(mdHTML, "ddev-windows-amd64"))

	// A missing artifact yields an empty string.
	require.Empty(t, parseArtifactURLFromHTML(anchorHTML, "ddev-windows-arm64"))

	// The bot edits one comment in place, so the last match wins.
	dup := anchorHTML + `<a href="https://nightly.link/ddev/ddev/actions/artifacts/222.zip" rel="nofollow">ddev-linux-amd64.zip</a>`
	require.Equal(t, "https://nightly.link/ddev/ddev/actions/artifacts/222.zip",
		parseArtifactURLFromHTML(dup, "ddev-linux-amd64"))
}
