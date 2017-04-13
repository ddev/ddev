package platform

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"time"

	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/drud/ddev/pkg/appimport"
	"github.com/drud/ddev/pkg/appports"
	"github.com/drud/ddev/pkg/cms/config"
	"github.com/drud/ddev/pkg/cms/model"
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/util"
	"github.com/drud/ddev/pkg/util/files"
	"github.com/drud/ddev/pkg/util/prompt"
	"github.com/drud/drud-go/utils/dockerutil"
	"github.com/drud/drud-go/utils/stringutil"
	"github.com/drud/drud-go/utils/system"
	"github.com/fsouza/go-dockerclient"
	"github.com/gosuri/uitable"
	"github.com/lextoumbourou/goodhosts"
)

// LocalApp implements the AppBase interface local development apps
type LocalApp struct {
	AppBase
	AppConfig *ddevapp.Config
}

// GetType returns the application type as a (lowercase) string
func (l *LocalApp) GetType() string {
	return strings.ToLower(l.AppConfig.AppType)
}

// Init populates LocalApp settings based on the current working directory.
func (l *LocalApp) Init(basePath string) error {
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

// FindContainerByType will find a container for this site denoted by the containerType if it is available.
func (l *LocalApp) FindContainerByType(containerType string) (docker.APIContainers, error) {
	labels := map[string]string{
		"com.ddev.site-name":      l.GetName(),
		"com.ddev.container-type": containerType,
	}

	return util.FindContainerByLabels(labels)
}

// Describe returns a string which provides detailed information on services associated with the running site.
func (l *LocalApp) Describe() (string, error) {
	maxWidth := uint(200)
	web, err := l.FindContainerByType("web")
	if err != nil {
		return "", err
	}
	db, err := l.FindContainerByType("db")
	if err != nil {
		return "", err
	}

	if db.State != "running" || web.State != "running" {
		return "", fmt.Errorf("ddev site is configured but not currently running")
	}

	var output string
	app := uitable.New()
	app.MaxColWidth = maxWidth
	app.AddRow("NAME", "LOCATION", "TYPE", "URL", "STATUS")
	app.AddRow(l.GetName(), l.AppRoot(), l.GetType(), l.URL(), "running")
	output = fmt.Sprint(app)

	output = output + "\n\nMySQL Credentials\n-----------------\n"
	dbTable := uitable.New()
	dbTable.MaxColWidth = maxWidth
	dbTable.AddRow("Username:", "root")
	dbTable.AddRow("Password:", "root")
	dbTable.AddRow("Database name:", "data")
	dbTable.AddRow("Connection Info:", l.HostName()+":3306")
	output = output + fmt.Sprint(dbTable)

	output = output + "\n\nOther Services\n--------------\n"
	other := uitable.New()
	other.AddRow("MailHog:", l.URL()+":"+appports.GetPort("mailhog"))
	other.AddRow("phpMyAdmin:", l.URL()+":"+appports.GetPort("dba"))
	output = output + fmt.Sprint(other)
	return output, nil
}

// AppRoot return the full path from root to the app directory
func (l *LocalApp) AppRoot() string {
	return l.AppConfig.AppRoot
}

// Docroot returns the docroot path for local app
func (l LocalApp) Docroot() string {
	return l.AppConfig.Docroot
}

// GetName returns the  name for local app
func (l *LocalApp) GetName() string {
	return l.AppConfig.Name
}

// ContainerPrefix returns the base name for local app containers
func (l *LocalApp) ContainerPrefix() string {
	return "local"
}

// ContainerName returns the base name for local app containers
func (l *LocalApp) ContainerName() string {
	return fmt.Sprintf("%s-%s", l.ContainerPrefix(), l.GetName())
}

// ImportDB takes a source sql dump and imports it to an active site's database container.
func (l *LocalApp) ImportDB(imPath string) error {
	l.DockerEnv()
	container := fmt.Sprintf("%s-db", l.ContainerName())
	dbPath := path.Join(l.AppRoot(), ".ddev", "data")

	if imPath == "" {
		fmt.Println("Provide the path to the database you wish to import.")
		fmt.Println("Import path: ")

		imPath = prompt.GetInput("")
	}

	importPath, err := appimport.ValidateAsset(imPath, "db")
	if err != nil {
		if err.Error() == "is archive" {
			if strings.HasSuffix(importPath, "sql.gz") {
				err := files.Ungzip(importPath, dbPath)
				if err != nil {
					return fmt.Errorf("failed to extract provided archive: %v", err)
				}
			} else {
				err := files.Untar(importPath, dbPath)
				if err != nil {
					return fmt.Errorf("failed to extract provided archive: %v", err)
				}
			}
			// empty the path so we don't try to copy
			importPath = ""
		} else {
			return err
		}
	}

	// an archive was not extracted, we need to copy
	if importPath != "" {
		err = files.CopyFile(importPath, path.Join(dbPath, "db.sql"))
		if err != nil {
			return err
		}
	}

	err = appimport.ImportSQLDump(l.DockerComposeYAMLPath(), container)
	if err != nil {
		return err
	}

	err = l.Config()
	if err != nil {
		if err.Error() != "app config exists" {
			return fmt.Errorf("failed to write configuration file for %s: %v", l.GetName(), err)
		}
		fmt.Println("A settings file already exists for your application, so ddev did not generate one.")
		fmt.Println("Run 'ddev describe' to find the database credentials for this application.")
	}

	return nil
}

// ImportFiles takes a source directory or archive and copies to the uploaded files directory of a given app.
func (l *LocalApp) ImportFiles(imPath string) error {
	var uploadDir string
	l.DockerEnv()

	if imPath == "" {
		fmt.Println("Provide the path to the directory or archive you wish to import. Please note, if the destination directory exists, it will be replaced with the import assets specified here.")
		fmt.Println("Import path: ")

		imPath = prompt.GetInput("")
	}

	if l.GetType() == "drupal7" || l.GetType() == "drupal8" {
		uploadDir = "sites/default/files"
	}

	if l.GetType() == "wordpress" {
		uploadDir = "wp-content/uploads"
	}

	destPath := path.Join(l.AppRoot(), l.Docroot(), uploadDir)

	// parent of destination dir should exist
	if !system.FileExists(path.Dir(destPath)) {
		return fmt.Errorf("unable to import to %s: parent directory does not exist", destPath)
	}

	// parent of destination dir should be writable
	err := os.Chmod(path.Dir(destPath), 0755)
	if err != nil {
		return err
	}

	// destination dir should not exist
	if system.FileExists(destPath) {
		err := os.RemoveAll(destPath)
		if err != nil {
			return err
		}
	}

	importPath, err := appimport.ValidateAsset(imPath, "files")
	if err != nil {
		if err.Error() != "is archive" {
			return err
		}
		err = files.Untar(importPath, destPath)
		if err != nil {
			return fmt.Errorf("failed to extract provided archive: %v", err)
		}
		return nil
	}

	err = files.CopyDir(importPath, destPath)
	if err != nil {
		return err
	}

	return nil
}

// DockerComposeYAMLPath returns the absolute path to where the docker-compose.yaml should exist for this app configuration.
// This is a bit redundant, but is here to avoid having to expose too many details of AppConfig.
func (l *LocalApp) DockerComposeYAMLPath() string {
	return l.AppConfig.DockerComposeYAMLPath()
}

// Start initiates docker-compose up
func (l *LocalApp) Start() error {
	composePath := l.DockerComposeYAMLPath()
	l.DockerEnv()

	// Write docker-compose.yaml (if it doesn't exist).
	// If the user went through the `ddev config` process it will be written already, but
	// we also do it here in the case of a manually created `.ddev/config.yaml` file.
	if !system.FileExists(l.AppConfig.DockerComposeYAMLPath()) {
		err := l.AppConfig.WriteDockerComposeConfig()
		if err != nil {
			return err
		}
	}

	EnsureDockerRouter()

	err := l.AddHostsEntry()
	if err != nil {
		log.Fatal(err)
	}

	return dockerutil.DockerCompose(
		"-f", composePath,
		"up",
		"-d",
	)
}

// DockerEnv sets environment variables for a docker-compose run.
func (l *LocalApp) DockerEnv() {
	envVars := map[string]string{
		"COMPOSE_PROJECT_NAME": "ddev-" + l.AppConfig.Name,
		"DDEV_SITENAME":        l.AppConfig.Name,
		"DDEV_DBIMAGE":         l.AppConfig.DBImage,
		"DDEV_DBAIMAGE":        l.AppConfig.DBAImage,
		"DDEV_WEBIMAGE":        l.AppConfig.WebImage,
		"DDEV_APPROOT":         l.AppConfig.AppRoot,
		"DDEV_DOCROOT":         filepath.Join(l.AppConfig.AppRoot, l.AppConfig.Docroot),
		"DDEV_URL":             l.URL(),
		"DDEV_HOSTNAME":        l.HostName(),
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
func (l *LocalApp) Stop() error {
	composePath := l.DockerComposeYAMLPath()
	l.DockerEnv()

	if !dockerutil.IsRunning(l.ContainerName()+"-db") && !dockerutil.IsRunning(l.ContainerName()+"-web") && !ComposeFileExists(l) {
		return fmt.Errorf("site does not exist or is malformed")
	}

	return dockerutil.DockerCompose(
		"-f", composePath,
		"stop",
	)
}

// Wait ensures that the app appears to be read before returning
func (l *LocalApp) Wait() (string, error) {
	err := l.ContainerWait(90, "web")
	if err != nil {
		return "", err
	}

	return l.URL(), nil
}

// FindPorts retrieves the public ports for db and web containers
func (l *LocalApp) FindPorts() error {
	var err error
	l.WebPublicPort, err = util.GetPodPort(l.ContainerName() + "-web")
	if err != nil {
		return err
	}

	l.DbPublicPort, err = util.GetPodPort(l.ContainerName() + "-db")
	return err
}

// Config creates the apps config file adding things like database host, name, and password
// as well as other sensitive data like salts.
func (l *LocalApp) Config() error {
	basePath := l.AppRoot()
	docroot := l.Docroot()
	settingsFilePath := path.Join(basePath, docroot)

	err := l.FindPorts()
	if err != nil {
		return err
	}

	if l.GetType() == "drupal7" || l.GetType() == "drupal8" {
		settingsFilePath = path.Join(settingsFilePath, "sites/default/settings.php")
	}

	if l.GetType() == "wordpress" {
		settingsFilePath = path.Join(settingsFilePath, "wp-config.php")
	}

	if system.FileExists(settingsFilePath) {
		return errors.New("app config exists")
	}

	if l.GetType() == "drupal7" || l.GetType() == "drupal8" {
		fmt.Println("Generating settings.php file for database connection.")
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
		dbPort, err := util.GetPodPort(l.ContainerName() + "-db")
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
		fmt.Println("Generating wp-config.php file for database connection.")
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
	l.DockerEnv()
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

// ContainerWait provides a wait loop to check for container in "healthy" status.
func (l *LocalApp) ContainerWait(timeout time.Duration, containerType string) error {
	timedOut := time.After(timeout * time.Second)
	tick := time.Tick(500 * time.Millisecond)
	for {
		select {
		case <-timedOut:
			return errors.New("health check timed out")
		case <-tick:
			container, err := l.FindContainerByType(containerType)
			if err != nil {
				return errors.New("failed to query container")
			}
			status := GetContainerHealth(container)
			if status == "healthy" {
				return nil
			}
		}
	}
}
