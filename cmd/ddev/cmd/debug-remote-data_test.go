package cmd

import (
	"encoding/json"
	"github.com/ddev/ddev/pkg/globalconfig"
	"os"
	"path/filepath"
	"testing"

	"github.com/ddev/ddev/pkg/config/remoteconfig/types"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/stretchr/testify/require"
)

// TestDebugRemoteDataCmd tests the ddev debug remote-data command
func TestDebugRemoteDataCmd(t *testing.T) {
	// Test help functionality
	t.Run("Help", func(t *testing.T) {
		out, err := exec.RunHostCommand(DdevBin, "debug", "remote-data", "--help")
		require.NoError(t, err)
		require.Contains(t, out, "Download and display remote data used by DDEV from GitHub repositories")
		require.Contains(t, out, "remote-config: DDEV remote configuration")
		require.Contains(t, out, "sponsorship-data: DDEV sponsorship information")
		require.Contains(t, out, "--type string")
		require.Contains(t, out, "--update-storage")
	})

	// Test invalid data type
	t.Run("InvalidDataType", func(t *testing.T) {
		_, err := exec.RunHostCommand(DdevBin, "debug", "remote-data", "--type=invalid")
		require.Error(t, err, "Should return error for invalid data type")
	})

	// Test remote config download (without updating storage)
	t.Run("RemoteConfigDownload", func(t *testing.T) {
		out, err := exec.RunHostCommandSeparateStreams(DdevBin, "debug", "remote-data", "--type=remote-config", "--update-storage=false")
		require.NoError(t, err, "Should successfully download remote config, output='%v'", out)

		// Parse the JSON output
		var remoteConfig types.RemoteConfigData
		err = json.Unmarshal([]byte(out), &remoteConfig)
		require.NoError(t, err, "Output should be valid JSON: %s", out)

		// Verify basic structure
		require.GreaterOrEqual(t, remoteConfig.UpdateInterval, 0, "Update interval should be non-negative")
		require.Greater(t, remoteConfig.Messages.Ticker.Interval, 0, "Ticker interval should be positive")
		require.Greater(t, len(remoteConfig.Messages.Ticker.Messages), 0, "Should have ticker messages")

		// Verify at least one ticker message has content
		hasValidMessage := false
		for _, msg := range remoteConfig.Messages.Ticker.Messages {
			if msg.Message != "" {
				hasValidMessage = true
				break
			}
		}
		require.True(t, hasValidMessage, "Should have at least one valid ticker message")
	})

	// Test sponsorship data download (without updating storage)
	t.Run("SponsorshipDataDownload", func(t *testing.T) {
		out, err := exec.RunHostCommandSeparateStreams(DdevBin, "debug", "remote-data", "--type=sponsorship-data", "--update-storage=false")
		require.NoError(t, err, "Should successfully download sponsorship data, output='%v'", out)

		// Parse the JSON output
		var sponsorshipData types.SponsorshipData
		err = json.Unmarshal([]byte(out), &sponsorshipData)
		require.NoError(t, err, "Output should be valid JSON: %s", out)

		// Verify basic structure - these should be reasonable values
		require.GreaterOrEqual(t, sponsorshipData.TotalMonthlyAverageIncome, 0.0, "Total income should be non-negative")
		require.GreaterOrEqual(t, sponsorshipData.GitHubDDEVSponsorships.TotalSponsors, 0, "GitHub DDEV sponsors should be non-negative")
		require.GreaterOrEqual(t, sponsorshipData.GitHubRfaySponsorships.TotalSponsors, 0, "GitHub rfay sponsors should be non-negative")

		// The update time should be recent-ish (within the last year)
		require.False(t, sponsorshipData.UpdatedDateTime.IsZero(), "Updated datetime should be set")
	})

	// Test default behavior (should default to remote-config)
	t.Run("DefaultBehavior", func(t *testing.T) {
		out, err := exec.RunHostCommandSeparateStreams(DdevBin, "debug", "remote-data", "--update-storage=false")
		require.NoError(t, err, "Should successfully download with default type, output='%v'", out)

		// Should parse as remote config since that's the default
		var remoteConfig types.RemoteConfigData
		err = json.Unmarshal([]byte(out), &remoteConfig)
		require.NoError(t, err, "Default should download remote config")

		require.Greater(t, len(remoteConfig.Messages.Ticker.Messages), 0, "Should have ticker messages")
	})
}

// TestDebugRemoteDataWithStorage tests storage update functionality
func TestDebugRemoteDataWithStorage(t *testing.T) {
	// Create a temporary directory for test storage
	tmpDir, err := os.MkdirTemp("", "ddev-download-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Use t.Setenv to set XDG_CONFIG_HOME to tmpDir so .ddev is created there
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Set up global config with remote config settings
	globalconfig.EnsureGlobalConfig()
	globalconfig.DdevGlobalConfig.RemoteConfig.RemoteConfigURL = "https://raw.githubusercontent.com/ddev/remote-config/main/remote-config.jsonc"
	globalconfig.DdevGlobalConfig.RemoteConfig.SponsorshipDataURL = "https://ddev.com/s/sponsorship-data.json"
	err = globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig)
	require.NoError(t, err)

	t.Run("StorageUpdateEnabled", func(t *testing.T) {
		// Test with storage update enabled (default)
		out, err := exec.RunHostCommand(DdevBin, "debug", "remote-data", "--type=remote-config")
		require.NoError(t, err, "Should successfully download and update storage, output='%v'", out)
		require.Contains(t, out, "Local remote config storage updated successfully")

		// Verify that the storage file was created
		storageFile := filepath.Join(globalconfig.GetGlobalDdevDir(), ".remote-config")
		_, err = os.Stat(storageFile)
		require.NoErrorf(t, err, "Storage file should have been created at %s", storageFile)
	})

	t.Run("StorageUpdateDisabled", func(t *testing.T) {
		storageFile := filepath.Join(globalconfig.GetGlobalDdevDir(), ".remote-config")
		os.Remove(storageFile)

		// Test with storage update disabled
		out, err := exec.RunHostCommand(DdevBin, "debug", "remote-data", "--type=remote-config", "--update-storage=false")
		require.NoError(t, err, "Should successfully download without updating storage, output='%v'", out)
		require.NotContains(t, out, "Local remote config storage updated successfully")

		// Verify that no storage file was created
		_, err = os.Stat(storageFile)
		require.Error(t, err, "Storage file should not have been created")
	})
}

// TestJSONValidation tests that the output is always valid JSON
func TestJSONValidation(t *testing.T) {
	t.Run("RemoteConfigValidJSON", func(t *testing.T) {
		out, err := exec.RunHostCommandSeparateStreams(DdevBin, "debug", "remote-data", "--type=remote-config", "--update-storage=false")
		require.NoError(t, err, "Should successfully download remote config, output='%v'", out)

		// Test that it's valid JSON by unmarshaling to interface{}
		var jsonData interface{}
		err = json.Unmarshal([]byte(out), &jsonData)
		require.NoError(t, err, "Output should be valid JSON")

		// Test that it can be re-marshaled (round-trip test)
		_, err = json.Marshal(jsonData)
		require.NoError(t, err, "JSON should be serializable")
	})

	t.Run("SponsorshipDataValidJSON", func(t *testing.T) {
		out, err := exec.RunHostCommandSeparateStreams(DdevBin, "debug", "remote-data", "--type=sponsorship-data", "--update-storage=false")
		require.NoError(t, err, "Should successfully download sponsorship data, output='%v'", out)

		// Test that it's valid JSON
		var jsonData interface{}
		err = json.Unmarshal([]byte(out), &jsonData)
		require.NoError(t, err, "Output should be valid JSON")

		// Test round-trip
		_, err = json.Marshal(jsonData)
		require.NoError(t, err, "JSON should be serializable")
	})
}
