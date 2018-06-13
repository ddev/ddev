package ddevapp

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"golang.org/x/crypto/ssh/terminal"

	"strings"

	osexec "os/exec"

	"os/user"

	"runtime"

	"github.com/drud/ddev/pkg/appimport"
	"github.com/drud/ddev/pkg/appports"
	"github.com/drud/ddev/pkg/archive"
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/util"
	"github.com/fsouza/go-dockerclient"
	"github.com/lextoumbourou/goodhosts"
	"github.com/mattn/go-shellwords"
)

const containerWaitTimeout = 61

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

// DdevFileSignature is the text we use to detect whether a settings file is managed by us.
// If this string is found, we assume we can replace/update the file.
const DdevFileSignature = "#ddev-generated"

// DdevApp is the struct that represents a ddev app, mostly its config
// from config.yaml.
type DdevApp struct {
	APIVersion            string               `yaml:"APIVersion"`
	Name                  string               `yaml:"name"`
	Type                  string               `yaml:"type"`
	Docroot               string               `yaml:"docroot"`
	PHPVersion            string               `yaml:"php_version"`
	WebImage              string               `yaml:"webimage,omitempty"`
	DBImage               string               `yaml:"dbimage,omitempty"`
	DBAImage              string               `yaml:"dbaimage,omitempty"`
	RouterHTTPPort        string               `yaml:"router_http_port"`
	RouterHTTPSPort       string               `yaml:"router_https_port"`
	XdebugEnabled         bool                 `yaml:"xdebug_enabled"`
	AdditionalHostnames   []string             `yaml:"additional_hostnames"`
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
			return fmt.Errorf("a project (web container) in %s state already exists for %s that was created at %s", web.State, app.Name, containerApproot)
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

	appDesc["name"] = app.GetName()
	appDesc["hostnames"] = app.GetHostnames()
	appDesc["status"] = app.SiteStatus()
	appDesc["type"] = app.GetType()
	appDesc["approot"] = app.GetAppRoot()
	appDesc["shortroot"] = shortRoot
	appDesc["httpurl"] = app.GetHTTPURL()
	appDesc["httpsurl"] = app.GetHTTPSURL()
	appDesc["urls"] = app.GetAllURLs()

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

		appDesc["mailhog_url"] = "http://" + app.GetHostname() + ":" + appports.GetPort("mailhog")
		appDesc["phpmyadmin_url"] = "http://" + app.GetHostname() + ":" + appports.GetPort("dba")
	}

	appDesc["router_status"] = GetRouterStatus()
	appDesc["php_version"] = app.GetPhpVersion()
	appDesc["router_http_port"] = app.RouterHTTPPort
	appDesc["router_https_port"] = app.RouterHTTPSPort
	appDesc["xdebug_enabled"] = app.XdebugEnabled

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

// GetPhpVersion returns the app's php version
func (app *DdevApp) GetPhpVersion() string {
	v := DdevDefaultPHPVersion
	if app.PHPVersion != "" {
		v = app.PHPVersion
	}
	return v
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
	case strings.HasSuffix(importPath, "sql.gz") || strings.HasSuffix(importPath, "mysql.gz"):
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

	matches, err := filepath.Glob(filepath.Join(dbPath, "*.*sql"))
	if err != nil {
		return err
	}

	if len(matches) < 1 {
		return fmt.Errorf("no .sql or .mysql files found to import")
	}

	_, _, err = app.Exec("db", "bash", "-c", "mysql --database=mysql -e 'DROP DATABASE IF EXISTS db; CREATE DATABASE db;' && cat /db/*.*sql | mysql db")
	if err != nil {
		return err
	}

	_, err = app.CreateSettingsFile()
	if err != nil {
		util.Warning("A custom settings file exists for your application, so ddev did not generate one.")
		util.Warning("Run 'ddev describe' to find the database credentials for this application.")
	}

	err = app.PostImportDBAction()
	if err != nil {
		return fmt.Errorf("failed to execute PostImportDBAction: %v", err)
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

	if uploadDir == "" {
		util.Failed("No upload directory has been specified for the project type %s", app.GetType())
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
// It has to put the docker-compose.y*l first
// It has to put the docker-compose.override.y*l last
func (app *DdevApp) ComposeFiles() ([]string, error) {
	files, err := filepath.Glob(filepath.Join(app.AppConfDir(), "docker-compose*.y*l"))
	if err != nil || len(files) == 0 {
		return []string{}, fmt.Errorf("failed to load any docker-compose.*y*l files: %v", err)
	}

	mainfiles, err := filepath.Glob(filepath.Join(app.AppConfDir(), "docker-compose.y*l"))
	// Glob doesn't return many errors, so just CheckErr()
	util.CheckErr(err)
	if len(mainfiles) == 0 {
		return []string{}, fmt.Errorf("failed to find a docker-compose.yml or docker-compose.yaml")

	}
	if len(mainfiles) > 1 {
		return []string{}, fmt.Errorf("there are more than one docker-compose.y*l, unable to continue")
	}

	overrides, err := filepath.Glob(filepath.Join(app.AppConfDir(), "docker-compose.override.y*l"))
	util.CheckErr(err)
	if len(overrides) > 1 {
		return []string{}, fmt.Errorf("there are more than one docker-compose.override.y*l, unable to continue")
	}

	orderedFiles := make([]string, 1)

	// Make sure the docker-compose.yaml goes first
	orderedFiles[0] = mainfiles[0]

	for _, file := range files {
		// We already have the main docker-compose.yaml, so skip when we hit it.
		// We'll add the override later, so skip it.
		if file == mainfiles[0] || (len(overrides) == 1 && file == overrides[0]) {
			continue
		}
		orderedFiles = append(orderedFiles, file)
	}
	if len(overrides) == 1 {
		orderedFiles = append(orderedFiles, overrides[0])
	}
	return orderedFiles, nil
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

			stdout, stderr, err := app.Exec("web", args...)
			if err != nil {
				return fmt.Errorf("%s exec failed: %v, stderr='%s'", hookName, err, stderr)
			}
			util.Success("--- %s exec command succeeded, output below ---", hookName)
			output.UserOut.Println(stdout + "\n" + stderr)
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

			out, err := exec.RunCommandPipe(cmd, args)
			dirErr := os.Chdir(cwd)
			util.CheckErr(dirErr)
			output.UserOut.Println(out)
			if err != nil {
				return fmt.Errorf("%s host command failed: %v %s", hookName, err, out)
			}
			util.Success("--- %s host command succeeded ---\n", hookName)
		}

		if c.RExec != "" {
			output.UserOut.Printf("--- Running rexec command: %s ---", c.RExec)

			args, err := shellwords.Parse(c.RExec)
			if err != nil {
				return fmt.Errorf("%s exec failed: %v", hookName, err)
			}

			stdout, stderr, err := app.ExecRoot("web", args...)
			if err != nil {
				return fmt.Errorf("%s rexec failed: %v, stderr='%s'", hookName, err, stderr)
			}

			util.Success("--- %s rexec command succeeded, output below ---", hookName)
			output.UserOut.Println(stdout + "\n" + stderr)
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

	// Warn the user if there is any custom configuration in use.
	app.CheckCustomConfig()

	// WriteConfig docker-compose.yaml
	err = app.WriteDockerComposeConfig()
	if err != nil {
		return err
	}

	err = app.prepSiteDirs()
	if err != nil {
		return err
	}

	err = app.AddHostsEntries()
	if err != nil {
		return err
	}

	files, err := app.ComposeFiles()
	if err != nil {
		return err
	}
	_, _, err = dockerutil.ComposeCmd(files, "up", "-d")
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

	ex := []string{"exec", "-T", service}
	ex = append(ex, cmd...)

	files, err := app.ComposeFiles()
	if err != nil {
		return "", "", err
	}

	return dockerutil.ComposeCmd(files, ex...)
}

// ExecRoot executes a given comment as root in the container of given type without allocating a pty
// Returns ComposeCmd results of stdout, stderr, err
func (app *DdevApp) ExecRoot(service string, cmd ...string) (string, string, error) {
	app.DockerEnv()

	ex := []string{"exec", "-uroot", "-T", service}
	ex = append(ex, cmd...)

	files, err := app.ComposeFiles()
	if err != nil {
		return "", "", err
	}

	return dockerutil.ComposeCmd(files, ex...)
}

// ExecWithTty executes a given command in the container of given type.
// It allocates a pty for interactive work.
func (app *DdevApp) ExecWithTty(service string, cmd ...string) error {
	app.DockerEnv()

	exec := []string{"exec", service}
	exec = append(exec, cmd...)

	files, err := app.ComposeFiles()
	if err != nil {
		return err
	}
	return dockerutil.ComposeNoCapture(files, exec...)
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
	curUser, err := user.Current()
	util.CheckErr(err)

	var uidInt, gidInt int
	uidStr := curUser.Uid
	gidStr := curUser.Gid
	// For windows the uidStr/gidStr are usually way outside linux range (ends at 60000)
	// so we have to run as root. We may have a host uidStr/gidStr greater in other contexts,
	// bail and run as root.
	if uidInt, err = strconv.Atoi(curUser.Uid); err != nil {
		uidStr = "0"
	}
	if gidInt, err = strconv.Atoi(curUser.Gid); err != nil {
		gidStr = "0"
	}

	// Warn about running as root if we're not on windows.
	if runtime.GOOS != "windows" && (uidInt > 60000 || gidInt > 60000 || uidInt == 0) {
		util.Warning("Warning: containers will run as root. This could be a security risk on Linux.")
	}

	// If the uidStr or gidStr is outside the range possible in container, use root
	if uidInt > 60000 || gidInt > 60000 {
		uidStr = "0"
		gidStr = "0"
	}

	envVars := map[string]string{
		"COMPOSE_PROJECT_NAME":          "ddev-" + app.Name,
		"COMPOSE_CONVERT_WINDOWS_PATHS": "true",
		"DDEV_SITENAME":                 app.Name,
		"DDEV_DBIMAGE":                  app.DBImage,
		"DDEV_DBAIMAGE":                 app.DBAImage,
		"DDEV_WEBIMAGE":                 app.WebImage,
		"DDEV_APPROOT":                  app.AppRoot,
		"DDEV_DOCROOT":                  app.Docroot,
		"DDEV_DATADIR":                  app.DataDir,
		"DDEV_IMPORTDIR":                app.ImportDir,
		"DDEV_URL":                      app.GetHTTPURL(),
		"DDEV_HOSTNAME":                 app.HostName(),
		"DDEV_UID":                      uidStr,
		"DDEV_GID":                      gidStr,
		"DDEV_PHP_VERSION":              app.PHPVersion,
		"DDEV_PROJECT_TYPE":             app.Type,
		"DDEV_ROUTER_HTTP_PORT":         app.RouterHTTPPort,
		"DDEV_ROUTER_HTTPS_PORT":        app.RouterHTTPSPort,
		"DDEV_XDEBUG_ENABLED":           strconv.FormatBool(app.XdebugEnabled),
	}

	// Find out terminal dimensions
	columns, lines, err := terminal.GetSize(0)
	if err != nil {
		columns = 80
		lines = 24
	}
	envVars["COLUMNS"] = strconv.Itoa(columns)
	envVars["LINES"] = strconv.Itoa(lines)

	if len(app.AdditionalHostnames) > 0 {
		// TODO: Warn people about additional names in use.
		envVars["DDEV_HOSTNAME"] = strings.Join(app.GetHostnames(), ",")
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
		return fmt.Errorf("ddev can no longer find your project files at %s. If you would like to continue using ddev to manage this project please restore your files to that directory. If you would like to remove this site from ddev, you may run 'ddev remove %s'", app.GetAppRoot(), app.GetName())
	}

	files, err := app.ComposeFiles()
	if err != nil {
		return err
	}
	_, _, err = dockerutil.ComposeCmd(files, "stop")

	if err != nil {
		return err
	}

	return StopRouterIfNoContainers()
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

// DetermineSettingsPathLocation figures out the path to the settings file for
// an app based on the contents/existence of app.SiteSettingsPath and
// app.SiteLocalSettingsPath.
func (app *DdevApp) DetermineSettingsPathLocation() (string, error) {
	possibleLocations := []string{app.SiteSettingsPath, app.SiteLocalSettingsPath}
	for _, loc := range possibleLocations {
		// If the file is found we need to check for a signature to determine if it's safe to use.
		if fileutil.FileExists(loc) {
			signatureFound, err := fileutil.FgrepStringInFile(loc, DdevFileSignature)
			util.CheckErr(err) // Really can't happen as we already checked for the file existence

			if signatureFound {
				return loc, nil
			}
		} else {
			// If the file is not found it's safe to use.
			return loc, nil
		}
	}

	return "", fmt.Errorf("settings files already exist and are being managed by the user")
}

// Down stops the docker containers for the project in current directory.
func (app *DdevApp) Down(removeData bool) error {
	app.DockerEnv()

	err := app.Stop()
	if err != nil {
		util.Warning("Failed to stop containers for %s: %v", app.GetName(), err)
	}

	// Remove all the containers and volumes for app.
	err = Cleanup(app)
	if err != nil {
		return fmt.Errorf("Failed to remove %s: %s", app.GetName(), err)
	}

	// Remove data/database if we need to.
	if removeData {
		// Check that app.DataDir is a directory that is safe to remove.
		err = validateDataDirRemoval(app)
		if err != nil {
			return fmt.Errorf("failed to remove data/database directories: %v", err)
		}
		// mysql data can be set to read-only on linux hosts. PurgeDirectory ensures files
		// are writable before we attempt to remove them.
		if !fileutil.FileExists(app.DataDir) {
			util.Warning("No project data/database to remove")
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
			util.Success("Project data/database removed")
		}
	}

	err = StopRouterIfNoContainers()
	return err
}

// GetHTTPURL returns the HTTP URL for an app.
func (app *DdevApp) GetHTTPURL() string {
	url := "http://" + app.GetHostname()
	if app.RouterHTTPPort != "80" {
		url = url + ":" + app.RouterHTTPPort
	}
	return url
}

// GetHTTPSURL returns the HTTPS URL for an app.
func (app *DdevApp) GetHTTPSURL() string {
	url := "https://" + app.GetHostname()
	if app.RouterHTTPSPort != "443" {
		url = url + ":" + app.RouterHTTPSPort
	}
	return url
}

// GetAllURLs returns an array of all the URLs for the project
func (app *DdevApp) GetAllURLs() []string {
	var URLs []string

	// Get configured URLs
	for _, name := range app.GetHostnames() {
		httpPort := ""
		httpsPort := ""
		if app.RouterHTTPPort != "80" {
			httpPort = ":" + app.RouterHTTPPort
		}
		if app.RouterHTTPSPort != "443" {
			httpsPort = ":" + app.RouterHTTPSPort
		}
		URLs = append(URLs, "http://"+name+httpPort, "https://"+name+httpsPort)
	}

	// Get direct address of web container
	dockerIP, err := dockerutil.GetDockerIP()
	if err != nil {
		util.Error("Unable to get Docker IP: %v", err)
		return URLs
	}

	webContainer, err := app.FindContainerByType("web")
	if err != nil {
		util.Error("Unable to find web container for app: %s, err %v", app.Name, err)
		return URLs
	}

	for _, p := range webContainer.Ports {
		if p.PrivatePort == 80 {
			URLs = append(URLs, fmt.Sprintf("http://%s:%d", dockerIP, p.PublicPort))
			break
		}
	}

	return URLs
}

// HostName returns the hostname of a given application.
func (app *DdevApp) HostName() string {
	return app.GetHostname()
}

// AddHostsEntries will add the site URL to the host's /etc/hosts.
func (app *DdevApp) AddHostsEntries() error {
	dockerIP, err := dockerutil.GetDockerIP()
	if err != nil {
		return fmt.Errorf("could not get Docker IP: %v", err)
	}

	hosts, err := goodhosts.NewHosts()
	if err != nil {
		util.Failed("could not open hostfile. %s", err)
	}
	for _, name := range app.GetHostnames() {

		if hosts.Has(dockerIP, name) {
			continue
		}

		_, err = osexec.LookPath("sudo")
		if (os.Getenv("DRUD_NONINTERACTIVE") != "") || err != nil {
			util.Warning("You must manually add the following entry to your hosts file:\n%s %s\nOr with root/administrative privileges execute 'ddev hostname %s %s'", dockerIP, name, name, dockerIP)
			return nil
		}

		ddevFullpath, err := os.Executable()
		util.CheckErr(err)

		output.UserOut.Printf("ddev needs to add an entry to your hostfile.\nIt will require administrative privileges via the sudo command, so you may be required\nto enter your password for sudo. ddev is about to issue the command:")

		hostnameArgs := []string{ddevFullpath, "hostname", name, dockerIP}
		command := strings.Join(hostnameArgs, " ")
		util.Warning(fmt.Sprintf("    sudo %s", command))
		output.UserOut.Println("Please enter your password if prompted.")
		_, err = exec.RunCommandPipe("sudo", hostnameArgs)
		if err != nil {
			util.Warning("Failed to execute sudo command, you will need to manually execute '%s' with administrative privileges", command)
		}
	}
	return nil
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
			return fmt.Errorf("error trying to create directory %s, err: %v", dir, err)
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
			return "", fmt.Errorf("Could not find a project in %s. Have you run 'ddev config'? Please specify a project name or change directories: %s", siteDir, err)
		}
	} else {
		var ok bool

		labels := map[string]string{
			"com.ddev.site-name":         siteName,
			"com.docker.compose.service": "web",
		}

		webContainer, err := dockerutil.FindContainerByLabels(labels)
		if err != nil {
			return "", fmt.Errorf("could not find a project named '%s'. Run 'ddev list' to see currently active projects", siteName)
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

	// Mostly ignore app.Init() error, since app.Init() fails if no directory found. Some errors should be handled though.
	// We already were successful with *finding* the app, and if we get an
	// incomplete one we have to add to it.
	err = app.Init(activeAppRoot)
	if err != nil && (strings.Contains(err.Error(), "is not a valid hostname") || strings.Contains(err.Error(), "is not a valid apptype") || strings.Contains(err.Error(), "config.yaml exists but cannot be read.") || strings.Contains(err.Error(), "a project (web container) in ")) {
		return app, err
	}

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
		return fmt.Errorf("error restoring AppConfig: no project name given")
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
