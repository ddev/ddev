package ddevapp

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

	"github.com/drud/ddev/pkg/appimport"
	"github.com/drud/ddev/pkg/appports"
	"github.com/drud/ddev/pkg/archive"
	"github.com/drud/ddev/pkg/cms/config"
	"github.com/drud/ddev/pkg/cms/model"
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/util"
	"github.com/fsouza/go-dockerclient"
	"github.com/lextoumbourou/goodhosts"
	shellwords "github.com/mattn/go-shellwords"
)

const containerWaitTimeout = 35

// SiteRunning defines the string used to denote running sites.
const SiteRunning = "running"

// SiteNotFound defines the string used to denote a site where the containers were not found/do not exist.
const SiteNotFound = "not found"

// SiteDirMissing defines the string used to denote when a site is missing its application directory.
const SiteDirMissing = "app directory missing"

// SiteConfigMissing defines the string used to denote when a site is missing its .ddev/config.yml file.
const SiteConfigMissing = ".ddev/config.yaml missing"

// SiteStopped defines the string used to denote when a site is in the stopped state.
const SiteStopped = "stopped"

// DdevApp is the struct that represents a ddev app, mostly its config
// from config.yaml.
type DdevApp struct {
	APIVersion            string               `yaml:"APIVersion"`
	Name                  string               `yaml:"name"`
	Type                  string               `yaml:"type"`
	Docroot               string               `yaml:"docroot"`
	WebImage              string               `yaml:"webimage"`
	DBImage               string               `yaml:"dbimage"`
	DBAImage              string               `yaml:"dbaimage"`
	ConfigPath            string               `yaml:"-"`
	AppRoot               string               `yaml:"-"`
	Platform              string               `yaml:"-"`
	Provider              string               `yaml:"provider,omitempty"`
	DataDir               string               `yaml:"-"`
	ImportDir             string               `yaml:"-"`
	SiteSettingsPath      string               `yaml:"-"`
	SiteLocalSettingsPath string               `yaml:"-"`
	providerInstance      Provider             `yaml:"-"`
	Commands              map[string][]Command `yaml:"hooks,omitempty"`
}

// GetType returns the application type as a (lowercase) string
func (app *DdevApp) GetType() string {
	return strings.ToLower(app.Type)
}

// Init populates DdevApp config based on the current working directory.
// It does not start the containers.
func (app *DdevApp) Init(basePath string) error {
	newApp, err := NewApp(basePath, "")
	if err != nil {
		return err
	}

	err = newApp.ValidateConfig()
	if err != nil {
		return err
	}

	*app = *newApp
	web, err := app.FindContainerByType("web")

	// if err == nil, it means we have found some containers. Make sure they have
	// the right stuff in them.
	if err == nil {
		containerApproot := web.Labels["com.ddev.approot"]
		if containerApproot != app.AppRoot {
			return fmt.Errorf("a web container in %s state already exists for %s that was created at %s", web.State, app.Name, containerApproot)
		}
		return nil
	} else if strings.Contains(err.Error(), "could not find containers") {
		// Init() is just putting together the DdevApp struct, the containers do
		// not have to exist (app doesn't have to have been started, so the fact
		// we didn't find any is not an error.
		return nil
	}

	return err
}

// FindContainerByType will find a container for this site denoted by the containerType if it is available.
func (app *DdevApp) FindContainerByType(containerType string) (docker.APIContainers, error) {
	labels := map[string]string{
		"com.ddev.site-name":         app.GetName(),
		"com.docker.compose.service": containerType,
	}

	return dockerutil.FindContainerByLabels(labels)
}

// Describe returns a map which provides detailed information on services associated with the running site.
func (app *DdevApp) Describe() (map[string]interface{}, error) {

	shortRoot := RenderHomeRootedDir(app.GetAppRoot())
	appDesc := make(map[string]interface{})

	var https bool
	var httpsURLString string
	httpURLString := fmt.Sprintf("http://%s", app.HostName())
	webCon, err := app.FindContainerByType("web")
	if err == nil {
		https = dockerutil.CheckForHTTPS(webCon)
	}
	if https {
		httpsURLString = fmt.Sprintf("https://%s", app.HostName())
	}

	appDesc["name"] = app.GetName()
	appDesc["status"] = app.SiteStatus()
	appDesc["type"] = app.GetType()
	appDesc["approot"] = app.GetAppRoot()
	appDesc["shortroot"] = shortRoot
	appDesc["httpurl"] = httpURLString
	appDesc["httpsurl"] = httpsURLString

	db, err := app.FindContainerByType("db")
	if err != nil {
		return nil, fmt.Errorf("Failed to find container of type db: %v", err)
	}

	// Only show extended status for running sites.
	if app.SiteStatus() == SiteRunning {
		dbinfo := make(map[string]interface{})
		dbinfo["username"] = "db"
		dbinfo["password"] = "db"
		dbinfo["dbname"] = "db"
		dbinfo["host"] = "db"
		port := appports.GetPort("db")
		dbinfo["port"] = port
		dbPrivatePort, err := strconv.ParseInt(port, 10, 64)
		util.CheckErr(err)
		dbinfo["published_port"] = fmt.Sprint(dockerutil.GetPublishedPort(dbPrivatePort, db))
		appDesc["dbinfo"] = dbinfo

		appDesc["mailhog_url"] = app.GetURL() + ":" + appports.GetPort("mailhog")
		appDesc["phpmyadmin_url"] = app.GetURL() + ":" + appports.GetPort("dba")
	}

	appDesc["router_status"] = GetRouterStatus()

	return appDesc, nil
}

// GetAppRoot return the full path from root to the app directory
func (app *DdevApp) GetAppRoot() string {
	return app.AppRoot
}

// AppConfDir returns the full path to the app's .ddev configuration directory
func (app *DdevApp) AppConfDir() string {
	return filepath.Join(app.AppRoot, ".ddev")
}

// GetDocroot returns the docroot path for ddev app
func (app DdevApp) GetDocroot() string {
	return app.Docroot
}

// GetName returns the app's name
func (app *DdevApp) GetName() string {
	return app.Name
}

// ImportDB takes a source sql dump and imports it to an active site's database container.
func (app *DdevApp) ImportDB(imPath string, extPath string) error {
	app.DockerEnv()
	var extPathPrompt bool
	dbPath := app.ImportDir

	err := app.ProcessHooks("pre-import-db")
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
		output.UserOut.Println("Provide the path to the database you wish to import.")
		fmt.Print("Import path: ")

		imPath = util.GetInput("")
	}

	importPath, err := appimport.ValidateAsset(imPath, "db")
	if err != nil {
		if err.Error() == "is archive" && extPathPrompt {
			output.UserOut.Println("You provided an archive. Do you want to extract from a specific path in your archive? You may leave this blank if you wish to use the full archive contents")
			fmt.Print("Archive extraction path:")

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

	_, _, err = app.Exec("db", "bash", "-c", "cat /db/*.sql | mysql")
	if err != nil {
		return err
	}

	err = app.CreateSettingsFile()
	if err != nil {
		if err.Error() != "app config exists" {
			return fmt.Errorf("failed to write configuration file for %s: %v", app.GetName(), err)
		}
		util.Warning("A custom settings file exists for your application, so ddev did not generate one.")
		util.Warning("Run 'ddev describe' to find the database credentials for this application.")
	}

	if app.GetType() == "wordpress" {
		util.Warning("Wordpress sites require a search/replace of the database when the URL is changed. You can run \"ddev exec 'wp search-replace [http://www.myproductionsite.example] %s'\" to update the URLs across your database. For more information, see http://wp-cli.org/commands/search-replace/", app.GetURL())
	}

	err = fileutil.PurgeDirectory(dbPath)
	if err != nil {
		return fmt.Errorf("failed to clean up %s after import: %v", dbPath, err)
	}

	err = app.ProcessHooks("post-import-db")
	if err != nil {
		return err
	}

	return nil
}

// SiteStatus returns the current status of an application determined from web and db service health.
func (app *DdevApp) SiteStatus() string {
	var siteStatus string
	services := map[string]string{"web": "", "db": ""}

	if !fileutil.FileExists(app.GetAppRoot()) {
		siteStatus = fmt.Sprintf("%s: %v", SiteDirMissing, app.GetAppRoot())
		return siteStatus
	}

	_, err := CheckForConf(app.GetAppRoot())
	if err != nil {
		siteStatus = fmt.Sprintf("%s", SiteConfigMissing)
		return siteStatus
	}

	for service := range services {
		container, err := app.FindContainerByType(service)
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
func (app *DdevApp) Import() error {
	provider, err := app.GetProvider()
	if err != nil {
		return err
	}

	err = provider.Validate()
	if err != nil {
		return err
	}

	if app.SiteStatus() != SiteRunning {
		output.UserOut.Println("Site is not currently running. Starting site before performing import.")
		err = app.Start()
		if err != nil {
			return err
		}
	}

	fileLocation, importPath, err := provider.GetBackup("database")
	if err != nil {
		return err
	}

	output.UserOut.Println("Importing database...")
	err = app.ImportDB(fileLocation, importPath)
	if err != nil {
		return err
	}

	fileLocation, importPath, err = provider.GetBackup("files")
	if err != nil {
		return err
	}

	output.UserOut.Println("Importing files...")
	err = app.ImportFiles(fileLocation, importPath)
	if err != nil {
		return err
	}

	return nil
}

// ImportFiles takes a source directory or archive and copies to the uploaded files directory of a given app.
func (app *DdevApp) ImportFiles(imPath string, extPath string) error {
	var uploadDir string
	var extPathPrompt bool

	app.DockerEnv()

	err := app.ProcessHooks("pre-import-files")
	if err != nil {
		return err
	}

	if imPath == "" {
		// ensure we prompt for extraction path if an archive is provided, while still allowing
		// non-interactive use of --src flag without providing a --extract-path flag.
		if extPath == "" {
			extPathPrompt = true
		}
		output.UserOut.Println("Provide the path to the directory or archive you wish to import. Please note, if the destination directory exists, it will be replaced with the import assets specified here.")
		fmt.Print("Import path: ")

		imPath = util.GetInput("")
	}

	if app.GetType() == "drupal7" || app.GetType() == "drupal8" {
		uploadDir = "sites/default/files"
	}

	if app.GetType() == "wordpress" {
		uploadDir = "wp-content/uploads"
	}

	destPath := filepath.Join(app.GetAppRoot(), app.GetDocroot(), uploadDir)

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
		err = os.MkdirAll(destPath, 0755)
		if err != nil {
			return err
		}
	}

	importPath, err := appimport.ValidateAsset(imPath, "files")
	if err != nil {
		if err.Error() == "is archive" && extPathPrompt {
			output.UserOut.Println("You provided an archive. Do you want to extract from a specific path in your archive? You may leave this blank if you wish to use the full archive contents")
			fmt.Print("Archive extraction path:")

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

	err = app.ProcessHooks("post-import-files")
	if err != nil {
		return err
	}

	return nil
}

// ComposeFiles returns a list of compose files for a project.
func (app *DdevApp) ComposeFiles() []string {
	files, err := filepath.Glob(filepath.Join(app.AppConfDir(), "docker-compose*"))
	if err != nil {
		util.Failed("Failed to load compose files: %v", err)
	}

	for i, file := range files {
		// ensure main docker-compose is first
		match, err := filepath.Match(filepath.Join(app.AppConfDir(), "docker-compose.y*l"), file)
		if err == nil && match {
			files = append(files[:i], files[i+1:]...)
			files = append([]string{file}, files...)
		}
		// ensure override is last
		match, err = filepath.Match(filepath.Join(app.AppConfDir(), "docker-compose.override.y*l"), file)
		if err == nil && match {
			files = append(files, file)
			files = append(files[:i], files[i+1:]...)
		}
	}

	return files
}

// ProcessHooks executes commands defined in a Command
func (app *DdevApp) ProcessHooks(hookName string) error {
	if cmds := app.Commands[hookName]; len(cmds) > 0 {
		output.UserOut.Printf("Executing %s commands...", hookName)
	}

	for _, c := range app.Commands[hookName] {
		if c.Exec != "" {
			output.UserOut.Printf("--- Running exec command: %s ---", c.Exec)

			args, err := shellwords.Parse(c.Exec)
			if err != nil {
				return fmt.Errorf("%s exec failed: %v", hookName, err)
			}

			_, _, err = app.Exec("web", args...)
			if err != nil {
				return fmt.Errorf("%s exec failed: %v", hookName, err)
			}
			util.Success("--- %s exec command succeeded ---", hookName)
		}
		if c.ExecHost != "" {
			output.UserOut.Printf("--- Running host command: %s ---", c.ExecHost)
			args := strings.Split(c.ExecHost, " ")
			cmd := args[0]
			args = append(args[:0], args[1:]...)

			// ensure exec-host runs from consistent location
			cwd, err := os.Getwd()
			util.CheckErr(err)
			err = os.Chdir(app.GetAppRoot())
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
func (app *DdevApp) Start() error {
	app.DockerEnv()

	err := app.ProcessHooks("pre-start")
	if err != nil {
		return err
	}

	// WriteConfig docker-compose.yaml (if it doesn't exist).
	// If the user went through the `ddev config` process it will be written already, but
	// we also do it here in the case of a manually created `.ddev/config.yaml` file.
	err = app.WriteDockerComposeConfig()
	if err != nil {
		return err
	}

	err = app.prepSiteDirs()
	if err != nil {
		return err
	}

	err = app.AddHostsEntry()
	if err != nil {
		return err
	}

	_, _, err = dockerutil.ComposeCmd(app.ComposeFiles(), "up", "-d")
	if err != nil {
		return err
	}

	err = StartDdevRouter()
	if err != nil {
		return err
	}

	err = app.Wait("web", "db")
	if err != nil {
		return err
	}

	err = app.ProcessHooks("post-start")
	if err != nil {
		return err
	}

	return nil
}

// Exec executes a given command in the container of given type without allocating a pty
// Returns ComposeCmd results of stdout, stderr, err
func (app *DdevApp) Exec(service string, cmd ...string) (string, string, error) {
	app.DockerEnv()

	exec := []string{"exec", "-T", service}
	exec = append(exec, cmd...)

	return dockerutil.ComposeCmd(app.ComposeFiles(), exec...)
}

// ExecWithTty executes a given command in the container of given type.
// It allocates a pty for interactive work.
func (app *DdevApp) ExecWithTty(service string, cmd ...string) error {
	app.DockerEnv()

	exec := []string{"exec", service}
	exec = append(exec, cmd...)

	return dockerutil.ComposeNoCapture(app.ComposeFiles(), exec...)
}

// Logs returns logs for a site's given container.
// See docker.LogsOptions for more information about valid tailLines values.
func (app *DdevApp) Logs(service string, follow bool, timestamps bool, tailLines string) error {
	container, err := app.FindContainerByType(service)
	if err != nil {
		return err
	}

	logOpts := docker.LogsOptions{
		Container:    container.ID,
		Stdout:       true,
		Stderr:       true,
		OutputStream: output.UserOut.Out,
		ErrorStream:  output.UserOut.Out,
		Follow:       follow,
		Timestamps:   timestamps,
	}

	if tailLines != "" {
		logOpts.Tail = tailLines
	}

	client := dockerutil.GetDockerClient()

	err = client.Logs(logOpts)
	if err != nil {
		return err
	}

	return nil
}

// DockerEnv sets environment variables for a docker-compose run.
func (app *DdevApp) DockerEnv() {
	envVars := map[string]string{
		"COMPOSE_PROJECT_NAME": "ddev-" + app.Name,
		"DDEV_SITENAME":        app.Name,
		"DDEV_DBIMAGE":         app.DBImage,
		"DDEV_DBAIMAGE":        app.DBAImage,
		"DDEV_WEBIMAGE":        app.WebImage,
		"DDEV_APPROOT":         app.AppRoot,
		"DDEV_DOCROOT":         app.Docroot,
		"DDEV_DATADIR":         app.DataDir,
		"DDEV_IMPORTDIR":       app.ImportDir,
		"DDEV_URL":             app.GetURL(),
		"DDEV_HOSTNAME":        app.HostName(),
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
			if err != nil {
				util.Error("Failed to set the environment variable %s=%s: %v", k, v, err)
			}
		}
	}
}

// Stop initiates docker-compose stop
func (app *DdevApp) Stop() error {
	app.DockerEnv()

	if app.SiteStatus() == SiteNotFound {
		return fmt.Errorf("no site to remove")
	}

	if strings.Contains(app.SiteStatus(), SiteDirMissing) || strings.Contains(app.SiteStatus(), SiteConfigMissing) {
		return fmt.Errorf("ddev can no longer find your application files at %s. If you would like to continue using ddev to manage this site please restore your files to that directory. If you would like to remove this site from ddev, you may run 'ddev remove %s'", app.GetAppRoot(), app.GetName())
	}

	_, _, err := dockerutil.ComposeCmd(app.ComposeFiles(), "stop")

	if err != nil {
		return err
	}

	return StopRouter()
}

// Wait ensures that the app service containers are healthy.
func (app *DdevApp) Wait(containerTypes ...string) error {
	for _, containerType := range containerTypes {
		labels := map[string]string{
			"com.ddev.site-name":         app.GetName(),
			"com.docker.compose.service": containerType,
		}
		err := dockerutil.ContainerWait(containerWaitTimeout, labels)
		if err != nil {
			return fmt.Errorf("%s service %v", containerType, err)
		}
	}

	return nil
}

func (app *DdevApp) determineSettingsPath() (string, error) {
	possibleLocations := []string{app.SiteSettingsPath, app.SiteLocalSettingsPath}
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

// CreateSettingsFile creates the app's settings.php or equivalent,
// adding things like database host, name, and password
func (app *DdevApp) CreateSettingsFile() error {
	// If neither settings file options are set, then
	if app.SiteLocalSettingsPath == "" && app.SiteSettingsPath == "" {
		return nil
	}

	settingsFilePath, err := app.determineSettingsPath()
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

	switch app.GetType() {
	case "drupal8":
		fallthrough
	case "drupal7":
		output.UserOut.Printf("Generating %s file for database connection.", fileName)
		drushSettingsPath := filepath.Join(app.GetAppRoot(), "drush.settings.php")

		// Retrieve published mysql port for drush settings file.
		db, err := app.FindContainerByType("db")
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

		if app.GetType() == "drupal8" {
			drupalConfig.IsDrupal8 = true
			drushConfig.IsDrupal8 = true
		}

		drupalConfig.DeployURL = app.GetURL()
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
		output.UserOut.Printf("Generating %s file for database connection.", fileName)
		wpConfig := model.NewWordpressConfig()
		wpConfig.DeployURL = app.GetURL()
		err := config.WriteWordpressConfig(wpConfig, settingsFilePath)
		if err != nil {
			return err
		}
	}
	return nil
}

// Down stops the docker containers for the project in current directory.
func (app *DdevApp) Down(removeData bool) error {
	app.DockerEnv()
	settingsFilePath := app.SiteSettingsPath

	// Remove all the containers and volumes for app.
	err := Cleanup(app)
	if err != nil {
		return fmt.Errorf("Failed to remove %s: %s", app.GetName(), err)
	}

	// Remove data/database if we need to.
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
		// Check that app.DataDir is a directory that is safe to remove.
		err = validateDataDirRemoval(app)
		if err != nil {
			return fmt.Errorf("failed to remove data directories: %v", err)
		}
		// mysql data can be set to read-only on linux hosts. PurgeDirectory ensures files
		// are writable before we attempt to remove them.
		if !fileutil.FileExists(app.DataDir) {
			util.Warning("No application data to remove")
		} else {
			err := fileutil.PurgeDirectory(app.DataDir)
			if err != nil {
				return fmt.Errorf("failed to remove data directories: %v", err)
			}
			// PurgeDirectory leaves the directory itself in place, so we remove it here.
			err = os.RemoveAll(app.DataDir)
			if err != nil {
				return fmt.Errorf("failed to remove data directory %s: %v", app.DataDir, err)
			}
			util.Success("Application data removed")
		}
	}

	err = StopRouter()
	return err
}

// GetURL returns the URL for an app.
func (app *DdevApp) GetURL() string {
	return "http://" + app.GetHostname()
}

// HostName returns the hostname of a given application.
func (app *DdevApp) HostName() string {
	return app.GetHostname()
}

// AddHostsEntry will add the site URL to the host's /etc/hosts.
func (app *DdevApp) AddHostsEntry() error {
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
		util.Failed("could not open hostfile. %s", err)
	}
	if hosts.Has(dockerIP, app.HostName()) {
		return nil
	}

	_, err = osexec.Command("sudo", "-h").Output()
	if (os.Getenv("DRUD_NONINTERACTIVE") != "") || err != nil {
		util.Warning("You must manually add the following entry to your hosts file:\n%s %s\nOr with root/administrative privileges execute 'ddev hostname %s %s'", dockerIP, app.HostName(), app.HostName(), dockerIP)
		return nil
	}

	ddevFullpath, err := os.Executable()
	util.CheckErr(err)

	output.UserOut.Printf("ddev needs to add an entry to your hostfile.\nIt will require root privileges via the sudo command, so you may be required\nto enter your password for sudo. ddev is about to issue the command:")
	hostnameArgs := []string{ddevFullpath, "hostname", app.HostName(), dockerIP}
	command := strings.Join(hostnameArgs, " ")
	util.Warning(fmt.Sprintf("    sudo %s", command))
	output.UserOut.Println("Please enter your password if prompted.")
	err = exec.RunCommandPipe("sudo", hostnameArgs)
	return err
}

// prepSiteDirs creates a site's directories for db container mounts
func (app *DdevApp) prepSiteDirs() error {

	dirs := []string{
		app.DataDir,
		app.ImportDir,
	}

	for _, dir := range dirs {
		fileInfo, err := os.Stat(dir)

		if os.IsNotExist(err) { // If it doesn't exist, create it.
			err = os.MkdirAll(dir, os.FileMode(int(0774)))
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
		_, err = CheckForConf(siteDir)
		if err != nil {
			return "", fmt.Errorf("Could not find a site in %s. Have you run 'ddev config'? Please specify a site name or change directories: %s", siteDir, err)
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
		// In the case of a missing .ddev/config.yml just return the site directory.
		return siteDir, nil
	}

	return appRoot, nil
}

// GetActiveApp returns the active App based on the current working directory or running siteName provided.
// To use the current working directory, siteName should be ""
func GetActiveApp(siteName string) (*DdevApp, error) {
	app := &DdevApp{}
	activeAppRoot, err := GetActiveAppRoot(siteName)
	if err != nil {
		return app, err
	}

	// Ignore app.Init() error, since app.Init() fails if no directory found.
	// We already were successful with *finding* the app, and if we get an
	// incomplete one we have to add to it.
	_ = app.Init(activeAppRoot)

	if app.Name == "" || app.DataDir == "" {
		err = restoreApp(app, siteName)
		if err != nil {
			return app, err
		}
	}

	return app, nil
}

// restoreApp recreates an AppConfig's Name and/or DataDir and returns an error
// if it cannot restore them.
func restoreApp(app *DdevApp, siteName string) error {
	if siteName == "" {
		return fmt.Errorf("error restoring AppConfig: no siteName given")
	}
	app.Name = siteName
	// Ensure that AppConfig.DataDir is set so that site data can be removed if necessary.
	dataDir := fmt.Sprintf("%s/%s", util.GetGlobalDdevDir(), app.GetName())
	app.DataDir = dataDir

	return nil
}

// validateDataDirRemoval validates that dataDir is a safe filepath to be removed by ddev.
func validateDataDirRemoval(app *DdevApp) error {
	dataDir := app.DataDir
	unsafeFilePathErr := fmt.Errorf("filepath: %s unsafe for removal", dataDir)
	// Check for an empty filepath
	if dataDir == "" {
		return unsafeFilePathErr
	}
	// Get the current working directory.
	currDir, err := os.Getwd()
	if err != nil {
		return err
	}
	// Check that dataDir is not the current directory.
	if dataDir == currDir {
		return unsafeFilePathErr
	}
	// Get the last element of dataDir and use it to check that there is something after GlobalDdevDir.
	lastPathElem := filepath.Base(dataDir)
	nextLastPathElem := filepath.Base(filepath.Dir(dataDir))
	if lastPathElem == ".ddev" || nextLastPathElem != app.Name || lastPathElem == "" {
		return unsafeFilePathErr
	}
	return nil
}

// GetProvider returns a pointer to the provider instance interface.
func (app *DdevApp) GetProvider() (Provider, error) {
	if app.providerInstance != nil {
		return app.providerInstance, nil
	}

	var provider Provider
	err := fmt.Errorf("unknown provider type: %s", app.Provider)

	switch app.Provider {
	case "pantheon":
		provider = &PantheonProvider{}
		err = provider.Init(app)
	case DefaultProviderName:
		provider = &DefaultProvider{}
		err = nil
	default:
		provider = &DefaultProvider{}
		// Use the default error from above.
	}
	app.providerInstance = provider
	return app.providerInstance, err
}
