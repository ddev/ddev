package storages

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"time"

	"github.com/ddev/ddev/pkg/github"
	"github.com/ddev/ddev/pkg/manifest/types"
	"github.com/ddev/ddev/pkg/util"
)

func NewGithubStorage(owner, repo, filepath string, options Options) types.ManifestStorage {
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

func (s *githubStorage) Pull() (manifest types.Manifest, err error) {
	ctx := context.Background()
	client := github.GetGithubClient(ctx)

	//fileContent, _, _, err := client.Repositories.GetContents(ctx, s.owner, s.repo, s.filepath, &s.options)
	var reader io.ReadCloser
	reader, _, err = client.Repositories.DownloadContents(ctx, s.owner, s.repo, s.filepath, &s.options)

	if err != nil {
		return
	}

	defer reader.Close()

	var b []byte
	b, err = io.ReadAll(reader)

	util.Debug("%s", b)

	if err != nil {
		return
	}

	err = json.Unmarshal(b, &manifest)
	util.Debug("%v", manifest)

	return
}

func (s *githubStorage) Push(_ types.Manifest) error {
	// do nothing, readonly storage
	return errors.New("failed to push manifest to readonly `githubStorage`")
}
