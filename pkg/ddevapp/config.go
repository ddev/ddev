package ddevapp

import (
	"bytes"
	"fmt"
	"github.com/Masterminds/sprig"
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/nodeps"
	"github.com/mitchellh/go-homedir"
	"html/template"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/drud/ddev/pkg/globalconfig"

	"regexp"

	"runtime"

	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/util"
	"github.com/drud/ddev/pkg/version"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// Regexp pattern to determine if a hostname is valid per RFC 1123.
var hostRegex = regexp.MustCompile(`^(([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]*[a-zA-Z0-9])\.)*([A-Za-z0-9]|[A-Za-z0-9][A-Za-z0-9\-]*[A-Za-z0-9])$`)

// Provider is the interface which all provider plugins must implement.
type Provider interface {
	Init(app *DdevApp) error
	ValidateField(string, string) error
	PromptForConfig() error
	Write(string) error
	Read(string) error
	Validate() error
	GetBackup(string, string) (fileLocation string, importPath string, err error)
}

// init() is for testing situations only, allowing us to override the default webserver type
// or caching behavior

func init() {
	// This is for automated testing only. It allows us to override the webserver type.
	if testWebServerType := os.Getenv("DDEV_TEST_WEBSERVER_TYPE"); testWebServerType != "" {
		nodeps.WebserverDefault = testWebServerType
	}
	if testNFSMount := os.Getenv("DDEV_TEST_USE_NFSMOUNT"); testNFSMount != "" {
		nodeps.NFSMountEnabledDefault = true
	}
}

// NewApp creates a new DdevApp struct with defaults set and overridden by any existing config.yml.
func NewApp(AppRoot string, includeOverrides bool, provider string) (*DdevApp, error) {
	// Set defaults.
	app := &DdevApp{}

	homeDir, _ := homedir.Dir()
	if AppRoot == filepath.Dir(globalconfig.GetGlobalDdevDir()) || app.AppRoot == homeDir {
		return nil, fmt.Errorf("ddev config is not useful in home directory (%s)", homeDir)
	}

	app.AppRoot = AppRoot
	if !fileutil.FileExists(AppRoot) {
		return app, fmt.Errorf("project root %s does not exist", AppRoot)
	}
	app.ConfigPath = app.GetConfigPath("config.yaml")
	app.APIVersion = version.DdevVersion
	app.Type = nodeps.AppTypePHP
	app.PHPVersion = nodeps.PHPDefault
	app.WebserverType = nodeps.WebserverDefault
	app.NFSMountEnabled = nodeps.NFSMountEnabledDefault
	app.RouterHTTPPort = nodeps.DdevDefaultRouterHTTPPort
	app.RouterHTTPSPort = nodeps.DdevDefaultRouterHTTPSPort
	app.PHPMyAdminPort = nodeps.DdevDefaultPHPMyAdminPort
	app.MailhogPort = nodeps.DdevDefaultMailhogPort
	// Provide a default app name based on directory name
	app.Name = filepath.Base(app.AppRoot)
	app.OmitContainers = globalconfig.DdevGlobalConfig.OmitContainers
	app.ProjectTLD = nodeps.DdevDefaultTLD
	app.UseDNSWhenPossible = true

	// These should always default to the latest image/tag names from the Version package.
	app.WebImage = version.GetWebImage()
	app.DBAImage = version.GetDBAImage()

	// Load from file if available. This will return an error if the file doesn't exist,
	// and it is up to the caller to determine if that's an issue.
	if _, err := os.Stat(app.ConfigPath); !os.IsNotExist(err) {
		_, err = app.ReadConfig(includeOverrides)
		if err != nil {
			return app, fmt.Errorf("%v exists but cannot be read. It may be invalid due to a syntax error.: %v", app.ConfigPath, err)
		}
	}
	app.SetApptypeSettingsPaths()

	// Allow override with provider.
	// Otherwise we accept whatever might have been in config file if there was anything.
	if provider == "" && app.Provider != "" {
		// Do nothing. This is the case where the config has a provider and no override is provided. Config wins.
	} else if provider == nodeps.ProviderPantheon || provider == nodeps.ProviderDrudS3 || provider == nodeps.ProviderDefault {
		app.Provider = provider // Use the provider passed-in. Function argument wins.
	} else if provider == "" && app.Provider == "" {
		app.Provider = nodeps.ProviderDefault // Nothing passed in, nothing configured. Set c.Provider to default
	} else {
		return app, fmt.Errorf("provider '%s' is not implemented", provider)
	}
	app.SetInstrumentationAppTags()
	return app, nil
}

// GetConfigPath returns the path to an application config file specified by filename.
func (app *DdevApp) GetConfigPath(filename string) string {
	return filepath.Join(app.AppRoot, ".ddev", filename)
}

// WriteConfig writes the app configuration into the .ddev folder.
func (app *DdevApp) WriteConfig() error {

	// Work against a copy of the DdevApp, since we don't want to actually change it.
	appcopy := *app
	// Update the "APIVersion" to be the ddev version.
	appcopy.APIVersion = version.DdevVersion

	// Only set the images on write if non-default values have been specified.
	if appcopy.WebImage == version.GetWebImage() {
		appcopy.WebImage = ""
	}
	// If the DBImage is actually just created/equal to the maria or mysql version
	// then remove it from the output.
	if appcopy.DBImage == version.GetDBImage(nodeps.MariaDB, appcopy.MariaDBVersion) || appcopy.DBImage == version.GetDBImage(nodeps.MySQL, appcopy.MySQLVersion) {
		appcopy.DBImage = ""
	}
	if appcopy.DBAImage == version.GetDBAImage() {
		appcopy.DBAImage = ""
	}
	if appcopy.DBAImage == version.GetDBAImage() {
		appcopy.DBAImage = ""
	}
	if appcopy.MailhogPort == nodeps.DdevDefaultMailhogPort {
		appcopy.MailhogPort = ""
	}
	if appcopy.PHPMyAdminPort == nodeps.DdevDefaultPHPMyAdminPort {
		appcopy.PHPMyAdminPort = ""
	}
	if appcopy.ProjectTLD == nodeps.DdevDefaultTLD {
		appcopy.ProjectTLD = ""
	}
	if appcopy.MariaDBVersion == version.GetDBImage(nodeps.MariaDB) {
		appcopy.MariaDBVersion = ""
	}

	// We now want to reserve the port we're writing for HostDBPort and HostWebserverPort and so they don't
	// accidentally get used for other projects.
	err := app.UpdateGlobalProjectList()
	if err != nil {
		return err
	}

	// Don't write default working dir values to config
	defaults := appcopy.DefaultWorkingDirMap()
	for service, defaultWorkingDir := range defaults {
		if app.WorkingDir[service] == defaultWorkingDir {
			delete(appcopy.WorkingDir, service)
		}
	}

	err = PrepDdevDirectory(filepath.Dir(appcopy.ConfigPath))
	if err != nil {
		return err
	}

	cfgbytes, err := yaml.Marshal(appcopy)
	if err != nil {
		return err
	}

	// Append current image information
	cfgbytes = append(cfgbytes, []byte(fmt.Sprintf("\n\n# This config.yaml was created with ddev version %s\n# webimage: %s\n# dbimage: %s\n# dbaimage: %s\n# However we do not recommend explicitly wiring these images into the\n# config.yaml as they may break future versions of ddev.\n# You can update this config.yaml using 'ddev config'.\n", version.DdevVersion, version.GetWebImage(), version.GetDBImage(nodeps.MariaDB), version.GetDBAImage()))...)

	// Append hook information and sample hook suggestions.
	cfgbytes = append(cfgbytes, []byte(ConfigInstructions)...)
	cfgbytes = append(cfgbytes, appcopy.GetHookDefaultComments()...)

	err = ioutil.WriteFile(appcopy.ConfigPath, cfgbytes, 0644)
	if err != nil {
		return err
	}

	provider, err := appcopy.GetProvider()
	if err != nil {
		return err
	}

	err = provider.Write(appcopy.GetConfigPath("import.yaml"))
	if err != nil {
		return err
	}

	// Allow project-specific post-config action
	err = appcopy.PostConfigAction()
	if err != nil {
		return err
	}

	// Write example Dockerfiles into build directories
	contents := []byte(`
# You can copy this Dockerfile.example to Dockerfile to add configuration
# or packages or anything else to your webimage
ARG BASE_IMAGE=` + app.WebImage + `
FROM $BASE_IMAGE
RUN npm install --global gulp-cli
`)

	err = WriteImageDockerfile(app.GetConfigPath("web-build")+"/Dockerfile.example", contents)
	if err != nil {
		return err
	}
	contents = []byte(`
# You can copy this Dockerfile.example to Dockerfile to add configuration
# or packages or anything else to your dbimage
ARG BASE_IMAGE=` + app.GetDBImage() + `
FROM $BASE_IMAGE
RUN echo "Built from ` + app.GetDBImage() + `" >/var/tmp/built-from.txt
`)

	err = WriteImageDockerfile(app.GetConfigPath("db-build")+"/Dockerfile.example", contents)
	if err != nil {
		return err
	}

	return nil
}

// UpdateGlobalProjectList updates any information about project that
// is tracked in global project list:
// - approot
// - configured host ports
// checks that configured host ports are not already
// reserved by another project
func (app *DdevApp) UpdateGlobalProjectList() error {
	portsToReserve := []string{}
	if app.HostDBPort != "" {
		portsToReserve = append(portsToReserve, app.HostDBPort)
	}
	if app.HostWebserverPort != "" {
		portsToReserve = append(portsToReserve, app.HostWebserverPort)
	}
	if app.HostHTTPSPort != "" {
		portsToReserve = append(portsToReserve, app.HostHTTPSPort)
	}

	if len(portsToReserve) > 0 {
		err := globalconfig.CheckHostPortsAvailable(app.Name, portsToReserve)
		if err != nil {
			return err
		}
	}
	err := globalconfig.ReservePorts(app.Name, portsToReserve)
	if err != nil {
		return err
	}
	err = globalconfig.SetProjectAppRoot(app.Name, app.AppRoot)
	if err != nil {
		return err
	}

	return nil
}

// ReadConfig reads project configuration from the config.yaml file
// It does not attempt to set default values; that's NewApp's job.
func (app *DdevApp) ReadConfig(includeOverrides bool) ([]string, error) {

	// Load config.yaml
	err := app.LoadConfigYamlFile(app.ConfigPath)
	if err != nil {
		return []string{}, fmt.Errorf("unable to load config file %s: %v", app.ConfigPath, err)
	}

	configOverrides := []string{}
	// Load config.*.y*ml after in glob order
	if includeOverrides {
		glob := filepath.Join(filepath.Dir(app.ConfigPath), "config.*.y*ml")
		configOverrides, err = filepath.Glob(glob)
		if err != nil {
			return []string{}, err
		}

		for _, item := range configOverrides {
			err = app.LoadConfigYamlFile(item)
			if err != nil {
				return []string{}, fmt.Errorf("unable to load config file %s: %v", item, err)
			}
		}
	}

	return append([]string{app.ConfigPath}, configOverrides...), nil
}

// LoadConfigYamlFile loads one config.yaml into app, overriding what might be there.
func (app *DdevApp) LoadConfigYamlFile(filePath string) error {
	source, err := ioutil.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("could not find an active ddev configuration at %s have you run 'ddev config'? %v", app.ConfigPath, err)
	}

	// validate extend command keys
	err = validateHookYAML(source)
	if err != nil {
		return fmt.Errorf("invalid configuration in %s: %v", app.ConfigPath, err)
	}

	// ReadConfig config values from file.
	err = yaml.Unmarshal(source, app)
	if err != nil {
		return err
	}
	return nil
}

// WarnIfConfigReplace just messages user about whether config is being replaced or created
func (app *DdevApp) WarnIfConfigReplace() {
	if app.ConfigExists() {
		util.Warning("You are reconfiguring the project at %s.\nThe existing configuration will be updated and replaced.", app.AppRoot)
	} else {
		util.Success("Creating a new ddev project config in the current directory (%s)", app.AppRoot)
		util.Success("Once completed, your configuration will be written to %s\n", app.ConfigPath)
	}
}

// PromptForConfig goes through a set of prompts to receive user input and generate an Config struct.
func (app *DdevApp) PromptForConfig() error {

	app.WarnIfConfigReplace()

	for {
		err := app.promptForName()

		if err == nil {
			break
		}

		output.UserOut.Printf("%v", err)
	}

	if err := app.docrootPrompt(); err != nil {
		return err
	}

	err := app.AppTypePrompt()
	if err != nil {
		return err
	}

	err = app.ConfigFileOverrideAction()
	if err != nil {
		return err
	}

	err = app.providerInstance.PromptForConfig()

	return err
}

// ValidateConfig ensures the configuration meets ddev's requirements.
func (app *DdevApp) ValidateConfig() error {
	provider, err := app.GetProvider()
	if err != nil {
		return err.(invalidProvider)
	}

	// validate project name
	if err = provider.ValidateField("Name", app.Name); err != nil {
		return err.(invalidAppName)
	}

	// validate hostnames
	for _, hn := range app.GetHostnames() {
		if !hostRegex.MatchString(hn) {
			return fmt.Errorf("invalid hostname: %s. See https://en.wikipedia.org/wiki/Hostname#Restrictions_on_valid_hostnames for valid hostname requirements", hn).(invalidHostname)
		}
	}

	// validate apptype
	if !IsValidAppType(app.Type) {
		return fmt.Errorf("invalid app type: %s", app.Type).(invalidAppType)
	}

	// validate PHP version
	if !nodeps.IsValidPHPVersion(app.PHPVersion) {
		return fmt.Errorf("invalid PHP version: %s, must be one of %v", app.PHPVersion, nodeps.GetValidPHPVersions()).(invalidPHPVersion)
	}

	// validate webserver type
	if !nodeps.IsValidWebserverType(app.WebserverType) {
		return fmt.Errorf("invalid webserver type: %s, must be one of %s", app.WebserverType, nodeps.GetValidWebserverTypes()).(invalidWebserverType)
	}

	if !nodeps.IsValidOmitContainers(app.OmitContainers) {
		return fmt.Errorf("invalid omit_containers: %s, must be one of %s", app.OmitContainers, nodeps.GetValidOmitContainers()).(InvalidOmitContainers)
	}

	if app.MariaDBVersion != "" {
		// Validate mariadb version version
		if !nodeps.IsValidMariaDBVersion(app.MariaDBVersion) {
			return fmt.Errorf("invalid mariadb_version: %s, must be one of %s", app.MariaDBVersion, nodeps.GetValidMariaDBVersions()).(invalidMariaDBVersion)
		}
	}
	if app.MySQLVersion != "" {
		// Validate /mysql version
		if !nodeps.IsValidMySQLVersion(app.MySQLVersion) {
			return fmt.Errorf("invalid mysql_version: %s, must be one of %s", app.MySQLVersion, nodeps.GetValidMySQLVersions()).(invalidMySQLVersion)
		}
	}

	// Validate db versions
	if app.MariaDBVersion != "" && app.MySQLVersion != "" {
		return fmt.Errorf("both mariadb_version (%v) and mysql_version (%v) are set, but they are mutually exclusive", app.MariaDBVersion, app.MySQLVersion)
	}

	// golang on windows is not able to time.LoadLocation unless
	// go is installed... so skip validation on Windows
	if runtime.GOOS != "windows" {
		_, err = time.LoadLocation(app.Timezone)
		if err != nil {
			// golang on Windows is often not able to time.LoadLocation.
			// It often works if go is installed and $GOROOT is set, but
			// that's not the norm for our users.
			return fmt.Errorf("invalid timezone %s: %v", app.Timezone, err)
		}
	}

	return nil
}

// DockerComposeYAMLPath returns the absolute path to where the
// docker-compose.yaml should exist for this app.
func (app *DdevApp) DockerComposeYAMLPath() string {
	return app.GetConfigPath("docker-compose.yaml")
}

// GetHostname returns the primary hostname of the app.
func (app *DdevApp) GetHostname() string {
	return app.Name + "." + app.ProjectTLD
}

// GetHostnames returns an array of all the configured hostnames.
func (app *DdevApp) GetHostnames() []string {

	// Use a map to make sure that we have unique hostnames
	// The value is useless, so just use the int 1 for assignment.
	nameListMap := make(map[string]int)

	for _, name := range app.AdditionalHostnames {
		nameListMap[name+"."+app.ProjectTLD] = 1
	}

	for _, name := range app.AdditionalFQDNs {
		nameListMap[name] = 1
	}

	// Now walk the map and extract the keys into an array.
	nameListArray := make([]string, 0, len(nameListMap))
	for k := range nameListMap {
		nameListArray = append(nameListArray, k)
	}
	sort.Strings(nameListArray)
	// We want the primary hostname to be first in the list.
	nameListArray = append([]string{app.GetHostname()}, nameListArray...)

	return nameListArray
}

// WriteDockerComposeConfig writes a docker-compose.yaml to the app configuration directory.
func (app *DdevApp) WriteDockerComposeConfig() error {
	var err error

	if fileutil.FileExists(app.DockerComposeYAMLPath()) {
		found, err := fileutil.FgrepStringInFile(app.DockerComposeYAMLPath(), DdevFileSignature)
		util.CheckErr(err)

		// If we did *not* find the ddev file signature in docker-compose.yaml, we'll back it up and warn about it.
		if !found {
			util.Warning("User-managed docker-compose.yaml will be replaced with ddev-generated docker-compose.yaml. Original file will be placed in docker-compose.yaml.bak")
			_ = os.Remove(app.DockerComposeYAMLPath() + ".bak")
			err = os.Rename(app.DockerComposeYAMLPath(), app.DockerComposeYAMLPath()+".bak")
			util.CheckErr(err)
		}
	}

	f, err := os.Create(app.DockerComposeYAMLPath())
	if err != nil {
		return err
	}
	defer util.CheckClose(f)

	rendered, err := app.RenderComposeYAML()
	if err != nil {
		return err
	}
	_, err = f.WriteString(rendered)
	if err != nil {
		return err
	}
	return err
}

// CheckCustomConfig warns the user if any custom configuration files are in use.
func (app *DdevApp) CheckCustomConfig() {

	// Get the path to .ddev for the current app.
	ddevDir := filepath.Dir(app.ConfigPath)

	customConfig := false
	if _, err := os.Stat(filepath.Join(ddevDir, "nginx-site.conf")); err == nil && app.WebserverType == nodeps.WebserverNginxFPM {
		util.Warning("Using custom nginx configuration in nginx-site.conf")
		customConfig = true
	}

	if _, err := os.Stat(filepath.Join(ddevDir, "apache", "apache-site.conf")); err == nil && app.WebserverType != nodeps.WebserverNginxFPM {
		util.Warning("Using custom apache configuration in apache/apache-site.conf")
		customConfig = true
	}

	nginxPath := filepath.Join(ddevDir, "nginx")
	if _, err := os.Stat(nginxPath); err == nil {
		nginxFiles, err := filepath.Glob(nginxPath + "/*.conf")
		util.CheckErr(err)
		if len(nginxFiles) > 0 {
			util.Warning("Using custom nginx partial configuration: %v", nginxFiles)
			customConfig = true
		}
	}

	mysqlPath := filepath.Join(ddevDir, "mysql")
	if _, err := os.Stat(mysqlPath); err == nil {
		mysqlFiles, err := filepath.Glob(mysqlPath + "/*.cnf")
		util.CheckErr(err)
		if len(mysqlFiles) > 0 {
			util.Warning("Using custom mysql configuration: %v", mysqlFiles)
			customConfig = true
		}
	}

	phpPath := filepath.Join(ddevDir, "php")
	if _, err := os.Stat(phpPath); err == nil {
		phpFiles, err := filepath.Glob(phpPath + "/*.ini")
		util.CheckErr(err)
		if len(phpFiles) > 0 {
			util.Warning("Using custom PHP configuration: %v", phpFiles)
			customConfig = true
		}
	}
	if customConfig {
		util.Warning("Custom configuration takes effect when container is created,\nusually on start, use 'ddev restart' if you're not seeing it take effect.")
	}

}

type composeYAMLVars struct {
	Name                 string
	Plugin               string
	AppType              string
	MailhogPort          string
	DBAPort              string
	DBPort               string
	DdevGenerated        string
	HostDockerInternalIP string
	ComposeVersion       string
	MountType            string
	WebMount             string
	WebBuildContext      string
	DBBuildContext       string
	SSHAgentBuildContext string
	OmitDB               bool
	OmitDBA              bool
	OmitSSHAgent         bool
	NFSMountEnabled      bool
	NFSSource            string
	DockerIP             string
	IsWindowsFS          bool
	Hostnames            []string
	Timezone             string
	Username             string
	UID                  string
	GID                  string
}

// RenderComposeYAML renders the contents of docker-compose.yaml.
func (app *DdevApp) RenderComposeYAML() (string, error) {
	var doc bytes.Buffer
	var err error
	templ, err := template.New("compose template").Funcs(sprig.HtmlFuncMap()).Parse(DDevComposeTemplate)
	if err != nil {
		return "", err
	}
	templ, err = templ.Parse(DDevComposeTemplate)
	if err != nil {
		return "", err
	}

	hostDockerInternalIP, err := dockerutil.GetHostDockerInternalIP()
	if err != nil {
		util.Warning("Could not determine host.docker.internal IP address: %v", err)
	}

	// The fallthrough default for hostDockerInternalIdentifier is the
	// hostDockerInternalHostname == host.docker.internal

	uid, gid, username := util.GetContainerUIDGid()

	templateVars := composeYAMLVars{
		Name:                 app.Name,
		Plugin:               "ddev",
		AppType:              app.Type,
		MailhogPort:          GetPort("mailhog"),
		DBAPort:              GetPort("dba"),
		DBPort:               GetPort("db"),
		DdevGenerated:        DdevFileSignature,
		HostDockerInternalIP: hostDockerInternalIP,
		ComposeVersion:       version.DockerComposeFileFormatVersion,
		OmitDB:               nodeps.ArrayContainsString(app.OmitContainers, "db"),
		OmitDBA:              nodeps.ArrayContainsString(app.OmitContainers, "dba") || nodeps.ArrayContainsString(app.OmitContainers, "db"),
		OmitSSHAgent:         nodeps.ArrayContainsString(app.OmitContainers, "ddev-ssh-agent"),
		NFSMountEnabled:      app.NFSMountEnabled,
		NFSSource:            "",
		IsWindowsFS:          runtime.GOOS == "windows",
		MountType:            "bind",
		WebMount:             "../",
		Hostnames:            app.GetHostnames(),
		Timezone:             app.Timezone,
		Username:             username,
		UID:                  uid,
		GID:                  gid,
		WebBuildContext:      app.GetConfigPath(".webimageBuild"),
		DBBuildContext:       app.GetConfigPath(".dbimageBuild"),
	}
	if app.NFSMountEnabled {
		templateVars.MountType = "volume"
		templateVars.WebMount = "nfsmount"
		templateVars.NFSSource = app.AppRoot
		// Workaround for Catalina sharing nfs as /System/Volumes/Data
		if runtime.GOOS == "darwin" && fileutil.IsDirectory(filepath.Join("/System/Volumes/Data", app.AppRoot)) {
			templateVars.NFSSource = filepath.Join("/System/Volumes/Data", app.AppRoot)
		}
		if runtime.GOOS == "windows" {
			// WinNFSD can only handle a mountpoint like /C/Users/rfay/workspace/d8git
			// and completely chokes in C:\Users\rfay...
			templateVars.NFSSource = dockerutil.MassageWindowsNFSMount(app.AppRoot)
		}
	}

	// Add web and db extra dockerfile info
	// If there is a user-provided Dockerfile, use that as the base and then add
	// our extra stuff like usernames, etc.
	err = WriteBuildDockerfile(app.GetConfigPath(".webimageBuild/Dockerfile"), app.GetConfigPath("web-build/Dockerfile"), app.WebImageExtraPackages)
	if err != nil {
		return "", err
	}

	err = WriteBuildDockerfile(app.GetConfigPath(".dbimageBuild/Dockerfile"), app.GetConfigPath("db-build/Dockerfile"), app.DBImageExtraPackages)

	if err != nil {
		return "", err
	}

	// SSH agent just needs extra to add the official related user, nothing else
	err = WriteBuildDockerfile(app.GetConfigPath(".sshimageBuild/Dockerfile"), "", nil)
	if err != nil {
		return "", err
	}

	templateVars.DockerIP, err = dockerutil.GetDockerIP()
	if err != nil {
		return "", err
	}

	err = templ.Execute(&doc, templateVars)
	return doc.String(), err
}

// WriteBuildDockerfile writes a Dockerfile to be used in the
// docker-compose 'build'
// It may include the contents of .ddev/<container>-build
func WriteBuildDockerfile(fullpath string, userDockerfile string, extraPackages []string) error {
	// Start with user-built dockerfile if there is one.
	err := os.MkdirAll(filepath.Dir(fullpath), 0755)
	if err != nil {
		return err
	}

	// Normal starting content is just the arg and base image
	contents := `
ARG BASE_IMAGE
FROM $BASE_IMAGE
`
	// If there is a user dockerfile, start with its contents
	if userDockerfile != "" && fileutil.FileExists(userDockerfile) {
		contents, err = fileutil.ReadFileIntoString(userDockerfile)
		if err != nil {
			return err
		}
	}
	contents = contents + `
ARG username
ARG uid
ARG gid
RUN (groupadd --gid $gid "$username" || groupadd "$username" || true) && (useradd  -l -m -s "/bin/bash" --gid "$username" --comment '' --uid $uid "$username" || useradd  -l -m -s "/bin/bash" --gid "$username" --comment '' "$username")
 `
	if extraPackages != nil {
		contents = contents + `
RUN apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y -o Dpkg::Options::="--force-confnew" --no-install-recommends --no-install-suggests ` + strings.Join(extraPackages, " ") + "\n"
	}
	return WriteImageDockerfile(fullpath, []byte(contents))
}

// WriteImageDockerfile writes a dockerfile at the fullpath (including the filename)
func WriteImageDockerfile(fullpath string, contents []byte) error {
	err := os.MkdirAll(filepath.Dir(fullpath), 0755)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(fullpath, contents, 0644)
	if err != nil {
		return err
	}
	return nil
}

// prompt for a project name.
func (app *DdevApp) promptForName() error {
	provider, err := app.GetProvider()
	if err != nil {
		return err
	}

	if app.Name == "" {
		dir, err := os.Getwd()
		// if working directory name is invalid for hostnames, we shouldn't suggest it
		if err == nil && hostRegex.MatchString(filepath.Base(dir)) {

			app.Name = filepath.Base(dir)
		}
	}

	app.Name = util.Prompt("Project name", app.Name)
	return provider.ValidateField("Name", app.Name)
}

// AvailableDocrootLocations returns an of default docroot locations to look for.
func AvailableDocrootLocations() []string {
	return []string{
		"web/public",
		"web",
		"docroot",
		"htdocs",
		"_www",
		"public",
		"pub",
	}
}

// DiscoverDefaultDocroot returns the default docroot directory.
func DiscoverDefaultDocroot(app *DdevApp) string {
	// Provide use the app.Docroot as the default docroot option.
	var defaultDocroot = app.Docroot
	if defaultDocroot == "" {
		for _, docroot := range AvailableDocrootLocations() {
			if _, err := os.Stat(filepath.Join(app.AppRoot, docroot)); err != nil {
				continue
			}

			if fileutil.FileExists(filepath.Join(app.AppRoot, docroot, "index.php")) {
				defaultDocroot = docroot
				break
			}
		}
	}
	return defaultDocroot
}

// Determine the document root.
func (app *DdevApp) docrootPrompt() error {
	provider, err := app.GetProvider()
	if err != nil {
		return err
	}

	// Determine the document root.
	util.Warning("\nThe docroot is the directory from which your site is served.\nThis is a relative path from your project root at %s", app.AppRoot)
	output.UserOut.Println("You may leave this value blank if your site files are in the project root")
	var docrootPrompt = "Docroot Location"
	var defaultDocroot = DiscoverDefaultDocroot(app)
	// If there is a default docroot, display it in the prompt.
	if defaultDocroot != "" {
		docrootPrompt = fmt.Sprintf("%s (%s)", docrootPrompt, defaultDocroot)
	} else if cd, _ := os.Getwd(); cd == filepath.Join(app.AppRoot, defaultDocroot) {
		// Preserve the case where the docroot is the current directory
		docrootPrompt = fmt.Sprintf("%s (current directory)", docrootPrompt)
	} else {
		// Explicitly state 'project root' when in a subdirectory
		docrootPrompt = fmt.Sprintf("%s (project root)", docrootPrompt)
	}

	fmt.Print(docrootPrompt + ": ")
	app.Docroot = util.GetInput(defaultDocroot)

	// Ensure the docroot exists. If it doesn't, prompt the user to verify they entered it correctly.
	fullPath := filepath.Join(app.AppRoot, app.Docroot)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		util.Warning("Warning: the provided docroot at %s does not currently exist.", fullPath)

		// Ask the user for permission to create the docroot
		if !util.Confirm(fmt.Sprintf("Create docroot at %s?", fullPath)) {
			return fmt.Errorf("docroot must exist to continue configuration")
		}

		if err = os.MkdirAll(fullPath, 0755); err != nil {
			return fmt.Errorf("unable to create docroot: %v", err)
		}

		util.Success("Created docroot at %s.", fullPath)
	}

	return provider.ValidateField("Docroot", app.Docroot)
}

// ConfigExists determines if a ddev config file exists for this application.
func (app *DdevApp) ConfigExists() bool {
	if _, err := os.Stat(app.ConfigPath); os.IsNotExist(err) {
		return false
	}
	return true
}

// AppTypePrompt handles the Type workflow.
func (app *DdevApp) AppTypePrompt() error {
	provider, err := app.GetProvider()
	if err != nil {
		return err
	}
	validAppTypes := strings.Join(GetValidAppTypes(), ", ")
	typePrompt := fmt.Sprintf("Project Type [%s]", validAppTypes)

	// First, see if we can auto detect what kind of site it is so we can set a sane default.

	detectedAppType := app.DetectAppType()
	// If the detected detectedAppType is php, we'll ask them to confirm,
	// otherwise go with it.
	// If we found an application type just set it and inform the user.
	util.Success("Found a %s codebase at %s.", detectedAppType, filepath.Join(app.AppRoot, app.Docroot))
	typePrompt = fmt.Sprintf("%s (%s)", typePrompt, detectedAppType)

	fmt.Printf(typePrompt + ": ")
	appType := strings.ToLower(util.GetInput(detectedAppType))

	for !IsValidAppType(appType) {
		output.UserOut.Errorf("'%s' is not a valid project type. Allowed project types are: %s\n", appType, validAppTypes)

		fmt.Printf(typePrompt + ": ")
		appType = strings.ToLower(util.GetInput(appType))
	}
	app.Type = appType
	return provider.ValidateField("Type", app.Type)
}

// PrepDdevDirectory creates a .ddev directory in the current working directory
func PrepDdevDirectory(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {

		log.WithFields(log.Fields{
			"directory": dir,
		}).Debug("Config Directory does not exist, attempting to create.")

		err := os.MkdirAll(dir, 0755)
		if err != nil {
			return err
		}
	}

	err := CreateGitIgnore(dir, "commands/*/*.example", "commands/*/README.txt", "commands/host/launch", "commands/db/mysql", "homeadditions/*.example", "homeadditions/README.txt", ".gitignore", "import.yaml", "docker-compose.yaml", "db_snapshots", "sequelpro.spf", "import-db", "config.*.y*ml", ".webimageBuild", ".dbimageBuild", ".sshimageBuild", ".webimageExtra", ".dbimageExtra", "*-build/Dockerfile.example")
	if err != nil {
		return fmt.Errorf("failed to create gitignore in %s: %v", dir, err)
	}

	return nil
}

// validateHookYAML validates command hooks and tasks defined in hooks for config.yaml
func validateHookYAML(source []byte) error {
	validHooks := []string{
		"pre-start",
		"post-start",
		"pre-import-db",
		"post-import-db",
		"pre-import-files",
		"post-import-files",
		"pre-composer",
		"post-composer",
		"pre-stop",
		"post-stop",
		"pre-config",
		"post-config",
		"pre-describe",
		"post-describe",
		"pre-exec",
		"post-exec",
		"pre-pause",
		"post-pause",
		"pre-pull",
		"post-pull",
		"pre-snapshot",
		"post-snapshot",
		"pre-restore-snapshot",
		"post-restore-snapshot",
	}

	validTasks := []string{
		"exec",
		"exec-host",
		"composer",
	}

	type Validate struct {
		Commands map[string][]map[string]interface{} `yaml:"hooks,omitempty"`
	}
	val := &Validate{}

	err := yaml.Unmarshal(source, val)
	if err != nil {
		return err
	}

	for foundHook, tasks := range val.Commands {
		var match bool
		for _, h := range validHooks {
			if foundHook == h {
				match = true
			}
		}
		if !match {
			return fmt.Errorf("invalid hook %s defined in config.yaml", foundHook)
		}

		for _, foundTask := range tasks {
			var match bool
			for _, validTaskName := range validTasks {
				if _, ok := foundTask[validTaskName]; ok {
					match = true
				}
			}
			if !match {
				return fmt.Errorf("invalid task '%s' defined for hook %s in config.yaml", foundTask, foundHook)
			}

		}

	}

	return nil
}
