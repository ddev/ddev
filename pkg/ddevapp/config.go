package ddevapp

import (
	"bytes"
	"fmt"
	"github.com/drud/ddev/pkg/dockerutil"
	"html/template"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/drud/ddev/pkg/globalconfig"

	"regexp"

	"runtime"

	"github.com/drud/ddev/pkg/appports"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/util"
	"github.com/drud/ddev/pkg/version"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// Regexp pattern to determine if a hostname is valid per RFC 1123.
var hostRegex = regexp.MustCompile(`^(([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]*[a-zA-Z0-9])\.)*([A-Za-z0-9]|[A-Za-z0-9][A-Za-z0-9\-]*[A-Za-z0-9])$`)

// Command defines commands to be run as pre/post hooks
type Command struct {
	Exec     string `yaml:"exec,omitempty"`
	ExecHost string `yaml:"exec-host,omitempty"`
}

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
		WebserverDefault = testWebServerType
	}
	if testWebcache := os.Getenv("DDEV_TEST_USE_WEBCACHE"); testWebcache != "" {
		WebcacheEnabledDefault = true
	}
	if testNFSMount := os.Getenv("DDEV_TEST_USE_NFSMOUNT"); testNFSMount != "" {
		NFSMountEnabledDefault = true
	}
}

// NewApp creates a new DdevApp struct with defaults set and overridden by any existing config.yml.
func NewApp(AppRoot string, includeOverrides bool, provider string) (*DdevApp, error) {
	// Set defaults.
	app := &DdevApp{}

	app.AppRoot = AppRoot
	app.ConfigPath = app.GetConfigPath("config.yaml")
	app.APIVersion = version.DdevVersion
	app.PHPVersion = PHPDefault
	app.WebserverType = WebserverDefault
	app.WebcacheEnabled = WebcacheEnabledDefault
	app.RouterHTTPPort = DdevDefaultRouterHTTPPort
	app.RouterHTTPSPort = DdevDefaultRouterHTTPSPort
	app.MariaDBVersion = version.MariaDBDefaultVersion

	// These should always default to the latest image/tag names from the Version package.
	app.WebImage = version.GetWebImage()
	app.DBImage = version.GetDBImage(app.MariaDBVersion)
	app.DBAImage = version.GetDBAImage()
	app.BgsyncImage = version.GetBgsyncImage()

	// Load from file if available. This will return an error if the file doesn't exist,
	// and it is up to the caller to determine if that's an issue.
	if _, err := os.Stat(app.ConfigPath); !os.IsNotExist(err) {
		_, err = app.ReadConfig(includeOverrides)
		if err != nil {
			return app, fmt.Errorf("%v exists but cannot be read. It may be invalid due to a syntax error.: %v", app.ConfigPath, err)
		}
	}

	// Turn off webcache_enabled except if macOS/darwin or global `developer_mode: true`
	if runtime.GOOS != "darwin" && app.WebcacheEnabled && !globalconfig.DdevGlobalConfig.DeveloperMode {
		app.WebcacheEnabled = false
		util.Warning("webcache_enabled is not yet supported on %s, disabling it", runtime.GOOS)
	}

	// Allow override with provider.
	// Otherwise we accept whatever might have been in config file if there was anything.
	if provider == "" && app.Provider != "" {
		// Do nothing. This is the case where the config has a provider and no override is provided. Config wins.
	} else if provider == ProviderPantheon || provider == ProviderDrudS3 || provider == ProviderDefault {
		app.Provider = provider // Use the provider passed-in. Function argument wins.
	} else if provider == "" && app.Provider == "" {
		app.Provider = ProviderDefault // Nothing passed in, nothing configured. Set c.Provider to default
	} else {
		return app, fmt.Errorf("provider '%s' is not implemented", provider)
	}
	app.SetRavenTags()
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
	if appcopy.DBImage == version.GetDBImage(appcopy.MariaDBVersion) {
		appcopy.DBImage = ""
	}
	if appcopy.DBAImage == version.GetDBAImage() {
		appcopy.DBAImage = ""
	}
	if appcopy.DBAImage == version.GetDBAImage() {
		appcopy.DBAImage = ""
	}
	if appcopy.BgsyncImage == version.GetBgsyncImage() {
		appcopy.BgsyncImage = ""
	}

	// Don't write default working dir values to config
	defaults := appcopy.DefaultWorkingDirMap()
	for service, defaultWorkingDir := range defaults {
		if app.WorkingDir[service] == defaultWorkingDir {
			delete(appcopy.WorkingDir, service)
		}
	}

	err := PrepDdevDirectory(filepath.Dir(appcopy.ConfigPath))
	if err != nil {
		return err
	}

	cfgbytes, err := yaml.Marshal(appcopy)
	if err != nil {
		return err
	}

	// Append current image information
	cfgbytes = append(cfgbytes, []byte(fmt.Sprintf("\n\n# This config.yaml was created with ddev version %s \n# webimage: %s\n# dbimage: %s\n# dbaimage: %s\n# bgsyncimage: %s\n# However we do not recommend explicitly wiring these images into the\n# config.yaml as they may break future versions of ddev.\n# You can update this config.yaml using 'ddev config'.\n", version.DdevVersion, version.GetWebImage(), version.GetDBImage(), version.GetDBAImage(), version.GetBgsyncImage()))...)

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

	return nil
}

// ReadConfig reads project configuration, falling
// back to defaults for config values not defined in the read config file.
func (app *DdevApp) ReadConfig(includeOverrides bool) ([]string, error) {

	// Load config.yaml
	err := app.LoadConfigYamlFile(app.ConfigPath)
	if err != nil {
		return []string{}, fmt.Errorf("unable to load config file %s: %v", app.ConfigPath, err)
	}
	app.DBImage = "" // DBImage will be set below

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

	// If any of these values aren't defined in the config file, set them to defaults.
	if app.Name == "" {
		app.Name = filepath.Base(app.AppRoot)
	}

	// If app.DBImage has not has been overridden, use it,
	// Otherwise just use GetDBImage to get the correct image.
	if app.DBImage == "" {
		app.DBImage = version.GetDBImage(app.MariaDBVersion)
	}

	if WebcacheEnabledDefault == true {
		app.WebcacheEnabled = WebcacheEnabledDefault
	}

	if NFSMountEnabledDefault == true {
		app.NFSMountEnabled = NFSMountEnabledDefault
	}

	if app.OmitContainers == nil {
		app.OmitContainers = globalconfig.DdevGlobalConfig.OmitContainers
	}

	app.SetApptypeSettingsPaths()

	return append([]string{app.ConfigPath}, configOverrides...), nil
}

// LoadConfigYamlFile loads one config.yaml into app, overriding what might be there.
func (app *DdevApp) LoadConfigYamlFile(filePath string) error {
	source, err := ioutil.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("could not find an active ddev configuration at %s have you run 'ddev config'? %v", app.ConfigPath, err)
	}

	// validate extend command keys
	err = validateCommandYaml(source)
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
		util.Warning("You are reconfiguring the project at %s. \nThe existing configuration will be updated and replaced.", app.AppRoot)
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
	if !IsValidPHPVersion(app.PHPVersion) {
		return fmt.Errorf("invalid PHP version: %s, must be one of %v", app.PHPVersion, GetValidPHPVersions()).(invalidPHPVersion)
	}

	// validate webserver type
	if !IsValidWebserverType(app.WebserverType) {
		return fmt.Errorf("invalid webserver type: %s, must be one of %s", app.WebserverType, GetValidWebserverTypes()).(invalidWebserverType)
	}

	if !IsValidOmitContainers(app.OmitContainers) {
		return fmt.Errorf("Invalid omit_containers: %s, must be one of %s", app.OmitContainers, GetValidOmitContainers()).(InvalidOmitContainers)
	}

	// Validate mariadb version
	if !IsValidMariaDBVersion(app.MariaDBVersion) {
		return fmt.Errorf("Invalid mariadb_version: %s, must be one of %s", app.MariaDBVersion, GetValidMariaDBVersions()).(invalidMariaDBVersion)
	}

	if app.WebcacheEnabled && app.NFSMountEnabled {
		return fmt.Errorf("webcache_enabled and nfs_mount_enabled cannot both be set to true, use one or the other")
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
	return app.Name + "." + version.DDevTLD
}

// GetHostnames returns an array of all the configured hostnames.
func (app *DdevApp) GetHostnames() []string {

	// Use a map to make sure that we have unique hostnames
	// The value is useless, so just use the int 1 for assignment.
	nameListMap := make(map[string]int)

	nameListMap[app.GetHostname()] = 1

	for _, name := range app.AdditionalHostnames {
		nameListMap[name+"."+version.DDevTLD] = 1
	}

	for _, name := range app.AdditionalFQDNs {
		nameListMap[name] = 1
	}

	// Now walk the map and extract the keys into an array.
	nameListArray := make([]string, 0, len(nameListMap))
	for k := range nameListMap {
		nameListArray = append(nameListArray, k)
	}

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
	if _, err := os.Stat(filepath.Join(ddevDir, "nginx-site.conf")); err == nil && app.WebserverType == WebserverNginxFPM {
		util.Warning("Using custom nginx configuration in nginx-site.conf")
		customConfig = true
	}

	if _, err := os.Stat(filepath.Join(ddevDir, "apache", "apache-site.conf")); err == nil && app.WebserverType != WebserverNginxFPM {
		util.Warning("Using custom apache configuration in apache/apache-site.conf")
		customConfig = true
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
		util.Warning("Custom configuration takes effect when container is created, \nusually on start, use 'ddev restart' or 'ddev rm  && ddev start' if you're not seeing it take effect.")
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
	OmitDBA              bool
	OmitSSHAgent         bool
	WebcacheEnabled      bool
	NFSMountEnabled      bool
	NFSSource            string
	IsWindowsFS          bool
}

// RenderComposeYAML renders the contents of docker-compose.yaml.
func (app *DdevApp) RenderComposeYAML() (string, error) {
	var doc bytes.Buffer
	var err error
	templ := template.New("compose template")
	templ, err = templ.Parse(DDevComposeTemplate)
	if err != nil {
		return "", err
	}

	hostDockerInternalIP, err := dockerutil.GetHostDockerInternalIP()
	if err != nil {
		return "", err
	}

	// The fallthrough default for hostDockerInternalIdentifier is the
	// hostDockerInternalHostname == host.docker.internal

	templateVars := composeYAMLVars{
		Name:                 app.Name,
		Plugin:               "ddev",
		AppType:              app.Type,
		MailhogPort:          appports.GetPort("mailhog"),
		DBAPort:              appports.GetPort("dba"),
		DBPort:               appports.GetPort("db"),
		DdevGenerated:        DdevFileSignature,
		HostDockerInternalIP: hostDockerInternalIP,
		ComposeVersion:       version.DockerComposeFileFormatVersion,
		OmitDBA:              util.ArrayContainsString(app.OmitContainers, "dba"),
		OmitSSHAgent:         util.ArrayContainsString(app.OmitContainers, "ddev-ssh-agent"),
		WebcacheEnabled:      app.WebcacheEnabled,
		NFSMountEnabled:      app.NFSMountEnabled,
		NFSSource:            "",
		IsWindowsFS:          runtime.GOOS == "windows",
		MountType:            "bind",
		WebMount:             "../",
	}
	if app.WebcacheEnabled {
		templateVars.MountType = "volume"
		templateVars.WebMount = "webcachevol"
	}
	if app.NFSMountEnabled {
		templateVars.MountType = "volume"
		templateVars.WebMount = "nfsmount"
		templateVars.NFSSource = app.AppRoot
		if runtime.GOOS == "windows" {
			// WinNFSD can only handle a mountpoint like /C/Users/rfay/workspace/d8git
			// and completely chokes in C:\Users\rfay...
			templateVars.NFSSource = dockerutil.MassageWIndowsNFSMount(app.AppRoot)
		}
	}

	err = templ.Execute(&doc, templateVars)
	return doc.String(), err
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

	err := CreateGitIgnore(dir, "import.yaml", "docker-compose.yaml", "db_snapshots", "sequelpro.spf", "import-db", ".bgsync*", "config.*.y*ml")
	if err != nil {
		return fmt.Errorf("failed to create gitignore in %s: %v", dir, err)
	}

	return nil
}

// validateCommandYaml validates command hooks and tasks defined in hooks for config.yaml
func validateCommandYaml(source []byte) error {
	validHooks := []string{
		"pre-start",
		"post-start",
		"pre-import-db",
		"post-import-db",
		"pre-import-files",
		"post-import-files",
	}

	validTasks := []string{
		"exec",
		"exec-host",
	}

	type Validate struct {
		Commands map[string][]map[string]interface{} `yaml:"hooks,omitempty"`
	}
	val := &Validate{}

	err := yaml.Unmarshal(source, val)
	if err != nil {
		return err
	}

	for command, tasks := range val.Commands {
		var match bool
		for _, hook := range validHooks {
			if command == hook {
				match = true
			}
		}
		if !match {
			return fmt.Errorf("invalid command hook %s defined in config.yaml", command)
		}

		for _, taskSet := range tasks {
			for taskName := range taskSet {
				var match bool
				for _, validTask := range validTasks {
					if taskName == validTask {
						match = true
					}
				}
				if !match {
					return fmt.Errorf("invalid task '%s' defined for %s hook in config.yaml", taskName, command)
				}
			}
		}

	}

	return nil
}
