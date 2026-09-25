package github

import (
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
