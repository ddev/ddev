package ddevapp

import (
	"github.com/ddev/ddev/pkg/config/types"
	"github.com/ddev/ddev/pkg/globalconfig"
)

// GetPerformanceStrategy returns performance strategy config respecting defaults.
func (app *DdevApp) GetPerformanceStrategy() types.PerformanceStrategy {
	switch app.PerformanceStrategy {
	case types.PerformanceStrategyEmpty, types.PerformanceStrategyDefault:
		return globalconfig.DdevGlobalConfig.GetPerformanceStrategy()
	default:
		return app.PerformanceStrategy
	}
}

// SetPerformanceStrategy sets the performance strategy config.
func (app *DdevApp) SetPerformanceStrategy(performanceStrategy string) *DdevApp {
	if types.IsValidPerformanceStrategy(performanceStrategy) {
		app.PerformanceStrategy = performanceStrategy
	}

	return app
}

// IsNFSMountEnabled determines whether NFS is enabled.
func (app *DdevApp) IsNFSMountEnabled() bool {
	return !app.IsMutagenEnabled() && app.GetPerformanceStrategy() == types.PerformanceStrategyNFS
}
