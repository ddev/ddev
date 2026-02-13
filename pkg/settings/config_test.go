package settings

import (
	"os"
	"path/filepath"
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

// TestConfig is a dummy struct for testing unmarshaling.
type TestConfig struct {
	Name       string `yaml:"name"`
	Type       string `yaml:"type"`
	PHPVersion string `yaml:"php_version"`
	Webserver  string `yaml:"webserver_type"`
}

// TestLoadGlobalConfig verifies that a single YAML file is correctly loaded.
func TestLoadGlobalConfig(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "global_config.yaml")

	content := `
name: global-config
type: php
`
	err := os.WriteFile(configPath, []byte(content), 0644)
	assert.NoError(t, err)

	var cfg TestConfig
	err = LoadGlobalConfig(configPath, &cfg)
	assert.NoError(t, err)
	assert.Equal(t, "global-config", cfg.Name)
	assert.Equal(t, "php", cfg.Type)
}

// TestLoadProjectConfig verifies that main config and overrides are merged correctly.
func TestLoadProjectConfig(t *testing.T) {
	tempDir := t.TempDir()
	mainPath := filepath.Join(tempDir, "config.yaml")
	overridePath := filepath.Join(tempDir, "config.override.yaml")

	mainContent := `
name: project-name
type: drupal10
php_version: "8.1"
webserver_type: nginx-fpm
`
	overrideContent := `
php_version: "8.3"
webserver_type: apache-fpm
`
	err := os.WriteFile(mainPath, []byte(mainContent), 0644)
	assert.NoError(t, err)
	err = os.WriteFile(overridePath, []byte(overrideContent), 0644)
	assert.NoError(t, err)

	var cfg TestConfig
	err = LoadProjectConfig(mainPath, []string{overridePath}, &cfg)
	assert.NoError(t, err)

	assert.Equal(t, "project-name", cfg.Name)    // From main
	assert.Equal(t, "drupal10", cfg.Type)        // From main
	assert.Equal(t, "8.3", cfg.PHPVersion)       // Overridden
	assert.Equal(t, "apache-fpm", cfg.Webserver) // Overridden
}

// TestNewConfigProviderIsolation ensures that separate providers do not share state.
func TestNewConfigProviderIsolation(t *testing.T) {
	p1 := NewConfigProvider()
	p2 := NewConfigProvider()

	p1.Set("key", "value1")
	p2.Set("key", "value2")

	assert.Equal(t, "value1", p1.GetString("key"))
	assert.Equal(t, "value2", p2.GetString("key"))
}

// TestUnmarshalYamlTags verifies that the Unmarshal method correctly respects 'yaml' tags.
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
