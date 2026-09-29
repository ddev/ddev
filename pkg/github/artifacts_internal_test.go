package github

import (
	"errors"
	"testing"
	"time"

	"github.com/google/go-github/v90/github"
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

func testRun(id int64, conclusion, branch string, age time.Duration) *github.WorkflowRun {
	return &github.WorkflowRun{
		ID:         new(id),
		Conclusion: new(conclusion),
		HeadBranch: new(branch),
		CreatedAt:  &github.Timestamp{Time: time.Now().Add(-age)},
	}
}

func TestNewestSuccessfulRun(t *testing.T) {
	runs := []*github.WorkflowRun{
		testRun(1, "success", "main", 3*time.Hour),
		testRun(2, "failure", "main", time.Hour),
		testRun(3, "success", "v1.25.3", 2*time.Hour),
		testRun(4, "success", "main", 2*time.Hour),
		testRun(5, "success", "feature", 0),
	}
	require.Equal(t, int64(4), newestSuccessfulRun(runs, "main").GetID())
	require.Equal(t, int64(5), newestSuccessfulRun(runs, "").GetID())
	require.Nil(t, newestSuccessfulRun(runs, "no-such-branch"))
	require.Nil(t, newestSuccessfulRun(nil, ""))
}

// TestFindSuccessfulRun's filtered lists are stale, as GitHub's branch filter
// can be, so findSuccessfulRun must use them only for a branch missing from the
// unfiltered list.
func TestFindSuccessfulRun(t *testing.T) {
	unfiltered := []*github.WorkflowRun{testRun(1, "success", "main", 0), testRun(2, "success", "feature", time.Hour)}
	filtered := map[string][]*github.WorkflowRun{
		"main":  {testRun(3, "success", "main", 30*24*time.Hour)},
		"quiet": {testRun(4, "success", "quiet", 30*24*time.Hour)},
	}
	var calls []string
	listRuns := func(branch string) ([]*github.WorkflowRun, error) {
		calls = append(calls, branch)
		if branch == "" {
			return unfiltered, nil
		}
		return filtered[branch], nil
	}
	find := func(branch string) (int64, []string) {
		calls = nil
		run, err := findSuccessfulRun(branch, listRuns)
		require.NoError(t, err)
		return run.GetID(), calls
	}

	id, got := find("main")
	require.Equal(t, int64(1), id)
	require.Equal(t, []string{""}, got)

	id, got = find("quiet")
	require.Equal(t, int64(4), id)
	require.Equal(t, []string{"", "quiet"}, got)

	id, got = find("")
	require.Equal(t, int64(1), id)
	require.Equal(t, []string{""}, got)

	id, got = find("no-such-branch")
	require.Zero(t, id)
	require.Equal(t, []string{"", "no-such-branch"}, got)

	_, err := findSuccessfulRun("main", func(string) ([]*github.WorkflowRun, error) { return nil, errors.New("rate limited") })
	require.ErrorContains(t, err, "rate limited")
}
