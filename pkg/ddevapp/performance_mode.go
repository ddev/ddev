package ddevapp

import (
	"github.com/ddev/ddev/pkg/config/types"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/util"
)

// GetPerformanceMode returns performance mode config respecting defaults.
func (app *DdevApp) GetPerformanceMode() types.PerformanceMode {
	mode := app.configuredPerformanceMode()

	// Apple Container backs each named volume with a block image that only one
	// attachment can hold, and Mutagen mounts project_mutagen at both /var/www and
	// /tmp/project_mutagen in the web container — two attachments of one volume in a
	// single container. The container then fails to boot with a Virtualization
	// framework "storage device attachment is invalid" error, so force Mutagen off
	// instead of letting the project fail. WarningOnce because this is called many
	// times per command.
	if mode == types.PerformanceModeMutagen && dockerutil.IsSocktainer() {
		util.WarningOnce("Mutagen is not supported on the socktainer/Apple Container provider; using performance_mode=none for this project. Apple Container cannot attach the project_mutagen volume twice in the web container.")
		return types.PerformanceModeNone
	}

	return mode
}

// configuredPerformanceMode is the mode as configured, before any
// provider-specific override.
func (app *DdevApp) configuredPerformanceMode() types.PerformanceMode {
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
