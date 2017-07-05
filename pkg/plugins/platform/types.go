package platform

import "fmt"
import (
	"strings"

	"github.com/fsouza/go-dockerclient"
)

// SiteRunning defines the string used to denote running sites.
const SiteRunning = "running"

// SiteNotFound defines the string used to denote a site where the containers were not found/do not exist.
const SiteNotFound = "not found"

// SiteStopped defines the string used to denote when a site is in the stopped state.
const SiteStopped = "stopped"

// App is an interface apps for Drud Local must implement to use shared functionality
type App interface {
	Init(string) error
	Describe() (string, error)
	GetType() string
	AppRoot() string
	GetName() string
	Start() error
	Stop() error
	DockerEnv()
	DockerComposeYAMLPath() string
	Down(removeData bool) error
	Config() error
	HostName() string
	URL() string
	ImportDB(imPath string, extPath string) error
	ImportFiles(imPath string, extPath string) error
	SiteStatus() string
	FindContainerByType(containerType string) (docker.APIContainers, error)
	Exec(service string, tty bool, cmd ...string) error
	Logs(service string, follow bool, timestamps bool, tail string) error
}

// PluginMap maps the name of the plugins to their implementation.
var PluginMap = map[string]App{
	"local": &LocalApp{},
}

// GetPluginApp will return an application of the type specified by pluginType
func GetPluginApp(pluginType string) (App, error) {
	switch strings.ToLower(pluginType) {
	case "local":
		return &LocalApp{}, nil
	default:
		return nil, fmt.Errorf("could not find plugin type %s", pluginType)
	}
}
