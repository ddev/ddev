package settings

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestViperConfiguration verifies that the Viper wrapper correctly handles
// configuration priority, environment variables, and defaults.
// This ensures that the underlying Viper implementation behaves as expected
// regarding precedence rules (Env > Config > Defaults).
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

// TestUnmarshalYamlTags verifies that the Unmarshal method correctly respects 'yaml' tags.
// This is critical because our configuration files use snake_case keys (yaml tags),
// but we map them to CamelCase struct fields.
func TestUnmarshalYamlTags(t *testing.T) {
	p := NewConfigProvider()
	p.Set("php_version", "8.2")
	p.Set("name", "test-app")

	var cfg TestConfig
	err := p.Unmarshal(&cfg)
	assert.NoError(t, err)

	assert.Equal(t, "test-app", cfg.Name)
	assert.Equal(t, "8.2", cfg.PHPVersion)
}

// TestUnmarshalExistingValues verifies that Unmarshal does not zero out existing fields if not in config.
// This ensures that we can load configuration into a struct that already has default values
// without wiping out those defaults if the config file doesn't specify them.
func TestUnmarshalExistingValues(t *testing.T) {
	p := NewConfigProvider()
	// No "name" set in provider

	cfg := TestConfig{
		Name: "existing-name",
	}
	err := p.Unmarshal(&cfg)
	assert.NoError(t, err)

	assert.Equal(t, "existing-name", cfg.Name, "Existing name should be preserved if not in config")
}

// TestViperUnmarshalEnv verifies that Unmarshal does NOT pick up arbitrary environment variables
// unless they are explicitly bound. This is a security and stability feature to prevent
// "poisoning" of the configuration from unrelated environment variables.
func TestViperUnmarshalEnv(t *testing.T) {
	os.Setenv("DDEV_TEST_PORT", "8888")
	defer os.Unsetenv("DDEV_TEST_PORT")

	type Config struct {
		TestPort string `yaml:"test_port"`
	}

	p := NewConfigProvider()

	var cfg Config
	err := p.Unmarshal(&cfg)
	assert.NoError(t, err)

	assert.Equal(t, "", cfg.TestPort, "Unmarshal SHOULD NOT pick up unbound environment variables")
	assert.Equal(t, "8888", p.GetString("test_port"), "GetString SHOULD pick up environment variables via AutomaticEnv")
}

// TestViperUnmarshalEnvStandardDDEV verifies that Unmarshal DOES pick up environment variables
// that are explicitly bound by the NewConfigProvider (bindStandardGlobalEnvs).
// This ensures that standard DDEV environment variables (like DDEV_ROUTER_HTTP_PORT)
// correctly override configuration defaults.
func TestViperUnmarshalEnvStandardDDEV(t *testing.T) {
	v := NewConfigProvider()

	type Config struct {
		RouterHTTPPort string `yaml:"router_http_port"`
	}

	_ = os.Setenv("DDEV_ROUTER_HTTP_PORT", "9999")
	defer os.Unsetenv("DDEV_ROUTER_HTTP_PORT")

	var cfg Config
	err := v.Unmarshal(&cfg)
	assert.NoError(t, err)

	// This should now be "9999" because it's bound in NewConfigProvider
	assert.Equal(t, "9999", cfg.RouterHTTPPort)
}
