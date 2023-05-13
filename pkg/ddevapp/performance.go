package ddevapp

import (
	"github.com/ddev/ddev/pkg/config/types"
	"github.com/ddev/ddev/pkg/globalconfig"
)

// GetPerformance returns performance config respecting defaults.
func (app *DdevApp) GetPerformance() types.Performance {
	switch app.Performance {
	case types.PerformanceNone, types.PerformanceDefault:
		return globalconfig.DdevGlobalConfig.GetPerformance()
	default:
		return app.Performance
	}
}

// IsNFSMountEnabled determines whether NFS is enabled.
func (app *DdevApp) IsNFSMountEnabled() bool {
	return !app.IsMutagenEnabled() && app.GetPerformance() == types.PerformanceNFS
}
