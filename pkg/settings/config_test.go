package settings

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestConfig is a dummy struct for testing unmarshaling.
type TestConfig struct {
	Name       string `yaml:"name"`
	Type       string `yaml:"type"`
	PHPVersion string `yaml:"php_version"`
	Webserver  string `yaml:"webserver_type"`
}

// TestLoadGlobalConfig verifies that a single YAML file is correctly loaded
// into a struct using the high-level LoadGlobalConfig function.
// It tests the happy path of configuration loading from a file.
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
// It ensures that specific project configurations can be overridden by local files,
// which is a key feature for DDEV's per-project extensibility.
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
// This confirms that our factory functions (NewConfigProvider) return truly independent
// instances, preventing crosstalk between different configuration loads.
func TestNewConfigProviderIsolation(t *testing.T) {
	p1 := NewConfigProvider()
	p2 := NewConfigProvider()

	p1.Set("key", "value1")
	p2.Set("key", "value2")

	assert.Equal(t, "value1", p1.GetString("key"))
	assert.Equal(t, "value2", p2.GetString("key"))
}
