package github

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/go-github/v81/github"
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
	// githubClientNoAuth is the singleton instance of Client without authentication
	githubClientNoAuth *Client
	// githubClientOnce ensures githubClient is initialized only once
	githubClientOnce sync.Once
)

// GetGitHubClient returns a singleton GitHub client and context, initializing it if necessary.
func GetGitHubClient(withAuth bool) (context.Context, *Client) {
	githubClientOnce.Do(func() {
		githubContext = context.Background()
		// Respect proxies set in the environment
		githubClientNoAuth = github.NewClientWithEnvProxy()
		githubClient = githubClientNoAuth
		githubToken, _ := GetGitHubToken()
		if githubToken != "" {
			githubClient = githubClient.WithAuthToken(githubToken)
		}
	})
	if withAuth {
		return githubContext, githubClient
	}
	return githubContext, githubClientNoAuth
}

// GetGitHubRelease gets the tarball URL and version for a GitHub repository release
func GetGitHubRelease(owner, repo, requestedVersion string) (tarballURL, downloadedRelease string, err error) {
	ctx, client := GetGitHubClient(true)

	releases, resp, err := client.Repositories.ListReleases(ctx, owner, repo, &ListOptions{PerPage: 100})
	var tokenErr error
	if err != nil {
		if tokenErr = HasInvalidGitHubToken(resp); tokenErr != nil {
			ctx, client = GetGitHubClient(false)
			releasesNoAuth, respNoAuth, errNoAuth := client.Repositories.ListReleases(ctx, owner, repo, &ListOptions{PerPage: 100})
			if errNoAuth == nil {
				releases = releasesNoAuth
				resp = respNoAuth
				err = errNoAuth
			}
		}
	}
	if err != nil {
		errorDetail := ""
		if resp != nil {
			rate := resp.Rate
			if rate.Limit != 0 {
				resetIn := time.Until(rate.Reset.Time).Round(time.Second)
				errorDetail += fmt.Sprintf("\nGitHub API Rate Limit: %d/%d remaining (resets in %s)", rate.Remaining, rate.Limit, resetIn)
			}
			if tokenErr != nil {
				errorDetail += "\nError: " + tokenErr.Error()
			}
		}
		return "", "", fmt.Errorf("unable to get releases for %v: %w%s", repo, err, errorDetail)
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
	if githubToken, _ := GetGitHubToken(); githubToken != "" {
		headers["Authorization"] = "Bearer " + githubToken
		// Use the same header as in vendor/github.com/google/go-github/v81/github/github.go
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

// GetGitHubToken returns the GitHub token from the environment and the name of the variable it was found in.
func GetGitHubToken() (string, string) {
	for _, token := range []string{"DDEV_GITHUB_TOKEN", "GH_TOKEN", "GITHUB_TOKEN"} {
		if githubToken := os.Getenv(token); githubToken != "" {
			return githubToken, token
		}
	}
	return "", ""
}

// HasGitHubToken returns true if a GitHub token is set in the environment.
func HasGitHubToken() bool {
	token, _ := GetGitHubToken()
	return token != ""
}

// HasInvalidGitHubToken checks if the response indicates an invalid GitHub token.
// This is possible if we get 401 or 404 response.
func HasInvalidGitHubToken(response interface{}) error {
	var httpResp *http.Response
	switch r := response.(type) {
	case *github.Response:
		if r == nil {
			return nil
		}
		httpResp = r.Response
	case *http.Response:
		httpResp = r
	default:
		return nil
	}
	if httpResp == nil || httpResp.Request == nil {
		return nil
	}
	if !isGitHubURL(httpResp.Request.URL.String()) {
		return nil
	}
	_, tokenVariable := GetGitHubToken()
	if tokenVariable == "" {
		return nil
	}
	if httpResp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("using %s, which is invalid or lacks required permissions", tokenVariable)
	}
	if httpResp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("using %s, which may be invalid or lack permissions", tokenVariable)
	}
	return nil
}
