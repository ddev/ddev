package webui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParseNews(t *testing.T) {
	feed := `<?xml version="1.0"?>
<rss version="2.0" xmlns:content="http://purl.org/rss/1.0/modules/content/"><channel><title>DDEV Blog</title>
<item><title>DDEV v1.25.6</title><link>https://ddev.com/blog/release-v1-25-6</link><pubDate>Fri, 20 Nov 2026 00:00:00 GMT</pubDate><category>Announcements</category><description> PHP 8.6.0. </description><content:encoded><![CDATA[<p>long</p>]]></content:encoded></item>
<item><title>Elsewhere</title><link>https://example.com/post</link></item>
<item><title>Second</title><link> https://ddev.com/blog/second </link><pubDate>not a date</pubDate></item>
</channel></rss>`
	posts, err := parseNews(strings.NewReader(feed))
	require.NoError(t, err)
	require.Equal(t, []newsPost{
		{Title: "DDEV v1.25.6", Link: "https://ddev.com/blog/release-v1-25-6", Date: "2026-11-20", Category: "Announcements", Description: "PHP 8.6.0."},
		{Title: "Second", Link: "https://ddev.com/blog/second", Date: "not a date"},
	}, posts, "only links to ddev.com, trimmed")

	_, err = parseNews(strings.NewReader("<rss><channel><item>"))
	require.Error(t, err)
}

// TestNewsCache checks that the posts are downloaded once, then only
// revalidated with their ETag, kept across restarts, and served offline.
func TestNewsCache(t *testing.T) {
	const feed = `<rss><channel><item><title>Post</title><link>https://ddev.com/blog/post</link></item></channel></rss>`
	var hits, revalidations int
	up := true
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if !up {
			http.Error(w, "down", http.StatusBadGateway)
			return
		}
		if r.Header.Get("If-None-Match") == `W/"v1"` {
			revalidations++
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("ETag", `W/"v1"`)
		_, _ = w.Write([]byte(feed))
	}))
	defer srv.Close()

	path := filepath.Join(t.TempDir(), ".gui-news.json")
	origURL, origPath, origActive := newsURL, newsPath, newsInternetActive
	t.Cleanup(func() {
		newsURL, newsPath, newsInternetActive = origURL, origPath, origActive
		newsCache.newsFile, newsCache.loaded, newsCache.retryAt = newsFile{}, false, time.Time{}
	})
	newsURL, newsPath, newsInternetActive = srv.URL, func() string { return path }, func() bool { return true }
	restart := func() { newsCache.newsFile, newsCache.loaded, newsCache.retryAt = newsFile{}, false, time.Time{} }
	get := func() []newsPost {
		w := httptest.NewRecorder()
		(&Server{}).handleNews(w, httptest.NewRequest(http.MethodGet, "/api/news", nil))
		var out struct{ Posts []newsPost }
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
		return out.Posts
	}
	want := []newsPost{{Title: "Post", Link: "https://ddev.com/blog/post"}}

	restart()
	require.Equal(t, want, get())
	require.Equal(t, want, get())
	require.Equal(t, 1, hits, "fresh posts are not fetched again")

	restart()
	require.Equal(t, want, get())
	require.Equal(t, 1, hits, "a restart reads the posts from disk")

	newsCache.Checked = time.Now().Add(-newsMaxAge - time.Minute)
	require.Equal(t, want, get())
	require.Equal(t, 2, hits)
	require.Equal(t, 1, revalidations, "old posts are revalidated with their ETag")
	require.Equal(t, want, get())
	require.Equal(t, 2, hits, "a 304 makes them fresh again")

	up = false
	newsCache.Checked = time.Now().Add(-newsMaxAge - time.Minute)
	require.Equal(t, want, get(), "the posts it has are served when ddev.com fails")
	require.Equal(t, want, get())
	require.Equal(t, 3, hits, "a failure waits before it asks again")
}
