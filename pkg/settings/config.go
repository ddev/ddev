package settings

import (
	"github.com/spf13/viper"
)

// ConfigProvider defines the interface for configuration providers.
type ConfigProvider interface {
	GetString(key string) string
	GetInt(key string) int
	GetBool(key string) bool
	SetDefault(key string, value any)
	BindEnv(key string, envVar string)
	Set(key string, value any)
	Unmarshal(rawVal any) error
	// Add more methods as needed
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

func (vc *viperConfig) BindEnv(key string, envVar string) {
	_ = vc.v.BindEnv(key, envVar)
}

func (vc *viperConfig) Set(key string, value any) {
	vc.v.Set(key, value)
}

func (vc *viperConfig) Unmarshal(rawVal any) error {
	return vc.v.Unmarshal(rawVal)
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

	// Bind standard environment variables that don't have DDEV_ prefix
	_ = v.BindEnv("XDG_CONFIG_HOME", "XDG_CONFIG_HOME")
	_ = v.BindEnv("CAROOT", "CAROOT")

	config = &viperConfig{v: v}
	return nil
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
