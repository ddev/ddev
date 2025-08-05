package downloader

import (
	"context"
	"io"
	"net/http"

	"github.com/ddev/ddev/pkg/github"
	"muzzammil.xyz/jsonc"
)

// JSONCDownloader defines the interface for downloading and unmarshaling JSONC files
type JSONCDownloader interface {
	Download(ctx context.Context, target interface{}) error
}

// GitHubJSONCDownloader implements JSONCDownloader for GitHub repositories
type GitHubJSONCDownloader struct {
	Owner    string
	Repo     string
	Filepath string
	Options  github.RepositoryContentGetOptions
}

// NewGitHubJSONCDownloader creates a new GitHub JSONC downloader
func NewGitHubJSONCDownloader(owner, repo, filepath string, options github.RepositoryContentGetOptions) JSONCDownloader {
	return &GitHubJSONCDownloader{
		Owner:    owner,
		Repo:     repo,
		Filepath: filepath,
		Options:  options,
	}
}

// Download downloads and unmarshals a JSONC file from GitHub into the target interface
func (d *GitHubJSONCDownloader) Download(ctx context.Context, target interface{}) error {
	client := github.GetGithubClient(ctx)

	reader, _, err := client.Repositories.DownloadContents(ctx, d.Owner, d.Repo, d.Filepath, &d.Options)
	if err != nil {
		return err
	}
	defer reader.Close()

	b, err := io.ReadAll(reader)
	if err != nil {
		return err
	}

	return jsonc.Unmarshal(b, target)
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
