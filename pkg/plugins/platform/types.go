package platform

// App is an interface apps for Drud Local must implement to use shared functionality
type App interface {
	Init(string) error
	Describe() (string, error)
	GetType() string
	GetResources() error
	UnpackResources() error
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
	Wait() (string, error)
	HostName() string
	URL() string
	ImportDB(string) error
	ImportFiles(string) error
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
