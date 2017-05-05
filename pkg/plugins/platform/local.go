package platform

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"

	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/drud/ddev/pkg/appimport"
	"github.com/drud/ddev/pkg/appports"
	"github.com/drud/ddev/pkg/cms/config"
	"github.com/drud/ddev/pkg/cms/model"
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/util"
	"github.com/drud/drud-go/utils/stringutil"
	"github.com/drud/drud-go/utils/system"
	"github.com/fsouza/go-dockerclient"
	"github.com/gosuri/uitable"
	"github.com/lextoumbourou/goodhosts"
)

// LocalApp implements the AppBase interface local development apps
type LocalApp struct {
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
		return fmt.Errorf("could not find an active ddev configuration, have you run 'ddev config'?: %v", err)
	}

	l.AppConfig = config

	err = PrepLocalSiteDirs(basePath)
	if err != nil {
		log.Fatalln(err)
	}

	web, err := l.FindContainerByType("web")
	if err == nil {
		containerApproot := web.Labels["com.ddev.approot"]
		if containerApproot != l.AppConfig.AppRoot {
			return fmt.Errorf("a web container in %s state already exists for %s that was created at %s", web.State, l.AppConfig.Name, containerApproot)
		}
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
	appTable := CreateAppTable()
	RenderAppRow(appTable, l)
	output = fmt.Sprint(appTable)

	output = output + "\n\nMySQL Credentials\n-----------------\n"
	dbTable := uitable.New()
	dbTable.MaxColWidth = maxWidth
	dbTable.AddRow("Username:", "root")
	dbTable.AddRow("Password:", "root")
	dbTable.AddRow("Database name:", "data")
	dbTable.AddRow("Connection Info:", l.HostName()+":"+appports.GetPort("db"))
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

// AppConfDir returns the full path to the app's .ddev configuration directory
func (l *LocalApp) AppConfDir() string {
	return path.Join(l.AppConfig.AppRoot, ".ddev")
}

// Docroot returns the docroot path for local app
func (l LocalApp) Docroot() string {
	return l.AppConfig.Docroot
}

// GetName returns the  name for local app
func (l *LocalApp) GetName() string {
	return l.AppConfig.Name
}

// ImportDB takes a source sql dump and imports it to an active site's database container.
func (l *LocalApp) ImportDB(imPath string) error {
	l.DockerEnv()
	dbPath := path.Join(l.AppRoot(), ".ddev", "data")

	if imPath == "" {
		fmt.Println("Provide the path to the database you wish to import.")
		fmt.Println("Import path: ")

		imPath = util.GetInput("")
	}

	importPath, err := appimport.ValidateAsset(imPath, "db")
	if err != nil {
		if err.Error() == "is archive" {
			if strings.HasSuffix(importPath, "sql.gz") {
				err := util.Ungzip(importPath, dbPath)
				if err != nil {
					return fmt.Errorf("failed to extract provided archive: %v", err)
				}
			} else {
				err := util.Untar(importPath, dbPath)
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
		err = util.CopyFile(importPath, path.Join(dbPath, "db.sql"))
		if err != nil {
			return err
		}
	}

	err = l.Exec("db", true, "./import.sh")
	if err != nil {
		return fmt.Errorf("failed to execute import: %v", err)
	}

	err = l.Config()
	if err != nil {
		if err.Error() != "app config exists" {
			return fmt.Errorf("failed to write configuration file for %s: %v", l.GetName(), err)
		}
		fmt.Println("A settings file already exists for your application, so ddev did not generate one.")
		fmt.Println("Run 'ddev describe' to find the database credentials for this application.")
	}

	if l.GetType() == "wordpress" {
		util.Warning("Wordpress sites require a search/replace of the database when the URL is changed. You can run \"ddev exec 'wp search-replace [http://www.myproductionsite.example] %s'\" to update the URLs across your database. For more information, see http://wp-cli.org/commands/search-replace/", l.URL())
	}

	return nil
}

// SiteStatus returns the current status of an application based on the web container.
func (l *LocalApp) SiteStatus() string {
	webContainer, err := l.FindContainerByType("web")
	if err != nil {
		return "not found"
	}

	status := util.GetContainerHealth(webContainer)
	if status == "exited" {
		return "stopped"
	}
	if status == "healthy" {
		return "running"
	}

	return status
}

// ImportFiles takes a source directory or archive and copies to the uploaded files directory of a given app.
func (l *LocalApp) ImportFiles(imPath string) error {
	var uploadDir string
	l.DockerEnv()

	if imPath == "" {
		fmt.Println("Provide the path to the directory or archive you wish to import. Please note, if the destination directory exists, it will be replaced with the import assets specified here.")
		fmt.Println("Import path: ")

		imPath = util.GetInput("")
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
		err = util.Untar(importPath, destPath)
		if err != nil {
			return fmt.Errorf("failed to extract provided archive: %v", err)
		}
		return nil
	}

	err = util.CopyDir(importPath, destPath)
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

// ComposeFiles returns a list of compose files for a project.
func (l *LocalApp) ComposeFiles() []string {
	files, err := filepath.Glob(filepath.Join(l.AppConfDir(), "docker-compose*"))
	if err != nil {
		log.Fatalf("Failed to load compose files: %v", err)
	}

	for i, file := range files {
		// ensure main docker-compose is first
		match, err := filepath.Match(filepath.Join(l.AppConfDir(), "docker-compose.y*l"), file)
		if err == nil && match {
			files = append(files[:i], files[i+1:]...)
			files = append([]string{file}, files...)
		}
		// ensure override is last
		match, err = filepath.Match(filepath.Join(l.AppConfDir(), "docker-compose.override.y*l"), file)
		if err == nil && match {
			files = append(files, file)
			files = append(files[:i], files[i+1:]...)
		}
	}

	return files
}

// Start initiates docker-compose up
func (l *LocalApp) Start() error {
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

	StartDockerRouter()

	err := l.AddHostsEntry()
	if err != nil {
		return err
	}

	return util.ComposeCmd(l.ComposeFiles(), "up", "-d")
}

// Exec executes a given command in the container of given type.
func (l *LocalApp) Exec(service string, tty bool, cmd ...string) error {
	l.DockerEnv()

	var exec []string
	if tty {
		exec = []string{"exec", "-T", service}
	} else {
		exec = []string{"exec", service}
	}
	exec = append(exec, cmd...)

	return util.ComposeCmd(l.ComposeFiles(), exec...)
}

// Logs returns logs for a site's given container.
func (l *LocalApp) Logs(service string, follow bool, timestamps bool, tail string) error {
	container, err := l.FindContainerByType(service)
	if err != nil {
		return err
	}

	logOpts := docker.LogsOptions{
		Container:    container.ID,
		Stdout:       true,
		Stderr:       true,
		OutputStream: os.Stdout,
		ErrorStream:  os.Stderr,
		Follow:       follow,
		Timestamps:   timestamps,
	}

	if tail != "" {
		logOpts.Tail = tail
	}

	client := util.GetDockerClient()

	err = client.Logs(logOpts)
	if err != nil {
		return err
	}

	return nil
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
	l.DockerEnv()

	if l.SiteStatus() != "running" {
		return fmt.Errorf("site does not appear to be running - web container %s", l.SiteStatus())
	}

	err := util.ComposeCmd(l.ComposeFiles(), "stop")

	if err != nil {
		return err
	}
	containersRunning, err := ddevContainersRunning()
	if err != nil {
		return err
	}

	if !containersRunning {
		return StopRouter()
	}
	return nil
}

// Wait ensures that the app appears to be read before returning
func (l *LocalApp) Wait(containerType string) error {
	labels := map[string]string{
		"com.ddev.site-name":      l.GetName(),
		"com.ddev.container-type": containerType,
	}
	err := util.ContainerWait(90, labels)
	if err != nil {
		return err
	}

	return nil
}

// Config creates the apps config file adding things like database host, name, and password
// as well as other sensitive data like salts.
func (l *LocalApp) Config() error {
	basePath := l.AppRoot()
	docroot := l.Docroot()
	settingsFilePath := path.Join(basePath, docroot)

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
		err := config.WriteDrupalConfig(drupalConfig, settingsFilePath)
		if err != nil {
			log.Fatalln(err)
		}

		// Setup a custom settings file for use with drush.
		dbPort, err := util.GetPodPort("local-" + l.GetName() + "-db")
		if err != nil {
			return err
		}

		drushSettingsPath := path.Join(basePath, "drush.settings.php")
		drushConfig := model.NewDrushConfig()
		drushConfig.DatabasePort = strconv.FormatInt(dbPort, 10)
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
		err := config.WriteWordpressConfig(wpConfig, settingsFilePath)
		if err != nil {
			log.Fatalln(err)
		}
	}
	return nil
}

// Down stops the docker containers for the local project.
func (l *LocalApp) Down() error {
	l.DockerEnv()
	err := util.ComposeCmd(l.ComposeFiles(), "down")
	if err != nil {
		util.Warning("Could not stop site with docker-compose. Attempting manual cleanup.")
		return Cleanup(l)
	}
	containersRunning, err := ddevContainersRunning()
	if err != nil {
		return err
	}

	if !containersRunning {
		return StopRouter()
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

	ddevFullpath, err := os.Executable()
	util.CheckErr(err)

	fmt.Println("ddev needs to add an entry to your hostfile.\nIt will require root privileges via the sudo command, so you may be required\nto enter your password for sudo. ddev is about to issue the command:")
	hostnameArgs := []string{ddevFullpath, "hostname", l.HostName(), "127.0.0.1"}
	command := strings.Join(hostnameArgs, " ")
	util.Warning(fmt.Sprintf("    sudo %s", command))
	fmt.Println("Please enter your password if prompted.")
	err = system.RunCommandPipe("sudo", hostnameArgs)
	return err
}
