package platform

import (
	"fmt"
	"path/filepath"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/fatih/color"
	"github.com/fsouza/go-dockerclient"
	"github.com/gosuri/uitable"

	"errors"

	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/util"
	homedir "github.com/mitchellh/go-homedir"
)

// GetApps returns a list of ddev applictions keyed by platform.
func GetApps() map[string][]App {
	apps := make(map[string][]App)
	for platformType := range PluginMap {
		labels := map[string]string{
			"com.ddev.platform":          "ddev",
			"com.docker.compose.service": "web",
		}
		sites, err := dockerutil.FindContainersByLabels(labels)

		if err == nil {
			for _, siteContainer := range sites {
				site, err := GetPluginApp(platformType)
				// This should absolutely never happen, so just fatal on the off chance it does.
				if err != nil {
					log.Fatalf("could not get application for plugin type %s", platformType)
				}
				approot, ok := siteContainer.Labels["com.ddev.approot"]
				if !ok {
					break
				}
				_, ok = apps[platformType]
				if !ok {
					apps[platformType] = []App{}
				}

				err = site.Init(approot)
				if err == nil {
					apps[platformType] = append(apps[platformType], site)
				}
			}
		}
	}

	return apps
}

// RenderAppTable will format a table for user display based on a list of apps.
func RenderAppTable(platform string, apps []App) {
	if len(apps) > 0 {
		fmt.Printf("%v %s %v found.\n", len(apps), platform, util.FormatPlural(len(apps), "site", "sites"))
		table := CreateAppTable()
		for _, site := range apps {
			RenderAppRow(table, site)
		}
		fmt.Println(table)
		fmt.Println(PrintRouterStatus())
	}

}

// CreateAppTable will create a new app table for describe and list output
func CreateAppTable() *uitable.Table {
	table := uitable.New()
	table.MaxColWidth = 140
	table.Separator = "  "
	table.AddRow("NAME", "TYPE", "LOCATION", "URL", "STATUS")
	return table
}

// RenderAppRow will add an application row to an existing table for describe and list output.
func RenderAppRow(table *uitable.Table, site App) {
	// test tilde expansion
	appRoot := site.AppRoot()
	userDir, err := homedir.Dir()
	if err == nil {
		appRoot = strings.Replace(appRoot, userDir, "~", 1)
	}
	status := site.SiteStatus()
	if status == "stopped" {
		status = color.YellowString(status)
	} else {
		status = color.CyanString(status)
	}
	table.AddRow(
		site.GetName(),
		site.GetType(),
		appRoot,
		site.URL(),
		status,
	)
}

// Cleanup will clean up ddev apps even if the composer file has been deleted.
func Cleanup(app App) error {
	client := dockerutil.GetDockerClient()

	// Find all containers which match the current site name.
	labels := map[string]string{
		"com.ddev.site-name": app.GetName(),
	}
	containers, err := dockerutil.FindContainersByLabels(labels)
	if err != nil {
		return err
	}

	// First, try stopping the listed containers if they are running.
	for i := range containers {
		if containers[i].State == "running" || containers[i].State == "restarting" || containers[i].State == "paused" {
			containerName := containers[i].Names[0][1:len(containers[i].Names[0])]
			fmt.Printf("Stopping container: %s\n", containerName)
			err = client.StopContainer(containers[i].ID, 60)
			if err != nil {
				return fmt.Errorf("could not stop container %s: %v", containerName, err)
			}
		}
	}

	// Try to remove the containers once they are stopped.
	for i := range containers {
		containerName := containers[i].Names[0][1:len(containers[i].Names[0])]
		removeOpts := docker.RemoveContainerOptions{
			ID:            containers[i].ID,
			RemoveVolumes: true,
			Force:         true,
		}
		fmt.Printf("Removing container: %s\n", containerName)
		if err := client.RemoveContainer(removeOpts); err != nil {
			return fmt.Errorf("could not remove container %s: %v", containerName, err)
		}
	}

	return StopRouter()
}

// CheckForConf checks for a config.yaml at the cwd or parent dirs.
func CheckForConf(confPath string) (string, error) {
	if fileutil.FileExists(confPath + "/.ddev/config.yaml") {
		return confPath, nil
	}
	pathList := strings.Split(confPath, "/")

	for _ = range pathList {
		confPath = filepath.Dir(confPath)
		if fileutil.FileExists(confPath + "/.ddev/config.yaml") {
			return confPath, nil
		}
	}

	return "", errors.New("no .ddev/config.yaml file was found in this directory or any parent")
}

// ddevContainersRunning determines if any ddev-controlled containers are currently running.
func ddevContainersRunning() (bool, error) {
	containers, err := dockerutil.GetDockerContainers(false)
	if err != nil {
		return false, err
	}

	for _, container := range containers {
		if _, ok := container.Labels["com.ddev.platform"]; ok {
			return true, nil
		}
	}
	return false, nil
}
