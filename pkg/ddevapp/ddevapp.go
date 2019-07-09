package ddevapp

import (
	"bytes"
	"fmt"
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/drud/ddev/pkg/nodeps"
	"github.com/lextoumbourou/goodhosts"
	"github.com/mattn/go-isatty"
	"golang.org/x/crypto/ssh/terminal"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"

	"strings"

	osexec "os/exec"

	"path"
	"time"

	"github.com/drud/ddev/pkg/appimport"
	"github.com/drud/ddev/pkg/archive"
	"github.com/drud/ddev/pkg/ddevhosts"
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/util"
	"github.com/drud/ddev/pkg/version"
	"github.com/fsouza/go-dockerclient"
)

// containerWaitTimeout is the max time we wait for all containers to become ready.
var containerWaitTimeout = 61

// bgsyncWaitTimeout is time time in seconds to wait for the bgsync container
// to become ready. This is normally just the normal container start time + docker cp
var bgsyncWaitTimeout = 900

// bgsyncSyncWaitTimeout is the max time in seconds we wait for bgsync to be syncing.
var bgsyncSyncWaitTimeout = 600

// SiteRunning defines the string used to denote running sites.
const SiteRunning = "running"

// SiteStarting
const SiteStarting = "starting"

// SiteStopped defines the string used to denote a site where the containers were not found/do not exist, but the project is there.
const SiteStopped = "stopped"

// SiteDirMissing defines the string used to denote when a site is missing its application directory.
const SiteDirMissing = "project directory missing"

// SiteConfigMissing defines the string used to denote when a site is missing its .ddev/config.yml file.
const SiteConfigMissing = ".ddev/config.yaml missing"

// SitePaused defines the string used to denote when a site is in the paused (docker stopped) state.
const SitePaused = "paused"

// DdevFileSignature is the text we use to detect whether a settings file is managed by us.
// If this string is found, we assume we can replace/update the file.
const DdevFileSignature = "#ddev-generated"

// DdevApp is the struct that represents a ddev app, mostly its config
// from config.yaml.
type DdevApp struct {
	APIVersion            string                `yaml:"APIVersion"`
	Name                  string                `yaml:"name"`
	Type                  string                `yaml:"type"`
	Docroot               string                `yaml:"docroot"`
	PHPVersion            string                `yaml:"php_version"`
	WebserverType         string                `yaml:"webserver_type"`
	WebImage              string                `yaml:"webimage,omitempty"`
	BgsyncImage           string                `yaml:"bgsyncimage,omitempty"`
	DBImage               string                `yaml:"dbimage,omitempty"`
	DBAImage              string                `yaml:"dbaimage,omitempty"`
	RouterHTTPPort        string                `yaml:"router_http_port"`
	RouterHTTPSPort       string                `yaml:"router_https_port"`
	XdebugEnabled         bool                  `yaml:"xdebug_enabled"`
	AdditionalHostnames   []string              `yaml:"additional_hostnames"`
	AdditionalFQDNs       []string              `yaml:"additional_fqdns"`
	MariaDBVersion        string                `yaml:"mariadb_version"`
	WebcacheEnabled       bool                  `yaml:"webcache_enabled,omitempty"`
	NFSMountEnabled       bool                  `yaml:"nfs_mount_enabled"`
	ConfigPath            string                `yaml:"-"`
	AppRoot               string                `yaml:"-"`
	Platform              string                `yaml:"-"`
	Provider              string                `yaml:"provider,omitempty"`
	DataDir               string                `yaml:"-"`
	SiteSettingsPath      string                `yaml:"-"`
	SiteDdevSettingsFile  string                `yaml:"-"`
	providerInstance      Provider              `yaml:"-"`
	Hooks                 map[string][]YAMLTask `yaml:"hooks,omitempty"`
	UploadDir             string                `yaml:"upload_dir,omitempty"`
	WorkingDir            map[string]string     `yaml:"working_dir,omitempty"`
	OmitContainers        []string              `yaml:"omit_containers,omitempty,flow"`
	HostDBPort            string                `yaml:"host_db_port,omitempty"`
	HostWebserverPort     string                `yaml:"host_webserver_port,omitempty"`
	HostHTTPSPort         string                `yaml:"host_https_port,omitempty"`
	MailhogPort           string                `yaml:"mailhog_port,omitempty"`
	PHPMyAdminPort        string                `yaml:"phpmyadmin_port,omitempty"`
	WebImageExtraPackages []string              `yaml:"webimage_extra_packages,omitempty,flow"`
	DBImageExtraPackages  []string              `yaml:"dbimage_extra_packages,omitempty,flow"`
	ProjectTLD            string                `yaml:"project_tld,omitempty"`
	UseDNSWhenPossible    bool                  `yaml:"use_dns_when_possible"`
	MkcertEnabled         bool                  `yaml:"-"`
	NgrokArgs             string                `yaml:"ngrok_args,omitempty"`
	Timezone              string                `yaml:"timezone"`
}

// GetType returns the application type as a (lowercase) string
func (app *DdevApp) GetType() string {
	return strings.ToLower(app.Type)
}

// Init populates DdevApp config based on the current working directory.
// It does not start the containers.
func (app *DdevApp) Init(basePath string) error {
	newApp, err := NewApp(basePath, true, "")
	if err != nil {
		return err
	}

	err = newApp.ValidateConfig()
	if err != nil {
		return err
	}

	*app = *newApp
	web, err := app.FindContainerByType("web")

	if err != nil {
		return err
	}

	if web != nil {
		containerApproot := web.Labels["com.ddev.approot"]
		isSameFile, err := fileutil.IsSameFile(containerApproot, app.AppRoot)
		if err != nil {
			return err
		}
		if !isSameFile {
			return fmt.Errorf("a project (web container) in %s state already exists for %s that was created at %s", web.State, app.Name, containerApproot).(webContainerExists)
		}
		return nil
	}
	// Init() is just putting together the DdevApp struct, the containers do
	// not have to exist (app doesn't have to have been started, so the fact
	// we didn't find any is not an error.
	return nil
}

// FindContainerByType will find a container for this site denoted by the containerType if it is available.
func (app *DdevApp) FindContainerByType(containerType string) (*docker.APIContainers, error) {
	labels := map[string]string{
		"com.ddev.site-name":         app.GetName(),
		"com.docker.compose.service": containerType,
	}

	return dockerutil.FindContainerByLabels(labels)
}

// Describe returns a map which provides detailed information on services associated with the running site.
func (app *DdevApp) Describe() (map[string]interface{}, error) {
	err := app.ProcessHooks("pre-describe")
	if err != nil {
		return nil, fmt.Errorf("Failed to process pre-describe hooks: %v", err)
	}

	shortRoot := RenderHomeRootedDir(app.GetAppRoot())
	appDesc := make(map[string]interface{})

	appDesc["name"] = app.GetName()
	appDesc["hostnames"] = app.GetHostnames()
	appDesc["status"] = app.SiteStatus()
	appDesc["sync_status"], _ = app.SyncStatus()
	appDesc["type"] = app.GetType()
	appDesc["approot"] = app.GetAppRoot()
	appDesc["shortroot"] = shortRoot
	appDesc["httpurl"] = app.GetHTTPURL()
	appDesc["httpsurl"] = app.GetHTTPSURL()
	appDesc["urls"] = app.GetAllURLs()

	// Only show extended status for running sites.
	if app.SiteStatus() == SiteRunning {
		dbinfo := make(map[string]interface{})
		dbinfo["username"] = "db"
		dbinfo["password"] = "db"
		dbinfo["dbname"] = "db"
		dbinfo["host"] = "db"
		dbPublicPort, err := app.GetPublishedPort("db")
		util.CheckErr(err)
		dbinfo["dbPort"] = GetPort("db")
		util.CheckErr(err)
		dbinfo["published_port"] = dbPublicPort
		dbinfo["mariadb_version"] = app.MariaDBVersion
		appDesc["dbinfo"] = dbinfo

		appDesc["mailhog_url"] = "http://" + app.GetHostname() + ":" + app.MailhogPort
		if !nodeps.ArrayContainsString(app.OmitContainers, "dba") {
			appDesc["phpmyadmin_url"] = "http://" + app.GetHostname() + ":" + app.PHPMyAdminPort
		}
	}

	routerStatus, logOutput := GetRouterStatus()
	appDesc["router_status"] = routerStatus
	appDesc["router_status_log"] = logOutput
	appDesc["ssh_agent_status"] = GetSSHAuthStatus()
	appDesc["php_version"] = app.GetPhpVersion()
	appDesc["webserver_type"] = app.GetWebserverType()

	appDesc["router_http_port"] = app.RouterHTTPPort
	appDesc["router_https_port"] = app.RouterHTTPSPort
	appDesc["xdebug_enabled"] = app.XdebugEnabled
	appDesc["webimg"] = app.WebImage
	appDesc["dbimg"] = app.WebImage
	appDesc["bgsyncimg"] = app.BgsyncImage
	appDesc["dbaimg"] = app.DBAImage

	err = app.ProcessHooks("post-describe")
	if err != nil {
		return nil, fmt.Errorf("Failed to process post-describe hooks: %v", err)
	}

	return appDesc, nil
}

// GetPublishedPort returns the host-exposed public port of a container.
func (app *DdevApp) GetPublishedPort(serviceName string) (int, error) {
	container, err := app.FindContainerByType(serviceName)
	if err != nil || container == nil {
		return -1, fmt.Errorf("Failed to find container of type %s: %v", serviceName, err)
	}

	privatePort, _ := strconv.ParseInt(GetPort(serviceName), 10, 16)

	publishedPort := dockerutil.GetPublishedPort(privatePort, *container)
	return publishedPort, nil
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
	v := PHPDefault
	if app.PHPVersion != "" {
		v = app.PHPVersion
	}
	return v
}

// GetWebserverType returns the app's webserver type (nginx-fpm/apache-fpm/apache-cgi)
func (app *DdevApp) GetWebserverType() string {
	v := WebserverDefault
	if app.WebserverType != "" {
		v = app.WebserverType
	}
	return v
}

// ImportDB takes a source sql dump and imports it to an active site's database container.
func (app *DdevApp) ImportDB(imPath string, extPath string, progress bool) error {
	app.DockerEnv()
	var extPathPrompt bool
	dbPath, err := ioutil.TempDir(filepath.Dir(app.ConfigPath), "importdb")
	//nolint: errcheck
	defer os.RemoveAll(dbPath)
	if err != nil {
		return err
	}

	err = app.ProcessHooks("pre-import-db")
	if err != nil {
		return err
	}

	// If they don't provide an import path and we're not on a tty (piped in stuff)
	// then prompt for path to db
	if imPath == "" && isatty.IsTerminal(os.Stdin.Fd()) {
		// ensure we prompt for extraction path if an archive is provided, while still allowing
		// non-interactive use of --src flag without providing a --extract-path flag.
		if extPath == "" {
			extPathPrompt = true
		}
		output.UserOut.Println("Provide the path to the database you wish to import.")
		fmt.Print("Pull path: ")

		imPath = util.GetInput("")
	}

	if imPath != "" {
		importPath, isArchive, err := appimport.ValidateAsset(imPath, "db")
		if err != nil {
			if isArchive && extPathPrompt {
				output.UserOut.Println("You provided an archive. Do you want to extract from a specific path in your archive? You may leave this blank if you wish to use the full archive contents")
				fmt.Print("Archive extraction path:")

				extPath = util.GetInput("")
			}

			if err != nil {
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
	}

	// Inside the container, the dir for imports will be at /mnt/ddev_config/<tmpdir_name>
	insideContainerImportPath := path.Join("/mnt/ddev_config", filepath.Base(dbPath))

	inContainerCommand := "pv " + insideContainerImportPath + "/*.*sql | mysql db"
	if imPath == "" && extPath == "" {
		inContainerCommand = "pv | mysql db"
	}

	_, _, err = app.Exec(&ExecOpts{
		Service: "db",
		Cmd:     inContainerCommand,
		Tty:     progress && isatty.IsTerminal(os.Stdin.Fd()),
	})

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

// ExportDB exports the db, with optional output to a file, default gzip
func (app *DdevApp) ExportDB(outFile string, gzip bool) error {
	app.DockerEnv()

	opts := &ExecOpts{
		Service:   "db",
		Cmd:       "mysqldump db",
		NoCapture: true,
	}
	if gzip {
		opts.Cmd = "mysqldump db | gzip"
	}
	if outFile != "" {
		f, err := os.OpenFile(outFile, os.O_RDWR|os.O_CREATE, 0644)
		if err != nil {
			return fmt.Errorf("failed to open %s: %v", outFile, err)
		}
		opts.Stdout = f
		// nolint: errcheck
		defer f.Close()
	}

	_, _, err := app.Exec(opts)

	if err != nil {
		return err
	}

	return nil
}

// SiteStatus returns the current status of an application determined from web and db service health.
func (app *DdevApp) SiteStatus() string {
	var siteStatus string
	statuses := map[string]string{"web": "", "db": ""}
	if app.WebcacheEnabled {
		statuses["bgsync"] = ""
	}

	if !fileutil.FileExists(app.GetAppRoot()) {
		siteStatus = fmt.Sprintf("%s: %v", SiteDirMissing, app.GetAppRoot())
		return siteStatus
	}

	_, err := CheckForConf(app.GetAppRoot())
	if err != nil {
		siteStatus = fmt.Sprintf("%s", SiteConfigMissing)
		return siteStatus
	}

	for service := range statuses {
		container, err := app.FindContainerByType(service)
		if err != nil {
			util.Error("app.FindContainerByType(%v) failed", service)
			return ""
		}
		if container == nil {
			statuses[service] = SiteStopped
		} else {
			status, _ := dockerutil.GetContainerHealth(container)

			switch status {
			case "exited":
				statuses[service] = SitePaused
			case "healthy":
				statuses[service] = SiteRunning
			case "starting":
				statuses[service] = SiteStarting
			default:
				statuses[service] = status
			}
		}
	}

	// Base the siteStatus on web container. Then override it if others are not the same.
	siteStatus = statuses["web"]
	for serviceName, status := range statuses {
		if status != siteStatus {
			siteStatus = siteStatus + "\n" + serviceName + ": " + status
		}
	}
	return siteStatus
}

func (app *DdevApp) SyncStatus() (string, error) {
	container, err := app.FindContainerByType("bgsync")
	if err != nil || container == nil {
		return "", err
	}
	_, syncStatus := dockerutil.GetContainerHealth(container)
	return syncStatus, nil
}

// PullOptions allows for customization of the pull process.
type PullOptions struct {
	SkipDb      bool
	SkipFiles   bool
	SkipImport  bool
	Environment string
}

// Pull performs an import from the a configured provider plugin, if one exists.
func (app *DdevApp) Pull(provider Provider, opts *PullOptions) error {
	var err error
	err = app.ProcessHooks("pre-pull")
	if err != nil {
		return fmt.Errorf("Failed to process pre-pull hooks: %v", err)
	}

	err = provider.Validate()
	if err != nil {
		return err
	}

	if app.SiteStatus() != SiteRunning {
		util.Warning("Project is not currently running. Starting project before performing pull.")
		err = app.Start()
		if err != nil {
			return err
		}
	}

	if opts.SkipDb {
		output.UserOut.Println("Skipping database pull.")
	} else {
		output.UserOut.Println("Downloading database...")
		fileLocation, importPath, err := provider.GetBackup("database", opts.Environment)
		if err != nil {
			return err
		}

		output.UserOut.Printf("Database downloaded to: %s", fileLocation)

		if opts.SkipImport {
			output.UserOut.Println("Skipping database import.")
		} else {
			output.UserOut.Println("Importing database...")
			err = app.ImportDB(fileLocation, importPath, true)
			if err != nil {
				return err
			}
		}
	}

	if opts.SkipFiles {
		output.UserOut.Println("Skipping files pull.")
	} else {
		output.UserOut.Println("Downloading file archive...")
		fileLocation, importPath, err := provider.GetBackup("files", opts.Environment)
		if err != nil {
			return err
		}

		output.UserOut.Printf("File archive downloaded to: %s", fileLocation)

		if opts.SkipImport {
			output.UserOut.Println("Skipping files import.")
		} else {
			output.UserOut.Println("Importing files...")
			err = app.ImportFiles(fileLocation, importPath)
			if err != nil {
				return err
			}
		}
	}
	err = app.ProcessHooks("post-pull")
	if err != nil {
		return fmt.Errorf("Failed to process post-pull hooks: %v", err)
	}

	return nil
}

// ImportFiles takes a source directory or archive and copies to the uploaded files directory of a given app.
func (app *DdevApp) ImportFiles(importPath string, extPath string) error {
	app.DockerEnv()

	if err := app.ProcessHooks("pre-import-files"); err != nil {
		return err
	}

	if err := app.ImportFilesAction(importPath, extPath); err != nil {
		return err
	}

	if err := app.ProcessHooks("post-import-files"); err != nil {
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

// ProcessHooks executes Tasks defined in Hooks
func (app *DdevApp) ProcessHooks(hookName string) error {
	if cmds := app.Hooks[hookName]; len(cmds) > 0 {
		output.UserOut.Printf("Executing %s hook...", hookName)
	}

	for _, c := range app.Hooks[hookName] {
		a := NewTask(app, c)
		if a == nil {
			return fmt.Errorf("unable to create task from %v", c)
		}

		output.UserOut.Printf("=== Running task: %s, output below", a.GetDescription())

		stdout, stderr, err := a.Execute()
		if err != nil {
			return fmt.Errorf("task failed: %v", err)
		}
		output.UserOut.Println(stdout + "\n" + stderr)
	}

	return nil
}

// Start initiates docker-compose up
func (app *DdevApp) Start() error {
	var err error

	app.DockerEnv()

	if app.APIVersion != version.DdevVersion {
		util.Warning("Your %s version is %s, but ddev is version %s. \nPlease run 'ddev config' to update your config.yaml. \nddev may not operate correctly until you do.", app.ConfigPath, app.APIVersion, version.DdevVersion)
	}

	// Make sure that any ports allocated are available.
	// and of course add to global project list as well
	err = app.UpdateGlobalProjectList()
	if err != nil {
		return err
	}

	err = app.ProcessHooks("pre-start")
	if err != nil {
		return err
	}

	if !nodeps.ArrayContainsString(app.OmitContainers, "ddev-ssh-agent") {
		err = app.EnsureSSHAgentContainer()
		if err != nil {
			return err
		}
	}

	// Warn the user if there is any custom configuration in use.
	app.CheckCustomConfig()

	caRoot := GetCAROOT()
	if caRoot == "" {
		util.Warning("mkcert may not be properly installed, please install it, `brew install mkcert nss`, `choco install -y mkcert`, etc. and then `mkcert -install`: %v", err)
	}
	router, _ := FindDdevRouter()
	// If the router doesn't exist, go ahead and push mkcert root ca certs into the ddev-global-cache/mkcert
	// This will often be redundant
	if router == nil {
		// Copy ca certs into ddev-global-cache/mkcert
		if caRoot != "" {
			output.UserOut.Info("Pushing mkcert rootca certs to ddev-global-cache")
			_, out, err := dockerutil.RunSimpleContainer("busybox:latest", "", []string{"sh", "-c", "mkdir -p /mnt/ddev-global-cache/composer && mkdir -p /mnt/ddev-global-cache/mkcert && chmod 777 /mnt/ddev-global-cache/* && cp -R /mnt/mkcert /mnt/ddev-global-cache"}, []string{}, []string{}, []string{"ddev-global-cache" + ":/mnt/ddev-global-cache", caRoot + ":/mnt/mkcert"}, "", true)
			if err != nil {
				util.Warning("failed to copy root CA into docker volume: %v, output='%s'", err, out)
			}
			util.Success("Pushed mkcert rootca certs to ddev-global-cache")
		}
	}

	// WriteConfig docker-compose.yaml
	err = app.WriteDockerComposeConfig()
	if err != nil {
		return err
	}

	err = app.AddHostsEntriesIfNeeded()
	if err != nil {
		return err
	}

	files, err := app.ComposeFiles()
	if err != nil {
		return err
	}

	// Delete the webcachevol and NFS volumes before we bring up docker-compose.
	// We don't care if the volume wasn't there
	_ = dockerutil.RemoveVolume(app.GetWebcacheVolName())
	_ = dockerutil.RemoveVolume(app.GetNFSMountVolName())

	// Pull the main images with full output, since docker-compose up won't
	// show enough output.
	for _, imageName := range []string{app.WebImage, app.DBImage} {
		err = dockerutil.Pull(imageName)
		if err != nil {
			return err
		}
	}
	_, _, err = dockerutil.ComposeCmd(files, "up", "--build", "-d")
	if err != nil {
		return err
	}

	err = StartDdevRouter()
	if err != nil {
		return err
	}

	requiredContainers := []string{"db", "web"}
	if app.WebcacheEnabled {
		requiredContainers = append(requiredContainers, "bgsync")
		err = app.precacheWebdir()
		if err != nil {
			return err
		}
	}

	err = app.Wait(requiredContainers)
	if err != nil {
		return err
	}

	err = app.PostStartAction()
	if err != nil {
		return err
	}

	err = app.ProcessHooks("post-start")
	if err != nil {
		return err
	}

	return nil
}

// ExecOpts contains options for running a command inside a container
type ExecOpts struct {
	// Service is the service, as in 'web', 'db', 'dba'
	Service string
	// Dir is the working directory inside the container
	Dir string
	// Cmd is the string to execute
	Cmd string
	// Nocapture if true causes use of ComposeNoCapture, so the stdout and stderr go right to stdout/stderr
	NoCapture bool
	// Tty if true causes a tty to be allocated
	Tty bool
	// Stdout can be overridden with a File
	Stdout *os.File
	// Stderr can be overridden with a File
	Stderr *os.File
}

// Exec executes a given command in the container of given type without allocating a pty
// Returns ComposeCmd results of stdout, stderr, err
// If Nocapture arg is true, stdout/stderr will be empty and output directly to stdout/stderr
func (app *DdevApp) Exec(opts *ExecOpts) (string, string, error) {
	app.DockerEnv()

	if opts.Service == "" {
		return "", "", fmt.Errorf("no service provided")
	}
	err := app.ProcessHooks("pre-exec")
	if err != nil {
		return "", "", fmt.Errorf("Failed to process pre-exec hooks: %v", err)
	}

	exec := []string{"exec"}
	if workingDir := app.getWorkingDir(opts.Service, opts.Dir); workingDir != "" {
		exec = append(exec, "-w", workingDir)
	}

	if !isatty.IsTerminal(os.Stdin.Fd()) || !opts.Tty {
		exec = append(exec, "-T")
	}

	exec = append(exec, opts.Service)

	if opts.Cmd == "" {
		return "", "", fmt.Errorf("no command provided")
	}

	// Cases to handle
	// - Free form, all unquoted. Like `ls -l -a`
	// - Quoted to delay pipes and other features to container, like `"ls -l -a | grep junk"`
	// Note that a set quoted on the host in ddev exec will come through as a single arg

	// Use bash for our containers, sh for 3rd-party containers
	// that may not have bash.
	shell := "bash"
	if !nodeps.ArrayContainsString([]string{"web", "db", "dba"}, opts.Service) {
		shell = "sh"
	}
	exec = append(exec, shell, "-c", opts.Cmd)

	files, err := app.ComposeFiles()
	if err != nil {
		return "", "", err
	}

	stdout := os.Stdout
	stderr := os.Stderr
	if opts.Stdout != nil {
		stdout = opts.Stdout
	}
	if opts.Stderr != nil {
		stderr = opts.Stderr
	}

	var stdoutResult, stderrResult string
	if opts.NoCapture || opts.Tty {
		err = dockerutil.ComposeWithStreams(files, os.Stdin, stdout, stderr, exec...)
	} else {
		stdoutResult, stderrResult, err = dockerutil.ComposeCmd(files, exec...)
	}

	hookErr := app.ProcessHooks("post-exec")
	if hookErr != nil {
		return stdoutResult, stderrResult, fmt.Errorf("Failed to process post-exec hooks: %v", hookErr)
	}

	return stdoutResult, stderrResult, err
}

// ExecWithTty executes a given command in the container of given type.
// It allocates a pty for interactive work.
func (app *DdevApp) ExecWithTty(opts *ExecOpts) error {
	app.DockerEnv()

	if opts.Service == "" {
		return fmt.Errorf("no service provided")
	}

	exec := []string{"exec"}
	if workingDir := app.getWorkingDir(opts.Service, opts.Dir); workingDir != "" {
		exec = append(exec, "-w", workingDir)
	}

	exec = append(exec, opts.Service)

	if opts.Cmd == "" {
		return fmt.Errorf("no command provided")
	}

	// Cases to handle
	// - Free form, all unquoted. Like `ls -l -a`
	// - Quoted to delay pipes and other features to container, like `"ls -l -a | grep junk"`
	// Note that a set quoted on the host in ddev exec will come through as a single arg

	// Use bash for our containers, sh for 3rd-party containers
	// that may not have bash.
	shell := "bash"
	if !nodeps.ArrayContainsString([]string{"web", "db", "dba"}, opts.Service) {
		shell = "sh"
	}
	exec = append(exec, shell, "-c", opts.Cmd)

	files, err := app.ComposeFiles()
	if err != nil {
		return err
	}

	return dockerutil.ComposeWithStreams(files, os.Stdin, os.Stdout, os.Stderr, exec...)
}

// Logs returns logs for a site's given container.
// See docker.LogsOptions for more information about valid tailLines values.
func (app *DdevApp) Logs(service string, follow bool, timestamps bool, tailLines string) error {
	client := dockerutil.GetDockerClient()

	var container *docker.APIContainers
	var err error
	// Let people access ddev-router and ddev-ssh-agent logs as well.
	if service == "ddev-router" || service == "ddev-ssh-agent" {
		container, err = dockerutil.FindContainerByLabels(map[string]string{"com.docker.compose.service": service})
	} else {
		container, err = app.FindContainerByType(service)
	}
	if err != nil {
		return err
	}
	if container == nil {
		util.Warning("No running service container %s was found", service)
		return nil
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

	err = client.Logs(logOpts)
	if err != nil {
		return err
	}

	return nil
}

// CaptureLogs returns logs for a site's given container.
// See docker.LogsOptions for more information about valid tailLines values.
func (app *DdevApp) CaptureLogs(service string, timestamps bool, tailLines string) (string, error) {
	client := dockerutil.GetDockerClient()

	var container *docker.APIContainers
	var err error
	// Let people access ddev-router and ddev-ssh-agent logs as well.
	if service == "ddev-router" || service == "ddev-ssh-agent" {
		container, err = dockerutil.FindContainerByLabels(map[string]string{"com.docker.compose.service": service})
	} else {
		container, err = app.FindContainerByType(service)
	}
	if err != nil {
		return "", err
	}
	if container == nil {
		util.Warning("No running service container %s was found", service)
		return "", nil
	}

	var out bytes.Buffer

	logOpts := docker.LogsOptions{
		Container:    container.ID,
		Stdout:       true,
		Stderr:       true,
		OutputStream: &out,
		ErrorStream:  &out,
		Follow:       false,
		Timestamps:   timestamps,
	}

	if tailLines != "" {
		logOpts.Tail = tailLines
	}

	err = client.Logs(logOpts)
	if err != nil {
		return "", err
	}

	return out.String(), nil
}

// DockerEnv sets environment variables for a docker-compose run.
func (app *DdevApp) DockerEnv() {

	uidStr, gidStr, _ := util.GetContainerUIDGid()

	// Warn about running as root if we're not on windows.
	if uidStr == "0" || gidStr == "0" {
		util.Warning("Warning: containers will run as root. This could be a security risk on Linux.")
	}

	envVars := map[string]string{
		"COMPOSE_PROJECT_NAME":          "ddev-" + app.Name,
		"COMPOSE_CONVERT_WINDOWS_PATHS": "true",
		"DDEV_SITENAME":                 app.Name,
		"DDEV_DBIMAGE":                  app.DBImage,
		"DDEV_DBAIMAGE":                 app.DBAImage,
		"DDEV_WEBIMAGE":                 app.WebImage,
		"DDEV_BGSYNCIMAGE":              app.BgsyncImage,
		"DDEV_APPROOT":                  app.AppRoot,
		"DDEV_HOST_DB_PORT":             app.HostDBPort,
		"DDEV_HOST_WEBSERVER_PORT":      app.HostWebserverPort,
		"DDEV_HOST_HTTPS_PORT":          app.HostHTTPSPort,
		"DDEV_PHPMYADMIN_PORT":          app.PHPMyAdminPort,
		"DDEV_MAILHOG_PORT":             app.MailhogPort,
		"DDEV_DOCROOT":                  app.Docroot,
		"DDEV_HOSTNAME":                 app.HostName(),
		"DDEV_UID":                      uidStr,
		"DDEV_GID":                      gidStr,
		"DDEV_PHP_VERSION":              app.PHPVersion,
		"DDEV_WEBSERVER_TYPE":           app.WebserverType,
		"DDEV_PROJECT_TYPE":             app.Type,
		"DDEV_ROUTER_HTTP_PORT":         app.RouterHTTPPort,
		"DDEV_ROUTER_HTTPS_PORT":        app.RouterHTTPSPort,
		"DDEV_XDEBUG_ENABLED":           strconv.FormatBool(app.XdebugEnabled),
	}

	// Set the mariadb_local command to empty to prevent docker-compose from complaining normally.
	// It's used for special startup on restoring to a snapshfot.
	if len(os.Getenv("DDEV_MARIADB_LOCAL_COMMAND")) == 0 {
		err := os.Setenv("DDEV_MARIADB_LOCAL_COMMAND", "")
		util.CheckErr(err)
	}

	// Find out terminal dimensions
	columns, lines, err := terminal.GetSize(0)
	if err != nil {
		columns = 80
		lines = 24
	}
	envVars["COLUMNS"] = strconv.Itoa(columns)
	envVars["LINES"] = strconv.Itoa(lines)

	if len(app.AdditionalHostnames) > 0 || len(app.AdditionalFQDNs) > 0 {
		envVars["DDEV_HOSTNAME"] = strings.Join(app.GetHostnames(), ",")
	}

	// Only set values if they don't already exist in env.
	for k, v := range envVars {
		if err := os.Setenv(k, v); err != nil {
			util.Error("Failed to set the environment variable %s=%s: %v", k, v, err)
		}
	}
}

// Pause initiates docker-compose stop
func (app *DdevApp) Pause() error {
	app.DockerEnv()

	if app.SiteStatus() == SiteStopped {
		return fmt.Errorf("no project to stop")
	}

	err := app.ProcessHooks("pre-pause")
	if err != nil {
		return err
	}

	files, err := app.ComposeFiles()
	if err != nil {
		return err
	}

	if _, _, err := dockerutil.ComposeCmd(files, "stop"); err != nil {
		return err
	}
	err = app.ProcessHooks("post-pause")
	if err != nil {
		return err
	}

	return StopRouterIfNoContainers()
}

// Wait ensures that the app service containers are healthy.
func (app *DdevApp) Wait(requiredContainers []string) error {
	for _, containerType := range requiredContainers {
		labels := map[string]string{
			"com.ddev.site-name":         app.GetName(),
			"com.docker.compose.service": containerType,
		}
		waitTime := containerWaitTimeout
		if containerType == BGSYNCContainer {
			waitTime = bgsyncWaitTimeout
		}
		logOutput, err := dockerutil.ContainerWait(waitTime, labels)
		if err != nil {
			return fmt.Errorf("%s container failed: log=%s, err=%v", containerType, logOutput, err)
		}
	}

	return nil
}

// WaitSync waits for bgsync container to be consistently syncing.
func (app *DdevApp) WaitSync() error {
	labels := map[string]string{
		"com.ddev.site-name":         app.GetName(),
		"com.docker.compose.service": BGSYNCContainer,
	}

	if !app.WebcacheEnabled {
		return nil
	}

	logOutput, err := dockerutil.ContainerWaitLog(bgsyncSyncWaitTimeout, labels, "sync active")
	if err != nil {
		return fmt.Errorf("bgsync container did not become ready: log=%s, err=%v", logOutput, err)
	}

	return nil
}

// StartAndWaitForSync() is primarily for use in tests.
// It does app.Start() but then waits for syncing to be in progress
// before returning.
// extraSleep arg in seconds is the time to wait if > 0
func (app *DdevApp) StartAndWaitForSync(extraSleep int) error {
	err := app.Start()
	if err != nil {
		return err
	}
	// Wait to make sure that sync is working, sleep a little for sync to have completed.
	if app.WebcacheEnabled {
		err = app.WaitSync()
		if err != nil {
			return err
		}
		if extraSleep > 0 {
			time.Sleep(time.Duration(extraSleep) * time.Second)
		}
	} else if nodeps.IsDockerToolbox() {
		// Docker Toolbox seems not to get the router properly
		// updated with certs as fast as we expect, use the extraSleep for that.
		if extraSleep > 0 {
			time.Sleep(time.Duration(extraSleep) * time.Second)
		}
	}
	return nil
}

// DetermineSettingsPathLocation figures out the path to the settings file for
// an app based on the contents/existence of app.SiteSettingsPath and
// app.SiteDdevSettingsFile.
func (app *DdevApp) DetermineSettingsPathLocation() (string, error) {
	possibleLocations := []string{app.SiteSettingsPath, app.SiteDdevSettingsFile}
	for _, loc := range possibleLocations {
		// If the file doesn't exist, it's safe to use
		if !fileutil.FileExists(loc) {
			return loc, nil
		}

		// If the file does exist, check for a signature indicating it's managed by ddev.
		signatureFound, err := fileutil.FgrepStringInFile(loc, DdevFileSignature)
		util.CheckErr(err) // Really can't happen as we already checked for the file existence

		// If the signature was found, it's safe to use.
		if signatureFound {
			return loc, nil
		}
	}

	return "", fmt.Errorf("settings files already exist and are being managed by the user")
}

// Snapshot forces a mariadb snapshot of the db to be written into .ddev/db_snapshots
// Returns the dirname of the snapshot and err
func (app *DdevApp) Snapshot(snapshotName string) (string, error) {
	err := app.ProcessHooks("pre-snapshot")
	if err != nil {
		return "", fmt.Errorf("Failed to process pre-stop hooks: %v", err)
	}

	if snapshotName == "" {
		t := time.Now()
		snapshotName = app.Name + "_" + t.Format("20060102150405")
	}
	// Container side has to use path.Join instead of filepath.Join because they are
	// targeted at the linux filesystem, so won't work with filepath on Windows
	snapshotDir := path.Join("db_snapshots", snapshotName)
	hostSnapshotDir := filepath.Join(filepath.Dir(app.ConfigPath), snapshotDir)
	containerSnapshotDir := path.Join("/mnt/ddev_config", snapshotDir)
	err = os.MkdirAll(hostSnapshotDir, 0777)
	if err != nil {
		return snapshotName, err
	}

	// Ensure that db container is up.
	labels := map[string]string{"com.ddev.site-name": app.Name, "com.docker.compose.service": "db"}
	_, err = dockerutil.ContainerWait(containerWaitTimeout, labels)
	if err != nil {
		return "", fmt.Errorf("unable to snapshot database, \nyour project %v is not running. \nPlease start the project if you want to snapshot it. \nIf removing, you can remove without a snapshot using \n'ddev stop --remove-data --omit-snapshot', \nwhich will destroy your database", app.Name)
	}

	util.Warning("Creating database snapshot %s", snapshotName)
	stdout, stderr, err := app.Exec(&ExecOpts{
		Service: "db",
		Cmd:     fmt.Sprintf("mariabackup --backup --target-dir=%s --user root --password root --socket=/var/tmp/mysql.sock 2>/var/log/mariadbackup_backup_%s.log && cp /var/lib/mysql/db_mariadb_version.txt %s", containerSnapshotDir, snapshotName, containerSnapshotDir),
	})

	if err != nil {
		util.Warning("Failed to create snapshot: %v, stdout=%s, stderr=%s", err, stdout, stderr)
		return "", err
	}

	util.Success("Created database snapshot %s in %s", snapshotName, hostSnapshotDir)
	err = app.ProcessHooks("post-snapshot")
	if err != nil {
		return snapshotName, fmt.Errorf("Failed to process pre-stop hooks: %v", err)
	}
	return snapshotName, nil
}

// RestoreSnapshot restores a mariadb snapshot of the db to be loaded
// The project must be stopped and docker volume removed and recreated for this to work.
func (app *DdevApp) RestoreSnapshot(snapshotName string) error {
	var err error
	err = app.ProcessHooks("pre-restore-snapshot")
	if err != nil {
		return fmt.Errorf("Failed to process pre-restore-snapshot hooks: %v", err)
	}

	snapshotDir := filepath.Join("db_snapshots", snapshotName)

	hostSnapshotDir := filepath.Join(app.AppConfDir(), snapshotDir)
	if !fileutil.FileExists(hostSnapshotDir) {
		return fmt.Errorf("Failed to find a snapshot in %s", hostSnapshotDir)
	}

	// Find out the mariadb version that correlates to the snapshot.
	versionFile := filepath.Join(hostSnapshotDir, "db_mariadb_version.txt")
	var snapshotMariaDBVersion string
	if fileutil.FileExists(versionFile) {
		snapshotMariaDBVersion, err = fileutil.ReadFileIntoString(versionFile)
		if err != nil {
			return fmt.Errorf("unable to read the version file in the snapshot (%s): %v", versionFile, err)
		}
	} else {
		snapshotMariaDBVersion = MariaDB101
	}
	snapshotMariaDBVersion = strings.Trim(snapshotMariaDBVersion, " \n\t")

	if snapshotMariaDBVersion == MariaDB101 && app.MariaDBVersion != MariaDB101 {
		//nolint: golint
		return fmt.Errorf("snapshot %s is a MariaDB 10.1 snapshot\nIt is not compatible with the configured ddev MariaDB version (%s).\nPlease use the instructions at %s to change the MariaDB version so you can restore it.", snapshotDir, app.MariaDBVersion, "https://ddev.readthedocs.io/en/stable/users/troubleshooting/#old-snapshot")
	}
	if snapshotMariaDBVersion != MariaDB101 && app.MariaDBVersion == MariaDB101 {
		//nolint: golint
		return fmt.Errorf("snapshot %s is a MariaDB %s snapshot\nIt is not compatible with the configured ddev MariaDB version (%s).", snapshotDir, snapshotMariaDBVersion, app.MariaDBVersion)
	}

	if app.SiteStatus() == SiteRunning || app.SiteStatus() == SitePaused {
		err := app.Stop(false, false)
		if err != nil {
			return fmt.Errorf("Failed to rm  project for RestoreSnapshot: %v", err)
		}
	}

	err = os.Setenv("DDEV_MARIADB_LOCAL_COMMAND", "restore_snapshot "+snapshotName)
	util.CheckErr(err)
	err = app.Start()
	if err != nil {
		return fmt.Errorf("Failed to start project for RestoreSnapshot: %v", err)
	}
	err = os.Unsetenv("DDEV_MARIADB_LOCAL_COMMAND")
	util.CheckErr(err)

	util.Success("Restored database snapshot: %s", hostSnapshotDir)
	err = app.ProcessHooks("post-restore-snapshot")
	if err != nil {
		return fmt.Errorf("Failed to process post-restore-snapshot hooks: %v", err)
	}
	return nil
}

// Stops and Removes the docker containers for the project in current directory.
func (app *DdevApp) Stop(removeData bool, createSnapshot bool) error {
	app.DockerEnv()
	var err error

	err = app.ProcessHooks("pre-stop")
	if err != nil {
		return fmt.Errorf("Failed to process pre-stop hooks: %v", err)
	}

	if createSnapshot == true {
		t := time.Now()
		_, err = app.Snapshot(app.Name + "_remove_data_snapshot_" + t.Format("20060102150405"))
		if err != nil {
			return err
		}
	}

	err = app.Pause()
	if err != nil {
		util.Warning("Failed to stop containers for %s: %v", app.GetName(), err)
	}
	// Remove all the containers and volumes for app.
	err = Cleanup(app)
	if err != nil {
		return err
	}

	// Remove data/database/projectInfo/hostname if we need to.
	if removeData {
		if err = app.RemoveHostsEntries(); err != nil {
			return fmt.Errorf("failed to remove hosts entries: %v", err)
		}
		app.RemoveGlobalProjectInfo()
		err = globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig)
		if err != nil {
			util.Warning("could not WriteGlobalConfig: %v", err)
		}

		for _, volName := range []string{app.Name + "-mariadb", app.GetUnisonCatalogVolName(), app.GetWebcacheVolName()} {
			err = dockerutil.RemoveVolume(volName)
			if err != nil {
				util.Warning("could not remove volume %s: %v", volName, err)
			}
		}
		util.Success("Project data/database removed from docker volume for project %s", app.Name)
	}

	err = app.ProcessHooks("post-stop")
	if err != nil {
		return fmt.Errorf("Failed to process post-stop hooks: %v", err)
	}

	err = StopRouterIfNoContainers()

	return err
}

// RemoveGlobalProjectInfo() deletes the project from ProjectList
func (app *DdevApp) RemoveGlobalProjectInfo() {
	_ = globalconfig.RemoveProjectInfo(app.Name)
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

		var url = "https://" + name + httpsPort
		if GetCAROOT() == "" {
			url = "http://" + name + httpPort
		}
		URLs = append(URLs, url)
	}

	if GetCAROOT() != "" {
		URLs = append(URLs, app.GetWebContainerDirectHTTPSURL())
	} else {
		URLs = append(URLs, app.GetWebContainerDirectHTTPURL())
	}

	return URLs
}

// GetWebContainerDirectHTTPURL returns the URL that can be used without the router to get to web container.
func (app *DdevApp) GetWebContainerDirectHTTPURL() string {
	// Get direct address of web container
	dockerIP, err := dockerutil.GetDockerIP()
	if err != nil {
		util.Warning("Unable to get Docker IP: %v", err)
	}
	port, _ := app.GetWebContainerPublicPort()
	return fmt.Sprintf("http://%s:%d", dockerIP, port)
}

// GetWebContainerDirectHTTPSURL returns the URL that can be used without the router to get to web container via https.
func (app *DdevApp) GetWebContainerDirectHTTPSURL() string {
	// Get direct address of web container
	dockerIP, err := dockerutil.GetDockerIP()
	if err != nil {
		util.Warning("Unable to get Docker IP: %v", err)
	}
	port, _ := app.GetWebContainerHTTPSPublicPort()
	return fmt.Sprintf("https://%s:%d", dockerIP, port)
}

// GetWebContainerPublicPort returns the direct-access public tcp port for http
func (app *DdevApp) GetWebContainerPublicPort() (int, error) {

	webContainer, err := app.FindContainerByType("web")
	if err != nil || webContainer == nil {
		return -1, fmt.Errorf("Unable to find web container for app: %s, err %v", app.Name, err)
	}

	for _, p := range webContainer.Ports {
		if p.PrivatePort == 80 {
			return int(p.PublicPort), nil
		}
	}
	return -1, fmt.Errorf("No public port found for private port 80")
}

// GetWebContainerHTTPSPublicPort returns the direct-access public tcp port for https
func (app *DdevApp) GetWebContainerHTTPSPublicPort() (int, error) {

	webContainer, err := app.FindContainerByType("web")
	if err != nil || webContainer == nil {
		return -1, fmt.Errorf("Unable to find https web container for app: %s, err %v", app.Name, err)
	}

	for _, p := range webContainer.Ports {
		if p.PrivatePort == 443 {
			return int(p.PublicPort), nil
		}
	}
	return -1, fmt.Errorf("No public https port found for private port 443")
}

// HostName returns the hostname of a given application.
func (app *DdevApp) HostName() string {
	return app.GetHostname()
}

// AddHostsEntriesIfNeeded will (optionally) add the site URL to the host's /etc/hosts.
func (app *DdevApp) AddHostsEntriesIfNeeded() error {
	dockerIP, err := dockerutil.GetDockerIP()
	if err != nil {
		return fmt.Errorf("could not get Docker IP: %v", err)
	}

	hosts, err := ddevhosts.New()
	if err != nil {
		util.Failed("could not open hostfile: %v", err)
	}

	ipPosition := hosts.GetIPPosition(dockerIP)
	if ipPosition != -1 && runtime.GOOS == "windows" {
		hostsLine := hosts.Lines[ipPosition]
		if len(hostsLine.Hosts) >= 10 {
			util.Error("You have more than 9 entries in your (windows) hostsfile entry for %s", dockerIP)
			util.Error("Please use `ddev hostname --remove-inactive` or edit the hosts file manually")
			util.Error("Please see %s for more information", "https://ddev.readthedocs.io/en/stable/users/troubleshooting/#windows-hosts-file-limited")
		}
	}

	for _, name := range app.GetHostnames() {
		if app.UseDNSWhenPossible {
			hostIPs, err := net.LookupHost(name)
			// If we had successful lookup and dockerIP matches
			// (which won't happen on Docker Toolbox) then don't bother
			// with adding to hosts file.
			if err == nil && len(hostIPs) > 0 && hostIPs[0] == dockerIP {
				continue
			}
		}

		// We likely won't hit the hosts.Has() as true because
		// we already did a lookup. But check anyway.
		if hosts.Has(dockerIP, name) {
			continue
		}
		err = addHostEntry(name, dockerIP)
		if err != nil {
			return err
		}
	}

	return nil
}

func addHostEntry(name string, ip string) error {
	_, err := osexec.LookPath("sudo")
	if (os.Getenv("DRUD_NONINTERACTIVE") != "") || err != nil {
		util.Warning("You must manually add the following entry to your hosts file:\n%s %s\nOr with root/administrative privileges execute 'ddev hostname %s %s'", ip, name, name, ip)
		return nil
	}

	ddevFullpath, err := os.Executable()
	util.CheckErr(err)

	output.UserOut.Printf("ddev needs to add an entry to your hostfile.\nIt will require administrative privileges via the sudo command, so you may be required\nto enter your password for sudo. ddev is about to issue the command:")

	hostnameArgs := []string{ddevFullpath, "hostname", name, ip}
	command := strings.Join(hostnameArgs, " ")
	util.Warning(fmt.Sprintf("    sudo %s", command))
	output.UserOut.Println("Please enter your password if prompted.")
	_, err = exec.RunCommandPipe("sudo", hostnameArgs)
	if err != nil {
		util.Warning("Failed to execute sudo command, you will need to manually execute '%s' with administrative privileges", command)
	}
	return nil
}

// RemoveHostsEntries will remote the site URL from the host's /etc/hosts.
func (app *DdevApp) RemoveHostsEntries() error {
	dockerIP, err := dockerutil.GetDockerIP()
	if err != nil {
		return fmt.Errorf("could not get Docker IP: %v", err)
	}

	hosts, err := goodhosts.NewHosts()
	if err != nil {
		util.Failed("could not open hostfile: %v", err)
	}

	for _, name := range app.GetHostnames() {
		if !hosts.Has(dockerIP, name) {
			continue
		}

		_, err = osexec.LookPath("sudo")
		if os.Getenv("DRUD_NONINTERACTIVE") != "" || err != nil {
			util.Warning("You must manually remove the following entry from your hosts file:\n%s %s\nOr with root/administrative privileges execute 'ddev hostname --remove %s %s", dockerIP, name, name, dockerIP)
			return nil
		}

		ddevFullPath, err := os.Executable()
		util.CheckErr(err)

		output.UserOut.Printf("ddev needs to remove an entry from your hosts file.\nIt will require administrative privileges via the sudo command, so you may be required\nto enter your password for sudo. ddev is about to issue the command:")

		hostnameArgs := []string{ddevFullPath, "hostname", "--remove", name, dockerIP}
		command := strings.Join(hostnameArgs, " ")
		util.Warning(fmt.Sprintf("    sudo %s", command))
		output.UserOut.Println("Please enter your password if prompted.")

		if _, err = exec.RunCommandPipe("sudo", hostnameArgs); err != nil {
			util.Warning("Failed to execute sudo command, you will need to manually execute '%s' with administrative privileges", command)
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
			return "", err
		}
		if webContainer == nil {
			return "", fmt.Errorf("could not find a project named '%s'. Run 'ddev list' to see currently active projects", siteName)
		}

		siteDir, ok = webContainer.Labels["com.ddev.approot"]
		if !ok {
			return "", fmt.Errorf("could not determine the location of %s from container: %s", siteName, dockerutil.ContainerName(*webContainer))
		}
	}
	appRoot, err := CheckForConf(siteDir)
	if err != nil {
		return siteDir, err
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
	if err = app.Init(activeAppRoot); err != nil {
		switch err.(type) {
		case webContainerExists, invalidConfigFile, invalidHostname, invalidAppType, invalidPHPVersion, invalidWebserverType, invalidProvider:
			return app, err
		}
	}

	if app.Name == "" {
		err = restoreApp(app, siteName)
		if err != nil {
			return app, err
		}
	}

	return app, nil
}

// restoreApp recreates an AppConfig's Name and returns an error
// if it cannot restore them.
func restoreApp(app *DdevApp, siteName string) error {
	if siteName == "" {
		return fmt.Errorf("error restoring AppConfig: no project name given")
	}
	app.Name = siteName
	return nil
}

// GetProvider returns a pointer to the provider instance interface.
func (app *DdevApp) GetProvider() (Provider, error) {
	if app.providerInstance != nil {
		return app.providerInstance, nil
	}

	var provider Provider
	err := fmt.Errorf("unknown provider type: %s, must be one of %v", app.Provider, GetValidProviders())

	switch app.Provider {
	case ProviderPantheon:
		provider = &PantheonProvider{}
		err = provider.Init(app)
	case ProviderDrudS3:
		provider = &DrudS3Provider{}
		err = provider.Init(app)
	case ProviderDefault:
		provider = &DefaultProvider{}
		err = nil
	default:
		provider = &DefaultProvider{}
		// Use the default error from above.
	}
	app.providerInstance = provider
	return app.providerInstance, err
}

// getWorkingDir will determine the appropriate working directory for an Exec/ExecWithTty command
// by consulting with the project configuration. If no dir is specified for the service, an
// empty string will be returned.
func (app *DdevApp) getWorkingDir(service, dir string) string {
	// Highest preference is for directories passed into the command directly
	if dir != "" {
		return dir
	}

	// The next highest preference is for directories defined in config.yaml
	if app.WorkingDir != nil {
		if workingDir := app.WorkingDir[service]; workingDir != "" {
			return workingDir
		}
	}

	// The next highest preference is for app type defaults
	return app.DefaultWorkingDirMap()[service]
}

// precacheWebdir() runs a container which just exits, but has the webcachedir mounted
// so we can "docker cp" to it.
func (app *DdevApp) precacheWebdir() error {
	containerName := "ddev-" + app.Name + "-bgsync"
	dockerArgs := []string{"cp", app.AppRoot + "/.", containerName + ":/fastdockermount"}

	start := time.Now()
	out, err := exec.RunCommand("docker", dockerArgs)
	if err != nil {
		return fmt.Errorf("docker %v failed: %v out=%s", dockerArgs, err, out)
	}
	util.Success("repo files push to container completed in %v", time.Now().Sub(start))

	// Set the flag to tell unison it can start syncing
	_, _, err = app.Exec(&ExecOpts{
		Service: BGSYNCContainer,
		Cmd:     "touch /var/tmp/unison_start_authorized",
	})
	if err != nil {
		return fmt.Errorf("could not start bgsync container syncing: %v", err)
	}

	return nil

}

// Returns the docker volume name of the webcachevol
func (app *DdevApp) GetWebcacheVolName() string {
	return strings.ToLower("ddev-" + app.Name + "_webcachevol")
}

// Returns the docker volume name of the unisoncatalogvolume
func (app *DdevApp) GetUnisonCatalogVolName() string {
	return strings.ToLower("ddev-" + app.Name + "_unisoncatalogvol")
}

// Returns the docker volume name of the nfs mount volume
func (app *DdevApp) GetNFSMountVolName() string {
	return strings.ToLower("ddev-" + app.Name + "_nfsmount")
}
