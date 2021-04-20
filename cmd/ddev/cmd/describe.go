package cmd

import (
	"fmt"
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/util"
	"github.com/gosuri/uitable"
	"github.com/spf13/cobra"
)

// DescribeCommand represents the `ddev config` command
var DescribeCommand = &cobra.Command{
	Use:     "describe [projectname]",
	Aliases: []string{"status", "st", "desc"},
	Short:   "Get a detailed description of a running ddev project.",
	Long: `Get a detailed description of a running ddev project. Describe provides basic
information about a ddev project, including its name, location, url, and status.
It also provides details for MySQL connections, and connection information for
additional services like MailHog and phpMyAdmin. You can run 'ddev describe' from
a project directory to describe that project, or you can specify a project to describe by
running 'ddev describe <projectname>.`,
	Example: "ddev describe\nddev describe <projectname>\nddev status\nddev st",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 1 {
			util.Failed("Too many arguments provided. Please use 'ddev describe' or 'ddev describe [projectname]'")
		}

		projects, err := getRequestedProjects(args, false)
		if err != nil {
			util.Failed("Failed to describe project(s): %v", err)
		}
		project := projects[0]

		if err := ddevapp.CheckForMissingProjectFiles(project); err != nil {
			util.Failed("Failed to describe %s: %v", project.Name, err)
		}

		desc, err := project.Describe(false)
		if err != nil {
			util.Failed("Failed to describe project %s: %v", project.Name, err)
		}

		renderedDesc, err := renderAppDescribe(desc)
		util.CheckErr(err) // We shouldn't ever end up with an unrenderable desc.
		output.UserOut.WithField("raw", desc).Print(renderedDesc)
	},
}

// renderAppDescribe takes the map describing the app and renders it for plain-text output
func renderAppDescribe(desc map[string]interface{}) (string, error) {

	var output string

	status := desc["status"]

	appTable := ddevapp.CreateAppTable()
	ddevapp.RenderAppRow(appTable, desc)
	output = fmt.Sprint(appTable)

	// Only show extended status for running sites.
	if status == ddevapp.SiteRunning {
		output = output + "\n\nProject Information\n-------------------\n"
		siteInfo := uitable.New()
		siteInfo.AddRow("PHP version:", desc["php_version"])
		siteInfo.AddRow("NFS mount enabled:", desc["nfs_mount_enabled"])
		var dbinfo map[string]interface{}
		if _, ok := desc["dbinfo"]; ok {
			dbinfo = desc["dbinfo"].(map[string]interface{})
			siteInfo.AddRow("Database type:", dbinfo["database_type"])
			if _, ok := dbinfo["mariadb_version"]; ok {
				siteInfo.AddRow("MariaDB version:", dbinfo["mariadb_version"])
			}
			if _, ok := dbinfo["mysql_version"]; ok {
				siteInfo.AddRow("MySQL version:", dbinfo["mysql_version"])
			}
		}

		output = output + fmt.Sprintln(siteInfo)
		urlTable := uitable.New()
		urlTable.MaxColWidth = 80
		for _, url := range desc["urls"].([]string) {
			urlTable.AddRow(url)
		}
		output = output + "\nURLs\n----\n"

		output = output + fmt.Sprintln(urlTable)

		dockerIP, err := dockerutil.GetDockerIP()
		if err != nil {
			return "", err
		}

		if dbinfo != nil {
			output = output + "\n" + "MySQL/MariaDB Credentials\n-------------------------\n" + `Username: "db", Password: "db", Default database: "db"` + "\n"
			output = output + "\n" + `or use root credentials when needed: Username: "root", Password: "root"` + "\n\n"

			output = output + "Database hostname and port INSIDE container: db:3306\n"
			output = output + fmt.Sprintf("To connect to db server inside container or in project settings files: \nmysql --host=db --user=db --password=db --database=db\n")

			output = output + fmt.Sprintf("Database hostname and port from HOST: %s:%d\n", dockerIP, dbinfo["published_port"])
			output = output + fmt.Sprintf("To connect to mysql from your host machine, \nmysql --host=%s --port=%d --user=db --password=db --database=db\n", dockerIP, dbinfo["published_port"])
		} else {
			output = output + "\n" + "DB container is excluded, so no db information provided\n"
		}

		output = output + "\nOther Services\n--------------\n"
		other := uitable.New()
		other.AddRow("MailHog (https):", desc["mailhog_https_url"])
		other.AddRow("MailHog:", desc["mailhog_url"])
		if _, ok := desc["phpmyadmin_https_url"]; ok {
			other.AddRow("phpMyAdmin (https):", desc["phpmyadmin_https_url"])
		}
		if _, ok := desc["phpmyadmin_url"]; ok {
			other.AddRow("phpMyAdmin:", desc["phpmyadmin_url"])
		}
		for k, v := range desc["extra_services"].(map[string]map[string]string) {
			if httpsURL, ok := v["https_url"]; ok {
				other.AddRow(k+" (https):", httpsURL)
			}
			if httpURL, ok := v["http_url"]; ok {
				other.AddRow(k+":", httpURL)
			}
		}
		output = output + fmt.Sprint(other)

		output = output + "\n" + ddevapp.RenderRouterStatus() + "\t" + ddevapp.RenderSSHAuthStatus()
	}

	return output, nil
}

func init() {
	RootCmd.AddCommand(DescribeCommand)
}
