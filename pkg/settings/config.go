package settings

import (
	"os"

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
	// If it's a string, also set it in the environment
	// This is important for child processes like docker-compose
	if s, ok := value.(string); ok {
		_ = os.Setenv(key, s)
	}
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
	v.SetEnvPrefix("DDEV")
	v.AutomaticEnv()
	return &viperConfig{v: v}
}

// NewConfigProvider returns a new isolated ConfigProvider with standard DDEV environment bindings.
func NewConfigProvider() ConfigProvider {
	cp := NewCleanConfigProvider()
	bindStandardGlobalEnvs(cp)
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

// bindStandardGlobalEnvs binds all the standard environment variables that DDEV uses
// and sets defaults so that Unmarshal will pick them up.
func bindStandardGlobalEnvs(v ConfigProvider) {
	// Simple strings
	binds := map[string]string{
		"router_http_port":     "DDEV_ROUTER_HTTP_PORT",
		"router_https_port":    "DDEV_ROUTER_HTTPS_PORT",
		"mailpit_http_port":    "DDEV_MAILPIT_HTTP_PORT",
		"mailpit_https_port":   "DDEV_MAILPIT_HTTPS_PORT",
		"xhgui_http_port":      "DDEV_XHGUI_HTTP_PORT",
		"xhgui_https_port":     "DDEV_XHGUI_HTTPS_PORT",
		"project_tld":          "DDEV_PROJECT_TLD",
		"traefik_monitor_port": "DDEV_TRAEFIK_MONITOR_PORT",
		"xdebug_ide_location":  "DDEV_XDEBUG_IDE_LOCATION",
		"letsencrypt_email":    "DDEV_LETSENCRYPT_EMAIL",
		"table_style":          "DDEV_TABLE_STYLE",
		"performance_mode":     "DDEV_PERFORMANCE_MODE",
		"xhprof_mode":          "DDEV_XHPROF_MODE",
		"last_started_version": "DDEV_LAST_STARTED_VERSION",
		"mkcert_caroot":        "DDEV_MKCERT_CAROOT",
		"XDG_CONFIG_HOME":      "XDG_CONFIG_HOME",
		"CAROOT":               "CAROOT",
		"CI":                   "CI",
		"CODESPACES":           "CODESPACES",
		"WSL_INTEROP":          "WSL_INTEROP",
		"WSL_DISTRO_NAME":      "WSL_DISTRO_NAME",
		"TZ":                   "TZ",
		"NO_COLOR":             "NO_COLOR",
		"GOTEST_SHORT":         "GOTEST_SHORT",
		"LANG":                 "LANG",
		"COMPOSE_PROJECT_NAME": "COMPOSE_PROJECT_NAME",
		"LOCALAPPDATA":         "LOCALAPPDATA",
		"PROGRAMFILES":         "PROGRAMFILES",
		"CODESPACE_NAME":       "CODESPACE_NAME",
		"GITHUB_CODESPACES_PORT_FORWARDING_DOMAIN": "GITHUB_CODESPACES_PORT_FORWARDING_DOMAIN",
		"MUTAGEN_DATA_DIRECTORY":                   "MUTAGEN_DATA_DIRECTORY",
		"VERSION":                                  "VERSION",
		"GITHUB_TOKEN":                             "GITHUB_TOKEN",
		"GH_TOKEN":                                 "GH_TOKEN",
		"HOME":                                     "HOME",
		"PWD":                                      "PWD",
		"USER":                                     "USER",
		"LOGNAME":                                  "LOGNAME",
		"SHELL":                                    "SHELL",
		"TERM":                                     "TERM",
		"DOCKER_CONTEXT":                           "DOCKER_CONTEXT",
		"DOCKER_HOST":                              "DOCKER_HOST",
		"TEMP":                                     "TEMP",
		"TMP":                                      "TMP",
		"USERPROFILE":                              "USERPROFILE",
		"GITHUB_ACTIONS":                           "GITHUB_ACTIONS",
	}

	for key, env := range binds {
		_ = v.BindEnv(key, env)
	}

	// Booleans
	boolBinds := map[string]string{
		"no_bind_mounts":               "DDEV_NO_BIND_MOUNTS",
		"use_hardened_images":          "DDEV_USE_HARDENED_IMAGES",
		"use_letsencrypt":              "DDEV_USE_LETSENCRYPT",
		"router_bind_all_interfaces":   "DDEV_ROUTER_BIND_ALL_INTERFACES",
		"simple_formatting":            "DDEV_SIMPLE_FORMATTING",
		"wsl2_no_windows_hosts_mgt":    "DDEV_WSL2_NO_WINDOWS_HOSTS_MGT",
		"omit_project_name_by_default": "DDEV_OMIT_PROJECT_NAME_BY_DEFAULT",
	}

	for key, env := range boolBinds {
		_ = v.BindEnv(key, env)
	}
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
