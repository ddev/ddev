package cmd

import (
	"fmt"

	"github.com/drud/ddev/pkg/plugins/platform"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// DescribeCommand represents the `ddev config` command
var DescribeCommand = &cobra.Command{
	Use:   "describe [sitename]",
	Short: "Get a detailed description of a running ddev site.",
	Long: `Get a detailed description of a running ddev site. Describe provides basic
information about a ddev site, including its name, location, url, and status.
It also provides details for MySQL connections, and connection information for
additional services like MailHog and phpMyAdmin. You can run 'ddev describe' from
a site directory to stop that site, or you can specify a site to describe by
running 'ddev stop <sitename>.`,
	Run: func(cmd *cobra.Command, args []string) {
		var siteName string

		if len(args) > 1 {
			util.Failed("Too many arguments provided. Please use `ddev describe` or `ddev describe [appname]`")
		}

		if len(args) == 1 {
			siteName = args[0]
		}

		out, err := describeApp(siteName)
		if err != nil {
			util.Failed("Could not describe app: %v", err)
		}
		fmt.Println(out)
	},
}

// describeApp will load and describe the app specified by appName. You may leave appName blank to use the app from the current working directory.
func describeApp(appName string) (string, error) {
	var err error

	app, err := platform.GetActiveApp(appName)
	if err != nil {
		return "", err
	}

	out, err := app.Describe()
	return out, err
}

func init() {
	RootCmd.AddCommand(DescribeCommand)
}
