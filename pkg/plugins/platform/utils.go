package platform

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path"
	"strings"

	log "github.com/Sirupsen/logrus"

	"errors"
	"github.com/docker/docker/pkg/homedir"
	"github.com/drud/ddev/pkg/appports"
	"github.com/drud/ddev/pkg/util"
	"github.com/drud/ddev/pkg/version"
	"github.com/drud/drud-go/utils/system"
	"github.com/fsouza/go-dockerclient"
)

// PrepLocalSiteDirs creates a site's directories for local dev in .ddev
func PrepLocalSiteDirs(base string) error {
	dirs := []string{
		".ddev",
		".ddev/data",
	}
	for _, d := range dirs {
		dirPath := path.Join(base, d)
		err := os.Mkdir(dirPath, os.FileMode(int(0774)))
		if err != nil {
			if !strings.Contains(err.Error(), "file exists") {
				return err
			}
		}
	}

	return nil
}

// SiteList will prepare and render a list of ddev sites running locally.
func SiteList(containers []docker.APIContainers) error {
	apps := map[string]map[string]map[string]string{}
	var appsFound bool

	for pName := range PluginMap {
		apps[pName] = map[string]map[string]string{}
	}

	for _, container := range containers {
		for _, containerName := range container.Names {
			for pName, plugin := range PluginMap {
				if strings.HasPrefix(containerName[1:], plugin.ContainerPrefix()) {
					util.ProcessContainer(apps[pName], pName, containerName[1:], container)
					break
				}
			}
		}
	}

	if len(apps) > 0 {
		for k, v := range apps {
			util.RenderAppTable(v, k)
		}
	}

	for _, appList := range apps {
		if len(appList) > 0 {
			appsFound = true
		}
	}

	if appsFound == false {
		fmt.Println("No Applications Found.")
	}

	return nil
}

// EnsureDockerRouter ensures the router is running.
func EnsureDockerRouter() {
	userHome := homedir.Get()
	routerdir := path.Join(userHome, ".ddev")
	err := os.MkdirAll(routerdir, 0755)
	if err != nil {
		log.Fatalf("unable to create directory for ddev router: %s", err)
	}

	var doc bytes.Buffer
	dest := path.Join(routerdir, "router-compose.yaml")
	f, ferr := os.Create(dest)
	if ferr != nil {
		log.Fatal(ferr)
	}
	defer util.CheckClose(f)

	templ := template.New("compose template")
	templ, err = templ.Parse(fmt.Sprintf(DrudRouterTemplate))
	if err != nil {
		log.Fatal(ferr)
	}

	templateVars := map[string]string{
		"router_image": version.RouterImage,
		"router_tag":   version.RouterTag,
		"mailhogport":  appports.GetPort("mailhog"),
		"dbaport":      appports.GetPort("dba"),
	}

	err = templ.Execute(&doc, templateVars)
	util.CheckErr(err)
	_, err = f.WriteString(doc.String())
	util.CheckErr(err)

	// run docker-compose up -d in the newly created directory
	out, err := system.RunCommand("docker-compose", []string{"-p", "ddev-router", "-f", dest, "up", "-d"})
	if err != nil {
		fmt.Println(fmt.Errorf("%s - %s", err.Error(), string(out)))
	}
}

// ComposeFileExists determines if a docker-compose.yml exists for a given app.
func ComposeFileExists(app App) bool {
	if _, err := os.Stat(app.DockerComposeYAMLPath()); os.IsNotExist(err) {
		return false
	}
	return true
}

// Cleanup will clean up ddev apps even if the composer file has been deleted.
func Cleanup(app App) error {
	client, _ := util.GetDockerClient()

	containers, err := client.ListContainers(docker.ListContainersOptions{All: false})
	if err != nil {
		return err
	}

	actions := []string{"stop", "rm"}
	for _, c := range containers {
		if strings.Contains(c.Names[0], app.ContainerName()) {
			for _, action := range actions {
				args := []string{action, c.ID}
				_, err := system.RunCommand("docker", args)
				if err != nil {
					return fmt.Errorf("Could not %s container %s: %s", action, c.Names[0], err)
				}
			}
		}

	}

	return nil
}

// CheckForConf checks for a config.yaml at the cwd or parent dirs.
func CheckForConf(confPath string) (string, error) {
	if system.FileExists(confPath + "/.ddev/config.yaml") {
		return confPath, nil
	}
	pathList := strings.Split(confPath, "/")

	for _ = range pathList {
		confPath = path.Dir(confPath)
		if system.FileExists(confPath + "/.ddev/config.yaml") {
			return confPath, nil
		}
	}

	return "", errors.New("no .ddev/config.yaml file was found in this directory or any parent")
}

// DetermineAppType uses some predetermined file checks to determine if a local app
// is of any of the known types
func DetermineAppType(basePath string) (string, error) {
	defaultLocations := map[string]string{
		"docroot/scripts/drupal.sh":      "drupal",
		"docroot/core/scripts/drupal.sh": "drupal8",
		"docroot/wp":                     "wp",
	}

	for k, v := range defaultLocations {
		if system.FileExists(path.Join(basePath, k)) {
			return v, nil
		}
	}

	return "", fmt.Errorf("unable to determine the application type")
}
