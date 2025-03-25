package types

import (
	"fmt"
	"strings"
)

type PerformanceMode = string

const (
	PerformanceModeEmpty   PerformanceMode = ""
	PerformanceModeGlobal  PerformanceMode = "global"
	PerformanceModeNone    PerformanceMode = "none"
	PerformanceModeMutagen PerformanceMode = "mutagen"
	PerformanceModeNFS     PerformanceMode = "nfs"
)

// ValidPerformanceModeOptions returns a slice of valid performance mode
// options for the project config or global config if global is true.
func ValidPerformanceModeOptions(configType ConfigType) []PerformanceMode {
	switch configType {
	case ConfigTypeGlobal:
		return []PerformanceMode{
			PerformanceModeNone,
			PerformanceModeMutagen,
			PerformanceModeNFS,
		}
	case ConfigTypeProject:
		return []PerformanceMode{
			PerformanceModeGlobal,
			PerformanceModeNone,
			PerformanceModeMutagen,
			PerformanceModeNFS,
		}
	default:
		panic(fmt.Errorf("invalid ConfigType: %v", configType))
	}
}

// IsValidPerformanceMode checks to see if the given performance mode
// option is valid.
func IsValidPerformanceMode(performanceMode string, configType ConfigType) bool {
	if performanceMode == PerformanceModeEmpty {
		return true
	}

	for _, o := range ValidPerformanceModeOptions(configType) {
		if performanceMode == o {
			return true
		}
	}

	return false
}

// CheckValidPerformance checks to see if the given performance mode option
// is valid and returns an error in case the value is not valid.
func CheckValidPerformanceMode(performanceMode string, configType ConfigType) error {
	if !IsValidPerformanceMode(performanceMode, configType) {
		return fmt.Errorf(
			"\"%s\" is not a valid performance mode option. Valid options include \"%s\"",
			performanceMode,
			strings.Join(ValidPerformanceModeOptions(configType), "\", \""),
		)
	}

	return nil
}

// Flag definitions
const FlagPerformanceModeName = "performance-mode"
const FlagPerformanceModeDefault = PerformanceModeEmpty

func FlagPerformanceModeDescription(configType ConfigType) string {
	return fmt.Sprintf(
		"Performance optimization mode, possible values are \"%s\"",
		strings.Join(ValidPerformanceModeOptions(configType), "\", \""),
	)
}

const FlagPerformanceModeResetName = "performance-mode-reset"

func FlagPerformanceModeResetDescription(configType ConfigType) string {
	switch configType {
	case ConfigTypeGlobal:
		return "Reset performance mode to operating system default"
	case ConfigTypeProject:
		return "Reset performance mode to global configuration"
	default:
		panic(fmt.Errorf("invalid ConfigType: %v", configType))
	}
}
