package drudapi

import (
	"encoding/json"
	"fmt"
	pathlib "path"
	"strings"

	"github.com/gosuri/uitable"
)

// BackUpLink is used to interacting with the gcs endpoint and retrieving signed urls to backups
type BackUpLink struct {
	AppID    string
	DeployID string // really the deplay.Name
	URL      string // will be set on GET from drudclient
	Type     string // currently just 'mysql' or 'files'
}

// Path returns DRUD API path for a signed backup url
func (l BackUpLink) Path(method string) string {
	return pathlib.Join("gcs", l.Type, l.AppID, l.DeployID)
}

// Unmarshal sets the URL that should be in data in the URL field
func (l *BackUpLink) Unmarshal(data []byte) error {
	var err error
	if len(data) == 0 {
		err = fmt.Errorf("No link to unmarshal!")
	}

	l.URL = string(data)
	return err
}

// LoginLink is used to interacting with the gcs endpoint and retrieving signed urls to backups
type LoginLink struct {
	AppName    string
	DeployName string
	ClientName string
	URL        string // will be set on GET from drudclient
}

// Path returns DRUD API path for a one time login link
func (l LoginLink) Path(method string) string {
	return pathlib.Join("login-link", l.ClientName, l.AppName, l.DeployName)
}

// Unmarshal sets the URL that should be in data in the URL field
func (l *LoginLink) Unmarshal(data []byte) error {
	var err error
	if len(data) == 0 {
		err = fmt.Errorf("No link to unmarshal!")
	}

	l.URL = string(data)
	return err
}

// Deploy ...
type Deploy struct {
	Name          string `json:"name,omitempty"`
	Template      string `json:"template,omitempty"`
	Branch        string `json:"branch,omitempty"`
	Hostname      string `json:"hostname,omitempty"`
	Protocol      string `json:"protocol,omitempty"`
	BasicAuthUser string `json:"basicauth_user,omitempty"`
	BasicAuthPass string `json:"basicauth_pass,omitempty"`
	AutoManaged   bool   `json:"auto_managed,omitempty"`
	MigrateFrom   string `json:"migrate_from,omitempty"`
	Url           string `json:"url,omitempty"`
}

// Application ...
type Application struct {
	AppID          string   `json:"app_id,omitempty"`
	Client         Client   `json:"client,omitempty"`
	Deploys        []Deploy `json:"deploys,omitempty"`
	GithubHookID   int      `json:"github_hook_id,omitempty"`
	RepoOrg        string   `json:"repo_org,omitempty"`
	Name           string   `json:"name,omitempty"`
	Repo           string   `json:"repo,omitempty"`
	SlackChannel   string   `json:"slack_channel,omitempty"`
	AuthKey        string   `json:"auth_key,omitempty"`
	SecureAuthKey  string   `json:"secure_auth_key,omitempty"`
	LoggedInKey    string   `json:"logged_in_key,omitempty"`
	NonceKey       string   `json:"nonce_key,omitempty"`
	AuthSalt       string   `json:"auth_salt,omitempty"`
	SecureAuthSalt string   `json:"secure_auth_salt,omitempty"`
	LoggedInSalt   string   `json:"logged_in_salt,omitempty"`
	NonceSalt      string   `json:"nonce_salt,omitempty"`
	RepoDetails    *struct {
		Host     string `json:"host,omitempty"`
		Name     string `json:"name,omitempty"`
		Org      string `json:"org,omitempty"`
		Branch   string `json:"branch,omitempty"`
		Dest     string `json:"dest,omitempty"`
		CloneURL string `json:"clone_url,omitempty"`
	} `json:"repo_details,omitempty"`
	Created string `json:"_created,omitempty"`
	Etag    string `json:"_etag,omitempty"`
	ID      string `json:"_id,omitempty"`
	Updated string `json:"_updated,omitempty"`
}

// Path ...
func (a Application) Path(method string) string {
	var path string

	if method == "POST" {
		path = "application"
	} else {
		path = "application/" + a.AppID
	}
	return path
}

// Unmarshal ...
func (a *Application) Unmarshal(data []byte) error {
	err := json.Unmarshal(data, a)
	return err
}

// GetDeploy looks for a deploy by name and returns it
func (a *Application) GetDeploy(name string) *Deploy {
	for _, d := range a.Deploys {
		if d.Name == name {
			return &d
		}
	}
	return nil
}

// JSON ...
func (a Application) JSON() []byte {
	a.ID = ""
	a.Etag = ""
	a.Created = ""
	a.Updated = ""
	a.RepoDetails = nil

	jbytes, _ := json.Marshal(a)
	return jbytes
}

// PatchJSON ...
func (a Application) PatchJSON() []byte {
	a.ID = ""
	a.Etag = ""
	a.Created = ""
	a.Updated = ""
	// removing name because it has been setup as the id param in drudapi and cannot be  patched
	a.AppID = ""

	jbytes, _ := json.Marshal(a)
	return jbytes
}

// ETAG ...
func (a Application) ETAG() string {
	return a.Etag
}

// GetFilesLink ...
func (a *Application) GetFilesLink(deployName string) (string, error) {
	deploy := a.GetDeploy(deployName)
	if deploy == nil {
		return "", fmt.Errorf("No deploy found by name %s", deployName)
	}
	return a.AppID + "/" + deploy.Name, nil
}

// GetMysqlLink ...
func (a *Application) GetMysqlLink(deployName string) (string, error) {
	deploy := a.GetDeploy(deployName)
	if deploy == nil {
		return "", fmt.Errorf("No deploy found by name %s", deployName)
	}
	return a.AppID + "/" + deploy.Name, nil
}

// RepoURL ...
func (a *Application) RepoURL(token string) string {
	var url string
	if a.RepoDetails != nil {
		if token != "" {
			url = fmt.Sprintf("https://%s@%s/%s/%s.git",
				token,
				a.RepoDetails.Host,
				a.RepoDetails.Org,
				a.Name,
			)
		} else {
			url = fmt.Sprintf("https://%s/%s/%s.git", a.RepoDetails.Host, a.RepoDetails.Org, a.Name)
		}
	}

	return url
}

// Describe an application..mostly used for displaying deploys
func (a *Application) Describe() {

	table := uitable.New()
	table.MaxColWidth = 50
	table.Wrap = true // wrap columns

	deployTable := uitable.New()

	// gather list of deploys by name
	var appNames []string
	for _, dep := range a.Deploys {
		appNames = append(appNames, dep.Name)
		var managed string

		url := dep.Protocol + "://" + dep.Url

		if dep.AutoManaged == true {
			managed = "✓"
		}
		deployTable.AddRow("DEPLOY NAME:", dep.Name)
		deployTable.AddRow("URL:", url)
		deployTable.AddRow("TEMPLATE:", dep.Template)
		deployTable.AddRow("BRANCH:", dep.Branch)
		deployTable.AddRow("AUTH USER:", dep.BasicAuthUser)
		deployTable.AddRow("AUTH PASS:", dep.BasicAuthPass)
		deployTable.AddRow("AUTO MANAGED:", managed)
		deployTable.AddRow("\n")
	}
	table.AddRow("APP NAME:", a.Name)
	table.AddRow("CLIENT:", a.Client.Name)
	table.AddRow("DEPLOY(s):", strings.Join(appNames, ","))
	table.AddRow("SLACK CHANNEL:", a.SlackChannel)
	table.AddRow("CREATED:", a.Created)

	fmt.Println(table)
	fmt.Printf("\n%v %v found.\n\n", len(a.Deploys), FormatPlural(len(a.Deploys), "deploy", "deploys"))
	fmt.Println(deployTable)

}

// ApplicationList entity
type ApplicationList struct {
	Name  string
	Items []Application `json:"_items"`
	Meta  struct {
		MaxResults int `json:"max_results"`
		Page       int `json:"page"`
		Total      int `json:"total"`
	} `json:"_meta"`
}

// Path ...
func (a ApplicationList) Path(method string) string {
	return "application"
}

// Unmarshal ...
func (a *ApplicationList) Unmarshal(data []byte) error {
	err := json.Unmarshal(data, &a)
	return err
}

// Describe pretty prints the entity
func (a *ApplicationList) Describe() {
	fmt.Printf("%v %v found.\n\n", len(a.Items), FormatPlural(len(a.Items), "application", "applications"))

	table := uitable.New()
	table.MaxColWidth = 50
	table.AddRow("NAME", "CLIENT", "DEPLOY(s)", "SLACK CHANNEL", "CREATED")
	for _, app := range a.Items {
		// gather list of deploys by name
		var appNames []string
		for _, dep := range app.Deploys {
			appNames = append(appNames, dep.Name)
		}
		table.AddRow(
			app.Name,
			app.Client.Name,
			strings.Join(appNames, ","),
			app.SlackChannel,
			app.Created,
		)
	}
	fmt.Println(table)

}
