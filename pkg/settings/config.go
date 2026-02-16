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
	MergeConfig(path string) error
}

var config ConfigProvider

func init() {
	// Initialize with a default provider so we never have nil panics
	_ = Init()
}

// Init initializes the settings system. Call this early in main() if you need to re-init.
func Init() error {
	v := NewConfigProvider()
	config = v
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

// LoadGlobalConfigWithEnv loads a global configuration file into the target struct,
// also enabling environment variable overrides for standard DDEV settings.
// Deprecated: Use LoadGlobalConfig instead, which now handles environment variables.
func LoadGlobalConfigWithEnv(path string, target any) error {
	return LoadGlobalConfig(path, target)
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

func SetDefault(key string, value interface{}) {
	config.SetDefault(key, value)
}

func Set(key string, value interface{}) {
	config.Set(key, value)
}
