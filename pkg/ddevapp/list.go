package ddevapp

import (
	"bytes"
	"time"

	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/globalconfig/types"
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
// activeOnly if true only shows projects that are currently Docker containers
// continuous if true keeps requesting and outputting continuously
// wrapTableText if true the text is wrapped instead of truncated to fit the row length
// continuousSleepTime is the time between reports
func List(settings ListCommandSettings) {
	defer util.TimeTrack()()

	var out bytes.Buffer

	for {
		apps, err := GetProjects(settings.ActiveOnly)
		if err != nil {
			util.Failed("Failed getting GetProjects: %v", err)
		}

		appDescs := make([]map[string]any, 0)

		if len(apps) < 1 {
			output.UserOut.WithField("raw", appDescs).Println("No DDEV projects were found.")
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
			routerURL := globalconfig.GetRouterURL()
			if routerStatus == SiteStopped {
				routerURL = ""
			}
			location := output.Hyperlink(output.FileURL(globalconfig.GetGlobalDdevDirLocation()), fileutil.ShortHomeJoin(globalconfig.GetGlobalDdevDirLocation()))
			extendedRouterStatus, errorInfo := RenderRouterStatus()
			if errorInfo != "" {
				location = text.WrapSoft(errorInfo, 35)
			}
			routerURL = output.Hyperlink(routerURL, routerURL)
			routerType := globalconfig.DdevGlobalConfig.Router
			if len(types.GetValidRouterTypes()) < 2 {
				routerType = ""
			}
			t.AppendFooter(table.Row{
				"Router", extendedRouterStatus, location, routerURL, routerType},
			)
			t.Render()
			output.UserOut.WithField("raw", appDescs).Print(out.String())
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
	if termWidth == 0 {
		termWidth = 80
	}

	// Table border/padding overhead for 5 columns (borders + 1-space padding each side)
	const tableOverhead = 17
	usableWidth := termWidth - tableOverhead

	// Fixed column widths: status wraps "running\n(ok)" naturally at 7
	statusWidth := 7
	typeWidth := 9 // "wordpress" is the longest common type
	nameWidth := 10

	// Distribute remaining width: location 40%, URL 60%
	remaining := max(usableWidth-nameWidth-statusWidth-typeWidth, 25)
	locationWidth := remaining * 2 / 5
	urlWidth := remaining - locationWidth

	util.Debug("termWidth=%v usableWidth=%d statusWidth=%d nameWidth=%d locationWidth=%d urlWidth=%d typeWidth=%d", termWidth, usableWidth, statusWidth, nameWidth, locationWidth, urlWidth, typeWidth)
	t.SortBy([]table.SortBy{{Name: "Name"}})

	if !globalconfig.DdevGlobalConfig.SimpleFormatting {
		snip := func(col string, maxLen int) string { return text.Snip(col, maxLen, "…") }
		locationConfig := table.ColumnConfig{Name: "Location", WidthMax: locationWidth, WidthMaxEnforcer: snip}
		urlConfig := table.ColumnConfig{Name: "URL", WidthMax: urlWidth, WidthMaxEnforcer: snip}
		if wrapTableText {
			// In wrap mode, show full paths and URLs on one unbroken line so they are selectable
			locationConfig = table.ColumnConfig{Name: "Location"}
			urlConfig = table.ColumnConfig{Name: "URL"}
		}
		t.SetColumnConfigs([]table.ColumnConfig{
			{Name: "Name", WidthMax: nameWidth},
			{Name: "Status", WidthMax: statusWidth},
			locationConfig,
			urlConfig,
			{Name: "Type", WidthMax: typeWidth, WidthMaxEnforcer: text.WrapText},
		})
	}
	if !wrapTableText {
		t.SetAllowedRowLength(termWidth)
	}
	styles.SetGlobalTableStyle(t, false)
	t.SetOutputMirror(out)
	return t
}
