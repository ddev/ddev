package ddevapp

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/ddev/ddev/pkg/appimport"
	"github.com/ddev/ddev/pkg/archive"
	"github.com/ddev/ddev/pkg/config/types"
	dockerImages "github.com/ddev/ddev/pkg/docker"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/ddev/ddev/pkg/versionconstants"
	docker "github.com/fsouza/go-dockerclient"
	"github.com/mattn/go-isatty"
	"github.com/otiai10/copy"
	"golang.org/x/term"
	"gopkg.in/yaml.v3"
)

const (
	// SiteRunning defines the string used to denote running sites.
	SiteRunning  = "running"
	SiteStarting = "starting"

	// SiteStopped means a site where the containers were not found/do not exist, but the project is there.
	SiteStopped = "stopped"

	// SiteDirMissing defines the string used to denote when a site is missing its application directory.
	SiteDirMissing = "project directory missing"

	// SiteConfigMissing defines the string used to denote when a site is missing its .ddev/config.yml file.
	SiteConfigMissing = ".ddev/config.yaml missing"

	// SitePaused defines the string used to denote when a site is in the paused (docker stopped) state.
	SitePaused = "paused"

	// SiteUnhealthy is the status for a project whose services are not all reporting healthy yet
	SiteUnhealthy = "unhealthy"
)

// DatabaseDefault is the default database/version
var DatabaseDefault = DatabaseDesc{nodeps.MariaDB, nodeps.MariaDBDefaultVersion}

type DatabaseDesc struct {
	Type    string `yaml:"type"`
	Version string `yaml:"version"`
}

type WebExposedPort struct {
	Name             string `yaml:"name"`
	WebContainerPort int    `yaml:"container_port"`
	HTTPPort         int    `yaml:"http_port"`
	HTTPSPort        int    `yaml:"https_port"`
}

type WebExtraDaemon struct {
	Name      string `yaml:"name"`
	Command   string `yaml:"command"`
	Directory string `yaml:"directory"`
}

// DdevApp is the struct that represents a ddev app, mostly its config
// from config.yaml.
type DdevApp struct {
	Name                      string                 `yaml:"name"`
	Type                      string                 `yaml:"type"`
	Docroot                   string                 `yaml:"docroot"`
	PHPVersion                string                 `yaml:"php_version"`
	WebserverType             string                 `yaml:"webserver_type"`
	WebImage                  string                 `yaml:"webimage,omitempty"`
	RouterHTTPPort            string                 `yaml:"router_http_port,omitempty"`
	RouterHTTPSPort           string                 `yaml:"router_https_port,omitempty"`
	XdebugEnabled             bool                   `yaml:"xdebug_enabled"`
	NoProjectMount            bool                   `yaml:"no_project_mount,omitempty"`
	AdditionalHostnames       []string               `yaml:"additional_hostnames"`
	AdditionalFQDNs           []string               `yaml:"additional_fqdns"`
	MariaDBVersion            string                 `yaml:"mariadb_version,omitempty"`
	MySQLVersion              string                 `yaml:"mysql_version,omitempty"`
	Database                  DatabaseDesc           `yaml:"database"`
	PerformanceMode           types.PerformanceMode  `yaml:"performance_mode,omitempty"`
	FailOnHookFail            bool                   `yaml:"fail_on_hook_fail,omitempty"`
	BindAllInterfaces         bool                   `yaml:"bind_all_interfaces,omitempty"`
	FailOnHookFailGlobal      bool                   `yaml:"-"`
	ConfigPath                string                 `yaml:"-"`
	AppRoot                   string                 `yaml:"-"`
	DataDir                   string                 `yaml:"-"`
	SiteSettingsPath          string                 `yaml:"-"`
	SiteDdevSettingsFile      string                 `yaml:"-"`
	ProviderInstance          *Provider              `yaml:"-"`
	Hooks                     map[string][]YAMLTask  `yaml:"hooks,omitempty"`
	UploadDirDeprecated       string                 `yaml:"upload_dir,omitempty"`
	UploadDirs                []string               `yaml:"upload_dirs,omitempty"`
	WorkingDir                map[string]string      `yaml:"working_dir,omitempty"`
	OmitContainers            []string               `yaml:"omit_containers,omitempty,flow"`
	OmitContainersGlobal      []string               `yaml:"-"`
	HostDBPort                string                 `yaml:"host_db_port,omitempty"`
	HostWebserverPort         string                 `yaml:"host_webserver_port,omitempty"`
	HostHTTPSPort             string                 `yaml:"host_https_port,omitempty"`
	MailhogPort               string                 `yaml:"mailhog_port,omitempty"`
	MailhogHTTPSPort          string                 `yaml:"mailhog_https_port,omitempty"`
	HostMailhogPort           string                 `yaml:"host_mailhog_port,omitempty"`
	WebImageExtraPackages     []string               `yaml:"webimage_extra_packages,omitempty,flow"`
	DBImageExtraPackages      []string               `yaml:"dbimage_extra_packages,omitempty,flow"`
	ProjectTLD                string                 `yaml:"project_tld,omitempty"`
	UseDNSWhenPossible        bool                   `yaml:"use_dns_when_possible"`
	MkcertEnabled             bool                   `yaml:"-"`
	NgrokArgs                 string                 `yaml:"ngrok_args,omitempty"`
	Timezone                  string                 `yaml:"timezone,omitempty"`
	ComposerRoot              string                 `yaml:"composer_root,omitempty"`
	ComposerVersion           string                 `yaml:"composer_version"`
	DisableSettingsManagement bool                   `yaml:"disable_settings_management,omitempty"`
	WebEnvironment            []string               `yaml:"web_environment"`
	NodeJSVersion             string                 `yaml:"nodejs_version"`
	DefaultContainerTimeout   string                 `yaml:"default_container_timeout,omitempty"`
	WebExtraExposedPorts      []WebExposedPort       `yaml:"web_extra_exposed_ports,omitempty"`
	WebExtraDaemons           []WebExtraDaemon       `yaml:"web_extra_daemons,omitempty"`
	OverrideConfig            bool                   `yaml:"override_config,omitempty"`
	DisableUploadDirsWarning  bool                   `yaml:"disable_upload_dirs_warning,omitempty"`
	ComposeYaml               map[string]interface{} `yaml:"-"`
}

// GetType returns the application type as a (lowercase) string
func (app *DdevApp) GetType() string {
	return strings.ToLower(app.Type)
}

// Init populates DdevApp config based on the current working directory.
// It does not start the containers.
func (app *DdevApp) Init(basePath string) error {
	defer util.TimeTrackC(fmt.Sprintf("app.Init(%s)", basePath))()

	newApp, err := NewApp(basePath, true)
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
	// not have to exist (app doesn't have to have been started), so the fact
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
func (app *DdevApp) Describe(short bool) (map[string]interface{}, error) {
	app.DockerEnv()
	err := app.ProcessHooks("pre-describe")
	if err != nil {
		return nil, fmt.Errorf("failed to process pre-describe hooks: %v", err)
	}

	shortRoot := RenderHomeRootedDir(app.GetAppRoot())
	appDesc := make(map[string]interface{})
	status, statusDesc := app.SiteStatus()

	appDesc["name"] = app.GetName()
	appDesc["status"] = status
	appDesc["status_desc"] = statusDesc
	appDesc["approot"] = app.GetAppRoot()
	appDesc["docroot"] = app.GetDocroot()
	appDesc["shortroot"] = shortRoot
	appDesc["httpurl"] = app.GetHTTPURL()
	appDesc["httpsurl"] = app.GetHTTPSURL()
	appDesc["router_disabled"] = IsRouterDisabled(app)
	appDesc["primary_url"] = app.GetPrimaryURL()
	appDesc["type"] = app.GetType()
	appDesc["mutagen_enabled"] = app.IsMutagenEnabled()
	appDesc["nodejs_version"] = app.NodeJSVersion
	appDesc["router"] = globalconfig.DdevGlobalConfig.Router
	if app.IsMutagenEnabled() {
		appDesc["mutagen_status"], _, _, err = app.MutagenStatus()
		if err != nil {
			appDesc["mutagen_status"] = err.Error() + " " + appDesc["mutagen_status"].(string)
		}
	}

	// if short is set, we don't need more information, so return what we have.
	if short {
		return appDesc, nil
	}
	appDesc["hostname"] = app.GetHostname()
	appDesc["hostnames"] = app.GetHostnames()
	appDesc["nfs_mount_enabled"] = app.IsNFSMountEnabled()
	appDesc["fail_on_hook_fail"] = app.FailOnHookFail || app.FailOnHookFailGlobal
	httpURLs, httpsURLs, allURLs := app.GetAllURLs()
	appDesc["httpURLs"] = httpURLs
	appDesc["httpsURLs"] = httpsURLs
	appDesc["urls"] = allURLs

	appDesc["database_type"] = app.Database.Type
	appDesc["database_version"] = app.Database.Version

	// Only show extended status for running sites.
	if status == SiteRunning {
		if !nodeps.ArrayContainsString(app.GetOmittedContainers(), "db") {
			dbinfo := make(map[string]interface{})
			dbinfo["username"] = "db"
			dbinfo["password"] = "db"
			dbinfo["dbname"] = "db"
			dbinfo["host"] = "db"
			dbPublicPort, err := app.GetPublishedPort("db")
			util.CheckErr(err)
			dbinfo["dbPort"] = GetExposedPort(app, "db")
			util.CheckErr(err)
			dbinfo["published_port"] = dbPublicPort
			dbinfo["database_type"] = nodeps.MariaDB // default
			dbinfo["database_type"] = app.Database.Type
			dbinfo["database_version"] = app.Database.Version

			appDesc["dbinfo"] = dbinfo
		}

		appDesc["mailhog_https_url"] = "https://" + app.GetHostname() + ":" + app.MailhogHTTPSPort
		appDesc["mailhog_url"] = "http://" + app.GetHostname() + ":" + app.MailhogPort
	}

	routerStatus, logOutput := GetRouterStatus()
	appDesc["router_status"] = routerStatus
	appDesc["router_status_log"] = logOutput
	appDesc["ssh_agent_status"] = GetSSHAuthStatus()
	appDesc["php_version"] = app.GetPhpVersion()
	appDesc["webserver_type"] = app.GetWebserverType()

	appDesc["router_http_port"] = app.GetRouterHTTPPort()
	appDesc["router_https_port"] = app.GetRouterHTTPSPort()
	appDesc["xdebug_enabled"] = app.XdebugEnabled
	appDesc["webimg"] = app.WebImage
	appDesc["dbimg"] = app.GetDBImage()
	appDesc["services"] = map[string]map[string]string{}

	containers, err := dockerutil.GetAppContainers(app.Name)
	if err != nil {
		return nil, err
	}
	services := appDesc["services"].(map[string]map[string]string)
	for _, k := range containers {
		serviceName := strings.TrimPrefix(k.Names[0], "/")
		shortName := strings.Replace(serviceName, fmt.Sprintf("ddev-%s-", app.Name), "", 1)

		c, err := dockerutil.InspectContainer(serviceName)
		if err != nil || c == nil {
			util.Warning("Could not get container info for %s", serviceName)
			continue
		}
		fullName := strings.TrimPrefix(serviceName, "/")
		services[shortName] = map[string]string{}
		services[shortName]["status"] = c.State.Status
		services[shortName]["full_name"] = fullName
		services[shortName]["image"] = strings.TrimSuffix(c.Config.Image, fmt.Sprintf("-%s-built", app.Name))
		services[shortName]["short_name"] = shortName
		var ports []string
		for pk := range c.Config.ExposedPorts {
			ports = append(ports, pk.Port())
		}
		services[shortName]["exposed_ports"] = strings.Join(ports, ",")
		var hostPorts []string
		for _, pv := range k.Ports {
			if pv.PublicPort != 0 {
				hostPorts = append(hostPorts, strconv.FormatInt(pv.PublicPort, 10))
			}
		}
		services[shortName]["host_ports"] = strings.Join(hostPorts, ",")

		// Extract HTTP_EXPOSE and HTTPS_EXPOSE for additional info
		if !IsRouterDisabled(app) {
			for _, e := range c.Config.Env {
				split := strings.SplitN(e, "=", 2)
				envName := split[0]
				if len(split) == 2 && (envName == "HTTP_EXPOSE" || envName == "HTTPS_EXPOSE") {
					envVal := split[1]

					envValStr := fmt.Sprintf("%s", envVal)
					portSpecs := strings.Split(envValStr, ",")
					// There might be more than one exposed UI port, but this only handles the first listed,
					// most often there's only one.
					if len(portSpecs) > 0 {
						// HTTPS portSpecs typically look like <exposed>:<containerPort>, for example - HTTPS_EXPOSE=1359:1358
						ports := strings.Split(portSpecs[0], ":")
						//services[shortName][envName.(string)] = ports[0]
						switch envName {
						case "HTTP_EXPOSE":
							services[shortName]["http_url"] = "http://" + appDesc["hostname"].(string)
							if ports[0] != "80" {
								services[shortName]["http_url"] = services[shortName]["http_url"] + ":" + ports[0]
							}
						case "HTTPS_EXPOSE":
							services[shortName]["https_url"] = "https://" + appDesc["hostname"].(string)
							if ports[0] != "443" {
								services[shortName]["https_url"] = services[shortName]["https_url"] + ":" + ports[0]
							}
						}
					}
				}
			}
		}
		if shortName == "web" {
			services[shortName]["host_http_url"] = app.GetWebContainerDirectHTTPURL()
			services[shortName]["host_https_url"] = app.GetWebContainerDirectHTTPSURL()
		}
	}

	err = app.ProcessHooks("post-describe")
	if err != nil {
		return nil, fmt.Errorf("failed to process post-describe hooks: %v", err)
	}

	return appDesc, nil
}

// GetPublishedPort returns the host-exposed public port of a container.
func (app *DdevApp) GetPublishedPort(serviceName string) (int, error) {
	exposedPort := GetExposedPort(app, serviceName)
	exposedPortInt, err := strconv.Atoi(exposedPort)
	if err != nil {
		return -1, err
	}
	return app.GetPublishedPortForPrivatePort(serviceName, int64(exposedPortInt))
}

// GetPublishedPortForPrivatePort returns the host-exposed public port of a container for a given private port.
func (app *DdevApp) GetPublishedPortForPrivatePort(serviceName string, privatePort int64) (publicPort int, err error) {
	container, err := app.FindContainerByType(serviceName)
	if err != nil || container == nil {
		return -1, fmt.Errorf("failed to find container of type %s: %v", serviceName, err)
	}
	publishedPort := dockerutil.GetPublishedPort(privatePort, *container)
	return publishedPort, nil
}

// GetOmittedContainers returns full list of global and local omitted containers
func (app *DdevApp) GetOmittedContainers() []string {
	omitted := app.OmitContainersGlobal
	omitted = append(omitted, app.OmitContainers...)
	return omitted
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

// GetAbsDocroot returns the absolute path to the docroot on the host or if
// inContainer is set to true in the container.
func (app DdevApp) GetAbsDocroot(inContainer bool) string {
	if inContainer {
		return path.Join(app.GetAbsAppRoot(true), app.GetDocroot())
	}

	return filepath.Join(app.GetAbsAppRoot(false), app.GetDocroot())
}

// GetAbsAppRoot returns the absolute path to the project root on the host or if
// inContainer is set to true in the container.
func (app DdevApp) GetAbsAppRoot(inContainer bool) string {
	if inContainer {
		return "/var/www/html"
	}

	return app.AppRoot
}

// GetComposerRoot will determine the absolute composer root directory where
// all Composer related commands will be executed.
// If inContainer set to true, the absolute path in the container will be
// returned, else the absolute path on the host.
// If showWarning set to true, a warning containing the composer root will be
// shown to the user to avoid confusion.
func (app *DdevApp) GetComposerRoot(inContainer, showWarning bool) string {
	var absComposerRoot string

	if inContainer {
		absComposerRoot = path.Join(app.GetAbsAppRoot(true), app.ComposerRoot)
	} else {
		absComposerRoot = filepath.Join(app.GetAbsAppRoot(false), app.ComposerRoot)
	}

	// If requested, let the user know we are not using the default composer
	// root directory to avoid confusion.
	if app.ComposerRoot != "" && showWarning {
		util.Warning("Using '%s' as composer root directory", absComposerRoot)
	}

	return absComposerRoot
}

// GetName returns the app's name
func (app *DdevApp) GetName() string {
	return app.Name
}

// GetPhpVersion returns the app's php version
func (app *DdevApp) GetPhpVersion() string {
	v := nodeps.PHPDefault
	if app.PHPVersion != "" {
		v = app.PHPVersion
	}
	return v
}

// GetWebserverType returns the app's webserver type (nginx-fpm/apache-fpm)
func (app *DdevApp) GetWebserverType() string {
	v := nodeps.WebserverDefault
	if app.WebserverType != "" {
		v = app.WebserverType
	}
	return v
}

// GetRouterHTTPPort returns app's router http port
// Start with global config and then override with project config
func (app *DdevApp) GetRouterHTTPPort() string {
	port := globalconfig.DdevGlobalConfig.RouterHTTPPort
	if app.RouterHTTPPort != "" {
		port = app.RouterHTTPPort
	}
	return port
}

// GetRouterHTTPSPort returns app's router https port
// Start with global config and then override with project config
func (app *DdevApp) GetRouterHTTPSPort() string {
	port := globalconfig.DdevGlobalConfig.RouterHTTPSPort
	if app.RouterHTTPSPort != "" {
		port = app.RouterHTTPSPort
	}
	return port
}

// ImportDB takes a source sql dump and imports it to an active site's database container.
func (app *DdevApp) ImportDB(dumpFile string, extractPath string, progress bool, noDrop bool, targetDB string) error {
	app.DockerEnv()
	dockerutil.CheckAvailableSpace()

	if targetDB == "" {
		targetDB = "db"
	}
	var extPathPrompt bool
	dbPath, err := os.MkdirTemp(filepath.Dir(app.ConfigPath), ".importdb")
	if err != nil {
		return err
	}
	err = os.Chmod(dbPath, 0777)
	if err != nil {
		return err
	}

	defer func() {
		_ = os.RemoveAll(dbPath)
	}()

	err = app.ProcessHooks("pre-import-db")
	if err != nil {
		return err
	}

	// If they don't provide an import path and we're not on a tty (piped in stuff)
	// then prompt for path to db
	if dumpFile == "" && isatty.IsTerminal(os.Stdin.Fd()) {
		// ensure we prompt for extraction path if an archive is provided, while still allowing
		// non-interactive use of --file flag without providing a --extract-path flag.
		if extractPath == "" {
			extPathPrompt = true
		}
		output.UserOut.Println("Provide the path to the database you want to import.")
		fmt.Print("Path to file: ")

		dumpFile = util.GetInput("")
	}

	if dumpFile != "" {
		importPath, isArchive, err := appimport.ValidateAsset(dumpFile, "db")
		if err != nil {
			if isArchive && extPathPrompt {
				output.UserOut.Println("You provided an archive. Do you want to extract from a specific path in your archive? You may leave this blank if you wish to use the full archive contents")
				fmt.Print("Archive extraction path:")

				extractPath = util.GetInput("")
			} else {
				return fmt.Errorf("Unable to validate import asset %s: %s", dumpFile, err)
			}
		}

		switch {
		case strings.HasSuffix(importPath, "sql.gz") || strings.HasSuffix(importPath, "mysql.gz"):
			err = archive.Ungzip(importPath, dbPath)
			if err != nil {
				return fmt.Errorf("failed to extract provided file: %v", err)
			}

		case strings.HasSuffix(importPath, "sql.bz2") || strings.HasSuffix(importPath, "mysql.bz2"):
			err = archive.UnBzip2(importPath, dbPath)
			if err != nil {
				return fmt.Errorf("failed to extract file: %v", err)
			}

		case strings.HasSuffix(importPath, "sql.xz") || strings.HasSuffix(importPath, "mysql.xz"):
			err = archive.UnXz(importPath, dbPath)
			if err != nil {
				return fmt.Errorf("failed to extract file: %v", err)
			}

		case strings.HasSuffix(importPath, "zip"):
			err = archive.Unzip(importPath, dbPath, extractPath)
			if err != nil {
				return fmt.Errorf("failed to extract provided archive: %v", err)
			}

		case strings.HasSuffix(importPath, "tar"):
			fallthrough
		case strings.HasSuffix(importPath, "tar.gz"):
			fallthrough
		case strings.HasSuffix(importPath, "tar.bz2"):
			fallthrough
		case strings.HasSuffix(importPath, "tar.xz"):
			fallthrough
		case strings.HasSuffix(importPath, "tgz"):
			err := archive.Untar(importPath, dbPath, extractPath)
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

	// default insideContainerImportPath is the one mounted from .ddev directory
	insideContainerImportPath := path.Join("/mnt/ddev_config/", filepath.Base(dbPath))
	// But if we don't have bind mounts, we have to copy dump into the container
	if globalconfig.DdevGlobalConfig.NoBindMounts {
		dbContainerName := GetContainerName(app, "db")
		if err != nil {
			return err
		}
		uid, _, _ := util.GetContainerUIDGid()
		// for postgres, must be written with postgres user
		if app.Database.Type == nodeps.Postgres {
			uid = "999"
		}

		insideContainerImportPath, _, err = dockerutil.Exec(dbContainerName, "mktemp -d", uid)
		if err != nil {
			return err
		}
		insideContainerImportPath = strings.Trim(insideContainerImportPath, "\n")

		err = dockerutil.CopyIntoContainer(dbPath, dbContainerName, insideContainerImportPath, "")
		if err != nil {
			return err
		}
	}

	err = app.MutagenSyncFlush()
	if err != nil {
		return err
	}
	// The perl manipulation removes statements like CREATE DATABASE and USE, which
	// throw off imports. This is a scary manipulation, as it must not match actual content
	// as has actually happened with https://www.ddevhq.org/ddev-local/ddev-local-database-management/
	// and in https://github.com/ddev/ddev/issues/2787
	// The backtick after USE is inserted via fmt.Sprintf argument because it seems there's
	// no way to escape a backtick in a string literal.
	inContainerCommand := []string{}
	preImportSQL := ""
	switch app.Database.Type {
	case nodeps.MySQL:
		fallthrough
	case nodeps.MariaDB:
		preImportSQL = fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s; GRANT ALL ON %s.* TO 'db'@'%%';", targetDB, targetDB)
		if !noDrop {
			preImportSQL = fmt.Sprintf("DROP DATABASE IF EXISTS %s; ", targetDB) + preImportSQL
		}

		// Case for reading from file
		inContainerCommand = []string{"bash", "-c", fmt.Sprintf(`set -eu -o pipefail && mysql -uroot -proot -e "%s" && pv %s/*.*sql |  perl -p -e 's/^(CREATE DATABASE \/\*|USE %s)[^;]*;//' | mysql %s`, preImportSQL, insideContainerImportPath, "`", targetDB)}

		// Alternate case where we are reading from stdin
		if dumpFile == "" && extractPath == "" {
			inContainerCommand = []string{"bash", "-c", fmt.Sprintf(`set -eu -o pipefail && mysql -uroot -proot -e "%s" && perl -p -e 's/^(CREATE DATABASE \/\*|USE %s)[^;]*;//' | mysql %s`, preImportSQL, "`", targetDB)}
		}

	case nodeps.Postgres:
		preImportSQL = ""
		if !noDrop { // Normal case, drop and recreate database
			preImportSQL = preImportSQL + fmt.Sprintf(`
				DROP DATABASE IF EXISTS %s;
				CREATE DATABASE %s;
			`, targetDB, targetDB)
		} else { // Leave database alone, but create if not exists
			preImportSQL = preImportSQL + fmt.Sprintf(`
				SELECT 'CREATE DATABASE %s' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = '%s')\gexec
			`, targetDB, targetDB)
		}
		preImportSQL = preImportSQL + fmt.Sprintf(`
			GRANT ALL PRIVILEGES ON DATABASE %s TO db;`, targetDB)

		// If there is no import path, we're getting it from stdin
		if dumpFile == "" && extractPath == "" {
			inContainerCommand = []string{"bash", "-c", fmt.Sprintf(`set -eu -o pipefail && (echo '%s' | psql -d postgres) && psql -v ON_ERROR_STOP=1 -d %s`, preImportSQL, targetDB)}
		} else { // otherwise getting it from mounted file
			inContainerCommand = []string{"bash", "-c", fmt.Sprintf(`set -eu -o pipefail && (echo "%s" | psql -q -d postgres -v ON_ERROR_STOP=1) && pv %s/*.*sql | psql -q -v ON_ERROR_STOP=1 %s >/dev/null`, preImportSQL, insideContainerImportPath, targetDB)}
		}
	}
	stdout, stderr, err := app.Exec(&ExecOpts{
		Service: "db",
		RawCmd:  inContainerCommand,
		Tty:     progress && isatty.IsTerminal(os.Stdin.Fd()),
	})

	if err != nil {
		return fmt.Errorf("failed to import database: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}

	// Wait for import to really complete
	if app.Database.Type != nodeps.Postgres {
		rowsImported := 0
		for i := 0; i < 10; i++ {

			stdout, _, err := app.Exec(&ExecOpts{
				Cmd:     `mysqladmin -uroot -proot extended -r 2>/dev/null | awk -F'|' '/Innodb_rows_inserted/ {print $3}'`,
				Service: "db",
			})
			if err != nil {
				util.Warning("mysqladmin command failed: %v", err)
			}
			stdout = strings.Trim(stdout, "\r\n\t ")
			newRowsImported, err := strconv.Atoi(stdout)
			if err != nil {
				util.Warning("Error converting '%s' to int", stdout)
				break
			}
			// See if mysqld is still importing. If it is, sleep and try again
			if newRowsImported == rowsImported {
				break
			}
			rowsImported = newRowsImported
			time.Sleep(time.Millisecond * 500)
		}
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
// targetDB is the db name if not default "db"
func (app *DdevApp) ExportDB(dumpFile string, compressionType string, targetDB string) error {
	app.DockerEnv()
	exportCmd := []string{"mysqldump"}
	if app.Database.Type == "postgres" {
		exportCmd = []string{"pg_dump", "-U", "db"}
	}
	if targetDB == "" {
		targetDB = "db"
	}
	exportCmd = append(exportCmd, targetDB)

	if compressionType != "" {
		exportCmd = []string{"bash", "-c", fmt.Sprintf(`set -eu -o pipefail; %s | %s`, strings.Join(exportCmd, " "), compressionType)}
	}

	opts := &ExecOpts{
		Service:   "db",
		RawCmd:    exportCmd,
		NoCapture: true,
	}
	if dumpFile != "" {
		f, err := os.OpenFile(dumpFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return fmt.Errorf("failed to open %s: %v", dumpFile, err)
		}
		opts.Stdout = f
		defer func() {
			_ = f.Close()
		}()
	}
	stdout, stderr, err := app.Exec(opts)

	if err != nil {
		return fmt.Errorf("unable to export db: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}

	confMsg := "Wrote database dump from project '" + app.Name + "' database '" + targetDB + "'"
	if dumpFile != "" {
		confMsg = confMsg + " to file " + dumpFile
	} else {
		confMsg = confMsg + " to stdout"
	}
	if compressionType != "" {
		confMsg = fmt.Sprintf("%s in %s format", confMsg, compressionType)
	} else {
		confMsg = confMsg + " in plain text format"
	}

	_, err = fmt.Fprintf(os.Stderr, confMsg+".\n")

	return err
}

// SiteStatus returns the current status of a project determined from web and db service health.
// returns status, statusDescription
// Can return SiteConfigMissing, SiteDirMissing, SiteStopped, SiteStarting, SiteRunning, SitePaused,
// or another status returned from dockerutil.GetContainerHealth(), including
// "exited", "restarting", "healthy"
func (app *DdevApp) SiteStatus() (string, string) {
	if !fileutil.FileExists(app.GetAppRoot()) {
		return SiteDirMissing, fmt.Sprintf(`%s: %v; Please "ddev stop --unlist %s"`, SiteDirMissing, app.GetAppRoot(), app.Name)
	}

	_, err := CheckForConf(app.GetAppRoot())
	if err != nil {
		return SiteConfigMissing, fmt.Sprintf("%s", SiteConfigMissing)
	}

	statuses := map[string]string{"web": ""}
	if !nodeps.ArrayContainsString(app.GetOmittedContainers(), "db") {
		statuses["db"] = ""
	}

	for service := range statuses {
		container, err := app.FindContainerByType(service)
		if err != nil {
			util.Error("app.FindContainerByType(%v) failed", service)
			return "", ""
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

	siteStatusDesc := ""
	for serviceName, status := range statuses {
		if status != statuses["web"] {
			siteStatusDesc += serviceName + ": " + status + "\n"
		}
	}

	// Base the siteStatus on web container. Then override it if others are not the same.
	if siteStatusDesc == "" {
		return app.determineStatus(statuses), statuses["web"]
	}

	return app.determineStatus(statuses), siteStatusDesc
}

// Return one of the Site* statuses to describe the overall status of the project
func (app *DdevApp) determineStatus(statuses map[string]string) string {
	hasCommonStatus, commonStatus := app.getCommonStatus(statuses)

	if hasCommonStatus {
		return commonStatus
	}

	for status := range statuses {
		if status == SiteStarting {
			return SiteStarting
		}
	}

	return SiteUnhealthy
}

// Check whether a common status applies to all services
func (app *DdevApp) getCommonStatus(statuses map[string]string) (bool, string) {
	commonStatus := ""

	for _, status := range statuses {
		if commonStatus != "" && status != commonStatus {
			return false, ""
		}

		commonStatus = status
	}

	return true, commonStatus
}

// ImportFiles takes a source directory or archive and copies to the uploaded files directory of a given app.
func (app *DdevApp) ImportFiles(uploadDir, importPath, extractPath string) error {
	app.DockerEnv()

	if err := app.ProcessHooks("pre-import-files"); err != nil {
		return err
	}

	if uploadDir == "" {
		uploadDir = app.GetUploadDir()
		if uploadDir == "" {
			return fmt.Errorf("upload_dirs is not set, cannot import files")
		}
	}

	if err := app.dispatchImportFilesAction(uploadDir, importPath, extractPath); err != nil {
		return err
	}

	//nolint: revive
	if err := app.ProcessHooks("post-import-files"); err != nil {
		return err
	}

	return nil
}

// ComposeFiles returns a list of compose files for a project.
// It has to put the .ddev/docker-compose.*.y*ml first
// It has to put the docker-compose.override.y*l last
func (app *DdevApp) ComposeFiles() ([]string, error) {
	origDir, _ := os.Getwd()
	defer func() {
		_ = os.Chdir(origDir)
	}()
	err := os.Chdir(app.AppConfDir())
	if err != nil {
		return nil, err
	}
	files, err := filepath.Glob("docker-compose.*.y*ml")
	if err != nil {
		return []string{}, fmt.Errorf("unable to glob docker-compose.*.y*ml in %s: err=%v", app.AppConfDir(), err)
	}

	mainFile := app.DockerComposeYAMLPath()
	if !fileutil.FileExists(mainFile) {
		return nil, fmt.Errorf("failed to find %s", mainFile)
	}

	overrides, err := filepath.Glob("docker-compose.override.y*ml")
	util.CheckErr(err)

	orderedFiles := make([]string, 1)

	// Make sure the main file goes first
	orderedFiles[0] = mainFile

	for _, file := range files {
		// We already have the main file, and it's not in the list anyway, so skip when we hit it.
		// We'll add the override later, so skip it.
		if len(overrides) == 1 && file == overrides[0] {
			continue
		}
		orderedFiles = append(orderedFiles, app.GetConfigPath(file))
	}
	if len(overrides) == 1 {
		orderedFiles = append(orderedFiles, app.GetConfigPath(overrides[0]))
	}
	return orderedFiles, nil
}

// ProcessHooks executes Tasks defined in Hooks
func (app *DdevApp) ProcessHooks(hookName string) error {
	if cmds := app.Hooks[hookName]; len(cmds) > 0 {
		output.UserOut.Debugf("Executing %s hook...", hookName)
	}

	for _, c := range app.Hooks[hookName] {
		a := NewTask(app, c)
		if a == nil {
			return fmt.Errorf("unable to create task from %v", c)
		}

		if hookName == "pre-start" {
			for k := range c {
				if k == "exec" || k == "composer" {
					return fmt.Errorf("pre-start hooks cannot contain %v", k)
				}
			}
		}

		output.UserOut.Debugf("=== Running task: %s, output below", a.GetDescription())

		err := a.Execute()

		if err != nil {
			if app.FailOnHookFail || app.FailOnHookFailGlobal {
				output.UserOut.Errorf("Task failed: %v: %v", a.GetDescription(), err)
				return fmt.Errorf("task failed: %v", err)
			}
			output.UserOut.Errorf("Task failed: %v: %v", a.GetDescription(), err)
			output.UserOut.Warn("A task failure does not mean that ddev failed, but your hook configuration has a command that failed.")
		}
	}

	return nil
}

// GetDBImage uses the available version info
func (app *DdevApp) GetDBImage() string {
	dbImage := dockerImages.GetDBImage(app.Database.Type, app.Database.Version)
	return dbImage
}

// Start initiates docker-compose up
func (app *DdevApp) Start() error {
	var err error

	if app.IsMutagenEnabled() && globalconfig.DdevGlobalConfig.UseHardenedImages {
		return fmt.Errorf("mutagen is not compatible with use-hardened-images")
	}

	app.DockerEnv()
	dockerutil.EnsureDdevNetwork()

	if err = dockerutil.CheckDockerCompose(); err != nil {
		util.Failed(`Your docker-compose version does not exist or is set to an invalid version.
Please use the built-in docker-compose.
Fix with 'ddev config global --required-docker-compose-version="" --use-docker-compose-from-path=false': %v`, err)
	}

	err = PullBaseContainerImages()
	if err != nil {
		util.Warning("Unable to pull docker images: %v", err)
	}

	if !nodeps.ArrayContainsString(app.GetOmittedContainers(), "db") {
		// OK to start if dbType is empty (nonexistent) or if it matches
		if dbType, err := app.GetExistingDBType(); err != nil || (dbType != "" && dbType != app.Database.Type+":"+app.Database.Version) {
			return fmt.Errorf("Unable to start project %s because the configured database type does not match the current actual database. Please change your database type back to %s and start again, export, delete, and then change configuration and start. To get back to existing type use 'ddev config --database=%s' and then you might want to try 'ddev debug migrate-database %s', see docs at %s", app.Name, dbType, dbType, app.Database.Type+":"+app.Database.Version, "https://ddev.readthedocs.io/en/latest/users/extend/database-types/")
		}
	}

	app.CreateUploadDirsIfNecessary()

	if app.IsMutagenEnabled() {
		err = app.GenerateMutagenYml()
		if err != nil {
			return err
		}
		if ok, volumeExists, info := CheckMutagenVolumeSyncCompatibility(app); !ok {
			util.Debug("mutagen sync session, configuration, and docker volume are in incompatible status: '%s', Removing mutagen sync session '%s' and docker volume %s", info, MutagenSyncName(app.Name), GetMutagenVolumeName(app))
			err = SyncAndPauseMutagenSession(app)
			if err != nil {
				util.Warning("Unable to SyncAndPauseMutagenSession() %s: %v", MutagenSyncName(app.Name), err)
			}
			terminateErr := TerminateMutagenSync(app)
			if terminateErr != nil {
				util.Warning("Unable to terminate mutagen sync %s: %v", MutagenSyncName(app.Name), err)
			}
			if volumeExists {
				// Remove mounting container if necessary.
				container, err := dockerutil.FindContainerByName("ddev-" + app.Name + "-web")
				if err == nil && container != nil {
					err = dockerutil.RemoveContainer(container.ID)
					if err != nil {
						return fmt.Errorf(`Unable to remove web container, please 'ddev restart': %v`, err)
					}
				}
				removeVolumeErr := dockerutil.RemoveVolume(GetMutagenVolumeName(app))
				if removeVolumeErr != nil {
					return fmt.Errorf(`Unable to remove mismatched mutagen docker volume '%s'. Please use 'ddev restart' or 'ddev mutagen reset': %v`, GetMutagenVolumeName(app), removeVolumeErr)
				}
			}
		}
		// Check again to make sure the mutagen docker volume exists. It's compatible if we found it above
		// so we can keep it in that case.
		if !dockerutil.VolumeExists(GetMutagenVolumeName(app)) {
			_, err = dockerutil.CreateVolume(GetMutagenVolumeName(app), "local", nil, map[string]string{mutagenSignatureLabelName: GetDefaultMutagenVolumeSignature(app)})
			if err != nil {
				return fmt.Errorf("Unable to create new mutagen docker volume %s: %v", GetMutagenVolumeName(app), err)
			}
		}
	}

	volumesNeeded := []string{"ddev-global-cache", "ddev-" + app.Name + "-snapshots"}
	for _, v := range volumesNeeded {
		_, err = dockerutil.CreateVolume(v, "local", nil, nil)
		if err != nil {
			return fmt.Errorf("unable to create docker volume %s: %v", v, err)
		}
	}

	err = app.CheckExistingAppInApproot()
	if err != nil {
		return err
	}

	// This is done early here so users won't see gitignored contents of .ddev for too long
	// It also gets done by `ddev config`
	err = PrepDdevDirectory(app)
	if err != nil {
		util.Warning("Unable to PrepDdevDirectory: %v", err)
	}

	err = PopulateCustomCommandFiles(app)
	if err != nil {
		util.Warning("Failed to populate custom command files: %v", err)
	}

	// The .ddev directory may still need to be populated, especially in tests
	err = PopulateExamplesCommandsHomeadditions(app.Name)
	if err != nil {
		return err
	}
	// Make sure that any ports allocated are available.
	// and of course add to global project list as well
	err = app.UpdateGlobalProjectList()
	if err != nil {
		return err
	}

	err = DownloadMutagenIfNeededAndEnabled(app)
	if err != nil {
		return err
	}

	err = app.ProcessHooks("pre-start")
	if err != nil {
		return err
	}

	err = app.GenerateWebserverConfig()
	if err != nil {
		return err
	}

	err = app.GeneratePostgresConfig()
	if err != nil {
		return err
	}

	dockerutil.CheckAvailableSpace()

	// Copy any homeadditions content into .ddev/.homeadditions
	tmpHomeadditionsPath := app.GetConfigPath(".homeadditions")
	err = os.RemoveAll(tmpHomeadditionsPath)
	if err != nil {
		util.Warning("unable to remove %s: %v", tmpHomeadditionsPath, err)
	}
	globalHomeadditionsPath := filepath.Join(globalconfig.GetGlobalDdevDir(), "homeadditions")
	if fileutil.IsDirectory(globalHomeadditionsPath) {
		err = copy.Copy(globalHomeadditionsPath, tmpHomeadditionsPath, copy.Options{OnSymlink: func(string) copy.SymlinkAction { return copy.Deep }})
		if err != nil {
			return err
		}
	}
	projectHomeAdditionsPath := app.GetConfigPath("homeadditions")
	if fileutil.IsDirectory(projectHomeAdditionsPath) {
		err = copy.Copy(projectHomeAdditionsPath, tmpHomeadditionsPath, copy.Options{OnSymlink: func(string) copy.SymlinkAction { return copy.Deep }})
		if err != nil {
			return err
		}
	}

	// Make sure that important volumes to mount already have correct ownership set
	// Additional volumes can be added here. This allows us to run a single privileged
	// container with a single focus of changing ownership, instead of having to use sudo
	// inside the container
	uid, _, _ := util.GetContainerUIDGid()

	if globalconfig.DdevGlobalConfig.NoBindMounts {
		err = dockerutil.CopyIntoVolume(app.GetConfigPath(""), app.Name+"-ddev-config", "", uid, "db_snapshots", true)
		if err != nil {
			return fmt.Errorf("failed to copy project .ddev directory to volume: %v", err)
		}
	}

	// TODO: We shouldn't be chowning /var/lib/mysql if postgresql?
	util.Debug("chowning /mnt/ddev-global-cache and /var/lib/mysql to %s", uid)
	_, out, err := dockerutil.RunSimpleContainer(dockerImages.GetWebImage(), "start-chown-"+util.RandString(6), []string{"sh", "-c", fmt.Sprintf("chown -R %s /var/lib/mysql /mnt/ddev-global-cache", uid)}, []string{}, []string{}, []string{app.GetMariaDBVolumeName() + ":/var/lib/mysql", "ddev-global-cache:/mnt/ddev-global-cache"}, "", true, false, map[string]string{"com.ddev.site-name": app.Name}, nil)
	if err != nil {
		return fmt.Errorf("failed to RunSimpleContainer to chown volumes: %v, output=%s", err, out)
	}
	util.Debug("done chowning /mnt/ddev-global-cache and /var/lib/mysql to %s", uid)

	// Chown the postgres volume; this shouldn't have to be a separate stanza, but the
	// uid is 999 instead of current user
	if app.Database.Type == nodeps.Postgres {
		util.Debug("chowning chowning /var/lib/postgresql/data to 999")
		_, out, err := dockerutil.RunSimpleContainer(dockerImages.GetWebImage(), "start-postgres-chown-"+util.RandString(6), []string{"sh", "-c", fmt.Sprintf("chown -R %s /var/lib/postgresql/data", "999:999")}, []string{}, []string{}, []string{app.GetPostgresVolumeName() + ":/var/lib/postgresql/data"}, "", true, false, map[string]string{"com.ddev.site-name": app.Name}, nil)
		if err != nil {
			return fmt.Errorf("failed to RunSimpleContainer to chown postgres volume: %v, output=%s", err, out)
		}
		util.Debug("done chowning /var/lib/postgresql/data")
	}

	if !nodeps.ArrayContainsString(app.GetOmittedContainers(), "ddev-ssh-agent") {
		err = app.EnsureSSHAgentContainer()
		if err != nil {
			return err
		}
	}

	// Warn the user if there is any custom configuration in use.
	app.CheckCustomConfig()

	// Warn user if there are deprecated items used in the config
	app.CheckDeprecations()

	// Fix any obsolete things like old shell commands, etc.
	app.FixObsolete()

	// WriteConfig .ddev-docker-compose-*.yaml
	err = app.WriteDockerComposeYAML()
	if err != nil {
		return err
	}

	// This needs to be done after WriteDockerComposeYAML() to get the right images
	err = app.PullContainerImages()
	if err != nil {
		util.Warning("Unable to pull docker images: %v", err)
	}

	err = app.CheckAddonIncompatibilities()
	if err != nil {
		return err
	}

	err = app.AddHostsEntriesIfNeeded()
	if err != nil {
		return err
	}

	// Delete the NFS volumes before we bring up docker-compose (and will be created again)
	// We don't care if the volume wasn't there
	_ = dockerutil.RemoveVolume(app.GetNFSMountVolumeName())

	// The db_snapshots subdirectory may be created on docker-compose up, so
	// we need to precreate it so permissions are correct (and not root:root)
	if !fileutil.IsDirectory(app.GetConfigPath("db_snapshots")) {
		err = os.MkdirAll(app.GetConfigPath("db_snapshots"), 0777)
		if err != nil {
			return err
		}
	}
	// db_snapshots gets mounted into container, may have different user/group, so need 777
	err = os.Chmod(app.GetConfigPath("db_snapshots"), 0777)
	if err != nil {
		return err
	}

	// Build extra layers on web and db images if necessary
	progress := "quiet"
	if globalconfig.DdevVerbose {
		progress = "auto"
	}
	util.Debug("Executing docker-compose -f %s build --progress=%s", app.DockerComposeFullRenderedYAMLPath(), progress)
	out, stderr, err := dockerutil.ComposeCmd([]string{app.DockerComposeFullRenderedYAMLPath()}, "--progress="+progress, "build")
	if err != nil {
		return fmt.Errorf("docker-compose build failed: %v, output='%s', stderr='%s'", err, out, stderr)
	}
	if globalconfig.DdevVerbose {
		util.Debug("docker-compose build output:\n%s\n\n", out)
	}

	util.Debug("Executing docker-compose -f %s up -d", app.DockerComposeFullRenderedYAMLPath())
	_, _, err = dockerutil.ComposeCmd([]string{app.DockerComposeFullRenderedYAMLPath()}, "up", "-d")
	if err != nil {
		return err
	}

	if !IsRouterDisabled(app) {
		caRoot := globalconfig.GetCAROOT()
		if caRoot == "" {
			util.Warning("mkcert may not be properly installed, we suggest installing it for trusted https support, `brew install mkcert nss`, `choco install -y mkcert`, etc. and then `mkcert -install`")
		}
		router, _ := FindDdevRouter()

		// If the router doesn't exist, go ahead and push mkcert root ca certs into the ddev-global-cache/mkcert
		// This will often be redundant
		if router == nil {
			// Copy ca certs into ddev-global-cache/mkcert
			if caRoot != "" {
				uid, _, _ := util.GetContainerUIDGid()
				err = dockerutil.CopyIntoVolume(caRoot, "ddev-global-cache", "mkcert", uid, "", false)
				if err != nil {
					util.Warning("failed to copy root CA into docker volume ddev-global-cache/mkcert: %v", err)
				} else {
					util.Debug("Pushed mkcert rootca certs to ddev-global-cache/mkcert")
				}
			}
		}

		// If TLS supported and using traefik, create cert/key and push into ddev-global-cache/traefik
		if globalconfig.DdevGlobalConfig.IsTraefikRouter() {
			err = configureTraefikForApp(app)
			if err != nil {
				return err
			}
		}

		// Push custom certs
		targetSubdir := "custom_certs"
		if globalconfig.DdevGlobalConfig.IsTraefikRouter() {
			targetSubdir = path.Join("traefik", "certs")
		}
		certPath := app.GetConfigPath("custom_certs")
		uid, _, _ := util.GetContainerUIDGid()
		if fileutil.FileExists(certPath) && globalconfig.DdevGlobalConfig.MkcertCARoot != "" {
			err = dockerutil.CopyIntoVolume(certPath, "ddev-global-cache", targetSubdir, uid, "", false)
			if err != nil {
				util.Warning("failed to copy custom certs into docker volume ddev-global-cache/custom_certs: %v", err)
			} else {
				util.Debug("Installed custom cert from %s", certPath)
			}
		}
	}

	if app.IsMutagenEnabled() {
		app.checkMutagenUploadDirs()

		mounted, err := IsMutagenVolumeMounted(app)
		if err != nil {
			return err
		}
		if !mounted {
			util.Failed("Mutagen docker volume is not mounted. Please use `ddev restart`")
		}
		output.UserOut.Printf("Starting mutagen sync process... This can take some time.")
		mutagenDuration := util.ElapsedDuration(time.Now())

		err = SetMutagenVolumeOwnership(app)
		if err != nil {
			return err
		}
		err = CreateOrResumeMutagenSync(app)
		if err != nil {
			return fmt.Errorf("Failed to create mutagen sync session '%s'. You may be able to resolve this problem using 'ddev mutagen reset' (err=%v)", MutagenSyncName(app.Name), err)
		}
		mStatus, _, _, err := app.MutagenStatus()
		if err != nil {
			return err
		}
		util.Debug("mutagen status after sync: %s", mStatus)

		dur := util.FormatDuration(mutagenDuration())
		if mStatus == "ok" {
			util.Success("Mutagen sync flush completed in %s.\nFor details on sync status 'ddev mutagen st %s -l'", dur, MutagenSyncName(app.Name))
		} else {
			util.Error("Mutagen sync completed with problems in %s.\nFor details on sync status 'ddev mutagen st %s -l'", dur, MutagenSyncName(app.Name))
		}
		err = fileutil.TemplateStringToFile(`#ddev-generated`, nil, app.GetConfigPath("mutagen/.start-synced"))
		if err != nil {
			util.Warning("could not create file %s: %v", app.GetConfigPath("mutagen/.start-synced"), err)
		}
	}

	// At this point we should have all files synced inside the container
	// Verify that we have composer.json inside container if we have it in project root
	// This is possibly a temporary need for debugging https://github.com/ddev/ddev/issues/5089
	// TODO: Consider removing this check when #5089 is resolved, or at least by 2024-01-01
	if !app.NoProjectMount && fileutil.FileExists(filepath.Join(app.GetComposerRoot(false, false), "composer.json")) {
		util.Debug("Checking for composer.json in container")
		stdout, stderr, err := app.Exec(&ExecOpts{
			Cmd: fmt.Sprintf("test -f %s", path.Join(app.GetComposerRoot(true, false), "composer.json")),
		})

		if err != nil {
			return fmt.Errorf("composer.json not found in container, stdout='%s', stderr='%s': %v; please report this situation, https://github.com/ddev/ddev/issues; probably can be fixed with ddev restart", stdout, stderr, err)
		}
	}

	util.Debug("Running /start.sh in ddev-webserver")
	stdout, stderr, err := app.Exec(&ExecOpts{
		// Send output to /var/tmp/logpipe to get it to docker logs
		// If start.sh dies, we want to make sure the container gets killed off
		// so send SIGTERM to process ID 1
		Cmd:    `/start.sh > /var/tmp/logpipe 2>&1 || kill -- -1`,
		Detach: true,
	})
	if err != nil {
		util.Warning("Unable to run /start.sh, stdout=%s, stderr=%s: %v", stdout, stderr, err)
	}

	// Wait for web/db containers to become healthy
	dependers := []string{"web"}
	if !nodeps.ArrayContainsString(app.GetOmittedContainers(), "db") {
		dependers = append(dependers, "db")
	}
	err = app.Wait(dependers)
	if err != nil {
		util.Warning("Failed waiting for web/db containers to become ready: %v", err)
	}

	if globalconfig.DdevVerbose {
		out, err = app.CaptureLogs("web", true, "200")
		if err != nil {
			util.Warning("Unable to capture logs from web container: %v", err)
		} else {
			util.Debug("docker-compose up output:\n%s\n\n", out)
		}
	}

	// WebExtraDaemons have to be started after mutagen sync is done, because so often
	// they depend on code being synced into the container/volume
	if len(app.WebExtraDaemons) > 0 {
		util.Debug("Starting web_extra_daaemons")
		stdout, stderr, err := app.Exec(&ExecOpts{
			Cmd: `supervisorctl start webextradaemons:*`,
		})
		if err != nil {
			util.Warning("Unable to start web_extra_daemons using supervisorctl, stdout=%s, stderr=%s: %v", stdout, stderr, err)
		}
	}

	util.Debug("Testing to see if /mnt/ddev_config is properly mounted")
	_, _, err = app.Exec(&ExecOpts{
		Cmd: `ls -l /mnt/ddev_config/nginx_full/nginx-site.conf >/dev/null`,
	})
	if err != nil {
		util.Warning("Something is wrong with docker or colima and /mnt/ddev_config is not mounted from the project .ddev folder. This can cause all kinds of problems.")
	}

	if !IsRouterDisabled(app) {
		err = StartDdevRouter()
		if err != nil {
			return err
		}
	}

	util.Debug("Waiting for all project containers to become ready")
	err = app.WaitByLabels(map[string]string{"com.ddev.site-name": app.GetName()})
	if err != nil {
		return err
	}
	util.Debug("Project containers are now ready")

	if _, err = app.CreateSettingsFile(); err != nil {
		return fmt.Errorf("failed to write settings file %s: %v", app.SiteDdevSettingsFile, err)
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

// Restart does a Stop() and a Start
func (app *DdevApp) Restart() error {
	err := app.Stop(false, false)
	if err != nil {
		return err
	}
	err = app.Start()
	return err
}

// PullContainerImages configured docker images with full output, since docker-compose up doesn't have nice output
func (app *DdevApp) PullContainerImages() error {
	images, err := app.FindAllImages()
	if err != nil {
		return err
	}

	images = append(images, dockerImages.GetRouterImage(), dockerImages.GetSSHAuthImage())
	for _, i := range images {
		err := dockerutil.Pull(i)
		if err != nil {
			return err
		}
		util.Debug("Pulled image for %s", i)
	}

	return nil
}

// PullBaseontainerImages pulls only the fundamentally needed images so they can be available early
// We always need web image and busybox just for housekeeping.
func PullBaseContainerImages() error {
	images := []string{dockerImages.GetWebImage(), versionconstants.BusyboxImage}
	if !nodeps.ArrayContainsString(globalconfig.DdevGlobalConfig.OmitContainersGlobal, SSHAuthName) {
		images = append(images, dockerImages.GetSSHAuthImage())
	}
	if !nodeps.ArrayContainsString(globalconfig.DdevGlobalConfig.OmitContainersGlobal, RouterProjectName) {
		images = append(images, dockerImages.GetRouterImage())
	}

	for _, i := range images {
		err := dockerutil.Pull(i)
		if err != nil {
			return err
		}
		util.Debug("Pulled image for %s", i)
	}

	return nil
}

// FindAllImages returns an array of image tags for all containers in the compose file
func (app *DdevApp) FindAllImages() ([]string, error) {
	var images []string
	if app.ComposeYaml == nil {
		return images, nil
	}
	if y, ok := app.ComposeYaml["services"]; ok {
		for _, v := range y.(map[string]interface{}) {
			if i, ok := v.(map[string]interface{})["image"]; ok {
				if strings.HasSuffix(i.(string), "-built") {
					i = strings.TrimSuffix(i.(string), "-built")
					if strings.HasSuffix(i.(string), "-"+app.Name) {
						i = strings.TrimSuffix(i.(string), "-"+app.Name)
					}
				}
				images = append(images, i.(string))
			}
		}
	}

	return images, nil
}

// FindMaxTimeout looks through all services and returns the max timeout found
// Defaults to 120
func (app *DdevApp) FindMaxTimeout() int {
	const defaultContainerTimeout = 120
	maxTimeout := defaultContainerTimeout
	if app.ComposeYaml == nil {
		return defaultContainerTimeout
	}
	if y, ok := app.ComposeYaml["services"]; ok {
		for _, v := range y.(map[string]interface{}) {
			if i, ok := v.(map[string]interface{})["healthcheck"]; ok {
				if timeout, ok := i.(map[string]interface{})["timeout"]; ok {
					duration, err := time.ParseDuration(timeout.(string))
					if err != nil {
						continue
					}
					t := int(duration.Seconds())
					if t > maxTimeout {
						maxTimeout = t
					}
				}
			}
		}
	}
	return maxTimeout
}

// CheckExistingAppInApproot looks to see if we already have a project in this approot with different name
func (app *DdevApp) CheckExistingAppInApproot() error {
	pList := globalconfig.GetGlobalProjectList()
	for name, v := range pList {
		if app.AppRoot == v.AppRoot && name != app.Name {
			return fmt.Errorf(`this project root %s already contains a project named %s. You may want to remove the existing project with "ddev stop --unlist %s"`, v.AppRoot, name, name)
		}
	}
	return nil
}

//go:embed webserver_config_assets
var webserverConfigAssets embed.FS

// GenerateWebserverConfig generates the default nginx and apache config files
func (app *DdevApp) GenerateWebserverConfig() error {
	// Prevent running as root for most cases
	// We really don't want ~/.ddev to have root ownership, breaks things.
	if os.Geteuid() == 0 {
		util.Warning("not generating webserver config files because running with root privileges")
		return nil
	}

	var items = map[string]string{
		"nginx":                         app.GetConfigPath(filepath.Join("nginx_full", "nginx-site.conf")),
		"apache":                        app.GetConfigPath(filepath.Join("apache", "apache-site.conf")),
		"nginx_second_docroot_example":  app.GetConfigPath(filepath.Join("nginx_full", "seconddocroot.conf.example")),
		"README.nginx_full.txt":         app.GetConfigPath(filepath.Join("nginx_full", "README.nginx_full.txt")),
		"README.apache.txt":             app.GetConfigPath(filepath.Join("apache", "README.apache.txt")),
		"apache_second_docroot_example": app.GetConfigPath(filepath.Join("apache", "seconddocroot.conf.example")),
	}
	for t, configPath := range items {
		err := os.MkdirAll(filepath.Dir(configPath), 0755)
		if err != nil {
			return err
		}

		if fileutil.FileExists(configPath) {
			sigExists, err := fileutil.FgrepStringInFile(configPath, nodeps.DdevFileSignature)
			if err != nil {
				return err
			}
			// If the signature doesn't exist, they have taken over the file, so return
			if !sigExists {
				return nil
			}
		}

		cfgFile := fmt.Sprintf("%s-site-%s.conf", t, app.Type)
		c, err := webserverConfigAssets.ReadFile(path.Join("webserver_config_assets", cfgFile))
		if err != nil {
			c, err = webserverConfigAssets.ReadFile(path.Join("webserver_config_assets", fmt.Sprintf("%s-site-php.conf", t)))
			if err != nil {
				return err
			}
		}
		content := string(c)
		docroot := path.Join("/var/www/html", app.Docroot)
		err = fileutil.TemplateStringToFile(content, map[string]interface{}{"Docroot": docroot}, configPath)
		if err != nil {
			return err
		}
	}
	return nil
}

func (app *DdevApp) GeneratePostgresConfig() error {
	if app.Database.Type != nodeps.Postgres {
		return nil
	}
	// Prevent running as root for most cases
	// We really don't want ~/.ddev to have root ownership, breaks things.
	if os.Geteuid() == 0 {
		util.Warning("not generating postgres config files because running with root privileges")
		return nil
	}

	var items = map[string]string{
		"postgresql.conf": app.GetConfigPath(filepath.Join("postgres", "postgresql.conf")),
	}
	for _, configPath := range items {
		err := os.MkdirAll(filepath.Dir(configPath), 0755)
		if err != nil {
			return err
		}

		if fileutil.FileExists(configPath) {
			err = os.Chmod(configPath, 0666)
			if err != nil {
				return err
			}
			sigExists, err := fileutil.FgrepStringInFile(configPath, nodeps.DdevFileSignature)
			if err != nil {
				return err
			}
			// If the signature doesn't exist, they have taken over the file, so return
			if !sigExists {
				return nil
			}
		}

		c, err := bundledAssets.ReadFile(path.Join("postgres", app.Database.Version, "postgresql.conf"))
		if err != nil {
			return err
		}
		err = fileutil.TemplateStringToFile(string(c), nil, configPath)
		if err != nil {
			return err
		}
		err = os.Chmod(configPath, 0666)
		if err != nil {
			return err
		}
	}
	return nil
}

// ExecOpts contains options for running a command inside a container
type ExecOpts struct {
	// Service is the service, as in 'web', 'db'
	Service string
	// Dir is the full path to the working directory inside the container
	Dir string
	// Cmd is the string to execute via bash/sh
	Cmd string
	// RawCmd is the array to execute if not using
	RawCmd []string
	// Nocapture if true causes use of ComposeNoCapture, so the stdout and stderr go right to stdout/stderr
	NoCapture bool
	// Tty if true causes a tty to be allocated
	Tty bool
	// Stdout can be overridden with a File
	Stdout *os.File
	// Stderr can be overridden with a File
	Stderr *os.File
	// Detach does docker-compose detach
	Detach bool
}

// Exec executes a given command in the container of given type without allocating a pty
// Returns ComposeCmd results of stdout, stderr, err
// If Nocapture arg is true, stdout/stderr will be empty and output directly to stdout/stderr
func (app *DdevApp) Exec(opts *ExecOpts) (string, string, error) {
	app.DockerEnv()

	defer util.TimeTrackC(fmt.Sprintf("app.Exec %v", opts))()

	if opts.Cmd == "" && len(opts.RawCmd) == 0 {
		return "", "", fmt.Errorf("no command provided")
	}

	if opts.Service == "" {
		opts.Service = "web"
	}

	state, err := dockerutil.GetContainerStateByName(fmt.Sprintf("ddev-%s-%s", app.Name, opts.Service))
	if err != nil || state != "running" {
		switch state {
		case "doesnotexist":
			return "", "", fmt.Errorf("service %s does not exist in project %s (state=%s)", opts.Service, app.Name, state)
		case "exited":
			return "", "", fmt.Errorf("service %s has exited; state=%s", opts.Service, state)
		default:
			return "", "", fmt.Errorf("service %s is not currently running in project %s (state=%s), use `ddev logs -s %s` to see what happened to it", opts.Service, app.Name, state, opts.Service)
		}
	}

	err = app.ProcessHooks("pre-exec")
	if err != nil {
		return "", "", fmt.Errorf("failed to process pre-exec hooks: %v", err)
	}

	baseComposeExecCmd := []string{"exec"}
	if opts.Dir != "" {
		baseComposeExecCmd = append(baseComposeExecCmd, "-w", opts.Dir)
	}

	if !isatty.IsTerminal(os.Stdin.Fd()) || !opts.Tty {
		baseComposeExecCmd = append(baseComposeExecCmd, "-T")
	}

	if opts.Detach {
		baseComposeExecCmd = append(baseComposeExecCmd, "--detach")
	}

	baseComposeExecCmd = append(baseComposeExecCmd, opts.Service)

	// Cases to handle
	// - Free form, all unquoted. Like `ls -l -a`
	// - Quoted to delay pipes and other features to container, like `"ls -l -a | grep junk"`
	// Note that a set quoted on the host in ddev e will come through as a single arg

	if len(opts.RawCmd) == 0 { // Use opts.Cmd and prepend with bash
		// Use bash for our containers, sh for 3rd-party containers
		// that may not have bash.
		shell := "bash"
		if !nodeps.ArrayContainsString([]string{"web", "db"}, opts.Service) {
			shell = "sh"
		}
		errcheck := "set -eu"
		opts.RawCmd = []string{shell, "-c", errcheck + ` && ( ` + opts.Cmd + `)`}
	}
	files := []string{app.DockerComposeFullRenderedYAMLPath()}
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
	var outRes, errRes string
	r := append(baseComposeExecCmd, opts.RawCmd...)
	if opts.NoCapture || opts.Tty {
		err = dockerutil.ComposeWithStreams(files, os.Stdin, stdout, stderr, r...)
	} else {
		outRes, errRes, err = dockerutil.ComposeCmd([]string{app.DockerComposeFullRenderedYAMLPath()}, r...)
		stdoutResult = outRes
		stderrResult = errRes
	}
	if err != nil {
		return stdoutResult, stderrResult, err
	}
	hookErr := app.ProcessHooks("post-exec")
	if hookErr != nil {
		return stdoutResult, stderrResult, fmt.Errorf("failed to process post-exec hooks: %v", hookErr)
	}
	return stdoutResult, stderrResult, err
}

// ExecWithTty executes a given command in the container of given type.
// It allocates a pty for interactive work.
func (app *DdevApp) ExecWithTty(opts *ExecOpts) error {
	app.DockerEnv()

	if opts.Service == "" {
		opts.Service = "web"
	}

	state, err := dockerutil.GetContainerStateByName(fmt.Sprintf("ddev-%s-%s", app.Name, opts.Service))
	if err != nil || state != "running" {
		return fmt.Errorf("service %s is not running in project %s (state=%s)", opts.Service, app.Name, state)
	}

	args := []string{"exec"}

	// In the case where this is being used without an available tty,
	// make sure we use the -T to turn off tty to avoid panic in docker-compose v2.2.3
	// see https://stackoverflow.com/questions/70855915/fix-panic-provided-file-is-not-a-console-from-docker-compose-in-github-action
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		args = append(args, "-T")
	}

	if opts.Dir != "" {
		args = append(args, "-w", opts.Dir)
	}

	args = append(args, opts.Service)

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
	if !nodeps.ArrayContainsString([]string{"web", "db"}, opts.Service) {
		shell = "sh"
	}
	args = append(args, shell, "-c", opts.Cmd)

	return dockerutil.ComposeWithStreams([]string{app.DockerComposeFullRenderedYAMLPath()}, os.Stdin, os.Stdout, os.Stderr, args...)
}

func (app *DdevApp) ExecOnHostOrService(service string, cmd string) error {
	var err error
	// Handle case on host
	if service == "host" {
		cwd, _ := os.Getwd()
		err = os.Chdir(app.GetAppRoot())
		if err != nil {
			return fmt.Errorf("unable to GetAppRoot: %v", err)
		}
		bashPath := "bash"
		if runtime.GOOS == "windows" {
			bashPath = util.FindBashPath()
			if bashPath == "" {
				return fmt.Errorf("unable to find bash.exe on Windows")
			}
		}

		args := []string{
			"-c",
			cmd,
		}

		app.DockerEnv()
		err = exec.RunInteractiveCommand(bashPath, args)
		_ = os.Chdir(cwd)
	} else { // handle case in container
		_, _, err = app.Exec(
			&ExecOpts{
				Service: service,
				Cmd:     cmd,
				Tty:     isatty.IsTerminal(os.Stdin.Fd()),
			})
	}
	return err
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

	// Warn about running as root if we're not on Windows.
	if uidStr == "0" || gidStr == "0" {
		util.Warning("Warning: containers will run as root. This could be a security risk on Linux.")
	}

	// For gitpod, codespaces
	// * provide default host-side port bindings, assuming only one project running,
	//   as is usual on gitpod, but if more than one project, can override with normal
	//   config.yaml settings.
	if nodeps.IsGitpod() || nodeps.IsCodespaces() {
		if app.HostWebserverPort == "" {
			app.HostWebserverPort = "8080"
		}
		if app.HostHTTPSPort == "" {
			app.HostHTTPSPort = "8443"
		}
		if app.HostDBPort == "" {
			app.HostDBPort = "3306"
		}
		if app.HostMailhogPort == "" {
			app.HostMailhogPort = "8027"
		}
		app.BindAllInterfaces = true
	}
	isWSL2 := "false"
	if nodeps.IsWSL2() {
		isWSL2 = "true"
	}

	// DDEV_HOST_DB_PORT is actually used for 2 things.
	// 1. To specify via base docker-compose file the value of host_db_port config. And it's expected to be empty
	//    there if the host_db_port is empty.
	// 2. To tell custom commands the db port. And it's expected always to be populated for them.
	dbPort, err := app.GetPublishedPort("db")
	dbPortStr := strconv.Itoa(dbPort)
	if dbPortStr == "-1" || err != nil {
		dbPortStr = ""
	}
	if app.HostDBPort != "" {
		dbPortStr = app.HostDBPort
	}

	// Figure out what the host-webserver port is
	hostHTTPPort, err := app.GetPublishedPort("web")
	hostHTTPPortStr := ""
	if hostHTTPPort > 0 || err == nil {
		hostHTTPPortStr = strconv.Itoa(hostHTTPPort)
	}

	// Figure out what the host-webserver https port is
	// the https port is rarely used because ddev-router does termination
	// for the vast majority of applications
	hostHTTPSPort, err := app.GetPublishedPortForPrivatePort("web", 443)
	hostHTTPSPortStr := ""
	if hostHTTPSPort > 0 || err == nil {
		hostHTTPSPortStr = strconv.Itoa(hostHTTPSPort)
	}

	// DDEV_DATABASE_FAMILY can be use for connection URLs
	// Eg. mysql://db@db:3033/db
	dbFamily := "mysql"
	if app.Database.Type == "postgres" {
		// 'postgres' & 'postgresql' are both valid, but we'll go with the shorter one.
		dbFamily = "postgres"
	}

	envVars := map[string]string{
		// The compose project name can no longer contain dots; must be lower-case
		"COMPOSE_PROJECT_NAME":           strings.ToLower("ddev-" + strings.Replace(app.Name, `.`, "", -1)),
		"COMPOSE_CONVERT_WINDOWS_PATHS":  "true",
		"COMPOSER_EXIT_ON_PATCH_FAILURE": "1",
		"DDEV_SITENAME":                  app.Name,
		"DDEV_TLD":                       app.ProjectTLD,
		"DDEV_DBIMAGE":                   app.GetDBImage(),
		"DDEV_PROJECT":                   app.Name,
		"DDEV_WEBIMAGE":                  app.WebImage,
		"DDEV_APPROOT":                   app.AppRoot,
		"DDEV_COMPOSER_ROOT":             app.GetComposerRoot(true, false),
		"DDEV_DATABASE_FAMILY":           dbFamily,
		"DDEV_DATABASE":                  app.Database.Type + ":" + app.Database.Version,
		"DDEV_FILES_DIR":                 app.getContainerUploadDir(),
		"DDEV_FILES_DIRS":                strings.Join(app.getContainerUploadDirs(), ","),

		"DDEV_HOST_DB_PORT":        dbPortStr,
		"DDEV_HOST_MAILHOG_PORT":   app.HostMailhogPort,
		"DDEV_HOST_HTTPS_PORT":     hostHTTPSPortStr,
		"DDEV_HOST_WEBSERVER_PORT": hostHTTPPortStr,
		"DDEV_MAILHOG_PORT":        app.MailhogPort,
		"DDEV_MAILHOG_HTTPS_PORT":  app.MailhogHTTPSPort,
		"DDEV_DOCROOT":             app.Docroot,
		"DDEV_HOSTNAME":            app.HostName(),
		"DDEV_UID":                 uidStr,
		"DDEV_GID":                 gidStr,
		"DDEV_MUTAGEN_ENABLED":     strconv.FormatBool(app.IsMutagenEnabled()),
		"DDEV_PHP_VERSION":         app.PHPVersion,
		"DDEV_WEBSERVER_TYPE":      app.WebserverType,
		"DDEV_PROJECT_TYPE":        app.Type,
		"DDEV_ROUTER_HTTP_PORT":    app.GetRouterHTTPPort(),
		"DDEV_ROUTER_HTTPS_PORT":   app.GetRouterHTTPSPort(),
		"DDEV_XDEBUG_ENABLED":      strconv.FormatBool(app.XdebugEnabled),
		"DDEV_PRIMARY_URL":         app.GetPrimaryURL(),
		"DDEV_VERSION":             versionconstants.DdevVersion,
		"DOCKER_SCAN_SUGGEST":      "false",
		"GOOS":                     runtime.GOOS,
		"GOARCH":                   runtime.GOARCH,
		"IS_DDEV_PROJECT":          "true",
		"IS_GITPOD":                strconv.FormatBool(nodeps.IsGitpod()),
		"IS_CODESPACES":            strconv.FormatBool(nodeps.IsCodespaces()),
		"IS_WSL2":                  isWSL2,
	}

	// Set the DDEV_DB_CONTAINER_COMMAND command to empty to prevent docker-compose from complaining normally.
	// It's used for special startup on restoring to a snapshot or for postgres.
	if len(os.Getenv("DDEV_DB_CONTAINER_COMMAND")) == 0 {
		v := ""
		if app.Database.Type == nodeps.Postgres { // config_file spec for postgres
			v = fmt.Sprintf("-c config_file=%s/postgresql.conf -c hba_file=%s/pg_hba.conf", nodeps.PostgresConfigDir, nodeps.PostgresConfigDir)
		}
		envVars["DDEV_DB_CONTAINER_COMMAND"] = v
	}

	// Find out terminal dimensions
	columns, lines := nodeps.GetTerminalWidthHeight()

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

	status, _ := app.SiteStatus()
	if status == SiteStopped {
		return nil
	}

	err := app.ProcessHooks("pre-pause")
	if err != nil {
		return err
	}

	_ = SyncAndPauseMutagenSession(app)

	if _, _, err := dockerutil.ComposeCmd([]string{app.DockerComposeFullRenderedYAMLPath()}, "stop"); err != nil {
		return err
	}
	err = app.ProcessHooks("post-pause")
	if err != nil {
		return err
	}

	return StopRouterIfNoContainers()
}

// WaitForServices waits for all the services in docker-compose to come up
func (app *DdevApp) WaitForServices() error {
	var requiredContainers []string
	if services, ok := app.ComposeYaml["services"].(map[string]interface{}); ok {
		for k := range services {
			requiredContainers = append(requiredContainers, k)
		}
	} else {
		util.Failed("unable to get required startup services to wait for")
	}
	output.UserOut.Printf("Waiting for these services to become ready: %v", requiredContainers)

	labels := map[string]string{
		"com.ddev.site-name": app.GetName(),
	}
	waitTime := app.FindMaxTimeout()
	_, err := dockerutil.ContainerWait(waitTime, labels)
	if err != nil {
		return fmt.Errorf("timed out waiting for containers (%v) to start: err=%v", requiredContainers, err)
	}
	return nil
}

// Wait ensures that the app service containers are healthy.
func (app *DdevApp) Wait(requiredContainers []string) error {
	for _, containerType := range requiredContainers {
		labels := map[string]string{
			"com.ddev.site-name":         app.GetName(),
			"com.docker.compose.service": containerType,
		}
		waitTime := app.FindMaxTimeout()
		logOutput, err := dockerutil.ContainerWait(waitTime, labels)
		if err != nil {
			return fmt.Errorf("%s container failed: log=%s, err=%v", containerType, logOutput, err)
		}
	}

	return nil
}

// WaitByLabels waits for containers found by list of labels to be
// ready
func (app *DdevApp) WaitByLabels(labels map[string]string) error {
	waitTime := app.FindMaxTimeout()
	err := dockerutil.ContainersWait(waitTime, labels)
	if err != nil {
		return fmt.Errorf("container(s) failed to become healthy before their configured timeout or in %d seconds. This may be just a problem with the healthcheck and not a functional problem. (%v)", waitTime, err)
	}
	return nil
}

// StartAndWait is primarily for use in tests.
// It does app.Start() but then waits for extra seconds
// before returning.
// extraSleep arg in seconds is the time to wait if > 0
func (app *DdevApp) StartAndWait(extraSleep int) error {
	err := app.Start()
	if err != nil {
		return err
	}
	if extraSleep > 0 {
		time.Sleep(time.Duration(extraSleep) * time.Second)
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
		signatureFound, err := fileutil.FgrepStringInFile(loc, nodeps.DdevFileSignature)
		util.CheckErr(err) // Really can't happen as we already checked for the file existence

		// If the signature was found, it's safe to use.
		if signatureFound {
			return loc, nil
		}
	}

	return "", fmt.Errorf("settings files already exist and are being managed by the user")
}

// Snapshot causes a snapshot of the db to be written into the snapshots volume
// Returns the name of the snapshot and err
func (app *DdevApp) Snapshot(snapshotName string) (string, error) {
	containerSnapshotDirBase := "/var/tmp"

	err := app.ProcessHooks("pre-snapshot")
	if err != nil {
		return "", fmt.Errorf("failed to process pre-stop hooks: %v", err)
	}

	if snapshotName == "" {
		t := time.Now()
		snapshotName = app.Name + "_" + t.Format("20060102150405")
	}

	snapshotFile := snapshotName + "-" + app.Database.Type + "_" + app.Database.Version + ".gz"

	existingSnapshots, err := app.ListSnapshots()
	if err != nil {
		return "", err
	}
	if nodeps.ArrayContainsString(existingSnapshots, snapshotName) {
		return "", fmt.Errorf("snapshot %s already exists, please use another snapshot name or clean up snapshots with `ddev snapshot --cleanup`", snapshotFile)
	}

	// Container side has to use path.Join instead of filepath.Join because they are
	// targeted at the linux filesystem, so won't work with filepath on Windows
	containerSnapshotDir := containerSnapshotDirBase

	// Ensure that db container is up.
	err = app.Wait([]string{"db"})
	if err != nil {
		return "", fmt.Errorf("unable to snapshot database, \nyour db container in project %v is not running. \nPlease start the project if you want to snapshot it. \nIf deleting project, you can delete without a snapshot using \n'ddev delete --omit-snapshot --yes', \nwhich will destroy your database", app.Name)
	}

	// For versions less than 8.0.32, we have to OPTIMIZE TABLES to make xtrabackup work
	// See https://docs.percona.com/percona-xtrabackup/8.0/em/instant.html and
	// https://www.percona.com/blog/percona-xtrabackup-8-0-29-and-instant-add-drop-columns/
	if app.Database.Type == "mysql" && app.Database.Version == nodeps.MySQL80 {
		stdout, stderr, err := app.Exec(&ExecOpts{
			Service: "db",
			Cmd:     `set -eu -o pipefail; MYSQL_PWD=root mysql -e 'SET SQL_NOTES=0'; mysql -N -uroot -e 'SELECT NAME FROM INFORMATION_SCHEMA.INNODB_TABLES WHERE TOTAL_ROW_VERSIONS > 0;'`,
		})
		if err != nil {
			util.Warning("could not check for tables to optimize (mysql 8.0): %v (stdout='%s', stderr='%s')", err, stdout, stderr)
		} else {
			stdout = strings.Trim(stdout, "\n\t ")
			tables := strings.Split(stdout, "\n")
			// util.Success("tables=%v len(tables)=%d stdout was '%s'", tables, len(tables), stdout)
			if len(stdout) > 0 && len(tables) > 0 {
				for _, t := range tables {
					r := strings.Split(t, `/`)
					if len(r) != 2 {
						util.Warning("unable to get database/table from %s", r)
						continue
					}
					d := r[0]
					t := r[1]
					stdout, stderr, err := app.Exec(&ExecOpts{
						Service: "db",
						Cmd:     fmt.Sprintf(`set -eu -o pipefail; MYSQL_PWD=root mysql -uroot -D %s -e 'OPTIMIZE TABLES %s';`, d, t),
					})
					if err != nil {
						util.Warning("unable to optimize table %s (mysql 8.0): %v (stdout='%s', stderr='%s')", t, err, stdout, stderr)
					}
				}
				util.Success("Optimized mysql 8.0 tables '%s' in preparation for snapshot", strings.Join(tables, `,'`))
			}
		}
	}
	util.Success("Creating database snapshot %s", snapshotName)

	c := getBackupCommand(app, path.Join(containerSnapshotDir, snapshotFile))
	stdout, stderr, err := app.Exec(&ExecOpts{
		Service: "db",
		Cmd:     fmt.Sprintf(`set -eu -o pipefail; %s `, c),
	})

	if err != nil {
		util.Warning("Failed to create snapshot: %v, stdout=%s, stderr=%s", err, stdout, stderr)
		return "", err
	}

	dbContainer, err := GetContainer(app, "db")
	if err != nil {
		return "", err
	}

	if globalconfig.DdevGlobalConfig.NoBindMounts {
		// If we're not using bind-mounts, we have to copy the snapshot back into
		// the host project's .ddev/db_snapshots directory
		defer util.TimeTrackC("CopySnapshotFromContainer")()
		// Copy snapshot back to the host
		err = dockerutil.CopyFromContainer(GetContainerName(app, "db"), path.Join(containerSnapshotDir, snapshotFile), app.GetConfigPath("db_snapshots"))
		if err != nil {
			return "", err
		}
	} else {
		// But if we are using bind-mounts, we can just copy it to where the snapshot is
		// mounted into the db container (/mnt/ddev_config/db_snapshots)
		c := fmt.Sprintf("cp -r %s/%s /mnt/ddev_config/db_snapshots", containerSnapshotDir, snapshotFile)
		uid, _, _ := util.GetContainerUIDGid()
		if app.Database.Type == nodeps.Postgres {
			uid = "999"
		}
		stdout, stderr, err = dockerutil.Exec(dbContainer.ID, c, uid)
		if err != nil {
			return "", fmt.Errorf("failed to '%s': %v, stdout=%s, stderr=%s", c, err, stdout, stderr)
		}
	}

	// Clean up the in-container dir that we just used
	_, _, err = dockerutil.Exec(dbContainer.ID, fmt.Sprintf("rm -f %s/%s", containerSnapshotDir, snapshotFile), "")
	if err != nil {
		return "", err
	}
	err = app.ProcessHooks("post-snapshot")
	if err != nil {
		return snapshotFile, fmt.Errorf("failed to process pre-stop hooks: %v", err)
	}

	return snapshotName, nil
}

// getBackupCommand returns the command to dump the entire db system for the various databases
func getBackupCommand(app *DdevApp, targetFile string) string {

	c := fmt.Sprintf(`mariabackup --backup --stream=mbstream --user=root --password=root --socket=/var/tmp/mysql.sock  2>/tmp/snapshot_%s.log | gzip > "%s"`, path.Base(targetFile), targetFile)

	oldMariaVersions := []string{"5.5", "10.0"}

	switch {
	// Old mariadb versions don't have mariabackup, use xtrabackup for them as well as MySQL
	case app.Database.Type == nodeps.MariaDB && nodeps.ArrayContainsString(oldMariaVersions, app.Database.Version):
		fallthrough
	case app.Database.Type == nodeps.MySQL:
		c = fmt.Sprintf(`xtrabackup --backup --stream=xbstream --user=root --password=root --socket=/var/tmp/mysql.sock  2>/tmp/snapshot_%s.log | gzip > "%s"`, path.Base(targetFile), targetFile)
	case app.Database.Type == nodeps.Postgres:
		c = fmt.Sprintf("rm -rf /var/tmp/pgbackup && pg_basebackup -D /var/tmp/pgbackup 2>/tmp/snapshot_%s.log && tar -czf %s -C /var/tmp/pgbackup/ .", path.Base(targetFile), targetFile)
	}
	return c
}

// fullDBFromVersion takes just a mariadb or mysql version number
// in x.xx format and returns something like mariadb-10.5
func fullDBFromVersion(v string) string {
	snapshotDBVersion := ""
	// The old way (when we only had mariadb and then when had mariadb and also mysql)
	// was to just have the version number and derive the database type from it,
	// so that's what is going on here. But we create a string like "mariadb_10.3" from
	// the version number
	switch {
	case v == "5.6" || v == "5.7" || v == "8.0":
		snapshotDBVersion = "mysql_" + v

	// 5.5 isn't actually necessarily correct, because could be
	// mysql 5.5. But maria and mysql 5.5 databases were compatible anyway.
	case v == "5.5" || v >= "10.0":
		snapshotDBVersion = "mariadb_" + v
	}
	return snapshotDBVersion
}

// Stop stops and Removes the docker containers for the project in current directory.
func (app *DdevApp) Stop(removeData bool, createSnapshot bool) error {
	app.DockerEnv()
	var err error

	if app.Name == "" {
		return fmt.Errorf("invalid app.Name provided to app.Stop(), app=%v", app)
	}

	status, _ := app.SiteStatus()
	if status != SiteStopped {
		err = app.ProcessHooks("pre-stop")
		if err != nil {
			return fmt.Errorf("failed to process pre-stop hooks: %v", err)
		}
	}

	if createSnapshot {
		if status != SiteRunning {
			util.Warning("Must start non-running project to do database snapshot")
			err = app.Start()
			if err != nil {
				return fmt.Errorf("failed to start project to perform database snapshot")
			}
		}
		t := time.Now()
		_, err = app.Snapshot(app.Name + "_remove_data_snapshot_" + t.Format("20060102150405"))
		if err != nil {
			return err
		}
	}

	err = SyncAndPauseMutagenSession(app)
	if err != nil {
		util.Warning("Unable to SyncAndterminateMutagenSession: %v", err)
	}

	if globalconfig.DdevGlobalConfig.IsTraefikRouter() && status == SiteRunning {
		_, _, err = app.Exec(&ExecOpts{
			Cmd: fmt.Sprintf("rm -f /mnt/ddev-global-cache/traefik/*/%s.{yaml,crt,key}", app.Name),
		})
		if err != nil {
			util.Warning("Unable to clean up traefik configuration: %v", err)
		}
	}
	// Clean up ddev-global-cache
	if removeData {
		c := fmt.Sprintf("rm -rf /mnt/ddev-global-cache/*/%s-{web,db} /mnt/ddev-global-cache/traefik/*/%s.{yaml,crt,key}", app.Name, app.Name)
		util.Debug("Cleaning ddev-global-cache with command '%s'", c)
		_, out, err := dockerutil.RunSimpleContainer(dockerImages.GetWebImage(), "clean-ddev-global-cache-"+util.RandString(6), []string{"bash", "-c", c}, []string{}, []string{}, []string{"ddev-global-cache:/mnt/ddev-global-cache"}, "", true, false, map[string]string{`com.ddev.site-name`: app.GetName()}, nil)
		if err != nil {
			util.Warning("Unable to clean up ddev-global-cache with command '%s': %v; output='%s'", c, err, out)
		}
	}

	if status == SiteRunning {
		err = app.Pause()
		if err != nil {
			util.Warning("Failed to pause containers for %s: %v", app.GetName(), err)
		}
	}
	// Remove all the containers and volumes for app.
	err = Cleanup(app)
	if err != nil {
		return err
	}

	// Remove data/database/projectInfo/hosts entry if we need to.
	if removeData {
		err = TerminateMutagenSync(app)
		if err != nil {
			util.Warning("unable to terminate mutagen session %s: %v", MutagenSyncName(app.Name), err)
		}

		// Remove .ddev/settings if it exists
		if fileutil.FileExists(app.GetConfigPath("settings")) {
			err = os.RemoveAll(app.GetConfigPath("settings"))
			if err != nil {
				util.Warning("Unable to remove %s: %v", app.GetConfigPath("settings"), err)
			}
		}

		if err = app.RemoveHostsEntriesIfNeeded(); err != nil {
			return fmt.Errorf("failed to remove hosts entries: %v", err)
		}
		app.RemoveGlobalProjectInfo()
		err = globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig)
		if err != nil {
			util.Warning("could not WriteGlobalConfig: %v", err)
		}

		vols := []string{app.GetMariaDBVolumeName(), app.GetPostgresVolumeName(), GetMutagenVolumeName(app)}
		if globalconfig.DdevGlobalConfig.NoBindMounts {
			vols = append(vols, app.Name+"-ddev-config")
		}
		for _, volName := range vols {
			err = dockerutil.RemoveVolume(volName)
			if err != nil {
				util.Warning("could not remove volume %s: %v", volName, err)
			} else {
				util.Success("Volume %s for project %s was deleted", volName, app.Name)
			}
		}
		deleteServiceVolumes(app)

		dbBuilt := app.GetDBImage() + "-" + app.Name + "-built"
		_ = dockerutil.RemoveImage(dbBuilt)

		webBuilt := dockerImages.GetWebImage() + "-" + app.Name + "-built"
		_ = dockerutil.RemoveImage(webBuilt)
		util.Success("Project %s was deleted. Your code and configuration are unchanged.", app.Name)
	}

	if status != SiteStopped {
		err = app.ProcessHooks("post-stop")
		if err != nil {
			return fmt.Errorf("failed to process post-stop hooks: %v", err)
		}
	}

	return nil
}

// deleteServiceVolumes finds all the volumes created by services and removes them.
// All volumes that are not external (likely not global) are removed.
func deleteServiceVolumes(app *DdevApp) {
	var err error
	y := app.ComposeYaml
	if s, ok := y["volumes"]; ok {
		for _, v := range s.(map[string]interface{}) {
			vol := v.(map[string]interface{})
			if vol["external"] == true {
				continue
			}
			if vol["name"] == nil {
				continue
			}
			volName := vol["name"].(string)

			if dockerutil.VolumeExists(volName) {
				err = dockerutil.RemoveVolume(volName)
				if err != nil {
					util.Warning("could not remove volume %s: %v", volName, err)
				} else {
					util.Success("Deleting third-party persistent volume %s", volName)
				}
			}
		}
	}
}

// RemoveGlobalProjectInfo deletes the project from ProjectList
func (app *DdevApp) RemoveGlobalProjectInfo() {
	_ = globalconfig.RemoveProjectInfo(app.Name)
}

// GetHTTPURL returns the HTTP URL for an app.
func (app *DdevApp) GetHTTPURL() string {
	url := ""
	if !IsRouterDisabled(app) {
		url = "http://" + app.GetHostname()
		if app.GetRouterHTTPPort() != "80" {
			url = url + ":" + app.GetRouterHTTPPort()
		}
	} else {
		url = app.GetWebContainerDirectHTTPURL()
	}
	return url
}

// GetHTTPSURL returns the HTTPS URL for an app.
func (app *DdevApp) GetHTTPSURL() string {
	url := ""
	if !IsRouterDisabled(app) {
		url = "https://" + app.GetHostname()
		p := app.GetRouterHTTPSPort()
		if p != "443" {
			url = url + ":" + p
		}
	} else {
		url = app.GetWebContainerDirectHTTPSURL()
	}
	return url
}

// GetAllURLs returns an array of all the URLs for the project
func (app *DdevApp) GetAllURLs() (httpURLs []string, httpsURLs []string, allURLs []string) {
	if nodeps.IsGitpod() {
		url, err := exec.RunHostCommand("gp", "url", app.HostWebserverPort)
		if err == nil {
			url = strings.Trim(url, "\n")
			httpsURLs = append(httpsURLs, url)
		}
	}
	if nodeps.IsCodespaces() {
		codespaceName := os.Getenv("CODESPACE_NAME")
		if codespaceName != "" {
			url := fmt.Sprintf("https://%s-%s.preview.app.github.dev", codespaceName, app.HostWebserverPort)
			httpsURLs = append(httpsURLs, url)
		}
	}

	// Get configured URLs
	for _, name := range app.GetHostnames() {
		httpPort := ""
		httpsPort := ""
		if app.GetRouterHTTPPort() != "80" {
			httpPort = ":" + app.GetRouterHTTPPort()
		}
		if app.GetRouterHTTPSPort() != "443" {
			httpsPort = ":" + app.GetRouterHTTPSPort()
		}

		httpsURLs = append(httpsURLs, "https://"+name+httpsPort)
		httpURLs = append(httpURLs, "http://"+name+httpPort)
	}

	if !IsRouterDisabled(app) {
		httpsURLs = append(httpsURLs, app.GetWebContainerDirectHTTPSURL())
	}
	httpURLs = append(httpURLs, app.GetWebContainerDirectHTTPURL())

	allURLs = append(httpsURLs, httpURLs...)
	return httpURLs, httpsURLs, allURLs
}

// GetPrimaryURL returns the primary URL that can be used, https or http
func (app *DdevApp) GetPrimaryURL() string {
	httpURLs, httpsURLs, _ := app.GetAllURLs()
	urlList := httpsURLs
	// If no mkcert trusted https, use the httpURLs instead
	if !nodeps.IsGitpod() && !nodeps.IsCodespaces() && (globalconfig.GetCAROOT() == "" || IsRouterDisabled(app)) {
		urlList = httpURLs
	}
	if len(urlList) > 0 {
		return urlList[0]
	}
	// Failure mode, just return empty string
	return ""
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
		return -1, fmt.Errorf("unable to find web container for app: %s, err %v", app.Name, err)
	}

	for _, p := range webContainer.Ports {
		if p.PrivatePort == 80 {
			return int(p.PublicPort), nil
		}
	}
	return -1, fmt.Errorf("no public port found for private port 80")
}

// GetWebContainerHTTPSPublicPort returns the direct-access public tcp port for https
func (app *DdevApp) GetWebContainerHTTPSPublicPort() (int, error) {

	webContainer, err := app.FindContainerByType("web")
	if err != nil || webContainer == nil {
		return -1, fmt.Errorf("unable to find https web container for app: %s, err %v", app.Name, err)
	}

	for _, p := range webContainer.Ports {
		if p.PrivatePort == 443 {
			return int(p.PublicPort), nil
		}
	}
	return -1, fmt.Errorf("no public https port found for private port 443")
}

// HostName returns the hostname of a given application.
func (app *DdevApp) HostName() string {
	return app.GetHostname()
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
			return "", fmt.Errorf("could not find a project in %s. Have you run 'ddev config'? Please specify a project name or change directories: %s", siteDir, err)
		}
		// Handle the case where it's registered globally but stopped
	} else if p := globalconfig.GetProject(siteName); p != nil {
		return p.AppRoot, nil
		// Or find it by looking at docker containers
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
func (app *DdevApp) GetProvider(providerName string) (*Provider, error) {

	var p Provider
	var err error

	if providerName != "" && providerName != nodeps.ProviderDefault {
		p = Provider{
			ProviderType: providerName,
			app:          app,
		}
		err = p.Init(providerName, app)
	}

	app.ProviderInstance = &p
	return app.ProviderInstance, err
}

// GetWorkingDir will determine the appropriate working directory for an Exec/ExecWithTty command
// by consulting with the project configuration. If no dir is specified for the service, an
// empty string will be returned.
func (app *DdevApp) GetWorkingDir(service string, dir string) string {
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

// GetNFSMountVolumeName returns the docker volume name of the nfs mount volume
func (app *DdevApp) GetNFSMountVolumeName() string {
	// This is lowercased because the automatic naming in docker-compose v1/2
	// defaulted to lowercase the name
	// Although some volume names are auto-lowercased by docker, this one
	// is explicitly specified by us and is not lowercased.
	return "ddev-" + app.Name + "_nfsmount"
}

// GetMariaDBVolumeName returns the docker volume name of the mariadb/database volume
// For historical reasons this isn't lowercased.
func (app *DdevApp) GetMariaDBVolumeName() string {
	return app.Name + "-mariadb"
}

// GetPostgresVolumeName returns the docker volume name of the Postgres/database volume
// For historical reasons this isn't lowercased.
func (app *DdevApp) GetPostgresVolumeName() string {
	return app.Name + "-postgres"
}

// StartAppIfNotRunning is intended to replace much-duplicated code in the commands.
func (app *DdevApp) StartAppIfNotRunning() error {
	var err error
	status, _ := app.SiteStatus()
	if status != SiteRunning {
		err = app.Start()
	}

	return err
}

// CheckAddonIncompatibilities looks for problems with docker-compose.*.yaml 3rd-party services
func (app *DdevApp) CheckAddonIncompatibilities() error {
	if _, ok := app.ComposeYaml["services"]; !ok {
		util.Warning("Unable to check 3rd-party services for missing networks stanza")
		return nil
	}
	// Look for missing "networks" stanza and request it.
	for s, v := range app.ComposeYaml["services"].(map[string]interface{}) {
		errMsg := fmt.Errorf("service '%s' does not have the 'networks: [default, ddev_default]' stanza, required since v1.19, please add it, see %s", s, "https://ddev.readthedocs.io/en/latest/users/extend/custom-compose-files/#docker-composeyaml-examples")
		var nets map[string]interface{}
		x := v.(map[string]interface{})
		ok := false
		if nets, ok = x["networks"].(map[string]interface{}); !ok {
			return errMsg
		}
		// Make sure both "default" and "ddev" networks are in there.
		for _, requiredNetwork := range []string{"default", "ddev_default"} {
			if _, ok := nets[requiredNetwork]; !ok {
				return errMsg
			}
		}
	}
	return nil
}

// UpdateComposeYaml updates app.ComposeYaml from available content
func (app *DdevApp) UpdateComposeYaml(content string) error {
	err := yaml.Unmarshal([]byte(content), &app.ComposeYaml)
	if err != nil {
		return err
	}
	return nil
}

// GetContainerName returns the contructed container name of the
// service provided.
func GetContainerName(app *DdevApp, service string) string {
	return "ddev-" + app.Name + "-" + service
}

// GetContainer returns the container struct of the app service name provided.
func GetContainer(app *DdevApp, service string) (*docker.APIContainers, error) {
	name := GetContainerName(app, service)
	container, err := dockerutil.FindContainerByName(name)
	if err != nil || container == nil {
		return nil, fmt.Errorf("unable to find container %s: %v", name, err)
	}
	return container, nil
}

// FormatSiteStatus formats "paused" or "running" with color
func FormatSiteStatus(status string) string {
	if status == SiteRunning {
		status = "OK"
	}
	formattedStatus := status

	switch {
	case strings.Contains(status, SitePaused):
		formattedStatus = util.ColorizeText(formattedStatus, "yellow")
	case strings.Contains(status, SiteStopped) || strings.Contains(status, SiteDirMissing) || strings.Contains(status, SiteConfigMissing):
		formattedStatus = util.ColorizeText(formattedStatus, "red")
	default:
		formattedStatus = util.ColorizeText(formattedStatus, "green")
	}
	return formattedStatus
}

// genericImportFilesAction defines the workflow for importing project files.
func genericImportFilesAction(app *DdevApp, uploadDir, importPath, extPath string) error {
	destPath := app.calculateHostUploadDirFullPath(uploadDir)

	// parent of destination dir should exist
	if !fileutil.FileExists(filepath.Dir(destPath)) {
		return fmt.Errorf("unable to import to %s: parent directory does not exist", destPath)
	}

	// parent of destination dir should be writable.
	if err := os.Chmod(filepath.Dir(destPath), 0755); err != nil {
		return err
	}

	// If the destination path exists, remove it as was warned
	if fileutil.FileExists(destPath) {
		if err := os.RemoveAll(destPath); err != nil {
			return fmt.Errorf("failed to cleanup %s before import: %v", destPath, err)
		}
	}

	if isTar(importPath) {
		if err := archive.Untar(importPath, destPath, extPath); err != nil {
			return fmt.Errorf("failed to extract provided archive: %v", err)
		}

		return nil
	}

	if isZip(importPath) {
		if err := archive.Unzip(importPath, destPath, extPath); err != nil {
			return fmt.Errorf("failed to extract provided archive: %v", err)
		}

		return nil
	}

	//nolint: revive
	if err := fileutil.CopyDir(importPath, destPath); err != nil {
		return err
	}

	return nil
}
