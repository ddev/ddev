package platform

// App is an interface apps for Drud Local must implement to use shared functionality
type App interface {
	SetOpts(AppOptions)
	Init(AppOptions)
	GetOpts() AppOptions
	GetType() string
	GetResources() error
	UnpackResources() error
	ContainerPrefix() string
	ContainerName() string
	AbsPath() string
	GetName() string
	Start() error
	Stop() error
	Down() error
	Config() error
	Wait() (string, error)
	URL() string
}

// AppBase is the parent type for all local app implementations
type AppBase struct {
	Name          string
	Plugin        string
	AppType       string
	Branch        string
	Repo          string
	Archive       string //absolute path to the downloaded archive
	WebPublicPort int64
	DbPublicPort  int64
	Status        string
}

// AppOptions ..
type AppOptions struct {
	Name        string
	Plugin      string
	AppType     string
	WebImage    string
	DbImage     string
	WebImageTag string
	DbImageTag  string
	Template    string
}

var PluginMap = map[string]App{
	"local": &LocalApp{},
}
