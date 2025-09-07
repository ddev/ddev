package github_test

import (
	"os"
	"testing"

	"github.com/ddev/ddev/pkg/github"
	"github.com/stretchr/testify/require"
)

// TestGetGitHubRelease tests the GetGitHubRelease function with various scenarios.
func TestGetGitHubRelease(t *testing.T) {
	if os.Getenv("DDEV_RUN_GET_TESTS") != "true" {
		t.Skip("Skipping because DDEV_RUN_GET_TESTS is not set")
	}

	// Test getting latest release
	t.Run("GetLatestRelease", func(t *testing.T) {
		tarballURL, version, err := github.GetGitHubRelease("ddev", "ddev-redis", "")
		require.NoError(t, err, "Should successfully get latest release")
		require.NotEmpty(t, tarballURL, "Tarball URL should not be empty")
		require.NotEmpty(t, version, "Version should not be empty")
		require.Contains(t, tarballURL, "github.com", "Tarball URL should be from GitHub")
	})

	// Test getting specific version
	t.Run("GetSpecificVersion", func(t *testing.T) {
		tarballURL, version, err := github.GetGitHubRelease("ddev", "ddev-redis", "v1.0.4")
		require.NoError(t, err, "Should successfully get specific version")
		require.Equal(t, "v1.0.4", version, "Should return requested version")
		require.NotEmpty(t, tarballURL, "Tarball URL should not be empty")
	})

	// Test non-existent repository
	t.Run("NonExistentRepo", func(t *testing.T) {
		_, _, err := github.GetGitHubRelease("ddev", "non-existent-repo", "")
		require.Error(t, err, "Should fail for non-existent repository")
		require.Contains(t, err.Error(), "unable to get releases", "Error should mention inability to get releases")
	})

	// Test non-existent version
	t.Run("NonExistentVersion", func(t *testing.T) {
		_, _, err := github.GetGitHubRelease("ddev", "ddev-redis", "v999.999.999")
		require.Error(t, err, "Should fail for non-existent version")
		require.Contains(t, err.Error(), "no release found", "Error should mention no release found")
	})

	// Test real addon with dependencies
	t.Run("RealAddonWithDependencies", func(t *testing.T) {
		// Test ddev-redis-commander which depends on ddev-redis
		tarballURL, version, err := github.GetGitHubRelease("ddev", "ddev-redis-commander", "")
		require.NoError(t, err, "Should successfully get ddev-redis-commander release")
		require.NotEmpty(t, tarballURL, "Tarball URL should not be empty")
		require.NotEmpty(t, version, "Version should not be empty")
		require.Contains(t, tarballURL, "github.com", "Tarball URL should be from GitHub")
	})
}

// TestGetGitHubToken tests the GetGitHubToken function environment variable precedence.
func TestGetGitHubToken(t *testing.T) {
	// Test precedence: DDEV_GITHUB_TOKEN > GH_TOKEN > GITHUB_TOKEN
	t.Run("Precedence", func(t *testing.T) {
		// Clean environment
		t.Setenv("DDEV_GITHUB_TOKEN", "")
		t.Setenv("GH_TOKEN", "")
		t.Setenv("GITHUB_TOKEN", "")

		// Test no tokens returns empty
		token := github.GetGitHubToken()
		require.Empty(t, token, "Should return empty when no tokens set")

		// Test GITHUB_TOKEN only
		t.Setenv("GITHUB_TOKEN", "github_token_value")
		token = github.GetGitHubToken()
		require.Equal(t, "github_token_value", token, "Should return GITHUB_TOKEN when it's the only one set")

		// Test GH_TOKEN overrides GITHUB_TOKEN
		t.Setenv("GH_TOKEN", "gh_token_value")
		token = github.GetGitHubToken()
		require.Equal(t, "gh_token_value", token, "Should return GH_TOKEN when both GH_TOKEN and GITHUB_TOKEN are set")

		// Test DDEV_GITHUB_TOKEN has the highest precedence
		t.Setenv("DDEV_GITHUB_TOKEN", "ddev_token_value")
		token = github.GetGitHubToken()
		require.Equal(t, "ddev_token_value", token, "Should return DDEV_GITHUB_TOKEN when all tokens are set")
	})
}

// TestGetGitHubHeaders tests the GetGitHubHeaders function URL filtering.
func TestGetGitHubHeaders(t *testing.T) {
	t.Run("GitHubURLs", func(t *testing.T) {
		t.Setenv("DDEV_GITHUB_TOKEN", "test_token")

		// Test https://github.com URLs
		headers := github.GetGitHubHeaders("https://github.com/owner/repo")
		require.Equal(t, "Bearer test_token", headers["Authorization"], "Should include auth header for https://github.com")
		require.Equal(t, "2022-11-28", headers["X-Github-Api-Version"], "Should include API version header")

		// Test https://api.github.com URLs
		headers = github.GetGitHubHeaders("https://api.github.com/repos/owner/repo")
		require.Equal(t, "Bearer test_token", headers["Authorization"], "Should include auth header for https://api.github.com")
		require.Equal(t, "2022-11-28", headers["X-Github-Api-Version"], "Should include API version header")
	})

	t.Run("NonGitHubURLs", func(t *testing.T) {
		t.Setenv("DDEV_GITHUB_TOKEN", "test_token")

		// Test non-GitHub URLs
		testURLs := []string{
			"https://gitlab.com/owner/repo",
			"https://bitbucket.org/owner/repo",
			"https://example.com/file.tar.gz",
			"https://notgithub.com/owner/repo",
		}

		for _, testURL := range testURLs {
			headers := github.GetGitHubHeaders(testURL)
			require.Empty(t, headers, "Should return empty headers for non-GitHub URL: %s", testURL)
		}
	})

	t.Run("NoToken", func(t *testing.T) {
		t.Setenv("DDEV_GITHUB_TOKEN", "")
		t.Setenv("GH_TOKEN", "")
		t.Setenv("GITHUB_TOKEN", "")

		headers := github.GetGitHubHeaders("https://github.com/owner/repo")
		require.Empty(t, headers, "Should return empty headers when no token is set")
	})
}
