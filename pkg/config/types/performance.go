package types

import (
	"fmt"
	"strings"
)

type Performance = string

const (
	PerformanceNone    Performance = ""
	PerformanceDefault Performance = "default"
	PerformanceOff     Performance = "off"
	PerformanceMutagen Performance = "mutagen"
	PerformanceNFS     Performance = "nfs"
)

// ValidPerformanceOptions returns a slice of valid performance options.
func ValidPerformanceOptions() []Performance {
	return []Performance{
		// PerformanceNone falls back to PerformanceDefault and should not be
		// shown as valid option and therefor is omitted from the list.
		PerformanceDefault,
		PerformanceOff,
		PerformanceMutagen,
		PerformanceNFS,
	}
}

// IsValidPerformance checks to see if the performance is valid.
func IsValidPerformance(performance string) bool {
	if performance == PerformanceNone {
		return true
	}

	for _, o := range ValidPerformanceOptions() {
		if performance == o {
			return true
		}
	}

	return false
}

// CheckValidPerformance checks to see if the performance is valid and returns
// an error in case the value is not valid.
func CheckValidPerformance(performance string) error {
	if !IsValidPerformance(performance) {
		return fmt.Errorf(
			"\"%s\" is not a valid performance option. Valid options include \"%s\"",
			performance,
			strings.Join(ValidPerformanceOptions(), "\", \""),
		)
	}

	return nil
}

// Flag definitions
const FlagPerformance = "performance"
const FlagPerformanceDefault = PerformanceNone

func FlagPerformanceDescription() string {
	return fmt.Sprintf(
		"Performance optimization strategy, possible values are \"%s\"",
		strings.Join(ValidPerformanceOptions(), "\", \""),
	)
}
