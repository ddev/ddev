package platform

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/gosuri/uitable"

	"errors"

	"github.com/drud/ddev/pkg/appports"
	"github.com/drud/ddev/pkg/util"
	"github.com/drud/ddev/pkg/version"
	"github.com/drud/drud-go/utils/system"
	"github.com/fsouza/go-dockerclient"
	homedir "github.com/mitchellh/go-homedir"
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

// GetApps returns a list of ddev applictions keyed by platform.
func GetApps() map[string][]App {
	apps := make(map[string][]App)
	for platformType, instance := range PluginMap {
		labels := map[string]string{
			"com.ddev.platform":       instance.ContainerPrefix(),
			"com.ddev.container-type": "web",
		}
		sites, err := util.FindContainersByLabels(labels)

		if err == nil {
			for _, siteContainer := range sites {
				site := GetPluginApp(platformType)
				approot, ok := siteContainer.Labels["com.ddev.approot"]
				if !ok {
					break
				}
				_, ok = apps[platformType]
				if !ok {
					apps[platformType] = []App{}
				}

				err := site.Init(approot)
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
	}

}

// CreateAppTable will create a new app table for describe and list output
func CreateAppTable() *uitable.Table {
	table := uitable.New()
	table.MaxColWidth = 200
	table.AddRow("NAME", "TYPE", "LOCATION", "URL", "DATABASE URL", "STATUS")
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
	table.AddRow(
		site.GetName(),
		site.GetType(),
		appRoot,
		site.URL(),
		fmt.Sprintf("%s:%s", site.HostName(), appports.GetPort("db")),
		site.SiteStatus(),
	)
}

// EnsureDockerRouter ensures the router is running.
func EnsureDockerRouter() {
	userHome, err := homedir.Dir()
	util.CheckErr(err)

	routerdir := path.Join(userHome, ".ddev")
	err = os.MkdirAll(routerdir, 0755)
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
	templ, err = templ.Parse(DrudRouterTemplate)
	if err != nil {
		log.Fatal(ferr)
	}

	templateVars := map[string]string{
		"router_image": version.RouterImage,
		"router_tag":   version.RouterTag,
		"mailhogport":  appports.GetPort("mailhog"),
		"dbaport":      appports.GetPort("dba"),
		"dbport":       appports.GetPort("db"),
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
