package github

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/google/go-github/v90/github"
	"github.com/stretchr/testify/require"
)

// TestNewestRelease's UnsortedList order is how GitHub's list endpoint
// returned these releases, with the newest one not first.
func TestNewestRelease(t *testing.T) {
	release := func(tag string, created string, draft bool) *github.RepositoryRelease {
		createdAt, err := time.Parse(time.RFC3339, created)
		require.NoError(t, err)
		return &github.RepositoryRelease{
			TagName:   tag,
			CreatedAt: github.Timestamp{Time: createdAt},
			Draft:     draft,
		}
	}

	t.Run("UnsortedList", func(t *testing.T) {
		newest := newestRelease([]*github.RepositoryRelease{
			release("1.0.0-beta9", "2026-09-17T15:40:38Z", false),
			release("1.0.0-beta8", "2026-09-17T07:39:38Z", false),
			release("1.0.0-beta10", "2026-09-17T15:56:17Z", false),
			release("1.0.0-beta7", "2026-09-15T14:52:36Z", false),
		})
		require.NotNil(t, newest)
		require.Equal(t, "1.0.0-beta10", newest.GetTagName())
	})

	t.Run("SkipsDrafts", func(t *testing.T) {
		newest := newestRelease([]*github.RepositoryRelease{
			release("1.0.0-beta9", "2026-09-17T15:40:38Z", false),
			release("1.0.0-beta11", "2026-09-18T10:00:00Z", true),
		})
		require.NotNil(t, newest)
		require.Equal(t, "1.0.0-beta9", newest.GetTagName())
	})

	t.Run("OnlyDrafts", func(t *testing.T) {
		require.Nil(t, newestRelease([]*github.RepositoryRelease{
			release("1.0.0-beta11", "2026-09-18T10:00:00Z", true),
		}))
	})

	t.Run("Empty", func(t *testing.T) {
		require.Nil(t, newestRelease(nil))
	})
}

// TestWithAuthFallbackRetry: a 404 for a missing tag must not spend the
// anonymous rate limit, which shared CI runner IPs often exhaust.
func TestWithAuthFallbackRetry(t *testing.T) {
	// Initialize the client singleton from the real environment first, so the
	// fake token below cannot leak into the other tests in this package.
	_, _, err := GetGitHubClient(true)
	require.NoError(t, err)
	t.Setenv("DDEV_GITHUB_TOKEN", "fake-token")

	req, err := http.NewRequest(http.MethodGet, "https://api.github.com/repos/ddev/ddev-redis/releases/tags/v999.999.999", nil)
	require.NoError(t, err)
	callsFor := func(status int) (int, error) {
		calls := 0
		_, err := withAuthFallback(func(_ context.Context, _ *Client) (string, *github.Response, error) {
			calls++
			resp := &github.Response{Response: &http.Response{StatusCode: status, Request: req}}
			return "", resp, &github.ErrorResponse{Response: resp.Response}
		})
		return calls, err
	}

	calls, err := callsFor(http.StatusNotFound)
	require.Equal(t, 1, calls, "a 404 should not retry anonymously")
	require.True(t, isNotFound(err), "should keep the 404, got %v", err)
	require.ErrorContains(t, err, "may be invalid or lack permissions")

	calls, err = callsFor(http.StatusUnauthorized)
	require.Equal(t, 2, calls, "a 401 should retry anonymously")
	require.ErrorContains(t, err, "is invalid or lacks required permissions")
}
