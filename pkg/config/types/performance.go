package types

import (
	"fmt"
	"strings"
)

type PerformanceStrategy = string

const (
	PerformanceStrategyEmpty   PerformanceStrategy = ""
	PerformanceStrategyDefault PerformanceStrategy = "default"
	PerformanceStrategyNone    PerformanceStrategy = "none"
	PerformanceStrategyMutagen PerformanceStrategy = "mutagen"
	PerformanceStrategyNFS     PerformanceStrategy = "nfs"
)

// ValidPerformanceStrategyOptions returns a slice of valid performance
// strategy options.
func ValidPerformanceStrategyOptions() []PerformanceStrategy {
	return []PerformanceStrategy{
		// PerformanceStrategyEmpty falls back to PerformanceStrategyDefault
		// and should not be shown as valid option and therefor is omitted
		// from the list.
		PerformanceStrategyDefault,
		PerformanceStrategyNone,
		PerformanceStrategyMutagen,
		PerformanceStrategyNFS,
	}
}

// IsValidPerformanceStrategy checks to see if the given performance strategy
// option is valid.
func IsValidPerformanceStrategy(performanceStrategy string) bool {
	if performanceStrategy == PerformanceStrategyEmpty {
		return true
	}

	for _, o := range ValidPerformanceStrategyOptions() {
		if performanceStrategy == o {
			return true
		}
	}

	return false
}

// CheckValidPerformance checks to see if the given performance strategy option
// is valid and returns an error in case the value is not valid.
func CheckValidPerformanceStrategy(performanceStrategy string) error {
	if !IsValidPerformanceStrategy(performanceStrategy) {
		return fmt.Errorf(
			"\"%s\" is not a valid performance strategy option. Valid options include \"%s\"",
			performanceStrategy,
			strings.Join(ValidPerformanceStrategyOptions(), "\", \""),
		)
	}

	return nil
}

// Flag definitions
const FlagPerformanceStrategyName = "performance-strategy"
const FlagPerformanceStrategyDefault = PerformanceStrategyEmpty

func FlagPerformanceDescription() string {
	return fmt.Sprintf(
		"Performance optimization strategy, possible values are \"%s\"",
		strings.Join(ValidPerformanceStrategyOptions(), "\", \""),
	)
}
