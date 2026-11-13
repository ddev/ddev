package ddevapp

import (
	"os"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/globalconfig/types"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/styles"
	"github.com/ddev/ddev/pkg/util"
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

	for {
		releaseContainers := dockerutil.CacheContainers()
		apps, err := GetProjects(settings.ActiveOnly)
		if err != nil {
			util.Failed("Failed getting GetProjects: %v", err)
		}

		appDescs := make([]map[string]any, 0)

		if len(apps) < 1 {
			output.UserOut.WithField("raw", appDescs).Println("No DDEV projects were found.")
		} else {
			t := CreateAppTable(settings.WrapTableText)
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
				// Without column limits, nothing else wraps the error
				if settings.WrapTableText || globalconfig.DdevGlobalConfig.SimpleFormatting {
					errorInfo = ansi.Wordwrap(errorInfo, 35, "")
				}
				location += "\n" + errorInfo
			}
			routerURL = output.Hyperlink(routerURL, routerURL)
			routerType := globalconfig.DdevGlobalConfig.Router
			if len(types.GetValidRouterTypes()) < 2 {
				routerType = ""
			}
			t.SetFooter("Router", extendedRouterStatus, location, routerURL, routerType)
			output.UserOut.WithField("raw", appDescs).Print(RenderAppTable(t))
		}
		releaseContainers()

		if !settings.Continuous {
			break
		}

		time.Sleep(time.Duration(settings.ContinuousSleepTime) * time.Second)
	}
}

// CreateAppTable returns a table for list output. With wrapTableText,
// nothing is wrapped or snipped to fit the terminal.
func CreateAppTable(wrapTableText bool) *styles.Table {
	t := styles.NewTable("Name", "Status", "Location", "URL", "Type")
	t.SetColumnStyle(0, styles.Strong())
	t.SetColumnStyle(3, styles.Link())
	// A path or URL is snipped to one line, but multi-line text, such as
	// a router error, wraps so none of it is lost
	t.SetSnip(2)
	t.SetSnip(3)
	t.SetUnlimited(wrapTableText || globalconfig.DdevGlobalConfig.SimpleFormatting)
	return t
}

// RenderAppTable sizes the columns of t for the current terminal width,
// then renders it
func RenderAppTable(t *styles.Table) string {
	termWidth, _ := nodeps.GetTerminalWidthHeight(os.Stdout)
	const nameMaxWidth = 20
	// A long name wraps rather than taking space a path or URL could use
	natural := t.NaturalWidths()
	natural[0] = min(natural[0], nameMaxWidth, termWidth/8)
	widths := styles.FitColumnWidths(natural, termWidth-t.Overhead())
	util.Debug("termWidth=%d natural=%v widths=%v", termWidth, natural, widths)
	t.SetMaxWidths(widths...)
	t.SortBy(0)
	return t.Render()
}
