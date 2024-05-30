package ddevapp

import (
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/util"
	"github.com/ddev/ddev/pkg/versionconstants"
)

func PowerOff() {
	apps, err := GetProjects(true)
	if err != nil {
		util.Failed("Failed to get project(s): %v", err)
	}

	// Remove any custom certs that may have been added
	// along with all Traefik configuration.
	_, _, err = dockerutil.RunSimpleContainer(versionconstants.BusyboxImage, "poweroff-"+util.RandString(6), []string{"sh", "-c", "rm -rf /mnt/ddev-global-cache/custom_certs/* /mnt/ddev-global-cache/traefik/*"}, []string{}, []string{}, []string{"ddev-global-cache" + ":/mnt/ddev-global-cache"}, "", true, false, map[string]string{"com.ddev.site-name": ""}, nil, &dockerutil.NoHealthCheck)
	if err != nil {
		util.Warning("Failed removing custom certs/traefik configuration: %v", err)
	}

	// Iterate through the list of apps built above, stopping each one.
	for _, app := range apps {
		if err := app.Stop(false, false); err != nil {
			util.Failed("Failed to stop project %s: \n%v", app.GetName(), err)
		}
		util.Success("Project %s has been stopped.", app.GetName())
	}

	// Any straggling containers that have label "com.ddev.site-name" should be removed.
	containers, err := dockerutil.FindContainersByLabels(map[string]string{"com.ddev.site-name": ""})

	if err == nil {
		for _, c := range containers {
			err = dockerutil.RemoveContainer(c.ID)
			if err != nil {
				util.Warning("Failed to remove container %s", c.Names[0])
			}
		}
	} else {
		util.Warning("Unable to run client.ListContainers(): %v", err)
	}

	StopMutagenDaemon("")

	if err := RemoveSSHAgentContainer(); err != nil {
		util.Error("Failed to remove ddev-ssh-agent: %v", err)
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
