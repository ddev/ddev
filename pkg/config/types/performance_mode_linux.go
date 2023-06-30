//go:build !(darwin || windows)

package types

// GetPerformanceModeDefault returns the default performance mode config
// depending on the OS.
func GetPerformanceModeDefault() PerformanceMode {
	return PerformanceModeNone
}
