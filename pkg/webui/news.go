package webui

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/util"
)

// newsURL is the DDEV blog's feed of headlines, without the posts' text,
// whose latest posts the start page lists.
var newsURL = "https://ddev.com/blog/rss-headlines.xml"

const (
	maxNewsPosts = 5
	// The feed changes a few times a week. After newsMaxAge the app asks
	// ddev.com whether it changed, which costs a 304 with no body when not.
	newsMaxAge = 6 * time.Hour
	newsRetry  = 10 * time.Minute
)

var (
	newsInternetActive = globalconfig.IsInternetActive
	// newsPath keeps the posts between runs, so a start downloads nothing
	// while they are fresh, and the posts still show offline.
	newsPath = func() string {
		return filepath.Join(globalconfig.GetGlobalDdevDir(), ".gui-news.json")
	}
	errNotModified = errors.New("not modified")
)

type newsPost struct {
	Title       string `json:"title"`
	Link        string `json:"link"`
	Date        string `json:"date"`
	Category    string `json:"category,omitempty"`
	Description string `json:"description,omitempty"`
}

type newsFile struct {
	Posts   []newsPost `json:"posts"`
	ETag    string     `json:"etag,omitempty"`
	Checked time.Time  `json:"checked"`
}

var newsCache struct {
	sync.Mutex
	newsFile
	loaded  bool
	retryAt time.Time
}

// parseNews returns the first posts of an RSS feed, with links to ddev.com
// only, so the page links nowhere else.
func parseNews(r io.Reader) ([]newsPost, error) {
	var feed struct {
		Items []struct {
			Title       string `xml:"title"`
			Link        string `xml:"link"`
			PubDate     string `xml:"pubDate"`
			Category    string `xml:"category"`
			Description string `xml:"description"`
		} `xml:"channel>item"`
	}
	if err := xml.NewDecoder(r).Decode(&feed); err != nil {
		return nil, err
	}
	posts := []newsPost{}
	for _, it := range feed.Items {
		link := strings.TrimSpace(it.Link)
		if it.Title == "" || !strings.HasPrefix(link, "https://ddev.com/") {
			continue
		}
		date := strings.TrimSpace(it.PubDate)
		if t, err := time.Parse(time.RFC1123, date); err == nil {
			date = t.Format("2006-01-02")
		}
		posts = append(posts, newsPost{Title: strings.TrimSpace(it.Title), Link: link, Date: date, Category: strings.TrimSpace(it.Category), Description: strings.TrimSpace(it.Description)})
		if len(posts) == maxNewsPosts {
			break
		}
	}
	return posts, nil
}

// fetchNews downloads the feed, or returns errNotModified when its ETag is
// still etag.
func fetchNews(etag string) ([]newsPost, string, error) {
	if !newsInternetActive() {
		return nil, "", errors.New("no internet")
	}
	req, err := http.NewRequest(http.MethodGet, newsURL, nil)
	if err != nil {
		return nil, "", err
	}
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}
	client := &http.Client{Timeout: 5 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusNotModified {
		return nil, etag, errNotModified
	}
	if res.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("%s: %s", newsURL, res.Status)
	}
	posts, err := parseNews(io.LimitReader(res.Body, 8<<20))
	return posts, res.Header.Get("ETag"), err
}

// handleNews serves the latest blog posts. When ddev.com can't be reached it
// serves the ones it has, or none, which the start page leaves out.
func (s *Server) handleNews(w http.ResponseWriter, _ *http.Request) {
	c := &newsCache
	c.Lock()
	defer c.Unlock()
	if !c.loaded {
		c.loaded = true
		if b, err := os.ReadFile(newsPath()); err == nil {
			_ = json.Unmarshal(b, &c.newsFile)
		}
	}
	now := time.Now()
	if now.Sub(c.Checked) > newsMaxAge && now.After(c.retryAt) {
		posts, etag, err := fetchNews(c.ETag)
		switch {
		case errors.Is(err, errNotModified):
			c.Checked = now
		case err != nil:
			util.Debug("Could not get the DDEV blog's posts: %v", err)
			c.retryAt = now.Add(newsRetry)
		default:
			c.Posts, c.ETag, c.Checked = posts, etag, now
		}
		if err == nil || errors.Is(err, errNotModified) {
			if b, err := json.Marshal(c.newsFile); err == nil {
				if err := os.WriteFile(newsPath(), b, 0o644); err != nil {
					util.Debug("Could not keep the DDEV blog's posts: %v", err)
				}
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string][]newsPost{"posts": append([]newsPost{}, c.Posts...)})
}
