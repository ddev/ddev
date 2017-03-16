package platform

import (
	"fmt"
	"os"
	"path"

	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/drud/ddev/pkg/cms/config"
	"github.com/drud/ddev/pkg/cms/model"
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/drud-go/utils/dockerutil"
	"github.com/drud/drud-go/utils/network"
	"github.com/drud/drud-go/utils/stringutil"
	"github.com/drud/drud-go/utils/system"
	"github.com/lextoumbourou/goodhosts"
)

// LocalApp implements the AppBase interface local development apps
type LocalApp struct {
	AppBase
	AppConfig *ddevapp.Config
}

// NewLocalApp creates a new LocalApp based on any application root specified by appRoot
func NewLocalApp(appRoot string) *LocalApp {
	app := &LocalApp{}
	config, err := ddevapp.NewConfig(appRoot)
	app.AppConfig = config

	err = PrepLocalSiteDirs(appRoot)
	if err != nil {
		log.Fatalln(err)
	}
	return app
}

// GetType returns the application type as a (lowercase) string
func (l *LocalApp) GetType() string {
	return strings.ToLower(l.AppConfig.AppType)
}

// Init populates LocalApp settings based on the current working directory.
func (l *LocalApp) Init() error {
	basePath := l.AbsPath()
	config, err := ddevapp.NewConfig(basePath)
	if err != nil {
		return err
	}
	l.AppConfig = config

	err = PrepLocalSiteDirs(basePath)
	if err != nil {
		log.Fatalln(err)
	}

	return nil
}

// AbsPath return the full path from root to the app directory
func (l LocalApp) AbsPath() string {
	cwd, err := os.Getwd()
	if err != nil {
		log.Printf("Error determining the current directory: %s", err)
	}

	appPath, err := CheckForConf(cwd)
	if err != nil {
		log.Fatalf("Unable to determine the application for this command: %s", err)
	}

	return appPath
}

// GetName returns the  name for local app
func (l LocalApp) GetName() string {
	return l.AppConfig.Name
}

// ContainerPrefix returns the base name for local app containers
func (l LocalApp) ContainerPrefix() string {
	return "local"
}

// ContainerName returns the base name for local app containers
func (l LocalApp) ContainerName() string {
	return fmt.Sprintf("%s-%s", l.ContainerPrefix(), l.GetName())
}

// GetResources downloads external data for this app
func (l *LocalApp) GetResources() error {

	fmt.Println("Getting Resources.")
	err := l.GetArchive()
	if err != nil {
		log.Println(err)
		fmt.Println(fmt.Errorf("Error retrieving site resources: %s", err))
	}

	return nil
}

// GetArchive downloads external data
func (l *LocalApp) GetArchive() error {
	name := fmt.Sprintf("production-%s.tar.gz", l.GetName())
	basePath := l.AbsPath()
	archive := path.Join(basePath, ".ddev", name)

	if system.FileExists(archive) {
		l.Archive = archive
	}
	return nil
}

// DockerComposeYAMLPath returns the absolute path to where the docker-compose.yaml should exist for this app configuration.
// This is a bit redundant, but is here to avoid having to expose too many details of AppConfig.
func (l LocalApp) DockerComposeYAMLPath() string {
	return l.AppConfig.DockerComposeYAMLPath()
}

// UnpackResources takes the archive from the GetResources method and
// unarchives it. Then the contents are moved to their proper locations.
func (l LocalApp) UnpackResources() error {
	basePath := l.AbsPath()
	fileDir := ""

	if l.GetType() == "wordpress" {
		fileDir = "content/uploads"
	} else if l.GetType() == "drupal7" || l.GetType() == "drupal8" {
		fileDir = "sites/default/files"
	}

	out, err := system.RunCommand(
		"tar",
		[]string{
			"-xzvf",
			l.Archive,
			"-C", path.Join(basePath, ".ddev", "files"),
			"--exclude=sites/default/settings.php",
			"--exclude=docroot/wp-config.php",
		},
	)
	if err != nil {
		fmt.Println(out)
		return err
	}

	err = os.Rename(
		path.Join(basePath, ".ddev", "files", l.GetName()+".sql"),
		path.Join(basePath, ".ddev", "data", "data.sql"),
	)
	if err != nil {
		return err
	}

	// Ensure sites/default is readable.
	if l.GetType() == "drupal7" || l.GetType() == "drupal8" {
		os.Chmod(path.Join(basePath, ".ddev", "files", "docroot", "sites", "default"), 0755)
	}

	rsyncFrom := path.Join(basePath, ".ddev", "files", "docroot", fileDir)
	rsyncTo := path.Join(basePath, "docroot", fileDir)
	out, err = system.RunCommand(
		"rsync",
		[]string{"-avz", "--recursive", rsyncFrom + "/", rsyncTo},
	)
	if err != nil {
		return fmt.Errorf("%s - %s", err.Error(), string(out))
	}

	// Ensure extracted files are writable so they can be removed when we're done.
	out, err = system.RunCommand(
		"chmod",
		[]string{"-R", "ugo+rw", path.Join(basePath, ".ddev", "files")},
	)
	if err != nil {
		return fmt.Errorf("%s - %s", err.Error(), string(out))
	}
	defer os.RemoveAll(path.Join(basePath, ".ddev", "files"))

	return nil
}

// Start initiates docker-compose up
func (l LocalApp) Start() error {
	composePath := l.DockerComposeYAMLPath()
	l.DockerEnv()

	EnsureDockerRouter()

	err := l.AddHostsEntry()
	if err != nil {
		log.Fatal(err)
	}

	cmdArgs := []string{"-f", composePath, "pull"}
	_, err = system.RunCommand("docker-compose", cmdArgs)
	if err != nil {
		return err
	}

	return dockerutil.DockerCompose(
		"-f", composePath,
		"up",
		"-d",
	)
}

// DockerEnv sets environment variables for a docker-compose run.
func (l LocalApp) DockerEnv() {
	envVars := map[string]string{
		"DRUD_DBIMAGE":  l.AppConfig.DBImage,
		"DRUD_WEBIMAGE": l.AppConfig.WebImage,
	}

	// Only set values if they don't already exist in env.
	for k, v := range envVars {
		if os.Getenv(k) == "" {

			log.WithFields(log.Fields{
				"Key":   k,
				"Value": v,
			}).Debug("Setting DockerEnv variable")

			err := os.Setenv(k, v)
			// @ TODO: I have no idea what a Setenv error would even look like, so I'm not sure what
			// to do other than notify the user.
			if err != nil {
				fmt.Printf("Could not set the environment variable %s=%s: %v\n", k, v, err)
			}
		}
	}
}

// Stop initiates docker-compose stop
func (l LocalApp) Stop() error {
	composePath := l.DockerComposeYAMLPath()
	l.DockerEnv()

	if !dockerutil.IsRunning(l.ContainerName()+"-db") && !dockerutil.IsRunning(l.ContainerName()+"-web") && !ComposeFileExists(&l) {
		return fmt.Errorf("site does not exist or is malformed")
	}

	return dockerutil.DockerCompose(
		"-f", composePath,
		"stop",
	)
}

// Wait ensures that the app appears to be read before returning
func (l *LocalApp) Wait() (string, error) {
	o := network.NewHTTPOptions("http://127.0.0.1")
	o.Timeout = 90
	o.Headers["Host"] = l.HostName()
	err := network.EnsureHTTPStatus(o)
	if err != nil {
		return "", fmt.Errorf("200 Was not returned from the web container")
	}

	return l.URL(), nil
}

// FindPorts retrieves the public ports for db and web containers
func (l *LocalApp) FindPorts() error {
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
func (l *LocalApp) Config() error {
	basePath := l.AbsPath()

	err := l.FindPorts()
	if err != nil {
		return err
	}

	settingsFilePath := ""
	if l.GetType() == "drupal7" || l.GetType() == "drupal8" {
		log.Printf("Drupal site. Creating settings.php file.")
		settingsFilePath = path.Join(basePath, "docroot/sites/default/settings.php")
		drupalConfig := model.NewDrupalConfig()
		drupalConfig.DatabaseHost = "db"
		if drupalConfig.HashSalt == "" {
			drupalConfig.HashSalt = stringutil.RandomString(64)
		}
		if l.GetType() == "drupal8" {
			drupalConfig.IsDrupal8 = true
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

		drushSettingsPath := path.Join(basePath, "drush.settings.php")
		drushConfig := model.NewDrushConfig()
		drushConfig.DatabasePort = dbPort
		if l.GetType() == "drupal8" {
			drushConfig.IsDrupal8 = true
		}
		err = config.WriteDrushConfig(drushConfig, drushSettingsPath)

		if err != nil {
			log.Fatalln(err)
		}
	} else if l.GetType() == "wordpress" {
		log.Printf("WordPress site. Creating wp-config.php file.")
		settingsFilePath = path.Join(basePath, "docroot/wp-config.php")
		wpConfig := model.NewWordpressConfig()
		wpConfig.DatabaseHost = "db"
		wpConfig.DeployURL = l.URL()
		wpConfig.AuthKey = stringutil.RandomString(64)
		wpConfig.AuthSalt = stringutil.RandomString(64)
		wpConfig.LoggedInKey = stringutil.RandomString(64)
		wpConfig.LoggedInSalt = stringutil.RandomString(64)
		wpConfig.NonceKey = stringutil.RandomString(64)
		wpConfig.NonceSalt = stringutil.RandomString(64)
		wpConfig.SecureAuthKey = stringutil.RandomString(64)
		wpConfig.SecureAuthSalt = stringutil.RandomString(64)
		err = config.WriteWordpressConfig(wpConfig, settingsFilePath)
		if err != nil {
			log.Fatalln(err)
		}
	}
	return nil
}

// Down stops the docker containers for the local project.
func (l *LocalApp) Down() error {
	composePath := l.DockerComposeYAMLPath()

	err := dockerutil.DockerCompose(
		"-f", composePath,
		"down",
	)
	if err != nil {
		return Cleanup(l)
	}

	return nil
}

// URL returns the URL for a given application.
func (l *LocalApp) URL() string {
	return "http://" + l.AppConfig.Hostname()
}

// HostName returns the hostname of a given application.
func (l *LocalApp) HostName() string {
	return l.AppConfig.Hostname()
}

// AddHostsEntry will add the local site URL to the local hostfile.
func (l *LocalApp) AddHostsEntry() error {
	if os.Getenv("DRUD_NONINTERACTIVE") != "" {
		fmt.Printf("DRUD_NONINTERACTIVE is set. If this message is not in a test you may want to add the following entry to your host file:\n127.0.0.1 %s\n", l.HostName())
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
	hostnameArgs := []string{"ddev", "hostname", l.HostName(), "127.0.0.1"}
	err = system.RunCommandPipe("sudo", hostnameArgs)
	return err
}
