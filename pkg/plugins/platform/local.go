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
	shellwords "github.com/mattn/go-shellwords"
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
	config, err := ddevapp.NewConfig(basePath, "")
	if err != nil {
		return err
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
		return "", fmt.Errorf("no site found. have you run `ddev start`?")
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
	dbPath := l.AppConfig.ImportDir

	err := l.ProcessHooks("pre-import-db")
	if err != nil {
		return err
	}

	err = fileutil.PurgeDirectory(dbPath)
	if err != nil {
		return fmt.Errorf("failed to cleanup %s before import: %v", dbPath, err)
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
		return fmt.Errorf("failed to clean up %s after import: %v", dbPath, err)
	}

	err = l.ProcessHooks("post-import-db")
	if err != nil {
		return err
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

// Import performs an import from the a configured provider plugin, if one exists.
func (l *LocalApp) Import() error {
	provider, err := l.AppConfig.GetProvider()
	if err != nil {
		return err
	}

	err = provider.Validate()
	if err != nil {
		return err
	}

	if l.SiteStatus() != SiteRunning {
		fmt.Println("Site is not currently running. Starting site before performing import.")
		err := l.Start()
		if err != nil {
			return err
		}
	}

	fileLocation, importPath, err := provider.GetBackup("database")
	if err != nil {
		return err
	}

	fmt.Println("Importing database...")
	err = l.ImportDB(fileLocation, importPath)
	if err != nil {
		return err
	}

	fileLocation, importPath, err = provider.GetBackup("files")
	if err != nil {
		return err
	}

	fmt.Println("Importing files...")
	err = l.ImportFiles(fileLocation, importPath)
	if err != nil {
		return err
	}

	return nil
}

// ImportFiles takes a source directory or archive and copies to the uploaded files directory of a given app.
func (l *LocalApp) ImportFiles(imPath string, extPath string) error {
	var uploadDir string
	var extPathPrompt bool

	l.DockerEnv()

	err := l.ProcessHooks("pre-import-files")
	if err != nil {
		return err
	}

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
	err = os.Chmod(filepath.Dir(destPath), 0755)
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

	err = l.ProcessHooks("post-import-files")
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

// ProcessHooks executes commands defined in a ddevapp.Command
func (l *LocalApp) ProcessHooks(hookName string) error {
	if cmds := l.AppConfig.Commands[hookName]; len(cmds) > 0 {
		fmt.Printf("Executing %s commands...\n", hookName)
	}

	for _, c := range l.AppConfig.Commands[hookName] {
		if c.Exec != "" {
			fmt.Printf("--- Running exec command: %s ---\n", c.Exec)

			args, err := shellwords.Parse(c.Exec)
			if err != nil {
				return fmt.Errorf("%s exec failed: %v", hookName, err)
			}

			err = l.Exec("web", true, args...)
			if err != nil {
				return fmt.Errorf("%s exec failed: %v", hookName, err)
			}
			util.Success("--- %s exec command succeeded ---", hookName)
		}
		if c.ExecHost != "" {
			fmt.Printf("--- Running host command: %s ---\n", c.ExecHost)
			args := strings.Split(c.ExecHost, " ")
			cmd := args[0]
			args = append(args[:0], args[1:]...)

			// ensure exec-host runs from consistent location
			cwd, err := os.Getwd()
			util.CheckErr(err)
			err = os.Chdir(l.AppRoot())
			util.CheckErr(err)

			err = exec.RunCommandPipe(cmd, args)
			dirErr := os.Chdir(cwd)
			util.CheckErr(dirErr)
			if err != nil {
				return fmt.Errorf("%s host command failed: %v", hookName, err)
			}
			util.Success("--- %s host command succeeded ---", hookName)
		}
	}

	return nil
}

// Start initiates docker-compose up
func (l *LocalApp) Start() error {
	l.DockerEnv()

	err := l.ProcessHooks("pre-start")
	if err != nil {
		return err
	}

	// Write docker-compose.yaml (if it doesn't exist).
	// If the user went through the `ddev config` process it will be written already, but
	// we also do it here in the case of a manually created `.ddev/config.yaml` file.
	err = l.AppConfig.WriteDockerComposeConfig()
	if err != nil {
		return err
	}

	err = l.prepSiteDirs()
	if err != nil {
		return err
	}

	err = l.AddHostsEntry()
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

	err = l.ProcessHooks("post-start")
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
		"DDEV_DATADIR":         l.AppConfig.DataDir,
		"DDEV_IMPORTDIR":       l.AppConfig.ImportDir,
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

	if l.SiteStatus() == SiteNotFound {
		return fmt.Errorf("no site to remove")
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

func (l *LocalApp) determineConfigLocation() (string, error) {
	possibleLocations := []string{l.AppConfig.SiteSettingsPath, l.AppConfig.SiteLocalSettingsPath}
	for _, loc := range possibleLocations {
		// If the file is found we need to check for a signature to determine if it's safe to use.
		if fileutil.FileExists(loc) {
			signatureFound, err := fileutil.FgrepStringInFile(loc, model.DdevSettingsFileSignature)
			util.CheckErr(err) // Really can't happen as we already checked for the file existence

			if signatureFound {
				return loc, nil
			}
		} else {
			// If the file is not found it's safe to use.
			return loc, nil
		}
	}

	return "", fmt.Errorf("settings files already exist and are being manged by the user")
}

// Config creates the apps config file adding things like database host, name, and password
// as well as other sensitive data like salts.
func (l *LocalApp) Config() error {
	// If neither settings file options are set, then
	if l.AppConfig.SiteLocalSettingsPath == "" && l.AppConfig.SiteSettingsPath == "" {
		return nil
	}

	settingsFilePath, err := l.determineConfigLocation()
	if err != nil {
		return err
	}

	// Drupal and WordPress love to change settings files to be unwriteable. Chmod them to something we can work with
	// in the event that they already exist.
	chmodTargets := []string{filepath.Dir(settingsFilePath), settingsFilePath}
	for _, fp := range chmodTargets {
		if fileInfo, err := os.Stat(fp); !os.IsNotExist(err) {
			perms := 0644
			if fileInfo.IsDir() {
				perms = 0755
			}

			err = os.Chmod(fp, os.FileMode(perms))
			if err != nil {
				return fmt.Errorf("could not change permissions on %s to make the file writeable", fp)
			}
		}
	}

	fileName := filepath.Base(settingsFilePath)

	switch l.GetType() {
	case "drupal8":
		fallthrough
	case "drupal7":
		fmt.Printf("Generating %s file for database connection.\n", fileName)
		drushSettingsPath := filepath.Join(l.AppRoot(), "drush.settings.php")

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
		fmt.Printf("Generating %s file for database connection.\n", fileName)
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
func (l *LocalApp) Down(removeData bool) error {
	l.DockerEnv()
	settingsFilePath := l.AppConfig.SiteSettingsPath

	err := dockerutil.ComposeCmd(l.ComposeFiles(), "down", "-v")
	if err != nil {
		util.Warning("Could not stop site with docker-compose. Attempting manual cleanup.")
		err = Cleanup(l)
		if err != nil {
			util.Warning("Received error from Cleanup, err=", err)
		}
	}

	if removeData {
		if fileutil.FileExists(settingsFilePath) {
			signatureFound, err := fileutil.FgrepStringInFile(settingsFilePath, model.DdevSettingsFileSignature)
			util.CheckErr(err) // Really can't happen as we already checked for the file existence
			if signatureFound {
				err = os.Chmod(settingsFilePath, 0644)
				if err != nil {
					return err
				}
				err = os.Remove(settingsFilePath)
				if err != nil {
					return err
				}
			}
		}
		dir := filepath.Dir(l.AppConfig.DataDir)
		// mysql data can be set to read-only on linux hosts. PurgeDirectory ensures files
		// are writable before we attempt to remove them.
		if !fileutil.FileExists(dir) {
			util.Warning("No application data to remove")
		} else {
			err := fileutil.PurgeDirectory(dir)
			if err != nil {
				return fmt.Errorf("failed to remove data directories: %v", err)
			}
			// PurgeDirectory leaves the directory itself in place, so we remove it here.
			err = os.RemoveAll(dir)
			if err != nil {
				return fmt.Errorf("failed to remove data directories: %v", err)
			}
			util.Success("Application data removed")
		}
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
	hosts, err := goodhosts.NewHosts()
	if err != nil {
		log.Fatalf("could not open hostfile. %s", err)
	}
	if hosts.Has(dockerIP, l.HostName()) {
		return nil
	}

	_, err = osexec.Command("sudo", "-h").Output()
	if (os.Getenv("DRUD_NONINTERACTIVE") != "") || err != nil {
		util.Warning("You must manually add the following entry to your hosts file:\n%s %s", dockerIP, l.HostName())
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

// prepSiteDirs creates a site's directories for db container mounts
func (l *LocalApp) prepSiteDirs() error {

	dirs := []string{
		l.AppConfig.DataDir,
		l.AppConfig.ImportDir,
	}

	for _, dir := range dirs {
		fileInfo, err := os.Stat(dir)

		if os.IsNotExist(err) { // If it doesn't exist, create it.
			err := os.MkdirAll(dir, os.FileMode(int(0774)))
			if err != nil {
				return fmt.Errorf("Failed to create directory %s, err: %v", dir, err)
			}
		} else if err == nil && fileInfo.IsDir() { // If the directory exists, we're fine and don't have to create it.
			continue
		} else { // But otherwise it must have existed as a file, so bail
			return fmt.Errorf("Error where trying to create directory %s, err: %v", dir, err)
		}
	}

	return nil
}

// GetActiveAppRoot returns the fully rooted directory of the active app, or an error
func GetActiveAppRoot(siteName string) (string, error) {
	var siteDir string
	var err error

	if siteName == "" {
		siteDir, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("error determining the current directory: %s", err)
		}
	} else {
		var ok bool

		labels := map[string]string{
			"com.ddev.site-name":         siteName,
			"com.docker.compose.service": "web",
		}

		webContainer, err := dockerutil.FindContainerByLabels(labels)
		if err != nil {
			return "", fmt.Errorf("could not find a site named '%s'. Run 'ddev list' to see currently active sites", siteName)
		}

		siteDir, ok = webContainer.Labels["com.ddev.approot"]
		if !ok {
			return "", fmt.Errorf("could not determine the location of %s from container: %s", siteName, dockerutil.ContainerName(webContainer))
		}
	}

	appRoot, err := CheckForConf(siteDir)
	if err != nil {
		return "", fmt.Errorf("unable to determine the application for this command. Have you run 'ddev config'? Error: %s", err)
	}

	return appRoot, nil
}

// GetActiveApp returns the active App based on the current working directory or running siteName provided.
func GetActiveApp(siteName string) (App, error) {
	app, err := GetPluginApp("local")
	if err != nil {
		return app, err
	}
	activeAppRoot, err := GetActiveAppRoot(siteName)
	if err != nil {
		return app, err
	}

	err = app.Init(activeAppRoot)
	return app, err
}
