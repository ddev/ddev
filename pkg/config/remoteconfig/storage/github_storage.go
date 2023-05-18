package storage

import (
	"context"
	"errors"
	"io"
	"time"

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

func (s *githubStorage) LastUpdate() time.Time {
	return time.Now()
}

func (s *githubStorage) Pull() (manifest internal.RemoteConfig, err error) {
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

	err = jsonc.Unmarshal(b, &manifest)

	return
}

func (s *githubStorage) Push(_ internal.RemoteConfig) error {
	// do nothing, readonly storage
	return errors.New("failed to push manifest to readonly `githubStorage`")
}
