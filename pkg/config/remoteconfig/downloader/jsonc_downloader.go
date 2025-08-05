package downloader

import (
	"context"
	"io"
	"net/http"

	"muzzammil.xyz/jsonc"
)

// JSONCDownloader defines the interface for downloading and unmarshaling JSONC files
type JSONCDownloader interface {
	Download(ctx context.Context, target interface{}) error
	GetURL() string
}

// URLJSONCDownloader implements JSONCDownloader for direct URL downloads
type URLJSONCDownloader struct {
	URL string
}

// NewURLJSONCDownloader creates a new URL JSONC downloader
func NewURLJSONCDownloader(url string) JSONCDownloader {
	return &URLJSONCDownloader{
		URL: url,
	}
}

// Download downloads and unmarshals a JSONC file from a URL into the target interface
func (d *URLJSONCDownloader) Download(ctx context.Context, target interface{}) error {
	req, err := http.NewRequestWithContext(ctx, "GET", d.URL, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return jsonc.Unmarshal(b, target)
}

// GetURL returns the URL for this downloader
func (d *URLJSONCDownloader) GetURL() string {
	return d.URL
}
