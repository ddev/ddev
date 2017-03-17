package platform

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"os"
	"path"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"

	"github.com/docker/docker/pkg/homedir"
	"github.com/drud/ddev/pkg/version"
	"github.com/drud/drud-go/utils/system"
	"github.com/drud/drud-go/utils/try"
	"github.com/fsouza/go-dockerclient"
	"github.com/gosuri/uitable"
)

// PrepLocalSiteDirs creates a site's directories for local dev in ~/.drud/client/site
func PrepLocalSiteDirs(base string) error {
	dirs := []string{
		".ddev",
		".ddev/files",
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

// WriteLocalAppYAML writes docker-compose.yaml to .ddev folder
func WriteLocalAppYAML(app App) error {

	basePath := app.AbsPath()

	f, err := os.Create(path.Join(basePath, ".ddev", "docker-compose.yaml"))
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	rendered, err := RenderComposeYAML(app)
	if err != nil {
		return err
	}
	f.WriteString(rendered)
	return nil
}

// GetPort determines and returns the public port for a given container.
func GetPort(name string) (int64, error) {
	client, _ := GetDockerClient()
	var publicPort int64

	containers, err := client.ListContainers(docker.ListContainersOptions{All: false})
	if err != nil {
		return publicPort, err
	}

	for _, ctr := range containers {
		if strings.Contains(ctr.Names[0][1:], name) {
			for _, port := range ctr.Ports {
				if port.PublicPort != 0 {
					publicPort = port.PublicPort
					return publicPort, nil
				}
			}
		}
	}
	return publicPort, fmt.Errorf("%s container not ready", name)
}

// GetPodPort provides a wait loop to help in successfully returning the public port for a given container.
func GetPodPort(name string) (int64, error) {
	var publicPort int64

	err := try.Do(func(attempt int) (bool, error) {
		var err error
		publicPort, err = GetPort(name)
		if err != nil {
			time.Sleep(2 * time.Second) // wait a couple seconds
		}
		return attempt < 70, err
	})
	if err != nil {
		return publicPort, err
	}

	return publicPort, nil
}

// GetDockerClient returns a docker client for a docker-machine.
func GetDockerClient() (*docker.Client, error) {
	// Create a new docker client talking to the default docker-machine.
	client, err := docker.NewClient("unix:///var/run/docker.sock")
	if err != nil {
		log.Fatal(err)
	}
	return client, err
}

// FormatPlural is a simple wrapper which returns different strings based on the count value.
func FormatPlural(count int, single string, plural string) string {
	if count == 1 {
		return single
	}
	return plural
}

// SiteList will prepare and render a list of drud sites running locally.
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
					ProcessContainer(apps[pName], pName, containerName[1:], container)
					break
				}
			}
		}
	}

	if len(apps) > 0 {
		for k, v := range apps {
			RenderAppTable(v, k)
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

// RenderAppTable will format a table for user display based on a list of apps.
func RenderAppTable(apps map[string]map[string]string, name string) {
	if len(apps) > 0 {
		fmt.Printf("%v %s %v found.\n", len(apps), name, FormatPlural(len(apps), "site", "sites"))
		table := uitable.New()
		table.MaxColWidth = 200
		table.AddRow("NAME", "TYPE", "URL", "DATABASE URL", "STATUS")

		for _, site := range apps {
			table.AddRow(
				site["name"],
				site["type"],
				site["url"],
				fmt.Sprintf("127.0.0.1:%s", site["DbPublicPort"]),
				site["status"],
			)
		}
		fmt.Println(table)
	}

}

// ProcessContainer will process a docker container for an app listing.
// Since apps contain multiple containers, ProcessContainer will be called once per container.
func ProcessContainer(l map[string]map[string]string, plugin string, containerName string, container docker.APIContainers) {

	label := container.Labels

	appName := label["com.drud.site-name"]
	appType := label["com.drud.app-type"]
	containerType := label["com.drud.container-type"]

	_, exists := l[appName]
	if exists == false {
		app := PluginMap[strings.ToLower(plugin)]
		opts := AppOptions{
			Name: appName,
		}
		app.SetOpts(opts)

		l[appName] = map[string]string{
			"name":   appName,
			"status": container.State,
			"url":    app.URL(),
			"type":   appType,
		}
	}

	var publicPort int64
	for _, port := range container.Ports {
		if port.PublicPort != 0 {
			publicPort = port.PublicPort
		}
	}

	if containerType == "web" {
		l[appName]["WebPublicPort"] = fmt.Sprintf("%d", publicPort)
	}

	if containerType == "db" {
		l[appName]["DbPublicPort"] = fmt.Sprintf("%d", publicPort)
	}

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
		if FileExists(path.Join(basePath, k)) {
			return v, nil
		}
	}

	return "", fmt.Errorf("unable to determine the application type")
}

// FileExists checks a file's existence
// @todo replace this with drud-go/utils version when merged
func FileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
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
	defer f.Close()

	templ := template.New("compose template")
	templ, err = templ.Parse(fmt.Sprintf(DrudRouterTemplate))
	if err != nil {
		log.Fatal(ferr)
	}

	templateVars := map[string]string{
		"router_image": version.RouterImage,
		"router_tag":   version.RouterTag,
	}

	templ.Execute(&doc, templateVars)
	f.WriteString(doc.String())

	// run docker-compose up -d in the newly created directory
	out, err := system.RunCommand("docker-compose", []string{"-f", dest, "up", "-d"})
	if err != nil {
		fmt.Println(fmt.Errorf("%s - %s", err.Error(), string(out)))
	}

}

// SubTag replaces current tag on an image or adds one if one does not exist
func SubTag(image string, tag string) string {
	if strings.HasSuffix(image, ":"+tag) {
		return image
	}
	if !strings.Contains(image, ":") || (strings.HasPrefix(image, "http") && strings.Count(image, ":") == 1) {
		return image + ":" + tag
	}
	parts := strings.Split(image, ":")
	parts[len(parts)-1] = tag
	return strings.Join(parts, ":")
}

// ComposeFileExists determines if a docker-compose.yml exists for a given app.
func ComposeFileExists(app App) bool {
	composeLOC := path.Join(app.AbsPath(), ".ddev", "docker-compose.yaml")
	if _, err := os.Stat(composeLOC); os.IsNotExist(err) {
		return false
	}
	return true
}

// Cleanup will clean up legacy apps even if the composer file has been deleted.
func Cleanup(app App) error {
	client, _ := GetDockerClient()

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

func RenderComposeYAML(app App) (string, error) {
	var doc bytes.Buffer
	var err error
	templ := template.New("compose template")
	templ, err = templ.Parse(app.GetTemplate())
	if err != nil {
		return "", err
	}

	opts := app.GetOpts()

	if opts.WebImage == "" {
		opts.WebImage = version.WebImg + ":" + version.WebTag
	}
	if opts.DbImage == "" {
		opts.DbImage = version.DBImg + ":" + version.DBTag
	}
	if opts.WebImageTag != "" {
		opts.WebImage = SubTag(opts.WebImage, opts.WebImageTag)
	}
	if opts.DbImageTag != "" {
		opts.DbImage = SubTag(opts.DbImage, opts.DbImageTag)
	}

	templateVars := map[string]string{
		"web_image": opts.WebImage,
		"db_image":  opts.DbImage,
		"name":      app.ContainerName(),
		"srctarget": "/var/www/html",
		"site_name": opts.Name,
		"apptype":   app.GetType(),
	}

	if opts.WebImageTag == "unison" || strings.HasSuffix(opts.WebImage, ":unison") {
		templateVars["srctarget"] = "/src"
	}

	templ.Execute(&doc, templateVars)
	return doc.String(), nil
}

// CheckForConf checks for a ddev.yaml at the cwd or parent dirs.
func CheckForConf(confPath string) (string, error) {
	if system.FileExists(confPath + "/ddev.yaml") {
		return confPath, nil
	}
	pathList := strings.Split(confPath, "/")

	for _ = range pathList {
		confPath = path.Dir(confPath)
		if system.FileExists(confPath + "/ddev.yaml") {
			return confPath, nil
		}
	}

	return "", errors.New("no ddev.yaml file in this directory or any parent")
}

// NetExists checks to see if the docker network for DRUD local development exists.
func NetExists(client *docker.Client, name string) bool {
	nets, _ := client.ListNetworks()
	for _, n := range nets {
		if n.Name == name {
			return true
		}
	}
	return false
}

// EnsureNetwork will ensure the docker network for DRUD local development is created.
func EnsureNetwork(client *docker.Client, name string) error {
	if !NetExists(client, name) {
		netOptions := docker.CreateNetworkOptions{
			Name:     name,
			Driver:   "bridge",
			Internal: false,
		}
		_, err := client.CreateNetwork(netOptions)
		if err != nil {
			return err
		}
		log.Println("Network", name, "created")

	}
	return nil
}
