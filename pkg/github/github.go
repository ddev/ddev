package github

import (
	"context"
	"os"

	"github.com/google/go-github/v52/github"
	"golang.org/x/oauth2"
)

type RepositoryContentGetOptions = github.RepositoryContentGetOptions

// GetGithubClient creates the required github client
func GetGithubClient(ctx context.Context) github.Client {
	// Use authenticated client for higher rate limit, normally only needed for tests
	githubToken := os.Getenv("DDEV_GITHUB_TOKEN")
	if githubToken != "" {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: githubToken},
		)
		tc := oauth2.NewClient(ctx, ts)
		return *github.NewClient(tc)
	}

	return *github.NewClient(nil)
}
