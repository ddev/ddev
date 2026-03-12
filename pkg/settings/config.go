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
	LoadProjectConfigFromContents(mainPath string, mainContent []byte, overrides map[string][]byte, target any) error
}

var (
	config  ConfigProvider
	factory ProviderFactory
)

func init() {
	// Initialize with a default provider so we never have nil panics
	_ = Init()
}

// Init initializes the settings system. Call this early in main() if you need to re-init.
func Init() error {
	factory = &ViperFactory{}
	config = factory.CreateConfigProvider()

	return nil
}

// LoadGlobalConfig loads a single global configuration file into the target struct.
// It is intentionally kept separate from LoadProjectConfig to remain fast and lightweight,
// as it does not need to support the heavy lifting of RecursiveMerge that project configurations
// require for deep map and slice overrides.
func LoadGlobalConfig(path string, target any) error {
	cfg := NewConfigProvider()
	if err := cfg.ReadConfig(path); err != nil {
		return err
	}
	return cfg.Unmarshal(target)
}

// LoadProjectConfig loads a main project config and merges optional overrides into the target struct.
func LoadProjectConfig(mainPath string, overridePaths []string, target any) error {
	return factory.LoadProjectConfig(mainPath, overridePaths, target)
}

// LoadProjectConfigFromContents loads a main project config and merges optional overrides from pre-read bytes.
func LoadProjectConfigFromContents(mainPath string, mainContent []byte, overrides map[string][]byte, target any) error {
	return factory.LoadProjectConfigFromContents(mainPath, mainContent, overrides, target)
}

// GetString returns the string value for a key using the current config provider.
func GetString(key string) string {
	return config.GetString(key)
}

func GetInt(key string) int {
	return config.GetInt(key)
}

func GetBool(key string) bool {
	return config.GetBool(key)
}

func SetDefault(key string, value interface{}) {
	config.SetDefault(key, value)
}

func Set(key string, value any) {
	config.Set(key, value)
}

func Unmarshal(rawVal any) error {
	return config.Unmarshal(rawVal)
}

// Unset unsets a key in the global configuration.
func Unset(key string) {
	config.Unset(key)
}

// NewConfigProvider returns a new ConfigProvider from the configured factory.
func NewConfigProvider() ConfigProvider {
	return factory.CreateConfigProvider()
}
