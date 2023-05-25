package main

import (
	"os"
	"path"

	"github.com/ddev/ddev/cmd/ddev/cmd"
	"github.com/ddev/ddev/pkg/amplitude"
	"github.com/ddev/ddev/pkg/config/remoteconfig"
	"github.com/ddev/ddev/pkg/config/state/storage/yaml"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/util"
)

func main() {
	defer util.TimeTrack()()

	// Initialization is currently done before via init() func somewhere while
	// creating the ddevapp. This should be cleaned up.
	amplitude.InitAmplitude()
	defer func() {
		amplitude.Flush()
		amplitude.CheckSetUp()
	}()

	// Prevent running as root for most cases
	// We really don't want ~/.ddev to have root ownership, breaks things.
	if os.Geteuid() == 0 && len(os.Args) > 1 && os.Args[1] != "hostname" {
		util.Failed("ddev is not designed to be run with root privileges, please run as normal user and without sudo")
	}

	// Create a global state to be injected later.
	state := yaml.NewState(path.Join(globalconfig.GetGlobalDdevDir(), "state.yaml"))

	// TODO for the time being this triggers the download from Github but
	// should be realized with a clean bootstrap as soon as it exists. The
	// download does not hurt here as it's done in a asynchronous call but it's
	// important to start it as early as possible to have an up to date
	// remote config at the end of the command execution.
	remoteconfig.InitGlobal(
		remoteconfig.Config{
			LocalSource: remoteconfig.LocalSource{
				Path: globalconfig.GetGlobalDdevDir(),
			},
			RemoteSource: remoteconfig.RemoteSource{
				Owner:    globalconfig.DdevGlobalConfig.RemoteConfig.Remote.Owner,
				Repo:     globalconfig.DdevGlobalConfig.RemoteConfig.Remote.Repo,
				Ref:      globalconfig.DdevGlobalConfig.RemoteConfig.Remote.Ref,
				Filepath: globalconfig.DdevGlobalConfig.RemoteConfig.Remote.Filepath,
			},
			UpdateInterval: globalconfig.DdevGlobalConfig.RemoteConfig.UpdateInterval,
			TickerDisabled: globalconfig.DdevGlobalConfig.Messages.DisableTicker,
			TickerInterval: globalconfig.DdevGlobalConfig.Messages.TickerInterval,
		},
		state,
		globalconfig.IsInternetActive,
	)

	cmd.Execute()
}
