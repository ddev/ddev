package local

import (
	"bytes"
	"errors"
	"fmt"
	"net/url"
	"path"
	"strings"
	"text/template"

	"github.com/drud/drud-go/secrets"
	"gopkg.in/yaml.v2"
)

// LegacyApp implements the LocalApp interface for Legacy Newmedia apps
type LegacyApp struct {
	Name        string
	Environment string
	AppType     string
	Template    string
	Branch      string
	Repo        string
}

// RenderComposeYAML returns teh contents of a docker compose config for this app
func (l LegacyApp) RenderComposeYAML() (string, error) {
	var doc bytes.Buffer
	var err error
	templ := template.New("compose template")
	templ, err = templ.Parse(l.Template)
	if err != nil {
		return "", err
	}
	templ.Execute(&doc, map[string]string{
		"image": fmt.Sprintf("drud/nginx-php-fpm-%s", l.AppType),
		"name":  fmt.Sprintf("legacy-%s", l.Name),
	})
	return doc.String(), nil
}

// Path returns the path from the '.drud' directory to this apps directory
func (l LegacyApp) Path() string {
	return path.Join("legacy", fmt.Sprintf("%s-%s", l.Name, l.Environment))
}

// GetRepoDetails uses the Environment field to get the relevant repo details about an app
func (l LegacyApp) GetRepoDetails() (RepoDetails, error) {
	dbag, err := GetDatabag(l.Name)
	if err != nil {
		return RepoDetails{}, err
	}

	details, err := dbag.GetRepoDetails(l.Environment)
	if err != nil {
		return RepoDetails{}, err
	}

	return details, nil
}

// GetDatabag returns databag info ad a Databag struct
func GetDatabag(name string) (Databag, error) {
	sobj := secrets.Secret{
		Path: "secret/databags/nmdhosting/" + name,
	}
	db := Databag{}

	err := sobj.Read()
	if err != nil {
		return db, err
	}

	yamlbytes, err := sobj.ToYAML()
	if err != nil {
		return db, err
	}

	err = yaml.Unmarshal(yamlbytes, &db)
	if err != nil {
		return db, err
	}

	return db, nil
}

// Databag models the outer most layer of a databag
type Databag struct {
	ID         string  `yaml:"id"`
	Default    SiteEnv `yaml:"_default"`
	Production SiteEnv `yaml:"production"`
	Staging    SiteEnv `yaml:"staging"`
}

// GetRepoDetails get the relevant repo details from a databag and returns a RepoDetails struct
func (d Databag) GetRepoDetails(env string) (RepoDetails, error) {
	details := RepoDetails{}
	siteEnviron := SiteEnv{}

	switch env {
	case "production":
		siteEnviron = d.Production
	case "staging":
		siteEnviron = d.Staging
	case "default":
		siteEnviron = d.Default
	default:
		return details, errors.New("Unrecognized environment name.")
	}

	repoPath := siteEnviron.Repository
	if !strings.HasPrefix("http", repoPath) {
		repoPath = "https://" + repoPath
	}

	u, err := url.Parse(repoPath)
	if err != nil {
		return details, err
	}

	hostOrg := strings.Split(u.Host, ":")
	details.Host = hostOrg[0]
	details.Org = hostOrg[1]
	details.User = u.User.String()
	// path becomes something like /repos-name.git so we extract the repo name
	details.Name = strings.Split(u.Path[1:], ".")[0]
	details.Branch = siteEnviron.Revision

	return details, nil
}

// SiteEnv models the iner contents of a databag
type SiteEnv struct {
	ActiveTheme        string   `yaml:"active_theme"`
	AdminMail          string   `yaml:"admin_mail"`
	AdminPassword      string   `yaml:"admin_password"`
	AdminUsername      string   `yaml:"admin_username"`
	ApacheGroup        string   `yaml:"apache_group"`
	ApacheOwner        string   `yaml:"apache_owner"`
	AwsAccessKey       string   `json:"aws_access_key"`
	AwsBucket          string   `json:"aws_bucket"`
	AwsSecretKey       string   `json:"aws_secret_key"`
	AwsUtfSymmetricKey string   `json:"aws_utf_symmetric_key"`
	AuthKey            string   `yaml:"auth_key"`
	AuthSalt           string   `yaml:"auth_salt"`
	CustomPort         string   `yaml:"custom_port"`
	DbHost             string   `yaml:"db_host"`
	DbName             string   `yaml:"db_name"`
	DbPort             string   `yaml:"db_port"`
	DbUserPassword     string   `yaml:"db_user_password"`
	DbUsername         string   `yaml:"db_username"`
	Docroot            string   `yaml:"docroot"`
	Hosts              []string `yaml:"hosts"`
	LoggedInKey        string   `yaml:"logged_in_key"`
	LoggedInSalt       string   `yaml:"logged_in_salt"`
	NonceKey           string   `yaml:"nonce_key"`
	NonceSalt          string   `yaml:"nonce_salt"`
	Php                struct {
		Version string `yaml:"version"`
	} `yaml:"php"`
	ProxyConfig struct {
		Auth bool `yaml:"auth"`
	} `yaml:"proxy_config"`
	Repository      string   `yaml:"repository"`
	Revision        string   `yaml:"revision"`
	SecureAuthKey   string   `yaml:"secure_auth_key"`
	SecureAuthSalt  string   `yaml:"secure_auth_salt"`
	ServerAliases   []string `yaml:"server_aliases"`
	Sitename        string   `yaml:"sitename"`
	SiteEnvironment string   `json:"site_environment"`
	Sitename2       string   `json:"site_name"`
	Type            string   `yaml:"type"`
	URL             string   `yaml:"url"`
}

// Name returns whichever site name happens to exist
func (s *SiteEnv) Name() string {
	if s.Sitename == "" {
		return s.Sitename2
	}
	return s.Sitename
}
