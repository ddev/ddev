package globalconfig

import (
	"github.com/ddev/ddev/pkg/config/types"
)

// GetPerformance returns the performance config respecting defaults.
func (c *GlobalConfig) GetPerformance() types.Performance {
	switch c.Performance {
	case types.PerformanceNone, types.PerformanceDefault:
		return types.GetPerformanceDefault()
	default:
		return c.Performance
	}
}

// IsMutagenEnabled returns true if Mutagen is enabled.
func (c *GlobalConfig) IsMutagenEnabled() bool {
	return c.GetPerformance() == types.PerformanceMutagen
}

// IsNFSMountEnabled returns true if NFS is enabled.
func (c *GlobalConfig) IsNFSMountEnabled() bool {
	return c.GetPerformance() == types.PerformanceNFS
}
