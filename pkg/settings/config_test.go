package settings

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

type TestConfig struct {
	Name       string            `yaml:"name"`
	Type       string            `yaml:"type"`
	PHPVersion string            `yaml:"php_version"`
	Webserver  string            `yaml:"webserver_type"`
	Hooks      []string          `yaml:"hooks"`
	WebEnv     map[string]string `yaml:"web_environment"`
}

// TestLoadGlobalConfig verifies that a single YAML file is correctly loaded
// into a struct using the high-level LoadGlobalConfig function.
func TestLoadGlobalConfig(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "global_config.yaml")

	content := `
name: global-config
type: php
`
	err := os.WriteFile(configPath, []byte(content), 0644)
	require.NoError(t, err)

	var cfg TestConfig
	err = LoadGlobalConfig(configPath, &cfg)
	require.NoError(t, err)
	require.Equal(t, "global-config", cfg.Name)
	require.Equal(t, "php", cfg.Type)
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
hooks:
  - "echo original-hook"
web_environment:
  KEY1: "value1"
  KEY2: "value2"
`
	overrideContent := `
php_version: "8.3"
webserver_type: apache-fpm
hooks:
  - "echo override-hook"
web_environment:
  KEY2: "overridden-value2"
  KEY3: "value3"
`
	err := os.WriteFile(mainPath, []byte(mainContent), 0644)
	require.NoError(t, err)
	err = os.WriteFile(overridePath, []byte(overrideContent), 0644)
	require.NoError(t, err)

	var cfg TestConfig
	err = LoadProjectConfig(mainPath, []string{overridePath}, &cfg)
	require.NoError(t, err)

	require.Equal(t, "project-name", cfg.Name)    // From main
	require.Equal(t, "drupal10", cfg.Type)        // From main
	require.Equal(t, "8.3", cfg.PHPVersion)       // Overridden
	require.Equal(t, "apache-fpm", cfg.Webserver) // Overridden

	// Map keys are merged and overridden correctly.
	expectedEnv := map[string]string{
		"key1": "value1",
		"key2": "overridden-value2",
		"key3": "value3",
	}
	require.Equal(t, expectedEnv, cfg.WebEnv)

	// Slices should be appended together
	require.Equal(t, []string{"echo original-hook", "echo override-hook"}, cfg.Hooks)
}

// TestNewConfigProviderIsolation ensures that separate providers do not share state.
func TestNewConfigProviderIsolation(t *testing.T) {
	p1 := NewConfigProvider()
	p2 := NewConfigProvider()

	p1.Set("key", "value1")
	p2.Set("key", "value2")

	require.Equal(t, "value1", p1.GetString("key"))
	require.Equal(t, "value2", p2.GetString("key"))
}

// TestViperConfiguration verifies that the Viper wrapper correctly handles
// configuration defaults and set/unset operations.
func TestViperConfiguration(t *testing.T) {
	p := NewConfigProvider()

	// 1. Test Default Values
	p.SetDefault("test_key", "default_value")
	require.Equal(t, "default_value", p.GetString("test_key"))

	// 2. Test Set/override
	p.Set("some_key", "manual_value")
	require.Equal(t, "manual_value", p.GetString("some_key"))

	// 3. Test Unset
	p.Unset("some_key")
	require.Equal(t, "", p.GetString("some_key"), "Unset should remove the key's value")
}

// TestUnmarshalYamlTags verifies that the Unmarshal method correctly respects 'yaml' tags.
func TestUnmarshalYamlTags(t *testing.T) {
	p := NewConfigProvider()
	p.Set("php_version", "8.2")
	p.Set("name", "test-app")

	var cfg TestConfig
	err := p.Unmarshal(&cfg)
	require.NoError(t, err)

	require.Equal(t, "test-app", cfg.Name)
	require.Equal(t, "8.2", cfg.PHPVersion)
}

// TestUnmarshalExistingValues verifies that Unmarshal does not zero out existing fields if not in config.
func TestUnmarshalExistingValues(t *testing.T) {
	p := NewConfigProvider()

	cfg := TestConfig{
		Name: "existing-name",
	}
	err := p.Unmarshal(&cfg)
	require.NoError(t, err)

	require.Equal(t, "existing-name", cfg.Name, "Existing name should be preserved if not in config")
}

// TestViperUnmarshalDoesNotPickUpEnv verifies that Unmarshal does NOT pick up arbitrary
// environment variables.
func TestViperUnmarshalDoesNotPickUpEnv(t *testing.T) {
	t.Setenv("DDEV_TEST_PORT", "8888")

	type Config struct {
		TestPort string `yaml:"test_port"`
	}

	p := NewConfigProvider()

	var cfg Config
	err := p.Unmarshal(&cfg)
	require.NoError(t, err)

	require.Equal(t, "", cfg.TestPort, "Unmarshal should NOT pick up environment variables")
	require.Equal(t, "", p.GetString("test_port"), "GetString should NOT pick up environment variables without AutomaticEnv")
}

// TestFloatToStringPreservation verifies that YAML float values like `8.0` are
// correctly preserved as "8.0" when unmarshaled into string struct fields.
func TestFloatToStringPreservation(t *testing.T) {
	testCases := []struct {
		name            string
		yamlContent     string
		expectedVersion string
	}{
		{
			name:            "whole number float 8.0 preserved",
			yamlContent:     "database:\n  type: mysql\n  version: 8.0",
			expectedVersion: "8.0",
		},
		{
			name:            "non-whole float 10.11 preserved",
			yamlContent:     "database:\n  type: mariadb\n  version: 10.11",
			expectedVersion: "10.11",
		},
		{
			name:            "integer 17 formatted as string",
			yamlContent:     "database:\n  type: postgres\n  version: 17",
			expectedVersion: "17",
		},
		{
			name:            "quoted string 8.0 stays as-is",
			yamlContent:     "database:\n  type: mysql\n  version: \"8.0\"",
			expectedVersion: "8.0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()
			configPath := filepath.Join(tempDir, "config.yaml")

			err := os.WriteFile(configPath, []byte(tc.yamlContent), 0644)
			require.NoError(t, err)

			app := &ReproAppConfig{}
			err = LoadProjectConfig(configPath, []string{}, app)
			require.NoError(t, err)

			require.Equal(t, tc.expectedVersion, app.Database.Version)
		})
	}
}

type ReproDatabaseDesc struct {
	Type    string `yaml:"type"`
	Version string `yaml:"version"`
}

type ReproAppConfig struct {
	Database ReproDatabaseDesc `yaml:"database"`
	Name     string            `yaml:"name"`
}

func TestReproUnmarshaling(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	content := `
name: my-app
database:
  type: postgres
  version: 17
`
	err := os.WriteFile(configPath, []byte(content), 0644)
	require.NoError(t, err)

	app := &ReproAppConfig{
		Name: "default-name",
		Database: ReproDatabaseDesc{
			Type:    "mariadb",
			Version: "10.11",
		},
	}

	err = LoadProjectConfig(configPath, []string{}, app)
	require.NoError(t, err)

	require.Equal(t, "my-app", app.Name)
	require.Equal(t, "postgres", app.Database.Type)
	require.Equal(t, "17", app.Database.Version)
}
