package storage

import (
	"context"
	"errors"

	"github.com/ddev/ddev/pkg/config/remoteconfig/downloader"
	"github.com/ddev/ddev/pkg/config/remoteconfig/types"
	"github.com/ddev/ddev/pkg/github"
)

func NewGithubStorage(owner, repo, filepath string, options Options) types.RemoteConfigStorage {
	return &githubStorage{
		downloader: downloader.NewGitHubJSONCDownloader(owner, repo, filepath, options),
	}
}

type Options = github.RepositoryContentGetOptions

type githubStorage struct {
	downloader downloader.JSONCDownloader
}

func (s *githubStorage) Read() (remoteConfig types.RemoteConfigData, err error) {
	ctx := context.Background()
	err = s.downloader.Download(ctx, &remoteConfig)
	return
}

func (s *githubStorage) Write(_ types.RemoteConfigData) error {
	// do nothing, readonly storage
	return errors.New("failed to push remoteConfig to readonly `githubStorage`")
}
