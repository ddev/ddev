//go:build darwin || windows

package types

// GetPerformanceStrategyDefault returns the default performance strategy
// config depending on the OS.
func GetPerformanceStrategyDefault() PerformanceStrategy {
	return PerformanceStrategyMutagen
}
