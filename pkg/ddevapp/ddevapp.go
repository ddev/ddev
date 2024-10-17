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
	ddevImages "github.com/ddev/ddev/pkg/docker"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/ddev/ddev/pkg/versionconstants"
	dockerTypes "github.com/docker/docker/api/types"
	dockerContainer "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/stdcopy"
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

// DdevApp is the struct that represents a DDEV app, mostly its config
// from config.yaml.
type DdevApp struct {
	Name                      string                 `yaml:"name"`
	Type                      string                 `yaml:"type"`
	AppRoot                   string                 `yaml:"-"`
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
	MailpitHTTPPort           string                 `yaml:"mailpit_http_port,omitempty"`
	MailpitHTTPSPort          string                 `yaml:"mailpit_https_port,omitempty"`
	HostMailpitPort           string                 `yaml:"host_mailpit_port,omitempty"`
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
	NodeJSVersion             string                 `yaml:"nodejs_version,omitempty"`
	CorepackEnable            bool                   `yaml:"corepack_enable"`
	DefaultContainerTimeout   string                 `yaml:"default_container_timeout,omitempty"`
	WebExtraExposedPorts      []WebExposedPort       `yaml:"web_extra_exposed_ports,omitempty"`
	WebExtraDaemons           []WebExtraDaemon       `yaml:"web_extra_daemons,omitempty"`
	OverrideConfig            bool                   `yaml:"override_config,omitempty"`
	DisableUploadDirsWarning  bool                   `yaml:"disable_upload_dirs_warning,omitempty"`
	DdevVersionConstraint     string                 `yaml:"ddev_version_constraint,omitempty"`
	ExtraLocales              []string               `yaml:"extra_locales,omitempty"`
	ComposeYaml               map[string]interface{} `yaml:"-"`
}

// Global variable that's set from --skip-hooks global flag.
// If true, all hooks would be skiped.
var SkipHooks = false

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
	// Init() is putting together the DdevApp struct, the containers do
	// not have to exist (app doesn't have to have been started), so the fact
	// we didn't find any is not an error.
	return nil
}

// FindContainerByType will find a container for this site denoted by the containerType if it is available.
func (app *DdevApp) FindContainerByType(containerType string) (*dockerTypes.Container, error) {
	labels := map[string]string{
		"com.ddev.site-name":         app.GetName(),
		"com.docker.compose.service": containerType,
	}

	return dockerutil.FindContainerByLabels(labels)
}

// Describe returns a map which provides detailed information on services associated with the running site.
// if short==true, then only the basic information is returned.
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
	appDesc["mailpit_https_url"] = "https://" + app.GetHostname() + ":" + app.GetMailpitHTTPSPort()
	appDesc["mailpit_url"] = "http://" + app.GetHostname() + ":" + app.GetMailpitHTTPPort()
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

	// If short is set, we don't need more information, so return what we have.
	if short {
		return appDesc, nil
	}
	appDesc["hostname"] = app.GetHostname()
	appDesc["hostnames"] = app.GetHostnames()
	appDesc["performance_mode"] = app.GetPerformanceMode()
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
			if err != nil {
				util.Warning("failed to GetPublishedPort(db): %v", err)
			}
			dbinfo["dbPort"] = GetExposedPort(app, "db")
			dbinfo["published_port"] = dbPublicPort
			dbinfo["database_type"] = nodeps.MariaDB // default
			dbinfo["database_type"] = app.Database.Type
			dbinfo["database_version"] = app.Database.Version

			appDesc["dbinfo"] = dbinfo
		}
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
		if err != nil {
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
				hostPorts = append(hostPorts, strconv.FormatInt(int64(pv.PublicPort), 10))
			}
		}
		services[shortName]["host_ports"] = strings.Join(hostPorts, ",")

		// Extract VIRTUAL_HOST, HTTP_EXPOSE and HTTPS_EXPOSE for additional info
		if !IsRouterDisabled(app) {
			envMap := make(map[string]string)

			for _, e := range c.Config.Env {
				split := strings.SplitN(e, "=", 2)
				envName := split[0]

				// Store the values first, so we can have them all before assigning
				if len(split) == 2 && (envName == "VIRTUAL_HOST" || envName == "HTTP_EXPOSE" || envName == "HTTPS_EXPOSE") {
					envMap[envName] = split[1]
				}
			}

			if virtualHost, ok := envMap["VIRTUAL_HOST"]; ok {
				vhostVal := virtualHost
				vhostValStr := fmt.Sprintf("%s", vhostVal)
				vhostsList := strings.Split(vhostValStr, ",")

				// There might be more than one VIRTUAL_HOST value, but this only handles the first listed,
				// most often there's only one.
				if len(vhostsList) > 0 {
					// VIRTUAL_HOSTS typically look like subdomain.domain.tld, for example - VIRTUAL_HOSTS=vhost1.myproject.ddev.site,vhost2.myproject.ddev.site
					vhost := strings.Split(vhostsList[0], ",")
					services[shortName]["virtual_host"] = vhost[0]
				}
			}

			hostname, ok := services[shortName]["virtual_host"]
			appHostname := ""

			if ok {
				appHostname = hostname
			} else {
				appHostname = appDesc["hostname"].(string)
			}

			for name, portMapping := range envMap {
				if name != "HTTP_EXPOSE" && name != "HTTPS_EXPOSE" {
					continue
				}

				// If the HTTP port is 80 (default), it doesn't get included in URL
				portDefault := "80"
				attributeName := "http_url"
				protocol := "http://"

				if name == "HTTPS_EXPOSE" {
					// If the HTTPS port is 443 (default), it doesn't get included in URL
					portDefault = "443"
					attributeName = "https_url"
					protocol = "https://"
				}

				portValStr := fmt.Sprintf("%s", portMapping)
				portSpecs := strings.Split(portValStr, ",")
				// There might be more than one exposed UI port, but this only handles the first listed,
				// most often there's only one.
				if len(portSpecs) > 0 {
					// HTTP(S) portSpecs typically look like <exposed>:<containerPort>, for example - HTTP_EXPOSE=1359:1358
					ports := strings.Split(portSpecs[0], ":")

					services[shortName][attributeName] = protocol + appHostname

					if ports[0] != portDefault {
						services[shortName][attributeName] = services[shortName][attributeName] + ":" + ports[0]
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
	return app.GetPublishedPortForPrivatePort(serviceName, uint16(exposedPortInt))
}

// GetPublishedPortForPrivatePort returns the host-exposed public port of a container for a given private port.
func (app *DdevApp) GetPublishedPortForPrivatePort(serviceName string, privatePort uint16) (publicPort int, err error) {
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

// GetDocroot returns the docroot path for DDEV app
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

// GetComposerRoot will determine the absolute Composer root directory where
// all Composer related commands will be executed.
// If inContainer set to true, the absolute path in the container will be
// returned, else the absolute path on the host.
// If showWarning set to true, a warning containing the Composer root will be
// shown to the user to avoid confusion.
func (app *DdevApp) GetComposerRoot(inContainer, showWarning bool) string {
	var absComposerRoot string

	if inContainer {
		absComposerRoot = path.Join(app.GetAbsAppRoot(true), app.ComposerRoot)
	} else {
		absComposerRoot = filepath.Join(app.GetAbsAppRoot(false), app.ComposerRoot)
	}

	// If requested, let the user know we are not using the default Composer
	// root directory to avoid confusion.
	if app.ComposerRoot != "" && showWarning {
		util.Warning("Using '%s' as Composer root directory", absComposerRoot)
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
func (app *DdevApp) GetRouterHTTPPort() string {

	// If the web container is running and HTTP_EXPOSE has a mapping,
	// return the host-side mapped port
	if httpExpose := app.GetWebEnvVar("HTTP_EXPOSE"); httpExpose != "" {
		//util.Debug("GetRouterHTTPPort(): HTTP_EXPOSE=%s", httpExpose)
		httpPort := app.PortFromExposeVariable(httpExpose, "80")
		if httpPort != "" {
			//util.Debug("GetRouterHTTPPort(): returning httpPort=%s found from HTTP_EXPOSE=%s", httpPort, httpExpose)
			return httpPort
		}
	}

	// If the project-level RouterHTTPPort is set, it takes priority
	// over the global RouterHTTPPort, so return that
	if app.RouterHTTPPort != "" {
		//util.Debug("GetRouterHTTPPort(): returning app.RouterHTTPPort=%s", app.RouterHTTPPort)
		return app.RouterHTTPPort
	}

	// Finally, return whatever is in the global RouterHTTPPort,
	// which will be port 80 by default, but could be something else
	// if configured there

	//util.Debug("GetRouterHTTPPort(): returning globalconfig.DdevGlobalConfig.RouterHTTPPort=%s", globalconfig.DdevGlobalConfig.RouterHTTPPort)
	return globalconfig.DdevGlobalConfig.RouterHTTPPort
}

// GetWebEnvVar() gets an environment variable from
// app.ComposeYaml["services"]["web"]["environment"]
// It returns empty string if there is no var or the ComposeYaml
// is just not set.
func (app *DdevApp) GetWebEnvVar(name string) string {
	if s, ok := app.ComposeYaml["services"].(map[string]interface{}); ok {
		if v, ok := s["web"].(map[string]interface{})["environment"].(map[string]interface{})[name]; ok {
			return v.(string)
		}
	}
	return ""
}

// PortFromExposeVariable() uses a string like HTTP_EXPOSE or HTTPS_EXPOSE, which is a
// comma-delimted list of colon-delimited port-pairs
// Given a target port (often "80" or "8025") its job is to get from HTTPS_EXPOSE or HTTP_EXPOSE
// the related port to be exposed on the router.
// It returns an empty string if the HTTP_EXPOSE/HTTPS_EXPOSE is not
// found or no valid port mapping is found.
func (app *DdevApp) PortFromExposeVariable(exposeEnvVar string, targetPort string) string {
	// Get the var
	// split it via comma
	// split it via colon into a map: rhs is the key, lhs is the value
	portMap := make(map[string]string)
	items := strings.Split(exposeEnvVar, ",")
	for _, item := range items {
		portPair := strings.Split(item, ":")
		if len(portPair) == 2 {
			portMap[portPair[1]] = portPair[0]
		}
	}
	if w, ok := portMap[targetPort]; ok {
		return w
	}
	return ""
}

// GetRouterHTTPSPort returns app's router https port
// It has to choose from (highest to lowest priority):
// 1. The actual port configured into running container via HTTPS_EXPOSE
// 2. The project router_http_port
// 3. The global router_http_port
func (app *DdevApp) GetRouterHTTPSPort() string {
	if httpsExpose := app.GetWebEnvVar("HTTPS_EXPOSE"); httpsExpose != "" {
		//util.Debug("GetRouterHTTPSPort(): HTTPS_EXPOSE='%s'", httpsExpose)
		httpsPort := app.PortFromExposeVariable(httpsExpose, "80")
		if httpsPort != "" {
			//util.Debug("GetRouterHTTPSPort(): returning httpsPort=%s derived from HTTPS_EXPOSE=%s", httpsPort, httpsExpose)
			return httpsPort
		}
	}

	if app.RouterHTTPSPort != "" {
		//util.Debug("GetRouterHTTPSPort(): app.RouterHTTPSPort=%s", app.RouterHTTPSPort)
		return app.RouterHTTPSPort
	}

	//util.Debug("GetRouterHTTPSPort(): returning globalconfig.DdevGlobalConfig.RouterHTTPSPort=%s", globalconfig.DdevGlobalConfig.RouterHTTPSPort)
	return globalconfig.DdevGlobalConfig.RouterHTTPSPort
}

// GetMailpitHTTPPort returns app's mailpit router http port
// If HTTP_EXPOSE has a mapping to port 8025 in the container, use that
// If not, use the global or project MailpitHTTPPort
func (app *DdevApp) GetMailpitHTTPPort() string {

	if httpExpose := app.GetWebEnvVar("HTTP_EXPOSE"); httpExpose != "" {
		httpPort := app.PortFromExposeVariable(httpExpose, "8025")
		if httpPort != "" {
			return httpPort
		}
	}

	port := globalconfig.DdevGlobalConfig.RouterMailpitHTTPPort
	if port == "" {
		port = nodeps.DdevDefaultMailpitHTTPPort
	}
	if app.MailpitHTTPPort != "" {
		port = app.MailpitHTTPPort
	}
	return port
}

// GetMailpitHTTPSPort returns app's mailpit router https port
// If HTTPS_EXPOSE has a mapping to port 8025 in the container, use that
// If not, use the global or project MailpitHTTPSPort
func (app *DdevApp) GetMailpitHTTPSPort() string {

	if httpsExpose := app.GetWebEnvVar("HTTPS_EXPOSE"); httpsExpose != "" {
		httpsPort := app.PortFromExposeVariable(httpsExpose, "8025")
		if httpsPort != "" {
			return httpsPort
		}
	}

	port := globalconfig.DdevGlobalConfig.RouterMailpitHTTPSPort
	if port == "" {
		port = nodeps.DdevDefaultMailpitHTTPSPort
	}

	if app.MailpitHTTPSPort != "" {
		port = app.MailpitHTTPSPort
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
	err = util.Chmod(dbPath, 0777)
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
				return fmt.Errorf("unable to validate import asset %s: %s", dumpFile, err)
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

	// Default insideContainerImportPath is the one mounted from .ddev directory
	insideContainerImportPath := path.Join("/mnt/ddev_config/", filepath.Base(dbPath))
	// But if we don't have bind mounts, we have to copy dump into the container
	if globalconfig.DdevGlobalConfig.NoBindMounts {
		dbContainerName := GetContainerName(app, "db")
		if err != nil {
			return err
		}
		uid, _, _ := util.GetContainerUIDGid()
		// for PostgreSQL, must be written with PostgreSQL user
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
	// The Perl manipulation removes statements like CREATE DATABASE and USE, which
	// throw off imports.
	// It also removes the new `/*!999999\- enable the sandbox mode */` introduced in
	// LTS versions of MariaDB 2024-05.
	// This is a scary manipulation, as it must not match actual content
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
		inContainerCommand = []string{"bash", "-c", fmt.Sprintf(`set -eu -o pipefail && mysql -uroot -proot -e "%s" %s && pv %s/*.*sql |  perl -p -e 's/^(\/\*.*999999.*enable the sandbox mode *|CREATE DATABASE \/\*|USE %s)[^;]*(;|\*\/)//' | mysql %s %s`, preImportSQL, nodeps.MySQLRemoveDeprecatedMessage, insideContainerImportPath, "`", targetDB, nodeps.MySQLRemoveDeprecatedMessage)}

		// Alternate case where we are reading from stdin
		if dumpFile == "" && extractPath == "" {
			inContainerCommand = []string{"bash", "-c", fmt.Sprintf(`set -eu -o pipefail && mysql -uroot -proot -e "%s" %s && perl -p -e 's/^(CREATE DATABASE \/\*|USE %s)[^;]*;//' | mysql %s %s`, preImportSQL, nodeps.MySQLRemoveDeprecatedMessage, "`", targetDB, nodeps.MySQLRemoveDeprecatedMessage)}
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

	_, err = app.CreateSettingsFile()
	if err != nil {
		util.Warning("A custom settings file exists for your application, so DDEV did not generate one.")
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
	if targetDB == "" {
		targetDB = "db"
	}

	exportCmd := "mysqldump " + targetDB + nodeps.MySQLRemoveDeprecatedMessage
	if app.Database.Type == "postgres" {
		exportCmd = "pg_dump -U db " + targetDB
	}

	if app.Database.Type == nodeps.MariaDB {
		// The `tail --lines=+2` is a workaround that removes the new mariadb directive added
		// 2024-05 in mariadb-dump. It removes the first line of the dump, which has
		// the offending /*!999999\- enable the sandbox mode */. See
		// https://mariadb.org/mariadb-dump-file-compatibility-change/
		// If not on a newer MariaDB version, this will remove the identification
		// line from the top of the dump.
		exportCmd = exportCmd + " | tail --lines=+2 "
	}

	if compressionType == "" {
		compressionType = "cat"
	}
	exportCmd = exportCmd + " | " + compressionType

	opts := &ExecOpts{
		Service:   "db",
		RawCmd:    []string{"bash", "-c", `set -eu -o pipefail; ` + exportCmd},
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
	if compressionType == "cat" {
		confMsg = confMsg + " in plain text format"
	} else {
		confMsg = fmt.Sprintf("%s in %s format", confMsg, compressionType)
	}

	_, err = fmt.Fprintf(os.Stderr, "%s.\n", confMsg)

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
// It has to put the .ddev/docker-compose.override.y*ml last
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

// EnvFiles returns a list of env files for a project.
// It has to put the .ddev/.env first
// It has to put the .ddev/.env.* second
// Env files ending with .example are ignored.
func (app *DdevApp) EnvFiles() ([]string, error) {
	origDir, _ := os.Getwd()
	defer func() {
		_ = os.Chdir(origDir)
	}()
	err := os.Chdir(app.AppConfDir())
	if err != nil {
		return nil, err
	}
	envFiles, err := filepath.Glob(".env.*")
	if err != nil {
		return []string{}, fmt.Errorf(".env.* in %s: err=%v", app.AppConfDir(), err)
	}

	var orderedEnvFiles []string

	webEnvFile := app.GetConfigPath(".env")
	if fileutil.FileExists(webEnvFile) {
		orderedEnvFiles = append(orderedEnvFiles, webEnvFile)
	}

	for _, file := range envFiles {
		// Skip .example files
		if strings.HasSuffix(file, ".example") {
			continue
		}
		orderedEnvFiles = append(orderedEnvFiles, app.GetConfigPath(file))
	}

	return orderedEnvFiles, nil
}

// ProcessHooks executes Tasks defined in Hooks
func (app *DdevApp) ProcessHooks(hookName string) error {
	if SkipHooks {
		output.UserOut.Debugf("Skipping the execution of %s hook...", hookName)
		return nil
	}
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
			output.UserOut.Warn("A task failure does not mean that DDEV failed, but your hook configuration has a command that failed.")
		}
	}

	return nil
}

// GetDBImage uses the available version info
func (app *DdevApp) GetDBImage() string {
	dbImage := ddevImages.GetDBImage(app.Database.Type, app.Database.Version)
	return dbImage
}

// GetLocalTimezone tries to find local timezone from $TZ or /etc/localtime symlink
func (app *DdevApp) GetLocalTimezone() (string, error) {
	timezone := ""
	if os.Getenv("TZ") != "" {
		timezone = os.Getenv("TZ")
	} else {
		localtimeFile := filepath.Join("/etc", "localtime")
		var err error
		timezone, err = filepath.EvalSymlinks(localtimeFile)
		if err != nil {
			return "", fmt.Errorf("unable to read timezone from %s file: %v", localtimeFile, err)
		}
	}
	return util.GetTimezone(timezone)
}

// Start initiates docker-compose up
func (app *DdevApp) Start() error {
	var err error
	if app.IsMutagenEnabled() && globalconfig.DdevGlobalConfig.UseHardenedImages {
		return fmt.Errorf("mutagen is not compatible with use-hardened-images")
	}

	// We don't yet know the ComposeYaml values, so make sure they're
	// not set.
	app.ComposeYaml = nil

	// Set up ports to be replaced with ephemeral ports if needed
	app.RouterHTTPPort = app.GetRouterHTTPPort()
	app.RouterHTTPSPort = app.GetRouterHTTPSPort()
	app.MailpitHTTPPort = app.GetMailpitHTTPPort()
	app.MailpitHTTPSPort = app.GetMailpitHTTPSPort()
	portsToCheck := []*string{&app.RouterHTTPPort, &app.RouterHTTPSPort, &app.MailpitHTTPPort, &app.MailpitHTTPSPort}
	GetEphemeralPortsIfNeeded(portsToCheck, true)

	app.DockerEnv()
	dockerutil.EnsureDdevNetwork()
	// The project network may have duplicates, we can remove them here.
	// See https://github.com/ddev/ddev/pull/5508
	ctx, client := dockerutil.GetDockerClient()
	dockerutil.RemoveNetworkDuplicates(ctx, client, app.GetDefaultNetworkName())

	if err = dockerutil.CheckDockerCompose(); err != nil {
		util.Failed(`Your docker-compose version does not exist or is set to an invalid version.
Please use the built-in docker-compose.
Fix with 'ddev config global --required-docker-compose-version="" --use-docker-compose-from-path=false': %v`, err)
	}

	if runtime.GOOS == "darwin" {
		failOnRosetta()
	}
	err = app.ProcessHooks("pre-start")
	if err != nil {
		return err
	}

	err = PullBaseContainerImages()
	if err != nil {
		util.Warning("Unable to pull Docker images: %v", err)
	}

	if !nodeps.ArrayContainsString(app.GetOmittedContainers(), "db") {
		// OK to start if dbType is empty (nonexistent) or if it matches
		if dbType, err := app.GetExistingDBType(); err != nil || (dbType != "" && dbType != app.Database.Type+":"+app.Database.Version) {
			return fmt.Errorf("unable to start project %s because the configured database type does not match the current actual database. Please change your database type back to %s and start again, export, delete, and then change configuration and start. To get back to existing type use 'ddev config --database=%s' and then you might want to try 'ddev debug migrate-database %s', see docs at %s", app.Name, dbType, dbType, app.Database.Type+":"+app.Database.Version, "https://ddev.readthedocs.io/en/stable/users/extend/database-types/")
		}
	}

	app.CreateUploadDirsIfNecessary()

	if app.IsMutagenEnabled() {
		if globalconfig.DdevGlobalConfig.NoBindMounts {
			util.Warning("Mutagen is enabled because `no_bind_mounts: true` is set.\n`ddev config global --no-bind-mounts=false` if you do not intend that.")
		}
		err = app.GenerateMutagenYml()
		if err != nil {
			return err
		}
		if ok, volumeExists, info := CheckMutagenVolumeSyncCompatibility(app); !ok {
			util.Debug("Mutagen sync session, configuration, and Docker volume are in incompatible status: '%s', Removing Mutagen sync session '%s' and Docker volume %s", info, MutagenSyncName(app.Name), GetMutagenVolumeName(app))
			err = SyncAndPauseMutagenSession(app)
			if err != nil {
				util.Warning("Unable to SyncAndPauseMutagenSession() %s: %v", MutagenSyncName(app.Name), err)
			}
			terminateErr := TerminateMutagenSync(app)
			if terminateErr != nil {
				util.Warning("Unable to terminate Mutagen sync %s: %v", MutagenSyncName(app.Name), err)
			}
			if volumeExists {
				// Remove mounting container if necessary.
				container, err := dockerutil.FindContainerByName("ddev-" + app.Name + "-web")
				if err == nil && container != nil {
					err = dockerutil.RemoveContainer(container.ID)
					if err != nil {
						return fmt.Errorf(`unable to remove web container, please 'ddev restart': %v`, err)
					}
				}
				removeVolumeErr := dockerutil.RemoveVolume(GetMutagenVolumeName(app))
				if removeVolumeErr != nil {
					return fmt.Errorf(`unable to remove mismatched Mutagen Docker volume '%s'. Please use 'ddev restart' or 'ddev mutagen reset': %v`, GetMutagenVolumeName(app), removeVolumeErr)
				}
			}
		}
		// Check again to make sure the Mutagen Docker volume exists. It's compatible if we found it above
		// so we can keep it in that case.
		if !dockerutil.VolumeExists(GetMutagenVolumeName(app)) {
			util.Debug("Creating new docker volume '%s' with signature '%v'", GetMutagenVolumeName(app), GetDefaultMutagenVolumeSignature(app))
			_, err = dockerutil.CreateVolume(GetMutagenVolumeName(app), "local", nil, map[string]string{mutagenSignatureLabelName: GetDefaultMutagenVolumeSignature(app)})
			if err != nil {
				return fmt.Errorf("unable to create new Mutagen Docker volume %s: %v", GetMutagenVolumeName(app), err)
			}
		}
	}

	volumesNeeded := []string{"ddev-global-cache", "ddev-" + app.Name + "-snapshots"}
	for _, v := range volumesNeeded {
		_, err = dockerutil.CreateVolume(v, "local", nil, nil)
		if err != nil {
			return fmt.Errorf("unable to create Docker volume %s: %v", v, err)
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

	// Make sure that any ports allocated are available.
	// and of course add to global project list as well
	err = app.UpdateGlobalProjectList()
	if err != nil {
		return err
	}

	// The .ddev directory may still need to be populated, especially in tests
	err = PopulateExamplesCommandsHomeadditions(app.Name)
	if err != nil {
		return err
	}

	err = DownloadMutagenIfNeededAndEnabled(app)
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

	// TODO: We shouldn't be chowning /var/lib/mysql if PostgreSQL?
	util.Debug("chowning /mnt/ddev-global-cache and /var/lib/mysql to %s", uid)
	_, out, err := dockerutil.RunSimpleContainer(ddevImages.GetWebImage(), "start-chown-"+util.RandString(6), []string{"sh", "-c", fmt.Sprintf("chown -R %s /var/lib/mysql /mnt/ddev-global-cache", uid)}, []string{}, []string{}, []string{app.GetMariaDBVolumeName() + ":/var/lib/mysql", "ddev-global-cache:/mnt/ddev-global-cache"}, "", true, false, map[string]string{"com.ddev.site-name": ""}, nil, &dockerutil.NoHealthCheck)
	if err != nil {
		return fmt.Errorf("failed to RunSimpleContainer to chown volumes: %v, output=%s", err, out)
	}
	util.Debug("done chowning /mnt/ddev-global-cache and /var/lib/mysql to %s", uid)

	// Chown the PostgreSQL volume; this shouldn't have to be a separate stanza, but the
	// uid is 999 instead of current user
	if app.Database.Type == nodeps.Postgres {
		util.Debug("chowning chowning /var/lib/postgresql/data to 999")
		_, out, err := dockerutil.RunSimpleContainer(ddevImages.GetWebImage(), "start-postgres-chown-"+util.RandString(6), []string{"sh", "-c", fmt.Sprintf("chown -R %s /var/lib/postgresql/data", "999:999")}, []string{}, []string{}, []string{app.GetPostgresVolumeName() + ":/var/lib/postgresql/data"}, "", true, false, map[string]string{"com.ddev.site-name": ""}, nil, &dockerutil.NoHealthCheck)
		if err != nil {
			return fmt.Errorf("failed to RunSimpleContainer to chown PostgreSQL volume: %v, output=%s", err, out)
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
		util.Warning("Unable to pull Docker images: %v", err)
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
	err = util.Chmod(app.GetConfigPath("db_snapshots"), 0777)
	if err != nil {
		return err
	}

	// Build extra layers on web and db images if necessary
	output.UserOut.Printf("Building project images...")
	buildDurationStart := util.ElapsedDuration(time.Now())
	progress := "plain"
	util.Debug("Executing docker-compose -f %s build --progress=%s", app.DockerComposeFullRenderedYAMLPath(), progress)
	out, stderr, err := dockerutil.ComposeCmd(&dockerutil.ComposeCmdOpts{
		ComposeFiles: []string{app.DockerComposeFullRenderedYAMLPath()},
		Action:       []string{"--progress=" + progress, "build"},
		Progress:     true,
	})
	if err != nil {
		return fmt.Errorf("docker-compose build failed: %v, output='%s', stderr='%s'", err, out, stderr)
	}
	if globalconfig.DdevVerbose {
		util.Debug("docker-compose build output:\n%s\n\n", out)
	}

	_, logStderrOutput, err := dockerutil.RunSimpleContainer(ddevImages.GetWebImage()+"-"+app.Name+"-built", "log-stderr-"+app.Name+"-"+util.RandString(6), []string{"sh", "-c", "log-stderr.sh --show 2>/dev/null || true"}, []string{}, []string{}, nil, uid, true, false, map[string]string{"com.ddev.site-name": ""}, nil, nil)
	// If the web image is dirty, try to rebuild it immediately
	if err == nil && strings.TrimSpace(logStderrOutput) != "" && globalconfig.IsInternetActive() {
		util.Debug("Executing docker-compose -f %s build web --progress=%s --no-cache", app.DockerComposeFullRenderedYAMLPath(), progress)
		out, stderr, err = dockerutil.ComposeCmd(&dockerutil.ComposeCmdOpts{
			ComposeFiles: []string{app.DockerComposeFullRenderedYAMLPath()},
			Action:       []string{"--progress=" + progress, "build", "web", "--no-cache"},
			Progress:     true,
		})
		if err != nil {
			return fmt.Errorf("docker-compose build web --no-cache failed: %v, output='%s', stderr='%s'", err, out, stderr)
		}
		if globalconfig.DdevVerbose {
			util.Debug("docker-compose build web --no-cache output:\n%s\n\n", out)
		}
	}

	buildDuration := util.FormatDuration(buildDurationStart())
	util.Success("Project images built in %s.", buildDuration)

	util.Debug("Removing dangling images for the project %s", app.GetComposeProjectName())
	danglingImages, err := dockerutil.FindImagesByLabels(map[string]string{"com.ddev.buildhost": "", "com.docker.compose.project": app.GetComposeProjectName()}, true)
	if err != nil {
		return fmt.Errorf("unable to get dangling images for the project %s: %v", app.GetComposeProjectName(), err)
	}
	for _, danglingImage := range danglingImages {
		_ = dockerutil.RemoveImage(danglingImage.ID)
	}

	util.Debug("Executing docker-compose -f %s up -d", app.DockerComposeFullRenderedYAMLPath())
	_, _, err = dockerutil.ComposeCmd(&dockerutil.ComposeCmdOpts{
		ComposeFiles: []string{app.DockerComposeFullRenderedYAMLPath()},
		Action:       []string{"up", "-d"},
	})
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
					util.Warning("Failed to copy root CA into Docker volume ddev-global-cache/mkcert: %v", err)
				} else {
					util.Debug("Pushed mkcert rootca certs to ddev-global-cache/mkcert")
				}
			}
		}

		// If TLS supported and using Traefik, create cert/key and push into ddev-global-cache/traefik
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
				util.Warning("Failed to copy custom certs into Docker volume ddev-global-cache/custom_certs: %v", err)
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
			util.Failed("Mutagen Docker volume is not mounted. Please use `ddev restart`")
		}
		output.UserOut.Printf("Starting Mutagen sync process...")
		mutagenDuration := util.ElapsedDuration(time.Now())

		err = SetMutagenVolumeOwnership(app)
		if err != nil {
			return err
		}
		err = CreateOrResumeMutagenSync(app)
		if err != nil {
			return fmt.Errorf("failed to CreateOrResumeMutagenSync on Mutagen sync session '%s'. You may be able to resolve this problem using 'ddev mutagen reset' (err=%v)", MutagenSyncName(app.Name), err)
		}
		mStatus, _, _, err := app.MutagenStatus()
		if err != nil {
			return err
		}
		util.Debug("Mutagen status after sync: %s", mStatus)

		dur := util.FormatDuration(mutagenDuration())
		if mStatus == "ok" {
			util.Success("Mutagen sync flush completed in %s.\nFor details on sync status 'ddev mutagen st %s -l'", dur, MutagenSyncName(app.Name))
		} else {
			util.Error("Mutagen sync completed with problems in %s.\nFor details on sync status 'ddev mutagen st %s -l'", dur, MutagenSyncName(app.Name))
		}
		err = fileutil.TemplateStringToFile(`#ddev-generated`, nil, app.GetConfigPath("mutagen/.start-synced"))
		if err != nil {
			util.Warning("Could not create file %s: %v", app.GetConfigPath("mutagen/.start-synced"), err)
		}
	}

	// At this point we should have all files synced inside the container
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
	output.UserOut.Printf("Waiting for containers to become ready: %v", dependers)
	waitErr := app.Wait(dependers)

	err = PopulateGlobalCustomCommandFiles()
	if err != nil {
		util.Warning("Failed to populate global custom command files: %v", err)
	}

	if globalconfig.DdevVerbose {
		out, err = app.CaptureLogs("web", true, "200")
		if err != nil {
			util.Warning("Unable to capture logs from web container: %v", err)
		} else {
			util.Debug("docker-compose up output:\n%s\n\n", out)
		}
	}

	if waitErr != nil {
		util.Failed("Failed waiting for web/db containers to become ready: %v", waitErr)
	}

	// WebExtraDaemons have to be started after Mutagen sync is done, because so often
	// they depend on code being synced into the container/volume
	if len(app.WebExtraDaemons) > 0 {
		output.UserOut.Printf("Starting web_extra_daemons...")
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
		util.Warning("Something is wrong with your Docker provider and /mnt/ddev_config is not mounted from the project .ddev folder. Your project cannot normally function successfully with this situation. Is your project in your home directory?")
	}

	util.Debug("Getting stderr output from 'log-stderr.sh --show'")
	logStderr, _, _ := app.Exec(&ExecOpts{
		Cmd: "log-stderr.sh --show 2>/dev/null || true",
	})
	logStderr = strings.TrimSpace(logStderr)
	if logStderr != "" {
		util.Warning(logStderr)
	}

	if !IsRouterDisabled(app) {
		output.UserOut.Printf("Starting %s if necessary...", nodeps.RouterContainer)
		err = StartDdevRouter()
		if err != nil {
			return err
		}
	}

	waitLabels := map[string]string{"com.ddev.site-name": app.GetName()}
	containersAwaited, err := dockerutil.FindContainersByLabels(waitLabels)
	if err != nil {
		return err
	}
	containerNames := dockerutil.GetContainerNames(containersAwaited, []string{GetContainerName(app, "web"), GetContainerName(app, "db")})
	if len(containerNames) > 0 {
		output.UserOut.Printf("Waiting %ds for additional project containers %v to become ready...", app.GetMaxContainerWaitTime(), containerNames)
	}
	err = app.WaitByLabels(waitLabels)
	if err != nil {
		return err
	}

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

	if logStderr != "" {
		util.Warning(`Some components of the project %s were not installed properly.
The project is running anyway, but see the warnings above for details.
If offline, run 'ddev restart' once you are back online.
If online, check your connection and run 'ddev restart' later.
If this seems to be a config issue, update it accordingly.`, app.Name)
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

// PullContainerImages configured Docker images with full output, since docker-compose up doesn't have nice output
func (app *DdevApp) PullContainerImages() error {
	images, err := app.FindAllImages()
	if err != nil {
		return err
	}
	images = append(images, FindNotOmittedImages(app)...)

	for _, i := range images {
		err := dockerutil.Pull(i)
		if err != nil {
			return err
		}
		util.Debug("Pulled image for %s", i)
	}

	return nil
}

// PullBaseContainerImages pulls only the fundamentally needed images so they can be available early.
// We always need web image, busybox, and ddev-utilities for housekeeping.
func PullBaseContainerImages() error {
	images := []string{ddevImages.GetWebImage(), versionconstants.BusyboxImage, versionconstants.UtilitiesImage}
	images = append(images, FindNotOmittedImages(nil)...)

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

// FindNotOmittedImages returns an array of image names not omitted by global or project configuration
func FindNotOmittedImages(app *DdevApp) []string {
	var images []string
	containerImageMap := map[string]func() string{
		SSHAuthName:            ddevImages.GetSSHAuthImage,
		nodeps.RouterContainer: ddevImages.GetRouterImage,
	}

	for containerName, getImage := range containerImageMap {
		if nodeps.ArrayContainsString(globalconfig.DdevGlobalConfig.OmitContainersGlobal, containerName) {
			continue
		}
		if app == nil || !nodeps.ArrayContainsString(app.OmitContainers, containerName) {
			images = append(images, getImage())
		}
	}

	return images
}

// GetMaxContainerWaitTime looks through all services and returns the max time we expect
// to wait for all containers to become `healthy`. Mostly this is healtcheck.start_period.
// Defaults to DefaultContainerTimeout (usually 120 unless overridden)
func (app *DdevApp) GetMaxContainerWaitTime() int {
	defaultContainerTimeout, _ := strconv.Atoi(app.DefaultContainerTimeout)
	maxWaitTime := defaultContainerTimeout

	if app.ComposeYaml == nil {
		return defaultContainerTimeout
	}
	if y, ok := app.ComposeYaml["services"]; ok {
		for _, v := range y.(map[string]interface{}) {
			if i, ok := v.(map[string]interface{})["healthcheck"]; ok {
				if startPeriod, ok := i.(map[string]interface{})["start_period"]; ok {
					duration, err := time.ParseDuration(startPeriod.(string))
					if err != nil {
						continue
					}
					t := int(duration.Seconds())
					if t > maxWaitTime {
						maxWaitTime = t
					}
				} else { /* In this case we didn't have a specified start_period, so guess at one */
					// Use defaults for interval and retries
					// https://docs.docker.com/reference/dockerfile/#healthcheck
					interval := 5
					retries := 3

					if intervalStr, ok := i.(map[string]interface{})["interval"]; ok {
						intervalInt, err := time.ParseDuration(intervalStr.(string))
						if err == nil {
							interval = int(intervalInt.Seconds())
						}
					}
					if retriesSpecified, ok := i.(map[string]interface{})["retries"]; ok {
						retries = retriesSpecified.(int)
					}
					// If the retries*interval is greater than what we've found before
					// then use it. This will be unusual.
					if retries*interval > maxWaitTime {
						maxWaitTime = retries * interval
					}
				}
			}
		}
	}
	return maxWaitTime
}

// CheckExistingAppInApproot looks to see if we already have a project in this approot with different name
func (app *DdevApp) CheckExistingAppInApproot() error {
	pList := globalconfig.GetGlobalProjectList()
	for name, v := range pList {
		if app.AppRoot == v.AppRoot && name != app.Name {
			return fmt.Errorf(`this project root '%s' already contains a project named '%s'. You may want to remove the existing project with "ddev stop --unlist %s"`, v.AppRoot, name, name)
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
		util.Warning("Not generating PostgreSQL config files because running with root privileges.")
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
			err = util.Chmod(configPath, 0666)
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
		err = util.Chmod(configPath, 0666)
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
	// Env is the array of environment variables
	Env []string
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

	if len(opts.Env) > 0 {
		for _, envVar := range opts.Env {
			baseComposeExecCmd = append(baseComposeExecCmd, "-e", envVar)
		}
	}

	baseComposeExecCmd = append(baseComposeExecCmd, opts.Service)

	// Cases to handle
	// - Free form, all unquoted. Like `ls -l -a`
	// - Quoted to delay pipes and other features to container, like `"ls -l -a | grep junk"`
	// Note that a set quoted on the host in ddev e will come through as a single arg

	if len(opts.RawCmd) == 0 { // Use opts.Cmd and prepend with bash
		// Use Bash for our containers, sh for 3rd-party containers
		// that may not have Bash.
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
		outRes, errRes, err = dockerutil.ComposeCmd(&dockerutil.ComposeCmdOpts{
			ComposeFiles: []string{app.DockerComposeFullRenderedYAMLPath()},
			Action:       r,
		})
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

	// Use Bash for our containers, sh for 3rd-party containers
	// that may not have Bash.
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
	ctx, client := dockerutil.GetDockerClient()

	var container *dockerTypes.Container
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

	logOpts := dockerContainer.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     follow,
		Timestamps: timestamps,
	}

	if tailLines != "" {
		logOpts.Tail = tailLines
	}

	rc, err := client.ContainerLogs(ctx, container.ID, logOpts)
	if err != nil {
		return err
	}
	defer rc.Close()

	// Copy logs to user output
	_, err = stdcopy.StdCopy(output.UserOut.Out, output.UserOut.Out, rc)
	if err != nil {
		return fmt.Errorf("failed to copy container logs: %v", err)
	}

	return nil
}

// CaptureLogs returns logs for a site's given container.
// See docker.LogsOptions for more information about valid tailLines values.
func (app *DdevApp) CaptureLogs(service string, timestamps bool, tailLines string) (string, error) {
	ctx, client := dockerutil.GetDockerClient()

	var container *dockerTypes.Container
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

	var stdout bytes.Buffer
	logOpts := dockerContainer.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     false,
		Timestamps: timestamps,
	}

	if tailLines != "" {
		logOpts.Tail = tailLines
	}

	rc, err := client.ContainerLogs(ctx, container.ID, logOpts)
	if err != nil {
		return "", err
	}
	defer rc.Close()

	_, err = stdcopy.StdCopy(&stdout, &stdout, rc)
	if err != nil {
		return "", fmt.Errorf("failed to copy container logs: %v", err)
	}

	return stdout.String(), nil
}

// DockerEnv sets environment variables for a docker-compose run.
func (app *DdevApp) DockerEnv() {

	uidStr, gidStr, _ := util.GetContainerUIDGid()

	// Warn about running as root if we're not on Windows.
	if uidStr == "0" || gidStr == "0" {
		util.Warning("Warning: containers will run as root. This could be a security risk on Linux.")
	}

	// For Gitpod, Codespaces
	// * provide default host-side port bindings, assuming only one project running,
	//   as is usual on Gitpod, but if more than one project, can override with normal
	//   config.yaml settings.
	// Codespaces stumbles if not on a "standard" port like port 80
	if nodeps.IsCodespaces() {
		if app.HostWebserverPort == "" {
			app.HostWebserverPort = "80"
		}
	}
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
		if app.HostMailpitPort == "" {
			app.HostMailpitPort = "8027"
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

	// Figure out what the host-webserver (host-http) port is
	// First we try to see if there's an existing webserver container and use that
	hostHTTPPort, err := app.GetPublishedPort("web")
	hostHTTPPortStr := ""
	// Otherwise we'll use the configured value from app.HostWebserverPort
	if hostHTTPPort > 0 || err == nil {
		hostHTTPPortStr = strconv.Itoa(hostHTTPPort)
	} else {
		hostHTTPPortStr = app.HostWebserverPort
	}

	// Figure out what the host-webserver https port is
	// the https port is rarely used because ddev-router does termination
	// for the vast majority of applications
	hostHTTPSPort, err := app.GetPublishedPortForPrivatePort("web", 443)
	hostHTTPSPortStr := ""
	if hostHTTPSPort > 0 || err == nil {
		hostHTTPSPortStr = strconv.Itoa(hostHTTPSPort)
	} else {
		hostHTTPSPortStr = app.HostHTTPSPort

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
		"COMPOSE_PROJECT_NAME":           app.GetComposeProjectName(),
		"COMPOSE_REMOVE_ORPHANS":         "true",
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
		"DDEV_FILES_DIR":                 app.GetContainerUploadDir(),
		"DDEV_FILES_DIRS":                strings.Join(app.GetContainerUploadDirs(), ","),

		"DDEV_HOST_DB_PORT":        dbPortStr,
		"DDEV_HOST_MAILHOG_PORT":   app.HostMailpitPort,
		"DDEV_HOST_MAILPIT_PORT":   app.HostMailpitPort,
		"DDEV_HOST_HTTP_PORT":      hostHTTPPortStr,
		"DDEV_HOST_HTTPS_PORT":     hostHTTPSPortStr,
		"DDEV_HOST_WEBSERVER_PORT": hostHTTPPortStr,
		"DDEV_MAILHOG_HTTPS_PORT":  app.GetMailpitHTTPSPort(),
		"DDEV_MAILHOG_PORT":        app.GetMailpitHTTPPort(),
		"DDEV_MAILPIT_HTTP_PORT":   app.GetMailpitHTTPPort(),
		"DDEV_MAILPIT_HTTPS_PORT":  app.GetMailpitHTTPSPort(),
		"DDEV_MAILPIT_PORT":        app.GetMailpitHTTPPort(),
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
	// It's used for special startup on restoring to a snapshot or for PostgreSQL.
	if len(os.Getenv("DDEV_DB_CONTAINER_COMMAND")) == 0 {
		v := ""
		if app.Database.Type == nodeps.Postgres { // config_file spec for PostgreSQL
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

	if _, _, err := dockerutil.ComposeCmd(&dockerutil.ComposeCmdOpts{
		ComposeFiles: []string{app.DockerComposeFullRenderedYAMLPath()},
		Action:       []string{"stop"},
	}); err != nil {
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
	waitTime := app.GetMaxContainerWaitTime()
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
		waitTime := app.GetMaxContainerWaitTime()
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
	waitTime := app.GetMaxContainerWaitTime()
	err := dockerutil.ContainersWait(waitTime, labels)
	if err != nil {
		return fmt.Errorf("container(s) failed to become healthy before their configured timeout or in %d seconds.\nThis might be a problem with the healthcheck and not a functional problem.\nThe error was '%v'", waitTime, err.Error())
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
		return "", fmt.Errorf("failed to process pre-snapshot hooks: %v", err)
	}

	if snapshotName == "" {
		t := time.Now()
		snapshotName = app.Name + "_" + t.Format("20060102150405")
	}

	snapshotFile := snapshotName + "-" + app.Database.Type + "_" + app.Database.Version + ".gz"

	existingSnapshots, err := app.ListSnapshotNames()
	if err != nil {
		return "", err
	}
	if nodeps.ArrayContainsString(existingSnapshots, snapshotName) {
		return "", fmt.Errorf("snapshot %s already exists, please use another snapshot name or clean up snapshots with `ddev snapshot --cleanup`", snapshotFile)
	}

	// Container side has to use path.Join instead of filepath.Join because they are
	// targeted at the Linux filesystem, so won't work with filepath on Windows
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
			util.Warning("Could not check for tables to optimize (mysql 8.0): %v (stdout='%s', stderr='%s')", err, stdout, stderr)
		} else {
			stdout = strings.Trim(stdout, "\n\t ")
			tables := strings.Split(stdout, "\n")
			// util.Success("tables=%v len(tables)=%d stdout was '%s'", tables, len(tables), stdout)
			if len(stdout) > 0 && len(tables) > 0 {
				for _, t := range tables {
					r := strings.Split(t, `/`)
					if len(r) != 2 {
						util.Warning("Unable to get database/table from %s", r)
						continue
					}
					d := r[0]
					t := r[1]
					stdout, stderr, err := app.Exec(&ExecOpts{
						Service: "db",
						Cmd:     fmt.Sprintf(`set -eu -o pipefail; MYSQL_PWD=root mysql -uroot -D %s -e 'OPTIMIZE TABLES %s';`, d, t),
					})
					if err != nil {
						util.Warning("Unable to optimize table %s (mysql 8.0): %v (stdout='%s', stderr='%s')", t, err, stdout, stderr)
					}
				}
				util.Success("Optimized MySQL 8.0 tables '%s' in preparation for snapshot", strings.Join(tables, `,'`))
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
		// But if we are using bind-mounts, we can copy it to where the snapshot is
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

	// Clean up the in-container dir that we used
	_, _, err = dockerutil.Exec(dbContainer.ID, fmt.Sprintf("rm -f %s/%s", containerSnapshotDir, snapshotFile), "")
	if err != nil {
		return "", err
	}
	err = app.ProcessHooks("post-snapshot")
	if err != nil {
		return snapshotFile, fmt.Errorf("failed to process post-snapshot hooks: %v", err)
	}

	return snapshotName, nil
}

// getBackupCommand returns the command to dump the entire db system for the various databases
func getBackupCommand(app *DdevApp, targetFile string) string {

	c := fmt.Sprintf(`mariabackup --backup --stream=mbstream --user=root --password=root --socket=/var/tmp/mysql.sock  2>/tmp/snapshot_%s.log | gzip > "%s"`, path.Base(targetFile), targetFile)

	oldMariaVersions := []string{"5.5", "10.0"}

	switch {
	// Old MariaDB versions don't have mariabackup, use xtrabackup for them as well as MySQL
	case app.Database.Type == nodeps.MariaDB && nodeps.ArrayContainsString(oldMariaVersions, app.Database.Version):
		fallthrough
	case app.Database.Type == nodeps.MySQL:
		c = fmt.Sprintf(`xtrabackup --backup --stream=xbstream --user=root --password=root --socket=/var/tmp/mysql.sock  2>/tmp/snapshot_%s.log | gzip > "%s"`, path.Base(targetFile), targetFile)
	case app.Database.Type == nodeps.Postgres:
		c = fmt.Sprintf("rm -rf /var/tmp/pgbackup && pg_basebackup -D /var/tmp/pgbackup 2>/tmp/snapshot_%s.log && tar -czf %s -C /var/tmp/pgbackup/ .", path.Base(targetFile), targetFile)
	}
	return c
}

// fullDBFromVersion takes a MariaDB or MySQL version number
// in x.xx format and returns something like mariadb-10.5
func fullDBFromVersion(v string) string {
	snapshotDBVersion := ""
	// The old way (when we only had MariaDB and then when had MariaDB and also MySQL)
	// was to have the version number and derive the database type from it,
	// so that's what is going on here. But we create a string like "mariadb_10.3" from
	// the version number
	switch {
	case v == "5.6" || v == "5.7" || v == "8.0":
		snapshotDBVersion = "mysql_" + v

	// 5.5 isn't actually necessarily correct, because could be
	// MySQL 5.5. But maria and MySQL 5.5 databases were compatible anyway.
	case v == "5.5" || v >= "10.0":
		snapshotDBVersion = "mariadb_" + v
	}
	return snapshotDBVersion
}

// Stop stops and Removes the Docker containers for the project in current directory.
func (app *DdevApp) Stop(removeData bool, createSnapshot bool) error {
	app.DockerEnv()
	var err error

	clear(EphemeralRouterPortsAssigned)
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

	if app.IsMutagenEnabled() {
		err = SyncAndPauseMutagenSession(app)
		if err != nil {
			util.Warning("Unable to SyncAndPauseMutagenSession: %v", err)
		}
	}

	if globalconfig.DdevGlobalConfig.IsTraefikRouter() && status == SiteRunning {
		_, _, err = app.Exec(&ExecOpts{
			Cmd: fmt.Sprintf("rm -f /mnt/ddev-global-cache/traefik/*/%s.{yaml,crt,key}", app.Name),
		})
		if err != nil {
			util.Warning("Unable to clean up Traefik configuration: %v", err)
		}
	}
	// Clean up ddev-global-cache
	if removeData {
		c := fmt.Sprintf("rm -rf /mnt/ddev-global-cache/*/%s-{web,db} /mnt/ddev-global-cache/traefik/*/%s.{yaml,crt,key}", app.Name, app.Name)
		util.Debug("Cleaning ddev-global-cache with command '%s'", c)
		_, out, err := dockerutil.RunSimpleContainer(ddevImages.GetWebImage(), "clean-ddev-global-cache-"+util.RandString(6), []string{"bash", "-c", c}, []string{}, []string{}, []string{"ddev-global-cache:/mnt/ddev-global-cache"}, "", true, false, map[string]string{`com.ddev.site-name`: ""}, nil, &dockerutil.NoHealthCheck)
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
		if app.IsMutagenEnabled() {
			err = TerminateMutagenSync(app)
			if err != nil {
				util.Warning("Unable to terminate Mutagen session %s: %v", MutagenSyncName(app.Name), err)
			}
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

		vols := []string{app.GetMariaDBVolumeName(), app.GetPostgresVolumeName(), GetMutagenVolumeName(app)}
		if globalconfig.DdevGlobalConfig.NoBindMounts {
			vols = append(vols, app.Name+"-ddev-config")
		}
		for _, volName := range vols {
			err = dockerutil.RemoveVolume(volName)
			if err != nil {
				util.Warning("Could not remove volume %s: %v", volName, err)
			} else {
				util.Success("Volume %s for project %s was deleted", volName, app.Name)
			}
		}
		deleteServiceVolumes(app)

		dbBuilt := app.GetDBImage() + "-" + app.Name + "-built"
		_ = dockerutil.RemoveImage(dbBuilt)

		webBuilt := ddevImages.GetWebImage() + "-" + app.Name + "-built"
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
					util.Warning("Could not remove volume %s: %v", volName, err)
				} else {
					util.Success("Deleting third-party persistent volume %s", volName)
				}
			}
		}
	}
}

// RemoveGlobalProjectInfo deletes the project from DdevProjectList
func (app *DdevApp) RemoveGlobalProjectInfo() {
	_ = globalconfig.RemoveProjectInfo(app.Name)
}

// GetHTTPURL returns the HTTP URL for an app.
func (app *DdevApp) GetHTTPURL() string {
	url := ""
	if !IsRouterDisabled(app) {
		url = "http://" + app.GetHostname()
		// If the HTTP port is the default "80", it's not included in the URL
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
		// If the HTTPS port is 443 (default), it doesn't get included in URL
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
		previewDomain := os.Getenv("GITHUB_CODESPACES_PORT_FORWARDING_DOMAIN")
		if codespaceName != "" && previewDomain != "" {
			url := fmt.Sprintf("https://%s-%s.%s", codespaceName, app.HostWebserverPort, previewDomain)
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
		// If the HTTPS port is 443 (default), it doesn't get included in URL
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
	if app.CanUseHTTPOnly() {
		urlList = httpURLs
	}
	if len(urlList) > 0 {
		return urlList[0]
	}
	// Failure mode, returns an empty string
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
		// Or find it by looking at Docker containers
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
		case webContainerExists, invalidConfigFile, invalidConstraint, invalidHostname, invalidAppType, invalidPHPVersion, invalidWebserverType, invalidProvider:
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

// NormalizeProjectName replaces underscores in the site name with hyphens.
func NormalizeProjectName(siteName string) string {
	return strings.ReplaceAll(siteName, "_", "-")
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

// GetHostWorkingDir will determine the appropriate working directory for the service on the host side
func (app *DdevApp) GetHostWorkingDir(service string, dir string) string {
	// We have a corresponding host working_dir for the "web" service only
	if service != "web" {
		return ""
	}
	if dir == "" && app.WorkingDir != nil {
		dir = app.WorkingDir[service]
	}
	containerWorkingDirPrefix := strings.TrimSuffix(app.GetAbsAppRoot(true), "/") + "/"
	if !strings.HasPrefix(dir, containerWorkingDirPrefix) {
		return ""
	}
	return filepath.Join(app.GetAbsAppRoot(false), strings.TrimPrefix(dir, containerWorkingDirPrefix))
}

// GetNFSMountVolumeName returns the Docker volume name of the nfs mount volume
func (app *DdevApp) GetNFSMountVolumeName() string {
	// This is lowercased because the automatic naming in docker-compose v1/2
	// defaulted to lowercase the name
	// Although some volume names are auto-lowercased by Docker, this one
	// is explicitly specified by us and is not lowercased.
	return "ddev-" + app.Name + "_nfsmount"
}

// GetMariaDBVolumeName returns the Docker volume name of the mariadb/database volume
// For historical reasons this isn't lowercased.
func (app *DdevApp) GetMariaDBVolumeName() string {
	return app.Name + "-mariadb"
}

// GetPostgresVolumeName returns the Docker volume name of the postgres/database volume
// For historical reasons this isn't lowercased.
func (app *DdevApp) GetPostgresVolumeName() string {
	return app.Name + "-postgres"
}

// GetComposeProjectName returns the name of the docker-compose project
func (app *DdevApp) GetComposeProjectName() string {
	return strings.ToLower("ddev-" + strings.Replace(app.Name, `.`, "", -1))
}

// GetDefaultNetworkName returns the default project network name
func (app *DdevApp) GetDefaultNetworkName() string {
	return app.GetComposeProjectName() + "_default"
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
		errMsg := fmt.Errorf("service '%s' does not have the 'networks: [default, ddev_default]' stanza, required since v1.19, please add it, see %s", s, "https://ddev.readthedocs.io/en/stable/users/extend/custom-compose-files/#docker-composeyaml-examples")
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
func GetContainer(app *DdevApp, service string) (*dockerTypes.Container, error) {
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
	if err := util.Chmod(filepath.Dir(destPath), 0755); err != nil {
		return err
	}

	// If the destination path exists, purge it as was warned
	if fileutil.FileExists(destPath) {
		if err := fileutil.PurgeDirectory(destPath); err != nil {
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

	if err := copy.Copy(importPath, destPath); err != nil {
		return err
	}

	return nil
}
