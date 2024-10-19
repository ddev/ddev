package cmd

import (
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// Get implements the ddev get command
var Get = &cobra.Command{
	Use:    "get <addonOrURL> [project]",
	Hidden: true,
	Short:  "Get/Download a 3rd party add-on (service, provider, etc.)",
	Long:   `Get/Download a 3rd party add-on (service, provider, etc.). This can be a GitHub repo, in which case the latest release will be used, or it can be a link to a .tar.gz in the correct format (like a particular release's .tar.gz) or it can be a local directory. Use 'ddev get --list' or 'ddev get --list --all' to see a list of available add-ons. Without --all it shows only official DDEV add-ons. To list installed add-ons, 'ddev get --installed', to remove an add-on 'ddev get --remove <add-on>'.`,
	Example: `ddev get ddev/ddev-redis
ddev get ddev/ddev-redis --version v1.0.4
ddev get https://github.com/ddev/ddev-drupal-solr/archive/refs/tags/v1.2.3.tar.gz
ddev get /path/to/package
ddev get /path/to/tarball.tar.gz
ddev get --list
ddev get --list --all
ddev get --installed
ddev get --remove someaddonname,
ddev get --remove someowner/ddev-someaddonname,
ddev get --remove ddev-someaddonname
`,
	Run: func(cmd *cobra.Command, args []string) {
		// Create deprecation notice.
		// Note we don't just set this as the "Deprecated" property of the cobra.Command
		// because that breaks JSON output
		if !output.JSONOutput {
			var replaceCommand string
			if cmd.Flags().Changed("installed") {
				replaceCommand = " list --installed"
			} else if cmd.Flags().Changed("list") {
				replaceCommand = " list"
				if cmd.Flags().Changed("all") {
					replaceCommand += " --all"
				}
			} else if cmd.Flags().Changed("remove") {
				replaceCommand = " remove"
			} else {
				replaceCommand = " get"
			}
			output.UserErr.Printf("Command \"get\" is deprecated, use 'ddev add-on%s' instead", replaceCommand)
		}

		// Hand off execution for ddev get --list and ddev get --list --all
		if cmd.Flags().Changed("list") {
			AddonListCmd.Run(cmd, args)
			return
		}

		// Hand off execution for ddev get --installed
		if cmd.Flags().Changed("installed") {
			AddonListCmd.Run(cmd, args)
			return
		}

		// Hand off execution for ddev get --remove
		if cmd.Flags().Changed("remove") {
			AddonRemoveCmd.Run(cmd, []string{cmd.Flag("remove").Value.String()})
			return
		}

		if len(args) < 1 {
			util.Failed("You must specify an add-on to download")
		}
		if len(args) > 1 {
			// Update --project flag if projects were passed as args.
			apps, err := getRequestedProjects(args[1:], false)
			if err != nil {
				util.Failed("Unable to get project(s) %v: %v", args, err)
			}
			if len(apps) > 0 {
				err = cmd.Flag("project").Value.Set(apps[0].GetName())
				if err != nil {
					util.Failed("Unable to set --project flag %v: %v", apps[0].GetName(), err)
				}
			}
		}
		// Hand off execution for ddev get <add-on>
		AddonGetCmd.Run(cmd, []string{args[0]})
	},
}

func init() {
	Get.Flags().Bool("list", true, `List available add-ons for 'ddev get'`)
	Get.Flags().Bool("all", true, `List unofficial add-ons for 'ddev get' in addition to the official ones`)
	Get.Flags().Bool("installed", true, `Show installed ddev-get add-ons`)
	Get.Flags().String("remove", "", `Remove a ddev-get add-on`)
	Get.Flags().String("version", "", `Specify a particular version of add-on to install`)
	Get.Flags().BoolP("verbose", "v", false, "Extended/verbose output for ddev get")
	Get.Flags().String("project", "", "Name of the project to install the add-on in")
	RootCmd.AddCommand(Get)
}
