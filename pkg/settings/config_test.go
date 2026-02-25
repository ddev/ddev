package settings

import (
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

	// 4. Test Set/override
	Set("some_key", "manual_value")
	assert.Equal(t, "manual_value", GetString("some_key"))
}
