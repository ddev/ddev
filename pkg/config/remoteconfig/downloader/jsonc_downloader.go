package downloader

import (
	"context"
	"io"

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
