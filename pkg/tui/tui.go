package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
)

// Launch starts the interactive TUI dashboard.
// AltScreen is set on the View so the TUI doesn't pollute scroll-back.
func Launch() error {
	p := tea.NewProgram(NewAppModel())
	_, err := p.Run()
	if err != nil {
		return fmt.Errorf("TUI error: %v", err)
	}
	return nil
}
