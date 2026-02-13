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
	BindEnv(key string, envVar string) error
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

func (vc *viperConfig) SetDefault(key string, value any) {
	vc.v.SetDefault(key, value)
}

func (vc *viperConfig) BindEnv(key string, envVar string) error {
	return vc.v.BindEnv(key, envVar)
}

func (vc *viperConfig) Set(key string, value any) {
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

// Init initializes the settings system. Call this early in main() if you need to re-init.
func Init() error {
	v := viper.New()
	v.SetEnvPrefix("DDEV")
	v.AutomaticEnv()

	// Bind standard environment variables that DDEV uses
	_ = v.BindEnv("XDG_CONFIG_HOME", "XDG_CONFIG_HOME")
	_ = v.BindEnv("CAROOT", "CAROOT")
	_ = v.BindEnv("CI", "CI")
	_ = v.BindEnv("CODESPACES", "CODESPACES")
	_ = v.BindEnv("WSL_INTEROP", "WSL_INTEROP")
	_ = v.BindEnv("WSL_DISTRO_NAME", "WSL_DISTRO_NAME")
	_ = v.BindEnv("TZ", "TZ")
	_ = v.BindEnv("NO_COLOR", "NO_COLOR")
	_ = v.BindEnv("GOTEST_SHORT", "GOTEST_SHORT")
	_ = v.BindEnv("LANG", "LANG")
	_ = v.BindEnv("COMPOSE_PROJECT_NAME", "COMPOSE_PROJECT_NAME")
	_ = v.BindEnv("LOCALAPPDATA", "LOCALAPPDATA")
	_ = v.BindEnv("PROGRAMFILES", "PROGRAMFILES")
	_ = v.BindEnv("CODESPACE_NAME", "CODESPACE_NAME")
	_ = v.BindEnv("GITHUB_CODESPACES_PORT_FORWARDING_DOMAIN", "GITHUB_CODESPACES_PORT_FORWARDING_DOMAIN")
	_ = v.BindEnv("MUTAGEN_DATA_DIRECTORY", "MUTAGEN_DATA_DIRECTORY")
	_ = v.BindEnv("VERSION", "VERSION")
	_ = v.BindEnv("GITHUB_TOKEN", "GITHUB_TOKEN")
	_ = v.BindEnv("GH_TOKEN", "GH_TOKEN")
	_ = v.BindEnv("HOME", "HOME")
	_ = v.BindEnv("PWD", "PWD")
	_ = v.BindEnv("USER", "USER")
	_ = v.BindEnv("LOGNAME", "LOGNAME")
	_ = v.BindEnv("SHELL", "SHELL")
	_ = v.BindEnv("TERM", "TERM")
	_ = v.BindEnv("DOCKER_CONTEXT", "DDEV_DOCKER_CONTEXT", "DOCKER_CONTEXT")
	_ = v.BindEnv("DOCKER_HOST", "DDEV_DOCKER_HOST", "DOCKER_HOST")
	_ = v.BindEnv("TEMP", "TEMP")
	_ = v.BindEnv("TMP", "TMP")
	_ = v.BindEnv("USERPROFILE", "USERPROFILE")
	_ = v.BindEnv("GITHUB_ACTIONS", "GITHUB_ACTIONS")

	config = &viperConfig{v: v}
	return nil
}

// NewConfigProvider returns a new isolated ConfigProvider.
func NewConfigProvider() ConfigProvider {
	return &viperConfig{v: viper.New()}
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

func SetDefault(key string, value any) {
	config.SetDefault(key, value)
}

func BindEnv(key string, envVar string) {
	config.BindEnv(key, envVar)
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
