//go:build !(darwin || windows)

package types

// getPerformanceDefault returns the default performance config depending on
// the OS.
func GetPerformanceDefault() Performance {
	return PerformanceOff
}
