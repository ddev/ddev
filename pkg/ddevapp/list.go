package ddevapp

import (
	"bytes"
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/drud/ddev/pkg/nodeps"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/styles"
	"github.com/drud/ddev/pkg/util"
	"github.com/jedib0t/go-pretty/v6/table"
	"time"
)

// List provides the functionality for `ddev list`
// activeOnly if true only shows projects that are currently docker containers
// continuous if true keeps requesting and outputting continuously
// continuousSleepTime is the time between reports
func List(activeOnly bool, continuous bool, continuousSleepTime int) {
	runTime := util.TimeTrack(time.Now(), "ddev list")
	defer runTime()

	var out bytes.Buffer

	for {
		apps, err := GetProjects(activeOnly)
		if err != nil {
			util.Failed("failed getting GetProjects: %v", err)
		}
		appDescs := make([]map[string]interface{}, 0)

		if len(apps) < 1 {
			output.UserOut.WithField("raw", appDescs).Println("No ddev projects were found.")
		} else {
			t := CreateAppTable(&out)
			for _, app := range apps {
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
			t.AppendFooter(table.Row{
				"Router", "", routerStatus},
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

		if !continuous {
			break
		}

		time.Sleep(time.Duration(continuousSleepTime) * time.Second)
	}
}

// CreateAppTable will create a new app table for describe and list output
func CreateAppTable(out *bytes.Buffer) table.Writer {
	t := table.NewWriter()
	t.AppendHeader(table.Row{"Name", "Type", "Location", "URL", "Status"})
	termWidth, _ := nodeps.GetTerminalWidthHeight()
	usableWidth := termWidth - 15
	statusWidth := 7 // Maybe just "running"
	nameWidth := 10
	typeWidth := 7 // drupal7
	locationWidth := 20
	urlWidth := 20
	if termWidth > 80 {
		urlWidth = urlWidth + (termWidth-80)/2
		locationWidth = locationWidth + (termWidth-80)/2
		statusWidth = statusWidth + (termWidth-80)/3
	}
	totUsedWidth := nameWidth + typeWidth + locationWidth + urlWidth + statusWidth

	util.Debug("detected terminal width=%v usableWidth=%d statusWidth=%d nameWidth=%d typeWIdth=%d locationWidth=%d urlWidth=%d totUsedWidth=%d", termWidth, usableWidth, statusWidth, nameWidth, typeWidth, locationWidth, urlWidth, totUsedWidth)
	t.SortBy([]table.SortBy{{Name: "Name"}})

	if !globalconfig.DdevGlobalConfig.SimpleFormatting {

		t.SetColumnConfigs([]table.ColumnConfig{
			{
				Name: "Name",
				//WidthMax: nameWidth,
			},
			{
				Name:     "Type",
				WidthMax: int(typeWidth),
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
				Name:     "Status",
				WidthMax: statusWidth,
			},
		})
	}
	styles.SetGlobalTableStyle(t)
	t.SetOutputMirror(out)
	return t
}
