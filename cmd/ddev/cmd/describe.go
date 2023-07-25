package cmd

import (
	"bytes"
	"fmt"
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/styles"
	"github.com/ddev/ddev/pkg/util"
	"github.com/ddev/ddev/pkg/version"
	"sort"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
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
additional services like MailHog. You can run 'ddev describe' from
a project directory to describe that project, or you can specify a project to describe by
running 'ddev describe <projectname>'.`,
	Example: "ddev describe\nddev describe <projectname>\nddev status\nddev st",
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
	var out bytes.Buffer

	t := table.NewWriter()
	t.SetOutputMirror(&out)
	styles.SetGlobalTableStyle(t)
	tWidth, _ := nodeps.GetTerminalWidthHeight()
	urlPortWidth := float64(35)
	infoWidth := 30
	urlPortWidthFactor := float64(2.5)
	if tWidth != 0 {
		urlPortWidth = float64(tWidth) / urlPortWidthFactor
		infoWidth = tWidth / 4
	}
	util.Debug("detected terminal width=%v urlPortWidth=%v infoWidth=%v", tWidth, urlPortWidth, infoWidth)
	if !globalconfig.DdevGlobalConfig.SimpleFormatting {
		t.SetColumnConfigs([]table.ColumnConfig{
			{
				Name:     "Service",
				WidthMax: 10,
			},
			{
				Name: "URL/Port",
				//WidthMax: int(urlPortWidth),
			},
			{
				Name:     "Info",
				WidthMax: infoWidth,
			},
		})
	}
	dockerPlatform, err := version.GetDockerPlatform()
	if err != nil {
		util.Warning("Unable to determine docker platform: %v", err)
	}

	router := globalconfig.DdevGlobalConfig.Router
	t.SetTitle(fmt.Sprintf("Project: %s %s %s\nDocker platform: %s\nRouter: %s", app.Name, desc["shortroot"].(string), app.GetPrimaryURL(), dockerPlatform, router))
	t.AppendHeader(table.Row{"Service", "Stat", "URL/Port", "Info"})

	// Only show extended status for running sites.
	if status == ddevapp.SiteRunning {
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

			httpURL := ""
			urlPortParts := []string{}

			switch {
			// Normal case, using ddev-router based URLs
			case !ddevapp.IsRouterDisabled(app):
				if httpsURL, ok := v["https_url"]; ok {
					urlPortParts = append(urlPortParts, httpsURL)
				} else if httpURL, ok = v["http_url"]; ok {
					urlPortParts = append(urlPortParts, httpURL)
				}
			// Gitpod, web container only, using port proxied by gitpod
			case (nodeps.IsGitpod() || nodeps.IsCodespaces()) && k == "web":
				urlPortParts = append(urlPortParts, app.GetPrimaryURL())

			// Router disabled, but not because of gitpod, use direct http url
			case ddevapp.IsRouterDisabled(app):
				httpURL = v["host_http_url"]
				if httpURL != "" {
					urlPortParts = append(urlPortParts, httpURL)
				}
			}

			if p, ok := v["exposed_ports"]; ok {
				urlPortParts = append(urlPortParts, "InDocker: "+v["short_name"]+":"+p)
			}
			if p, ok := v["host_ports"]; ok && p != "" {
				urlPortParts = append(urlPortParts, "Host: 127.0.0.1:"+p)
			}

			extraInfo := []string{}

			// Get extra info for web container
			if k == "web" {
				extraInfo = append(extraInfo, fmt.Sprintf("%s PHP%s\n%s\ndocroot:'%s'", desc["type"], desc["php_version"], desc["webserver_type"], desc["docroot"]))
				if desc["nfs_mount_enabled"].(bool) {
					extraInfo = append(extraInfo, fmt.Sprintf("NFS Enabled"))
				}
				if desc["mutagen_enabled"].(bool) {
					extraInfo = append(extraInfo, fmt.Sprintf("Mutagen enabled (%s)", ddevapp.FormatSiteStatus(desc["mutagen_status"].(string))))
				}
				if v, ok := desc["nodejs_version"].(string); ok {
					extraInfo = append(extraInfo, fmt.Sprintf("NodeJS:%s", v))
				}
			}

			// Get extra info for db container
			if k == "db" {
				extraInfo = append(extraInfo, app.Database.Type+":"+app.Database.Version)
				extraInfo = append(extraInfo, "User/Pass: 'db/db'\nor 'root/root'")
			}
			t.AppendRow(table.Row{k, ddevapp.FormatSiteStatus(v["status"]), strings.Join(urlPortParts, "\n"), strings.Join(extraInfo, "\n")})
		}

		if !ddevapp.IsRouterDisabled(app) {
			// MailHog stanza
			mailhogURL := ""
			if _, ok := desc["mailhog_url"]; ok {
				mailhogURL = desc["mailhog_url"].(string)
			}
			if _, ok := desc["mailhog_https_url"]; ok {
				mailhogURL = desc["mailhog_https_url"].(string)
			}
			t.AppendRow(table.Row{"Mailhog", "", fmt.Sprintf("MailHog: %s\n`ddev launch -m`", mailhogURL)})

			//WebExtraExposedPorts stanza
			for _, extraPort := range app.WebExtraExposedPorts {
				t.AppendRow(table.Row{extraPort.Name, "", fmt.Sprintf("InDocker: localhost:%d https://%s:%d http://%s:%d", extraPort.WebContainerPort, app.GetHostname(), extraPort.HTTPSPort, app.GetHostname(), extraPort.HTTPPort)})
			}

			// All URLs stanza
			_, _, urls := app.GetAllURLs()
			s := strings.Join(urls, ", ")
			urlString := text.WrapSoft(s, int(urlPortWidth))
			t.AppendRow(table.Row{"All URLs", "", urlString})
		}
		bindInfo := []string{}
		if app.BindAllInterfaces {
			bindInfo = append(bindInfo, "bind-all-interfaces ENABLED")
		}
		if globalconfig.DdevGlobalConfig.RouterBindAllInterfaces && !ddevapp.IsRouterDisabled(app) {
			bindInfo = append(bindInfo, "router-bind-all-interfaces ENABLED")
		}
		if len(bindInfo) > 0 {
			t.AppendRow(table.Row{"Network", "", strings.Join(bindInfo, "\n")})
		}
	}

	t.Render()

	return out.String(), nil
}

func init() {
	RootCmd.AddCommand(DescribeCommand)
}
