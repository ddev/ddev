package ddevapp

import (
	"github.com/ddev/ddev/pkg/config/types"
	"github.com/ddev/ddev/pkg/globalconfig"
)

// GetPerformanceMode returns performance mode config respecting defaults.
func (app *DdevApp) GetPerformanceMode() types.PerformanceMode {
	switch app.PerformanceMode {
	case types.PerformanceModeEmpty, types.PerformanceModeGlobal:
		return globalconfig.DdevGlobalConfig.GetPerformanceMode()
	default:
		return app.PerformanceMode
	}
}

// SetPerformanceMode sets the performance mode config.
func (app *DdevApp) SetPerformanceMode(performanceMode string) *DdevApp {
	if types.IsValidPerformanceMode(performanceMode, types.ConfigTypeProject) {
		app.PerformanceMode = performanceMode
	}

	return app
}

// IsNFSMountEnabled determines whether NFS is enabled.
func (app *DdevApp) IsNFSMountEnabled() bool {
	return !app.IsMutagenEnabled() && app.GetPerformanceMode() == types.PerformanceModeNFS
}
