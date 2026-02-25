package settings

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestViperConfiguration verifies that the Viper wrapper correctly handles
// configuration defaults and set/unset operations.
func TestViperConfiguration(t *testing.T) {
	// Initialize the settings system
	err := Init()
	assert.NoError(t, err)

	// 1. Test Default Values
	SetDefault("test_key", "default_value")
	assert.Equal(t, "default_value", GetString("test_key"))

	// 2. Test Set/override
	Set("some_key", "manual_value")
	assert.Equal(t, "manual_value", GetString("some_key"))

	// 3. Test Unset
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

// TestViperUnmarshalDoesNotPickUpEnv verifies that Unmarshal does NOT pick up arbitrary
// environment variables. Since AutomaticEnv is not enabled, Viper should only return
// values explicitly set via Set() or loaded from YAML files.
func TestViperUnmarshalDoesNotPickUpEnv(t *testing.T) {
	t.Setenv("DDEV_TEST_PORT", "8888")

	type Config struct {
		TestPort string `yaml:"test_port"`
	}

	p := NewConfigProvider()

	var cfg Config
	err := p.Unmarshal(&cfg)
	assert.NoError(t, err)

	assert.Equal(t, "", cfg.TestPort, "Unmarshal should NOT pick up environment variables")
	assert.Equal(t, "", p.GetString("test_port"), "GetString should NOT pick up environment variables without AutomaticEnv")
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
	err := Init()
	assert.NoError(t, err)

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	content := `
name: my-app
database:
  type: postgres
  version: 17
`
	err = os.WriteFile(configPath, []byte(content), 0644)
	assert.NoError(t, err)

	// Defaults similar to NewApp
	app := &ReproAppConfig{
		Name: "default-name",
		Database: ReproDatabaseDesc{
			Type:    "mariadb",
			Version: "10.11",
		},
	}

	err = LoadProjectConfig(configPath, []string{}, app)
	assert.NoError(t, err)

	assert.Equal(t, "my-app", app.Name)
	assert.Equal(t, "postgres", app.Database.Type)
	assert.Equal(t, "17", app.Database.Version)
}
