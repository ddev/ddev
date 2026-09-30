package ddevapp

import (
	"bytes"
	"fmt"
	"os"
	"slices"
	"strings"
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

	for {
		var out bytes.Buffer
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
				// Without column limits, nothing else wraps the error
				if settings.WrapTableText || globalconfig.DdevGlobalConfig.SimpleFormatting {
					errorInfo = text.WrapSoft(errorInfo, 35)
				}
				location += "\n" + errorInfo
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

// FitTableCell wraps each line of a table cell to maxLen. A line holding an
// OSC 8 hyperlink is snipped instead, because wrapping corrupts the link, and
// its target stays clickable.
func FitTableCell(col string, maxLen int) string {
	wrapped := make([]string, 0, strings.Count(col, "\n")+1)
	for line := range strings.SplitSeq(col, "\n") {
		switch {
		case text.StringWidthWithoutEscSequences(line) <= maxLen:
			wrapped = append(wrapped, line)
		case strings.Contains(line, "\x1b]8;"):
			wrapped = append(wrapped, text.Snip(line, maxLen, "…"))
		default:
			wrapped = append(wrapped, strings.Split(text.WrapSoft(line, maxLen), "\n")...)
		}
	}
	return strings.Join(wrapped, "\n")
}

// appTable measures its cells as they are added and sizes its columns from
// them when it renders, so the table fits the terminal at that moment
type appTable struct {
	table.Writer
	natural       []int
	wrapTableText bool
}

// CreateAppTable will create a new app table for list output
func CreateAppTable(out *bytes.Buffer, wrapTableText bool) table.Writer {
	header := table.Row{"Name", "Status", "Location", "URL", "Type"}
	t := &appTable{Writer: table.NewWriter(), natural: make([]int, len(header)), wrapTableText: wrapTableText}
	t.measure(header)
	t.AppendHeader(header)
	t.SortBy([]table.SortBy{{Name: "Name"}})
	styles.SetGlobalTableStyle(t, false)
	t.SetOutputMirror(out)
	return t
}

// AppendRow adds a row and measures it
func (t *appTable) AppendRow(row table.Row, configs ...table.RowConfig) {
	t.measure(row)
	t.Writer.AppendRow(row, configs...)
}

// AppendRows adds rows and measures them
func (t *appTable) AppendRows(rows []table.Row, configs ...table.RowConfig) {
	for _, row := range rows {
		t.measure(row)
	}
	t.Writer.AppendRows(rows, configs...)
}

// AppendFooter adds a footer row and measures it
func (t *appTable) AppendFooter(row table.Row, configs ...table.RowConfig) {
	t.measure(row)
	t.Writer.AppendFooter(row, configs...)
}

// measure records the widest line of each cell in row
func (t *appTable) measure(row table.Row) {
	for i, cell := range row {
		for line := range strings.SplitSeq(fmt.Sprint(cell), "\n") {
			t.natural[i] = max(t.natural[i], text.StringWidthWithoutEscSequences(line))
		}
	}
}

// Render sizes the columns for the current terminal width, then renders
func (t *appTable) Render() string {
	termWidth, _ := nodeps.GetTerminalWidthHeight(os.Stdout)
	if !globalconfig.DdevGlobalConfig.SimpleFormatting {
		// Table border/padding overhead for 5 columns (borders + 1-space padding each side)
		const tableOverhead = 17
		const nameMaxWidth = 20
		// A long name wraps rather than taking space a path or URL could use
		natural := slices.Clone(t.natural)
		natural[0] = min(natural[0], nameMaxWidth, termWidth/8)
		widths := fitColumnWidths(natural, termWidth-tableOverhead)
		util.Debug("termWidth=%d natural=%v widths=%v", termWidth, t.natural, widths)

		// A path or URL is snipped to one line, but multi-line text, such as
		// a router error, wraps so none of it is lost
		fit := func(col string, maxLen int) string {
			if !strings.Contains(col, "\n") {
				return text.Snip(col, maxLen, "…")
			}
			return FitTableCell(col, maxLen)
		}
		locationConfig := table.ColumnConfig{Name: "Location", WidthMax: widths[2], WidthMaxEnforcer: fit}
		urlConfig := table.ColumnConfig{Name: "URL", WidthMax: widths[3], WidthMaxEnforcer: fit}
		if t.wrapTableText {
			// In wrap mode, show full paths and URLs on one unbroken line so they are selectable
			locationConfig = table.ColumnConfig{Name: "Location"}
			urlConfig = table.ColumnConfig{Name: "URL"}
		}
		t.SetColumnConfigs([]table.ColumnConfig{
			{Name: "Name", WidthMax: widths[0]},
			{Name: "Status", WidthMax: widths[1]},
			locationConfig,
			urlConfig,
			{Name: "Type", WidthMax: widths[4], WidthMaxEnforcer: text.WrapText},
		})
	}
	if !t.wrapTableText {
		// Backstop for a terminal too narrow for even the minimum widths
		t.SetAllowedRowLength(termWidth)
	}
	return t.Writer.Render()
}

// fitColumnWidths returns the natural widths when they fit in available.
// Otherwise the widest columns are capped at one shared width, lowered until
// the row fits or it reaches minWidth, so narrow columns keep their full
// width and the space goes where it's needed.
func fitColumnWidths(natural []int, available int) []int {
	// Wide enough for "running" and "stopped"
	const minWidth = 7
	widths := make([]int, len(natural))
	for limit := slices.Max(natural); limit >= minWidth; limit-- {
		total := 0
		for i, n := range natural {
			widths[i] = min(n, limit)
			total += widths[i]
		}
		if total <= available {
			break
		}
	}
	return widths
}
