package settings

import (
	"fmt"
	"math"
	"reflect"

	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
)

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

func (vc *viperConfig) Set(key string, value any) {
	vc.v.Set(key, value)
}

func (vc *viperConfig) Unset(key string) {
	// Viper doesn't have a direct Unset, so we set to nil.
	vc.v.Set(key, nil)
}

func (vc *viperConfig) Unmarshal(rawVal any) error {
	return vc.v.Unmarshal(rawVal, func(dc *mapstructure.DecoderConfig) {
		dc.TagName = "yaml"
		dc.WeaklyTypedInput = true
		// Preserve float-to-string precision (e.g. YAML 8.0 → "8.0", not "8")
		dc.DecodeHook = floatToStringHook()
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

// floatToStringHook returns a mapstructure DecodeHookFunc that preserves
// decimal representation when converting floats to strings.
// YAML parses unquoted values like `8.0` as float64(8), but when that value
// targets a string field (e.g. database version), mapstructure's weak typing
// would format it as "8". This hook ensures "8.0" is preserved by detecting
// whole-number floats and appending ".0".
func floatToStringHook() mapstructure.DecodeHookFunc {
	return func(from reflect.Type, to reflect.Type, data any) (any, error) {
		if from.Kind() == reflect.Float64 && to.Kind() == reflect.String {
			f := data.(float64)
			// If the float is a whole number (e.g. 8.0), format with one decimal
			// place to preserve the ".0" that was in the original YAML.
			// Non-whole floats (e.g. 10.11) are formatted normally.
			if f == math.Trunc(f) {
				return fmt.Sprintf("%.1f", f), nil
			}
			return fmt.Sprintf("%g", f), nil
		}
		return data, nil
	}
}

// ViperFactory implements ProviderFactory using Viper.
type ViperFactory struct{}

// CreateCleanConfigProvider returns a new isolated ConfigProvider without any bindings.
func (vf *ViperFactory) CreateCleanConfigProvider(delimiter string) ConfigProvider {
	if delimiter == "" {
		delimiter = "."
	}
	v := viper.NewWithOptions(viper.KeyDelimiter(delimiter))
	return &viperConfig{v: v}
}

// CreateConfigProvider returns a new isolated ConfigProvider with standard DDEV environment bindings.
func (vf *ViperFactory) CreateConfigProvider(delimiter string) ConfigProvider {
	cp := vf.CreateCleanConfigProvider(delimiter)
	return cp
}

// LoadProjectConfig loads a main project config and merges optional overrides into the target struct.
func (vf *ViperFactory) LoadProjectConfig(mainPath string, overridePaths []string, target any) error {
	// First load the main config into a map
	mainMap := make(map[string]any)
	cfg := vf.CreateConfigProvider("")
	if err := cfg.ReadConfig(mainPath); err != nil {
		return err
	}
	if err := cfg.Unmarshal(&mainMap); err != nil {
		return err
	}

	// Now load and merge each override
	for _, path := range overridePaths {
		overrideCfg := vf.CreateConfigProvider("")
		if err := overrideCfg.ReadConfig(path); err != nil {
			return err
		}

		overrideMap := make(map[string]any)
		if err := overrideCfg.Unmarshal(&overrideMap); err != nil {
			return err
		}

		// Merge the override map into the main map using custom logic
		if err := RecursiveMerge(mainMap, overrideMap); err != nil {
			return err
		}
	}

	// Decode the merged map into the target struct, using the float-to-string
	// hook to preserve decimal representations (e.g. database version "8.0").
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		TagName:          "yaml",
		WeaklyTypedInput: true,
		DecodeHook:       floatToStringHook(),
		Result:           target,
	})
	if err != nil {
		return err
	}

	return decoder.Decode(mainMap)
}

// RecursiveMerge merges src into dst.
// Maps are merged recursively. Slices are appended. Values are overridden.
func RecursiveMerge(dst, src map[string]any) error {
	for key, srcVal := range src {
		if dstVal, ok := dst[key]; ok {
			// If both are maps, recurse
			srcMap, srcIsMap := srcVal.(map[string]any)
			dstMap, dstIsMap := dstVal.(map[string]any)
			if srcIsMap && dstIsMap {
				if err := RecursiveMerge(dstMap, srcMap); err != nil {
					return err
				}
				continue
			}

			// If both are slices, append
			srcSlice, srcIsSlice := srcVal.([]any)
			dstSlice, dstIsSlice := dstVal.([]any)
			if srcIsSlice && dstIsSlice {
				dst[key] = append(dstSlice, srcSlice...)
				continue
			}
		}
		// Otherwise overwrite (including false booleans, empty strings, etc.)
		dst[key] = srcVal
	}
	return nil
}
