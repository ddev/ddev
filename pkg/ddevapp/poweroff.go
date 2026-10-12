package ddevapp

import (
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/util"
)

func PowerOff() {
	apps, err := GetProjects(true)
	if err != nil {
		util.Warning("Failed to get project(s): %v", err)
	}

	// Iterate through the list of apps built above, stopping each one.
	for _, app := range apps {
		if err := app.Stop(false, false); err != nil {
			util.Warning("Failed to stop project %s: \n%v", app.GetName(), err)
		} else {
			util.Success("Project %s has been stopped.", app.GetName())
		}
	}

	// Any straggling containers that have label "com.ddev.site-name" should be removed.
	containers, err := dockerutil.FindContainersByLabels(map[string]string{"com.ddev.site-name": ""})

	if err == nil {
		for _, c := range containers {
			err = dockerutil.RemoveContainer(c.ID)
			if err != nil {
				util.Warning("Failed to remove container %+v", c)
			}
		}
	} else {
		util.Warning("Unable to run client.ListContainers(): %v", err)
	}

	StopMutagenDaemon("")

	// A concurrent start could otherwise lose its router config or network mid-setup.
	defer dockerutil.AcquireGlobalLock("poweroff cleanup")()

	// Clean up Traefik staging directories after all projects are stopped
	// This prevents issues when downgrading DDEV versions
	if err := CleanupGlobalTraefikStaging(); err != nil {
		util.Warning("Failed to clean up Traefik staging directories: %v", err)
	}

	if err := removeSSHAgentContainer(); err != nil {
		util.Warning("Failed to remove ddev-ssh-agent: %v", err)
	}
	if err := removeRouterContainer(); err != nil {
		util.Warning("Failed to remove ddev-router: %v", err)
	}

	// Remove global DDEV default network
	dockerutil.RemoveNetworkWithWarningOnError(dockerutil.NetName)

	// Remove all networks created with DDEV
	removals, err := dockerutil.FindNetworksWithLabel("com.ddev.platform")
	if err == nil {
		for _, network := range removals {
			dockerutil.RemoveNetworkWithWarningOnError(network.Name)
		}
	} else {
		util.Warning("Unable to run dockerutil.FindNetworksWithLabel(): %v", err)
	}
}
