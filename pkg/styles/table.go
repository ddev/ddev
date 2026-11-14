package styles

import (
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/output"
)

// cellPadding is the space on each side of a cell's content.
const cellPadding = 1

// Table is a table rendered with lipgloss in the look of the configured
// table_style. Callers add cells, which may hold color and OSC 8 hyperlinks,
// optionally limit column widths, and render. With simple formatting it is
// plain text without borders.
type Table struct {
	headers   []string
	rows      [][]string
	footer    []string
	title     []string
	maxWidths []int
	snip      []bool
	colStyles []lipgloss.Style
	unlimited bool
}

// NewTable returns an empty table with headers.
func NewTable(headers ...string) *Table {
	return &Table{
		headers:   headers,
		maxWidths: make([]int, len(headers)),
		snip:      make([]bool, len(headers)),
		colStyles: make([]lipgloss.Style, len(headers)),
	}
}

// profile is what stdout supports: NoTTY for plain text, ASCII for text
// attributes and symbols without color, ANSI or more for color. It follows
// the TERM, NO_COLOR, CLICOLOR, and CLICOLOR_FORCE conventions.
var profile = sync.OnceValue(func() colorprofile.Profile {
	p := colorprofile.Detect(os.Stdout, os.Environ())
	// colorprofile reads NO_COLOR only as a boolean and ignores CLICOLOR=0
	clicolor, err := strconv.ParseBool(os.Getenv("CLICOLOR"))
	if !output.ColorsEnabled() || (err == nil && !clicolor) {
		p = min(p, colorprofile.ASCII)
	}
	return p
})

// StylingEnabled reports whether output gets text attributes such as bold,
// and status symbols: stdout is a terminal and simple formatting is off.
func StylingEnabled() bool {
	return !SimpleFormattingRequired() && profile() >= colorprofile.ASCII
}

// ColorEnabled reports whether output gets colors: styling is on, and
// neither NO_COLOR nor CLICOLOR=0 is set.
func ColorEnabled() bool {
	return !SimpleFormattingRequired() && profile() >= colorprofile.ANSI
}

func color(c ansi.Color) ansi.Color {
	if !ColorEnabled() {
		return lipgloss.NoColor{}
	}
	return c
}

// Text styles shared by the tables. Colors are the terminal's own ANSI
// colors, as in ddev tui, so they follow its theme. Gray, the theme's bright
// black, is unreadable in some themes, so it marks only borders and status
// symbols, never text that must be read. Blue is too dark on many dark
// backgrounds, so headers are bold and links underlined, without color.
var (
	Link    = func() lipgloss.Style { return attr(lipgloss.NewStyle().Underline(true)) }
	Muted   = func() lipgloss.Style { return lipgloss.NewStyle().Foreground(color(lipgloss.BrightBlack)) }
	Good    = func() lipgloss.Style { return lipgloss.NewStyle().Foreground(color(lipgloss.Green)) }
	Warning = func() lipgloss.Style { return lipgloss.NewStyle().Foreground(color(lipgloss.Yellow)) }
	Bad     = func() lipgloss.Style { return lipgloss.NewStyle().Foreground(color(lipgloss.Red)) }
	Strong  = func() lipgloss.Style { return attr(lipgloss.NewStyle().Bold(true)) }
	Note    = func() lipgloss.Style { return attr(lipgloss.NewStyle().Italic(true)) }
	// Hint is secondary text. Faint dims the theme's own text color, which
	// stays readable where gray is not, and terminals without it show
	// normal text.
	Hint = func() lipgloss.Style { return attr(lipgloss.NewStyle().Faint(true)) }
)

// attr drops text attributes such as bold when styling is off.
func attr(s lipgloss.Style) lipgloss.Style {
	if !StylingEnabled() {
		return lipgloss.NewStyle()
	}
	return s
}

// AppendRow adds a row. Missing trailing cells are empty.
func (t *Table) AppendRow(cells ...string) {
	t.rows = append(t.rows, t.pad(cells))
}

// SetFooter sets a last row, separated from the others by a line.
func (t *Table) SetFooter(cells ...string) {
	t.footer = t.pad(cells)
}

// SetTitle sets lines shown above the table.
func (t *Table) SetTitle(lines ...string) {
	t.title = lines
}

// SortBy sorts the rows by the visible text of column col.
func (t *Table) SortBy(col int) {
	slices.SortStableFunc(t.rows, func(a, b []string) int {
		return strings.Compare(ansi.Strip(a[col]), ansi.Strip(b[col]))
	})
}

// SetMaxWidths limits the content width of each column, 0 for no limit.
// Text in a column wider than its limit wraps.
func (t *Table) SetMaxWidths(widths ...int) {
	copy(t.maxWidths, widths)
}

// SetSnip makes a one-line cell, or a line without spaces, in column col
// that is wider than the column's limit end in "…" instead of wrapping.
// Its hyperlink keeps the full target.
func (t *Table) SetSnip(col int) {
	t.snip[col] = true
}

// SetColumnStyle sets the text style of a column's cells.
func (t *Table) SetColumnStyle(col int, style lipgloss.Style) {
	t.colStyles[col] = style
}

// SetUnlimited ignores the width limits, so nothing is wrapped or snipped.
func (t *Table) SetUnlimited(unlimited bool) {
	t.unlimited = unlimited
}

// NaturalWidths returns the widest line of each column, header and footer
// included.
func (t *Table) NaturalWidths() []int {
	widths := make([]int, len(t.headers))
	for _, row := range t.allRows() {
		for i, cell := range row {
			for line := range strings.SplitSeq(cell, "\n") {
				widths[i] = max(widths[i], ansi.StringWidth(line))
			}
		}
	}
	return widths
}

// FitTerminal limits the columns' widths so the table fits the terminal,
// unless it is set to be unlimited.
func (t *Table) FitTerminal() {
	termWidth, _ := nodeps.GetTerminalWidthHeight(os.Stdout)
	t.SetMaxWidths(FitColumnWidths(t.NaturalWidths(), termWidth-t.Overhead())...)
}

// Overhead returns the width that borders and padding add to the columns'
// content in the configured style.
func (t *Table) Overhead() int {
	n := len(t.headers)
	overhead := 2 * cellPadding * n
	look := t.look(nil, nil)
	if look.borderColumn {
		overhead += n - 1
	}
	if look.borderSides {
		overhead += 2
	}
	return overhead
}

// Render returns the table as text ending in a newline.
func (t *Table) Render() string {
	widths := t.columnWidths()
	body := slices.Clone(t.rows)
	for i := range body {
		body[i] = t.fit(body[i], widths)
	}
	look := t.look(body, widths)
	headers := make([]string, len(t.headers))
	for i, h := range t.headers {
		headers[i] = strings.ToUpper(h)
	}

	var b strings.Builder
	for _, line := range t.title {
		b.WriteString(line + "\n")
	}
	main := t.newLipglossTable(look, widths).Headers(headers...).Rows(body...)
	if t.footer == nil {
		b.WriteString(main.Render() + "\n")
		return b.String()
	}

	// lipgloss draws no line before only the last row, so the footer is a
	// second table of the same column widths, joined by that line.
	main.BorderBottom(false)
	b.WriteString(main.Render() + "\n")
	if look.borderHeader {
		b.WriteString(t.separator(look, widths) + "\n")
	}
	footer := t.newLipglossTable(look, widths).BorderTop(false).Rows(t.fit(t.footer, widths))
	b.WriteString(footer.Render() + "\n")
	return b.String()
}

// tableLook is how the configured table_style draws a table.
type tableLook struct {
	border       lipgloss.Border
	borderSides  bool // top, bottom, left, and right
	borderColumn bool
	borderHeader bool
	borderRow    bool
	header       lipgloss.Style
}

// look returns the look of the configured table_style for rows in columns
// of widths. "default" has a rounded outer border and space between
// columns, "bold" heavy borders everywhere, and "bright" no borders and a
// reverse-video header, which reads well in any theme. Rows are separated
// by a line, or by a blank line without borders, when some row takes more
// than one line. Simple formatting has no borders or styling.
func (t *Table) look(rows [][]string, widths []int) tableLook {
	if SimpleFormattingRequired() {
		return tableLook{border: lipgloss.HiddenBorder()}
	}
	header := Strong()
	multiline := hasMultilineRow(rows, widths)
	switch globalconfig.GetTableStyle() {
	case "bold":
		return tableLook{border: lipgloss.ThickBorder(), borderSides: true, borderColumn: true, borderHeader: true, header: header}
	case "bright":
		return tableLook{border: lipgloss.HiddenBorder(), borderRow: multiline, header: header.Reverse(StylingEnabled())}
	}
	return tableLook{border: lipgloss.RoundedBorder(), borderSides: true, borderHeader: true, borderRow: multiline, header: header}
}

// hasMultilineRow reports whether any of rows takes more than one line in
// columns of widths, which is when lines between rows help tell them apart.
func hasMultilineRow(rows [][]string, widths []int) bool {
	for _, row := range rows {
		for i, cell := range row {
			if strings.Contains(cell, "\n") || ansi.StringWidth(cell) > widths[i] {
				return true
			}
		}
	}
	return false
}

// columnWidths returns each column's content width: its natural width,
// held to its limit.
func (t *Table) columnWidths() []int {
	widths := t.NaturalWidths()
	if t.unlimited {
		return widths
	}
	for i, limit := range t.maxWidths {
		if limit > 0 {
			widths[i] = min(widths[i], limit)
		}
	}
	return widths
}

// hyperlink matches the start of an OSC 8 hyperlink with a target.
var hyperlink = regexp.MustCompile("\x1b]8;[^;\x07\x1b]*;[^\x07\x1b]")

// fit snips lines wider than their column that read badly broken: a whole
// cell, or a line without spaces such as a path, in a snip column, and any
// line holding a hyperlink. lipgloss wraps the rest, such as an error
// message, keeping colors and hyperlinks intact across the lines.
func (t *Table) fit(row []string, widths []int) []string {
	fitted := slices.Clone(row)
	for i, cell := range fitted {
		lines := strings.Split(cell, "\n")
		for j, line := range lines {
			if ansi.StringWidth(line) > widths[i] && ((t.snip[i] && (len(lines) == 1 || !strings.Contains(ansi.Strip(line), " "))) || hyperlink.MatchString(line)) {
				lines[j] = ansi.Truncate(line, widths[i], "…")
			}
		}
		fitted[i] = strings.Join(lines, "\n")
	}
	return fitted
}

// newLipglossTable returns a lipgloss table with look and fixed column
// widths, so that the main table and its footer line up.
func (t *Table) newLipglossTable(look tableLook, widths []int) *table.Table {
	borderStyle := Muted()
	total := 0
	for _, w := range widths {
		total += w + 2*cellPadding
	}
	if look.borderColumn {
		total += len(widths) - 1
	}
	if look.borderSides {
		total += 2
	}
	return table.New().
		Border(look.border).
		BorderStyle(borderStyle).
		BorderTop(look.borderSides).
		BorderBottom(look.borderSides).
		BorderLeft(look.borderSides).
		BorderRight(look.borderSides).
		BorderColumn(look.borderColumn).
		BorderHeader(look.borderHeader).
		BorderRow(look.borderRow).
		Wrap(true).
		Width(total).
		StyleFunc(func(row, col int) lipgloss.Style {
			s := t.colStyles[col]
			if row == table.HeaderRow {
				s = look.header
			}
			return s.Padding(0, cellPadding).Width(widths[col] + 2*cellPadding)
		})
}

// separator returns the line between the table and its footer.
func (t *Table) separator(look tableLook, widths []int) string {
	var b strings.Builder
	if look.borderSides {
		b.WriteString(look.border.MiddleLeft)
	}
	for i, w := range widths {
		b.WriteString(strings.Repeat(look.border.Bottom, w+2*cellPadding))
		if i < len(widths)-1 && look.borderColumn {
			b.WriteString(look.border.Middle)
		}
	}
	if look.borderSides {
		b.WriteString(look.border.MiddleRight)
	}
	return Muted().Render(b.String())
}

func (t *Table) allRows() [][]string {
	all := append([][]string{t.headers}, t.rows...)
	if t.footer != nil {
		all = append(all, t.footer)
	}
	return all
}

// pad returns cells with as many entries as there are columns.
func (t *Table) pad(cells []string) []string {
	row := make([]string, len(t.headers))
	copy(row, cells)
	return row
}

// FitColumnWidths returns the natural widths when they fit in available.
// Otherwise the widest columns are capped at one shared width, lowered until
// the row fits or it reaches minWidth, so narrow columns keep their full
// width and the space goes where it's needed.
func FitColumnWidths(natural []int, available int) []int {
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
