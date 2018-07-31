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
	"github.com/spf13/pflag"
)

// docrootRelPath is the relative path to the docroot where index.php is
var docrootRelPath string

// siteName is the name of the site
var siteName string

// appType is the ddev app type, like drupal7/drupal8/wordpress
var appType string

// showConfigLocation if set causes the command to show the config locatio
var showConfigLocation bool

// extraFlagsHandlingFunc does specific handling for additional flags, and is different per provider.
var extraFlagsHandlingFunc func(cmd *cobra.Command, args []string, app *ddevapp.DdevApp) error

var providerName = ddevapp.DefaultProviderName

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

	// Find out if flags have been provided
	flagsProvided := false
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		if flag.Changed {
			flagsProvided = true
		}
	})

	if !flagsProvided {
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
	err = provider.Write(app.GetConfigPath("import.yaml"))
	if err != nil {
		util.Failed("Failed to write provider config: %v", err)
	}

	util.Success("Configuration complete. You may now run 'ddev start'.")
}

func init() {
	validAppTypes := strings.Join(ddevapp.GetValidAppTypes(), ", ")
	apptypeUsage := fmt.Sprintf("Provide the project type (one of %s). This is autodetected and this flag is necessary only to override the detection.", validAppTypes)
	projectNameUsage := fmt.Sprintf("Provide the project name of project to configure (normally the same as the last part of directory name)")

	ConfigCommand.Flags().StringVarP(&siteName, "projectname", "", "", projectNameUsage)
	ConfigCommand.Flags().StringVarP(&docrootRelPath, "docroot", "", "", "Provide the relative docroot of the project, like 'docroot' or 'htdocs' or 'web', defaults to empty, the current directory")
	ConfigCommand.Flags().StringVarP(&appType, "projecttype", "", "", apptypeUsage)
	// apptype flag is there for backwards compatibility.
	ConfigCommand.Flags().StringVarP(&appType, "apptype", "", "", apptypeUsage+" This is the same as --projecttype and is included only for backwards compatibility.")
	ConfigCommand.Flags().BoolVarP(&showConfigLocation, "show-config-location", "", false, "Output the location of the config.yaml file if it exists, or error that it doesn't exist.")
	ConfigCommand.Flags().StringVarP(&siteName, "sitename", "", "", projectNameUsage+" This is the same as projectname and is included only for backwards compatibility")
	err := ConfigCommand.Flags().MarkDeprecated("sitename", "The sitename flag is deprecated in favor of --projectname")
	util.CheckErr(err)
	err = ConfigCommand.Flags().MarkDeprecated("apptype", "The apptype flag is deprecated in favor of --projecttype")
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

	// app.Name gets set to basename if not provided, or set to siteName if provided
	if app.Name != "" && siteName == "" { // If we already have a c.Name and no siteName, leave c.Name alone
		// Sorry this is empty but it makes the logic clearer.
	} else if siteName != "" { // if we have a siteName passed in, use it for c.Name
		app.Name = siteName
	} else { // No siteName passed, c.Name not set: use c.Name from the directory
		// nolint: vetshadow
		pwd, err := os.Getwd()
		util.CheckErr(err)
		app.Name = filepath.Base(pwd)
	}

	// docrootRelPath must exist
	if docrootRelPath != "" {
		app.Docroot = docrootRelPath
		if _, err = os.Stat(docrootRelPath); os.IsNotExist(err) {
			util.Failed("The docroot provided (%v) does not exist", docrootRelPath)
		}
	} else if !cmd.Flags().Changed("docroot") {
		app.Docroot = ddevapp.DiscoverDefaultDocroot(app)
	}

	if appType != "" && !ddevapp.IsValidAppType(appType) {
		validAppTypes := strings.Join(ddevapp.GetValidAppTypes(), ", ")
		util.Failed("apptype must be one of %s", validAppTypes)
	}

	detectedApptype := app.DetectAppType()
	fullPath, pathErr := filepath.Abs(app.Docroot)
	if pathErr != nil {
		util.Failed("Failed to get absolute path to Docroot %s: %v", app.Docroot, pathErr)
	}
	if appType == "" || appType == detectedApptype { // Found an app, matches passed-in or no apptype passed
		appType = detectedApptype
		util.Success("Found a %s codebase at %s", detectedApptype, fullPath)
	} else if appType != "" { // apptype was passed, but we found no app at all
		util.Warning("You have specified a project type of %s but no project of that type is found in %s", appType, fullPath)
	} else if appType != "" && detectedApptype != appType { // apptype was passed, app was found, but not the same type
		util.Warning("You have specified a project type of %s but a project of type %s was discovered in %s", appType, detectedApptype, fullPath)
	}
	app.Type = appType

	err = app.ConfigFileOverrideAction()
	if err != nil {
		util.Failed("failed to run ConfigFileOverrideAction: %v", err)
	}

	err = app.WriteConfig()
	if err != nil {
		return fmt.Errorf("could not write ddev config file %s: %v", app.ConfigPath, err)
	}

	_, err = app.CreateSettingsFile()
	if err != nil {
		return fmt.Errorf("Could not write settings file: %v", err)
	}
	return nil
}
