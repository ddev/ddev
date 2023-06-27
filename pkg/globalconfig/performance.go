package globalconfig

import (
	"github.com/ddev/ddev/pkg/config/types"
)

// GetPerformanceStrategy returns the performance strategy config respecting
// defaults.
func (c *GlobalConfig) GetPerformanceStrategy() types.PerformanceStrategy {
	switch c.PerformanceStrategy {
	case types.PerformanceStrategyEmpty, types.PerformanceStrategyDefault:
		return types.GetPerformanceStrategyDefault()
	default:
		return c.PerformanceStrategy
	}
}

// SetPerformanceStrategy sets the performance strategy config.
func (c *GlobalConfig) SetPerformanceStrategy(performanceStrategy string) *GlobalConfig {
	if types.IsValidPerformanceStrategy(performanceStrategy) {
		c.PerformanceStrategy = performanceStrategy
	}

	return c
}

// IsMutagenEnabled returns true if Mutagen is enabled.
func (c *GlobalConfig) IsMutagenEnabled() bool {
	return c.GetPerformanceStrategy() == types.PerformanceStrategyMutagen
}

// IsNFSMountEnabled returns true if NFS is enabled.
func (c *GlobalConfig) IsNFSMountEnabled() bool {
	return c.GetPerformanceStrategy() == types.PerformanceStrategyNFS
}
