package webui

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ddev/ddev/pkg/globalconfig"
)

// newsURL is the DDEV blog's feed, whose latest posts the start page lists.
var newsURL = "https://ddev.com/blog/rss.xml"

const maxNewsPosts = 5

type newsPost struct {
	Title    string `json:"title"`
	Link     string `json:"link"`
	Date     string `json:"date"`
	Category string `json:"category,omitempty"`
}

// newsCache keeps the posts for hours: the feed changes a few times a week.
var newsCache struct {
	sync.Mutex
	posts   []newsPost
	expires time.Time
}

// parseNews returns the first posts of an RSS feed, with links to ddev.com
// only, so the page links nowhere else.
func parseNews(r io.Reader) ([]newsPost, error) {
	var feed struct {
		Items []struct {
			Title    string `xml:"title"`
			Link     string `xml:"link"`
			PubDate  string `xml:"pubDate"`
			Category string `xml:"category"`
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
		posts = append(posts, newsPost{Title: strings.TrimSpace(it.Title), Link: link, Date: date, Category: strings.TrimSpace(it.Category)})
		if len(posts) == maxNewsPosts {
			break
		}
	}
	return posts, nil
}

func fetchNews() ([]newsPost, error) {
	if !globalconfig.IsInternetActive() {
		return nil, errors.New("no internet")
	}
	client := &http.Client{Timeout: 5 * time.Second}
	res, err := client.Get(newsURL)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: %s", newsURL, res.Status)
	}
	// The feed holds every post's full text.
	return parseNews(io.LimitReader(res.Body, 8<<20))
}

// handleNews serves the latest blog posts, or none when ddev.com can't be
// reached, which the start page leaves out.
func (s *Server) handleNews(w http.ResponseWriter, _ *http.Request) {
	newsCache.Lock()
	defer newsCache.Unlock()
	if time.Now().After(newsCache.expires) {
		posts, err := fetchNews()
		if err != nil {
			newsCache.expires = time.Now().Add(10 * time.Minute)
		} else {
			newsCache.posts, newsCache.expires = posts, time.Now().Add(6*time.Hour)
		}
	}
	writeJSON(w, http.StatusOK, map[string][]newsPost{"posts": append([]newsPost{}, newsCache.posts...)})
}
