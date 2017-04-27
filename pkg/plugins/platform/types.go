package platform

import "fmt"
import "strings"

// App is an interface apps for Drud Local must implement to use shared functionality
type App interface {
	Init(string) error
	Describe() (string, error)
	GetType() string
	ContainerPrefix() string
	ContainerName() string
	AppRoot() string
	GetName() string
	Start() error
	Stop() error
	DockerEnv()
	DockerComposeYAMLPath() string
	Down() error
	Config() error
	Wait(string) error
	HostName() string
	URL() string
	ImportDB(string) error
	ImportFiles(string) error
	SiteStatus() string
}

// AppBase is the parent type for all local app implementations
type AppBase struct {
	Plugin        string
	Archive       string //absolute path to the downloaded archive
	WebPublicPort int64
	DbPublicPort  int64
	Status        string
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
