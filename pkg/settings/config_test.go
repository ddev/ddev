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
	SetDefault("test_key", "default_value")
	assert.Equal(t, "default_value", GetString("test_key"))

	// 2. Test Environment Variable Overrides
	// Set the environment variable. Prefix is DDEV_, so DDEV_TEST_VAR -> GetString("TEST_VAR")
	os.Setenv("DDEV_TEST_VAR", "env_value")
	defer os.Unsetenv("DDEV_TEST_VAR")

	assert.Equal(t, "env_value", GetString("TEST_VAR"), "Environment variable should be accessible via GetString")

	// 3. Test Type Safety
	os.Setenv("DDEV_BOOL_VAR", "true")
	defer os.Unsetenv("DDEV_BOOL_VAR")
	assert.True(t, GetBool("BOOL_VAR"), "String 'true' should be parsed as boolean true")

	os.Setenv("DDEV_INT_VAR", "123")
	defer os.Unsetenv("DDEV_INT_VAR")
	assert.Equal(t, 123, GetInt("INT_VAR"), "String '123' should be parsed as integer 123")
	
	// 4. Test Set/override
	Set("some_key", "manual_value")
	assert.Equal(t, "manual_value", GetString("some_key"))

	// 5. Test Unset
	Unset("some_key")
	assert.Equal(t, "", GetString("some_key"), "Unset should remove the key's value")
}
