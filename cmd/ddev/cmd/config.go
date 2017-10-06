package cmd

import (
	"log"
	"os"

	"path/filepath"

	"path"

	"github.com/drud/ddev/pkg/ddevapp"
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
			util.Failed("Could not determine current working directory: %v\n", err)

		}

		provider := ddevapp.DefaultProviderName

		if len(args) > 1 {
			log.Fatal("Invalid argument detected. Please use 'ddev config' or 'ddev config [provider]' to configure a site.")
		}

		if len(args) == 1 {
			provider = args[0]
		}

		c, err := ddevapp.NewConfig(appRoot, provider)
		if err != nil {
			util.Failed("Could not create new config: %v", err)
		}

		// Set the provider value after load so we can ensure we use the passed in provider value
		// for this configuration.
		c.Provider = provider
		c.Name = siteName
		c.Docroot = docrootRelPath

		// If they have not given us any flags, we prompt for full info. Otherwise, we assume they're in control.
		if siteName == "" && docrootRelPath == "" && pantheonEnvironment == "" && appType == "" {
			err = c.PromptForConfig()
			if err != nil {
				util.Failed("There was a problem configuring your application: %v\n", err)
			}
		} else { // In this case we have to validate the provided items, or set to sane defaults
			// siteName gets set to basename if not provided
			if siteName == "" {
				// nolint: vetshadow
				pwd, err := os.Getwd()
				util.CheckErr(err)
				siteName = path.Base(pwd)
			}
			c.Name = siteName

			// docrootRelPath must exist
			if _, err = os.Stat(docrootRelPath); docrootRelPath != "" && os.IsNotExist(err) {
				util.Failed("The docroot provided (%v) does not exist", docrootRelPath)
			}
			// pantheonEnvironment must be appropriate, and can only be used with pantheon provider.
			if provider != "pantheon" && pantheonEnvironment != "" {
				util.Failed("--pantheon-environment can only be used with pantheon provider, for example ddev config pantheon --pantheon-environment=dev --docroot=docroot")
			}
			if appType != "" && appType != "drupal7" && appType != "drupal8" && appType != "wordpress" {
				util.Failed("apptype must be drupal7 or drupal8 or wordpress")
			}
			// Handle the case where flags have been passed in and we config that way
			if appType == "" {
				appType, err = ddevapp.DetermineAppType(c.Docroot)
				if err != nil {
					fullPath, _ := filepath.Abs(c.Docroot)
					util.Failed("Failed to determine app type (drupal7/drupal8/wordpress).\nYour docroot %v may be incorrect - looking in directory %v, error=%v", c.Docroot, fullPath, err)
				}
			}

			// If we found an application type just set it and inform the user.
			util.Success("Found a %s codebase at %s\n", c.AppType, filepath.Join(c.AppRoot, c.Docroot))
			prov, _ := c.GetProvider()

			c.AppType = appType

			if provider == "pantheon" {
				pantheonProvider := prov.(*ddevapp.PantheonProvider)
				if pantheonEnvironment == "" {
					pantheonEnvironment = fallbackPantheonEnvironment // assume a basic default if they haven't provided one.
				}
				pantheonProvider.SetSiteNameAndEnv(pantheonEnvironment)
			}
			// But pantheon *does* validate "Name"
			err = prov.Validate()
			if err != nil {
				util.Failed("Failed to validate sitename %v and environment %v with provider %v: %v", c.Name, pantheonEnvironment, provider, err)
			}

		}
		err = c.Write()
		if err != nil {
			util.Failed("Could not write ddev config file: %v\n", err)

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
