package types

import (
	"fmt"
	"strings"
)

type PerformanceMode = string

const (
	PerformanceModeEmpty   PerformanceMode = ""
	PerformanceModeDefault PerformanceMode = "default"
	PerformanceModeNone    PerformanceMode = "none"
	PerformanceModeMutagen PerformanceMode = "mutagen"
	PerformanceModeNFS     PerformanceMode = "nfs"
)

// ValidPerformanceModeOptions returns a slice of valid performance
// mode options.
func ValidPerformanceModeOptions() []PerformanceMode {
	return []PerformanceMode{
		// PerformanceModeEmpty falls back to PerformanceModeDefault
		// and should not be shown as valid option and therefor is omitted
		// from the list.
		PerformanceModeDefault,
		PerformanceModeNone,
		PerformanceModeMutagen,
		PerformanceModeNFS,
	}
}

// IsValidPerformanceMode checks to see if the given performance mode
// option is valid.
func IsValidPerformanceMode(performanceMode string) bool {
	if performanceMode == PerformanceModeEmpty {
		return true
	}

	for _, o := range ValidPerformanceModeOptions() {
		if performanceMode == o {
			return true
		}
	}

	return false
}

// CheckValidPerformance checks to see if the given performance mode option
// is valid and returns an error in case the value is not valid.
func CheckValidPerformanceMode(performanceMode string) error {
	if !IsValidPerformanceMode(performanceMode) {
		return fmt.Errorf(
			"\"%s\" is not a valid performance mode option. Valid options include \"%s\"",
			performanceMode,
			strings.Join(ValidPerformanceModeOptions(), "\", \""),
		)
	}

	return nil
}

// Flag definitions
const FlagPerformanceModeName = "performance-mode"
const FlagPerformanceModeDefault = PerformanceModeEmpty

func FlagPerformanceDescription() string {
	return fmt.Sprintf(
		"Performance optimization mode, possible values are \"%s\"",
		strings.Join(ValidPerformanceModeOptions(), "\", \""),
	)
}
