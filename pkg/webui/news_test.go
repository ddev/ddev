package webui

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseNews(t *testing.T) {
	feed := `<?xml version="1.0"?>
<rss version="2.0" xmlns:content="http://purl.org/rss/1.0/modules/content/"><channel><title>DDEV Blog</title>
<item><title>DDEV v1.25.6</title><link>https://ddev.com/blog/release-v1-25-6</link><pubDate>Fri, 20 Nov 2026 00:00:00 GMT</pubDate><category>Announcements</category><content:encoded><![CDATA[<p>long</p>]]></content:encoded></item>
<item><title>Elsewhere</title><link>https://example.com/post</link></item>
<item><title>Second</title><link> https://ddev.com/blog/second </link><pubDate>not a date</pubDate></item>
</channel></rss>`
	posts, err := parseNews(strings.NewReader(feed))
	require.NoError(t, err)
	require.Equal(t, []newsPost{
		{Title: "DDEV v1.25.6", Link: "https://ddev.com/blog/release-v1-25-6", Date: "2026-11-20", Category: "Announcements"},
		{Title: "Second", Link: "https://ddev.com/blog/second", Date: "not a date"},
	}, posts, "only links to ddev.com, trimmed")

	_, err = parseNews(strings.NewReader("<rss><channel><item>"))
	require.Error(t, err)
}
