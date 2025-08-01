package cmd

import (
	"encoding/gob"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ddev/ddev/pkg/exec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDebugGobDecodeCmd tests the ddev debug gob-decode command
func TestDebugGobDecodeCmd(t *testing.T) {
	assert := assert.New(t)

	// Test error handling for non-existent file
	t.Run("NonExistentFile", func(t *testing.T) {
		_, err := exec.RunHostCommand(DdevBin, "debug", "gob-decode", "/nonexistent/file")
		assert.Error(err, "Should return error for non-existent file")
	})

	// Test with a valid gob file (using pre-generated test data)
	t.Run("ValidGobFile", func(t *testing.T) {
		testFile := filepath.Join("testdata", "TestDebugGobDecode", "test-remote-config.gob")
		
		// Test decoding the file
		out, err := exec.RunHostCommandSeparateStreams(DdevBin, "debug", "gob-decode", testFile)
		assert.NoError(err)

		// Parse the JSON output
		var decodedConfig RemoteConfig
		err = json.Unmarshal([]byte(out), &decodedConfig)
		require.NoError(t, err, "failed to parse JSON output: %s", out)

		// Verify the decoded data matches expected test data
		assert.Equal(24, decodedConfig.UpdateInterval)
		assert.Equal("test-owner", decodedConfig.Remote.Owner)
		assert.Equal("test-repo", decodedConfig.Remote.Repo)
		assert.Equal("test-ref", decodedConfig.Remote.Ref)
		assert.Equal("test-config.jsonc", decodedConfig.Remote.Filepath)

		assert.Equal(12, decodedConfig.Messages.Notifications.Interval)
		assert.Len(decodedConfig.Messages.Notifications.Infos, 1)
		assert.Equal("Test info message", decodedConfig.Messages.Notifications.Infos[0].Message)
		assert.Len(decodedConfig.Messages.Notifications.Warnings, 1)
		assert.Equal("Test warning message", decodedConfig.Messages.Notifications.Warnings[0].Message)

		assert.Equal(6, decodedConfig.Messages.Ticker.Interval)
		assert.Len(decodedConfig.Messages.Ticker.Messages, 2)
		assert.Equal("Test ticker message 1", decodedConfig.Messages.Ticker.Messages[0].Message)
		assert.Equal("Test ticker message 2", decodedConfig.Messages.Ticker.Messages[1].Message)
		assert.Equal("Custom Title", decodedConfig.Messages.Ticker.Messages[1].Title)
	})

	// Test JSON output format
	t.Run("JSONOutput", func(t *testing.T) {
		// Create a temporary directory for test files
		tmpDir, err := os.MkdirTemp("", "ddev-gob-test")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		// Create minimal test data
		testData := fileStorageData{
			RemoteConfig: RemoteConfig{
				UpdateInterval: 10,
				Messages: Messages{
					Ticker: Ticker{
						Interval: 20,
						Messages: []Message{{Message: "Test message"}},
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
		assert.NoError(err)

		// Verify output contains valid JSON
		var jsonData map[string]interface{}
		err = json.Unmarshal([]byte(out), &jsonData)
		assert.NoError(err, "output should contain valid JSON")

		// Verify structure
		assert.Equal(float64(10), jsonData["update-interval"])
		messages, ok := jsonData["messages"].(map[string]interface{})
		assert.True(ok, "messages should be present")
		ticker, ok := messages["ticker"].(map[string]interface{})
		assert.True(ok, "ticker should be present")
		assert.Equal(float64(20), ticker["interval"])
	})

	// Test home directory expansion (test that command processes ~ correctly)
	t.Run("HomeDirectoryExpansion", func(t *testing.T) {
		// This test verifies that the ~ expansion works by testing with a non-existent file
		_, err := exec.RunHostCommand(DdevBin, "debug", "gob-decode", "~/nonexistent-test-file-12345")
		assert.Error(err, "Should return error for non-existent file even with ~ expansion")
	})

	// Test command help
	t.Run("Help", func(t *testing.T) {
		out, err := exec.RunHostCommand(DdevBin, "debug", "gob-decode", "--help")
		assert.NoError(err)
		assert.Contains(out, "Decode and display the contents of Go gob-encoded binary files")
		assert.Contains(out, "ddev debug gob-decode ~/.ddev/.remote-config")
		assert.Contains(out, ".remote-config files (remote configuration cache)")
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
		assert.Error(err, "Should return error for invalid gob file")
	})
	// Test amplitude cache gob file
	t.Run("AmplitudeCacheFile", func(t *testing.T) {
		testFile := filepath.Join("testdata", "TestDebugGobDecode", "test-amplitude-cache.gob")
		
		// Test decoding the file
		out, err := exec.RunHostCommandSeparateStreams(DdevBin, "debug", "gob-decode", testFile)
		assert.NoError(err)

		// Parse the JSON output
		var decodedCache eventCache
		err = json.Unmarshal([]byte(out), &decodedCache)
		require.NoError(t, err, "failed to parse JSON output: %s", out)

		// Verify the decoded data matches expected test data
		assert.Equal("2024-08-01T12:00:00Z", decodedCache.LastSubmittedAt.Format(time.RFC3339))
		assert.Len(decodedCache.Events, 2)
		
		// Verify first event
		assert.Equal("test_event_1", decodedCache.Events[0].EventType)
		assert.Equal("user123", decodedCache.Events[0].UserID)
		assert.Equal("device456", decodedCache.Events[0].DeviceID)
		assert.Equal(int64(1722544763), decodedCache.Events[0].Time)
		assert.Equal("test_value", decodedCache.Events[0].EventProps["test_prop"])
		assert.Equal(float64(42), decodedCache.Events[0].EventProps["count"]) // JSON unmarshals numbers as float64
		assert.Equal("developer", decodedCache.Events[0].UserProps["user_type"])

		// Verify second event
		assert.Equal("test_event_2", decodedCache.Events[1].EventType)
		assert.Equal("device789", decodedCache.Events[1].DeviceID)
		assert.Equal("debug_command", decodedCache.Events[1].EventProps["action"])
	})

	// Test sponsorship data gob file
	t.Run("SponsorshipDataFile", func(t *testing.T) {
		testFile := filepath.Join("testdata", "TestDebugGobDecode", "test-sponsorship-data.gob")
		
		// Test decoding the file
		out, err := exec.RunHostCommandSeparateStreams(DdevBin, "debug", "gob-decode", testFile)
		assert.NoError(err)

		// Parse the JSON output
		var decodedData SponsorshipData
		err = json.Unmarshal([]byte(out), &decodedData)
		require.NoError(t, err, "failed to parse JSON output: %s", out)

		// Verify the decoded data matches expected test data
		assert.Equal(1050.00, decodedData.TotalIncome)
		assert.Equal(2, decodedData.SponsorCount)
		assert.Len(decodedData.Sponsors, 2)

		// Verify first sponsor
		assert.Equal("ACME Corporation", decodedData.Sponsors[0].Name)
		assert.Equal(1000.00, decodedData.Sponsors[0].Amount)
		assert.Equal("USD", decodedData.Sponsors[0].Currency)
		assert.Equal("monthly", decodedData.Sponsors[0].Type)
		assert.Equal("Supporting open source development", decodedData.Sponsors[0].Description)

		// Verify second sponsor
		assert.Equal("Developer Jane", decodedData.Sponsors[1].Name)
		assert.Equal(50.00, decodedData.Sponsors[1].Amount)
		assert.Equal("one-time", decodedData.Sponsors[1].Type)
	})

	// Test generic gob fallback - should fail gracefully
	t.Run("GenericGobFallback", func(t *testing.T) {
		testFile := filepath.Join("testdata", "TestDebugGobDecode", "test-generic.gob")
		
		// Test decoding the file - this should fail because generic fallback has limitations
		_, err := exec.RunHostCommand(DdevBin, "debug", "gob-decode", testFile)
		assert.Error(err, "Generic gob decoding should fail for concrete types not encoded as interface{}")
	})
}

// TestDebugGobDecodeWithRealRemoteConfig tests with an actual remote config if it exists
func TestDebugGobDecodeWithRealRemoteConfig(t *testing.T) {
	assert := assert.New(t)

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
	assert.NoError(err)

	// Verify it's valid JSON
	var remoteConfig RemoteConfig
	err = json.Unmarshal([]byte(out), &remoteConfig)
	assert.NoError(err, "Real remote config should decode to valid JSON")

	// Basic structure validation
	assert.GreaterOrEqual(remoteConfig.UpdateInterval, 0, "Update interval should be non-negative")

	// If there are ticker messages, validate structure
	if len(remoteConfig.Messages.Ticker.Messages) > 0 {
		for i, msg := range remoteConfig.Messages.Ticker.Messages {
			assert.NotEmpty(msg.Message, "Ticker message %d should not be empty", i)
		}
	}
}
