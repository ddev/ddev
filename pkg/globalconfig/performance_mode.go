package globalconfig

import (
	"github.com/ddev/ddev/pkg/config/types"
)

// GetPerformanceMode returns the performance mode config respecting
// defaults.
func (c *GlobalConfig) GetPerformanceMode() types.PerformanceMode {
	switch c.PerformanceMode {
	case types.PerformanceModeEmpty:
		return types.GetPerformanceModeDefault()
	default:
		return c.PerformanceMode
	}
}

// SetPerformanceMode sets the performance mode config.
func (c *GlobalConfig) SetPerformanceMode(performanceMode string) *GlobalConfig {
	if types.IsValidPerformanceMode(performanceMode, types.ConfigTypeGlobal) {
		c.PerformanceMode = performanceMode
	}

	return c
}

// IsMutagenEnabled returns true if Mutagen is enabled.
func (c *GlobalConfig) IsMutagenEnabled() bool {
	return c.GetPerformanceMode() == types.PerformanceModeMutagen
}

// IsNFSMountEnabled returns true if NFS is enabled.
func (c *GlobalConfig) IsNFSMountEnabled() bool {
	return c.GetPerformanceMode() == types.PerformanceModeNFS
}
