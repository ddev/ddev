package cmd

import (
	"bytes"
	"fmt"
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/util"
	"github.com/fatih/color"
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
	var res bytes.Buffer

	// Only show extended status for running sites.
	if status == ddevapp.SiteRunning {
		// Build our service table.
		t := table.NewWriter()
		t.SetOutputMirror(&res)
		t.AppendHeader(table.Row{"Service", "Status", "URL/Port", "Info"})

		for k, v := range desc["services"].(map[string]map[string]string) {
			// Gather inside networking information
			urlPortParts := []string{}
			if httpsURL, ok := v["https_url"]; ok {
				urlPortParts = append(urlPortParts, httpsURL)
			} else if httpURL, ok := v["http_url"]; ok {
				urlPortParts = append(urlPortParts, httpURL)
			}
			if p, ok := v["exposed_ports"]; ok {
				urlPortParts = append(urlPortParts, "Inside: "+v["full_name"]+":"+p)
			}

			extraInfo := []string{}

			// Get extra info for web container
			if k == "web" {
				extraInfo = append(extraInfo, fmt.Sprintf("PHP %s %s", desc["php_version"], desc["webserver_type"]))
				if desc["nfs_mount_enabled"].(bool) {
					extraInfo = append(extraInfo, fmt.Sprintf("NFS Enabled"))
				}
				if desc["mutagen_enabled"].(bool) {
					extraInfo = append(extraInfo, fmt.Sprintf("Mutagen enabled (%s)", formatStatus(desc["mutagen_status"].(string))))
				}
			}

			// Get extra info for db container
			if k == "db" {
				if _, ok := desc["dbinfo"]; ok {
					dbinfo := desc["dbinfo"].(map[string]interface{})
					if _, ok := dbinfo["mariadb_version"]; ok {
						extraInfo = append(extraInfo, "MariaDB "+dbinfo["mariadb_version"].(string))
					} else if _, ok := dbinfo["mysql_version"]; ok {
						extraInfo = append(extraInfo, "MySQL"+dbinfo["mysql_version"].(string))
					} else {
						extraInfo = append(extraInfo, dbinfo["database_type"].(string))
					}
				}
			}

			t.AppendRow(table.Row{k, formatStatus(v["status"]), strings.Join(urlPortParts, "\n"), strings.Join(extraInfo, "\n")})
		}

		// Output our service table.
		t.SetStyle(table.StyleColoredBright)
		t.Render()

		//for k, v := range desc["services"].(map[string]map[string]string) {
		//	if verbose || service == k {
		//		out = out + "\n" + k + " URLs\n------------\n"
		//		other := uitable.New()
		//		if httpsURL, ok := v["https_url"]; ok {
		//			other.AddRow("HTTPS", httpsURL)
		//		} else if httpURL, ok := v["http_url"]; ok {
		//			other.AddRow("HTTP", httpURL)
		//		}
		//		out = out + fmt.Sprintln(other)
		//	}
		//}

		//out = out + "\n" + ddevapp.RenderRouterStatus() + "\t" + ddevapp.RenderSSHAuthStatus()
	}

	return res.String(), nil
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
