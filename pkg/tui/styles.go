package tui

import (
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/muesli/termenv"
)

// Styles holds all the lipgloss styles used by the TUI.
type Styles struct {
	Title       lipgloss.Style
	StatusBar   lipgloss.Style
	HelpKey     lipgloss.Style
	HelpDesc    lipgloss.Style
	Running     lipgloss.Style
	Stopped     lipgloss.Style
	Paused      lipgloss.Style
	ProjectName lipgloss.Style
	ProjectType lipgloss.Style
	URL         lipgloss.Style
	Cursor      lipgloss.Style
	Divider     lipgloss.Style
	HelpOverlay lipgloss.Style
	DetailLabel lipgloss.Style
	DetailValue lipgloss.Style
}

// NewStyles creates styles respecting SimpleFormatting and NO_COLOR.
func NewStyles() Styles {
	noColor := globalconfig.DdevGlobalConfig.SimpleFormatting ||
		os.Getenv("NO_COLOR") != "" ||
		termenv.ColorProfile() == termenv.Ascii

	if noColor {
		return Styles{
			Title:       lipgloss.NewStyle().Bold(true),
			StatusBar:   lipgloss.NewStyle(),
			HelpKey:     lipgloss.NewStyle().Bold(true),
			HelpDesc:    lipgloss.NewStyle(),
			Running:     lipgloss.NewStyle(),
			Stopped:     lipgloss.NewStyle(),
			Paused:      lipgloss.NewStyle(),
			ProjectName: lipgloss.NewStyle().Bold(true),
			ProjectType: lipgloss.NewStyle(),
			URL:         lipgloss.NewStyle(),
			Cursor:      lipgloss.NewStyle().Bold(true),
			Divider:     lipgloss.NewStyle(),
			HelpOverlay: lipgloss.NewStyle().Padding(1, 2),
			DetailLabel: lipgloss.NewStyle().Bold(true).Width(14),
			DetailValue: lipgloss.NewStyle(),
		}
	}

	green := lipgloss.Color("2")
	red := lipgloss.Color("1")
	yellow := lipgloss.Color("3")
	blue := lipgloss.Color("4")

	return Styles{
		Title:       lipgloss.NewStyle().Bold(true).Foreground(blue),
		StatusBar:   lipgloss.NewStyle().Faint(true),
		HelpKey:     lipgloss.NewStyle().Bold(true).Foreground(blue),
		HelpDesc:    lipgloss.NewStyle().Faint(true),
		Running:     lipgloss.NewStyle().Foreground(green),
		Stopped:     lipgloss.NewStyle().Foreground(red),
		Paused:      lipgloss.NewStyle().Foreground(yellow),
		ProjectName: lipgloss.NewStyle().Bold(true),
		ProjectType: lipgloss.NewStyle().Faint(true),
		URL:         lipgloss.NewStyle().Foreground(blue),
		Cursor:      lipgloss.NewStyle().Bold(true).Foreground(blue),
		Divider:     lipgloss.NewStyle().Faint(true),
		HelpOverlay: lipgloss.NewStyle().Padding(1, 2).Border(lipgloss.RoundedBorder()).BorderForeground(blue),
		DetailLabel: lipgloss.NewStyle().Bold(true).Faint(true).Width(14),
		DetailValue: lipgloss.NewStyle(),
	}
}
