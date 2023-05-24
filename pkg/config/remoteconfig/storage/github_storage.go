package storage

import (
	"context"
	"errors"
	"io"

	"github.com/ddev/ddev/pkg/config/remoteconfig/internal"
	"github.com/ddev/ddev/pkg/config/remoteconfig/types"
	"github.com/ddev/ddev/pkg/github"
	"muzzammil.xyz/jsonc"
)

func NewGithubStorage(owner, repo, filepath string, options Options) types.RemoteConfigStorage {
	return &githubStorage{
		owner:    owner,
		repo:     repo,
		filepath: filepath,
		options:  options,
	}
}

type Options = github.RepositoryContentGetOptions

type githubStorage struct {
	owner    string
	repo     string
	filepath string
	options  Options
}

func (s *githubStorage) Read() (remoteConfig internal.RemoteConfig, err error) {
	ctx := context.Background()
	client := github.GetGithubClient(ctx)

	var reader io.ReadCloser
	reader, _, err = client.Repositories.DownloadContents(ctx, s.owner, s.repo, s.filepath, &s.options)

	if err != nil {
		return
	}

	defer reader.Close()

	var b []byte
	b, err = io.ReadAll(reader)

	if err != nil {
		return
	}

	err = jsonc.Unmarshal(b, &remoteConfig)

	return
}

func (s *githubStorage) Write(_ internal.RemoteConfig) error {
	// do nothing, readonly storage
	return errors.New("failed to push remoteConfig to readonly `githubStorage`")
}
