package cmd

import (
	"bytes"
	"fmt"
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/nodeps"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/util"
	"github.com/fatih/color"
	"github.com/jedib0t/go-pretty/v6/list"
	"github.com/jedib0t/go-pretty/v6/text"
	"sort"
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
		t.AppendHeader(table.Row{"Service", "Stat", "URL/Port", "Info"})

		serviceNames := []string{}
		// Get a list of services in the order we want them, with web and db first
		serviceMap := desc["services"].(map[string]map[string]string)
		for k := range serviceMap {
			if k != "web" && k != "db" {
				serviceNames = append(serviceNames, k)
			}
		}
		sort.Strings(serviceNames)

		if _, ok := desc["dbinfo"]; ok {
			serviceNames = append([]string{"db"}, serviceNames...)
		}
		serviceNames = append([]string{"web"}, serviceNames...)

		for _, k := range serviceNames {
			v := serviceMap[k]

			// Gather inside networking information
			urlPortParts := []string{}
			if httpsURL, ok := v["https_url"]; ok {
				urlPortParts = append(urlPortParts, httpsURL)
			} else if httpURL, ok := v["http_url"]; ok {
				urlPortParts = append(urlPortParts, httpURL)
			}
			if p, ok := v["exposed_ports"]; ok {
				urlPortParts = append(urlPortParts, "InDocker: "+v["full_name"]+":"+p)
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
				extraInfo = append(extraInfo, "User/Pass: 'db/db'\nor 'root/root'")
			}

			t.AppendRow(table.Row{k, formatStatus(v["status"]), strings.Join(urlPortParts, "\n"), strings.Join(extraInfo, "\n")})
		}

		// Output our service table.
		t.SetStyle(table.StyleColoredBright)
		tWidth, _ := nodeps.GetTerminalWidthHeight()
		urlPortWidth := 40
		infoWidth := 30
		if tWidth != 0 {
			urlPortWidth = tWidth / 2
			infoWidth = tWidth / 4
		}
		t.SetColumnConfigs([]table.ColumnConfig{
			{
				Name:     "Service",
				WidthMax: 8,
			},
			{
				Name:     "URL/Port",
				WidthMax: urlPortWidth,
			},
			{
				Name:     "Info",
				WidthMax: infoWidth,
			},
		})
		t.Render()
	}

	_, _ = fmt.Fprintf(&res, text.Bold.Sprint("\nMore Info:\n"))
	l := list.NewWriter()
	l.SetOutputMirror(&res)

	l.AppendItem("Project Name: " + app.Name)
	l.AppendItem("Location: " + desc["shortroot"].(string))
	phpMyAdminURL := ""
	if _, ok := desc["phpmyadmin_url"]; ok {
		phpMyAdminURL = desc["phpmyadmin_url"].(string)
	}
	if _, ok := desc["phpmyadmin_https_url"]; ok {
		phpMyAdminURL = desc["phpmyadmin_https_url"].(string)
	}
	if phpMyAdminURL != "" {
		l.AppendItem(fmt.Sprintf("PHPMyAdmin: %s or `ddev launch -p`", desc["phpmyadmin_url"]))
	}

	mailhogURL := ""
	if _, ok := desc["mailhog_url"]; ok {
		mailhogURL = desc["mailhog_url"].(string)
	}
	if _, ok := desc["mailhog_https_url"]; ok {
		mailhogURL = desc["mailhog_https_url"].(string)
	}

	l.AppendItem(fmt.Sprintf("MailHog: %s or `ddev launch -m`", mailhogURL))
	_, _, urls := app.GetAllURLs()
	l.AppendItem("All URLs:")
	l.Indent()
	for _, url := range urls {
		l.AppendItem(url)
	}
	l.SetStyle(list.StyleBulletCircle)
	l.Render()

	return res.String(), nil
}

func init() {
	RootCmd.AddCommand(DescribeCommand)
	DescribeCommand.Flags().StringVarP(&service, "service", "s", "", "The service for additional information")
	DescribeCommand.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show extended output for all services")
}

func formatStatus(status string) string {
	if status == ddevapp.SiteRunning {
		status = "OK"
	}
	formattedStatus := status

	switch {
	case strings.Contains(status, ddevapp.SitePaused):
		formattedStatus = text.FgYellow.Sprint(formattedStatus)
	case strings.Contains(status, ddevapp.SiteStopped) || strings.Contains(status, ddevapp.SiteDirMissing) || strings.Contains(status, ddevapp.SiteConfigMissing):
		formattedStatus = text.FgRed.Sprint(formattedStatus)
	default:
		formattedStatus = color.CyanString(formattedStatus)
	}
	return formattedStatus
}
