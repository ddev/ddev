package github

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"

	"github.com/google/go-github/v90/github"
)

// WorkflowRunFilter selects a GitHub Actions workflow run by branch and/or head SHA.
type WorkflowRunFilter struct {
	Branch  string
	HeadSHA string
}

// ErrNoSuccessfulRun is returned by FindWorkflowArtifact when no successful run
// matches the filter.
var ErrNoSuccessfulRun = errors.New("no successful run found")

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

// GetCommitSHA returns the full SHA that ref, such as an abbreviated SHA,
// points to.
func GetCommitSHA(owner, repo, ref string) (string, error) {
	sha, err := withAuthFallback(func(ctx context.Context, client *Client) (string, *github.Response, error) {
		return client.Repositories.GetCommitSHA1(ctx, owner, repo, ref, "")
	})
	if err != nil {
		return "", fmt.Errorf("unable to look up commit %s in %s/%s: %w", ref, owner, repo, err)
	}
	return sha, nil
}

// FindWorkflowArtifact finds the newest successful run of workflowFile matching
// filter and returns the ID of its artifact named artifactName. Listing runs and
// artifacts works anonymously for public repositories, subject to GitHub's
// unauthenticated rate limit.
func FindWorkflowArtifact(owner, repo, workflowFile, artifactName string, filter WorkflowRunFilter) (int64, error) {
	run, err := findSuccessfulRun(filter.Branch, func(branch string) ([]*github.WorkflowRun, error) {
		opts := &github.ListWorkflowRunsOptions{
			Branch:  branch,
			HeadSHA: filter.HeadSHA,
			PerPage: 30,
		}
		runs, err := withAuthFallback(func(ctx context.Context, client *Client) (*github.WorkflowRuns, *github.Response, error) {
			return client.Actions.ListWorkflowRunsByFileName(ctx, owner, repo, workflowFile, opts)
		})
		return runs.GetWorkflowRuns(), err
	})
	if err != nil {
		return 0, err
	}
	if run == nil {
		return 0, fmt.Errorf("%w for %s", ErrNoSuccessfulRun, workflowFile)
	}
	runID := run.GetID()

	artifacts, err := withAuthFallback(func(ctx context.Context, client *Client) (*github.ArtifactList, *github.Response, error) {
		return client.Actions.ListWorkflowRunArtifacts(ctx, owner, repo, runID, &github.ListOptions{PerPage: 100})
	})
	if err != nil {
		return 0, err
	}

	var found *github.Artifact
	for _, a := range artifacts.Artifacts {
		if a.GetName() == artifactName {
			found = a
			break
		}
	}
	if found == nil {
		return 0, fmt.Errorf("workflow run %d has no artifact named %q", runID, artifactName)
	}
	if found.GetExpired() {
		return 0, fmt.Errorf("artifact %q (run %d) has expired; GitHub keeps run artifacts for ~90 days, so choose a newer run", artifactName, runID)
	}
	return found.GetID(), nil
}

// ArtifactDownloadURL returns the short-lived (~1 minute) presigned URL for an
// artifact's .zip. GitHub requires a token for this even on public repositories,
// so unlike the lookups it never retries anonymously.
func ArtifactDownloadURL(owner, repo string, artifactID int64) (string, error) {
	ctx, client, err := GetGitHubClient(true)
	if err != nil {
		return "", err
	}
	u, resp, err := client.Actions.DownloadArtifact(ctx, owner, repo, artifactID, 5)
	if err != nil {
		if tokenErr := HasInvalidGitHubToken(resp); tokenErr != nil {
			return "", tokenErr
		}
		return "", err
	}
	if u == nil {
		return "", fmt.Errorf("GitHub returned no download URL for artifact %d", artifactID)
	}
	return u.String(), nil
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

// findSuccessfulRun returns the newest successful run on branch, or on any
// branch if branch is empty. GitHub's branch filter sometimes returns an
// out-of-date run list, so it checks the latest unfiltered runs first and uses
// the filter only for a branch not among them.
func findSuccessfulRun(branch string, listRuns func(branch string) ([]*github.WorkflowRun, error)) (*github.WorkflowRun, error) {
	if branch != "" {
		runs, err := listRuns("")
		if err != nil {
			return nil, err
		}
		if run := newestSuccessfulRun(runs, branch); run != nil {
			return run, nil
		}
	}
	runs, err := listRuns(branch)
	if err != nil {
		return nil, err
	}
	return newestSuccessfulRun(runs, branch), nil
}

// newestSuccessfulRun returns the most recently created run whose conclusion is
// "success" and, if branch is not empty, whose head branch is branch, or nil if
// there is none.
func newestSuccessfulRun(runs []*github.WorkflowRun, branch string) *github.WorkflowRun {
	var best *github.WorkflowRun
	for _, r := range runs {
		if r.GetConclusion() != "success" || (branch != "" && r.GetHeadBranch() != branch) {
			continue
		}
		if best == nil || r.GetCreatedAt().After(best.GetCreatedAt().Time) {
			best = r
		}
	}
	return best
}
