package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// Launch starts the interactive TUI dashboard.
// It uses the alternate screen buffer so the TUI doesn't pollute scroll-back.
func Launch() error {
	model := NewAppModel()
	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err := p.Run()
	if err != nil {
		return fmt.Errorf("TUI error: %v", err)
	}
	return nil
}
