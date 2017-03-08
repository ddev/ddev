package platform

// App is an interface apps for Drud Local must implement to use shared functionality
type App interface {
	SetOpts(AppOptions)
	Init(AppOptions)
	GetOpts() AppOptions
	GetType() string
	RelPath() string                      // returns path from root dir ('$HOME/.drud') to app
	GetRepoDetails() (RepoDetails, error) // returns struct with branch, org, host, etc...
	GetResources() error
	GetTemplate() string
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
	Environment   string
	Plugin        string
	AppType       string
	Template      string
	Branch        string
	Repo          string
	Archive       string //absolute path to the downloaded archive
	WebPublicPort int64
	DbPublicPort  int64
	Status        string
	SkipYAML      bool
}

// AppOptions ..
type AppOptions struct {
	Name        string
	Environment string
	Plugin      string
	AppType     string
	Client      string
	WebImage    string
	DbImage     string
	WebImageTag string
	DbImageTag  string
	SkipYAML    bool
	Template    string
	CFG         *Config
}

var PluginMap = map[string]App{
	"drud":   &DrudApp{},
	"legacy": &LegacyApp{},
	"local":  &LocalApp{},
}
