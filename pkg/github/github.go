package github

import (
	"context"
	"fmt"
	"os"

	"github.com/google/go-github/v72/github"
	"golang.org/x/oauth2"
)

type RepositoryContentGetOptions = github.RepositoryContentGetOptions

// GetGithubClient creates the required GitHub client
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

// GetGitHubRelease gets the tarball URL and version for a GitHub repository release
func GetGitHubRelease(owner, repo, requestedVersion string) (tarballURL, downloadedRelease string, err error) {
	ctx := context.Background()
	client := GetGithubClient(ctx)

	releases, resp, err := client.Repositories.ListReleases(ctx, owner, repo, &github.ListOptions{PerPage: 100})
	if err != nil {
		var rate github.Rate
		if resp != nil {
			rate = resp.Rate
		}
		return "", "", fmt.Errorf("unable to get releases for %v: %v\nresp.Rate=%v", repo, err, rate)
	}
	if len(releases) == 0 {
		return "", "", fmt.Errorf("no releases found for %v", repo)
	}

	releaseItem := 0
	releaseFound := false
	if requestedVersion != "" {
		for i, release := range releases {
			if release.GetTagName() == requestedVersion {
				releaseItem = i
				releaseFound = true
				break
			}
		}
		if !releaseFound {
			return "", "", fmt.Errorf("no release found for %v with tag %v", repo, requestedVersion)
		}
	}

	tarballURL = releases[releaseItem].GetTarballURL()
	downloadedRelease = releases[releaseItem].GetTagName()
	return tarballURL, downloadedRelease, nil
}
