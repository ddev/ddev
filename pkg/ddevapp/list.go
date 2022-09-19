package ddevapp

import (
	"bytes"
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/drud/ddev/pkg/nodeps"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/styles"
	"github.com/drud/ddev/pkg/util"
	"github.com/drud/ddev/pkg/versionconstants"
	"github.com/jedib0t/go-pretty/v6/table"
	"time"
)

// List provides the functionality for `ddev list`
// activeOnly if true only shows projects that are currently docker containers
// continuous if true keeps requesting and outputting continuously
// wrapTableText if true the text is wrapped instead of truncated to fit the row length
// continuousSleepTime is the time between reports
func List(activeOnly bool, continuous bool, wrapTableText bool, continuousSleepTime int) {
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
			t := CreateAppTable(&out, wrapTableText)
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
				"Router", "", routerStatus, versionconstants.GetRouterImage()},
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
				Name:     "Type",
				WidthMax: int(typeWidth),
			},
		})
	}
	styles.SetGlobalTableStyle(t)
	t.SetOutputMirror(out)
	return t
}
