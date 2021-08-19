package cmd

import (
	"fmt"
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/util"
	"github.com/fatih/color"
	"github.com/gosuri/uitable"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

// Define flags for the describe command
var (
	// service is the service for additional output.
	service string
	// verbose is whether we want full output.
	verbose bool
)

// DescribeCommand represents the `ddev config` command
var DescribeCommand = &cobra.Command{
	Use:     "describe [projectname] [--service=name]",
	Aliases: []string{"status", "st", "desc"},
	Short:   "Get a detailed description of a running ddev project.",
	Long: `Get a detailed description of a running ddev project. Describe provides basic
information about a ddev project, including its name, location, url, and status.
It also provides details for MySQL connections, and connection information for
additional services like MailHog and phpMyAdmin. You can run 'ddev describe' from
a project directory to describe that project, or you can specify a project to describe by
running 'ddev describe <projectname>.`,
	Example: "ddev describe\nddev describe <projectname>\nddev describe --service=web\nddev status\nddev st",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 1 {
			util.Failed("Too many arguments provided. Please use 'ddev describe' or 'ddev describe [projectname]'")
		}

		apps, err := getRequestedProjects(args, false)
		if err != nil {
			util.Failed("Failed to describe project(s): %v", err)
		}
		app := apps[0]

		if err := ddevapp.CheckForMissingProjectFiles(app); err != nil {
			util.Failed("Failed to describe %s: %v", app.Name, err)
		}

		desc, err := app.Describe(false)
		if err != nil {
			util.Failed("Failed to describe project %s: %v", app.Name, err)
		}

		renderedDesc, err := renderAppDescribe(app, desc)
		util.CheckErr(err) // We shouldn't ever end up with an unrenderable desc.
		output.UserOut.WithField("raw", desc).Print(renderedDesc)
	},
}

// renderAppDescribe takes the map describing the app and renders it for plain-text output
func renderAppDescribe(app *ddevapp.DdevApp, desc map[string]interface{}) (string, error) {
	status := desc["status"]

	url := ""
	out := ""
	if status == ddevapp.SiteRunning {
		if globalconfig.GetCAROOT() != "" {
			url = desc["httpsurl"].(string)
		} else {
			url = desc["httpurl"].(string)
		}
	}

	// Only show extended status for running sites.
	if status == ddevapp.SiteRunning {
		dockerIP, err := dockerutil.GetDockerIP()
		if err != nil {
			return "", err
		}

		// Build our service table.
		t := table.NewWriter()
		t.SetOutputMirror(output.UserOut.Out)
		t.AppendHeader(table.Row{"Service", "Status", "URL/Port", "Info"})

		// Basic info about the web container.
		webStatus, _ := dockerutil.GetContainerStateByName("web")
		mutagenStat := "mutagen disabled"
		nfsStat := "NFS disabled"
		if desc["mutagen_enabled"].(bool) {
			mutagenStat = fmt.Sprintf("Mutagen enabled (%s)", formatStatus(desc["mutagen_status"].(string)))
		}
		if desc["nfs_mount_enabled"].(bool) {
			nfsStat = "NFS mount enabled"
		}

		t.AppendRow(table.Row{
			"web", webStatus, fmt.Sprintf("%s\nInside: http://localhost", url), fmt.Sprintf("PHP %s %s\n%s\n%s", desc["php_version"].(string), desc["webserver_type"].(string), nfsStat, mutagenStat)})

		// Basic info about the database container.
		var dbinfo map[string]interface{}
		var dbinfoString = ""
		dbStatus, _ := dockerutil.GetContainerStateByName("db")

		if _, ok := desc["dbinfo"]; ok {
			dbinfo = desc["dbinfo"].(map[string]interface{})
			if _, ok := dbinfo["mariadb_version"]; ok {
				dbinfoString = "MariaDB " + dbinfo["mariadb_version"].(string)
			} else if _, ok := dbinfo["mysql_version"]; ok {
				dbinfoString = "MySQL" + dbinfo["mysql_version"].(string)
			} else {
				dbinfoString = dbinfo["database_type"].(string)
			}
		}

		if dbinfo != nil {
			t.AppendRow(table.Row{
				"db", dbStatus, fmt.Sprintf("Host: %s:%d\nInside: %s:%d", dockerIP, dbinfo["published_port"], ddevapp.GetDBHostname(app), 3306), dbinfoString})
		}

		phpmyadminStatus, _ := dockerutil.GetContainerStateByName("dba")

		phpmyadminURL := desc["phpmyadmin_http_url"]
		if _, ok := desc["phpmyadmin_https_url"]; ok {
			phpmyadminURL = desc["phpmyadmin_https_url"]
		}
		t.AppendRow(table.Row{
			"dba", phpmyadminStatus, phpmyadminURL})

		for k, v := range desc["extra_services"].(map[string]map[string]string) {
			url = ""
			if httpsURL, ok := v["https_url"]; ok {
				url = httpsURL
			} else if httpURL, ok := v["http_url"]; ok {
				url = httpURL
			}

			t.AppendRow(table.Row{
				k, formatStatus(v["status"]), url, v["version"]})

		}

		// Output our service table.
		t.SetStyle(table.StyleColoredBright)
		t.Render()

		// Extended info about the web container.
		if verbose || service == "web" {
			urlTable := uitable.New()
			urlTable.MaxColWidth = 80
			for _, url := range desc["urls"].([]string) {
				urlTable.AddRow(url)
			}
			out = out + "\nURLs\n----\n"

			out = out + fmt.Sprintln(urlTable)
		}

		// Extended info about the database container.
		if verbose || service == "db" {
			if dbinfo != nil {
				out = out + "\n" + "MySQL/MariaDB Credentials\n-------------------------\n" + `Username: "db", Password: "db", Default database: "db"` + "\n"
				out = out + "\n" + `or use root credentials when needed: Username: "root", Password: "root"` + "\n\n"

				out = out + fmt.Sprintf("Database hostname and port INSIDE container: %s:3306\n", ddevapp.GetDBHostname(app))
				out = out + fmt.Sprintf("To connect to db server inside container or in project settings files: \nmysql --host=%s --user=db --password=db --database=db\n", ddevapp.GetDBHostname(app))

				out = out + fmt.Sprintf("Database hostname and port from HOST: %s:%d\n", dockerIP, dbinfo["published_port"])
				out = out + fmt.Sprintf("To connect to mysql from your host machine, \nmysql --host=%s --port=%d --user=db --password=db --database=db\n", dockerIP, dbinfo["published_port"])

				// Extended info about MailHog.
				if verbose || service == "mailhog" {
					out = out + "\n" + "MailHog URLs\n------------\n"
					mailhog := uitable.New()
					mailhog.AddRow("HTTPS", desc["mailhog_https_url"])
					mailhog.AddRow("HTTP", desc["mailhog_url"])
					out = out + fmt.Sprintln(mailhog)
				}

				// Extended info about database administration.
				//if _, ok := desc["phpmyadmin_https_url"]; ok {
				//	if verbose || service == "dba" {
				//		out = out + "\n" + "phpMyAdmin URLs\n------------\n"
				//		dba := uitable.New()
				//		if _, ok := desc["phpmyadmin_https_url"]; ok {
				//			dba.AddRow("HTTPS", desc["phpmyadmin_https_url"])
				//		}
				//		if _, ok := desc["phpmyadmin_url"]; ok {
				//			dba.AddRow("HTTP", desc["phpmyadmin_url"])
				//		}
				//		out = out + fmt.Sprintln(dba)
				//	}
				//}

				for k, v := range desc["extra_services"].(map[string]map[string]string) {
					if verbose || service == k {
						out = out + "\n" + k + " URLs\n------------\n"
						other := uitable.New()
						if httpsURL, ok := v["https_url"]; ok {
							other.AddRow("HTTPS", httpsURL)
						} else if httpURL, ok := v["http_url"]; ok {
							other.AddRow("HTTP", httpURL)
						}
						out = out + fmt.Sprintln(other)
					}
				}

				out = out + "\n" + ddevapp.RenderRouterStatus() + "\t" + ddevapp.RenderSSHAuthStatus()
			}
			output = output + fmt.Sprint(other)

			output = output + "\n" + ddevapp.RenderRouterStatus() + "\t" + ddevapp.RenderSSHAuthStatus()
		}
	}

	return out, nil
}

func init() {
	RootCmd.AddCommand(DescribeCommand)
	DescribeCommand.Flags().StringVarP(&service, "service", "s", "", "The service for additional information")
	DescribeCommand.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show extended output for all services")
}

func formatStatus(status string) string {
	formattedStatus := fmt.Sprint(status)
	switch {
	case strings.Contains(formattedStatus, ddevapp.SitePaused):
		formattedStatus = color.YellowString(formattedStatus)
	case strings.Contains(formattedStatus, ddevapp.SiteStopped):
		formattedStatus = color.RedString(formattedStatus)
	case strings.Contains(formattedStatus, ddevapp.SiteDirMissing):
		formattedStatus = color.RedString(formattedStatus)
	case strings.Contains(formattedStatus, ddevapp.SiteConfigMissing):
		formattedStatus = color.RedString(formattedStatus)
	default:
		formattedStatus = color.CyanString(formattedStatus)
	}
	return formattedStatus
}
