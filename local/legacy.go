package local

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/drud/bootstrap/cli/cache"
	"github.com/drud/drud-go/secrets"
	"github.com/drud/drud-go/utils"
	"gopkg.in/yaml.v2"
)

var cacher *cache.Cache

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

// GetAppResources downloads external data for this app
func (l LegacyApp) GetAppResources() error {
	homedir, err := utils.GetHomeDir()
	if err != nil {
		log.Fatalln(err)
	}

	basePath := path.Join(homedir, ".drud", l.Path())

	dbag, err := GetDatabag(l.Name)
	if err != nil {
		return err
	}

	s, err := dbag.GetEnv(l.Environment)
	if err != nil {
		return err
	}

	bucket := "nmdarchive"
	if s.AwsBucket != "" {
		bucket = s.AwsBucket
	}

	awsID := s.AwsAccessKey
	awsSecret := s.AwsSecretKey
	if awsID == "" {
		sobj := secrets.Secret{
			Path: "secret/shared/services/awscfg",
		}

		err := sobj.Read()
		if err != nil {
			log.Fatal(err)
		}

		awsID = sobj.Data["accesskey"].(string)
		awsSecret = sobj.Data["secretkey"].(string)
	}

	os.Setenv("AWS_ACCESS_KEY_ID", awsID)
	os.Setenv("AWS_SECRET_ACCESS_KEY", awsSecret)

	svc := s3.New(session.New(&aws.Config{Region: aws.String("us-west-2")}))
	prefix := fmt.Sprintf("%[1]s/%[2]s-%[1]s-", l.Name, l.Environment)

	params := &s3.ListObjectsInput{
		Bucket: aws.String(bucket),
		Prefix: &prefix,
	}

	resp, err := svc.ListObjects(params)
	if err != nil {
		return err
	}

	archive := resp.Contents[len(resp.Contents)-1]
	fmt.Println(archive)

	file, err := os.Create(path.Join(basePath, filepath.Base(*archive.Key)))
	if err != nil {
		log.Fatal("Failed to create file", err)
	}
	defer file.Close()

	downloader := s3manager.NewDownloader(session.New(&aws.Config{Region: aws.String("us-west-2")}))
	numBytes, err := downloader.Download(
		file,
		&s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(*archive.Key),
		},
	)
	if err != nil {
		return err
	}

	fmt.Println("Downloaded file", file.Name(), numBytes, "bytes")

	out, err := utils.RunCommand(
		"tar",
		[]string{"-xzvf", file.Name(), "-C", path.Join(basePath, "files")},
	)
	if err != nil {
		fmt.Println(out)
		return err
	}

	err = os.Rename(
		path.Join(basePath, "files", l.Name+".sql"),
		path.Join(basePath, "data", l.Name+".sql"),
	)
	if err != nil {
		return err
	}

	rsyncFrom := path.Join(basePath, "files", "docroot")
	rsyncTo := path.Join(basePath, "src", "docroot")
	out, err = utils.RunCommand(
		"rsync",
		[]string{"-avz", "--recursive", rsyncFrom + "/", rsyncTo},
	)
	if err != nil {
		return fmt.Errorf("%s - %s", err.Error(), string(out))
	}

	return nil
}

// GetDatabag returns databag info ad a Databag struct
func GetDatabag(name string) (Databag, error) {
	if cacher == nil {
		cacher = cache.New()
	}

	cacheDb := cacher.Get(name + "-databag")
	if cacheDb != nil {
		fmt.Println("usign cached object")
		return cacheDb.(Databag), nil
	}
	fmt.Println("Not usign cached object")

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
