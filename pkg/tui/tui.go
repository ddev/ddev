package tui

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/ddev/ddev/pkg/ddevapp"
)

// Launch starts the interactive TUI dashboard.
// AltScreen is set on the View so the TUI doesn't pollute scroll-back.
func Launch() error {
	baseEnv = os.Environ()
	// Hook output would print over the screen; the subcommands the TUI runs,
	// such as start, still run their hooks.
	ddevapp.SkipHooks = true
	defer captureLogs()()
	p := tea.NewProgram(NewAppModel())
	_, err := p.Run()
	if err != nil {
		return fmt.Errorf("TUI error: %v", err)
	}
	return nil
}
