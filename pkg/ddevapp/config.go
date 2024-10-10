package ddevapp

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/ddev/ddev/pkg/config/types"
	"github.com/ddev/ddev/pkg/docker"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/ddev/ddev/pkg/versionconstants"
	copy2 "github.com/otiai10/copy"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// Regexp pattern to determine if a hostname is valid per RFC 1123.
var hostRegex = regexp.MustCompile(`^(([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]*[a-zA-Z0-9])\.)*([A-Za-z0-9]|[A-Za-z0-9][A-Za-z0-9\-]*[A-Za-z0-9])$`)

// RunValidateConfig controls whether to run ValidateConfig() function.
// In some cases we don't actually need to check the config, e.g. when deleting the project.
// It is enabled by default.
var RunValidateConfig = true

// init() is for testing situations only, allowing us to override the default webserver type
// or caching behavior
func init() {
	var err error
	// This is for automated testing only. It allows us to override the webserver type.
	if testWebServerType := os.Getenv("DDEV_TEST_WEBSERVER_TYPE"); testWebServerType != "" {
		nodeps.WebserverDefault = testWebServerType
	}
	if testNFSMount := os.Getenv("DDEV_TEST_USE_NFSMOUNT"); testNFSMount == "true" {
		nodeps.PerformanceModeDefault = types.PerformanceModeNFS
	}
	if testMutagen := os.Getenv("DDEV_TEST_USE_MUTAGEN"); testMutagen == "true" {
		nodeps.PerformanceModeDefault = types.PerformanceModeMutagen
	}
	if os.Getenv("DDEV_TEST_NO_BIND_MOUNTS") == "true" {
		nodeps.NoBindMountsDefault = true
	}
	if os.Getenv("DDEV_TEST_USE_NGINX_PROXY_ROUTER") == "true" {
		nodeps.UseNginxProxyRouter = true
	}
	if g := os.Getenv("DDEV_TEST_GOROUTINE_LIMIT"); g != "" {
		nodeps.GoroutineLimit, err = strconv.Atoi(g)
		if err != nil {
			util.Failed("DDEV_TEST_GOROUTINE_LIMIT must be empty or numeric value, not '%v'", g)
		}
	}
}

// NewApp creates a new DdevApp struct with defaults set and overridden by any existing config.yml.
func NewApp(appRoot string, includeOverrides bool) (*DdevApp, error) {
	defer util.TimeTrackC(fmt.Sprintf("ddevapp.NewApp(%s)", appRoot))()

	app := &DdevApp{}

	if appRoot == "" {
		app.AppRoot, _ = os.Getwd()
	} else {
		app.AppRoot = appRoot
	}

	homeDir, _ := os.UserHomeDir()
	if appRoot == filepath.Dir(globalconfig.GetGlobalDdevDir()) || app.AppRoot == homeDir {
		return nil, fmt.Errorf("ddev config is not useful in your home directory (%s)", homeDir)
	}

	if _, err := os.Stat(app.AppRoot); err != nil {
		return app, err
	}

	app.ConfigPath = app.GetConfigPath("config.yaml")
	app.Type = nodeps.AppTypeNone
	app.PHPVersion = nodeps.PHPDefault
	app.ComposerVersion = nodeps.ComposerDefault
	app.NodeJSVersion = nodeps.NodeJSDefault
	app.WebserverType = nodeps.WebserverDefault
	app.SetPerformanceMode(nodeps.PerformanceModeDefault)

	// Turn off Mutagen on Python projects until initial setup can be done
	if app.WebserverType == nodeps.WebserverNginxGunicorn {
		app.SetPerformanceMode(types.PerformanceModeNone)
	}

	app.FailOnHookFail = nodeps.FailOnHookFailDefault
	app.FailOnHookFailGlobal = globalconfig.DdevGlobalConfig.FailOnHookFailGlobal

	// Provide a default app name based on directory name
	app.Name = NormalizeProjectName(filepath.Base(app.AppRoot))

	// Gather containers to omit, adding ddev-router for gitpod/codespaces
	app.OmitContainersGlobal = globalconfig.DdevGlobalConfig.OmitContainersGlobal
	if nodeps.IsGitpod() || nodeps.IsCodespaces() {
		app.OmitContainersGlobal = append(app.OmitContainersGlobal, "ddev-router")
	}

	app.ProjectTLD = globalconfig.DdevGlobalConfig.ProjectTldGlobal
	if globalconfig.DdevGlobalConfig.ProjectTldGlobal == "" {
		app.ProjectTLD = nodeps.DdevDefaultTLD
	}
	app.UseDNSWhenPossible = true

	app.WebImage = docker.GetWebImage()

	// Load from file if available. This will return an error if the file doesn't exist,
	// and it is up to the caller to determine if that's an issue.
	if _, err := os.Stat(app.ConfigPath); !os.IsNotExist(err) {
		_, err = app.ReadConfig(includeOverrides)
		if err != nil {
			return app, fmt.Errorf("%v exists but cannot be read. It may be invalid due to a syntax error: %v", app.ConfigPath, err)
		}
	}

	// Upgrade any pre-v1.19.0 config that has mariadb_version or mysql_version
	if app.MariaDBVersion != "" {
		app.Database = DatabaseDesc{Type: nodeps.MariaDB, Version: app.MariaDBVersion}
		app.MariaDBVersion = ""
	}
	if app.MySQLVersion != "" {
		app.Database = DatabaseDesc{Type: nodeps.MySQL, Version: app.MySQLVersion}
		app.MySQLVersion = ""
	}
	if app.Database.Type == "" {
		app.Database = DatabaseDefault
	}

	if app.DefaultContainerTimeout == "" {
		app.DefaultContainerTimeout = nodeps.DefaultDefaultContainerTimeout
		// On Windows the default timeout may be too short for mutagen to succeed.
		if runtime.GOOS == "windows" {
			app.DefaultContainerTimeout = "240"
		}
	}

	// Migrate UploadDir to UploadDirs
	if app.UploadDirDeprecated != "" {
		uploadDirDeprecated := app.UploadDirDeprecated
		app.UploadDirDeprecated = ""
		app.addUploadDir(uploadDirDeprecated)
	}

	// Remove dba
	if nodeps.ArrayContainsString(app.OmitContainers, "dba") || nodeps.ArrayContainsString(app.OmitContainersGlobal, "dba") {
		app.OmitContainers = nodeps.RemoveItemFromSlice(app.OmitContainers, "dba")
		app.OmitContainersGlobal = nodeps.RemoveItemFromSlice(app.OmitContainersGlobal, "dba")
	}

	app.SetApptypeSettingsPaths()

	// Rendered yaml is not there until after ddev config or ddev start
	if fileutil.FileExists(app.ConfigPath) && fileutil.FileExists(app.DockerComposeFullRenderedYAMLPath()) {
		content, err := fileutil.ReadFileIntoString(app.DockerComposeFullRenderedYAMLPath())
		if err != nil {
			return app, err
		}
		err = app.UpdateComposeYaml(content)
		if err != nil {
			return app, err
		}
	}

	// If non-php type, use non-php webserver type
	if app.WebserverType == nodeps.WebserverDefault && app.Type == nodeps.AppTypeDjango4 {
		app.WebserverType = nodeps.WebserverNginxGunicorn
	}

	// TODO: Enable once the bootstrap is clean and every project is loaded once only
	//app.TrackProject()

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

	// Only set the images on write if non-default values have been specified.
	if appcopy.WebImage == docker.GetWebImage() {
		appcopy.WebImage = ""
	}
	if appcopy.RouterHTTPPort == nodeps.DdevDefaultRouterHTTPPort {
		appcopy.RouterHTTPPort = ""
	}
	if appcopy.RouterHTTPSPort == nodeps.DdevDefaultRouterHTTPSPort {
		appcopy.RouterHTTPSPort = ""
	}
	if appcopy.MailpitHTTPPort == nodeps.DdevDefaultMailpitHTTPPort {
		appcopy.MailpitHTTPPort = ""
	}
	if appcopy.MailpitHTTPSPort == nodeps.DdevDefaultMailpitHTTPSPort {
		appcopy.MailpitHTTPSPort = ""
	}
	if appcopy.ProjectTLD == globalconfig.DdevGlobalConfig.ProjectTldGlobal {
		appcopy.ProjectTLD = ""
	}
	if appcopy.DefaultContainerTimeout == nodeps.DefaultDefaultContainerTimeout {
		appcopy.DefaultContainerTimeout = ""
	}

	if appcopy.NodeJSVersion == nodeps.NodeJSDefault {
		appcopy.NodeJSVersion = ""
	}

	// Ensure valid type
	if appcopy.Type == nodeps.AppTypeNone {
		appcopy.Type = nodeps.AppTypePHP
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

	err = PrepDdevDirectory(&appcopy)
	if err != nil {
		return err
	}

	cfgbytes, err := yaml.Marshal(appcopy)
	if err != nil {
		return err
	}

	// Append hook information and sample hook suggestions.
	cfgbytes = append(cfgbytes, []byte(ConfigInstructions)...)
	cfgbytes = append(cfgbytes, appcopy.GetHookDefaultComments()...)

	err = os.WriteFile(appcopy.ConfigPath, cfgbytes, 0644)
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
#ddev-generated
# You can copy this Dockerfile.example to Dockerfile to add configuration
# or packages or anything else to your webimage
# These additions will be appended last to ddev's own Dockerfile
RUN npm install --global forever
RUN echo "Built on $(date)" > /build-date.txt
`)

	err = WriteImageDockerfile(app.GetConfigPath("web-build")+"/Dockerfile.example", contents)
	if err != nil {
		return err
	}
	contents = []byte(`
#ddev-generated
# You can copy this Dockerfile.example to Dockerfile to add configuration
# or packages or anything else to your dbimage
RUN echo "Built on $(date)" > /build-date.txt
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
// Checks that configured host ports are not already
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
// returns the list of config files read
func (app *DdevApp) ReadConfig(includeOverrides bool) ([]string, error) {

	// Load base .ddev/config.yaml - original config
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
			err = app.mergeAdditionalConfigIntoApp(item)

			if err != nil {
				return []string{}, fmt.Errorf("unable to load config file %s: %v", item, err)
			}
		}
	}

	return append([]string{app.ConfigPath}, configOverrides...), nil
}

// LoadConfigYamlFile loads one config.yaml into app, overriding what might be there.
func (app *DdevApp) LoadConfigYamlFile(filePath string) error {
	source, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("could not find an active DDEV configuration at %s have you run 'ddev config'? %v", app.ConfigPath, err)
	}

	// Validate extend command keys
	err = validateHookYAML(source)
	if err != nil {
		return fmt.Errorf("invalid configuration in %s: %v", app.ConfigPath, err)
	}

	// ReadConfig config values from file.
	err = yaml.Unmarshal(source, app)
	if err != nil {
		return err
	}

	// Handle UploadDirs value which can take multiple types.
	err = app.validateUploadDirs()
	if err != nil {
		return err
	}

	return nil
}

// WarnIfConfigReplace messages user about whether config is being replaced or created
func (app *DdevApp) WarnIfConfigReplace() {
	if app.ConfigExists() {
		util.Warning("You are reconfiguring the project at %s.\nThe existing configuration will be updated and replaced.", app.AppRoot)
	} else {
		util.Success("Creating a new DDEV project config in the current directory (%s)", app.AppRoot)
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

	err = app.ConfigFileOverrideAction(false)
	if err != nil {
		return err
	}

	err = app.ValidateConfig()
	if err != nil {
		return err
	}

	return nil
}

// ValidateProjectName checks to see if the project name works for a proper hostname
func ValidateProjectName(name string) error {
	match := hostRegex.MatchString(name)
	if !match {
		return fmt.Errorf("%s is not a valid project name. Please enter a project name in your configuration that will allow for a valid hostname. See https://en.wikipedia.org/wiki/Hostname#Syntax for valid hostname requirements", name)
	}
	return nil
}

// ValidateConfig ensures the configuration meets ddev's requirements.
func (app *DdevApp) ValidateConfig() error {
	// Skip project validation on request.
	if !RunValidateConfig {
		return nil
	}

	// Validate ddev version constraint, if any
	if app.DdevVersionConstraint != "" {
		err := CheckDdevVersionConstraint(app.DdevVersionConstraint, fmt.Sprintf("unable to start the '%s' project", app.Name), "or update the `ddev_version_constraint` in your .ddev/config.yaml file")
		if err != nil {
			return err
		}
	}

	// Validate project name
	if err := ValidateProjectName(app.Name); err != nil {
		return err
	}

	// Skip any validation below this check if there is nothing to validate
	if err := CheckForMissingProjectFiles(app); err != nil {
		// Do not return an error here because not all DDEV commands should be stopped by this check
		// It matters when you start a project, but not when you stop or delete it
		// This check is reused elsewhere where appropriate
		return nil
	}

	// Validate hostnames
	for _, hn := range app.GetHostnames() {
		// If they have provided "*.<hostname>" then ignore the *. part.
		hn = strings.TrimPrefix(hn, "*.")
		if hn == nodeps.DdevDefaultTLD {
			return fmt.Errorf("wildcarding the full hostname\nor using 'ddev.site' as FQDN for the project %s is not allowed\nbecause other projects would not work in that case", app.Name)
		}
		if !hostRegex.MatchString(hn) {
			return fmt.Errorf("the %s project has an invalid hostname: '%s', see https://en.wikipedia.org/wiki/Hostname#Syntax for valid hostname requirements", app.Name, hn).(invalidHostname)
		}
	}

	// Validate apptype
	if !IsValidAppType(app.Type) {
		return fmt.Errorf("the %s project has an invalid app type: %s", app.Name, app.Type).(invalidAppType)
	}

	// Validate PHP version
	if !nodeps.IsValidPHPVersion(app.PHPVersion) {
		return fmt.Errorf("the %s project has an unsupported PHP version: %s, DDEV only supports the following versions: %v", app.Name, app.PHPVersion, nodeps.GetValidPHPVersions()).(invalidPHPVersion)
	}

	// Validate webserver type
	if !nodeps.IsValidWebserverType(app.WebserverType) {
		return fmt.Errorf("the %s project has an unsupported webserver type: %s, DDEV (%s) only supports the following webserver types: %s", app.Name, app.WebserverType, runtime.GOARCH, nodeps.GetValidWebserverTypes()).(invalidWebserverType)
	}

	if !nodeps.IsValidOmitContainers(app.OmitContainers) {
		return fmt.Errorf("the %s project has an unsupported omit_containers: %s, DDEV (%s) only supports the following for omit_containers: %s", app.Name, app.OmitContainers, runtime.GOARCH, nodeps.GetValidOmitContainers()).(InvalidOmitContainers)
	}

	if !nodeps.IsValidDatabaseVersion(app.Database.Type, app.Database.Version) {
		return fmt.Errorf("the %s project has an unsupported database type/version: '%s:%s', DDEV %s only supports the following database types and versions: mariadb: %v, mysql: %v, postgres: %v", app.Name, app.Database.Type, app.Database.Version, runtime.GOARCH, nodeps.GetValidMariaDBVersions(), nodeps.GetValidMySQLVersions(), nodeps.GetValidPostgresVersions())
	}

	// This check is too intensive for app.Init() and ddevapp.GetActiveApp(), slows things down dramatically
	// If the database already exists in volume and is not of this type, then throw an error
	// if !nodeps.ArrayContainsString(app.GetOmittedContainers(), "db") {
	// 	if dbType, err := app.GetExistingDBType(); err != nil || (dbType != "" && dbType != app.Database.Type+":"+app.Database.Version) {
	// 		return fmt.Errorf("unable to configure project %s with database type %s because that database type does not match the current actual database. Please change your database type back to %s and start again, export, delete, and then change configuration and start. To get back to existing type use 'ddev config --database=%s', see docs at %s", app.Name, dbType, dbType, dbType, "https://ddev.readthedocs.io/en/stable/users/extend/database-types/")
	// 	}
	// }

	// Golang on Windows is not able to time.LoadLocation unless
	// Go is installed... so skip validation on Windows
	if runtime.GOOS != "windows" {
		_, err := time.LoadLocation(app.Timezone)
		if err != nil {
			// Golang on Windows is often not able to time.LoadLocation.
			// It often works if go is installed and $GOROOT is set, but
			// that's not the norm for our users.
			return fmt.Errorf("the %s project has an invalid timezone %s: %v", app.Name, app.Timezone, err)
		}
	}

	return nil
}

// DockerComposeYAMLPath returns the absolute path to where the
// base generated yaml file should exist for this project.
func (app *DdevApp) DockerComposeYAMLPath() string {
	return app.GetConfigPath(".ddev-docker-compose-base.yaml")
}

// DockerComposeFullRenderedYAMLPath returns the absolute path to where the
// the complete generated yaml file should exist for this project.
func (app *DdevApp) DockerComposeFullRenderedYAMLPath() string {
	return app.GetConfigPath(".ddev-docker-compose-full.yaml")
}

// GetHostname returns the primary hostname of the app.
func (app *DdevApp) GetHostname() string {
	return strings.ToLower(app.Name) + "." + app.ProjectTLD
}

// GetHostnames returns a slice of all the configured hostnames.
func (app *DdevApp) GetHostnames() []string {

	// Use a map to make sure that we have unique hostnames
	// The value is useless, so use the int 1 for assignment.
	nameListMap := make(map[string]int)
	nameListArray := []string{}

	if !IsRouterDisabled(app) {
		for _, name := range app.AdditionalHostnames {
			name = strings.ToLower(name)
			nameListMap[name+"."+app.ProjectTLD] = 1
		}

		for _, name := range app.AdditionalFQDNs {
			name = strings.ToLower(name)
			nameListMap[name] = 1
		}

		// Make sure the primary hostname didn't accidentally get added, it will be prepended
		delete(nameListMap, app.GetHostname())

		// Now walk the map and extract the keys into an array.
		for k := range nameListMap {
			nameListArray = append(nameListArray, k)
		}
		sort.Strings(nameListArray)
		// We want the primary hostname to be first in the list.
		nameListArray = append([]string{app.GetHostname()}, nameListArray...)
	}
	return nameListArray
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
	nginxFullConfigPath := app.GetConfigPath("nginx_full/nginx-site.conf")
	sigFound, _ := fileutil.FgrepStringInFile(nginxFullConfigPath, nodeps.DdevFileSignature)
	if !sigFound && app.WebserverType == nodeps.WebserverNginxFPM {
		util.Warning("Using custom nginx configuration in %s", nginxFullConfigPath)
		customConfig = true
	}

	apacheFullConfigPath := app.GetConfigPath("apache/apache-site.conf")
	sigFound, _ = fileutil.FgrepStringInFile(apacheFullConfigPath, nodeps.DdevFileSignature)
	if !sigFound && app.WebserverType != nodeps.WebserverNginxFPM {
		util.Warning("Using custom apache configuration in %s", apacheFullConfigPath)
		customConfig = true
	}

	nginxPath := filepath.Join(ddevDir, "nginx")
	if _, err := os.Stat(nginxPath); err == nil {
		nginxFiles, err := filepath.Glob(nginxPath + "/*.conf")
		util.CheckErr(err)
		if len(nginxFiles) > 0 {
			printableFiles, _ := util.ArrayToReadableOutput(nginxFiles)
			util.Warning("Using nginx snippets: %v", printableFiles)
			customConfig = true
		}
	}

	mysqlPath := filepath.Join(ddevDir, "mysql")
	if _, err := os.Stat(mysqlPath); err == nil {
		mysqlFiles, err := filepath.Glob(mysqlPath + "/*.cnf")
		util.CheckErr(err)
		if len(mysqlFiles) > 0 {
			printableFiles, _ := util.ArrayToReadableOutput(mysqlFiles)
			util.Warning("Using custom MySQL configuration: %v", printableFiles)
			customConfig = true
		}
	}

	phpPath := filepath.Join(ddevDir, "php")
	if _, err := os.Stat(phpPath); err == nil {
		phpFiles, err := filepath.Glob(phpPath + "/*.ini")
		util.CheckErr(err)
		if len(phpFiles) > 0 {
			printableFiles, _ := util.ArrayToReadableOutput(phpFiles)
			util.Warning("Using custom PHP configuration: %v", printableFiles)
			customConfig = true
		}
	}

	webEntrypointPath := filepath.Join(ddevDir, "web-entrypoint.d")
	if _, err := os.Stat(webEntrypointPath); err == nil {
		entrypointFiles, err := filepath.Glob(webEntrypointPath + "/*.sh")
		util.CheckErr(err)
		if len(entrypointFiles) > 0 {
			printableFiles, _ := util.ArrayToReadableOutput(entrypointFiles)
			util.Warning("Using custom web-entrypoint.d configuration: %v", printableFiles)
			customConfig = true
		}
	}

	if customConfig {
		util.Warning("Custom configuration is updated on restart.\nIf you don't see your custom configuration taking effect, run 'ddev restart'.")
	}

}

// CheckDeprecations warns the user if anything in use is deprecated.
func (app *DdevApp) CheckDeprecations() {

}

// FixObsolete removes files that may be obsolete, etc.
func (app *DdevApp) FixObsolete() {
	// Remove old in-project commands (which have been moved to global)
	for _, command := range []string{"db/mysql", "host/launch", "web/xdebug"} {
		cmdPath := app.GetConfigPath(filepath.Join("commands", command))
		signatureFound, err := fileutil.FgrepStringInFile(cmdPath, nodeps.DdevFileSignature)
		if err == nil && signatureFound {
			err = os.Remove(cmdPath)
			if err != nil {
				util.Warning("attempted to remove %s but failed, you may want to remove it manually: %v", cmdPath, err)
			}
		}
	}

	// Remove old provider/*.example as we migrate to not needing them.
	for _, providerFile := range []string{"acquia.yaml.example", "platform.yaml.example"} {
		providerFilePath := app.GetConfigPath(filepath.Join("providers", providerFile))
		err := os.Remove(providerFilePath)
		if err == nil {
			util.Success("Removed obsolete file %s", providerFilePath)
		}
	}

	// Remove old global commands
	for _, command := range []string{"host/yarn"} {
		cmdPath := filepath.Join(globalconfig.GetGlobalDdevDir(), "commands/", command)
		if _, err := os.Stat(cmdPath); err == nil {
			err1 := os.Remove(cmdPath)
			if err1 != nil {
				util.Warning("attempted to remove %s but failed, you may want to remove it manually: %v", cmdPath, err)
			}
		}
	}

	// Remove old router router-build Dockerfile, etc.
	for _, f := range []string{"Dockerfile", "traefik_healthcheck.sh"} {
		routerBuildPath := filepath.Join(globalconfig.GetGlobalDdevDir(), "router-build")

		item := filepath.Join(routerBuildPath, f)
		signatureFound, err := fileutil.FgrepStringInFile(item, nodeps.DdevFileSignature)
		if err == nil && signatureFound {
			err = os.Remove(item)
			if err != nil {
				util.Warning("attempted to remove %s but failed, you may want to remove it manually: %v", item, err)
			}
		}
	}

	// Remove old global traefik configuuration.
	for _, f := range []string{"static_config.yaml"} {
		traefikGlobalConfigPath := filepath.Join(globalconfig.GetGlobalDdevDir(), "traefik")

		item := filepath.Join(traefikGlobalConfigPath, f)
		signatureFound, err := fileutil.FgrepStringInFile(item, nodeps.DdevFileSignature)
		if err == nil && signatureFound {
			err = os.Remove(item)
			if err != nil {
				util.Warning("attempted to remove %s but failed, you may want to remove it manually: %v", item, err)
			}
		}
	}

	// Remove old .global_commands directory
	legacyCommandDir := app.GetConfigPath(".global_commands")
	if fileutil.IsDirectory(legacyCommandDir) {
		err := os.RemoveAll(legacyCommandDir)
		if err != nil {
			util.Warning("attempted to remove %s but failed, you may want to remove it manually: %v", legacyCommandDir, err)
		}
	}
}

type composeYAMLVars struct {
	Name                            string
	Plugin                          string
	AppType                         string
	MailpitPort                     string
	HostMailpitPort                 string
	DBType                          string
	DBVersion                       string
	DBMountDir                      string
	DBAPort                         string
	DBPort                          string
	DdevGenerated                   string
	HostDockerInternalIP            string
	NFSServerAddr                   string
	DisableSettingsManagement       bool
	MountType                       string
	WebMount                        string
	WebBuildContext                 string
	DBBuildContext                  string
	WebBuildDockerfile              string
	DBBuildDockerfile               string
	SSHAgentBuildContext            string
	OmitDB                          bool
	OmitDBA                         bool
	OmitRouter                      bool
	OmitSSHAgent                    bool
	BindAllInterfaces               bool
	MariaDBVolumeName               string
	PostgresVolumeName              string
	MutagenEnabled                  bool
	MutagenVolumeName               string
	NFSMountEnabled                 bool
	NFSSource                       string
	NFSMountVolumeName              string
	DockerIP                        string
	IsWindowsFS                     bool
	NoProjectMount                  bool
	Hostnames                       []string
	Timezone                        string
	ComposerVersion                 string
	Username                        string
	UID                             string
	GID                             string
	FailOnHookFail                  bool
	WebWorkingDir                   string
	DBWorkingDir                    string
	DBAWorkingDir                   string
	WebEnvironment                  []string
	NoBindMounts                    bool
	Docroot                         string
	UploadDirsMap                   []string
	GitDirMount                     bool
	IsGitpod                        bool
	IsCodespaces                    bool
	DefaultContainerTimeout         string
	UseHostDockerInternalExtraHosts bool
	WebExtraHTTPPorts               string
	WebExtraHTTPSPorts              string
	WebExtraExposedPorts            string
}

// RenderComposeYAML renders the contents of .ddev/.ddev-docker-compose*.
func (app *DdevApp) RenderComposeYAML() (string, error) {
	var doc bytes.Buffer
	var err error

	hostDockerInternalIP, err := dockerutil.GetHostDockerInternalIP()
	if err != nil {
		util.Warning("Could not determine host.docker.internal IP address: %v", err)
	}
	nfsServerAddr, err := dockerutil.GetNFSServerAddr()
	if err != nil {
		util.Warning("Could not determine NFS server IP address: %v", err)
	}

	// The fallthrough default for hostDockerInternalIdentifier is the
	// hostDockerInternalHostname == host.docker.internal

	webEnvironment := globalconfig.DdevGlobalConfig.WebEnvironment
	localWebEnvironment := app.WebEnvironment
	for _, v := range localWebEnvironment {
		// docker-compose won't accept a duplicate environment value
		if !nodeps.ArrayContainsString(webEnvironment, v) {
			webEnvironment = append(webEnvironment, v)
		}
	}

	uid, gid, username := util.GetContainerUIDGid()
	_, err = app.GetProvider("")
	if err != nil {
		return "", err
	}

	timezone := app.Timezone
	if timezone == "" {
		timezone, err = app.GetLocalTimezone()
		if err != nil {
			util.Debug("Unable to autodetect timezone: %v", err.Error())
		} else {
			util.Debug("Using automatically detected timezone: TZ=%s", timezone)
		}
	}

	templateVars := composeYAMLVars{
		Name:                      app.Name,
		Plugin:                    "ddev",
		AppType:                   app.Type,
		MailpitPort:               GetExposedPort(app, "mailpit"),
		HostMailpitPort:           app.HostMailpitPort,
		DBType:                    app.Database.Type,
		DBVersion:                 app.Database.Version,
		DBMountDir:                "/var/lib/mysql",
		DBPort:                    GetExposedPort(app, "db"),
		DdevGenerated:             nodeps.DdevFileSignature,
		HostDockerInternalIP:      hostDockerInternalIP,
		NFSServerAddr:             nfsServerAddr,
		DisableSettingsManagement: app.DisableSettingsManagement,
		OmitDB:                    nodeps.ArrayContainsString(app.GetOmittedContainers(), nodeps.DBContainer),
		OmitRouter:                nodeps.ArrayContainsString(app.GetOmittedContainers(), globalconfig.DdevRouterContainer),
		OmitSSHAgent:              nodeps.ArrayContainsString(app.GetOmittedContainers(), "ddev-ssh-agent"),
		BindAllInterfaces:         app.BindAllInterfaces,
		MutagenEnabled:            app.IsMutagenEnabled(),

		NFSMountEnabled:    app.IsNFSMountEnabled(),
		NFSSource:          "",
		IsWindowsFS:        runtime.GOOS == "windows",
		NoProjectMount:     app.NoProjectMount,
		MountType:          "bind",
		WebMount:           "../",
		Hostnames:          app.GetHostnames(),
		Timezone:           timezone,
		ComposerVersion:    app.ComposerVersion,
		Username:           username,
		UID:                uid,
		GID:                gid,
		WebBuildContext:    "./.webimageBuild",
		DBBuildContext:     "./.dbimageBuild",
		FailOnHookFail:     app.FailOnHookFail || app.FailOnHookFailGlobal,
		WebWorkingDir:      app.GetWorkingDir("web", ""),
		DBWorkingDir:       app.GetWorkingDir("db", ""),
		WebEnvironment:     webEnvironment,
		MariaDBVolumeName:  app.GetMariaDBVolumeName(),
		PostgresVolumeName: app.GetPostgresVolumeName(),
		NFSMountVolumeName: app.GetNFSMountVolumeName(),
		NoBindMounts:       globalconfig.DdevGlobalConfig.NoBindMounts,
		Docroot:            app.GetDocroot(),
		UploadDirsMap:      app.getUploadDirsHostContainerMapping(),
		GitDirMount:        false,
		IsGitpod:           nodeps.IsGitpod(),
		IsCodespaces:       nodeps.IsCodespaces(),
		// Default max time we wait for containers to be healthy
		DefaultContainerTimeout: app.DefaultContainerTimeout,
		// Only use the extra_hosts technique for Linux and only if not WSL2 and not Colima
		// If WSL2 we have to figure out other things, see GetHostDockerInternalIP()
		UseHostDockerInternalExtraHosts: (runtime.GOOS == "linux" && !nodeps.IsWSL2() && !dockerutil.IsColima()) || (nodeps.IsWSL2() && globalconfig.DdevGlobalConfig.XdebugIDELocation == globalconfig.XdebugIDELocationWSL2),
	}
	// We don't want to bind-mount Git directory if it doesn't exist
	if fileutil.IsDirectory(filepath.Join(app.AppRoot, ".git")) {
		templateVars.GitDirMount = true
	}

	webimageExtraHTTPPorts := []string{}
	webimageExtraHTTPSPorts := []string{}
	exposedPorts := []int{}
	for _, a := range app.WebExtraExposedPorts {
		webimageExtraHTTPPorts = append(webimageExtraHTTPPorts, fmt.Sprintf("%d:%d", a.HTTPPort, a.WebContainerPort))
		webimageExtraHTTPSPorts = append(webimageExtraHTTPSPorts, fmt.Sprintf("%d:%d", a.HTTPSPort, a.WebContainerPort))
		exposedPorts = append(exposedPorts, a.WebContainerPort)
	}
	if len(exposedPorts) != 0 {
		templateVars.WebExtraHTTPPorts = "," + strings.Join(webimageExtraHTTPPorts, ",")
		templateVars.WebExtraHTTPSPorts = "," + strings.Join(webimageExtraHTTPSPorts, ",")

		templateVars.WebExtraExposedPorts = "expose:\n    - "
		// Odd way to join ints into a string from https://stackoverflow.com/a/37533144/215713
		templateVars.WebExtraExposedPorts = templateVars.WebExtraExposedPorts + strings.Trim(strings.Join(strings.Fields(fmt.Sprint(exposedPorts)), "\n    - "), "[]")
	}

	if app.Database.Type == nodeps.Postgres {
		templateVars.DBMountDir = "/var/lib/postgresql/data"
	}
	// TODO: Determine if mount to /bitnami is for all mysql/bitnami or just newest
	if app.Database.Type == nodeps.MySQL {
		templateVars.DBMountDir = "/bitnami/mysql/data"
	}
	if app.IsNFSMountEnabled() {
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

	if app.IsMutagenEnabled() {
		templateVars.MutagenVolumeName = GetMutagenVolumeName(app)
	}

	// Add web and db extra dockerfile info
	// If there is a user-provided Dockerfile, use that as the base and then add
	// our extra stuff like usernames, etc.
	// The db-build and web-build directories are used for context
	// so must exist. They usually do.

	for _, d := range []string{".webimageBuild", ".dbimageBuild"} {
		err = os.MkdirAll(app.GetConfigPath(d), 0755)
		if err != nil {
			return "", err
		}
		// We must start with a clean base directory
		err := fileutil.PurgeDirectory(app.GetConfigPath(d))
		if err != nil {
			util.Warning("unable to clean up directory %s, you may want to delete it manually: %v", d, err)
		}
	}
	err = os.MkdirAll(app.GetConfigPath("db-build"), 0755)
	if err != nil {
		return "", err
	}

	err = os.MkdirAll(app.GetConfigPath("web-build"), 0755)
	if err != nil {
		return "", err
	}

	extraWebContent := "\nRUN mkdir -p /home/$username && chown $username /home/$username && chmod 600 /home/$username/.pgpass"
	extraWebContent = extraWebContent + "\nENV NVM_DIR=/home/$username/.nvm"
	if app.NodeJSVersion != nodeps.NodeJSDefault {
		extraWebContent = extraWebContent + fmt.Sprintf(`
ENV N_PREFIX=/home/$username/.n
ENV N_INSTALL_VERSION="%s"
`, app.NodeJSVersion)
	}
	if app.CorepackEnable {
		extraWebContent = extraWebContent + "\nRUN corepack enable"
	}
	if app.Type == nodeps.AppTypeDrupal {
		// TODO: When ddev-webserver has required drupal 11+ sqlite version we can remove this.
		// These packages must be retrieved from snapshot.debian.org. We hope they'll be there
		// when we need them.
		drupalVersion, err := GetDrupalVersion(app)
		if err == nil && drupalVersion == "11" {
			extraWebContent = extraWebContent + "\n" + fmt.Sprintf(`
### Drupal 11+ requires a minimum sqlite3 version (3.45 currently)
ARG SQLITE_VERSION=%s
RUN log-stderr.sh bash -c "mkdir -p /tmp/sqlite3 && \
wget -O /tmp/sqlite3/sqlite3.deb https://snapshot.debian.org/archive/debian/20240203T152533Z/pool/main/s/sqlite3/sqlite3_${SQLITE_VERSION}-1_${TARGETPLATFORM##linux/}.deb && \
wget -O /tmp/sqlite3/libsqlite3.deb https://snapshot.debian.org/archive/debian/20240203T152533Z/pool/main/s/sqlite3/libsqlite3-0_${SQLITE_VERSION}-1_${TARGETPLATFORM##linux/}.deb && \
apt-get install -y /tmp/sqlite3/*.deb && \
rm -rf /tmp/sqlite3" || true
			`, versionconstants.Drupal11RequiredSqlite3Version)
		}
	}

	// Add supervisord config for WebExtraDaemons
	var supervisorGroup []string
	for _, appStart := range app.WebExtraDaemons {
		supervisorGroup = append(supervisorGroup, appStart.Name)
		supervisorConf := fmt.Sprintf(`
[program:%s]
group=webextradaemons
command=bash -c "%s; exit_code=$?; if [ $exit_code -ne 0 ]; then sleep 2; fi; exit $exit_code"
directory=%s
autostart=false
autorestart=true
startsecs=3 # Must stay up 3 sec, because "sleep 2" in case of fail
startretries=15
stdout_logfile=/var/tmp/logpipe
stdout_logfile_maxbytes=0
redirect_stderr=true
`, appStart.Name, appStart.Command, appStart.Directory)
		err = os.WriteFile(app.GetConfigPath(fmt.Sprintf(".webimageBuild/%s.conf", appStart.Name)), []byte(supervisorConf), 0755)
		if err != nil {
			return "", fmt.Errorf("failed to write .webimageBuild/%s.conf: %v", appStart.Name, err)
		}
		extraWebContent = extraWebContent + fmt.Sprintf("\nADD %s.conf /etc/supervisor/conf.d\nRUN chmod 644 /etc/supervisor/conf.d/%s.conf", appStart.Name, appStart.Name)
	}
	if len(supervisorGroup) > 0 {
		err = os.WriteFile(app.GetConfigPath(".webimageBuild/webextradaemons.conf"), []byte("[group:webextradaemons]\nprograms="+strings.Join(supervisorGroup, ",")), 0755)
		if err != nil {
			return "", fmt.Errorf("failed to write .webimageBuild/webextradaemons.conf: %v", err)
		}
		extraWebContent = extraWebContent + "\nADD webextradaemons.conf /etc/supervisor/conf.d\nRUN chmod 644 /etc/supervisor/conf.d/webextradaemons.conf\n"
	}
	// For MySQL 5.5+ we'll install the matching mysql client (and mysqldump) in the ddev-webserver
	if app.Database.Type == nodeps.MySQL {
		extraWebContent = extraWebContent + "\nRUN mysql-client-install.sh || true\n"
	}
	// Some MariaDB versions may have their own client in the ddev-webserver
	if app.Database.Type == nodeps.MariaDB {
		extraWebContent = extraWebContent + "\nRUN mariadb-client-install.sh || true\n"
	}

	err = WriteBuildDockerfile(app, app.GetConfigPath(".webimageBuild/Dockerfile"), app.GetConfigPath("web-build"), app.WebImageExtraPackages, app.ComposerVersion, extraWebContent)
	if err != nil {
		return "", err
	}

	// Add .pgpass to homedir on PostgreSQL
	extraDBContent := ""
	if app.Database.Type == nodeps.Postgres {
		// PostgreSQL 9/10/11 upstream images are stretch-based, out of support from Debian.
		// PostgreSQL 9/10 are out of support by PostgreSQL and no new images being pushed, see
		// https://github.com/docker-library/postgres/issues/1012
		// However, they do have a postgres:11-bullseye, but we won't start using it yet
		// because of awkward changes to $DBIMAGE. PostgreSQL 11 will be EOL Nov 2023
		if nodeps.ArrayContainsString([]string{nodeps.Postgres9, nodeps.Postgres10, nodeps.Postgres11}, app.Database.Version) {
			extraDBContent = extraDBContent + `
RUN rm -f /etc/apt/sources.list.d/pgdg.list
RUN echo "deb http://archive.debian.org/debian/ stretch main contrib non-free" > /etc/apt/sources.list
RUN apt-get update || true
RUN apt-get -y install apt-transport-https
RUN printf "deb http://apt-archive.postgresql.org/pub/repos/apt/ stretch-pgdg main" > /etc/apt/sources.list.d/pgdg.list
`
		}
		extraDBContent = extraDBContent + `
ENV PATH=$PATH:/usr/lib/postgresql/$PG_MAJOR/bin
ADD postgres_healthcheck.sh /
RUN chmod ugo+rx /postgres_healthcheck.sh
RUN mkdir -p /etc/postgresql/conf.d && chmod 777 /etc/postgresql/conf.d
RUN echo "*:*:db:db:db" > ~postgres/.pgpass && chown postgres:postgres ~postgres/.pgpass && chmod 600 ~postgres/.pgpass && chmod 777 /var/tmp && ln -sf /mnt/ddev_config/postgres/postgresql.conf /etc/postgresql && echo "restore_command = 'true'" >> /var/lib/postgresql/recovery.conf
RUN printf "# TYPE DATABASE USER CIDR-ADDRESS  METHOD \nhost  all  all 0.0.0.0/0 md5\nlocal all all trust\nhost    replication    db             0.0.0.0/0  trust\nhost replication all 0.0.0.0/0 trust\nlocal replication all trust\nlocal replication all peer\n" >/etc/postgresql/pg_hba.conf
RUN (apt-get update || true) && DEBIAN_FRONTEND=noninteractive apt-get install -y -o Dpkg::Options::="--force-confold" --no-install-recommends --no-install-suggests bzip2 less procps pv vim
`
	}

	err = WriteBuildDockerfile(app, app.GetConfigPath(".dbimageBuild/Dockerfile"), app.GetConfigPath("db-build"), app.DBImageExtraPackages, "", extraDBContent)

	// CopyEmbedAssets of postgres healthcheck has to be done after we WriteBuildDockerfile
	// because that deletes the .dbimageBuild directory
	if app.Database.Type == nodeps.Postgres {
		err = fileutil.CopyEmbedAssets(bundledAssets, "healthcheck/db/postgres", app.GetConfigPath(".dbimageBuild"), nil)
		if err != nil {
			return "", err
		}
	}

	if err != nil {
		return "", err
	}

	// SSH agent needs extra to add the official related user, nothing else
	err = WriteBuildDockerfile(app, filepath.Join(globalconfig.GetGlobalDdevDir(), ".sshimageBuild/Dockerfile"), "", nil, "", "")
	if err != nil {
		return "", err
	}

	templateVars.DockerIP, err = dockerutil.GetDockerIP()
	if err != nil {
		return "", err
	}
	if app.BindAllInterfaces {
		templateVars.DockerIP = "0.0.0.0"
	}

	t, err := template.New("app_compose_template.yaml").Funcs(getTemplateFuncMap()).ParseFS(bundledAssets, "app_compose_template.yaml")
	if err != nil {
		return "", err
	}

	err = t.Execute(&doc, templateVars)
	return doc.String(), err
}

// WriteBuildDockerfile writes a Dockerfile to be used in the
// docker-compose 'build'
// It may include the contents of .ddev/<container>-build
func WriteBuildDockerfile(app *DdevApp, fullpath string, userDockerfilePath string, extraPackages []string, composerVersion string, extraContent string) error {

	// Start with user-built dockerfile if there is one.
	err := os.MkdirAll(filepath.Dir(fullpath), 0755)
	if err != nil {
		return err
	}

	// Normal starting content is the arg and base image
	contents := `
#ddev-generated - Do not modify this file; your modifications will be overwritten.

### DDEV-injected base Dockerfile contents
ARG BASE_IMAGE="scratch"
FROM $BASE_IMAGE
SHELL ["/bin/bash", "-c"]
`
	contents = contents + `
ARG TARGETPLATFORM
ARG TARGETARCH
ARG TARGETOS
ARG username
ARG uid
ARG gid
ARG DDEV_PHP_VERSION
ARG DDEV_DATABASE
RUN (groupadd --gid $gid "$username" || groupadd "$username" || true) && (useradd  -l -m -s "/bin/bash" --gid "$username" --comment '' --uid $uid "$username" || useradd  -l -m -s "/bin/bash" --gid "$username" --comment '' "$username" || useradd  -l -m -s "/bin/bash" --gid "$gid" --comment '' "$username" || useradd -l -m -s "/bin/bash" --comment '' $username )
`
	// If there are user pre.Dockerfile* files, insert their contents
	if userDockerfilePath != "" {
		files, err := filepath.Glob(userDockerfilePath + "/pre.Dockerfile*")
		if err != nil {
			return err
		}

		for _, file := range files {
			userContents, err := fileutil.ReadFileIntoString(file)
			if err != nil {
				return err
			}

			contents = contents + "\n\n### From user Dockerfile " + file + ":\n" + userContents
		}
	}

	if extraContent != "" {
		contents = contents + fmt.Sprintf(`
### DDEV-injected extra content
%s
`, extraContent)
	}

	if extraPackages != nil {
		contents = contents + `
### DDEV-injected from webimage_extra_packages or dbimage_extra_packages
RUN (apt-get -qq update || true) && DEBIAN_FRONTEND=noninteractive apt-get -qq install -y -o Dpkg::Options::="--force-confold" --no-install-recommends --no-install-suggests ` + strings.Join(extraPackages, " ") + "\n"
	}

	// For webimage, update to latest Composer.
	if strings.Contains(fullpath, "webimageBuild") {
		// Version to run composer self-update to the version
		var composerSelfUpdateArg string

		// Remove leading and trailing spaces
		composerSelfUpdateArg = strings.TrimSpace(composerVersion)

		// Composer v2 is default
		if composerSelfUpdateArg == "" {
			composerSelfUpdateArg = "2"
		}

		// Major and minor versions have to be provided as option so add '--' prefix.
		// E.g. a major version can be 1 or 2, a minor version 2.2 or 2.1 etc.
		if strings.Count(composerVersion, ".") < 2 {
			composerSelfUpdateArg = "--" + composerSelfUpdateArg
		}

		// Try composer self-update twice because of troubles with Composer downloads
		// breaking testing.
		// First of all Composer is updated to latest stable release to ensure
		// new options of the self-update command can be used properly e.g.
		// selecting a branch instead of a major version only.
		contents = contents + fmt.Sprintf(`
### DDEV-injected composer update
RUN export XDEBUG_MODE=off; composer self-update --stable || composer self-update --stable || true; composer self-update %s || log-stderr.sh composer self-update %s || true
`, composerSelfUpdateArg, composerSelfUpdateArg)

		// For Postgres, install the relevant PostgreSQL clients
		if app.Database.Type == nodeps.Postgres {
			psqlVersion := app.Database.Version
			if psqlVersion == nodeps.Postgres9 {
				psqlVersion = "9.6"
			}
			contents = contents + fmt.Sprintf(`
RUN EXISTING_PSQL_VERSION=$(psql --version | awk -F '[\. ]*' '{ print $3 }'); \
if [ "${EXISTING_PSQL_VERSION}" != "%s" ]; then \
  log-stderr.sh bash -c "apt-get -qq update -o Dir::Etc::sourcelist="sources.list.d/pgdg.sources" -o Dir::Etc::sourceparts="-" -o APT::Get::List-Cleanup="0" && \
  apt-get install -y postgresql-client-%s && \
  apt-get remove -y postgresql-client-${EXISTING_PSQL_VERSION}" || true; \
fi`, app.Database.Version, psqlVersion) + "\n\n"
		}

	}

	// If there are user dockerfiles, appends their contents
	if userDockerfilePath != "" {
		files, err := filepath.Glob(userDockerfilePath + "/Dockerfile*")
		if err != nil {
			return err
		}

		for _, file := range files {
			// Skip the example file
			if file == userDockerfilePath+"/Dockerfile.example" {
				continue
			}

			userContents, err := fileutil.ReadFileIntoString(file)
			if err != nil {
				return err
			}

			// Backward compatible fix, remove unnecessary BASE_IMAGE references
			re, err := regexp.Compile(`ARG BASE_IMAGE.*\n|FROM \$BASE_IMAGE.*\n`)
			if err != nil {
				return err
			}

			userContents = re.ReplaceAllString(userContents, "")
			contents = contents + "\n\n### From user Dockerfile " + file + ":\n" + userContents
		}
	}

	// Assets in the web-build directory copied to .webimageBuild so .webimageBuild can be "context"
	// This actually copies the Dockerfile, but it is then immediately overwritten by WriteImageDockerfile()
	if userDockerfilePath != "" {
		err = copy2.Copy(userDockerfilePath, filepath.Dir(fullpath))
		if err != nil {
			return err
		}
	}

	// Some packages have default folder/file permissions described in /usr/lib/tmpfiles.d/*.conf files.
	// For example, when you upgrade systemd, it sets 755 for /var/log.
	// This may cause problems with previously set permissions when installing/upgrading packages.
	// Place this at the very end of the Dockerfile.
	if strings.Contains(fullpath, "webimageBuild") {
		contents = contents + fmt.Sprintf(`
### DDEV-injected folders permission fix
RUN chmod 777 /run/php /var/log
`)
	}

	return WriteImageDockerfile(fullpath, []byte(contents))
}

// WriteImageDockerfile writes a dockerfile at the fullpath (including the filename)
func WriteImageDockerfile(fullpath string, contents []byte) error {
	err := os.MkdirAll(filepath.Dir(fullpath), 0755)
	if err != nil {
		return err
	}
	err = os.WriteFile(fullpath, contents, 0644)
	if err != nil {
		return err
	}
	return nil
}

// Prompt for a project name.
func (app *DdevApp) promptForName() error {
	if app.Name == "" {
		dir, err := os.Getwd()
		if err == nil && hostRegex.MatchString(NormalizeProjectName(filepath.Base(dir))) {
			app.Name = NormalizeProjectName(filepath.Base(dir))
		}
	}

	name := util.Prompt("Project name", app.Name)
	if err := ValidateProjectName(name); err != nil {
		return err
	}
	app.Name = name

	err := app.CheckExistingAppInApproot()
	if err != nil {
		util.Failed(err.Error())
	}

	return nil
}

// AvailablePHPDocrootLocations returns an of default docroot locations to look for.
func AvailablePHPDocrootLocations() []string {
	return []string{
		"_www",
		"docroot",
		"htdocs",
		"html",
		"pub",
		"public",
		"web",
		"web/public",
		"webroot",
	}
}

// DiscoverDefaultDocroot returns the default docroot directory.
func DiscoverDefaultDocroot(app *DdevApp) string {
	// Provide use the app.Docroot as the default docroot option.
	var defaultDocroot = app.Docroot
	if defaultDocroot == "" {
		for _, docroot := range AvailablePHPDocrootLocations() {
			if _, err := os.Stat(filepath.Join(app.AppRoot, docroot)); err != nil {
				continue
			}

			if fileutil.FileExists(filepath.Join(app.AppRoot, docroot, "index.php")) {
				defaultDocroot = docroot
				break
			}
		}
	}
	dir, err := fileutil.FindFilenameInDirectory(app.AppRoot, []string{"manage.py"})
	if err == nil && dir != "" {
		defaultDocroot, err = filepath.Rel(app.AppRoot, dir)
		if err != nil {
			util.Warning("failed to filepath.Rel(%s, %s): %v", app.AppRoot, dir, err)
			defaultDocroot = ""
		}
	}

	return defaultDocroot
}

// Determine the document root.
func (app *DdevApp) docrootPrompt() error {

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
		if err = os.MkdirAll(fullPath, 0755); err != nil {
			return fmt.Errorf("unable to create docroot: %v", err)
		}

		util.Success("Created docroot at %s.", fullPath)
	}

	return nil
}

// ConfigExists determines if a DDEV config file exists for this application.
func (app *DdevApp) ConfigExists() bool {
	if _, err := os.Stat(app.ConfigPath); os.IsNotExist(err) {
		return false
	}
	return true
}

// AppTypePrompt handles the Type workflow.
func (app *DdevApp) AppTypePrompt() error {
	// First, see if we can auto detect what kind of site it is so we can set a sane default.
	detectedAppType := app.DetectAppType()

	// If we found an application type set it and inform the user.
	util.Success("Found a %s codebase at %s.", detectedAppType, filepath.Join(app.AppRoot, app.Docroot))

	validAppTypes := strings.Join(GetValidAppTypesWithoutAliases(), ", ")
	typePrompt := "Project Type [%s] (%s): "

	defaultAppType := app.Type
	if app.Type == nodeps.AppTypeNone || !IsValidAppType(app.Type) {
		defaultAppType = detectedAppType
	}

	fmt.Printf(typePrompt, validAppTypes, defaultAppType)
	appType := strings.ToLower(util.GetInput(defaultAppType))

	for !IsValidAppType(appType) {
		output.UserOut.Errorf("'%s' is not a valid project type. Allowed project types are: %s\n", appType, validAppTypes)

		fmt.Printf(typePrompt, validAppTypes, appType)
		return fmt.Errorf("invalid project type")
	}

	app.Type = appType

	return nil
}

// PrepDdevDirectory creates a .ddev directory in the current working directory
func PrepDdevDirectory(app *DdevApp) error {
	var err error
	dir := app.GetConfigPath("")
	if _, err := os.Stat(dir); os.IsNotExist(err) {

		log.WithFields(log.Fields{
			"directory": dir,
		}).Debug("Config Directory does not exist, attempting to create.")

		err = os.MkdirAll(dir, 0755)
		if err != nil {
			return err
		}
	}

	// Pre-create a few dirs so we can be sure they are owned by the user and not root.
	dirs := []string{
		"web-entrypoint.d",
		"xhprof",
	}
	for _, subdir := range dirs {
		err = os.MkdirAll(filepath.Join(dir, subdir), 0755)
		if err != nil {
			return err
		}
	}

	// Some of the listed items are wildcards or directories, and if they are, there's an error
	// opening them and they innately get added to the .gitignore.
	err = CreateGitIgnore(dir, "**/*.example", ".dbimageBuild", ".ddev-docker-*.yaml", ".*downloads", ".homeadditions", ".importdb*", ".sshimageBuild", ".venv", ".webimageBuild", "apache/apache-site.conf", "commands/.gitattributes", "config.local.y*ml", "db_snapshots", "mutagen/mutagen.yml", "mutagen/.start-synced", "nginx_full/nginx-site.conf", "postgres/postgresql.conf", "providers/acquia.yaml", "providers/lagoon.yaml", "providers/pantheon.yaml", "providers/platform.yaml", "providers/upsun.yaml", "sequelpro.spf", "settings/settings.ddev.py", fmt.Sprintf("traefik/config/%s.yaml", app.Name), fmt.Sprintf("traefik/certs/%s.crt", app.Name), fmt.Sprintf("traefik/certs/%s.key", app.Name), "xhprof/xhprof_prepend.php", "**/README.*")
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
		"pre-push",
		"post-push",
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
