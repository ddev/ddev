package cmd

import (
	"os"

	"path"

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

// ConfigCommand represents the `ddev config` command
var ConfigCommand = &cobra.Command{
	Use:   "config [provider]",
	Short: "Create or modify a ddev application config in the current directory",
	Run: func(cmd *cobra.Command, args []string) {

		appRoot, err := os.Getwd()
		if err != nil {
			util.Failed("Could not determine current working directory: %v", err)

		}

		provider := ddevapp.DefaultProviderName

		if len(args) > 1 {
			output.UserOut.Fatal("Invalid argument detected. Please use 'ddev config' or 'ddev config [provider]' to configure a site.")
		}

		if len(args) == 1 {
			provider = args[0]
		}

		app, err := ddevapp.NewApp(appRoot, provider)
		if err != nil {
			util.Failed("Could not create new config: %v", err)
		}

		// If they have not given us any flags, we prompt for full info. Otherwise, we assume they're in control.
		if siteName == "" && docrootRelPath == "" && pantheonEnvironment == "" && appType == "" {
			err = app.PromptForConfig()
			if err != nil {
				util.Failed("There was a problem configuring your application: %v", err)
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
				app.Name = path.Base(pwd)
			}

			// docrootRelPath must exist
			if docrootRelPath != "" {
				app.Docroot = docrootRelPath
				if _, err = os.Stat(docrootRelPath); os.IsNotExist(err) {
					util.Failed("The docroot provided (%v) does not exist", docrootRelPath)
				}
			}
			// pantheonEnvironment must be appropriate, and can only be used with pantheon provider.
			if provider != "pantheon" && pantheonEnvironment != "" {
				util.Failed("--pantheon-environment can only be used with pantheon provider, for example 'ddev config pantheon --pantheon-environment=dev --docroot=docroot'")
			}
			if appType != "" && appType != "drupal7" && appType != "drupal8" && appType != "wordpress" {
				util.Failed("apptype must be drupal7, drupal8, or wordpress")
			}

			foundAppType, appTypeErr := ddevapp.DetermineAppType(app.Docroot)
			fullPath, pathErr := filepath.Abs(app.Docroot)
			if pathErr != nil {
				util.Failed("Failed to get absolute path to Docroot %s: %v", app.Docroot, pathErr)
			}
			if appTypeErr == nil && (appType == "" || appType == foundAppType) { // Found an app, matches passed-in or no apptype passed
				appType = foundAppType
				util.Success("Found a %s codebase at %s", foundAppType, fullPath)
			} else if appType != "" && appTypeErr != nil { // apptype was passed, but we found no app at all
				util.Warning("You have specified an apptype of %s but no app of that type is found in %s", appType, fullPath)
			} else if appType != "" && appTypeErr == nil && foundAppType != appType { // apptype was passed, app was found, but not the same type
				util.Warning("You have specified an apptype of %s but an app of type %s was discovered in %s", appType, foundAppType, fullPath)
			} else {
				util.Failed("Failed to determine app type (drupal7/drupal8/wordpress).\nYour docroot %v may be incorrect - looking in directory %v, error=%v", app.Docroot, fullPath, appTypeErr)
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
			appTypeErr = prov.Validate()
			if appTypeErr != nil {
				util.Failed("Failed to validate sitename %v and environment %v with provider %v: %v", app.Name, pantheonEnvironment, provider, appTypeErr)
			} else {
				util.Success("Using pantheon sitename '%s' and environment '%s'.", app.Name, pantheonEnvironment)
			}

		}
		err = app.WriteConfig()
		if err != nil {
			util.Failed("Could not write ddev config file: %v", err)
		}

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
	ConfigCommand.Flags().StringVarP(&siteName, "sitename", "", "", "Provide the sitename of site to configure (normally the same as the directory name)")
	ConfigCommand.Flags().StringVarP(&docrootRelPath, "docroot", "", "", "Provide the relative docroot of the site, like 'docroot' or 'htdocs' or 'web', defaults to empty, the current directory")
	ConfigCommand.Flags().StringVarP(&pantheonEnvironment, "pantheon-environment", "", "", "Choose the environment for a Pantheon site (dev/test/prod) (Pantheon-only)")
	ConfigCommand.Flags().StringVarP(&appType, "apptype", "", "", "Provide the app type (like wordpress or drupal7 or drupal8). This is normally autodetected and this flag is not necessary")

	RootCmd.AddCommand(ConfigCommand)
}
