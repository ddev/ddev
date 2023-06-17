package ddevapp

import (
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/util"
	"github.com/ddev/ddev/pkg/versionconstants"
	"github.com/fsouza/go-dockerclient"
)

func PowerOff() {
	apps, err := GetProjects(true)
	if err != nil {
		util.Failed("Failed to get project(s): %v", err)
	}

	// Remove any custom certs that may have been added
	// along with all traefik configuration.
	_, _, err = dockerutil.RunSimpleContainer(versionconstants.BusyboxImage, "poweroff-"+util.RandString(6), []string{"sh", "-c", "rm -rf /mnt/ddev-global-cache/custom_certs/* /mnt/ddev-global-cache/traefik/*"}, []string{}, []string{}, []string{"ddev-global-cache" + ":/mnt/ddev-global-cache"}, "", true, false, map[string]string{"com.ddev.site-name": ""}, nil)
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
	client := dockerutil.GetDockerClient()
	containers, err := client.ListContainers(docker.ListContainersOptions{
		All:     true,
		Filters: map[string][]string{"label": {"com.ddev.site-name"}},
	})
	if err == nil {
		for _, c := range containers {
			err = dockerutil.RemoveContainer(c.ID)
			if err != nil {
				util.Warning("Failed to remove container %s", c.Names[0])
			}
		}
	} else {
		util.Warning("unable to run client.ListContainers(): %v", err)
	}

	StopMutagenDaemon()

	if err := RemoveSSHAgentContainer(); err != nil {
		util.Error("Failed to remove ddev-ssh-agent: %v", err)
	}
	// Remove current global network ("ddev") plus the former "ddev_default"
	removals := []string{"ddev_default"}
	for _, networkName := range removals {
		err = dockerutil.RemoveNetwork(networkName)
		_, isNoSuchNetwork := err.(*docker.NoSuchNetwork)
		// If it's a "no such network" there's no reason to report error
		if err != nil && !isNoSuchNetwork {
			util.Warning("Unable to remove network %s: %v", "ddev", err)
		}
	}
}
