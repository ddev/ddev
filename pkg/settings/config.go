package settings

// ConfigProvider defines the interface for configuration providers.
type ConfigProvider interface {
	GetString(key string) string
	GetInt(key string) int
	GetBool(key string) bool
	SetDefault(key string, value any)
	Set(key string, value any)
	Unmarshal(rawVal any) error
	Unset(key string)
	ReadConfig(path string) error
	ReadConfigFromBytes(data []byte) error
	MergeConfig(path string) error
}

// ProviderFactory defines the interface for creating ConfigProviders.
type ProviderFactory interface {
	CreateConfigProvider() ConfigProvider
	LoadProjectConfig(mainPath string, overridePaths []string, target any) error
	LoadProjectConfigFromContents(mainPath string, mainContent []byte, overrides []OverrideConfig, target any) error
}

// OverrideConfig represents a configuration override with its source path and contents.
type OverrideConfig struct {
	Path    string
	Content []byte
}

// LoadGlobalConfig loads a single global configuration file into the target struct.
// It is intentionally kept separate from LoadProjectConfig to remain fast and lightweight,
// as it does not need to support the heavy lifting of RecursiveMerge that project configurations
// require for deep map and slice overrides.
func LoadGlobalConfig(path string, target any) error {
	factory := getDefaultFactory()
	cfg := factory.CreateConfigProvider()
	if err := cfg.ReadConfig(path); err != nil {
		return err
	}
	return cfg.Unmarshal(target)
}

// LoadProjectConfig loads a main project config and merges optional overrides into the target struct.
func LoadProjectConfig(mainPath string, overridePaths []string, target any) error {
	factory := getDefaultFactory()
	return factory.LoadProjectConfig(mainPath, overridePaths, target)
}

// LoadProjectConfigFromContents loads a main project config and merges optional overrides from pre-read bytes.
func LoadProjectConfigFromContents(mainPath string, mainContent []byte, overrides []OverrideConfig, target any) error {
	factory := getDefaultFactory()
	return factory.LoadProjectConfigFromContents(mainPath, mainContent, overrides, target)
}

// NewConfigProvider returns a new ConfigProvider from the default factory.
func NewConfigProvider() ConfigProvider {
	factory := getDefaultFactory()
	return factory.CreateConfigProvider()
}

// getDefaultFactory returns the default ProviderFactory implementation.
// This centralizes the choice of the underlying provider (currently Viper).
func getDefaultFactory() ProviderFactory {
	return &ViperFactory{}
}
