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

	"github.com/drud/dcfg/dcfglib"
	"github.com/drud/ddev/pkg/version"
	"github.com/drud/drud-go/utils/system"
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

	details, err := app.GetRepoDetails()
	if err != nil {
		return err
	}

	cloneURL, err := details.GetCloneURL()
	if err != nil {
		return err
	}
	cfg, _ := GetConfig()
	basePath := path.Join(cfg.Workspace, app.RelPath(), "src")

	out, err := system.RunCommand("git", []string{
		"clone", "-b", details.Branch, cloneURL, basePath,
	})
	if err != nil {
		if !strings.Contains(string(out), "already exists") {
			return fmt.Errorf("%s - %s", err.Error(), string(out))
		}

		fmt.Print("Local copy of site exists, updating... ")

		out, err = system.RunCommand("git", []string{
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
	// load the "pre-start" group of the drud.yaml
	dcfg, err := DrudCfgTaskSet(path.Join(basePath, "src", "drud.yaml"), "pre-start")
	if err != nil {
		return "", err
	}

	env := dcfg.Env
	for k, v := range env {
		if k == "appType" {
			return v, nil
		}
	}

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

	return "", fmt.Errorf("could not determine the app type")
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
	var doc bytes.Buffer
	cfg, _ := GetConfig()
	dest := path.Join(cfg.Workspace, "router-compose.yaml")
	f, ferr := os.Create(dest)
	if ferr != nil {
		log.Fatal(ferr)
	}
	defer f.Close()

	templ := template.New("compose template")
	templ, err := templ.Parse(fmt.Sprintf(DrudRouterTemplate))
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

// Config represents the data that is or can be stored in $HOME/.drud
type Config struct {
	APIVersion      string `yaml:"apiversion"`
	ActiveApp       string `yaml:"activeapp"`
	ActiveDeploy    string `yaml:"activedeploy"`
	Client          string `yaml:"client"`
	DrudHost        string `yaml:"drudhost"`
	GithubAuthToken string `yaml:"githubauthtoken"`
	GithubAuthOrg   string `yaml:"githubauthorg"`
	Protocol        string `yaml:"protocol"`
	VaultAddr       string `yaml:"vaultaddr"`
	VaultAuthToken  string `yaml:"vaultauthtoken"`
	Workspace       string `yaml:"workspace"`
}

// EveHost creates the eve host string from the config.
func (cfg *Config) EveHost() string {
	return fmt.Sprintf("%s://%s/%s/", cfg.Protocol, cfg.DrudHost, cfg.APIVersion)
}

func parseConfigFlag() string {
	var value string

	for i, arg := range os.Args {
		if strings.HasPrefix(arg, "--config=") {
			value = strings.TrimPrefix(arg, "--config=")
		} else if arg == "--config" {
			value = os.Args[i+1]
		}
	}
	if value == "" {
		home, _ := system.GetHomeDir()
		value = fmt.Sprintf("%v/drud.yaml", home)
	}

	if _, err := os.Stat(value); os.IsNotExist(err) {
		var cFile, err = os.Create(value)
		if err != nil {
			log.Fatal(err)
		}
		cFile.Close()
	}
	return value
}

// GetConfig Loads a config structure from yaml and environment.
func GetConfig() (cfg *Config, err error) {
	cfgFile := parseConfigFlag()

	source, err := ioutil.ReadFile(cfgFile)
	if err != nil {
		panic(err)
	}

	c := &Config{}
	err = yaml.Unmarshal(source, c)
	if err != nil {
		panic(err)
	}

	if c.APIVersion == "" && os.Getenv("DRUD_APIVERSION") != "" {
		c.APIVersion = os.Getenv("DRUD_APIVERSION")
	}
	if c.ActiveApp == "" && os.Getenv("DRUD_ACTIVEAPP") != "" {
		c.ActiveApp = os.Getenv("DRUD_ACTIVEAPP")
	}
	if c.ActiveDeploy == "" && os.Getenv("DRUD_ACTIVEDEPLOY") != "" {
		c.ActiveDeploy = os.Getenv("DRUD_ACTIVEDEPLOY")
	}
	if c.Client == "" && os.Getenv("DRUD_CLIENT") != "" {
		c.Client = os.Getenv("DRUD_CLIENT")
	}
	if c.DrudHost == "" && os.Getenv("DRUD_DRUDHOST") != "" {
		c.DrudHost = os.Getenv("DRUD_DRUDHOST")
	}
	if c.GithubAuthToken == "" && os.Getenv("DRUD_GITHUBAUTHTOKEN") != "" {
		c.GithubAuthToken = os.Getenv("DRUD_GITHUBAUTHTOKEN")
	}
	if c.GithubAuthOrg == "" && os.Getenv("DRUD_GITHUBAUTHORG") != "" {
		c.GithubAuthOrg = os.Getenv("DRUD_GITHUBAUTHORG")
	}
	if c.Protocol == "" && os.Getenv("DRUD_PROTOCOL") != "" {
		c.Protocol = os.Getenv("DRUD_PROTOCOL")
	}
	if c.VaultAddr == "" && os.Getenv("DRUD_VAULTADDR") != "" {
		c.VaultAddr = os.Getenv("DRUD_VAULTADDR")
	}
	if c.VaultAuthToken == "" && os.Getenv("DRUD_VAULTAUTHTOKEN") != "" {
		c.VaultAuthToken = os.Getenv("DRUD_VAULTAUTHTOKEN")
	}
	if c.Workspace == "" && os.Getenv("DRUD_WORKSPACE") != "" {
		c.Workspace = os.Getenv("DRUD_WORKSPACE")
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
		"web_image":  opts.WebImage,
		"db_image":   opts.DbImage,
		"name":       app.ContainerName(),
		"srctarget":  "/var/www/html",
		"deploy_url": app.URL(),
	}

	if opts.WebImageTag == "unison" || strings.HasSuffix(opts.WebImage, ":unison") {
		templateVars["srctarget"] = "/src"
	}

	templ.Execute(&doc, templateVars)
	return doc.String(), nil
}

// DrudCfgTaskSet returns a TaskSet of the defined group
func DrudCfgTaskSet(filePath string, group string) (dcfglib.TaskSet, error) {
	var set dcfglib.TaskSet
	cfg, err := ioutil.ReadFile(filePath)
	if err != nil {
		fmt.Println(err)
	}

	setList, err := dcfglib.GetTaskSetList(cfg)
	if err != nil {
		fmt.Println(err)
	}

	for _, taskSet := range setList {
		if taskSet.Name == group {
			set = taskSet
		}
	}
	return set, nil
}
