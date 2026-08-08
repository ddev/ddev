package github

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"time"

	"github.com/google/go-github/v88/github"
)

// WorkflowRunFilter selects a GitHub Actions workflow run by branch and/or head SHA.
type WorkflowRunFilter struct {
	Branch  string
	HeadSHA string
	Event   string
}

// GetLatestReleaseTag returns the tag name of the latest release for owner/repo.
// It uses GitHub's "latest release" endpoint, which excludes drafts and prereleases.
func GetLatestReleaseTag(owner, repo string) (string, error) {
	release, err := withAuthFallback(func(ctx context.Context, client *Client) (*github.RepositoryRelease, *github.Response, error) {
		return client.Repositories.GetLatestRelease(ctx, owner, repo)
	})
	if err != nil {
		return "", fmt.Errorf("unable to get latest release for %s/%s: %w", owner, repo, err)
	}
	return release.GetTagName(), nil
}

// GetPullRequestHeadSHA returns the head commit SHA of a pull request.
func GetPullRequestHeadSHA(owner, repo string, number int) (string, error) {
	pr, err := withAuthFallback(func(ctx context.Context, client *Client) (*github.PullRequest, *github.Response, error) {
		return client.PullRequests.Get(ctx, owner, repo, number)
	})
	if err != nil {
		return "", fmt.Errorf("unable to look up PR #%d in %s/%s: %w", number, owner, repo, err)
	}
	sha := pr.GetHead().GetSHA()
	if sha == "" {
		return "", fmt.Errorf("could not determine head SHA for PR #%d", number)
	}
	return sha, nil
}

// ResolveWorkflowArtifactURL finds the newest successful run of workflowFile
// matching filter and returns a URL from which the artifact named artifactName
// can be downloaded as a .zip.
//
// Listing runs and artifacts works anonymously for public repositories (subject
// to GitHub's unauthenticated rate limit). The artifact zip itself cannot be
// downloaded anonymously from the GitHub API, so when a token is available this
// returns the authenticated Actions download URL; otherwise it returns a
// nightly.link URL (a third-party redirector that works only for public repos).
func ResolveWorkflowArtifactURL(owner, repo, workflowFile, artifactName string, filter WorkflowRunFilter) (string, error) {
	opts := &github.ListWorkflowRunsOptions{
		Branch:      filter.Branch,
		HeadSHA:     filter.HeadSHA,
		Event:       filter.Event,
		ListOptions: github.ListOptions{PerPage: 30},
	}
	runs, err := withAuthFallback(func(ctx context.Context, client *Client) (*github.WorkflowRuns, *github.Response, error) {
		return client.Actions.ListWorkflowRunsByFileName(ctx, owner, repo, workflowFile, opts)
	})
	if err != nil {
		return "", err
	}
	run := newestSuccessfulRun(runs.WorkflowRuns)
	if run == nil {
		return "", fmt.Errorf("no successful %s run found", workflowFile)
	}
	runID := run.GetID()

	artifacts, err := withAuthFallback(func(ctx context.Context, client *Client) (*github.ArtifactList, *github.Response, error) {
		return client.Actions.ListWorkflowRunArtifacts(ctx, owner, repo, runID, &github.ListOptions{PerPage: 100})
	})
	if err != nil {
		return "", err
	}

	var found *github.Artifact
	for _, a := range artifacts.Artifacts {
		if a.GetName() == artifactName {
			found = a
			break
		}
	}
	if found == nil {
		return "", fmt.Errorf("workflow run %d has no artifact named %q", runID, artifactName)
	}
	if found.GetExpired() {
		return "", fmt.Errorf("artifact %q (run %d) has expired; GitHub keeps run artifacts for ~90 days, so choose a newer run", artifactName, runID)
	}
	artifactID := found.GetID()

	// With a token, use the authenticated Actions API, which works for private
	// repos too and returns a short-lived (~1 minute) presigned URL.
	if HasGitHubToken() {
		u, downloadErr := withAuthFallback(func(ctx context.Context, client *Client) (*url.URL, *github.Response, error) {
			return client.Actions.DownloadArtifact(ctx, owner, repo, artifactID, 5)
		})
		if downloadErr == nil && u != nil {
			return u.String(), nil
		}
	}

	// No token (or the API download failed): fall back to nightly.link.
	return NightlyLinkArtifactURL(owner, repo, artifactID), nil
}

// NightlyLinkArtifactURL returns the nightly.link URL for a specific artifact ID.
// nightly.link is a third-party service that serves GitHub Actions artifacts for
// public repositories without requiring authentication.
func NightlyLinkArtifactURL(owner, repo string, artifactID int64) string {
	return fmt.Sprintf("https://nightly.link/%s/%s/actions/artifacts/%d.zip", owner, repo, artifactID)
}

// PullRequestArtifactURL returns a nightly.link download URL for artifactName by
// reading the pull request's HTML page and extracting the link posted there by
// the pr-artifacts-comment workflow. It uses no GitHub API token or API-rate
// budget, so it is a useful fallback when the API is rate-limited, but it works
// only for public repositories and only once that bot comment exists (that is,
// after the PR's build has finished).
func PullRequestArtifactURL(owner, repo string, number int, artifactName string) (string, error) {
	pageURL := fmt.Sprintf("https://github.com/%s/%s/pull/%d", owner, repo, number)
	req, err := http.NewRequest(http.MethodGet, pageURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "ddev")
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("unable to fetch %s: %w", pageURL, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("no pull request #%d in %s/%s", number, owner, repo)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %q fetching %s", resp.Status, pageURL)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 20<<20))
	if err != nil {
		return "", err
	}
	u := parseArtifactURLFromHTML(string(body), artifactName)
	if u == "" {
		return "", fmt.Errorf("no %s.zip download link found on %s (the PR build may not have finished)", artifactName, pageURL)
	}
	return u, nil
}

// parseArtifactURLFromHTML extracts the nightly.link download URL for
// artifactName from a rendered pull request page. The pr-artifacts-comment
// workflow posts each artifact as a Markdown link, which GitHub serves as
// <a href="https://nightly.link/.../artifacts/<id>.zip">artifactName.zip</a>.
// The last match wins, since the bot edits a single comment in place.
func parseArtifactURLFromHTML(html, artifactName string) string {
	name := regexp.QuoteMeta(artifactName + ".zip")
	for _, pattern := range []string{
		`href="(https://nightly\.link/[^"]+/actions/artifacts/\d+\.zip)"[^>]*>\s*` + name,
		`\[` + name + `\]\((https://nightly\.link/[^)]+/actions/artifacts/\d+\.zip)\)`,
	} {
		matches := regexp.MustCompile(pattern).FindAllStringSubmatch(html, -1)
		if len(matches) > 0 {
			return matches[len(matches)-1][1]
		}
	}
	return ""
}

// newestSuccessfulRun returns the most recently created run whose conclusion is
// "success", or nil if there is none.
func newestSuccessfulRun(runs []*github.WorkflowRun) *github.WorkflowRun {
	var best *github.WorkflowRun
	for _, r := range runs {
		if r.GetConclusion() != "success" {
			continue
		}
		if best == nil || r.GetCreatedAt().After(best.GetCreatedAt().Time) {
			best = r
		}
	}
	return best
}
