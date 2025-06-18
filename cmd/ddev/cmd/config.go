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
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Define flags for the config command
var (
	// docrootRelPathArg is the relative path to the docroot where index.php is.
	docrootRelPathArg string

	// composerRootRelPathArg allows a user to define the Composer root directory for the web service.
	composerRootRelPathArg string

	// composerRootRelPathDefaultArg allows a user to unset a web service Composer root directory override.
	composerRootRelPathDefaultArg bool

	// projectNameArg is the name of the site.
	projectNameArg string

	// projectTypeArg is the DDEV project type.
	projectTypeArg string

	// phpVersionArg overrides the default version of PHP to be used in the web container, like 5.6-8.4 etc.
	phpVersionArg string

	// routerHTTPPortArg overrides the default router HTTP port (80).
	routerHTTPPortArg string

	// routerHTTPSPortArg overrides the default router HTTPS port (443).
	routerHTTPSPortArg string

	// xdebugEnabledArg allows a user to enable Xdebug from a command flag.
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

	// mailpitHTTPPortArg overrides the default Mailpit HTTP port (8025).
	mailpitHTTPPortArg string

	// mailpitHTTPSPortArg overrides the default Mailpit HTTPS port (8026).
	mailpitHTTPSPortArg string

	// projectTLDArg specifies a project top-level-domain; defaults to ddevapp.DdevDefaultTLD
	projectTLDArg string

	// useDNSWhenPossibleArg specifies to use DNS for lookup (or not), defaults to true
	useDNSWhenPossibleArg bool

	// ngrokArgs provides additional args to the ngrok command in `ddev share`
	ngrokArgs string

	webEnvironmentLocal string

	// ddevVersionConstraint sets a ddev version constraint to validate the ddev against
	ddevVersionConstraint string
)

var providerName = nodeps.ProviderDefault

// extraFlagsHandlingFunc does specific handling for additional flags, and is different per provider.
var extraFlagsHandlingFunc func(cmd *cobra.Command, args []string, app *ddevapp.DdevApp) error

// ConfigCommand represents the `ddev config` command
var ConfigCommand = &cobra.Command{
	Use:     "config [provider or 'global']",
	Short:   "Create or modify a DDEV project configuration in the current directory",
	Example: `"ddev config" or "ddev config --docroot=web --project-type=drupal11"`,
	Args:    cobra.ExactArgs(0),
	Run:     handleConfigRun,
}

// handleConfigRun handles all the flag processing for any provider
func handleConfigRun(cmd *cobra.Command, args []string) {
	app := getConfigApp(providerName)

	err := ddevapp.HasAllowedLocation(app)
	if err != nil {
		util.Failed("Unable to run `ddev config`: %v", err)
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
		err = app.WriteConfig()
		if err != nil {
			util.Failed("Failed to write config: %v", err)
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
	validAppTypes := strings.Join(ddevapp.GetValidAppTypes(), ", ")
	projectTypeUsage := fmt.Sprintf("Provide the project type (one of %s). This is autodetected and this flag is necessary only to override the detection.", validAppTypes)
	projectNameUsage := "Provide the project name of project to configure (normally the same as the last part of directory name)"

	ConfigCommand.Flags().StringVar(&projectNameArg, "project-name", "", projectNameUsage)
	ConfigCommand.Flags().StringVar(&docrootRelPathArg, "docroot", "", "Provide the relative docroot of the project, like 'docroot' or 'htdocs' or 'web', defaults to empty, the current directory")
	ConfigCommand.Flags().StringVar(&composerRootRelPathArg, "composer-root", "", "The relative path, from the project root, to the directory containing composer.json (This is where all Composer-related commands are executed.)")
	ConfigCommand.Flags().BoolVar(&composerRootRelPathDefaultArg, "composer-root-default", false, `Unsets a web service Composer root directory override, the same as --composer-root=""`)
	ConfigCommand.Flags().StringVar(&projectTypeArg, "project-type", "", projectTypeUsage)
	_ = ConfigCommand.RegisterFlagCompletionFunc("project-type", configCompletionFunc(ddevapp.GetValidAppTypes()))
	ConfigCommand.Flags().StringVar(&phpVersionArg, "php-version", nodeps.PHPDefault, "PHP version that will be enabled in the web container")
	_ = ConfigCommand.RegisterFlagCompletionFunc("php-version", configCompletionFunc(nodeps.GetValidPHPVersions()))
	ConfigCommand.Flags().StringVar(&routerHTTPPortArg, "router-http-port", nodeps.DdevDefaultRouterHTTPPort, "The router HTTP port for this project")
	_ = ConfigCommand.RegisterFlagCompletionFunc("router-http-port", configCompletionFunc([]string{nodeps.DdevDefaultRouterHTTPPort}))
	ConfigCommand.Flags().StringVar(&routerHTTPSPortArg, "router-https-port", nodeps.DdevDefaultRouterHTTPSPort, "The router HTTPS port for this project")
	_ = ConfigCommand.RegisterFlagCompletionFunc("router-https-port", configCompletionFunc([]string{nodeps.DdevDefaultRouterHTTPSPort}))
	ConfigCommand.Flags().BoolVar(&xdebugEnabledArg, "xdebug-enabled", false, "Whether Xdebug is enabled in the web container")
	_ = ConfigCommand.RegisterFlagCompletionFunc("xdebug-enabled", configCompletionFunc([]string{"true", "false"}))
	ConfigCommand.Flags().BoolVar(&noProjectMountArg, "no-project-mount", false, "Whether to skip mounting project code into the web container")
	_ = ConfigCommand.RegisterFlagCompletionFunc("no-project-mount", configCompletionFunc([]string{"true", "false"}))
	ConfigCommand.Flags().StringVar(&additionalHostnamesArg, "additional-hostnames", "", `Comma-delimited list of project hostnames or --additional-hostnames="" to remove any configured additional hostnames`)
	ConfigCommand.Flags().StringVar(&additionalFQDNsArg, "additional-fqdns", "", `Comma-delimited list of project FQDNs or --additional-fqdns="" to remove any configured FQDNs`)
	ConfigCommand.Flags().StringVar(&omitContainersArg, "omit-containers", "", "Comma-delimited list of container types that should not be started when the project is started")
	_ = ConfigCommand.RegisterFlagCompletionFunc("omit-containers", configCompletionFuncWithCommas(nodeps.GetValidOmitContainers()))
	ConfigCommand.Flags().StringVar(&webEnvironmentLocal, "web-environment", "", `Set the environment variables in the web container: --web-environment="TYPO3_CONTEXT=Development,SOMEENV=someval" or --web-environment="" to remove any previously configured values`)
	ConfigCommand.Flags().StringVar(&webEnvironmentLocal, "web-environment-add", "", `Append environment variables to the web container: --web-environment-add="TYPO3_CONTEXT=Development,SOMEENV=someval"`)
	ConfigCommand.Flags().BoolVar(&showConfigLocation, "show-config-location", false, "Output the location of the .ddev/config.yaml file if it exists, or error that it doesn't exist")
	ConfigCommand.Flags().StringSlice("upload-dirs", []string{}, `Set the project's upload directories, the destination directories of the 'ddev import-files' command, or --upload-dirs="" to remove previously configured values`)
	ConfigCommand.Flags().String("upload-dir", "", "Set the project's upload directories, the destination directories of the import-files command")
	_ = ConfigCommand.Flags().MarkDeprecated("upload-dir", "please use --upload-dirs instead")
	ConfigCommand.Flags().StringVar(&webserverTypeArg, "webserver-type", nodeps.WebserverDefault, fmt.Sprintf("Set the project's desired webserver type: %s", strings.Join(nodeps.GetValidWebserverTypes(), "/")))
	_ = ConfigCommand.RegisterFlagCompletionFunc("webserver-type", configCompletionFunc(nodeps.GetValidWebserverTypes()))
	ConfigCommand.Flags().StringVar(&webImageArg, "web-image", "", "Set the web container image")
	ConfigCommand.Flags().BoolVar(&webImageDefaultArg, "web-image-default", false, `Sets the default web container image, the same as --web-image=""`)
	ConfigCommand.Flags().StringVar(&webWorkingDirArg, "web-working-dir", "", "Override the default working directory for the web service")
	ConfigCommand.Flags().StringVar(&dbWorkingDirArg, "db-working-dir", "", "Override the default working directory for the db service")
	ConfigCommand.Flags().BoolVar(&webWorkingDirDefaultArg, "web-working-dir-default", false, `Unset a web service working directory override, the same as --web-working-dir=""`)
	ConfigCommand.Flags().BoolVar(&dbWorkingDirDefaultArg, "db-working-dir-default", false, `Unset a db service working directory override, the same as --db-working-dir=""`)
	ConfigCommand.Flags().BoolVar(&workingDirDefaultsArg, "working-dir-defaults", false, "Unset all service working directory overrides")
	ConfigCommand.Flags().Bool("mutagen-enabled", false, "Enable Mutagen asynchronous update of project in web container")
	_ = ConfigCommand.Flags().MarkDeprecated("mutagen-enabled", fmt.Sprintf("please use --%s instead", types.FlagPerformanceModeName))
	ConfigCommand.Flags().String(types.FlagPerformanceModeName, types.FlagPerformanceModeDefault, types.FlagPerformanceModeDescription(types.ConfigTypeProject))
	_ = ConfigCommand.RegisterFlagCompletionFunc(types.FlagPerformanceModeName, configCompletionFunc(types.ValidPerformanceModeOptions(types.ConfigTypeProject)))
	ConfigCommand.Flags().Bool(types.FlagPerformanceModeResetName, false, types.FlagPerformanceModeResetDescription(types.ConfigTypeProject))

	ConfigCommand.Flags().String(types.FlagXHProfModeName, types.FlagXHProfModeDefault, types.FlagXHProfModeDescription(types.ConfigTypeProject))
	_ = ConfigCommand.RegisterFlagCompletionFunc(types.FlagXHProfModeName, configCompletionFunc(types.ValidXHProfModeOptions(types.ConfigTypeProject)))
	ConfigCommand.Flags().Bool(types.FlagXHProfModeResetName, false, types.FlagXHProfModeResetDescription(types.ConfigTypeProject))

	ConfigCommand.Flags().Bool("nfs-mount-enabled", false, "Enable NFS mounting of project in container")
	_ = ConfigCommand.Flags().MarkDeprecated("nfs-mount-enabled", fmt.Sprintf("please use --%s instead", types.FlagPerformanceModeName))
	ConfigCommand.Flags().BoolVar(&failOnHookFail, "fail-on-hook-fail", false, "Decide whether 'ddev start' should be interrupted by a failing hook")
	_ = ConfigCommand.RegisterFlagCompletionFunc("fail-on-hook-fail", configCompletionFunc([]string{"true", "false"}))
	ConfigCommand.Flags().StringVar(&hostWebserverPortArg, "host-webserver-port", "", "The web container's localhost-bound HTTP port")
	ConfigCommand.Flags().StringVar(&hostHTTPSPortArg, "host-https-port", "", "The web container's localhost-bound HTTPS port")

	ConfigCommand.Flags().StringVar(&hostDBPortArg, "host-db-port", "", "The db container's localhost-bound port")

	ConfigCommand.Flags().StringVar(&mailpitHTTPPortArg, "mailpit-http-port", nodeps.DdevDefaultMailpitHTTPPort, "Router HTTP port to be used for Mailpit HTTP access")
	_ = ConfigCommand.RegisterFlagCompletionFunc("mailpit-http-port", configCompletionFunc([]string{nodeps.DdevDefaultMailpitHTTPPort}))

	ConfigCommand.Flags().StringVar(&mailpitHTTPSPortArg, "mailpit-https-port", nodeps.DdevDefaultMailpitHTTPSPort, "Router port to be used for Mailpit HTTPS access")
	_ = ConfigCommand.RegisterFlagCompletionFunc("mailpit-https-port", configCompletionFunc([]string{nodeps.DdevDefaultMailpitHTTPSPort}))

	ConfigCommand.Flags().String("webimage-extra-packages", "", `Comma-delimited list of Debian packages that should be added to web container when the project is started or --webimage-extra-packages="" to remove previously configured packages`)

	ConfigCommand.Flags().String("dbimage-extra-packages", "", `Comma-delimited list of Debian packages that should be added to db container when the project is started or --dbimage-extra-packages="" to remove previously configured packages`)

	ConfigCommand.Flags().StringVar(&projectTLDArg, "project-tld", nodeps.DdevDefaultTLD, "Set the top-level domain to be used for projects")
	_ = ConfigCommand.RegisterFlagCompletionFunc("project-tld", configCompletionFunc([]string{nodeps.DdevDefaultTLD}))

	ConfigCommand.Flags().BoolVarP(&useDNSWhenPossibleArg, "use-dns-when-possible", "", true, "Use DNS for hostname resolution instead of /etc/hosts when possible")
	_ = ConfigCommand.RegisterFlagCompletionFunc("use-dns-when-possible", configCompletionFunc([]string{"true", "false"}))

	ConfigCommand.Flags().StringVarP(&ngrokArgs, "ngrok-args", "", "", "Provide extra args to ngrok in ddev share")

	ConfigCommand.Flags().String("timezone", "", "Specify timezone for containers and PHP, like Europe/London or America/Denver or GMT or UTC. If unset, DDEV will attempt to derive it from the host system timezone")

	ConfigCommand.Flags().Bool("disable-settings-management", false, "Prevent DDEV from creating or updating CMS settings files")
	_ = ConfigCommand.RegisterFlagCompletionFunc("disable-settings-management", configCompletionFunc([]string{"true", "false"}))

	ConfigCommand.Flags().String("composer-version", nodeps.ComposerDefault, `Specify override for Composer version in web container. This may be "", "1", "2", "2.2", "stable", "preview", "snapshot" or a specific version`)
	_ = ConfigCommand.RegisterFlagCompletionFunc("composer-version", configCompletionFunc([]string{"2", "2.2", "1", "stable", "preview", "snapshot"}))

	ConfigCommand.Flags().Bool("auto", false, `Automatically run config without prompting`)
	ConfigCommand.Flags().Bool("bind-all-interfaces", false, `Bind host ports on all interfaces, not only on the localhost network interface`)
	_ = ConfigCommand.RegisterFlagCompletionFunc("bind-all-interfaces", configCompletionFunc([]string{"true", "false"}))
	ConfigCommand.Flags().String("database", nodeps.MariaDB+":"+nodeps.MariaDBDefaultVersion, "Specify the database type:version to use")
	_ = ConfigCommand.RegisterFlagCompletionFunc("database", configCompletionFunc(nodeps.GetValidDatabaseVersions()))
	ConfigCommand.Flags().String("nodejs-version", nodeps.NodeJSDefault, "Specify the Node.js version to use")
	_ = ConfigCommand.RegisterFlagCompletionFunc("nodejs-version", configCompletionFunc([]string{nodeps.NodeJSDefault, "auto", "engine"}))
	ConfigCommand.Flags().Int("default-container-timeout", 120, `Default time in seconds that DDEV waits for all containers to become ready on start`)
	_ = ConfigCommand.RegisterFlagCompletionFunc("default-container-timeout", configCompletionFunc([]string{nodeps.DefaultDefaultContainerTimeout}))
	ConfigCommand.Flags().Bool("disable-upload-dirs-warning", false, `Disable warnings about upload-dirs not being set when using --performance-mode=mutagen`)
	_ = ConfigCommand.RegisterFlagCompletionFunc("disable-upload-dirs-warning", configCompletionFunc([]string{"true", "false"}))
	ConfigCommand.Flags().StringVar(&ddevVersionConstraint, "ddev-version-constraint", "", `Specify a ddev_version_constraint to validate ddev against`)
	ConfigCommand.Flags().Bool("corepack-enable", false, `Whether to run 'corepack enable' on Node.js configuration`)
	_ = ConfigCommand.RegisterFlagCompletionFunc("corepack-enable", configCompletionFunc([]string{"true", "false"}))
	ConfigCommand.Flags().Bool("update", false, `Update project settings based on detection and project-type overrides (except for 'generic' type)`)

	// Keep old flag names for backwards compatibility
	var renamedFlags = map[string]string{
		"http-port":          "router-http-port",
		"https-port":         "router-https-port",
		"mailhog-port":       "mailpit-http-port",
		"mailhog-https-port": "mailpit-https-port",
		"projectname":        "project-name",
		"projecttype":        "project-type",
		"apptype":            "project-type",
		"sitename":           "project-name",
		"image-defaults":     "web-image-default",
	}
	ConfigCommand.Flags().SetNormalizeFunc(func(_ *pflag.FlagSet, name string) pflag.NormalizedName {
		if newName, ok := renamedFlags[name]; ok {
			_, _ = fmt.Fprintf(os.Stderr, "Flag --%s has been deprecated, use --%s instead\n", name, newName)
			return pflag.NormalizedName(newName)
		}
		return pflag.NormalizedName(name)
	})

	// Keep removed flags for backwards compatibility
	var removedFlags = []string{
		"create-docroot",
		"db-image",
		"db-image-default",
	}
	for _, removedFlag := range removedFlags {
		// allow any values passed in here
		ConfigCommand.Flags().String(removedFlag, "", "")
		ConfigCommand.Flags().Lookup(removedFlag).NoOptDefVal = "true"
		_ = ConfigCommand.Flags().MarkHidden(removedFlag)
		_ = ConfigCommand.Flags().MarkDeprecated(removedFlag, "it is no longer needed or used")
	}

	RootCmd.AddCommand(ConfigCommand)

	// Add hidden pantheon subcommand for people who have it in their fingers
	ConfigCommand.AddCommand(&cobra.Command{
		Use:    "pantheon",
		Short:  "ddev config pantheon is no longer needed, see docs",
		Hidden: true,
		Run: func(_ *cobra.Command, _ []string) {
			output.UserOut.Print("`ddev config pantheon` is no longer needed, see docs")
		},
	})
}

// getConfigApp() does the basic setup of the app (with provider) and returns it.
func getConfigApp(_ string) *ddevapp.DdevApp {
	appRoot, err := os.Getwd()
	if err != nil {
		util.Failed("Could not determine the current working directory: %v", err)
	}

	// Check for an existing config in a parent dir
	otherRoot, _ := ddevapp.CheckForConf(appRoot)
	if otherRoot != "" && otherRoot != appRoot {
		appRoot = otherRoot
		err = os.Chdir(appRoot)
		if err != nil {
			util.Failed("Unable to chdir to %v: %v", appRoot, err)
		}
	}
	app, err := ddevapp.NewApp(appRoot, false)
	if err != nil {
		util.Failed("Could not create a new config: %v", err)
	}
	if hasConfigNameOverride, newName := app.HasConfigNameOverride(); hasConfigNameOverride {
		app.Name = newName
	}
	return app
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

		if activeApp != nil && activeApp.ConfigPath != "" && activeApp.ConfigExists() {
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
		// Sorry this is empty, but it makes the logic clearer.
	} else if projectNameArg != "" { // if we have a siteNameArg passed in, use it for c.Name
		app.Name = projectNameArg
	} else { // No siteNameArg passed, c.Name not set: use c.Name from the directory
		pwd, err := os.Getwd()
		util.CheckErr(err)
		app.Name = ddevapp.NormalizeProjectName(filepath.Base(pwd))
	}

	err = app.CheckExistingAppInApproot()
	if err != nil {
		util.Failed(err.Error())
	}

	if cmd.Flags().Changed("docroot") {
		if err := ddevapp.ValidateDocroot(docrootRelPathArg); err != nil {
			util.Failed("Failed to validate docroot: %v", err)
		}
		app.Docroot = docrootRelPathArg
		// Ensure that the docroot exists
		if err = app.CreateDocroot(); err != nil {
			util.Failed("Could not create docroot at %s: %v", app.GetAbsDocroot(false), err)
		}
	} else {
		app.Docroot = ddevapp.DiscoverDefaultDocroot(app)
	}

	// Set Composer root directory overrides
	if cmd.Flag("composer-root").Changed {
		app.ComposerRoot = composerRootRelPathArg
	}

	if composerRootRelPathDefaultArg {
		app.ComposerRoot = ""
	}

	if cmd.Flag("project-type").Changed && !ddevapp.IsValidAppType(projectTypeArg) {
		validAppTypes := strings.Join(ddevapp.GetValidAppTypes(), ", ")
		util.Failed("Project type must be one of %s", validAppTypes)
	}

	detectedApptype := app.DetectAppType()
	fullPath := app.GetAbsDocroot(false)

	doUpdate, _ := cmd.Flags().GetBool("update")
	switch {
	case doUpdate:
		if projectTypeArg == "" {
			projectTypeArg = detectedApptype
		}

		app.Type = projectTypeArg
		util.Success("Auto-updating project configuration because update is requested.\nConfiguring a '%s' project with docroot '%s' at '%s'", app.Type, app.Docroot, fullPath)
		err = app.ConfigFileOverrideAction(true)
		if err != nil {
			util.Warning("ConfigOverrideAction failed: %v")
		}
	case app.Type != nodeps.AppTypeNone && app.Type != nodeps.AppTypeGeneric && projectTypeArg == "" && detectedApptype != app.Type: // apptype was not passed, but we found an app of a different type
		util.Warning("A project of type '%s' was found in %s, but the project is configured with type '%s'", detectedApptype, fullPath, app.Type)
	default:
		if projectTypeArg == "" {
			projectTypeArg = detectedApptype
		}

		app.Type = projectTypeArg
		util.Success("Configuring a '%s' project named '%s' with docroot '%s' at '%s'.\nFor full details use 'ddev describe'.", app.Type, app.Name, app.Docroot, fullPath)
	}

	// App overrides are done after app type is detected, but
	// before user-defined flags are set.
	err = app.ConfigFileOverrideAction(false)
	if err != nil {
		util.Failed("Failed to run ConfigFileOverrideAction: %v", err)
	}

	if cmd.Flag("php-version").Changed {
		app.PHPVersion = phpVersionArg
	}

	if cmd.Flag("router-http-port").Changed {
		app.RouterHTTPPort = routerHTTPPortArg
	}

	if cmd.Flag("router-https-port").Changed {
		app.RouterHTTPSPort = routerHTTPSPortArg
	}

	if cmd.Flag("host-webserver-port").Changed {
		app.HostWebserverPort = hostWebserverPortArg
	}
	if cmd.Flag("host-https-port").Changed {
		app.HostHTTPSPort = hostHTTPSPortArg
	}

	if cmd.Flag("host-db-port").Changed {
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

	if cmd.Flag(types.FlagXHProfModeName).Changed {
		xhprofMode, _ := cmd.Flags().GetString(types.FlagXHProfModeName)

		if err := types.CheckValidXHProfMode(xhprofMode, types.ConfigTypeProject); err != nil {
			util.Error("%s. Not changing value of `xhprof_mode` option.", err)
		} else {
			app.XHProfMode = xhprofMode
		}
	}

	if cmd.Flag(types.FlagXHProfModeResetName).Changed {
		xhprofModeReset, _ := cmd.Flags().GetBool(types.FlagXHProfModeResetName)

		if xhprofModeReset {
			app.XHProfMode = types.FlagXHProfModeDefault
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

	if cmd.Flag("mailpit-http-port").Changed {
		app.MailpitHTTPPort = mailpitHTTPPortArg
	}
	if cmd.Flag("mailpit-https-port").Changed {
		app.MailpitHTTPSPort = mailpitHTTPSPortArg
	}

	// Check if the 'additional-hostnames' flag has been set and not default
	app.AdditionalHostnames = processFlag(cmd, "additional-hostnames", app.AdditionalHostnames)

	// Check if the 'additional-fqdns' flag has been set and not default
	app.AdditionalFQDNs = processFlag(cmd, "additional-fqdns", app.AdditionalFQDNs)

	// Check if the 'omit-containers' flag has been set and not default
	app.OmitContainers = processFlag(cmd, "omit-containers", app.OmitContainers)

	if cmd.Flag("web-environment").Changed {
		env := strings.TrimSpace(webEnvironmentLocal)
		if env == "" || env == `""` || env == `''` {
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
		val := cmd.Flag("webimage-extra-packages").Value.String()
		if val == "" || val == `""` || val == `''` {
			app.WebImageExtraPackages = nil
		} else {
			app.WebImageExtraPackages = strings.Split(val, ",")
		}
	}

	if cmd.Flag("dbimage-extra-packages").Changed {
		val := cmd.Flag("dbimage-extra-packages").Value.String()
		if val == "" || val == `""` || val == `''` {
			app.DBImageExtraPackages = nil
		} else {
			app.DBImageExtraPackages = strings.Split(val, ",")
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
			util.Failed("Incorrect database value: %s - use something like 'mariadb:10.11' or 'mysql:8.0'. Options are %v", raw, nodeps.GetValidDatabaseVersions())
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

	if cmd.Flag("corepack-enable").Changed {
		app.CorepackEnable, _ = cmd.Flags().GetBool("corepack-enable")
	}

	if cmd.Flag("webserver-type").Changed {
		app.WebserverType = webserverTypeArg
	}

	if cmd.Flag("web-image").Changed {
		app.WebImage = webImageArg
	}

	if webImageDefaultArg {
		app.WebImage = ""
	}

	if app.WorkingDir == nil {
		app.WorkingDir = map[string]string{}
	}

	// Set working directory overrides
	if cmd.Flag("web-working-dir").Changed {
		app.WorkingDir["web"] = webWorkingDirArg
	}

	if cmd.Flag("db-working-dir").Changed {
		app.WorkingDir["db"] = dbWorkingDirArg
	}

	// If default working directory overrides are requested, they take precedence
	defaults := app.DefaultWorkingDirMap()
	if workingDirDefaultsArg {
		app.WorkingDir = defaults
	}

	if app.WorkingDir["web"] == "" || webWorkingDirDefaultArg {
		app.WorkingDir["web"] = defaults["web"]
	}

	if app.WorkingDir["db"] == "" || dbWorkingDirDefaultArg {
		app.WorkingDir["db"] = defaults["db"]
	}

	if cmd.Flag("ddev-version-constraint").Changed {
		app.DdevVersionConstraint = ddevVersionConstraint
	}

	// Ensure the configuration passes validation before writing config file.
	if err := app.ValidateConfig(); err != nil {
		return fmt.Errorf("failed to validate config: %v", err)
	}

	// If the database already exists in volume and is not of this type, then throw an error
	if !nodeps.ArrayContainsString(app.GetOmittedContainers(), "db") {
		if dbType, err := app.GetExistingDBType(); err != nil || (dbType != "" && dbType != app.Database.Type+":"+app.Database.Version) {
			return fmt.Errorf("unable to configure project %s with database type %s because that database type does not match the current actual database. Please change your database type back to %s and start again, export, delete, and then change configuration and start. To get back to existing type use 'ddev config --database=%s', and you can try a migration with 'ddev debug migrate-database %s' see docs at %s", app.Name, app.Database.Type+":"+app.Database.Version, dbType, dbType, app.Database.Type+":"+app.Database.Version, "https://ddev.readthedocs.io/en/stable/users/extend/database-types/")
		}
	}

	if err := app.WriteConfig(); err != nil {
		return fmt.Errorf("could not write DDEV config file %s: %v", app.ConfigPath, err)
	}

	return nil
}

// processFlag checks if a flag has changed and processes its value accordingly.
func processFlag(cmd *cobra.Command, flagName string, currentValue []string) []string {
	// If the flag hasn't changed, return the current value.
	if !cmd.Flag(flagName).Changed {
		return currentValue
	}

	arg := cmd.Flag(flagName).Value.String()

	// Remove all spaces from the flag value.
	arg = strings.ReplaceAll(arg, " ", "")

	// If the flag value is an empty string, return an empty slice.
	if arg == "" || arg == `""` || arg == `''` {
		return []string{}
	}

	// If the flag value is not an empty string, split it by commas and return the resulting slice.
	return strings.Split(arg, ",")
}

// configCompletionFunc returns a Cobra completion function with static values.
func configCompletionFunc(values []string) func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return values, cobra.ShellCompDirectiveNoFileComp
	}
}

// configCompletionFuncWithCommas returns a Cobra completion function for comma-separated values.
func configCompletionFuncWithCommas(values []string) func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return func(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		entered := strings.Split(toComplete, ",")
		last := entered[len(entered)-1]
		prefix := entered[:len(entered)-1]

		used := make(map[string]bool, len(prefix))
		for _, e := range prefix {
			used[e] = true
		}

		var completions []string
		for _, value := range values {
			if strings.HasPrefix(value, last) && !used[value] {
				completion := append(prefix, value)
				completions = append(completions, strings.Join(completion, ","))
			}
		}
		return completions, cobra.ShellCompDirectiveNoSpace
	}
}
