package local

import (
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"time"

	"gopkg.in/yaml.v2"

	log "github.com/Sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/drud/drud-go/utils"
	"github.com/drud/drud-go/utils/try"
	"github.com/fsouza/go-dockerclient"
	"github.com/gosuri/uitable"
)

// PrepLocalSiteDirs creates a site's directories for local dev in ~/.drud/client/site
func PrepLocalSiteDirs(base string) error {
	err := os.MkdirAll(base, os.FileMode(int(0774)))
	if err != nil {
		return err
	}

	dirs := []string{
		"src",
		"files",
		"data",
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

// WriteLocalAppYAML writes docker-compose.yaml to $HOME/.drud/app.Path()
func WriteLocalAppYAML(app App) error {

	basePath := app.AbsPath()

	f, err := os.Create(path.Join(basePath, "docker-compose.yaml"))
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

// CloneSource clones or pulls a repo
func CloneSource(app App) error {
	homedir, err := utils.GetHomeDir()
	if err != nil {
		log.Fatalln(err)
	}

	details, err := app.GetRepoDetails()
	if err != nil {
		return err
	}

	cloneURL, err := details.GetCloneURL()
	if err != nil {
		return err
	}

	basePath := path.Join(homedir, ".drud", app.RelPath(), "src")

	out, err := utils.RunCommand("git", []string{
		"clone", "-b", details.Branch, cloneURL, basePath,
	})
	if err != nil {
		if !strings.Contains(string(out), "already exists") {
			return fmt.Errorf("%s - %s", err.Error(), string(out))
		}

		fmt.Print("Local copy of site exists, updating... ")

		out, err = utils.RunCommand("git", []string{
			"-C", basePath,
			"pull", "origin", details.Branch,
		})
		if err != nil {
			return fmt.Errorf("%s - %s", err.Error(), string(out))
		}

		fmt.Printf("Updated to latest in %s branch\n", details.Branch)
	}

	if len(out) > 0 {
		log.Info(string(out))
	}

	return nil
}

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

// GetPodPort clones or pulls a repo
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

func FilterNonAppDirs(files []os.FileInfo) []os.FileInfo {

	var filtered []os.FileInfo
	for _, v := range files {
		name := v.Name()
		parts := strings.SplitN(name, "-", 2)

		if len(parts) != 2 {
			continue
		}

		filtered = append(filtered, v)
	}
	return filtered
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
		table.AddRow("NAME", "ENVIRONMENT", "TYPE", "URL", "DATABASE URL", "STATUS")

		for _, site := range apps {
			table.AddRow(
				site["name"],
				site["environment"],
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
	parts := strings.Split(containerName, "-")

	if len(parts) == 4 {
		appid := parts[1] + "-" + parts[2]

		_, exists := l[appid]
		if exists == false {
			app := PluginMap[strings.ToLower(plugin)]
			opts := AppOptions{
				Name:        parts[1],
				Environment: parts[2],
			}
			app.SetOpts(opts)

			l[appid] = map[string]string{
				"name":        parts[1],
				"environment": parts[2],
				"status":      container.State,
				"url":         app.URL(),
				"type":        app.GetType(),
			}
		}

		var publicPort int64
		for _, port := range container.Ports {
			if port.PublicPort != 0 {
				publicPort = port.PublicPort
			}
		}

		if parts[3] == "web" {
			l[appid]["WebPublicPort"] = fmt.Sprintf("%d", publicPort)
		}

		if parts[3] == "db" {
			l[appid]["DbPublicPort"] = fmt.Sprintf("%d", publicPort)
		}

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
		if FileExists(path.Join(basePath, "src", k)) {
			return v, nil
		}
	}

	return "", fmt.Errorf("Couldn't determine app's type!")
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
	homeDir, err := utils.GetHomeDir()
	if err != nil {
		log.Fatal("could not find home directory")
	}
	dest := path.Join(homeDir, ".drud", "router-compose.yaml")
	f, ferr := os.Create(dest)
	if ferr != nil {
		log.Fatal(ferr)
	}
	defer f.Close()

	template := fmt.Sprintf(DrudRouterTemplate)
	f.WriteString(template)

	// run docker-compose up -d in the newly created directory
	out, err := utils.RunCommand("docker-compose", []string{"-f", dest, "up", "-d"})
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

// Config represents the data that is or can be stored in $HOME/.drud
type Config struct {
	Protocol     string `yaml:"protocol"`
	Client       string `yaml:"client"`
	DrudHost     string `yaml:"drudhost"`
	VaultHost    string `yaml:"vaulthost"`
	Version      string `yaml:"version"`
	Dev          bool   `yaml:"dev"`
	ActiveApp    string `yaml:"activeapp"`
	ActiveDeploy string `yaml:"activedeploy"`
}

// EveHost creates the eve host string from the config.
func (cfg *Config) EveHost() string {
	return fmt.Sprintf("%s://%s/%s/", cfg.Protocol, cfg.DrudHost, cfg.Version)
}

// GetConfig ...
func GetConfig() (cfg *Config, err error) {

	c := &Config{}
	err = viper.Unmarshal(c)

	// if c.Version == "" {
	// 	c.Version = drudapiVersion
	// }

	if c.VaultHost == "" {
		c.VaultHost = "https://sanctuary.drud.io:8200"
	}

	if c.Protocol == "" {
		c.Protocol = "https"
	}

	if c.DrudHost == "" {
		c.DrudHost = "drudapi.drud.io"
	}

	return c, nil
}

// WriteConfig writes each config value to the BoltDB and updates the
// global cfg as well.
func (cfg *Config) WriteConfig(f string) (err error) {
	cfgbytes, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(f, cfgbytes, 0644)
	if err != nil {
		return err
	}

	return nil
}

func ComposeFileExists(app App) bool {
	composeLOC := path.Join(app.AbsPath(), "docker-compose.yaml")
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
				_, err := utils.RunCommand("docker", args)
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
		opts.WebImage = "drud/nginx-php-fpm-local:latest"
	}
	if opts.DbImage == "" {
		opts.DbImage = "drud/mysql-docker-local:5.7"
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
	}

	if opts.WebImageTag == "unison" || strings.HasSuffix(opts.WebImage, ":unison") {
		templateVars["srctarget"] = "/src"
	}

	templ.Execute(&doc, templateVars)
	return doc.String(), nil
}
