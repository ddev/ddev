package github

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/google/go-github/v72/github"
)

// Aliases to avoid direct imports

type Client = github.Client
type ListOptions = github.ListOptions
type Repository = github.Repository
type SearchOptions = github.SearchOptions

var (
	// githubContext is the Go context used for GitHub API requests
	githubContext context.Context
	// githubClient is the singleton instance of Client
	githubClient *Client
	// githubClientOnce ensures githubClient is initialized only once
	githubClientOnce sync.Once
)

// GetGitHubClient returns a singleton GitHub client and context, initializing it if necessary.
func GetGitHubClient() (context.Context, *Client) {
	githubClientOnce.Do(func() {
		githubContext = context.Background()
		// Respect proxies set in the environment
		githubClient = github.NewClientWithEnvProxy()
		if githubToken := GetGitHubToken(); githubToken != "" {
			githubClient = githubClient.WithAuthToken(githubToken)
		}
	})
	return githubContext, githubClient
}

// GetGitHubRelease gets the tarball URL and version for a GitHub repository release
func GetGitHubRelease(owner, repo, requestedVersion string) (tarballURL, downloadedRelease string, err error) {
	ctx, client := GetGitHubClient()

	releases, resp, err := client.Repositories.ListReleases(ctx, owner, repo, &ListOptions{PerPage: 100})
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

// GetGitHubHeaders returns headers to be used in GitHub REST API requests if the URL is for GitHub.
// See https://docs.github.com/en/rest/authentication/authenticating-to-the-rest-api
func GetGitHubHeaders(requestURL string) map[string]string {
	headers := map[string]string{}
	if !isGitHubURL(requestURL) {
		return headers
	}
	githubToken := GetGitHubToken()
	if githubToken != "" {
		headers["Authorization"] = "Bearer " + githubToken
		// Use the same header as in vendor/github.com/google/go-github/v72/github/github.go
		headers["X-Github-Api-Version"] = "2022-11-28"
	}
	return headers
}

// isGitHubURL checks if the given URL is for GitHub or any subdomain of GitHub
func isGitHubURL(requestURL string) bool {
	if requestURL == "" {
		return false
	}
	u, err := url.Parse(requestURL)
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Host)
	return host == "github.com" || strings.HasSuffix(host, ".github.com")
}

// GetGitHubToken returns the GitHub access token from the environment variable.
func GetGitHubToken() string {
	for _, token := range []string{"DDEV_GITHUB_TOKEN", "GH_TOKEN", "GITHUB_TOKEN"} {
		if githubToken := os.Getenv(token); githubToken != "" {
			return githubToken
		}
	}
	return ""
}
