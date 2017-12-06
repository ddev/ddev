package platform

import (
	"github.com/fsouza/go-dockerclient"
)

// SiteRunning defines the string used to denote running sites.
const SiteRunning = "running"

// SiteNotFound defines the string used to denote a site where the containers were not found/do not exist.
const SiteNotFound = "not found"

// SiteDirMissing defines the string used to denote when a site is missing its application directory.
const SiteDirMissing = "app directory missing"

// SiteConfigMissing defines the string used to denote when a site is missing its .ddev/config.yml file.
const SiteConfigMissing = ".ddev/config.yaml missing"

// SiteStopped defines the string used to denote when a site is in the stopped state.
const SiteStopped = "stopped"

// App is an interface apps for Drud Local must implement to use shared functionality
type App interface {
	Init(string) error
	Describe() (map[string]interface{}, error)
	GetType() string
	AppRoot() string
	GetName() string
	Start() error
	Stop() error
	DockerEnv()
	DockerComposeYAMLPath() string
	Down(removeData bool) error
	CreateSettingsFile() error
	HostName() string
	URL() string
	Import() error
	ImportDB(imPath string, extPath string) error
	ImportFiles(imPath string, extPath string) error
	SiteStatus() string
	FindContainerByType(containerType string) (docker.APIContainers, error)
	// Returns err, stdout, stderr
	Exec(service string, cmd ...string) (string, string, error)
	ExecWithTty(service string, cmd ...string) error
	Logs(service string, follow bool, timestamps bool, tail string) error
}

// GetApp will return an empty LocalApp
func GetApp() *LocalApp {
	return &LocalApp{}
}
