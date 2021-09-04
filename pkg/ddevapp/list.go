package ddevapp

import (
	"bytes"
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/drud/ddev/pkg/nodeps"
	"github.com/drud/ddev/pkg/output"
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
			tWidth, _ := nodeps.GetTerminalWidthHeight()
			t.SetAllowedRowLength(tWidth)
			util.Debug("detected terminal width=%v", tWidth)
			t.SortBy([]table.SortBy{{Name: "Name"}})
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
