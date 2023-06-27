//go:build !(darwin || windows)

package types

// GetPerformanceDefault returns the default performance strategy config
// depending on the OS.
func GetPerformanceStrategyDefault() PerformanceStrategy {
	return PerformanceStrategyNone
}
