package local

import (
	"errors"
	"fmt"

	"github.com/drud/drud-go/drudapi"
)

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
	DrudClient  *drudapi.Request
}

// RepoDetails encapsulates repository info
type RepoDetails struct {
	Host   string `json:"host,omitempty"` // will generally default to github.com
	User   string
	Name   string `json:"name,omitempty"`
	Org    string `json:"org,omitempty"`
	Branch string `json:"branch,omitempty"` // will default to master
}

// GetCloneURL returns a url of the format: git@github.com:drud/sanctuary-ui.git
func (r RepoDetails) GetCloneURL() (string, error) {
	repoHost := "github.com"
	if r.Host != "" {
		repoHost = r.Host
	}

	if r.Org == "" || r.Name == "" {
		return "", errors.New("Org and Name must be set in order to create a clone URL.")
	}

	user := "git"
	if r.User != "" {
		user = r.User
	}

	return fmt.Sprintf("%s@%s:%s/%s.git", user, repoHost, r.Org, r.Name), nil
}

var PluginMap = map[string]App{
	"drud":   &DrudApp{},
	"legacy": &LegacyApp{},
}
