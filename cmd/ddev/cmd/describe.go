package cmd

import (
	"fmt"

	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/util"
	"github.com/gosuri/uitable"
	"github.com/spf13/cobra"
)

// DescribeCommand represents the `ddev config` command
var DescribeCommand = &cobra.Command{
	Use:   "describe [projectname]",
	Short: "Get a detailed description of a running ddev project.",
	Long: `Get a detailed description of a running ddev project. Describe provides basic
information about a ddev project, including its name, location, url, and status.
It also provides details for MySQL connections, and connection information for
additional services like MailHog and phpMyAdmin. You can run 'ddev describe' from
a project directory to stop that project, or you can specify a project to describe by
running 'ddev stop <projectname>.`,
	Run: func(cmd *cobra.Command, args []string) {
		var siteName string

		if len(args) > 1 {
			util.Failed("Too many arguments provided. Please use 'ddev describe' or 'ddev describe [appname]'")
		}

		if len(args) == 1 {
			siteName = args[0]
		}

		site, err := ddevapp.GetActiveApp(siteName)
		if err != nil {
			util.Failed("Unable to find any active project named %s: %v", siteName, err)
		}

		// Do not show any describe output if we can't find the site.
		if site.SiteStatus() == ddevapp.SiteNotFound {
			util.Failed("no project found. have you run 'ddev start'?")
		}

		desc, err := site.Describe()
		if err != nil {
			util.Failed("Failed to describe project %s: %v", err)
		}

		renderedDesc, err := renderAppDescribe(desc)
		util.CheckErr(err) // We shouldn't ever end up with an unrenderable desc.
		output.UserOut.WithField("raw", desc).Print(renderedDesc)
	},
}

// renderAppDescribe takes the map describing the app and renders it for plain-text output
func renderAppDescribe(desc map[string]interface{}) (string, error) {

	maxWidth := uint(200)
	var output string

	appTable := ddevapp.CreateAppTable()
	ddevapp.RenderAppRow(appTable, desc)
	output = fmt.Sprint(appTable)

	output = output + "\n\nProject Information\n-----------------\n"
	siteInfo := uitable.New()
	siteInfo.AddRow("PHP version:", desc["php_version"])
	output = output + fmt.Sprint(siteInfo)

	// Only show extended status for running sites.
	if desc["status"] == ddevapp.SiteRunning {
		output = output + "\n\nMySQL Credentials\n-----------------\n"
		dbTable := uitable.New()

		dbinfo := desc["dbinfo"].(map[string]interface{})

		if _, ok := dbinfo["username"].(string); ok {
			dbTable.MaxColWidth = maxWidth
			dbTable.AddRow("Username:", dbinfo["username"])
			dbTable.AddRow("Password:", dbinfo["password"])
			dbTable.AddRow("Database name:", dbinfo["dbname"])
			dbTable.AddRow("Host:", dbinfo["host"])
			dbTable.AddRow("Port:", dbinfo["port"])
			output = output + fmt.Sprint(dbTable)
			output = output + fmt.Sprintf("\nTo connect to mysql from your host machine, use port %[1]v on 127.0.0.1.\nFor example: mysql --host=127.0.0.1 --port=%[1]v --user=db --password=db --database=db", dbinfo["published_port"])
		}
		output = output + "\n\nOther Services\n--------------\n"
		other := uitable.New()
		other.AddRow("MailHog:", desc["mailhog_url"])
		other.AddRow("phpMyAdmin:", desc["phpmyadmin_url"])
		output = output + fmt.Sprint(other)
	}

	output = output + "\n" + ddevapp.RenderRouterStatus()

	return output, nil
}

func init() {
	RootCmd.AddCommand(DescribeCommand)
}
