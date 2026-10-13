package tui

import (
	"image/color"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/textinput"
	"charm.land/lipgloss/v2"
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/output"
)

// Styles holds the lipgloss styles used by the TUI. Colors are the terminal's
// own ANSI colors, or derived from its text and background, so they follow
// its theme; Bubble Tea drops them where color is off (NO_COLOR).
type Styles struct {
	Title    lipgloss.Style
	Muted    lipgloss.Style
	Selected lipgloss.Style
	Name     lipgloss.Style
	Match    lipgloss.Style
	URL      lipgloss.Style
	Label    lipgloss.Style
	Section  lipgloss.Style
	Running  lipgloss.Style
	Paused   lipgloss.Style
	Error    lipgloss.Style
	Prompt   lipgloss.Style
	// Info is every message that is not an error or a question. It has no
	// color, which is kept for what needs attention.
	Info     lipgloss.Style
	HelpKey  lipgloss.Style
	HelpDesc lipgloss.Style
}

// NewStyles returns the TUI styles for the terminal's text and background
// colors, nil while unknown, without colors when simple formatting is
// configured or colors are disabled.
func NewStyles(fg, bg color.Color) Styles {
	gray := mutedColor(fg, bg)
	var green, red, yellow, blue color.Color = lipgloss.Green, lipgloss.Red, lipgloss.Yellow, lipgloss.Blue
	if globalconfig.DdevGlobalConfig.SimpleFormatting || !output.ColorsEnabled() {
		green, red, yellow, blue, gray = lipgloss.NoColor{}, lipgloss.NoColor{}, lipgloss.NoColor{}, lipgloss.NoColor{}, lipgloss.NoColor{}
	}
	return Styles{
		Title:    lipgloss.NewStyle().Bold(true).Foreground(blue),
		Muted:    lipgloss.NewStyle().Foreground(gray),
		Selected: lipgloss.NewStyle().Bold(true).Foreground(blue),
		Name:     lipgloss.NewStyle().Bold(true),
		Match:    lipgloss.NewStyle().Underline(true),
		URL:      lipgloss.NewStyle().Foreground(blue),
		Label:    lipgloss.NewStyle().Foreground(gray),
		Section:  lipgloss.NewStyle().Bold(true),
		Running:  lipgloss.NewStyle().Foreground(green),
		Paused:   lipgloss.NewStyle().Foreground(yellow),
		Error:    lipgloss.NewStyle().Foreground(red),
		Prompt:   lipgloss.NewStyle().Bold(true).Foreground(yellow),
		Info:     lipgloss.NewStyle(),
		HelpKey:  lipgloss.NewStyle().Bold(true).Foreground(blue),
		HelpDesc: lipgloss.NewStyle().Foreground(gray),
	}
}

// mutedColor is the color of secondary text. It is a color rather than the
// faint attribute, which some terminals ignore. Between the terminal's own
// text and background colors, it is readable in any theme; the theme's
// bright black, used while they are unknown, is too dark in many.
func mutedColor(fg, bg color.Color) color.Color {
	if fg == nil || bg == nil {
		return lipgloss.BrightBlack
	}
	return lipgloss.Blend1D(11, bg, fg)[6]
}

// statusSymbol is as wide as the symbol and space before every status.
const statusSymbol = "● "

// status renders a project, service or container health status with a
// symbol and color, so the states differ by shape as well as color. Stopped
// is normal, so it is gray rather than red.
func (s Styles) status(status string) string {
	if status == "" {
		return ""
	}
	symbol, style := s.statusLook(status)
	return style.Render(symbol + " " + statusLabel(status))
}

// statusLook is the symbol and style status is shown with.
func (s Styles) statusLook(status string) (string, lipgloss.Style) {
	switch status {
	case statusLoading:
		return "◌", s.Muted
	case ddevapp.SiteRunning, "healthy":
		return "●", s.Running
	case ddevapp.SitePaused, ddevapp.SiteStarting:
		return "◐", s.Paused
	case ddevapp.SiteStopped:
		return "○", s.Muted
	case containerDisabled:
		return "⊘", s.Muted
	}
	return "✗", s.Error
}

// statusLabel shortens the statuses too long for the status column, which
// is as wide as unhealthy.
func statusLabel(status string) string {
	switch status {
	case ddevapp.SiteDirMissing:
		return "missing"
	case ddevapp.SiteConfigMissing:
		return "no config"
	}
	return status
}

// help returns styles for the help bubble.
func (s Styles) help() help.Styles {
	return help.Styles{
		ShortKey:       s.HelpKey,
		ShortDesc:      s.HelpDesc,
		ShortSeparator: s.Muted,
		Ellipsis:       s.Muted,
		FullKey:        s.HelpKey,
		FullDesc:       s.HelpDesc,
		FullSeparator:  s.Muted,
	}
}

// filter returns styles for the project list's filter input.
func (s Styles) filter() textinput.Styles {
	filter := textinput.DefaultStyles(true)
	filter.Focused.Prompt = s.Prompt
	filter.Blurred.Prompt = s.Prompt
	filter.Focused.Text = lipgloss.NewStyle()
	filter.Cursor.Color = nil
	return filter
}
