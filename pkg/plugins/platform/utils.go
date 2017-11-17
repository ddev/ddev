package platform

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	docker "github.com/fsouza/go-dockerclient"
	"github.com/gosuri/uitable"

	"errors"

	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/util"
	gohomedir "github.com/mitchellh/go-homedir"
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
					util.Failed("could not get application for plugin type %s: %v", platformType, err)
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
				if err != nil {
					// Cast 'site' from type App to type LocalApp, so we can manually enter AppConfig values.
					siteStruct, ok := site.(*LocalApp)
					if !ok {
						util.Failed("Failed to cast siteStruct(type App) to *LocalApp{}. site=%v", site)
					}

					siteStruct.AppConfig.Name = siteContainer.Labels["com.ddev.site-name"]
					siteStruct.AppConfig.AppType = siteContainer.Labels["com.ddev.app-type"]
				}
				apps[platformType] = append(apps[platformType], site)
			}
		}
	}

	return apps
}

// CreateAppTable will create a new app table for describe and list output
func CreateAppTable() *uitable.Table {
	table := uitable.New()
	table.MaxColWidth = 140
	table.Separator = "  "
	table.Wrap = true
	table.AddRow("NAME", "TYPE", "LOCATION", "URL(s)", "STATUS")
	return table
}

// RenderHomeRootedDir shortens a directory name to replace homedir with ~
func RenderHomeRootedDir(path string) string {
	userDir, err := gohomedir.Dir()
	util.CheckErr(err)
	result := strings.Replace(path, userDir, "~", 1)
	result = strings.Replace(result, "\\", "/", -1)
	return result
}

// RenderAppRow will add an application row to an existing table for describe and list output.
func RenderAppRow(table *uitable.Table, row map[string]interface{}) {
	status := fmt.Sprint(row["status"])

	switch {
	case strings.Contains(status, SiteStopped):
		status = color.YellowString(status)
	case strings.Contains(status, SiteNotFound):
		status = color.RedString(status)
	case strings.Contains(status, SiteDirMissing):
		status = color.RedString(status)
	case strings.Contains(status, SiteConfigMissing):
		status = color.RedString(status)
	default:
		status = color.CyanString(status)
	}

	urls := row["httpurl"].(string) + "\n" + row["httpsurl"].(string)
	table.AddRow(
		row["name"],
		row["type"],
		row["shortroot"],
		urls,
		status,
	)

}

// Cleanup will remove ddev containers and volumes even if docker-compose.yml
// has been deleted.
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
		containerName := containers[i].Names[0][1:len(containers[i].Names[0])]
		removeOpts := docker.RemoveContainerOptions{
			ID:            containers[i].ID,
			RemoveVolumes: true,
			Force:         true,
		}
		output.UserOut.Printf("Removing container: %s", containerName)
		if err = client.RemoveContainer(removeOpts); err != nil {
			return fmt.Errorf("could not remove container %s: %v", containerName, err)
		}
	}

	err = StopRouter()
	return err
}

// CheckForConf checks for a config.yaml at the cwd or parent dirs.
func CheckForConf(confPath string) (string, error) {
	if fileutil.FileExists(confPath + "/.ddev/config.yaml") {
		return confPath, nil
	}
	pathList := strings.Split(confPath, "/")

	for range pathList {
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
