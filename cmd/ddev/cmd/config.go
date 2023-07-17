package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/ddev/ddev/pkg/config/types"
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// Define flags for the config command
var (
	// createDocroot will allow a nonexistent docroot to be created
	createDocroot bool

	// docrootRelPathArg is the relative path to the docroot where index.php is.
	docrootRelPathArg string

	// composerRootRelPathArg allows a user to define the composer root directory for the web service.
	composerRootRelPathArg string

	// composerRootRelPathDefaultArg allows a user to unset a web service composer root directory override.
	composerRootRelPathDefaultArg bool

	// projectNameArg is the name of the site.
	projectNameArg string

	// projectTypeArg is the ddev app type, like drupal7/drupal8/wordpress.
	projectTypeArg string

	// phpVersionArg overrides the default version of PHP to be used in the web container, like 5.6/7.0/7.1/7.2/7.3/7.4/8.0/8.1/8.2/etc.
	phpVersionArg string

	// httpPortArg overrides the default HTTP port (80).
	httpPortArg string

	// httpsPortArg overrides the default HTTPS port (443).
	httpsPortArg string

	// xdebugEnabledArg allows a user to enable XDebug from a command flag.
	xdebugEnabledArg bool

	// noProjectMountArg allows a user to skip the project mount from a command flag.
	noProjectMountArg bool

	// additionalHostnamesArg allows a user to provide a comma-delimited list of hostnames from a command flag.
	additionalHostnamesArg string

	// additionalFQDNsArg allows a user to provide a comma-delimited list of FQDNs from a command flag.
	additionalFQDNsArg string

	// showConfigLocation, if set, causes the command to show the config location.
	showConfigLocation bool

	// webserverTypeArgs allows a user to set the project's webserver type
	webserverTypeArg string

	// webImageArg allows a user to set the project's web server container image
	webImageArg string

	// webImageDefaultArg allows a user to unset the specific web server container image
	webImageDefaultArg bool

	// dbImageArg allows a user to set the project's db server container image
	dbImageArg string

	// dbImageDefaultArg allows a user to use the specific db server container image
	dbImageDefaultArg bool

	// imageDefaultsArg allows a user to unset all specific container images
	imageDefaultsArg bool

	// webWorkingDirArg allows a user to define the working directory for the web service
	webWorkingDirArg string

	// webWorkingDirDefaultArg allows a user to unset a web service working directory override
	webWorkingDirDefaultArg bool

	// dbWorkingDirArg allows a user to define the working directory for the db service
	dbWorkingDirArg string

	// defaultDbaWorkingDirArg allows a user to unset a db service working directory override
	dbWorkingDirDefaultArg bool

	// workingDirDefaultsArg allows a user to unset all service working directory overrides
	workingDirDefaultsArg bool

	// omitContainersArg allows user to determine value of omit_containers
	omitContainersArg string

	// failOnHookFail sets fail_on_hook_fail
	failOnHookFail bool

	// hostDBPortArg sets host_db_port
	hostDBPortArg string

	// hostWebserverPortArg sets host_webserver_port
	hostWebserverPortArg string

	// hostHTTPSPortArg sets host_https_port
	hostHTTPSPortArg string

	// mailhogPortArg is arg for mailhog port
	mailhogPortArg      string
	mailhogHTTPSPortArg string

	// projectTLDArg specifies a project top-level-domain; defaults to ddevapp.DdevDefaultTLD
	projectTLDArg string

	// useDNSWhenPossibleArg specifies to use DNS for lookup (or not), defaults to true
	useDNSWhenPossibleArg bool

	// ngrokArgs provides additional args to the ngrok command in `ddev share`
	ngrokArgs string

	webEnvironmentLocal string
)

var providerName = nodeps.ProviderDefault

// extraFlagsHandlingFunc does specific handling for additional flags, and is different per provider.
var extraFlagsHandlingFunc func(cmd *cobra.Command, args []string, app *ddevapp.DdevApp) error

// ConfigCommand represents the `ddev config` command
var ConfigCommand = &cobra.Command{
	Use:     "config [provider or 'global']",
	Short:   "Create or modify a ddev project configuration in the current directory",
	Example: `"ddev config" or "ddev config --docroot=web  --project-type=drupal8"`,
	Args:    cobra.ExactArgs(0),
	Run:     handleConfigRun,
}

// handleConfigRun handles all the flag processing for any provider
func handleConfigRun(cmd *cobra.Command, args []string) {
	app, err := getConfigApp(providerName)
	if err != nil {
		util.Failed(err.Error())
	}

	homeDir, _ := os.UserHomeDir()
	if app.AppRoot == filepath.Dir(globalconfig.GetGlobalDdevDir()) || app.AppRoot == homeDir {
		util.Failed("Please do not use `ddev config` in your home directory")
	}

	err = app.CheckExistingAppInApproot()
	if err != nil {
		util.Failed(err.Error())
	}

	err = app.ProcessHooks("pre-config")
	if err != nil {
		util.Failed(err.Error())
		util.Failed("Failed to process hook 'pre-config'")
	}

	// If no flags are provided, prompt for configuration
	if cmd.Flags().NFlag() == 0 {
		err = app.PromptForConfig()
		if err != nil {
			util.Failed("There was a problem configuring your project: %v", err)
		}
	} else {
		err = handleMainConfigArgs(cmd, args, app)
		if err != nil {
			util.Failed(err.Error())
		}
		if extraFlagsHandlingFunc != nil {
			err = extraFlagsHandlingFunc(cmd, args, app)
			if err != nil {
				util.Failed("failed to handle per-provider extra flags: %v", err)
			}
		}
	}

	err = app.WriteConfig()
	if err != nil {
		util.Failed("Failed to write config: %v", err)
	}

	_, err = app.CreateSettingsFile()
	if err != nil {
		util.Warning("Could not write settings file: %v", err)
	}

	err = app.ProcessHooks("post-config")
	if err != nil {
		util.Failed("Failed to process hook 'post-config'")
	}

	util.Success("Configuration complete. You may now run 'ddev start'.")
}

func init() {
	var err error

	validAppTypes := strings.Join(ddevapp.GetValidAppTypes(), ", ")
	projectTypeUsage := fmt.Sprintf("Provide the project type (one of %s). This is autodetected and this flag is necessary only to override the detection.", validAppTypes)
	projectNameUsage := fmt.Sprintf("Provide the project name of project to configure (normally the same as the last part of directory name)")

	ConfigCommand.Flags().StringVar(&projectNameArg, "project-name", "", projectNameUsage)
	ConfigCommand.Flags().StringVar(&docrootRelPathArg, "docroot", "", "Provide the relative docroot of the project, like 'docroot' or 'htdocs' or 'web', defaults to empty, the current directory")
	ConfigCommand.Flags().StringVar(&composerRootRelPathArg, "composer-root", "", "Overrides the default composer root directory for the web service")
	ConfigCommand.Flags().BoolVar(&composerRootRelPathDefaultArg, "composer-root-default", false, "Unsets a web service composer root directory override")
	ConfigCommand.Flags().StringVar(&projectTypeArg, "project-type", "", projectTypeUsage)
	ConfigCommand.Flags().StringVar(&phpVersionArg, "php-version", "", "The version of PHP that will be enabled in the web container")
	ConfigCommand.Flags().StringVar(&httpPortArg, "http-port", "", "The router HTTP port for this project")
	ConfigCommand.Flags().StringVar(&httpPortArg, "router-http-port", "", "The router HTTP port for this project")
	ConfigCommand.Flags().StringVar(&httpsPortArg, "https-port", "", "The router HTTPS port for this project")
	ConfigCommand.Flags().StringVar(&httpsPortArg, "router-https-port", "", "The router HTTPS port for this project")
	ConfigCommand.Flags().BoolVar(&xdebugEnabledArg, "xdebug-enabled", false, "Whether or not XDebug is enabled in the web container")
	ConfigCommand.Flags().BoolVar(&noProjectMountArg, "no-project-mount", false, "Whether or not to skip mounting project code into the web container")
	ConfigCommand.Flags().StringVar(&additionalHostnamesArg, "additional-hostnames", "", "A comma-delimited list of hostnames for the project")
	ConfigCommand.Flags().StringVar(&additionalFQDNsArg, "additional-fqdns", "", "A comma-delimited list of FQDNs for the project")
	ConfigCommand.Flags().StringVar(&omitContainersArg, "omit-containers", "", "A comma-delimited list of container types that should not be started when the project is started")
	ConfigCommand.Flags().StringVar(&webEnvironmentLocal, "web-environment", "", `Set the environment variables in the web container: --web-environment="TYPO3_CONTEXT=Development,SOMEENV=someval"`)
	ConfigCommand.Flags().StringVar(&webEnvironmentLocal, "web-environment-add", "", `Append environment variables to the web container: --web-environment="TYPO3_CONTEXT=Development,SOMEENV=someval"`)
	ConfigCommand.Flags().BoolVar(&createDocroot, "create-docroot", false, "Prompts ddev to create the docroot if it doesn't exist")
	ConfigCommand.Flags().BoolVar(&showConfigLocation, "show-config-location", false, "Output the location of the config.yaml file if it exists, or error that it doesn't exist.")
	ConfigCommand.Flags().StringSlice("upload-dirs", []string{}, "Sets the project's upload directories, the destination directories of the import-files command.")
	ConfigCommand.Flags().String("upload-dir", "", "Sets the project's upload directories, the destination directories of the import-files command.")
	_ = ConfigCommand.Flags().MarkDeprecated("upload-dir", "please use --upload-dirs instead")
	ConfigCommand.Flags().StringVar(&webserverTypeArg, "webserver-type", "", "Sets the project's desired webserver type: nginx-fpm/apache-fpm/nginx-gunicorn")
	ConfigCommand.Flags().StringVar(&webImageArg, "web-image", "", "Sets the web container image")
	ConfigCommand.Flags().BoolVar(&webImageDefaultArg, "web-image-default", false, "Sets the default web container image for this ddev version")
	ConfigCommand.Flags().StringVar(&dbImageArg, "db-image", "", "Sets the db container image")
	ConfigCommand.Flags().BoolVar(&dbImageDefaultArg, "db-image-default", false, "Sets the default db container image for this ddev version")
	ConfigCommand.Flags().BoolVar(&imageDefaultsArg, "image-defaults", false, "Sets the default web and db container images")
	ConfigCommand.Flags().StringVar(&webWorkingDirArg, "web-working-dir", "", "Overrides the default working directory for the web service")
	ConfigCommand.Flags().StringVar(&dbWorkingDirArg, "db-working-dir", "", "Overrides the default working directory for the db service")
	ConfigCommand.Flags().BoolVar(&webWorkingDirDefaultArg, "web-working-dir-default", false, "Unsets a web service working directory override")
	ConfigCommand.Flags().BoolVar(&dbWorkingDirDefaultArg, "db-working-dir-default", false, "Unsets a db service working directory override")
	ConfigCommand.Flags().BoolVar(&workingDirDefaultsArg, "working-dir-defaults", false, "Unsets all service working directory overrides")
	ConfigCommand.Flags().Bool("mutagen-enabled", false, "Enable mutagen asynchronous update of project in web container")
	_ = ConfigCommand.Flags().MarkDeprecated("mutagen-enabled", fmt.Sprintf("please use --%s instead", types.FlagPerformanceModeName))
	ConfigCommand.Flags().String(types.FlagPerformanceModeName, types.FlagPerformanceModeDefault, types.FlagPerformanceModeDescription(types.ConfigTypeProject))
	ConfigCommand.Flags().Bool(types.FlagPerformanceModeResetName, true, types.FlagPerformanceModeResetDescription(types.ConfigTypeProject))

	ConfigCommand.Flags().Bool("nfs-mount-enabled", false, "Enable NFS mounting of project in container")
	_ = ConfigCommand.Flags().MarkDeprecated("nfs-mount-enabled", fmt.Sprintf("please use --%s instead", types.FlagPerformanceModeName))
	ConfigCommand.Flags().BoolVar(&failOnHookFail, "fail-on-hook-fail", false, "Decide whether 'ddev start' should be interrupted by a failing hook")
	ConfigCommand.Flags().StringVar(&hostWebserverPortArg, "host-webserver-port", "", "The web container's localhost-bound port")
	ConfigCommand.Flags().StringVar(&hostHTTPSPortArg, "host-https-port", "", "The web container's localhost-bound https port")

	ConfigCommand.Flags().StringVar(&hostDBPortArg, "host-db-port", "", "The db container's localhost-bound port")

	ConfigCommand.Flags().StringVar(&mailhogPortArg, "mailhog-port", "", "Router port to be used for mailhog access")
	ConfigCommand.Flags().StringVar(&mailhogHTTPSPortArg, "mailhog-https-port", "", "Router port to be used for mailhog access (https)")

	// projectname flag exists for backwards compatability.
	ConfigCommand.Flags().StringVar(&projectNameArg, "projectname", "", projectNameUsage)
	err = ConfigCommand.Flags().MarkDeprecated("projectname", "please use --project-name instead")
	util.CheckErr(err)

	// apptype flag exists for backwards compatability.
	ConfigCommand.Flags().StringVar(&projectTypeArg, "projecttype", "", projectTypeUsage)
	err = ConfigCommand.Flags().MarkDeprecated("projecttype", "please use --project-type instead")
	util.CheckErr(err)

	// apptype flag exists for backwards compatibility.
	ConfigCommand.Flags().StringVar(&projectTypeArg, "apptype", "", projectTypeUsage+" This is the same as --project-type and is included only for backwards compatibility.")
	err = ConfigCommand.Flags().MarkDeprecated("apptype", "please use --project-type instead")
	util.CheckErr(err)

	// sitename flag exists for backwards compatibility.
	ConfigCommand.Flags().StringVar(&projectNameArg, "sitename", "", projectNameUsage+" This is the same as project-name and is included only for backwards compatibility")
	err = ConfigCommand.Flags().MarkDeprecated("sitename", "please use --project-name instead")
	util.CheckErr(err)

	ConfigCommand.Flags().String("webimage-extra-packages", "", "A comma-delimited list of Debian packages that should be added to web container when the project is started")

	ConfigCommand.Flags().String("dbimage-extra-packages", "", "A comma-delimited list of Debian packages that should be added to db container when the project is started")

	ConfigCommand.Flags().StringVar(&projectTLDArg, "project-tld", nodeps.DdevDefaultTLD, "set the top-level domain to be used for projects, defaults to "+nodeps.DdevDefaultTLD)

	ConfigCommand.Flags().BoolVarP(&useDNSWhenPossibleArg, "use-dns-when-possible", "", true, "Use DNS for hostname resolution instead of /etc/hosts when possible")

	ConfigCommand.Flags().StringVarP(&ngrokArgs, "ngrok-args", "", "", "Provide extra args to ngrok in ddev share")

	ConfigCommand.Flags().String("timezone", "", "Specify timezone for containers and php, like Europe/London or America/Denver or GMT or UTC")

	ConfigCommand.Flags().Bool("disable-settings-management", false, "Prevent ddev from creating or updating CMS settings files")

	ConfigCommand.Flags().String("composer-version", "", `Specify override for composer version in web container. This may be "", "1", "2", "2.2", "stable", "preview", "snapshot" or a specific version.`)

	ConfigCommand.Flags().Bool("auto", true, `Automatically run config without prompting.`)
	ConfigCommand.Flags().Bool("bind-all-interfaces", false, `Bind host ports on all interfaces, not just on localhost network interface`)
	ConfigCommand.Flags().String("database", "", fmt.Sprintf(`Specify the database type:version to use. Defaults to mariadb:%s`, nodeps.MariaDBDefaultVersion))
	ConfigCommand.Flags().String("nodejs-version", "", fmt.Sprintf(`Specify the nodejs version to use if you don't want the default NodeJS %s`, nodeps.NodeJSDefault))
	ConfigCommand.Flags().Int("default-container-timeout", 120, `default time in seconds that ddev waits for all containers to become ready on start`)
	ConfigCommand.Flags().Bool("disable-upload-dirs-warning", true, `Disable warnings about upload-dirs not being set when using performance-mode=mutagen.`)

	RootCmd.AddCommand(ConfigCommand)

	// Add hidden pantheon subcommand for people who have it in their fingers
	ConfigCommand.AddCommand(&cobra.Command{
		Use:    "pantheon",
		Short:  "ddev config pantheon is no longer needed, see docs",
		Hidden: true,
		Run: func(cmd *cobra.Command, args []string) {
			output.UserOut.Print("`ddev config pantheon` is no longer needed, see docs")
		},
	})
}

// getConfigApp() does the basic setup of the app (with provider) and returns it.
func getConfigApp(_ string) (*ddevapp.DdevApp, error) {
	appRoot, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("could not determine current working directory: %v", err)
	}

	// Check for an existing config in a parent dir
	otherRoot, _ := ddevapp.CheckForConf(appRoot)
	if otherRoot != "" && otherRoot != appRoot {
		return nil, fmt.Errorf("It usually does not make sense to `ddev config` in a subdirectory of an existing project. Is it possible you wanted to `ddev config` in parent directory %s?", otherRoot)
	}
	app, err := ddevapp.NewApp(appRoot, false)
	if err != nil {
		return nil, fmt.Errorf("could not create new config: %v", err)
	}
	return app, nil
}

// handleMainConfigArgs() validates and processes the main config args (docroot, etc.)
func handleMainConfigArgs(cmd *cobra.Command, _ []string, app *ddevapp.DdevApp) error {
	var err error

	// Support the show-config-location flag.
	if showConfigLocation {
		activeApp, err := ddevapp.GetActiveApp("")
		if err != nil {
			if strings.Contains(err.Error(), "Have you run 'ddev config'") {
				util.Failed("No project configuration currently exists")
			} else {
				util.Failed("Failed to access project configuration: %v", err)
			}
		}

		if activeApp.ConfigPath != "" && activeApp.ConfigExists() {
			rawResult := make(map[string]interface{})
			rawResult["configpath"] = activeApp.ConfigPath
			rawResult["approot"] = activeApp.AppRoot

			friendlyMsg := fmt.Sprintf("The project config location is %s", activeApp.ConfigPath)
			output.UserOut.WithField("raw", rawResult).Print(friendlyMsg)
			return nil
		}
	}

	// Let them know if we're replacing the config.yaml
	app.WarnIfConfigReplace()

	// app.Name gets set to basename if not provided, or set to siteNameArg if provided
	// nolint:revive
	if app.Name != "" && projectNameArg == "" { // If we already have a c.Name and no siteNameArg, leave c.Name alone
		// Sorry this is empty but it makes the logic clearer.
	} else if projectNameArg != "" { // if we have a siteNameArg passed in, use it for c.Name
		app.Name = projectNameArg
	} else { // No siteNameArg passed, c.Name not set: use c.Name from the directory
		pwd, err := os.Getwd()
		util.CheckErr(err)
		app.Name = filepath.Base(pwd)
	}

	// Ensure that the docroot exists
	if docrootRelPathArg != "" {
		app.Docroot = docrootRelPathArg
		if _, err = os.Stat(docrootRelPathArg); os.IsNotExist(err) {
			// If the user has indicated that the docroot should be created, create it.
			if !createDocroot {
				util.Failed("The provided docroot %s does not exist. Allow ddev to create it with the --create-docroot flag.", docrootRelPathArg)
			}

			var docrootAbsPath string
			docrootAbsPath, err = filepath.Abs(app.Docroot)
			if err != nil {
				util.Failed("Could not create docroot at %s: %v", docrootRelPathArg, err)
			}

			if err = os.MkdirAll(docrootAbsPath, 0755); err != nil {
				util.Failed("Could not create docroot at %s: %v", docrootAbsPath, err)
			}

			util.Success("Created docroot at %s", docrootAbsPath)
		}
	} else if !cmd.Flags().Changed("docroot") {
		app.Docroot = ddevapp.DiscoverDefaultDocroot(app)
	}

	// Set composer root directory overrides
	if composerRootRelPathArg != "" {
		app.ComposerRoot = composerRootRelPathArg
	}

	if composerRootRelPathDefaultArg {
		app.ComposerRoot = ""
	}

	if projectTypeArg != "" && !ddevapp.IsValidAppType(projectTypeArg) {
		validAppTypes := strings.Join(ddevapp.GetValidAppTypes(), ", ")
		util.Failed("apptype must be one of %s", validAppTypes)
	}

	detectedApptype := app.DetectAppType()
	fullPath, pathErr := filepath.Abs(app.Docroot)
	if pathErr != nil {
		util.Failed("Failed to get absolute path to Docroot %s: %v", app.Docroot, pathErr)
	}

	switch {
	case projectTypeArg != "" && detectedApptype == nodeps.AppTypePHP: // apptype was passed, but we found no app at all
		app.Type = projectTypeArg
		util.Warning("You have specified a project type of '%s' but no project of that type is found in %s", projectTypeArg, fullPath)
	case projectTypeArg != "" && detectedApptype != projectTypeArg: // apptype was passed, app was found, but not the same type
		app.Type = projectTypeArg
		util.Warning("You have specified a project type of '%s' but a project of type '%s' was discovered in %s", projectTypeArg, detectedApptype, fullPath)
	case app.Type != nodeps.AppTypeNone && projectTypeArg == "" && detectedApptype != app.Type: // apptype was not passed, but we found an app of a different type
		util.Warning("A project of type '%s' was found in %s, but the project is configured with type '%s'", detectedApptype, fullPath, app.Type)
	default:
		if projectTypeArg == "" {
			projectTypeArg = detectedApptype
		}

		app.Type = projectTypeArg

		if detectedApptype == nodeps.AppTypePHP {
			util.Success("Configuring unrecognized codebase as project of type '%s' at %s", app.Type, fullPath)
		} else {
			util.Success("Configuring a '%s' codebase with docroot '%s' at %s", detectedApptype, app.Docroot, fullPath)
		}
	}

	// App overrides are done after app type is detected, but
	// before user-defined flags are set.
	err = app.ConfigFileOverrideAction()
	if err != nil {
		util.Failed("failed to run ConfigFileOverrideAction: %v", err)
	}

	if phpVersionArg != "" {
		app.PHPVersion = phpVersionArg
	}

	if httpPortArg != "" {
		app.RouterHTTPPort = httpPortArg
	}

	if httpsPortArg != "" {
		app.RouterHTTPSPort = httpsPortArg
	}

	if hostWebserverPortArg != "" {
		app.HostWebserverPort = hostWebserverPortArg
	}
	if hostHTTPSPortArg != "" {
		app.HostHTTPSPort = hostHTTPSPortArg
	}

	if hostDBPortArg != "" {
		app.HostDBPort = hostDBPortArg
	}

	if cmd.Flag("nfs-mount-enabled").Changed {
		if v, _ := cmd.Flags().GetBool("nfs-mount-enabled"); v {
			app.SetPerformanceMode(types.PerformanceModeNFS)
		}
	}

	if cmd.Flag("mutagen-enabled").Changed {
		if v, _ := cmd.Flags().GetBool("mutagen-enabled"); v {
			app.SetPerformanceMode(types.PerformanceModeMutagen)
		}
	}

	if cmd.Flag(types.FlagPerformanceModeName).Changed {
		performanceMode, _ := cmd.Flags().GetString(types.FlagPerformanceModeName)

		if err := types.CheckValidPerformanceMode(performanceMode, types.ConfigTypeProject); err != nil {
			util.Error("%s. Not changing value of `performance_mode` option.", err)
		} else {
			app.SetPerformanceMode(performanceMode)
		}
	}

	if cmd.Flag(types.FlagPerformanceModeResetName).Changed {
		performanceModeReset, _ := cmd.Flags().GetBool(types.FlagPerformanceModeResetName)

		if performanceModeReset {
			app.SetPerformanceMode(types.PerformanceModeEmpty)
		}
	}

	if cmd.Flag("fail-on-hook-fail").Changed {
		app.FailOnHookFail = failOnHookFail
	}

	// This bool flag is false by default, so only use the value if the flag was explicitly set.
	if cmd.Flag("xdebug-enabled").Changed {
		app.XdebugEnabled = xdebugEnabledArg
	}

	// This bool flag is false by default, so only use the value if the flag was explicitly set.
	if cmd.Flag("no-project-mount").Changed {
		app.NoProjectMount = noProjectMountArg
	}

	if cmd.Flag("mailhog-port").Changed {
		app.MailhogPort = mailhogPortArg
	}
	if cmd.Flag("mailhog-https-port").Changed {
		app.MailhogHTTPSPort = mailhogHTTPSPortArg
	}

	if additionalHostnamesArg != "" {
		app.AdditionalHostnames = strings.Split(additionalHostnamesArg, ",")
	}

	if additionalFQDNsArg != "" {
		app.AdditionalFQDNs = strings.Split(additionalFQDNsArg, ",")
	}

	if omitContainersArg != "" {
		app.OmitContainers = strings.Split(omitContainersArg, ",")
	}

	if cmd.Flag("web-environment").Changed {
		env := strings.TrimSpace(webEnvironmentLocal)
		if env == "" {
			app.WebEnvironment = []string{}
		} else {
			app.WebEnvironment = strings.Split(env, ",")
		}
	}

	if cmd.Flag("web-environment-add").Changed {
		env := strings.TrimSpace(webEnvironmentLocal)
		if env != "" {
			envspl := strings.Split(env, ",")
			conc := append(app.WebEnvironment, envspl...)
			// Convert to a hashmap to remove duplicate values.
			hashmap := make(map[string]string)
			for i := 0; i < len(conc); i++ {
				hashmap[conc[i]] = conc[i]
			}
			keys := []string{}
			for key := range hashmap {
				keys = append(keys, key)
			}
			app.WebEnvironment = keys
			sort.Strings(app.WebEnvironment)
		}
	}

	if cmd.Flag("webimage-extra-packages").Changed {
		if cmd.Flag("webimage-extra-packages").Value.String() == "" {
			app.WebImageExtraPackages = nil
		} else {
			app.WebImageExtraPackages = strings.Split(cmd.Flag("webimage-extra-packages").Value.String(), ",")
		}
	}

	if cmd.Flag("dbimage-extra-packages").Changed {
		if cmd.Flag("dbimage-extra-packages").Value.String() == "" {
			app.DBImageExtraPackages = nil
		} else {
			app.DBImageExtraPackages = strings.Split(cmd.Flag("dbimage-extra-packages").Value.String(), ",")
		}
	}

	if cmd.Flag("use-dns-when-possible").Changed {
		app.UseDNSWhenPossible = useDNSWhenPossibleArg
	}

	if cmd.Flag("ngrok-args").Changed {
		app.NgrokArgs = ngrokArgs
	}

	if cmd.Flag("project-tld").Changed {
		app.ProjectTLD = projectTLDArg
	}

	if cmd.Flag("timezone").Changed {
		app.Timezone, err = cmd.Flags().GetString("timezone")
		if err != nil {
			util.Failed("Incorrect timezone: %v", err)
		}
	}

	if cmd.Flag("composer-version").Changed {
		app.ComposerVersion, err = cmd.Flags().GetString("composer-version")
		if err != nil {
			util.Failed("Incorrect composer-version: %v", err)
		}
	}

	if cmd.Flag("nodejs-version").Changed {
		app.NodeJSVersion, err = cmd.Flags().GetString("nodejs-version")
		if err != nil {
			util.Failed("Incorrect nodejs-version: %v", err)
		}
	}

	if cmd.Flag("disable-settings-management").Changed {
		app.DisableSettingsManagement, _ = cmd.Flags().GetBool("disable-settings-management")
	}

	if cmd.Flag("bind-all-interfaces").Changed {
		app.BindAllInterfaces, _ = cmd.Flags().GetBool("bind-all-interfaces")
	}

	if cmd.Flag("default-container-timeout").Changed {
		t, _ := cmd.Flags().GetInt("default-container-timeout")
		app.DefaultContainerTimeout = strconv.Itoa(t)
		if app.DefaultContainerTimeout == "" {
			app.DefaultContainerTimeout = nodeps.DefaultDefaultContainerTimeout
		}
	}

	if cmd.Flag("database").Changed {
		raw, err := cmd.Flags().GetString("database")
		if err != nil {
			util.Failed("Incorrect value for database: %v", err)
		}
		parts := strings.Split(raw, ":")
		if len(parts) != 2 {
			util.Failed("Incorrect database value: %s - use something like 'mariadb:10.3 or mysql:8.0. Options are %v", raw, nodeps.GetValidDatabaseVersions())
		}
		app.Database.Type = parts[0]
		app.Database.Version = parts[1]
	}

	if cmd.Flag("upload-dir").Changed {
		uploadDirRaw, _ := cmd.Flags().GetString("upload-dir")
		app.UploadDirs = []string{uploadDirRaw}
	}

	if cmd.Flag("upload-dirs").Changed {
		app.UploadDirs, _ = cmd.Flags().GetStringSlice("upload-dirs")
	}

	if cmd.Flag("disable-upload-dirs-warning").Changed {
		app.DisableUploadDirsWarning, _ = cmd.Flags().GetBool("disable-upload-dirs-warning")
	}

	if webserverTypeArg != "" {
		app.WebserverType = webserverTypeArg
	}

	if webImageArg != "" {
		app.WebImage = webImageArg
	}

	if webImageDefaultArg {
		app.WebImage = ""
	}

	if imageDefaultsArg {
		app.WebImage = ""
	}

	if app.WorkingDir == nil {
		app.WorkingDir = map[string]string{}
	}

	// Set working directory overrides
	if webWorkingDirArg != "" {
		app.WorkingDir["web"] = webWorkingDirArg
	}

	if dbWorkingDirArg != "" {
		app.WorkingDir["db"] = dbWorkingDirArg
	}

	// If default working directory overrides are requested, they take precedence
	defaults := app.DefaultWorkingDirMap()
	if workingDirDefaultsArg {
		app.WorkingDir = defaults
	}

	if webWorkingDirDefaultArg {
		app.WorkingDir["web"] = defaults["web"]
	}

	if dbWorkingDirDefaultArg {
		app.WorkingDir["db"] = defaults["db"]
	}

	// Ensure the configuration passes validation before writing config file.
	if err := app.ValidateConfig(); err != nil {
		return fmt.Errorf("failed to validate config: %v", err)
	}

	// If the database already exists in volume and is not of this type, then throw an error
	if !nodeps.ArrayContainsString(app.GetOmittedContainers(), "db") {
		if dbType, err := app.GetExistingDBType(); err != nil || (dbType != "" && dbType != app.Database.Type+":"+app.Database.Version) {
			return fmt.Errorf("Unable to configure project %s with database type %s because that database type does not match the current actual database. Please change your database type back to %s and start again, export, delete, and then change configuration and start. To get back to existing type use 'ddev config --database=%s', and you can try a migration with 'ddev debug migrate-database %s' see docs at %s", app.Name, dbType, dbType, dbType, app.Database.Type+":"+app.Database.Version, "https://ddev.readthedocs.io/en/latest/users/extend/database-types/")
		}
	}

	if err := app.WriteConfig(); err != nil {
		return fmt.Errorf("could not write ddev config file %s: %v", app.ConfigPath, err)
	}

	return nil
}
