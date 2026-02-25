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

// MockConfigProvider is a mock implementation of ConfigProvider for testing factory swaps.
type MockConfigProvider struct {
	data map[string]any
}

func (m *MockConfigProvider) GetString(key string) string {
	if val, ok := m.data[key]; ok {
		return val.(string)
	}
	return ""
}
func (m *MockConfigProvider) GetInt(key string) int {
	if val, ok := m.data[key]; ok {
		return val.(int)
	}
	return 0
}
func (m *MockConfigProvider) GetBool(key string) bool {
	if val, ok := m.data[key]; ok {
		return val.(bool)
	}
	return false
}
func (m *MockConfigProvider) SetDefault(key string, value any)        { m.data[key] = value }
func (m *MockConfigProvider) BindEnv(key string, envVar string) error { return nil }
func (m *MockConfigProvider) Set(key string, value any)               { m.data[key] = value }
func (m *MockConfigProvider) Unmarshal(rawVal any) error              { return nil }
func (m *MockConfigProvider) Unset(key string)                        { delete(m.data, key) }
func (m *MockConfigProvider) ReadConfig(path string) error            { return nil }
func (m *MockConfigProvider) MergeConfig(path string) error           { return nil }

// MockFactory is a mock implementation of ProviderFactory.
type MockFactory struct{}

func (f *MockFactory) CreateConfigProvider(delimiter string) ConfigProvider {
	return &MockConfigProvider{data: make(map[string]any)}
}

func (f *MockFactory) CreateCleanConfigProvider(delimiter string) ConfigProvider {
	return &MockConfigProvider{data: make(map[string]any)}
}

func (f *MockFactory) LoadProjectConfig(mainPath string, overridePaths []string, target any) error {
	return nil
}

// TestAbstractFactorySwap verifies that we can swap the underlying factory
// and that the global functions delegate to it.
func TestAbstractFactorySwap(t *testing.T) {
	// Backup original factory and config
	origFactory := factory
	origConfig := config
	defer func() {
		factory = origFactory
		config = origConfig
	}()

	// Inject MockFactory
	factory = &MockFactory{}
	// Re-init global config with the new factory
	config = factory.CreateConfigProvider("")

	// Verify that NewConfigProvider returns a MockConfigProvider
	provider := NewConfigProvider()
	_, ok := provider.(*MockConfigProvider)
	assert.True(t, ok, "NewConfigProvider should return a MockConfigProvider when MockFactory is injected")

	// Verify that the global config is also using the mock
	Set("mock_key", "mock_value")
	assert.Equal(t, "mock_value", GetString("mock_key"))
}
