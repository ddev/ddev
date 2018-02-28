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

// docrootRelPath is the relative path to the docroot where index.php is
var docrootRelPath string

// siteName is the name of the site
var siteName string

// pantheonEnvironment is the environment for pantheon, dev/test/prod
var pantheonEnvironment string

// fallbackPantheonEnvironment is our assumption that "dev" will be available in any case
const fallbackPantheonEnvironment = "dev"

// appType is the ddev app type, like drupal7/drupal8/wordpress
var appType string

// showConfigLocation if set causes the command to show the config location.
var showConfigLocation bool

// ConfigCommand represents the `ddev config` command
var ConfigCommand *cobra.Command = &cobra.Command{
	Use:   "config [provider]",
	Short: "Create or modify a ddev project configuration in the current directory",
	Run: func(cmd *cobra.Command, args []string) {

		appRoot, err := os.Getwd()
		if err != nil {
			util.Failed("Could not determine current working directory: %v", err)
		}

		provider := ddevapp.DefaultProviderName

		if len(args) > 1 {
			output.UserOut.Fatal("Invalid argument detected. Please use 'ddev config' or 'ddev config [provider]' to configure a project.")
		}

		if len(args) == 1 {
			provider = args[0]
		}

		app, err := ddevapp.NewApp(appRoot, provider)
		if err != nil {
			util.Failed("Could not create new config: %v", err)
		}

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
				return
			}
		}

		// If they have not given us any flags, we prompt for full info. Otherwise, we assume they're in control.
		if siteName == "" && docrootRelPath == "" && pantheonEnvironment == "" && appType == "" {
			err = app.PromptForConfig()
			if err != nil {
				util.Failed("There was a problem configuring your project: %v", err)
			}
		} else { // In this case we have to validate the provided items, or set to sane defaults

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

			// pantheonEnvironment must be appropriate, and can only be used with pantheon provider.
			if provider != "pantheon" && pantheonEnvironment != "" {
				util.Failed("--pantheon-environment can only be used with pantheon provider, for example 'ddev config pantheon --pantheon-environment=dev --docroot=docroot'")
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

			prov, _ := app.GetProvider()

			if provider == "pantheon" {
				pantheonProvider := prov.(*ddevapp.PantheonProvider)
				if pantheonEnvironment == "" {
					pantheonEnvironment = fallbackPantheonEnvironment // assume a basic default if they haven't provided one.
				}
				pantheonProvider.SetSiteNameAndEnv(pantheonEnvironment)
			}
			// But pantheon *does* validate "Name"
			appTypeErr := prov.Validate()
			if appTypeErr != nil {
				util.Failed("Failed to validate project name %v and environment %v with provider %v: %v", app.Name, pantheonEnvironment, provider, appTypeErr)
			} else {
				util.Success("Using project name '%s' and environment '%s'.", app.Name, pantheonEnvironment)
			}
			err = app.ConfigFileOverrideAction()
			if err != nil {
				util.Failed("Failed to run ConfigFileOverrideAction: %v", err)
			}

		}
		err = app.WriteConfig()
		if err != nil {
			util.Failed("Could not write ddev config file: %v", err)
		}

		_, _ = app.CreateSettingsFile()

		// If a provider is specified, prompt about whether to do an import after config.
		switch provider {
		case ddevapp.DefaultProviderName:
			util.Success("Configuration complete. You may now run 'ddev start'.")
		default:
			util.Success("Configuration complete. You may now run 'ddev start' or 'ddev pull'")
		}
	},
}

func init() {
	validAppTypes := strings.Join(ddevapp.GetValidAppTypes(), ", ")
	apptypeUsage := fmt.Sprintf("Provide the project type (one of %s). This is autodetected and this flag is necessary only to override the detection.", validAppTypes)
	projectNameUsage := fmt.Sprintf("Provide the project name of project to configure (normally the same as the last part of directory name)")

	ConfigCommand.Flags().StringVarP(&siteName, "projectname", "", "", projectNameUsage)
	ConfigCommand.Flags().StringVarP(&docrootRelPath, "docroot", "", "", "Provide the relative docroot of the project, like 'docroot' or 'htdocs' or 'web', defaults to empty, the current directory")
	ConfigCommand.Flags().StringVarP(&pantheonEnvironment, "pantheon-environment", "", "", "Choose the environment for a Pantheon site (dev/test/prod) (Pantheon-only)")
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
