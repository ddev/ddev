package local

import (
	"errors"
	"net/url"
	"strings"

	"github.com/drud/drud-go/secrets"
	"github.com/drud/drud-go/utils"
	"gopkg.in/yaml.v2"
)

var cacher *utils.Cache

// Databag models the outer most layer of a databag
type Databag struct {
	ID         string  `yaml:"id"`
	Default    SiteEnv `yaml:"_default"`
	Production SiteEnv `yaml:"production"`
	Staging    SiteEnv `yaml:"staging"`
}

// GetID satisfies cache interface
func (d Databag) GetID() string {
	return d.ID + "-databag"
}

// GetEnv returns SiteEnv for the environment you want
func (d *Databag) GetEnv(name string) (*SiteEnv, error) {
	var siteEnviron *SiteEnv

	switch name {
	case "production":
		siteEnviron = &d.Production
	case "staging":
		siteEnviron = &d.Staging
	case "default":
		siteEnviron = &d.Default
	default:
		return siteEnviron, errors.New("Unrecognized environment name.")
	}
	return siteEnviron, nil
}

// GetRepoDetails get the relevant repo details from a databag and returns a RepoDetails struct
func (d Databag) GetRepoDetails(env string) (RepoDetails, error) {
	details := RepoDetails{}

	siteEnviron, err := d.GetEnv(env)
	if err != nil {
		return details, err
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

// SiteEnv models the contents of a databag
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
	HashSalt           string   `yaml:"hash_salt"`
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

// GetDatabag returns databag info ad a Databag struct
func GetDatabag(name string) (Databag, error) {
	if cacher == nil {
		cacher = utils.New()
	}

	cacheDb := cacher.Get(name + "-databag")
	if cacheDb != nil {
		return cacheDb.(Databag), nil
	}

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

	cacher.Add(db)

	return db, nil
}
