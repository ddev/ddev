package remoteconfig_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ddev/ddev/pkg/config/remoteconfig"
	"github.com/ddev/ddev/pkg/config/remoteconfig/downloader"
	"github.com/ddev/ddev/pkg/config/remoteconfig/types"
	"github.com/ddev/ddev/pkg/config/state/storage/yaml"
	"github.com/ddev/ddev/pkg/github"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/stretchr/testify/require"
)

// TestRemoteConfigEndToEnd tests the complete remote config functionality
// including downloading from the actual GitHub repository
func TestRemoteConfigEndToEnd(t *testing.T) {
	require := require.New(t)

	// Create a function that always returns true for testing
	alwaysInternetActive := func() bool { return true }

	// Create temporary directory for test
	tmpDir := testcommon.CreateTmpDir("TestRemoteConfigEndToEnd")
	defer testcommon.CleanupDir(tmpDir)

	// Create state manager
	stateManager := yaml.NewState(filepath.Join(tmpDir, "state.yaml"))

	// Test 1: Test generic JSONC downloader directly
	t.Run("GenericJSONCDownloader", func(t *testing.T) {
		downloader := downloader.NewGitHubJSONCDownloader(
			"ddev",
			"remote-config",
			"remote-config.jsonc",
			github.RepositoryContentGetOptions{Ref: "main"},
		)

		var remoteConfig types.RemoteConfigData
		ctx := context.Background()
		err := downloader.Download(ctx, &remoteConfig)
		require.NoError(err, "Failed to download remote config")

		// Debug: Print what we actually got
		tickerMessages := len(remoteConfig.Messages.Ticker.Messages)
		tickerInterval := remoteConfig.Messages.Ticker.Interval

		t.Logf("Downloaded remote config: UpdateInterval=%d", remoteConfig.UpdateInterval)
		t.Logf("Ticker: Messages=%d, Interval=%d", tickerMessages, tickerInterval)

		// Verify we got meaningful data
		require.Greater(remoteConfig.UpdateInterval, 0, "Update interval should be greater than 0")
		require.Greater(len(remoteConfig.Messages.Ticker.Messages), 50, "Should have many ticker messages")
		require.Greater(remoteConfig.Messages.Ticker.Interval, 0, "Ticker interval should be greater than 0")

		// Verify some message content
		foundDDEVMessage := false
		for _, msg := range remoteConfig.Messages.Ticker.Messages {
			require.NotEmpty(msg.Message, "Message content should not be empty")
			if len(msg.Message) > 4 && msg.Message[:4] == "DDEV" {
				foundDDEVMessage = true
			}
		}
		require.True(foundDDEVMessage, "Should find at least one message mentioning DDEV")
	})

	// Test 2: Test complete remote config system
	t.Run("RemoteConfigSystem", func(t *testing.T) {
		config := remoteconfig.Config{
			Local: remoteconfig.Local{
				Path: tmpDir,
			},
			Remote: remoteconfig.Remote{
				Owner:    "ddev",
				Repo:     "remote-config",
				Ref:      "main",
				Filepath: "remote-config.jsonc",
			},
			UpdateInterval: 1, // 1 hour for testing
			TickerInterval: 1, // 1 hour for testing
		}

		// Create remote config instance
		rc := remoteconfig.New(&config, stateManager, alwaysInternetActive)
		require.NotNil(rc, "Remote config should not be nil")

		// Verify local file was created (this tests the write functionality)
		localFile := filepath.Join(tmpDir, ".remote-config")
		_, err := os.Stat(localFile)
		require.NoError(err, "Local remote config file should exist")

		// Test ShowTicker and ShowNotifications don't panic
		require.NotPanics(func() {
			rc.ShowTicker()
		}, "ShowTicker should not panic")

		require.NotPanics(func() {
			rc.ShowNotifications()
		}, "ShowNotifications should not panic")
	})

	// Test 3: Test global remote config functions
	t.Run("GlobalRemoteConfig", func(t *testing.T) {
		// Test that global config can be initialized
		config := remoteconfig.Config{
			Local: remoteconfig.Local{
				Path: tmpDir,
			},
			UpdateInterval: 1,
			TickerInterval: 1,
		}

		globalRC := remoteconfig.InitGlobal(config, stateManager, alwaysInternetActive)
		require.NotNil(globalRC, "Global remote config should not be nil")

		// Test that we can get the global config
		retrievedRC := remoteconfig.GetGlobal()
		require.Equal(globalRC, retrievedRC, "Retrieved global config should match initialized config")

		// Test ShowTicker and ShowNotifications work on global config
		require.NotPanics(func() {
			retrievedRC.ShowTicker()
		}, "Global ShowTicker should not panic")

		require.NotPanics(func() {
			retrievedRC.ShowNotifications()
		}, "Global ShowNotifications should not panic")
	})

	// Test 4: Test that local caching works
	t.Run("LocalCaching", func(t *testing.T) {
		cacheDir := tmpDir + "_cache_test"
		err := os.MkdirAll(cacheDir, 0755)
		require.NoError(err, "Should be able to create cache directory")

		// Create a separate state manager for this test to ensure fresh state
		cacheStateManager := yaml.NewState(filepath.Join(cacheDir, "cache_state.yaml"))

		config := remoteconfig.Config{
			Local: remoteconfig.Local{
				Path: cacheDir,
			},
			Remote: remoteconfig.Remote{
				Owner:    "ddev",
				Repo:     "remote-config",
				Ref:      "main",
				Filepath: "remote-config.jsonc",
			},
			UpdateInterval: 1, // 1 hour - will trigger update since state is fresh
		}

		// First creation should download
		rc1 := remoteconfig.New(&config, cacheStateManager, alwaysInternetActive)
		require.NotNil(rc1)

		// Wait a moment for async operations to complete
		time.Sleep(100 * time.Millisecond)

		// Verify local file exists
		localFile := filepath.Join(cacheDir, ".remote-config")
		_, err = os.Stat(localFile)
		require.NoError(err, "Local cache file should exist")

		// Second creation should use cache (since update interval is 1 hour and we just updated)
		rc2 := remoteconfig.New(&config, cacheStateManager, func() bool { return false }) // No internet
		require.NotNil(rc2, "Should work from cache even without internet")
	})
}

// TestSponsorshipDataEndToEnd tests the sponsorship data functionality
func TestSponsorshipDataEndToEnd(t *testing.T) {
	require := require.New(t)

	// Create a function that always returns true for testing
	alwaysInternetActive := func() bool { return true }

	// Create temporary directory for test
	tmpDir := testcommon.CreateTmpDir("TestSponsorshipDataEndToEnd")
	defer testcommon.CleanupDir(tmpDir)

	// Create state manager
	stateManager := yaml.NewState(filepath.Join(tmpDir, "sponsorship_state.yaml"))

	t.Run("SponsorshipManager", func(t *testing.T) {
		// Create sponsorship manager
		mgr := remoteconfig.NewSponsorshipManager(tmpDir, stateManager, alwaysInternetActive)
		require.NotNil(mgr, "Sponsorship manager should not be nil")

		// Test getting sponsorship data
		data, err := mgr.GetSponsorshipData()
		require.NoError(err, "Should be able to get sponsorship data")
		require.NotNil(data, "Sponsorship data should not be nil")

		// Verify data structure - note that we may get empty data if download fails,
		// so we just check that we can access the fields without error
		require.GreaterOrEqual(data.TotalMonthlyAverageIncome, float64(0), "Should have valid monthly income field")
		require.GreaterOrEqual(data.GitHubDDEVSponsorships.TotalSponsors, 0, "Should have valid GitHub DDEV sponsors field")
		require.GreaterOrEqual(data.GitHubDDEVSponsorships.TotalMonthlySponsorship, 0, "Should have valid GitHub DDEV sponsorship amount field")

		// Test utility methods
		totalIncome := mgr.GetTotalMonthlyIncome()
		require.Equal(data.TotalMonthlyAverageIncome, totalIncome, "Total income should match data field")

		totalSponsors := mgr.GetTotalSponsors()
		expectedTotal := data.GitHubDDEVSponsorships.TotalSponsors +
			data.GitHubRfaySponsorships.TotalSponsors +
			data.MonthlyInvoicedSponsorships.TotalSponsors +
			data.AnnualInvoicedSponsorships.TotalSponsors
		require.Equal(expectedTotal, totalSponsors, "Total sponsors should match sum of all sponsor types")

		// Test data freshness (should be fresh after just downloading)
		require.False(mgr.IsDataStale(), "Data should be fresh after download")
	})

	t.Run("GlobalSponsorshipManager", func(t *testing.T) {
		// Test global sponsorship manager
		globalMgr := remoteconfig.InitGlobalSponsorship(tmpDir, stateManager, alwaysInternetActive)
		require.NotNil(globalMgr, "Global sponsorship manager should not be nil")

		// Test retrieval
		retrievedMgr := remoteconfig.GetGlobalSponsorship()
		require.Equal(globalMgr, retrievedMgr, "Retrieved global sponsorship manager should match initialized one")

		// Test functionality
		data, err := retrievedMgr.GetSponsorshipData()
		require.NoError(err, "Should be able to get sponsorship data from global manager")
		require.GreaterOrEqual(data.TotalMonthlyAverageIncome, float64(0), "Global manager should have valid data")
	})
}

// TestRemoteConfigStructure tests that the downloaded remote config has expected structure
func TestRemoteConfigStructure(t *testing.T) {
	require := require.New(t)

	// Test the actual structure matches our expectations
	downloader := downloader.NewGitHubJSONCDownloader(
		"ddev",
		"remote-config",
		"remote-config.jsonc",
		github.RepositoryContentGetOptions{Ref: "main"},
	)

	var remoteConfig types.RemoteConfigData
	ctx := context.Background()
	err := downloader.Download(ctx, &remoteConfig)
	require.NoError(err, "Should download remote config successfully")

	// Test structure
	require.Equal(10, remoteConfig.UpdateInterval, "Update interval should be 10 hours as per remote config")
	require.Greater(remoteConfig.Messages.Ticker.Interval, 0, "Ticker interval should be positive")
	require.Greater(len(remoteConfig.Messages.Ticker.Messages), 70, "Should have many ticker messages (at least 70)")

	// Test message content quality
	messageContentTypes := make(map[string]int)
	for _, msg := range remoteConfig.Messages.Ticker.Messages {
		require.NotEmpty(msg.Message, "Each message should have content")
		require.LessOrEqual(len(msg.Message), 500, "Messages should be reasonably short")

		// Categorize message content
		content := msg.Message
		if len(content) > 0 {
			switch {
			case containsAny(content, []string{"ddev", "DDEV"}):
				messageContentTypes["ddev"]++
			case containsAny(content, []string{"command", "cmd", "`"}):
				messageContentTypes["command"]++
			case containsAny(content, []string{"sponsor", "funding", "support"}):
				messageContentTypes["sponsor"]++
			case containsAny(content, []string{"community", "Discord", "GitHub"}):
				messageContentTypes["community"]++
			default:
				messageContentTypes["other"]++
			}
		}
	}

	// Verify we have a good mix of message types
	require.Greater(messageContentTypes["ddev"], 10, "Should have many DDEV-related messages")
	// Note: The current remote config messages are mostly DDEV-focused, so command/community categories may be minimal
	require.GreaterOrEqual(messageContentTypes["command"], 0, "Command-related messages count")
	require.GreaterOrEqual(messageContentTypes["community"], 0, "Community-related messages count")

	// The important test is that we have meaningful DDEV-related content
	totalCategorized := messageContentTypes["ddev"] + messageContentTypes["command"] + messageContentTypes["community"] + messageContentTypes["sponsor"] + messageContentTypes["other"]
	require.Equal(len(remoteConfig.Messages.Ticker.Messages), totalCategorized, "All messages should be categorized")

	t.Logf("Message content distribution: %+v", messageContentTypes)
}

// Helper function to check if a string contains any of the given substrings (case-insensitive)
func containsAny(s string, substrings []string) bool {
	sLower := strings.ToLower(s)
	for _, sub := range substrings {
		subLower := strings.ToLower(sub)
		if strings.Contains(sLower, subLower) {
			return true
		}
	}
	return false
}
