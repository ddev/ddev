package local

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/hashicorp/vault/api"
	"github.com/lextoumbourou/goodhosts"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"

	"github.com/drud/bootstrap/cli/cms/config"
	"github.com/drud/bootstrap/cli/cms/model"
	"github.com/drud/drud-go/secrets"
	"github.com/drud/drud-go/utils"
)

const (
	containerRunning = "running"
)

// LegacyApp implements the AppBase interface for Legacy Newmedia apps
type LegacyApp struct {
	AppBase
	Options *AppOptions
	Vault   *api.Logical
}

// NewLegacyApp returns an empty legacy app with options struct pre inserted
func NewLegacyApp(name string, environment string) *LegacyApp {
	app := &LegacyApp{
		Options: &AppOptions{},
	}
	app.AppBase.Name = name
	app.AppBase.Environment = environment

	return app
}

func (l *LegacyApp) SetOpts(opts AppOptions) {
	l.Options = &opts
	l.Name = opts.Name
	l.Environment = opts.Environment
	//l.AppType = opts.AppType
	l.Template = LegacyComposeTemplate
	if opts.Template != "" {
		l.Template = opts.Template
	}
	l.SkipYAML = opts.SkipYAML
}

func (l *LegacyApp) GetOpts() AppOptions {
	return *l.Options
}

func (l *LegacyApp) GetTemplate() string {
	return l.Template
}

func (l *LegacyApp) GetType() string {
	if l.AppType == "" {
		l.SetType()
	}
	return l.AppType
}

// Init sets values from the AppInitOptions on the Drud app object
func (l *LegacyApp) Init(opts AppOptions) {
	l.SetOpts(opts)

	if !l.DatabagExists() {
		log.Fatal("No legacy site by that name.")
	}

	basePath := l.AbsPath()
	err := PrepLocalSiteDirs(basePath)
	if err != nil {
		log.Fatalln(err)
	}

}

// RelPath returns the path from the '.drud' directory to this apps directory
func (l LegacyApp) RelPath() string {
	return path.Join("legacy", fmt.Sprintf("%s-%s", l.Name, l.Environment))
}

// AbsPath return the full path from root to the app directory
func (l LegacyApp) AbsPath() string {
	homedir, err := utils.GetHomeDir()
	if err != nil {
		log.Fatalln(err)
	}
	return path.Join(homedir, ".drud", l.RelPath())
}

// GetName returns the  name for legacy app
func (l LegacyApp) GetName() string {
	return l.Name
}

// ContainerPrefix returns the base name for legacy app containers
func (l LegacyApp) ContainerPrefix() string {
	return "legacy-"
}

// ContainerName returns the base name for legacy app containers
func (l LegacyApp) ContainerName() string {
	return fmt.Sprintf("%s%s-%s", l.ContainerPrefix(), l.Name, l.Environment)
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

// DatabagExists checks if a databag exists or not.
func (l LegacyApp) DatabagExists() bool {
	_, err := GetDatabag(l.Name)
	if err != nil {
		return false
	}
	return true
}

// GetResources downloads external data for this app
func (l *LegacyApp) GetResources() error {

	// save errors for when the wait group has finished executing
	errChannel := make(chan error, 1)
	// apparently this is necessary
	finished := make(chan bool, 1)

	// limit logical processors to 3
	runtime.GOMAXPROCS(2)
	// set up wait group

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()

		fmt.Println("Getting source code.")

		err := CloneSource(l)
		if err != nil {
			log.Println(err)
			errChannel <- fmt.Errorf("Error cloning source: %s", err)
		}
	}()

	go func() {
		defer wg.Done()

		fmt.Println("Getting Resources.")
		err := l.GetArchive()
		if err != nil {
			log.Println(err)
			errChannel <- fmt.Errorf("Error retrieving site resources: %s", err)
		}
	}()

	wg.Wait()
	close(finished)

	select {
	case <-finished:
	case err := <-errChannel:
		if err != nil {
			return fmt.Errorf("Unable to retrieve one or more resources.")
		}
	}

	err := l.SetType()
	if err != nil {
		return err
	}

	if !l.SkipYAML {
		err = WriteLocalAppYAML(l)
		if err != nil {
			log.Println("Could not create docker-compose.yaml")
			return err
		}
	}

	return nil
}

// GetArchive downloads external data for this app
func (l *LegacyApp) GetArchive() error {
	basePath := l.AbsPath()

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

	if len(resp.Contents) == 0 {
		return errors.New("No site archive found")
	}

	archive := resp.Contents[len(resp.Contents)-1]
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

	log.Println("Downloaded file", file.Name(), numBytes, "bytes")
	l.Archive = file.Name()

	return nil
}

// UnpackResources takes the archive from the GetResources method and
// unarchives it. Then the contents are moved to their proper locations.
func (l LegacyApp) UnpackResources() error {
	basePath := l.AbsPath()
	fileDir := ""

	if l.AppType == "wp" {
		fileDir = "content/uploads"
	} else if l.AppType == "drupal" || l.AppType == "drupal8" {
		fileDir = "sites/default/files"
	}

	out, err := utils.RunCommand(
		"tar",
		[]string{
			"-xzvf",
			l.Archive,
			"-C", path.Join(basePath, "files"),
			"--exclude=sites/default/settings.php",
			"--exclude=docroot/wp-config.php",
		},
	)
	if err != nil {
		fmt.Println(out)
		return err
	}

	err = os.Remove(l.Archive)
	if err != nil {
		return err
	}

	err = os.Rename(
		path.Join(basePath, "files", l.Name+".sql"),
		path.Join(basePath, "data", "data.sql"),
	)
	if err != nil {
		return err
	}

	rsyncFrom := path.Join(basePath, "files", "docroot", fileDir)
	rsyncTo := path.Join(basePath, "src", "docroot", fileDir)
	out, err = utils.RunCommand(
		"rsync",
		[]string{"-avz", "--recursive", rsyncFrom + "/", rsyncTo},
	)
	if err != nil {
		return fmt.Errorf("%s - %s", err.Error(), string(out))
	}

	dcfgFile := path.Join(basePath, "src", "drud.yaml")
	if utils.FileExists(dcfgFile) {
		log.Println("copying drud.yaml to data container")
		out, err := utils.RunCommand("cp", []string{
			dcfgFile,
			path.Join(basePath, "data/drud.yaml"),
		})
		if err != nil {
			fmt.Println(out)
			return err
		}
	}

	return nil
}

// Start initiates docker-compose up
func (l LegacyApp) Start() error {

	composePath := path.Join(l.AbsPath(), "docker-compose.yaml")

	err := l.SetType()
	if err != nil {
		return err
	}

	if !l.SkipYAML {
		fmt.Println("Creating docker-compose config.")
		err = WriteLocalAppYAML(&l)
		if err != nil {
			return err
		}
	}

	EnsureDockerRouter()

	err = l.AddHostsEntry()
	if err != nil {
		log.Fatal(err)
	}

	cmdArgs := []string{"-f", composePath, "pull"}
	_, err = utils.RunCommand("docker-compose", cmdArgs)
	if err != nil {
		return err
	}

	return utils.DockerCompose(
		"-f", composePath,
		"up",
		"-d",
	)
}

// Stop initiates docker-compose stop
func (l LegacyApp) Stop() error {
	composePath := path.Join(l.AbsPath(), "docker-compose.yaml")

	if !utils.IsRunning(l.ContainerName()+"-db") && !utils.IsRunning(l.ContainerName()+"-web") && !ComposeFileExists(&l) {
		return fmt.Errorf("Site does not exist or is malformed.")
	}

	return utils.DockerCompose(
		"-f", composePath,
		"stop",
	)
}

// SetType determines the app type and sets it
func (l *LegacyApp) SetType() error {
	appType, err := DetermineAppType(l.AbsPath())
	if err != nil {
		return err
	}
	l.AppType = appType
	return nil
}

// Wait ensures that the app appears to be read before returning
func (l *LegacyApp) Wait() (string, error) {
	o := utils.NewHTTPOptions("http://127.0.0.1")
	o.Timeout = 420
	o.Headers["Host"] = l.HostName()
	err := utils.EnsureHTTPStatus(o)
	if err != nil {
		return "", fmt.Errorf("200 Was not returned from the web container")
	}

	return l.URL(), nil
}

func (l *LegacyApp) FindPorts() error {
	var err error
	l.WebPublicPort, err = GetPodPort(l.ContainerName() + "-web")
	if err != nil {
		return err
	}

	l.DbPublicPort, err = GetPodPort(l.ContainerName() + "-db")
	return err
}

// Config creates the apps config file adding things like database host, name, and password
// as well as other sensitive data like salts.
func (l *LegacyApp) Config() error {
	basePath := l.AbsPath()

	if l.AppType == "" {
		err := l.SetType()
		if err != nil {
			return err
		}
	}

	dbag, err := GetDatabag(l.Name)
	if err != nil {
		return err
	}

	env, err := dbag.GetEnv(l.Environment)
	if err != nil {
		return err
	}

	err = l.FindPorts()
	if err != nil {
		return err
	}

	settingsFilePath := ""
	if l.AppType == "drupal" {
		log.Printf("Drupal site. Creating settings.php file.")
		settingsFilePath = path.Join(basePath, "src", "docroot/sites/default/settings.php")
		drupalConfig := model.NewDrupalConfig()
		drupalConfig.DatabaseHost = "db"
		drupalConfig.HashSalt = env.HashSalt
		if drupalConfig.HashSalt == "" {
			drupalConfig.HashSalt = utils.RandomString(64)
		}

		drupalConfig.DeployURL = l.URL()
		err = config.WriteDrupalConfig(drupalConfig, settingsFilePath)
		if err != nil {
			log.Fatalln(err)
		}

		// Setup a custom settings file for use with drush.
		dbPort, err := GetPodPort(l.ContainerName() + "-db")
		if err != nil {
			return err
		}

		drushSettingsPath := path.Join(basePath, "src", "drush.settings.php")
		drushConfig := model.NewDrushConfig()
		drushConfig.DatabasePort = dbPort
		err = config.WriteDrushConfig(drushConfig, drushSettingsPath)

		if err != nil {
			log.Fatalln(err)
		}
	} else if l.AppType == "wp" {
		log.Printf("WordPress site. Creating wp-config.php file.")
		settingsFilePath = path.Join(basePath, "src", "docroot/wp-config.php")
		wpConfig := model.NewWordpressConfig()
		wpConfig.DatabaseHost = "db"
		wpConfig.DeployURL = l.URL()
		wpConfig.AuthKey = env.AuthKey
		wpConfig.AuthSalt = env.AuthSalt
		wpConfig.LoggedInKey = env.LoggedInKey
		wpConfig.LoggedInSalt = env.LoggedInSalt
		wpConfig.NonceKey = env.NonceKey
		wpConfig.NonceSalt = env.NonceSalt
		wpConfig.SecureAuthKey = env.SecureAuthKey
		wpConfig.SecureAuthSalt = env.SecureAuthSalt
		err = config.WriteWordpressConfig(wpConfig, settingsFilePath)
		if err != nil {
			log.Fatalln(err)
		}
	}
	return nil
}

// Down stops the docker containers for the legacy project.
func (l *LegacyApp) Down() error {
	composePath := path.Join(l.AbsPath(), "docker-compose.yaml")

	err := utils.DockerCompose(
		"-f", composePath,
		"down",
	)
	if err != nil {
		return Cleanup(l)
	}

	return nil

}

// URL returns the URL for a given application.
func (l *LegacyApp) URL() string {
	return "http://" + l.HostName()
}

// HostName returns the hostname of a given application.
func (l *LegacyApp) HostName() string {
	return l.ContainerName()
}

// AddHostsEntry will add the legacy site URL to the local hostfile.
func (l *LegacyApp) AddHostsEntry() error {
	if os.Getenv("DRUD_NONINTERACTIVE") != "" {
		fmt.Printf("Please add the following entry to your host file:\n127.0.0.1 %s\n", l.HostName())
		return nil
	}

	hosts, err := goodhosts.NewHosts()
	if err != nil {
		log.Fatalf("could not open hostfile. %s", err)
	}
	if hosts.Has("127.0.0.1", l.HostName()) {
		return nil
	}

	fmt.Println("\n\n\nAdding hostfile entry. You will be prompted for your password.")
	hostnameArgs := []string{"drud", "dev", "hostname", l.HostName(), "127.0.0.1"}
	err = utils.RunCommandPipe("sudo", hostnameArgs)
	return err
}
