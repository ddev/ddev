package cmd

import (
	"fmt"
	"os"
	"strings"

	"path/filepath"

	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// Define flags for the config command
var (
	// createDocroot will allow a nonexistent docroot to be created
	createDocroot bool

	// docrootRelPathArg is the relative path to the docroot where index.php is.
	docrootRelPathArg string

	// siteNameArg is the name of the site.
	siteNameArg string

	// appTypeArg is the ddev app type, like drupal7/drupal8/wordpress.
	appTypeArg string

	// phpVersionArg overrides the default version of PHP to be used in the web container, like 5.6/7.0/7.1/7.2.
	phpVersionArg string

	// httpPortArg overrides the default HTTP port (80).
	httpPortArg string

	// httpsPortArg overrides the default HTTPS port (443).
	httpsPortArg string

	// xdebugEnabledArg allows a user to enable XDebug from a command flag.
	xdebugEnabledArg bool

	// additionalHostnamesArg allows a user to provide a comma-delimited list of hostnames from a command flag.
	additionalHostnamesArg string

	// additionalFQDNsArg allows a user to provide a comma-delimited list of FQDNs from a command flag.
	additionalFQDNsArg string

	// showConfigLocation, if set, causes the command to show the config location.
	showConfigLocation bool

	// uploadDirArg allows a user to set the project's upload directory, the destination directory for import-files.
	uploadDirArg string

	// webserverTypeArgs allows a user to set the project's webserver type
	webserverTypeArg string
)

var providerName = ddevapp.ProviderDefault

// extraFlagsHandlingFunc does specific handling for additional flags, and is different per provider.
var extraFlagsHandlingFunc func(cmd *cobra.Command, args []string, app *ddevapp.DdevApp) error

// ConfigCommand represents the `ddev config` command
var ConfigCommand *cobra.Command = &cobra.Command{
	Use:     "config [provider]",
	Short:   "Create or modify a ddev project configuration in the current directory",
	Example: `"ddev config" or "ddev config --docroot=. --projectname=d7-kickstart --projecttype=drupal7"`,
	Args:    cobra.ExactArgs(0),
	Run:     handleConfigRun,
}

// handleConfigRun handles all the flag processing for any provider
func handleConfigRun(cmd *cobra.Command, args []string) {
	app, err := getConfigApp(providerName)
	if err != nil {
		util.Failed(err.Error())
	}

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

	provider, err := app.GetProvider()
	if err != nil {
		util.Failed("Failed to get provider: %v", err)
	}
	err = provider.Validate()
	if err != nil {
		util.Failed("Failed to validate project name %v: %v", app.Name, err)
	}

	err = app.WriteConfig()
	if err != nil {
		util.Failed("Failed to write config: %v", err)
	}

	_, err = app.CreateSettingsFile()
	if err != nil {
		util.Warning("Could not write settings file: %v", err)
	}

	err = provider.Write(app.GetConfigPath("import.yaml"))
	if err != nil {
		util.Failed("Failed to write provider config: %v", err)
	}

	util.Success("Configuration complete. You may now run 'ddev start'.")
}

func init() {
	var err error

	validAppTypes := strings.Join(ddevapp.GetValidAppTypes(), ", ")
	apptypeUsage := fmt.Sprintf("Provide the project type (one of %s). This is autodetected and this flag is necessary only to override the detection.", validAppTypes)
	projectNameUsage := fmt.Sprintf("Provide the project name of project to configure (normally the same as the last part of directory name)")

	ConfigCommand.Flags().StringVar(&siteNameArg, "projectname", "", projectNameUsage)
	ConfigCommand.Flags().StringVar(&docrootRelPathArg, "docroot", "", "Provide the relative docroot of the project, like 'docroot' or 'htdocs' or 'web', defaults to empty, the current directory")
	ConfigCommand.Flags().StringVar(&appTypeArg, "projecttype", "", apptypeUsage)
	ConfigCommand.Flags().StringVar(&phpVersionArg, "php-version", "", "The version of PHP that will be enabled in the web container")
	ConfigCommand.Flags().StringVar(&httpPortArg, "http-port", "", "The web container's exposed HTTP port")
	ConfigCommand.Flags().StringVar(&httpsPortArg, "https-port", "", "The web container's exposed HTTPS port")
	ConfigCommand.Flags().BoolVar(&xdebugEnabledArg, "xdebug-enabled", false, "Whether or not XDebug is enabled in the web container")
	ConfigCommand.Flags().StringVar(&additionalHostnamesArg, "additional-hostnames", "", "A comma-delimited list of hostnames for the project")
	ConfigCommand.Flags().StringVar(&additionalFQDNsArg, "additional-fqdns", "", "A comma-delimited list of FQDNs for the project")
	ConfigCommand.Flags().BoolVar(&createDocroot, "create-docroot", false, "Prompts ddev to create the docroot if it doesn't exist")
	ConfigCommand.Flags().BoolVar(&showConfigLocation, "show-config-location", false, "Output the location of the config.yaml file if it exists, or error that it doesn't exist.")
	ConfigCommand.Flags().StringVar(&uploadDirArg, "upload-dir", "", "Sets the project's upload directory, the destination directory of the import-files command.")
	ConfigCommand.Flags().StringVar(&webserverTypeArg, "webserver-type", "", "Sets the project's desired webserver type: nginx-fpm, apache-fpm, or apache-cgi")

	// apptype flag exists for backwards compatibility.
	ConfigCommand.Flags().StringVar(&appTypeArg, "apptype", "", apptypeUsage+" This is the same as --projecttype and is included only for backwards compatibility.")
	err = ConfigCommand.Flags().MarkDeprecated("apptype", "The apptype flag is deprecated in favor of --projecttype")
	util.CheckErr(err)

	// sitename flag exists for backwards compatibility.
	ConfigCommand.Flags().StringVar(&siteNameArg, "sitename", "", projectNameUsage+" This is the same as projectname and is included only for backwards compatibility")
	err = ConfigCommand.Flags().MarkDeprecated("sitename", "The sitename flag is deprecated in favor of --projectname")
	util.CheckErr(err)

	RootCmd.AddCommand(ConfigCommand)
}

// getConfigApp() does the basic setup of the app (with provider) and returns it.
func getConfigApp(providerName string) (*ddevapp.DdevApp, error) {
	appRoot, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("could not determine current working directory: %v", err)
	}
	// TODO: Handle case where config may be in parent directories.

	app, err := ddevapp.NewApp(appRoot, providerName)
	if err != nil {
		return nil, fmt.Errorf("could not create new config: %v", err)
	}
	return app, nil
}

// handleMainConfigArgs() validates and processes the main config args (docroot, etc.)
func handleMainConfigArgs(cmd *cobra.Command, args []string, app *ddevapp.DdevApp) error {
	var err error

	// Support the show-config-location flag.
	if showConfigLocation {
		// nolint: vetshadow
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
	if app.Name != "" && siteNameArg == "" { // If we already have a c.Name and no siteNameArg, leave c.Name alone
		// Sorry this is empty but it makes the logic clearer.
	} else if siteNameArg != "" { // if we have a siteNameArg passed in, use it for c.Name
		app.Name = siteNameArg
	} else { // No siteNameArg passed, c.Name not set: use c.Name from the directory
		// nolint: vetshadow
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

	if appTypeArg != "" && !ddevapp.IsValidAppType(appTypeArg) {
		validAppTypes := strings.Join(ddevapp.GetValidAppTypes(), ", ")
		util.Failed("apptype must be one of %s", validAppTypes)
	}

	detectedApptype := app.DetectAppType()
	fullPath, pathErr := filepath.Abs(app.Docroot)
	if pathErr != nil {
		util.Failed("Failed to get absolute path to Docroot %s: %v", app.Docroot, pathErr)
	}
	if appTypeArg == "" || appTypeArg == detectedApptype { // Found an app, matches passed-in or no apptype passed
		appTypeArg = detectedApptype
		util.Success("Found a %s codebase at %s", detectedApptype, fullPath)
	} else if appTypeArg != "" { // apptype was passed, but we found no app at all
		util.Warning("You have specified a project type of %s but no project of that type is found in %s", appTypeArg, fullPath)
	} else if appTypeArg != "" && detectedApptype != appTypeArg { // apptype was passed, app was found, but not the same type
		util.Warning("You have specified a project type of %s but a project of type %s was discovered in %s", appTypeArg, detectedApptype, fullPath)
	}
	app.Type = appTypeArg

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

	// This bool flag is false by default, so only use the value if the flag was explicity set.
	if cmd.Flag("xdebug-enabled").Changed {
		app.XdebugEnabled = xdebugEnabledArg
	}

	if additionalHostnamesArg != "" {
		app.AdditionalHostnames = strings.Split(additionalHostnamesArg, ",")
	}

	if additionalFQDNsArg != "" {
		app.AdditionalFQDNs = strings.Split(additionalFQDNsArg, ",")
	}

	if uploadDirArg != "" {
		app.UploadDir = uploadDirArg
	}

	if webserverTypeArg != "" {
		app.WebserverType = webserverTypeArg
	}

	// Ensure the configuration passes validation before writing config file.
	if err := app.ValidateConfig(); err != nil {
		return fmt.Errorf("failed to validate config: %v", err)
	}

	if err := app.WriteConfig(); err != nil {
		return fmt.Errorf("could not write ddev config file %s: %v", app.ConfigPath, err)
	}

	return nil
}
