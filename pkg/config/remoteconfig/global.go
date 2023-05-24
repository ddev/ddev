// TODO inject to a global interface e.g. command factory as soon as it exists
// and remove this file.
package remoteconfig

import (
	"github.com/ddev/ddev/pkg/config/remoteconfig/types"
	statetypes "github.com/ddev/ddev/pkg/config/state/types"
)

var (
	globalRemoteConfig types.RemoteConfig
)

// InitGlobal initializes the global remote config. This is done once,
// subsequent calls do not have any effect.
func InitGlobal(config Config, stateManager statetypes.State, isInternetActive func() bool) types.RemoteConfig {
	if globalRemoteConfig == nil {
		globalRemoteConfig = New(&config, stateManager, isInternetActive)
	}

	return globalRemoteConfig
}

// GetGlobal returns the global remote config. InitGlobal must be
// called in advance or the function will panic.
func GetGlobal() types.RemoteConfig {
	if globalRemoteConfig == nil {
		panic("error remoteconfig.InitGlobal was not called before accessing remote config")
	}

	return globalRemoteConfig
}
