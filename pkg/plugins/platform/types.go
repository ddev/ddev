package platform

// App is an interface apps for Drud Local must implement to use shared functionality
type App interface {
	Init() error
	GetType() string
	GetResources() error
	UnpackResources() error
	ContainerPrefix() string
	ContainerName() string
	AbsPath() string
	GetName() string
	Start() error
	Stop() error
	DockerComposeYAMLPath() string
	Down() error
	Config() error
	Wait() (string, error)
	URL() string
}

// AppBase is the parent type for all local app implementations
type AppBase struct {
	Plugin        string
	Archive       string //absolute path to the downloaded archive
	WebPublicPort int64
	DbPublicPort  int64
	Status        string
}

var PluginMap = map[string]App{
	"local": &LocalApp{},
}
