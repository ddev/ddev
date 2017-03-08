package platform

import (
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/drud/drud-go/utils/system"
	log "github.com/mgutz/logxi/v1"
)

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
