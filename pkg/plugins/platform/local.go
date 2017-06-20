package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"strings"

	"net/url"
	osexec "os/exec"

	"os/user"
	"runtime"

	"errors"

	log "github.com/Sirupsen/logrus"
	"github.com/drud/ddev/pkg/appimport"
	"github.com/drud/ddev/pkg/appports"
	"github.com/drud/ddev/pkg/archive"
	"github.com/drud/ddev/pkg/cms/config"
	"github.com/drud/ddev/pkg/cms/model"
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/util"
	"github.com/fsouza/go-dockerclient"
	"github.com/gosuri/uitable"
	"github.com/lextoumbourou/goodhosts"
)

const containerWaitTimeout = 35

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

	err = config.Validate()
	if err != nil {
		return err
	}

	l.AppConfig = config

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
		"com.ddev.site-name":         l.GetName(),
		"com.docker.compose.service": containerType,
	}

	return dockerutil.FindContainerByLabels(labels)
}

// Describe returns a string which provides detailed information on services associated with the running site.
func (l *LocalApp) Describe() (string, error) {
	maxWidth := uint(200)
	var output string
	siteStatus := l.SiteStatus()

	// Do not show any describe output if we can't find the site.
	if siteStatus == SiteNotFound {
		return "", fmt.Errorf("no site found. have you ran `ddev start`?")
	}
	appTable := CreateAppTable()

	RenderAppRow(appTable, l)
	output = fmt.Sprint(appTable)

	db, err := l.FindContainerByType("db")
	if err != nil {
		return "", err
	}

	dbPrivatePort, err := strconv.ParseInt(appports.GetPort("db"), 10, 64)
	if err != nil {
		return "", err
	}

	dbPublishPort := fmt.Sprint(dockerutil.GetPublishedPort(dbPrivatePort, db))

	// Only show extended status for running sites.
	if siteStatus == SiteRunning {
		output = output + "\n\nMySQL Credentials\n-----------------\n"
		dbTable := uitable.New()
		dbTable.MaxColWidth = maxWidth
		dbTable.AddRow("Username:", "db")
		dbTable.AddRow("Password:", "db")
		dbTable.AddRow("Database name:", "db")
		dbTable.AddRow("Host:", "db")
		dbTable.AddRow("Port:", appports.GetPort("db"))
		output = output + fmt.Sprint(dbTable)
		output = output + fmt.Sprintf("\nTo connect to mysql from your host machine, use port %[1]v on 127.0.0.1.\nFor example: mysql --host=127.0.0.1 --port=%[1]v --user=db --password=db --database=db", dbPublishPort)

		output = output + "\n\nOther Services\n--------------\n"
		other := uitable.New()
		other.AddRow("MailHog:", l.URL()+":"+appports.GetPort("mailhog"))
		other.AddRow("phpMyAdmin:", l.URL()+":"+appports.GetPort("dba"))
		output = output + fmt.Sprint(other)
	}

	output = output + "\n" + PrintRouterStatus()

	return output, nil
}

// AppRoot return the full path from root to the app directory
func (l *LocalApp) AppRoot() string {
	return l.AppConfig.AppRoot
}

// AppConfDir returns the full path to the app's .ddev configuration directory
func (l *LocalApp) AppConfDir() string {
	return filepath.Join(l.AppConfig.AppRoot, ".ddev")
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
func (l *LocalApp) ImportDB(imPath string, extPath string) error {
	l.DockerEnv()
	var extPathPrompt bool
	dbPath := filepath.Join(l.AppRoot(), ".ddev", "data")

	err := fileutil.PurgeDirectory(dbPath)
	if err != nil {
		return fmt.Errorf("failed to cleanup .ddev/data before import: %v", err)
	}

	if imPath == "" {
		// ensure we prompt for extraction path if an archive is provided, while still allowing
		// non-interactive use of --src flag without providing a --extract-path flag.
		if extPath == "" {
			extPathPrompt = true
		}
		fmt.Println("Provide the path to the database you wish to import.")
		fmt.Println("Import path: ")

		imPath = util.GetInput("")
	}

	importPath, err := appimport.ValidateAsset(imPath, "db")
	if err != nil {
		if err.Error() == "is archive" && extPathPrompt {
			fmt.Println("You provided an archive. Do you want to extract from a specific path in your archive? You may leave this blank if you wish to use the full archive contents")
			fmt.Println("Archive extraction path:")

			extPath = util.GetInput("")
		}
		if err.Error() != "is archive" {
			return err
		}
	}

	switch {
	case strings.HasSuffix(importPath, "sql.gz"):
		err = archive.Ungzip(importPath, dbPath)
		if err != nil {
			return fmt.Errorf("failed to extract provided archive: %v", err)
		}

	case strings.HasSuffix(importPath, "zip"):
		err = archive.Unzip(importPath, dbPath, extPath)
		if err != nil {
			return fmt.Errorf("failed to extract provided archive: %v", err)
		}

	case strings.HasSuffix(importPath, "tar"):
		fallthrough
	case strings.HasSuffix(importPath, "tar.gz"):
		fallthrough
	case strings.HasSuffix(importPath, "tgz"):
		err := archive.Untar(importPath, dbPath, extPath)
		if err != nil {
			return fmt.Errorf("failed to extract provided archive: %v", err)
		}

	default:
		err = fileutil.CopyFile(importPath, filepath.Join(dbPath, "db.sql"))
		if err != nil {
			return err
		}
	}

	matches, err := filepath.Glob(filepath.Join(dbPath, "*.sql"))
	if err != nil {
		return err
	}

	if len(matches) < 1 {
		return fmt.Errorf("no .sql files found to import")
	}

	fmt.Println("Importing database...")
	err = l.Exec("db", true, "bash", "-c", "cat /db/*.sql | mysql")
	if err != nil {
		return err
	}

	err = l.Config()
	if err != nil {
		if err.Error() != "app config exists" {
			return fmt.Errorf("failed to write configuration file for %s: %v", l.GetName(), err)
		}
		util.Warning("A custom settings file exists for your application, so ddev did not generate one.")
		util.Warning("Run 'ddev describe' to find the database credentials for this application.")
	}

	if l.GetType() == "wordpress" {
		util.Warning("Wordpress sites require a search/replace of the database when the URL is changed. You can run \"ddev exec 'wp search-replace [http://www.myproductionsite.example] %s'\" to update the URLs across your database. For more information, see http://wp-cli.org/commands/search-replace/", l.URL())
	}

	err = fileutil.PurgeDirectory(dbPath)
	if err != nil {
		return fmt.Errorf("failed to cleanup .ddev/data after import: %v", err)
	}

	return nil
}

// SiteStatus returns the current status of an application determined from web and db service health.
func (l *LocalApp) SiteStatus() string {
	var siteStatus string
	services := map[string]string{"web": "", "db": ""}

	for service := range services {
		container, err := l.FindContainerByType(service)
		if err != nil {
			services[service] = SiteNotFound
			siteStatus = service + " service " + SiteNotFound
		} else {
			status := dockerutil.GetContainerHealth(container)

			switch status {
			case "exited":
				services[service] = SiteStopped
				siteStatus = service + " service " + SiteStopped
			case "healthy":
				services[service] = SiteRunning
			default:
				services[service] = status
			}
		}
	}

	if services["web"] == services["db"] {
		siteStatus = services["web"]
	} else {
		for service, status := range services {
			if status != SiteRunning {
				siteStatus = service + " service " + status
			}
		}
	}

	return siteStatus
}

// ImportFiles takes a source directory or archive and copies to the uploaded files directory of a given app.
func (l *LocalApp) ImportFiles(imPath string, extPath string) error {
	var uploadDir string
	var extPathPrompt bool

	l.DockerEnv()

	if imPath == "" {
		// ensure we prompt for extraction path if an archive is provided, while still allowing
		// non-interactive use of --src flag without providing a --extract-path flag.
		if extPath == "" {
			extPathPrompt = true
		}
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

	destPath := filepath.Join(l.AppRoot(), l.Docroot(), uploadDir)

	// parent of destination dir should exist
	if !fileutil.FileExists(filepath.Dir(destPath)) {
		return fmt.Errorf("unable to import to %s: parent directory does not exist", destPath)
	}

	// parent of destination dir should be writable
	err := os.Chmod(filepath.Dir(destPath), 0755)
	if err != nil {
		return err
	}

	if fileutil.FileExists(destPath) {
		// ensure existing directory is empty
		err := fileutil.PurgeDirectory(destPath)
		if err != nil {
			return fmt.Errorf("failed to cleanup %s before import: %v", destPath, err)
		}
	} else {
		// create destination directory
		err := os.MkdirAll(destPath, 0755)
		if err != nil {
			return err
		}
	}

	importPath, err := appimport.ValidateAsset(imPath, "files")
	if err != nil {
		if err.Error() == "is archive" && extPathPrompt {
			fmt.Println("You provided an archive. Do you want to extract from a specific path in your archive? You may leave this blank if you wish to use the full archive contents")
			fmt.Println("Archive extraction path:")

			extPath = util.GetInput("")
		}
		if err.Error() != "is archive" {
			return err
		}
	}

	switch {
	case strings.HasSuffix(importPath, "tar"):
		fallthrough
	case strings.HasSuffix(importPath, "tar.gz"):
		fallthrough
	case strings.HasSuffix(importPath, "tgz"):
		err = archive.Untar(importPath, destPath, extPath)
		if err != nil {
			return fmt.Errorf("failed to extract provided archive: %v", err)
		}
	case strings.HasSuffix(importPath, "zip"):
		err = archive.Unzip(importPath, destPath, extPath)
		if err != nil {
			return fmt.Errorf("failed to extract provided archive: %v", err)
		}

	default:
		// Simple file copy if none of the archive formats
		err = fileutil.CopyDir(importPath, destPath)
		if err != nil {
			return err
		}
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
	if !fileutil.FileExists(l.AppConfig.DockerComposeYAMLPath()) {
		err := l.AppConfig.WriteDockerComposeConfig()
		if err != nil {
			return err
		}
	}

	err := l.AddHostsEntry()
	if err != nil {
		return err
	}

	err = dockerutil.ComposeCmd(l.ComposeFiles(), "up", "-d")
	if err != nil {
		return err
	}

	err = StartDdevRouter()
	if err != nil {
		return err
	}

	err = l.Wait("web", "db")
	if err != nil {
		return err
	}

	return nil
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

	return dockerutil.ComposeCmd(l.ComposeFiles(), exec...)
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

	client := dockerutil.GetDockerClient()

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
		"DDEV_DOCROOT":         l.AppConfig.Docroot,
		"DDEV_URL":             l.URL(),
		"DDEV_HOSTNAME":        l.HostName(),
		"DDEV_UID":             "",
		"DDEV_GID":             "",
	}
	if runtime.GOOS == "linux" {
		curUser, err := user.Current()
		util.CheckErr(err)

		envVars["DDEV_UID"] = curUser.Uid
		envVars["DDEV_GID"] = curUser.Gid
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

	if l.SiteStatus() != SiteRunning {
		return fmt.Errorf("site does not appear to be running - web container %s", l.SiteStatus())
	}

	err := dockerutil.ComposeCmd(l.ComposeFiles(), "stop")

	if err != nil {
		return err
	}

	return StopRouter()
}

// Wait ensures that the app service containers are healthy.
func (l *LocalApp) Wait(containerTypes ...string) error {
	for _, containerType := range containerTypes {
		labels := map[string]string{
			"com.ddev.site-name":         l.GetName(),
			"com.docker.compose.service": containerType,
		}
		err := dockerutil.ContainerWait(containerWaitTimeout, labels)
		if err != nil {
			return fmt.Errorf("%s service %v", containerType, err)
		}
	}

	return nil
}

// Config creates the apps config file adding things like database host, name, and password
// as well as other sensitive data like salts.
func (l *LocalApp) Config() error {
	basePath := l.AppRoot()
	docroot := l.Docroot()
	settingsFilePath := filepath.Join(basePath, docroot)

	switch l.GetType() {
	case "drupal8":
		fallthrough
	case "drupal7":
		settingsFilePath = filepath.Join(settingsFilePath, "sites", "default", "settings.php")
		if fileutil.FileExists(settingsFilePath) {
			signatureFound, err := fileutil.FgrepStringInFile(settingsFilePath, model.DdevSettingsFileSignature)
			util.CheckErr(err) // Really can't happen as we already checked for the file existence
			if !signatureFound {
				return errors.New("app config exists")
			}
			// Otherwise we'll go on our way and recreate the settings file.
		}

		drushSettingsPath := filepath.Join(basePath, "drush.settings.php")

		// Retrieve published mysql port for drush settings file.
		db, err := l.FindContainerByType("db")
		if err != nil {
			return err
		}

		dbPrivatePort, err := strconv.ParseInt(appports.GetPort("db"), 10, 64)
		if err != nil {
			return err
		}
		dbPublishPort := dockerutil.GetPublishedPort(dbPrivatePort, db)

		fmt.Println("Generating settings.php file for database connection.")

		drupalConfig := model.NewDrupalConfig()
		drushConfig := model.NewDrushConfig()

		if l.GetType() == "drupal8" {
			drupalConfig.IsDrupal8 = true
			drushConfig.IsDrupal8 = true
		}

		drupalConfig.DeployURL = l.URL()
		err = config.WriteDrupalConfig(drupalConfig, settingsFilePath)
		if err != nil {
			return err
		}

		drushConfig.DatabasePort = strconv.FormatInt(dbPublishPort, 10)
		err = config.WriteDrushConfig(drushConfig, drushSettingsPath)
		if err != nil {
			return err
		}
	case "wordpress":
		settingsFilePath = filepath.Join(settingsFilePath, "wp-config.php")
		if fileutil.FileExists(settingsFilePath) {
			signatureFound, err := fileutil.FgrepStringInFile(settingsFilePath, model.DdevSettingsFileSignature)
			util.CheckErr(err) // Really can't happen as we already checked for the file existence
			if !signatureFound {
				return errors.New("app config exists")
			}
			// Otherwise we'll go on our way and recreate the settings file.
		}

		fmt.Println("Generating wp-config.php file for database connection.")
		wpConfig := model.NewWordpressConfig()
		wpConfig.DeployURL = l.URL()
		err := config.WriteWordpressConfig(wpConfig, settingsFilePath)
		if err != nil {
			return err
		}
	}
	return nil
}

// Down stops the docker containers for the local project.
func (l *LocalApp) Down() error {
	l.DockerEnv()
	err := dockerutil.ComposeCmd(l.ComposeFiles(), "down", "-v")
	if err != nil {
		util.Warning("Could not stop site with docker-compose. Attempting manual cleanup.")
		return Cleanup(l)
	}

	return StopRouter()
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
	dockerIP := "127.0.0.1"
	dockerHostRawURL := os.Getenv("DOCKER_HOST")
	if dockerHostRawURL != "" {
		dockerHostURL, err := url.Parse(dockerHostRawURL)
		if err != nil {
			return fmt.Errorf("Failed to parse $DOCKER_HOST: %v, err: %v", dockerHostRawURL, err)
		}
		dockerIP = dockerHostURL.Hostname()
	}

	_, err := osexec.Command("sudo", "-h").Output()
	if (os.Getenv("DRUD_NONINTERACTIVE") != "") || err != nil {
		fmt.Printf("You must manually add the following entry to your host file:\n%s %s\n", dockerIP, l.HostName())
		return nil
	}

	hosts, err := goodhosts.NewHosts()
	if err != nil {
		log.Fatalf("could not open hostfile. %s", err)
	}
	if hosts.Has(dockerIP, l.HostName()) {
		return nil
	}

	ddevFullpath, err := os.Executable()
	util.CheckErr(err)

	fmt.Println("ddev needs to add an entry to your hostfile.\nIt will require root privileges via the sudo command, so you may be required\nto enter your password for sudo. ddev is about to issue the command:")
	hostnameArgs := []string{ddevFullpath, "hostname", l.HostName(), dockerIP}
	command := strings.Join(hostnameArgs, " ")
	util.Warning(fmt.Sprintf("    sudo %s", command))
	fmt.Println("Please enter your password if prompted.")
	err = exec.RunCommandPipe("sudo", hostnameArgs)
	return err
}
