package ddevapp

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
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
	copy2 "github.com/otiai10/copy"
	"go.yaml.in/yaml/v4"
)

// Regexp pattern to determine if a hostname is valid per RFC 1123.
var hostRegex = regexp.MustCompile(`^(([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]*[a-zA-Z0-9])\.)*([A-Za-z0-9]|[A-Za-z0-9][A-Za-z0-9\-]*[A-Za-z0-9])$`)

// Regexp pattern to match Composer v1 versions: "1" or "1.x.y"
var composerV1Regex = regexp.MustCompile(`^1(\.\d+\.\d+)?$`)

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
	if testMutagen := os.Getenv("DDEV_TEST_USE_MUTAGEN"); testMutagen == "true" {
		nodeps.PerformanceModeDefault = types.PerformanceModeMutagen
	}
	if os.Getenv("DDEV_TEST_NO_BIND_MOUNTS") == "true" {
		nodeps.NoBindMountsDefault = true
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
	defer util.TimeTrackC(fmt.Sprintf("ddevapp.NewApp(%s, includeOverrides=%t)", appRoot, includeOverrides))()

	app := &DdevApp{}

	if appRoot == "" {
		app.AppRoot, _ = os.Getwd()
	} else {
		app.AppRoot = appRoot
	}

	if err := HasAllowedLocation(app); err != nil {
		return app, err
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
	app.XHProfMode = nodeps.XHProfModeDefault

	app.FailOnHookFail = nodeps.FailOnHookFailDefault
	app.FailOnHookFailGlobal = globalconfig.DdevGlobalConfig.FailOnHookFailGlobal

	// Provide a default app name based on directory name
	app.Name = NormalizeProjectName(filepath.Base(app.AppRoot))

	// Gather containers to omit, adding ddev-router for codespaces/devcontainer
	app.OmitContainersGlobal = globalconfig.DdevGlobalConfig.OmitContainersGlobal
	if nodeps.IsDevcontainer() {
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

	// Make "drupal" an alias to latest modern drupal version
	if app.Type == nodeps.AppTypeDrupal {
		app.Type = nodeps.AppTypeDrupalLatestStable
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

	if app.WebserverType == "" {
		app.WebserverType = nodeps.WebserverDefault
	}

	if app.DefaultContainerTimeout == "" {
		app.DefaultContainerTimeout = nodeps.DefaultDefaultContainerTimeout
		// On Windows the default timeout may be too short for mutagen to succeed.
		if nodeps.IsWindows() {
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
		if err := app.ReadDockerComposeYAML(); err != nil {
			util.Verbose("Unable to read '%s' project config at %s: %v", app.Name, app.DockerComposeFullRenderedYAMLPath(), err)
		}
	}

	// TODO: Enable once the bootstrap is clean and every project is loaded once only
	//app.TrackProject()

	return app, nil
}

// GetConfigPath returns the path to an application config file specified by filename.
func (app *DdevApp) GetConfigPath(filename string) string {
	return filepath.Join(app.AppRoot, ".ddev", filename)
}

// GetProcessedProjectConfigYAML returns the processed project configuration as YAML
// This is equivalent to what 'ddev utility configyaml' shows - the project configuration
// after all config.*.yaml files have been merged and processed
func (app *DdevApp) GetProcessedProjectConfigYAML(omitKeys ...string) ([]byte, error) {
	// Ensure we have the latest processed configuration
	_, err := app.ReadConfig(true)
	if err != nil {
		return nil, fmt.Errorf("failed to read project configuration: %v", err)
	}

	// Marshal the fully processed DdevApp struct to YAML
	configYAML, err := yaml.Marshal(app)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal project configuration to YAML: %v", err)
	}

	// If no keys to omit, return as-is
	if len(omitKeys) == 0 {
		return configYAML, nil
	}

	// Parse YAML into a map to filter keys
	var configMap map[string]interface{}
	err = yaml.Unmarshal(configYAML, &configMap)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML for filtering: %v", err)
	}

	// Remove specified keys
	for _, key := range omitKeys {
		delete(configMap, strings.TrimSpace(key))
	}

	// Marshal back to YAML
	filteredYAML, err := yaml.Marshal(configMap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal filtered YAML: %v", err)
	}

	return filteredYAML, nil
}

// WriteConfig writes the app configuration into the .ddev folder.
func (app *DdevApp) WriteConfig() error {
	// Work against a copy of the DdevApp, since we don't want to actually change it.
	appcopy := *app

	// If the app name has been changed by `config.*.yaml`,
	// remove it from the main config.yaml file.
	if hasConfigNameOverride, _ := app.HasConfigNameOverride(); hasConfigNameOverride {
		appcopy.Name = ""
	}

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

	if composerV1Regex.MatchString(appcopy.ComposerVersion) {
		appcopy.ComposerVersion = "2.2"
		util.WarningOnce(`Project '%s' now uses Composer v2.2 LTS. Composer v1 is no longer supported by Packagist, see https://blog.packagist.com/shutting-down-packagist-org-support-for-composer-1-x/`, app.Name)
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

	// The .ddev directory may still need to be populated, especially in tests
	err = PopulateExamplesCommandsHomeadditions(app.Name)
	if err != nil {
		return err
	}

	// Allow project-specific post-config action
	err = appcopy.PostConfigAction()
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
	if app.ConfigPath == "" {
		app.ConfigPath = app.GetConfigPath("config.yaml")
	}
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

	// Sort WebExtraExposedPorts so the entry matching configured router ports comes first
	SortWebExtraExposedPorts(app)

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
	var err error
	if err = app.projectNamePrompt(); err != nil {
		return err
	}
	if err = app.docrootPrompt(); err != nil {
		return err
	}
	if err = app.AppTypePrompt(); err != nil {
		return err
	}
	if err = app.ConfigFileOverrideAction(false); err != nil {
		return err
	}
	if err = app.ValidateConfig(); err != nil {
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

	// Validate docroot
	if err := ValidateDocroot(app.Docroot); err != nil {
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
	// 		return fmt.Errorf("unable to configure project %s with database type %s because that database type does not match the current actual database. Please change your database type back to %s and start again, export, delete, and then change configuration and start. To get back to existing type use 'ddev config --database=%s', see docs at %s", app.Name, dbType, dbType, dbType, "https://docs.ddev.com/en/stable/users/extend/database-types/")
	// 	}
	// }

	usedNames := make(map[string]bool)
	usedWebContainerPorts := make(map[int]bool)
	usedHTTPAndHTTPSPorts := make(map[int]bool)
	for _, extraPort := range app.WebExtraExposedPorts {
		if usedNames[extraPort.Name] {
			return fmt.Errorf("the %s project has a duplicate 'name: %s' in web_extra_exposed_ports", app.Name, extraPort.Name)
		}
		usedNames[extraPort.Name] = true

		if extraPort.HTTPPort == extraPort.HTTPSPort {
			return fmt.Errorf("the %s project has the same 'http_port: %d' and 'https_port: %d' for 'name: %s' in web_extra_exposed_ports", app.Name, extraPort.HTTPPort, extraPort.HTTPSPort, extraPort.Name)
		}

		for name, port := range map[string]int{"container_port": extraPort.WebContainerPort, "http_port": extraPort.HTTPPort, "https_port": extraPort.HTTPSPort} {
			if err := dockerutil.ValidatePort(port); err != nil {
				return fmt.Errorf("the %s project has an invalid '%s: %d' for 'name: %s' in web_extra_exposed_ports", app.Name, name, port, extraPort.Name)
			}
		}

		if usedWebContainerPorts[extraPort.WebContainerPort] {
			return fmt.Errorf("the %s project has a duplicate 'container_port: %d' for 'name: %s' in web_extra_exposed_ports", app.Name, extraPort.WebContainerPort, extraPort.Name)
		}
		usedWebContainerPorts[extraPort.WebContainerPort] = true

		if usedHTTPAndHTTPSPorts[extraPort.HTTPPort] {
			return fmt.Errorf("the %s project has a duplicate 'http_port: %d' for 'name: %s' in web_extra_exposed_ports", app.Name, extraPort.HTTPPort, extraPort.Name)
		}
		usedHTTPAndHTTPSPorts[extraPort.HTTPPort] = true

		if usedHTTPAndHTTPSPorts[extraPort.HTTPSPort] {
			return fmt.Errorf("the %s project has a duplicate 'https_port: %d' for 'name: %s' in web_extra_exposed_ports", app.Name, extraPort.HTTPSPort, extraPort.Name)
		}
		usedHTTPAndHTTPSPorts[extraPort.HTTPSPort] = true
	}

	// Golang on Windows is not able to time.LoadLocation unless
	// Go is installed... so skip validation on Windows
	if !nodeps.IsWindows() {
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

// ValidateDocroot makes sure we have a usable docroot
// The docroot must remain inside the project root.
func ValidateDocroot(docroot string) error {
	switch {
	case filepath.IsAbs(docroot):
		return fmt.Errorf("docroot ('%s') cannot be an absolute path, it must be a relative path from the project root", docroot)
	case strings.HasPrefix(docroot, ".."):
		return fmt.Errorf("docroot ('%s') cannot begin with '..', it should be a relative path from project root but must remain inside the project", docroot)
	}
	return nil
}

// isCustomConfigFile returns true if the file exists and is not marked with
// either the standard DDEV signature or the silent-no-warn marker.
func isCustomConfigFile(filePath string) bool {
	if !fileutil.FileExists(filePath) {
		return false
	}
	sigFound, _ := fileutil.FgrepStringInFile(filePath, nodeps.DdevFileSignature)
	silentNoWarnFound, _ := fileutil.FgrepStringInFile(filePath, nodeps.DdevSilentNoWarn)
	return !sigFound && !silentNoWarnFound
}

// filterCustomConfigFiles returns only files that qualify as custom config files.
func filterCustomConfigFiles(files []string) []string {
	out := []string{}
	for _, f := range files {
		if isCustomConfigFile(f) {
			out = append(out, f)
		}
	}
	return out
}

// DockerComposeYAMLPath returns the absolute path to where the
// base generated yaml file should exist for this project.
func (app *DdevApp) DockerComposeYAMLPath() string {
	return app.GetConfigPath(".ddev-docker-compose-base.yaml")
}

// DockerComposeFullRenderedYAMLPath returns the absolute path to where the
// complete generated yaml file should exist for this project.
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

	mutagenConfigPath := app.GetConfigPath("mutagen/mutagen.yml")
	if isCustomConfigFile(mutagenConfigPath) && app.IsMutagenEnabled() {
		util.Warning("Using custom mutagen configuration in %s", mutagenConfigPath)
		customConfig = true
	}

	routerComposeFiles, err := filepath.Glob(filepath.Join(globalconfig.GetGlobalDdevDir(), "router-compose.*.yaml"))
	util.CheckErr(err)
	if len(routerComposeFiles) > 0 {
		customRouterComposeFiles := filterCustomConfigFiles(routerComposeFiles)
		if len(customRouterComposeFiles) > 0 {
			printableFiles, _ := util.ArrayToReadableOutput(customRouterComposeFiles)
			util.Warning("Using custom global router-compose configuration (use `docker logs ddev-router` for troubleshooting): %v", printableFiles)
			customConfig = true
		}
	}

	sshAuthComposeFiles, err := filepath.Glob(filepath.Join(globalconfig.GetGlobalDdevDir(), "ssh-auth-compose.*.yaml"))
	util.CheckErr(err)
	if len(sshAuthComposeFiles) > 0 {
		customSSHAuthComposeFiles := filterCustomConfigFiles(sshAuthComposeFiles)
		if len(customSSHAuthComposeFiles) > 0 {
			printableFiles, _ := util.ArrayToReadableOutput(customSSHAuthComposeFiles)
			util.Warning("Using custom global ssh-auth-compose configuration (use `docker logs ddev-ssh-agent` for troubleshooting): %v", printableFiles)
			customConfig = true
		}
	}

	traefikGlobalConfigPath := filepath.Join(globalconfig.GetGlobalDdevDir(), "traefik")
	if _, err := os.Stat(traefikGlobalConfigPath); err == nil {
		traefikGlobalFiles, err := filepath.Glob(filepath.Join(traefikGlobalConfigPath, "static_config.*.yaml"))
		util.CheckErr(err)
		if len(traefikGlobalFiles) > 0 {
			customTraefikGlobalFiles := filterCustomConfigFiles(traefikGlobalFiles)
			if len(customTraefikGlobalFiles) > 0 {
				printableFiles, _ := util.ArrayToReadableOutput(customTraefikGlobalFiles)
				util.Warning("Using custom global Traefik configuration (use `docker logs ddev-router` for troubleshooting): %v", printableFiles)
				customConfig = true
			}
		}
		// Check for custom-global-config directory (middleware, routers, etc.)
		customGlobalConfigDir := filepath.Join(traefikGlobalConfigPath, "custom-global-config")
		if fileutil.IsDirectory(customGlobalConfigDir) {
			customGlobalFiles, err := fileutil.ListFilesInDir(customGlobalConfigDir)
			// Remove README.md and local-auth.yaml.example from the list
			customGlobalFiles = slices.DeleteFunc(customGlobalFiles, func(f string) bool {
				base := filepath.Base(f)
				return base == "README.md" || base == "local-auth.yaml.example"
			})

			if err == nil && len(customGlobalFiles) > 0 {
				printableFiles, _ := util.ArrayToReadableOutput(customGlobalFiles)
				util.Warning("Using custom global Traefik dynamic configuration from %s: %v", customGlobalConfigDir, printableFiles)
				customConfig = true
			}
		}
	}

	// Warn if traefik config overridden
	mainProjectTraefikFile := app.GetConfigPath("traefik/config/" + app.Name + ".yaml")
	if isCustomConfigFile(mainProjectTraefikFile) {
		util.Warning("Using custom Traefik configuration (use `docker logs ddev-router` for troubleshooting): %v", mainProjectTraefikFile)
		customConfig = true
	}
	traefikProjectConfigPath := app.GetConfigPath("traefik/config")
	if _, err := os.Stat(traefikProjectConfigPath); err == nil {
		traefikFiles, err := filepath.Glob(filepath.Join(traefikProjectConfigPath, "*.yaml"))
		util.CheckErr(err)
		// Remove the main project traefik file from the list
		traefikFiles = slices.DeleteFunc(traefikFiles, func(f string) bool {
			return filepath.Base(f) == app.Name+".yaml"
		})
		ignoredTraefikFiles := filterCustomConfigFiles(traefikFiles)
		// Warn if there are unused files in project .ddev/traefik/config
		if len(ignoredTraefikFiles) > 0 {
			printableFiles, _ := util.ArrayToReadableOutput(ignoredTraefikFiles)
			util.Warning("Ignored project traefik config files found in .ddev/traefik/config (only %s will be used): %v", app.Name+".yaml", printableFiles)
		}
	}

	nginxFullConfigPath := app.GetConfigPath("nginx_full/nginx-site.conf")
	if isCustomConfigFile(nginxFullConfigPath) && app.WebserverType == nodeps.WebserverNginxFPM {
		util.Warning("Using custom nginx configuration in %s", nginxFullConfigPath)
		customConfig = true
	}

	apacheFullConfigPath := app.GetConfigPath("apache/apache-site.conf")
	if isCustomConfigFile(apacheFullConfigPath) && app.WebserverType == nodeps.WebserverApacheFPM {
		util.Warning("Using custom apache configuration in %s", apacheFullConfigPath)
		customConfig = true
	}

	if app.WebserverType == nodeps.WebserverNginxFPM {
		nginxPath := filepath.Join(ddevDir, "nginx")
		if _, err := os.Stat(nginxPath); err == nil {
			nginxFiles, err := filepath.Glob(filepath.Join(nginxPath, "*.conf"))
			util.CheckErr(err)
			customNginxFiles := filterCustomConfigFiles(nginxFiles)
			if len(customNginxFiles) > 0 {
				printableFiles, _ := util.ArrayToReadableOutput(customNginxFiles)
				util.Warning("Using nginx snippets: %v", printableFiles)
				customConfig = true
			}
		}
	}

	mysqlPath := filepath.Join(ddevDir, "mysql")
	if _, err := os.Stat(mysqlPath); err == nil {
		mysqlFiles, err := filepath.Glob(filepath.Join(mysqlPath, "*.cnf"))
		util.CheckErr(err)
		customMySQLFiles := filterCustomConfigFiles(mysqlFiles)
		if len(customMySQLFiles) > 0 {
			printableFiles, _ := util.ArrayToReadableOutput(customMySQLFiles)
			util.Warning("Using custom MySQL configuration: %v", printableFiles)
			customConfig = true
		}
	}

	phpPath := filepath.Join(ddevDir, "php")
	if _, err := os.Stat(phpPath); err == nil {
		phpFiles, err := filepath.Glob(filepath.Join(phpPath, "*.ini"))
		util.CheckErr(err)
		customPHPFiles := filterCustomConfigFiles(phpFiles)
		if len(customPHPFiles) > 0 {
			printableFiles, _ := util.ArrayToReadableOutput(customPHPFiles)
			util.Warning("Using custom PHP configuration: %v", printableFiles)
			customConfig = true
		}
	}

	for _, buildType := range []string{"web-build", "db-build"} {
		customDockerPath := filepath.Join(ddevDir, buildType)
		if _, err := os.Stat(customDockerPath); err == nil {
			mainDockerFiles, err := filepath.Glob(filepath.Join(customDockerPath, "Dockerfile*"))
			util.CheckErr(err)
			preDockerFiles, err := filepath.Glob(filepath.Join(customDockerPath, "pre.Dockerfile*"))
			util.CheckErr(err)
			prependDockerFiles, err := filepath.Glob(filepath.Join(customDockerPath, "prepend.Dockerfile*"))
			util.CheckErr(err)
			var dockerFiles []string
			dockerFiles = append(dockerFiles, prependDockerFiles...)
			dockerFiles = append(dockerFiles, preDockerFiles...)
			dockerFiles = append(dockerFiles, mainDockerFiles...)
			dockerFiles = slices.DeleteFunc(dockerFiles, func(s string) bool {
				return strings.HasSuffix(s, ".example")
			})
			if len(dockerFiles) > 0 {
				printableFiles, _ := util.ArrayToReadableOutput(dockerFiles)
				util.Warning("Using custom %s configuration: %v", buildType, printableFiles)
				customConfig = true
			}
		}
	}

	webEntrypointPath := filepath.Join(ddevDir, "web-entrypoint.d")
	if _, err := os.Stat(webEntrypointPath); err == nil {
		entrypointFiles, err := filepath.Glob(filepath.Join(webEntrypointPath, "*.sh"))
		util.CheckErr(err)
		customEntrypointFiles := filterCustomConfigFiles(entrypointFiles)
		if len(customEntrypointFiles) > 0 {
			printableFiles, _ := util.ArrayToReadableOutput(customEntrypointFiles)
			util.Warning("Using custom web-entrypoint.d configuration: %v", printableFiles)
			customConfig = true
		}
	}

	configPath := filepath.Join(ddevDir, "")
	if _, err := os.Stat(configPath); err == nil {
		configFiles, err := filepath.Glob(filepath.Join(configPath, "config.*.y*ml"))
		util.CheckErr(err)
		customConfigFiles := filterCustomConfigFiles(configFiles)
		if len(customConfigFiles) > 0 {
			printableFiles, _ := util.ArrayToReadableOutput(customConfigFiles)
			util.Warning("Using custom config.*.yaml configuration: %v", printableFiles)
			customConfig = true
		}
	}

	if customConfig {
		util.Warning("Custom configuration is updated on restart.\nIf you don't see your custom configuration taking effect, run 'ddev restart'.")
	}
}

// CheckDeprecations warns the user if anything in use is deprecated.
func (app *DdevApp) CheckDeprecations() {
	if composerV1Regex.MatchString(app.ComposerVersion) {
		app.ComposerVersion = "2.2"
		util.WarningOnce(`Project '%s' now uses Composer v2.2 LTS. Composer v1 is no longer supported by Packagist, see https://blog.packagist.com/shutting-down-packagist-org-support-for-composer-1-x/
Run 'ddev config --auto' to remove this Composer warning.`, app.Name)
	}
}

// FixObsolete removes files that may be obsolete, etc.
func (app *DdevApp) FixObsolete() {
	// Remove old in-project commands (which have been moved to global)
	for _, command := range []string{"db/mysql", "host/launch", "host/xhgui", "web/xdebug"} {
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
	for _, providerFile := range []string{"acquia.yaml.example", "pantheon.yaml.example", "platform.yaml.example"} {
		providerFilePath := app.GetConfigPath(filepath.Join("providers", providerFile))
		err := os.Remove(providerFilePath)
		if err == nil {
			util.Success("Removed obsolete file %s", providerFilePath)
		}
	}

	// Remove old global commands
	for _, command := range []string{"host/yarn", "host/xhgui", "web/nvm", "web/autocomplete/nvm"} {
		cmdPath := filepath.Join(globalconfig.GetGlobalDdevDir(), "commands/", command)
		signatureFound, err := fileutil.FgrepStringInFile(cmdPath, nodeps.DdevFileSignature)
		if err == nil && signatureFound {
			err = os.Remove(cmdPath)
			if err != nil {
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

	addOns := GetInstalledAddonNames(app)
	if slices.Contains(addOns, "xhgui") {
		util.Warning("The xhgui add-on is no longer necessary with this version of DDEV, removing it.")
		err := RemoveAddon(app, "xhgui", false, true)
		if err != nil {
			util.Warning("Error removing xhgui add-on: %v", err)
		}
		// Reload the hooks because we don't want to run the deleted hooks
		appCopy, err := NewApp(app.AppRoot, true)
		if err != nil {
			util.Warning("Error reloading app hooks after removing xhgui add-on: %v", err)
		} else {
			app.Hooks = appCopy.Hooks
		}
	}
}

type composeYAMLVars struct {
	Name                      string
	Plugin                    string
	AppType                   string
	WebserverType             string
	MailpitPort               string
	HostMailpitPort           string
	DBType                    string
	DBVersion                 string
	DBMountDir                string
	DBDataPath                string
	DBAPort                   string
	DBPort                    string
	DdevGenerated             string
	DisableSettingsManagement bool
	MountType                 string
	WebMount                  string
	WebBuildContext           string
	DBBuildContext            string
	WebBuildDockerfile        string
	DBBuildDockerfile         string
	SSHAgentBuildContext      string
	OmitDB                    bool
	OmitDBA                   bool
	OmitRouter                bool
	OmitSSHAgent              bool
	BindAllInterfaces         bool
	MariaDBVolumeName         string
	PostgresVolumeName        string
	MutagenEnabled            bool
	MutagenVolumeName         string
	DockerIP                  string
	IsWindowsFS               bool
	NoProjectMount            bool
	Timezone                  string
	ComposerVersion           string
	Username                  string
	UID                       string
	GID                       string
	FailOnHookFail            bool
	WebWorkingDir             string
	DBWorkingDir              string
	DBAWorkingDir             string
	WebEnvironment            []string
	NoBindMounts              bool
	Docroot                   string
	UploadDirsMap             []string
	GitDirMount               bool
	IsCodespaces              bool
	IsDevcontainer            bool
	DefaultContainerTimeout   string
	WebExtraContainerPorts    []int
	WebExtraHTTPPorts         string
	WebExtraHTTPSPorts        string
	WebExtraExposedPorts      string
	BitnamiVolumeDir          string
	UseHardenedImages         bool
	XHGuiHTTPPort             string
	XHGuiHTTPSPort            string
	XHGuiPort                 string
	HostXHGuiPort             string
	XhguiImage                string
	XHProfMode                types.XHProfMode
}

// RenderComposeYAML renders the contents of .ddev/.ddev-docker-compose*.
func (app *DdevApp) RenderComposeYAML() (string, error) {
	var doc bytes.Buffer
	var err error

	hostDockerInternal := dockerutil.GetHostDockerInternal()
	util.Debug("%s", hostDockerInternal.Message)

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

	uid, gid, username := dockerutil.GetContainerUser()
	_, err = app.GetProvider("")
	if err != nil {
		return "", err
	}

	timezone := app.Timezone
	if timezone == "" {
		timezone, err = util.GetLocalTimezone()
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
		WebserverType:             app.WebserverType,
		MailpitPort:               GetInternalPort(app, "mailpit"),
		HostMailpitPort:           app.HostMailpitPort,
		DBType:                    app.Database.Type,
		DBVersion:                 app.Database.Version,
		DBMountDir:                "/var/lib/mysql",
		DBPort:                    GetInternalPort(app, "db"),
		DdevGenerated:             nodeps.DdevFileSignature,
		DisableSettingsManagement: app.DisableSettingsManagement,
		OmitDB:                    nodeps.ArrayContainsString(app.GetOmittedContainers(), nodeps.DBContainer),
		OmitRouter:                nodeps.ArrayContainsString(app.GetOmittedContainers(), globalconfig.DdevRouterContainer),
		OmitSSHAgent:              nodeps.ArrayContainsString(app.GetOmittedContainers(), "ddev-ssh-agent"),
		BindAllInterfaces:         app.BindAllInterfaces,
		MutagenEnabled:            app.IsMutagenEnabled(),

		IsWindowsFS:        nodeps.IsWindows(),
		NoProjectMount:     app.NoProjectMount,
		MountType:          "bind",
		WebMount:           "../",
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
		NoBindMounts:       globalconfig.DdevGlobalConfig.NoBindMounts,
		Docroot:            app.GetDocroot(),
		UploadDirsMap:      app.getUploadDirsHostContainerMapping(),
		GitDirMount:        false,
		IsCodespaces:       nodeps.IsCodespaces(),
		IsDevcontainer:     nodeps.IsDevcontainer(),
		// Default max time we wait for containers to be healthy
		DefaultContainerTimeout: app.DefaultContainerTimeout,
		XHGuiHTTPPort:           app.GetXHGuiHTTPPort(),
		XHGuiHTTPSPort:          app.GetXHGuiHTTPSPort(),
		XHGuiPort:               GetInternalPort(app, "xhgui"),
		HostXHGuiPort:           app.HostXHGuiPort,
		XhguiImage:              docker.GetXhguiImage(),
		XHProfMode:              app.GetXHProfMode(),
		BitnamiVolumeDir:        "",
		UseHardenedImages:       globalconfig.DdevGlobalConfig.UseHardenedImages,
	}
	// We don't want to bind-mount Git directory if it doesn't exist
	if fileutil.IsDirectory(filepath.Join(app.AppRoot, ".git")) {
		templateVars.GitDirMount = true
	}

	webimageExtraHTTPPorts := []string{}
	webimageExtraHTTPSPorts := []string{}
	webExtraContainerPorts := []int{}
	for _, a := range app.WebExtraExposedPorts {
		webimageExtraHTTPPorts = append(webimageExtraHTTPPorts, fmt.Sprintf("%d:%d", a.HTTPPort, a.WebContainerPort))
		webimageExtraHTTPSPorts = append(webimageExtraHTTPSPorts, fmt.Sprintf("%d:%d", a.HTTPSPort, a.WebContainerPort))
		webExtraContainerPorts = append(webExtraContainerPorts, a.WebContainerPort)
	}
	templateVars.WebExtraContainerPorts = webExtraContainerPorts
	if len(webExtraContainerPorts) != 0 {
		templateVars.WebExtraHTTPPorts = "," + strings.Join(webimageExtraHTTPPorts, ",")
		templateVars.WebExtraHTTPSPorts = "," + strings.Join(webimageExtraHTTPSPorts, ",")

		templateVars.WebExtraExposedPorts = "expose:\n    - "
		// Odd way to join ints into a string from https://stackoverflow.com/a/37533144/215713
		templateVars.WebExtraExposedPorts = templateVars.WebExtraExposedPorts + strings.Trim(strings.Join(strings.Fields(fmt.Sprint(webExtraContainerPorts)), "\n    - "), "[]")
	}

	if app.Database.Type == nodeps.Postgres {
		templateVars.DBMountDir = app.GetPostgresDataDir()
		templateVars.DBDataPath = app.GetPostgresDataPath()
	}
	// TODO: Determine if mount to /bitnami is for all mysql/bitnami or just newest
	// If we expand to using bitnami for mariadb this will change.
	if app.Database.Type == nodeps.MySQL && (app.Database.Version == nodeps.MySQL80 || app.Database.Version == nodeps.MySQL84) {
		templateVars.BitnamiVolumeDir = "/bitnami/mysql"
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
	if app.NodeJSVersion != nodeps.NodeJSDefault {
		extraWebContent = extraWebContent + fmt.Sprintf(`
ENV N_PREFIX=/home/$username/.n
ENV N_INSTALL_VERSION="%s"
`, app.NodeJSVersion)
	}
	if app.CorepackEnable {
		extraWebContent = extraWebContent + "\nRUN corepack enable"
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
stopasgroup=true
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
		extraWebContent = extraWebContent + "\nRUN log-stderr.sh mysql-client-install.sh || true\n"
	}
	if app.Database.Type == nodeps.MariaDB {
		// Some MariaDB versions may have their own client in the ddev-webserver
		// Search for CHANGE_MARIADB_CLIENT to update related code
		if slices.Contains([]string{nodeps.MariaDB1011, nodeps.MariaDB114}, app.Database.Version) {
			extraWebContent = extraWebContent + "\nRUN log-stderr.sh mariadb-client-install.sh || true\n"
		}
	}
	// MariaDB uses mariadb-* command names, but legacy mysql* commands are commonly used
	// Install compatibility wrappers for MariaDB, remove them when not needed
	extraWebContent = extraWebContent + "\nRUN log-stderr.sh mariadb-compat-install.sh || true\n"

	err = WriteBuildDockerfile(app, app.GetConfigPath(".webimageBuild/Dockerfile"), app.GetConfigPath("web-build"), app.WebImageExtraPackages, app.ComposerVersion, extraWebContent)
	if err != nil {
		return "", err
	}

	// Patch legacy postgres images for EOL Debian compatibility
	// Rewrites APT sources for Stretch-based images (PostgreSQL 9-11)
	// Ref: https://serverfault.com/a/1131653
	// Buster is added to the list in case people themselves override $BASE_IMAGE
	// Adds PGDG archive repo and installs required keys/certs
	// Configures postgres environment: healthcheck, .pgpass, config mounts, pg_hba.conf
	extraDBContent := ""
	if app.Database.Type == nodeps.Postgres {
		extraDBContent = extraDBContent + fmt.Sprintf(`
ENV PATH=$PATH:/usr/lib/postgresql/$PG_MAJOR/bin
ADD healthcheck.sh /

RUN <<EOF
set -eu -o pipefail
source /etc/os-release || true
if [ "${VERSION_CODENAME:-}" = "stretch" ] || [ "${VERSION_CODENAME:-}" = "buster" ]; then
    rm -f /etc/apt/sources.list.d/pgdg.list
    echo "deb http://archive.debian.org/debian/ ${VERSION_CODENAME} main contrib non-free" >/etc/apt/sources.list
    echo "deb http://archive.debian.org/debian-security/ ${VERSION_CODENAME}/updates main contrib non-free" >>/etc/apt/sources.list
    timeout %[1]d apt-get -qq update -o Acquire::AllowInsecureRepositories=true \
        -o Acquire::AllowDowngradeToInsecureRepositories=true -o APT::Get::AllowUnauthenticated=true || true
    DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends --no-install-suggests -o APT::Get::AllowUnauthenticated=true \
        debian-archive-keyring apt-transport-https ca-certificates
    echo "deb http://apt-archive.postgresql.org/pub/repos/apt/ ${VERSION_CODENAME}-pgdg-archive main" >/etc/apt/sources.list.d/pgdg.list
fi
EOF

USER "$username"
RUN echo "*:*:db:db:db" > ~/.pgpass && chmod 600 ~/.pgpass
USER root

RUN <<EOF
set -eu -o pipefail
chmod ugo+rx /healthcheck.sh
mkdir -p /etc/postgresql/conf.d
chmod 777 /etc/postgresql/conf.d
chmod 777 /var/tmp
ln -sf /mnt/ddev_config/postgres/postgresql.conf /etc/postgresql

echo "# TYPE DATABASE USER CIDR-ADDRESS  METHOD
host  all         all 0.0.0.0/0 md5
local all         all trust
host  replication db  0.0.0.0/0 trust
host  replication all 0.0.0.0/0 trust
local replication all trust
local replication all peer" >/etc/postgresql/pg_hba.conf

timeout %[1]d apt-get update || true
DEBIAN_FRONTEND=noninteractive apt-get install -y \
    -o Dpkg::Options::="--force-confold" --no-install-recommends --no-install-suggests \
    apt-transport-https bzip2 ca-certificates less procps pv vim-tiny zstd
update-alternatives --install /usr/bin/vim vim /usr/bin/vim.tiny 10

# Install tzdata-legacy to avoid issues with deprecated timezones
if apt-cache show tzdata-legacy >/dev/null 2>&1; then
    DEBIAN_FRONTEND=noninteractive apt-get install -y -o Dpkg::Options::="--force-confold" tzdata-legacy
fi

# Change directories owned by postgres (and everything inside them)
find / -type d \( -user postgres -o -group postgres \) -exec chown -Rh %[2]s:%[3]s {} + 2>/dev/null || true
# Change any remaining individual files owned by postgres
find / -type f \( -user postgres -o -group postgres \) -exec chown -h %[2]s:%[3]s {} + 2>/dev/null || true
EOF
`, app.GetMaxContainerWaitTime(), uid, gid)
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
ARG BASE_IMAGE="scratch"
`
	// If there are user prepend.Dockerfile* files, insert their contents
	if userDockerfilePath != "" {
		files, err := filepath.Glob(filepath.Join(userDockerfilePath, "prepend.Dockerfile*"))
		if err != nil {
			return err
		}

		for _, file := range files {
			// Skip example files
			if strings.HasSuffix(file, ".example") {
				continue
			}
			userContents, err := fileutil.ReadFileIntoString(file)
			if err != nil {
				return err
			}

			contents = contents + "\n\n### From user Dockerfile " + file + ":\n" + userContents
		}
	}

	contents = contents + `
### DDEV-injected base Dockerfile contents
FROM $BASE_IMAGE
SHELL ["/bin/bash", "-c"]
`
	// bitnami/mysql inappropriately sets ENV HOME=/, see https://github.com/bitnami/containers/issues/75578
	// Setting HOME="" allows it to have its normal behavior for the added user.
	if app.Database.Type == nodeps.MySQL && (app.Database.Version == nodeps.MySQL80 || app.Database.Version == nodeps.MySQL84) {
		contents = contents + `
ENV HOME=""
`
	}

	//  The ENV HOME="" is added for bitnami/mysql habit of overriding ENV HOME=/
	contents = contents + `
ARG TARGETPLATFORM
ARG TARGETARCH
ARG TARGETOS
ARG username
ARG uid
ARG gid
ARG DDEV_PHP_VERSION
ARG DDEV_DATABASE
RUN getent group tty || groupadd tty
RUN (groupadd --gid "$gid" "$username" || groupadd "$username" || true) && \
    (useradd -G tty -l -m -s "/bin/bash" --gid "$username" --comment '' --uid "$uid" "$username" || \
    useradd -G tty -l -m -s "/bin/bash" --gid "$username" --comment '' "$username" || \
    useradd -G tty -l -m -s "/bin/bash" --gid "$gid" --comment '' "$username" || \
    useradd -G tty -l -m -s "/bin/bash" --comment '' "$username")
`

	// If there are user pre.Dockerfile* files, insert their contents
	if userDockerfilePath != "" {
		files, err := filepath.Glob(filepath.Join(userDockerfilePath, "pre.Dockerfile*"))
		if err != nil {
			return err
		}

		for _, file := range files {
			// Skip example files
			if strings.HasSuffix(file, ".example") {
				continue
			}
			userContents, err := fileutil.ReadFileIntoString(file)
			if err != nil {
				return err
			}

			contents = contents + "\n\n### From user Dockerfile " + file + ":\n" + userContents
		}
	}

	// If our PHP version is not already provided in the ddev-webserver, add it now
	if strings.Contains(fullpath, "webimageBuild") {
		if _, ok := nodeps.PreinstalledPHPVersions[app.PHPVersion]; !ok {
			contents = contents + fmt.Sprintf(`
### DDEV-injected addition of not-preinstalled PHP version
RUN /usr/local/bin/install_php_extensions.sh "php%s" "${TARGETARCH}"
`, app.PHPVersion)
		}
	}

	if extraContent != "" {
		contents = contents + fmt.Sprintf(`
### DDEV-injected extra content
%s
`, extraContent)
	}

	if extraPackages != nil {
		contents = contents + fmt.Sprintf(`
### DDEV-injected from webimage_extra_packages or dbimage_extra_packages
RUN (timeout %d apt-get update || true) && DEBIAN_FRONTEND=noninteractive apt-get install -y -o Dpkg::Options::="--force-confold" --no-install-recommends --no-install-suggests %v
`, app.GetMaxContainerWaitTime(), strings.Join(extraPackages, " "))
	}

	// webimage only things
	if strings.Contains(fullpath, "webimageBuild") {
		// For webimage, update to latest Composer.
		// Version to run composer self-update to the version
		var composerSelfUpdateArg string

		// Remove leading and trailing spaces
		composerSelfUpdateArg = strings.TrimSpace(composerVersion)

		// If no version is specified, use "stable"
		if composerSelfUpdateArg == "" {
			composerSelfUpdateArg = "stable"
		}

		// Major and minor versions have to be provided as option so add '--' prefix.
		// E.g. a major version can be 2, a minor version 2.2 or 2.1 etc.
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

		if _, ok := nodeps.PreinstalledPHPVersions[app.PHPVersion]; !ok {
			contents = contents + fmt.Sprintf(`
### DDEV-injected php default version setting
RUN update-alternatives --set php /usr/bin/php%s
RUN chmod ugo+rw /var/log/php-fpm.log && chmod ugo+rwx /var/run && ln -sf /usr/sbin/php-fpm%s /usr/sbin/php-fpm
RUN mkdir -p /tmp/xhprof
RUN chmod -fR ugo+w /etc/php /var/lib/php/modules /tmp/xhprof
RUN phpdismod blackfire xdebug xhprof
`, app.PHPVersion, app.PHPVersion)
		}

		// For Postgres, install the relevant PostgreSQL clients
		if app.Database.Type == nodeps.Postgres {
			psqlVersion := app.Database.Version
			if psqlVersion == nodeps.Postgres9 {
				psqlVersion = "9.6"
			}
			contents = contents + fmt.Sprintf(`
### DDEV-injected postgresql-client setup
RUN <<EOF
set -eu -o pipefail
EXISTING_PSQL_VERSION=$(psql --version | awk -F '[\. ]*' '{ print $3 }' || true)
if [ "${EXISTING_PSQL_VERSION}" != "%s" ]; then
  log-stderr.sh apt-get update -o Acquire::Retries=5 -o Dir::Etc::sourcelist="sources.list.d/pgdg.sources" -o Dir::Etc::sourceparts="-" -o APT::Get::List-Cleanup="0" || true
  log-stderr.sh apt-get install -y postgresql-client-%s && apt-get remove -y --autoremove postgresql-client || true
fi
EOF
`, app.Database.Version, psqlVersion) + "\n\n"
		}
	}

	// If there are user dockerfiles, appends their contents
	if userDockerfilePath != "" {
		files, err := filepath.Glob(filepath.Join(userDockerfilePath, "Dockerfile*"))
		if err != nil {
			return err
		}

		for _, file := range files {
			// Skip example files
			if strings.HasSuffix(file, ".example") {
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
	if userDockerfilePath != "" {
		err = copy2.Copy(userDockerfilePath, filepath.Dir(fullpath), copy2.Options{
			Skip: func(_ os.FileInfo, src, _ string) (bool, error) {
				// Do not copy file if it's not a context file
				return isNotDockerfileContextFile(userDockerfilePath, src)
			},
		})
		if err != nil {
			return err
		}
	}

	// Some packages have default folder/file permissions described in /usr/lib/tmpfiles.d/*.conf files.
	// For example, when you upgrade systemd, it sets 755 for /var/log.
	// This may cause problems with previously set permissions when installing/upgrading packages.
	// Place this at the very end of the Dockerfile.
	if strings.Contains(fullpath, "webimageBuild") {
		contents = contents + `
### DDEV-injected folders permission fix
RUN chmod 777 /run/php /var/log && \
    chmod -f ugo+rwx /usr/local/bin /usr/local/bin/* && \
    mkdir -p /tmp/xhprof && chmod -R ugo+w /etc/php /var/lib/php /tmp/xhprof
`
		// Files from containers/ddev-webserver/ddev-webserver-base-files/var/www/html
		// are added to host on `ddev start` when using Podman with Mutagen enabled
		if dockerutil.IsPodman() && app.IsMutagenEnabled() {
			contents = contents + `
### DDEV-injected cleanup of /var/www/html for Podman with Mutagen
RUN rm -rf /var/www/html && mkdir -p /var/www/html
`
		}
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

// HasAllowedLocation returns an error if the project location is not recommended
func HasAllowedLocation(app *DdevApp) error {
	// Do not run this check if we want to delete the project.
	if !RunValidateConfig {
		return nil
	}
	homeDir, _ := os.UserHomeDir()
	if app.AppRoot == homeDir || app.AppRoot == filepath.Dir(globalconfig.GetGlobalDdevDir()) {
		return fmt.Errorf("a project is not allowed in your home directory (%v)", app.AppRoot)
	}
	rel, err := filepath.Rel(app.AppRoot, homeDir)
	if err == nil && !strings.HasPrefix(rel, "..") {
		return fmt.Errorf("a project is not allowed in the parent directory of your home directory (%v)", app.AppRoot)
	}
	rel, err = filepath.Rel(globalconfig.GetGlobalDdevDir(), app.AppRoot)
	if err == nil && !strings.HasPrefix(rel, "..") {
		return fmt.Errorf("a project is not allowed in your global config directory (%v)", app.AppRoot)
	}
	if fileutil.FileExists(filepath.Join(app.AppRoot, "cmd/ddev/main.go")) && fileutil.FileExists(filepath.Join(app.AppRoot, "cmd/ddev_gen_autocomplete/ddev_gen_autocomplete.go")) {
		return fmt.Errorf("a project cannot be created in the DDEV source code (%v)", app.AppRoot)
	}
	// If this is an existing project, allow it.
	if fileutil.FileExists(app.GetConfigPath("config.yaml")) {
		return nil
	}
	// Check all projects if they are located in the subdirectories of the project we are in.
	projectMap := globalconfig.GetGlobalProjectList()
	projectList := make([]*globalconfig.ProjectInfo, 0, len(projectMap))
	for _, project := range projectMap {
		projectList = append(projectList, project)
	}
	// Sort the projects by AppRoot in reverse alphabetical order,
	// this ensures that subdirectory projects are checked first.
	sort.Slice(projectList, func(i, j int) bool {
		return projectList[i].AppRoot > projectList[j].AppRoot
	})
	for _, project := range projectList {
		// Without sorting, a parent directory might be matched first,
		// causing the function to return without checking the project in the subdirectory.
		if app.AppRoot == project.AppRoot {
			return nil
		}
		// Do not allow 'ddev config' in any parent directory of any project
		rel, err = filepath.Rel(app.AppRoot, project.AppRoot)
		if err == nil && !strings.HasPrefix(rel, "..") {
			return fmt.Errorf("a project is not allowed in %s because another project exists in the subdirectory %s\nUnlist this project (if it exists) with 'cd \"%s\" && ddev stop --unlist'\nOr run 'ddev stop --unlist' for all projects in the subdirectories of this project directory", app.AppRoot, project.AppRoot, app.AppRoot)
		}
	}
	return nil
}

// projectNamePrompt Prompt for a project name.
func (app *DdevApp) projectNamePrompt() error {
	if app.Name == "" {
		dir, err := os.Getwd()
		if err == nil && hostRegex.MatchString(NormalizeProjectName(filepath.Base(dir))) {
			app.Name = NormalizeProjectName(filepath.Base(dir))
		}
	}

	for {
		name := util.Prompt("Project name", app.Name)
		if err := ValidateProjectName(name); err != nil {
			output.UserOut.Print(util.ColorizeText(err.Error(), "yellow"))
		} else {
			app.Name = name
			break
		}
	}

	err := app.CheckExistingAppInApproot()
	if err != nil {
		return err
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

// CreateDocroot creates the docroot for the project if it doesn't exist
func (app *DdevApp) CreateDocroot() error {
	if app.GetDocroot() == "" {
		return nil
	}
	if err := ValidateDocroot(app.GetDocroot()); err != nil {
		return err
	}
	docrootAbsPath := app.GetAbsDocroot(false)
	if !fileutil.IsDirectory(docrootAbsPath) {
		if err := os.MkdirAll(docrootAbsPath, 0755); err != nil {
			return err
		}
		util.Success("Created docroot at %s", docrootAbsPath)
	}
	return nil
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
	return defaultDocroot
}

// docrootPrompt Determine the document root.
func (app *DdevApp) docrootPrompt() error {
	// Determine the document root.
	output.UserOut.Printf("\nThe docroot is the directory from which your site is served.\nThis is a relative path from your project root at %s\n", app.AppRoot)
	output.UserOut.Printf("Leave docroot empty (hit <RETURN>) to use the location shown in parentheses.\nOr specify a custom path if your index.php is in a different directory.\nOr use '.' (a dot) to explicitly set it to the project root.\n")
	var docrootPrompt = "Docroot Location"
	var defaultDocroot = DiscoverDefaultDocroot(app)
	// If there is a default docroot, display it in the prompt.
	if defaultDocroot != "" {
		docrootPrompt = fmt.Sprintf("%s (%s)", docrootPrompt, defaultDocroot)
	} else {
		docrootPrompt = fmt.Sprintf("%s (project root)", docrootPrompt)
	}

	for {
		fmt.Print(docrootPrompt + ": ")
		app.Docroot = util.GetQuotedInput(defaultDocroot)

		// Ensure that the docroot exists
		if err := app.CreateDocroot(); err != nil {
			output.UserOut.Print(util.ColorizeText(err.Error(), "yellow"))
		} else {
			output.UserOut.Println()
			break
		}
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
	util.Success("Found a %s codebase at %s.", detectedAppType, app.GetAbsDocroot(false))

	validAppTypes := strings.Join(GetValidAppTypes(), ", ")
	typePrompt := "Project Type [%s] (%s): "

	defaultAppType := app.Type
	if app.Type == nodeps.AppTypeNone || !IsValidAppType(app.Type) {
		defaultAppType = detectedAppType
	}

	appType := ""

	for {
		fmt.Printf(typePrompt, validAppTypes, defaultAppType)
		appType = strings.ToLower(util.GetInput(defaultAppType))
		if IsValidAppType(appType) {
			break
		}
		output.UserOut.Print(util.ColorizeText(fmt.Sprintf("'%s' is not a valid project type", appType), "yellow"))
	}

	app.Type = appType

	return nil
}

// PrepDdevDirectory creates a .ddev directory in the current working directory
func PrepDdevDirectory(app *DdevApp) error {
	var err error
	dir := app.GetConfigPath("")
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		output.UserOut.WithFields(output.Fields{
			"directory": dir,
		}).Debug("Config Directory does not exist, attempting to create.")

		err = os.MkdirAll(dir, 0755)
		if err != nil {
			return err
		}
	}

	// Some of the listed items are wildcards or directories, and if they are, there's an error
	// opening them and they innately get added to the .gitignore.
	err = CreateGitIgnore(dir, "**/*.example", ".build-hash", ".dbimageBuild", ".ddev-docker-*.yaml", ".*downloads", ".homeadditions", ".importdb*", ".sshimageBuild", ".webimageBuild", "apache/apache-site.conf", "commands/.gitattributes", "config.local.y*ml", "config.*.local.y*ml", "db_snapshots", "mutagen/mutagen.yml", "mutagen/.start-synced", "nginx_full/nginx-site.conf", "postgres/postgresql.conf", "providers/acquia.yaml", "providers/lagoon.yaml", "providers/pantheon.yaml", "providers/platform.yaml", "providers/upsun.yaml", "sequelpro.spf", "share-providers/cloudflared.sh", "share-providers/ngrok.sh", fmt.Sprintf("traefik/config/%s.yaml", app.Name), fmt.Sprintf("traefik/certs/%s.crt", app.Name), fmt.Sprintf("traefik/certs/%s.key", app.Name), "xhprof/xhprof_prepend.php", "**/README.*")
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
		"pre-share",
		"post-share",
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

// isNotDockerfileContextFile returns true if the given file is NOT a Dockerfile context file
// We consider files in the .ddev/web-build and .ddev/db-build directory to be context files
// excluding /Dockerfile*, /pre.Dockerfile*, /prepend.Dockerfile* and /README.txt
func isNotDockerfileContextFile(userDockerfilePath string, file string) (bool, error) {
	// Directories are always context.
	if fileutil.IsDirectory(file) {
		return false, nil
	}
	// Get the relative path of the file from userDockerfilePath
	relPath, err := filepath.Rel(userDockerfilePath, file)
	if err != nil {
		return false, err
	}
	// If this is not a top-level file, it's a context file
	if strings.Contains(relPath, string(filepath.Separator)) {
		return false, nil
	}
	filename := filepath.Base(file)
	// Return true for not context Dockerfiles
	if strings.HasPrefix(filename, "Dockerfile") || strings.HasPrefix(filename, "pre.Dockerfile") || strings.HasPrefix(filename, "prepend.Dockerfile") {
		return true, nil
	}
	// Return true for not context README.txt if it is managed by DDEV
	if filename == "README.txt" {
		if err := fileutil.CheckSignatureOrNoFile(file, nodeps.DdevFileSignature); err == nil {
			return true, nil
		}
	}
	// Otherwise, it's a context file
	return false, nil
}
