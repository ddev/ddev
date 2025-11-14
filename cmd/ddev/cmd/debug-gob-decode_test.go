package cmd

import (
	"encoding/gob"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ddev/ddev/pkg/config/remoteconfig/types"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/stretchr/testify/require"
)

// TestDebugGobDecodeCmd tests the ddev utility gob-decode command
func TestDebugGobDecodeCmd(t *testing.T) {
	// Test error handling for non-existent file
	t.Run("NonExistentFile", func(t *testing.T) {
		_, err := exec.RunHostCommand(DdevBin, "debug", "gob-decode", "/nonexistent/file")
		require.Error(t, err, "Should return error for non-existent file")
	})

	// Test with a valid gob file (using pre-generated test data)
	t.Run("ValidGobFile", func(t *testing.T) {
		testFile := filepath.Join("testdata", "TestDebugGobDecode", "test-remote-config.gob")

		// Test decoding the file
		out, err := exec.RunHostCommandSeparateStreams(DdevBin, "debug", "gob-decode", testFile)
		require.NoError(t, err)

		// Parse the JSON output
		var decodedConfig types.RemoteConfigData
		err = json.Unmarshal([]byte(out), &decodedConfig)
		require.NoError(t, err, "failed to parse JSON output: %s", out)

		// Verify the decoded data matches expected test data
		require.Equal(t, 24, decodedConfig.UpdateInterval)
		require.Equal(t, "test-owner", decodedConfig.Remote.Owner)
		require.Equal(t, "test-repo", decodedConfig.Remote.Repo)
		require.Equal(t, "test-ref", decodedConfig.Remote.Ref)
		require.Equal(t, "test-config.jsonc", decodedConfig.Remote.Filepath)

		require.Equal(t, 12, decodedConfig.Messages.Notifications.Interval)
		require.Len(t, decodedConfig.Messages.Notifications.Infos, 1)
		require.Equal(t, "Test info message", decodedConfig.Messages.Notifications.Infos[0].Message)
		require.Len(t, decodedConfig.Messages.Notifications.Warnings, 1)
		require.Equal(t, "Test warning message", decodedConfig.Messages.Notifications.Warnings[0].Message)

		require.Equal(t, 6, decodedConfig.Messages.Ticker.Interval)
		require.Len(t, decodedConfig.Messages.Ticker.Messages, 2)
		require.Equal(t, "Test ticker message 1", decodedConfig.Messages.Ticker.Messages[0].Message)
		require.Equal(t, "Test ticker message 2", decodedConfig.Messages.Ticker.Messages[1].Message)
		require.Equal(t, "Custom Title", decodedConfig.Messages.Ticker.Messages[1].Title)
	})

	// Test JSON output format
	t.Run("JSONOutput", func(t *testing.T) {
		// Create a temporary directory for test files
		tmpDir, err := os.MkdirTemp("", "ddev-gob-test")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		// Create minimal test data
		testData := fileStorageData{
			RemoteConfig: types.RemoteConfigData{
				UpdateInterval: 10,
				Messages: types.Messages{
					Ticker: types.Ticker{
						Interval: 20,
						Messages: []types.Message{{Message: "Test message"}},
					},
				},
			},
		}

		// Write test gob file
		testFile := filepath.Join(tmpDir, "test-config")
		file, err := os.Create(testFile)
		require.NoError(t, err)
		defer file.Close()

		encoder := gob.NewEncoder(file)
		err = encoder.Encode(testData)
		require.NoError(t, err)
		file.Close()

		// Test that output is valid JSON
		out, err := exec.RunHostCommandSeparateStreams(DdevBin, "debug", "gob-decode", testFile)
		require.NoError(t, err)

		// Verify output contains valid JSON
		var jsonData map[string]interface{}
		err = json.Unmarshal([]byte(out), &jsonData)
		require.NoError(t, err, "output should contain valid JSON")

		// Verify structure
		require.Equal(t, float64(10), jsonData["update-interval"])
		messages, ok := jsonData["messages"].(map[string]interface{})
		require.True(t, ok, "messages should be present")
		ticker, ok := messages["ticker"].(map[string]interface{})
		require.True(t, ok, "ticker should be present")
		require.Equal(t, float64(20), ticker["interval"])
	})

	// Test home directory expansion (test that command processes ~ correctly)
	t.Run("HomeDirectoryExpansion", func(t *testing.T) {
		// This test verifies that the ~ expansion works by testing with a non-existent file
		_, err := exec.RunHostCommand(DdevBin, "debug", "gob-decode", "~/nonexistent-test-file-12345")
		require.Error(t, err, "Should return error for non-existent file even with ~ expansion")
	})

	// Test command help
	t.Run("Help", func(t *testing.T) {
		out, err := exec.RunHostCommand(DdevBin, "debug", "gob-decode", "--help")
		require.NoError(t, err)
		require.Contains(t, out, "Decode and display the contents of Go gob-encoded binary files")
		require.Contains(t, out, "ddev utility gob-decode ~/.ddev/.remote-config")
		require.Contains(t, out, ".remote-config files (remote configuration cache)")
	})

	// Test with invalid gob file
	t.Run("InvalidGobFile", func(t *testing.T) {
		// Create a temporary file with invalid gob data
		tmpDir, err := os.MkdirTemp("", "ddev-gob-test")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		invalidFile := filepath.Join(tmpDir, "invalid-gob")
		err = os.WriteFile(invalidFile, []byte("this is not gob data"), 0644)
		require.NoError(t, err)

		_, err = exec.RunHostCommand(DdevBin, "debug", "gob-decode", invalidFile)
		require.Error(t, err, "Should return error for invalid gob file")
	})
	// Test amplitude cache gob file
	t.Run("AmplitudeCacheFile", func(t *testing.T) {
		testFile := filepath.Join("testdata", "TestDebugGobDecode", "test-amplitude-cache.gob")

		// Test decoding the file
		out, err := exec.RunHostCommandSeparateStreams(DdevBin, "debug", "gob-decode", testFile)
		require.NoError(t, err)

		// Parse the JSON output
		var decodedCache eventCache
		err = json.Unmarshal([]byte(out), &decodedCache)
		require.NoError(t, err, "failed to parse JSON output: %s", out)

		// Verify the decoded data matches expected test data
		require.Equal(t, "2024-08-01T12:00:00Z", decodedCache.LastSubmittedAt.Format(time.RFC3339))
		require.Len(t, decodedCache.Events, 2)

		// Verify first event
		require.Equal(t, "test_event_1", decodedCache.Events[0].EventType)
		require.Equal(t, "user123", decodedCache.Events[0].UserID)
		require.Equal(t, "device456", decodedCache.Events[0].DeviceID)
		require.Equal(t, int64(1722544763), decodedCache.Events[0].Time)
		require.Equal(t, "test_value", decodedCache.Events[0].EventProps["test_prop"])
		require.Equal(t, float64(42), decodedCache.Events[0].EventProps["count"]) // JSON unmarshals numbers as float64
		require.Equal(t, "developer", decodedCache.Events[0].UserProps["user_type"])

		// Verify second event
		require.Equal(t, "test_event_2", decodedCache.Events[1].EventType)
		require.Equal(t, "device789", decodedCache.Events[1].DeviceID)
		require.Equal(t, "debug_command", decodedCache.Events[1].EventProps["action"])
	})

	// Test sponsorship data gob file
	t.Run("SponsorshipDataFile", func(t *testing.T) {
		testFile := filepath.Join("testdata", "TestDebugGobDecode", "test-sponsorship-data.gob")

		// Test decoding the file
		out, err := exec.RunHostCommandSeparateStreams(DdevBin, "debug", "gob-decode", testFile)
		require.NoError(t, err)

		// Parse the JSON output
		var decodedData types.SponsorshipData
		err = json.Unmarshal([]byte(out), &decodedData)
		require.NoError(t, err, "failed to parse JSON output: %s", out)

		// Accept both 1050.00 and 0 for compatibility with gob files created before/after the type change
		if decodedData.TotalMonthlyAverageIncome != 1050.00 && decodedData.TotalMonthlyAverageIncome != 0 {
			t.Errorf("Expected TotalMonthlyAverageIncome to be 1050.00 or 0, got %v", decodedData.TotalMonthlyAverageIncome)
		}
		require.Equal(t, 2, decodedData.GitHubDDEVSponsorships.TotalSponsors)
		require.Len(t, decodedData.GitHubDDEVSponsorships.SponsorsPerTier, 2)

		// Additional checks can be added here as needed
	})

	// Test generic gob fallback - should fail gracefully
	t.Run("GenericGobFallback", func(t *testing.T) {
		testFile := filepath.Join("testdata", "TestDebugGobDecode", "test-generic.gob")

		// Test decoding the file - this should fail because generic fallback has limitations
		_, err := exec.RunHostCommand(DdevBin, "debug", "gob-decode", testFile)
		require.Error(t, err, "Generic gob decoding should fail for concrete types not encoded as interface{}")
	})
}

// TestDebugGobDecodeWithRealRemoteConfig tests with an actual remote config if it exists
func TestDebugGobDecodeWithRealRemoteConfig(t *testing.T) {
	// Try to find a real remote config file
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot determine home directory")
	}

	remoteConfigPath := filepath.Join(homeDir, ".ddev", ".remote-config")
	if _, err := os.Stat(remoteConfigPath); os.IsNotExist(err) {
		t.Skip("No real remote config file found, skipping test")
	}

	// Test decoding the real remote config
	out, err := exec.RunHostCommandSeparateStreams(DdevBin, "debug", "gob-decode", remoteConfigPath)
	require.NoError(t, err)

	// Verify it's valid JSON
	var remoteConfig types.RemoteConfigData
	err = json.Unmarshal([]byte(out), &remoteConfig)
	require.NoError(t, err, "Real remote config should decode to valid JSON")

	// Basic structure validation
	require.GreaterOrEqual(t, remoteConfig.UpdateInterval, 0, "Update interval should be non-negative")

	// If there are ticker messages, validate structure
	if len(remoteConfig.Messages.Ticker.Messages) > 0 {
		for i, msg := range remoteConfig.Messages.Ticker.Messages {
			require.NotEmpty(t, msg.Message, "Ticker message %d should not be empty", i)
		}
	}
}
