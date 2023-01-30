package storages

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/ddev/ddev/pkg/github"
	"github.com/ddev/ddev/pkg/messages/types"
)

func NewGithubStorage(owner string, repo string, filepath string) types.MessagesStorage {
	return &githubStorage{
		owner:    owner,
		repo:     repo,
		filepath: filepath,
	}
}

type githubStorage struct {
	owner    string
	repo     string
	filepath string
}

func (s *githubStorage) LastUpdate() time.Time {
	return time.Now()
}

func (s *githubStorage) Pull() (messages types.Messages, err error) {
	client := github.GetGithubClient(context.Background())
	reader, _, err := client.Repositories.DownloadContents(context.Background(), s.owner, s.repo, s.filepath, &github.RepositoryContentGetOptions{})

	if err != nil {
		return
	}

	dec := json.NewDecoder(reader)
	err = dec.Decode(&messages)

	return
}

func (s *githubStorage) Push(_ *types.Messages) error {
	// do nothing, readonly storage
	return errors.New("failed to push messages to readonly `githubStorage`")
}
