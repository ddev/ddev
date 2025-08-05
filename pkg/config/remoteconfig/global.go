// TODO: Inject to a global interface e.g. command factory as soon as it exists
// and remove this file.
package remoteconfig

import (
	"github.com/ddev/ddev/pkg/config/remoteconfig/types"
	statetypes "github.com/ddev/ddev/pkg/config/state/types"
)

var (
	globalRemoteConfig       types.RemoteConfig
	globalSponsorshipManager types.SponsorshipManager
)

// InitGlobal initializes the global remote config. This is done once,
// subsequent calls do not have any effect.
func InitGlobal(config Config, stateManager statetypes.State, isInternetActive func() bool) types.RemoteConfig {
	if globalRemoteConfig == nil {
		globalRemoteConfig = New(&config, stateManager, isInternetActive)
	}

	return globalRemoteConfig
}

// GetGlobal returns the global remote config. Returns nil if InitGlobal
// was not called in advance.
func GetGlobal() types.RemoteConfig {
	return globalRemoteConfig
}

// InitGlobalSponsorship initializes the global sponsorship manager using a direct URL. This is done once,
// subsequent calls do not have any effect.
func InitGlobalSponsorship(localPath string, stateManager statetypes.State, isInternetActive func() bool, updateInterval int, url string) types.SponsorshipManager {
	if globalSponsorshipManager == nil {
		globalSponsorshipManager = NewSponsorshipManager(localPath, stateManager, isInternetActive, updateInterval, url)
	}

	return globalSponsorshipManager
}

// GetGlobalSponsorship returns the global sponsorship manager. Returns nil if
// InitGlobalSponsorship was not called in advance.
func GetGlobalSponsorship() types.SponsorshipManager {
	return globalSponsorshipManager
}
