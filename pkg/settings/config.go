package settings

import (
	"github.com/spf13/viper"
)

// ConfigProvider defines the interface for configuration providers.
type ConfigProvider interface {
	GetString(key string) string
	GetInt(key string) int
	GetBool(key string) bool
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

var config ConfigProvider

// Init initializes the settings system. Call this early in main().
func Init() {
	v := viper.New()
	// Example: set config file name and path
	// v.SetConfigName("config")
	// v.AddConfigPath(".")
	// v.AutomaticEnv() // read in environment variables that match

	// Optionally, read a config file
	// err := v.ReadInConfig()
	// if err != nil {
	// 	// log warning or handle error
	// }

	config = &viperConfig{v: v}
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

// Optionally, add Set, Unmarshal, etc. as needed
