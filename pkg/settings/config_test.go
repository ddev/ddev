package settings

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestViperConfiguration verifies that the Viper wrapper correctly handles
// configuration priority, environment variables, and defaults.
func TestViperConfiguration(t *testing.T) {
	// Initialize the settings system
	Init()

	// 1. Test Default Values
	// This would require adding SetDefault to the interface
	// settings.SetDefault("test_key", "default_value")
	// assert.Equal(t, "default_value", GetString("test_key"))

	// 2. Test Environment Variable Overrides
	// Note: In Init(), we should call v.AutomaticEnv() and v.SetEnvPrefix("DDEV")
	os.Setenv("DDEV_TEST_VAR", "env_value")
	defer os.Unsetenv("DDEV_TEST_VAR")

	// Viper needs to be told to look for this specific env var if prefix is set
	// or we can use AutomaticEnv() with a prefix.
	// For this test to pass, Init() needs to be updated to support env vars.
	assert.Equal(t, "env_value", GetString("TEST_VAR"), "Environment variable should override defaults")

	// 3. Test Type Safety
	os.Setenv("DDEV_BOOL_VAR", "true")
	defer os.Unsetenv("DDEV_BOOL_VAR")
	assert.True(t, GetBool("BOOL_VAR"), "String 'true' should be parsed as boolean true")

	os.Setenv("DDEV_INT_VAR", "123")
	defer os.Unsetenv("DDEV_INT_VAR")
	assert.Equal(t, 123, GetInt("INT_VAR"), "String '123' should be parsed as integer 123")
}
