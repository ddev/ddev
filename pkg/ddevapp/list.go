package ddevapp

import (
	"bytes"
	"time"

	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/styles"
	"github.com/ddev/ddev/pkg/util"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

// ListCommandSettings conains all filters and settings of the `ddev list` command
type ListCommandSettings struct {
	// ActiveOnly, if set, shows only running projects
	ActiveOnly bool

	// Continuous, if set, makes list continuously output
	Continuous bool

	// WrapListTable allow that the text in the table of ddev list wraps instead of cutting it to fit the terminal width
	WrapTableText bool

	// ContinuousSleepTime is time to sleep between reads with --continuous
	ContinuousSleepTime int

	// TypeFilter contains the project type which is then used to filter the project list
	TypeFilter string
}

// List provides the functionality for `ddev list`
// activeOnly if true only shows projects that are currently docker containers
// continuous if true keeps requesting and outputting continuously
// wrapTableText if true the text is wrapped instead of truncated to fit the row length
// continuousSleepTime is the time between reports
func List(settings ListCommandSettings) {
	defer util.TimeTrack()()

	var out bytes.Buffer

	for {
		apps, err := GetProjects(settings.ActiveOnly)
		if err != nil {
			util.Failed("failed getting GetProjects: %v", err)
		}

		appDescs := make([]map[string]interface{}, 0)

		if len(apps) < 1 {
			output.UserOut.WithField("raw", appDescs).Println("No ddev projects were found.")
		} else {
			t := CreateAppTable(&out, settings.WrapTableText)
			for _, app := range apps {
				// Filter by project type
				if settings.TypeFilter != "" && settings.TypeFilter != app.Type {
					continue
				}

				desc, err := app.Describe(true)
				if err != nil {
					util.Error("Failed to describe project %s: %v", app.GetName(), err)
				}
				appDescs = append(appDescs, desc)
				RenderAppRow(t, desc)
			}

			routerStatus, _ := GetRouterStatus()
			var extendedRouterStatus = RenderRouterStatus()
			if nodeps.ArrayContainsString(globalconfig.DdevGlobalConfig.OmitContainersGlobal, globalconfig.DdevRouterContainer) {
				extendedRouterStatus = "disabled"
			}
			router := globalconfig.DdevGlobalConfig.Router
			t.AppendFooter(table.Row{
				"Router", routerStatus, "~/.ddev", globalconfig.GetRouterURL(), router},
			)
			t.Render()
			output.UserOut.WithField("raw", appDescs).Print(out.String())
			if routerStatus != "healthy" {
				rawResult := map[string]string{
					"routerStatus":         routerStatus,
					"extendedRouterStatus": extendedRouterStatus,
				}
				rawResult["routerStatus"] = routerStatus
				rawResult["extendedStatus"] = extendedRouterStatus
				output.UserOut.WithField("raw", rawResult)
			}
		}

		if !settings.Continuous {
			break
		}

		time.Sleep(time.Duration(settings.ContinuousSleepTime) * time.Second)
	}
}

// CreateAppTable will create a new app table for describe and list output
func CreateAppTable(out *bytes.Buffer, wrapTableText bool) table.Writer {
	t := table.NewWriter()
	t.AppendHeader(table.Row{"Name", "Status", "Location", "URL", "Type"})
	termWidth, _ := nodeps.GetTerminalWidthHeight()
	usableWidth := termWidth - 15
	statusWidth := 7 // Maybe just "running"
	nameWidth := 10
	typeWidth := 9 // drupal7, magento2 or wordpress
	locationWidth := 20
	urlWidth := 20
	if termWidth > 80 {
		urlWidth = urlWidth + (termWidth-80)/2
		locationWidth = locationWidth + (termWidth-80)/2
		statusWidth = statusWidth + (termWidth-80)/3
	}
	totUsedWidth := nameWidth + statusWidth + locationWidth + urlWidth + typeWidth
	if !wrapTableText {
		t.SetAllowedRowLength(termWidth)
	}

	util.Debug("detected terminal width=%v usableWidth=%d statusWidth=%d nameWidth=%d locationWidth=%d urlWidth=%d typeWidth=%d totUsedWidth=%d", termWidth, usableWidth, statusWidth, nameWidth, locationWidth, urlWidth, typeWidth, totUsedWidth)
	t.SortBy([]table.SortBy{{Name: "Name"}})

	if !globalconfig.DdevGlobalConfig.SimpleFormatting {

		t.SetColumnConfigs([]table.ColumnConfig{
			{
				Name: "Name",
				//WidthMax: nameWidth,
			},
			{
				Name:     "Status",
				WidthMax: statusWidth,
			},
			{
				Name: "Location",
				//WidthMax: locationWidth,
			},
			{
				Name: "URL",
				//WidthMax: urlWidth,
			},
			{
				Name:             "Type",
				WidthMax:         int(typeWidth),
				WidthMaxEnforcer: text.WrapText,
			},
		})
	}
	styles.SetGlobalTableStyle(t)
	t.SetOutputMirror(out)
	return t
}
