package settings

// ConfigProvider defines the interface for configuration providers.
type ConfigProvider interface {
	GetString(key string) string
	GetInt(key string) int
	GetBool(key string) bool
	SetDefault(key string, value any)
	BindEnv(key string, envVar string) error
	Set(key string, value any)
	Unmarshal(rawVal any) error
	Unset(key string)
	ReadConfig(path string) error
	MergeConfig(path string) error
}

// ProviderFactory defines the interface for creating ConfigProviders.
type ProviderFactory interface {
	CreateConfigProvider(delimiter string) ConfigProvider
	CreateCleanConfigProvider(delimiter string) ConfigProvider
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
	config = factory.CreateConfigProvider("")
	return nil
}

// LoadGlobalConfig loads a global configuration file into the target struct.
func LoadGlobalConfig(path string, target any) error {
	cfg := NewConfigProvider()
	if err := cfg.ReadConfig(path); err != nil {
		return err
	}
	return cfg.Unmarshal(target)
}

// LoadProjectConfig loads a main project config and merges optional overrides into the target struct.
func LoadProjectConfig(mainPath string, overridePaths []string, target any) error {
	cfg := NewConfigProvider()
	if err := cfg.ReadConfig(mainPath); err != nil {
		return err
	}

	for _, path := range overridePaths {
		if err := cfg.MergeConfig(path); err != nil {
			return err
		}
	}

	return cfg.Unmarshal(target)
}

// LoadCleanConfig loads a configuration file into the target struct without any environment variable bindings.
// This is useful for loading map-based configs like project_list.yaml where environment bindings
// can cause type conflicts (poisoning).
func LoadCleanConfig(path string, target any) error {
	cfg := NewCleanConfigProvider()
	if err := cfg.ReadConfig(path); err != nil {
		return err
	}
	return cfg.Unmarshal(target)
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

func SetDefault(key string, value any) {
	config.SetDefault(key, value)
}

func BindEnv(key string, envVar string) {
	_ = config.BindEnv(key, envVar)
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
	return factory.CreateConfigProvider("")
}

// NewCleanConfigProvider returns a new CleanConfigProvider from the configured factory.
func NewCleanConfigProvider() ConfigProvider {
	return factory.CreateCleanConfigProvider("")
}

// NewProjectListConfigProvider returns a new ProjectListConfigProvider from the configured factory.
func NewProjectListConfigProvider() ConfigProvider {
	return factory.CreateCleanConfigProvider("::")
}

// LoadProjectListConfig loads a configuration file into the target struct using a custom key delimiter.
// This is specifically for project_list.yaml where project names (keys) can contain dots.
func LoadProjectListConfig(path string, target any) error {
	cfg := NewProjectListConfigProvider()
	if err := cfg.ReadConfig(path); err != nil {
		return err
	}
	return cfg.Unmarshal(target)
}
