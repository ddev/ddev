package cmd

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/netutil"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/styles"
	"github.com/ddev/ddev/pkg/util"
	"github.com/ddev/ddev/pkg/version"
	"github.com/ddev/ddev/pkg/versionconstants"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/spf13/cobra"
)

// DescribeCommand represents the `ddev config` command
var DescribeCommand = &cobra.Command{
	ValidArgsFunction: ddevapp.GetProjectNamesFunc("all", 1),
	Use:               "describe [projectname]",
	Aliases:           []string{"status", "st", "desc"},
	Short:             "Get a detailed description of a running DDEV project.",
	Long: `Get a detailed description of a running DDEV project. Describe provides basic
information about a DDEV project, including its name, location, url, and status.
It also provides details for MySQL connections, and connection information for
additional services like Mailpit. You can run 'ddev describe' from
a project directory to describe that project, or you can specify a project to describe by
running 'ddev describe <projectname>'.`,
	Example: "ddev describe\nddev describe <projectname>\nddev status\nddev st",
	Run: func(_ *cobra.Command, args []string) {
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

		// Ensure we have all services to describe for never-started projects.
		if !fileutil.FileExists(app.DockerComposeFullRenderedYAMLPath()) {
			_ = app.DockerEnv()
			err = app.WriteDockerComposeYAML()
			if err != nil {
				util.Failed("Failed to run `docker-compose config` for '%s': %v", app.Name, err)
			}
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
	dockerIP, _ := dockerutil.GetDockerIP()
	if dockerIP == "" {
		dockerIP = "127.0.0.1"
	}

	var out bytes.Buffer

	t := table.NewWriter()
	t.SetOutputMirror(&out)
	styles.SetGlobalTableStyle(t, false)
	tWidth, _ := nodeps.GetTerminalWidthHeight()
	urlPortWidth := float64(35)
	infoWidth := 30
	urlPortWidthFactor := float64(2.5)
	if tWidth != 0 {
		urlPortWidth = float64(tWidth) / urlPortWidthFactor
		infoWidth = tWidth / 4
	}
	if !globalconfig.DdevGlobalConfig.SimpleFormatting {
		t.SetColumnConfigs([]table.ColumnConfig{
			{
				Name:     "Service",
				WidthMax: 12,
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
		util.Warning("Unable to determine Docker platform: %v", err)
	}

	router := globalconfig.DdevGlobalConfig.Router
	if nodeps.ArrayContainsString(app.GetOmittedContainers(), `ddev-router`) {
		router = "disabled"
	}

	t.SetTitle(fmt.Sprintf("Project: %s %s %s\nDocker platform: %s\nRouter: %s\nDDEV version: %s", app.Name, desc["shortroot"].(string), app.GetPrimaryURL(), dockerPlatform, router, versionconstants.DdevVersion))
	t.AppendHeader(table.Row{"Service", "Stat", "URL/Port", "Info"})

	// Only show extended status for running sites.
	serviceNames := []string{}
	// Get a list of services in the order we want them, with web and db first
	serviceMap := desc["services"].(map[string]map[string]interface{})
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
		extraInfo := []string{}

		switch {
		// Normal case, using ddev-router based URLs
		case !ddevapp.IsRouterDisabled(app):
			if httpsURL, ok := v["https_url"].(string); ok && !app.CanUseHTTPOnly() {
				urlPortParts = append(urlPortParts, netutil.NormalizeURL(httpsURL))
			} else if httpURL, ok = v["http_url"].(string); ok {
				urlPortParts = append(urlPortParts, netutil.NormalizeURL(httpURL))
			}

		// Codespaces, web container only, using port proxied by Codespaces/Devcontainer
		case nodeps.IsDevcontainer() && k == "web":
			urlPortParts = append(urlPortParts, app.GetPrimaryURL())

		// Router disabled, but not because of Codespaces, use direct http url
		case ddevapp.IsRouterDisabled(app):
			if httpURL, ok := v["host_http_url"].(string); ok && httpURL != "" {
				urlPortParts = append(urlPortParts, netutil.NormalizeURL(httpURL))
			}
		}

		portStr := "InDocker"
		var portMappingDockerHost = map[string]string{}
		if p, ok := v["host_ports_mapping"].([]map[string]string); ok && len(p) != 0 {
			portStr += " -> Host"
			for _, portMapping := range p {
				portMappingDockerHost[portMapping["exposed_port"]] = portMapping["host_port"]
			}
		}
		portStr += ":"

		if p, ok := v["exposed_ports"].(string); ok {
			if p != "" {
				for _, exposedPort := range strings.Split(p, ",") {
					portStr += "\n - " + v["short_name"].(string) + ":" + exposedPort
					if host, ok := portMappingDockerHost[exposedPort]; ok {
						portStr += " -> " + dockerIP + ":" + host
					}
				}
				urlPortParts = append(urlPortParts, portStr)
			} else {
				urlPortParts = append(urlPortParts, portStr+" "+v["short_name"].(string))
			}
		}

		// Get extra info for web container
		if k == "web" {
			projectType := fmt.Sprintf("%s PHP %s", desc["type"], desc["php_version"])
			// For generic type, we don't show PHP version
			if app.WebserverType == nodeps.WebserverGeneric {
				projectType = fmt.Sprintf("%s", desc["type"])
			}
			extraInfo = append(extraInfo, fmt.Sprintf("%s\nServer: %s\nDocroot: '%s'", projectType, desc["webserver_type"], desc["docroot"]))
			extraInfo = append(extraInfo, fmt.Sprintf("Perf mode: %s", desc["performance_mode"].(string)))
			if v, ok := desc["nodejs_version"].(string); ok {
				extraInfo = append(extraInfo, fmt.Sprintf("Node.js: %s", v))
			}
		}

		// Get extra info for db container
		if k == "db" {
			extraInfo = append(extraInfo, app.Database.Type+":"+app.Database.Version)
			extraInfo = append(extraInfo, "User/Pass: 'db/db'\nor 'root/root'")
		}

		// Add x-ddev.describe-url-port to URL/Port column if it exists
		if desc, ok := v["describe-url-port"].(string); ok && desc != "" {
			urlPortParts = append(urlPortParts, desc)
		}

		// Add x-ddev.describe-info to info column if it exists
		if desc, ok := v["describe-info"].(string); ok && desc != "" {
			extraInfo = append(extraInfo, desc)
		}
		status, ok := v["status"].(string)
		if !ok {
			status = ddevapp.SiteStopped
		}
		t.AppendRow(table.Row{k, ddevapp.FormatSiteStatus(status), strings.Join(urlPortParts, "\n"), strings.Join(extraInfo, "\n")})
	}

	if !ddevapp.IsRouterDisabled(app) {
		// Mailpit stanza
		mailpitURL := ""
		if _, ok := desc["mailpit_url"]; ok {
			mailpitURL = desc["mailpit_url"].(string)
		}
		if _, ok := desc["mailpit_https_url"]; ok && !app.CanUseHTTPOnly() {
			mailpitURL = desc["mailpit_https_url"].(string)
		}
		t.AppendRow(table.Row{"Mailpit", "", fmt.Sprintf("Mailpit: %s\nLaunch: ddev mailpit", mailpitURL)})

		ddevapp.SyncGenericWebserverPortsWithRouterPorts(app)

		//WebExtraExposedPorts stanza
		for _, extraPort := range app.WebExtraExposedPorts {
			if app.CanUseHTTPOnly() {
				url := netutil.NormalizeURL(fmt.Sprintf("http://%s:%d", app.GetHostname(), extraPort.HTTPPort))
				t.AppendRow(table.Row{extraPort.Name, "", fmt.Sprintf("%s\nInDocker: web:%d", url, extraPort.WebContainerPort)})
			} else {
				url := netutil.NormalizeURL(fmt.Sprintf("https://%s:%d", app.GetHostname(), extraPort.HTTPSPort))
				t.AppendRow(table.Row{extraPort.Name, "", fmt.Sprintf("%s\nInDocker: web:%d", url, extraPort.WebContainerPort)})
			}
		}

		// All URLs stanza
		_, _, urls := app.GetAllURLs()
		if len(urls) > 0 {
			s := strings.Join(urls, ", ")
			urlString := text.WrapSoft(s, int(urlPortWidth))
			t.AppendRow(table.Row{"Project URLs", "", urlString})
		}
	}
	bindInfo := []string{}
	if app.BindAllInterfaces || dockerutil.IsRemoteDockerHost() {
		bindInfo = append(bindInfo, "bind-all-interfaces ENABLED")
	}
	if (globalconfig.DdevGlobalConfig.RouterBindAllInterfaces || dockerutil.IsRemoteDockerHost()) && !ddevapp.IsRouterDisabled(app) {
		bindInfo = append(bindInfo, "router-bind-all-interfaces ENABLED")
	}
	if len(bindInfo) > 0 {
		t.AppendRow(table.Row{"Network", "", strings.Join(bindInfo, "\n")})
	}
	if !ddevapp.IsRouterDisabled(app) {
		// If there is a problem with the router, add it to the table
		routerStatus, errorInfo := ddevapp.RenderRouterStatus()
		if errorInfo != "" {
			t.AppendFooter(table.Row{
				"Router", routerStatus, text.WrapSoft(errorInfo, int(urlPortWidth)),
			})
		}
	}

	t.Render()

	return out.String(), nil
}

func init() {
	RootCmd.AddCommand(DescribeCommand)
}
