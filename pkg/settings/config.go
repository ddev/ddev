package settings

import (
	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
)

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

// viperConfig implements ConfigProvider using Viper.
type viperConfig struct {
	v *viper.Viper
}

func (vc *viperConfig) GetString(key string) string {
	return vc.v.GetString(key)
}

func (vc *viperConfig) GetInt(key string) int {
	return vc.v.GetInt(key)
}

func (vc *viperConfig) GetBool(key string) bool {
	return vc.v.GetBool(key)
}

func (vc *viperConfig) SetDefault(key string, value interface{}) {
	vc.v.SetDefault(key, value)
}

func (vc *viperConfig) Set(key string, value interface{}) {
	vc.v.Set(key, value)
}

func (vc *viperConfig) Unset(key string) {
	// Viper doesn't have a direct Unset, so we set to nil or empty string?
	// Actually, the common way is to set it to nil.
	vc.v.Set(key, nil)
}

func (vc *viperConfig) Unmarshal(rawVal any) error {
	return vc.v.Unmarshal(rawVal, func(dc *mapstructure.DecoderConfig) {
		dc.TagName = "yaml"
	})
}

func (vc *viperConfig) ReadConfig(path string) error {
	vc.v.SetConfigFile(path)
	vc.v.SetConfigType("yaml")
	return vc.v.ReadInConfig()
}

func (vc *viperConfig) MergeConfig(path string) error {
	vc.v.SetConfigFile(path)
	vc.v.SetConfigType("yaml")
	return vc.v.MergeInConfig()
}

var config ConfigProvider

func init() {
	// Initialize with a default provider so we never have nil panics
	_ = Init()
}

// NewCleanConfigProvider returns a new isolated ConfigProvider without any bindings.
func NewCleanConfigProvider() ConfigProvider {
	v := viper.New()
	return &viperConfig{v: v}
}

// NewConfigProvider returns a new isolated ConfigProvider with standard DDEV environment bindings.
func NewConfigProvider() ConfigProvider {
	cp := NewCleanConfigProvider()
	return cp
}

// LoadGlobalConfigWithEnv loads a global configuration file into the target struct,
// also enabling environment variable overrides for standard DDEV settings.
// Deprecated: Use LoadGlobalConfig instead, which now handles environment variables.
func LoadGlobalConfigWithEnv(path string, target interface{}) error {
	return LoadGlobalConfig(path, target)
}

// LoadCleanConfig loads a configuration file into the target struct without any environment variable bindings.
// This is useful for loading map-based configs like project_list.yaml where environment bindings
// can cause type conflicts (poisoning).
func LoadCleanConfig(path string, target interface{}) error {
	cfg := NewCleanConfigProvider()
	if err := cfg.ReadConfig(path); err != nil {
		return err
	}
	return cfg.Unmarshal(target)
}

// Init initializes the settings system. Call this early in main() if you need to re-init.
func Init() error {
	v := NewConfigProvider()
	config = v
	return nil
}

// LoadGlobalConfig loads a global configuration file into the target struct.
func LoadGlobalConfig(path string, target interface{}) error {
	cfg := NewConfigProvider()
	if err := cfg.ReadConfig(path); err != nil {
		return err
	}
	return cfg.Unmarshal(target)
}

// LoadProjectConfig loads a main project config and merges optional overrides into the target struct.
func LoadProjectConfig(mainPath string, overridePaths []string, target interface{}) error {
	cfg := NewConfigProvider().(*viperConfig)
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
