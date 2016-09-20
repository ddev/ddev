package local

import (
	"errors"
	"fmt"
)

// App is an interface apps for Drud Local must implement to use shared functionality
type App interface {
	RenderComposeYAML() (string, error)   // returns contents for docke rcompose config
	RelPath() string                      // returns path from root dir ('$HOME/.drud') to app
	GetRepoDetails() (RepoDetails, error) // retuirns struct with branch, org, host, etc...
	ContainerName() string
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
