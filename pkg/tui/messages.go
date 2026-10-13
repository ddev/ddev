package tui

import "os"

// projectsListedMsg carries the project list as read from configuration,
// before the status of the projects is known.
type projectsListedMsg struct {
	projects []ProjectInfo
}

// projectsLoadedMsg carries the project list.
type projectsLoadedMsg struct {
	projects []ProjectInfo
	router   health
	sshAgent health
	err      error
	// warning is the last line ddevapp logged while loading.
	warning string
}

// projectDetailLoadedMsg carries the detail of the project at appRoot.
type projectDetailLoadedMsg struct {
	appRoot string
	detail  ProjectDetail
	err     error
	warning string
}

// xdebugStatusMsg carries whether Xdebug is enabled in the project at appRoot.
type xdebugStatusMsg struct {
	appRoot string
	enabled bool
	err     error
	toggled bool
}

// latestAddonsMsg carries the latest release of each add-on, by repository.
type latestAddonsMsg struct {
	appRoot  string
	versions map[string]string
}

// execFinishedMsg is sent when a command that took over the terminal exits.
type execFinishedMsg struct {
	title string
	// done replaces "<title>: done" in the status line.
	done string
	err  error
	// selectCurrent selects the working directory's project once it is listed.
	selectCurrent bool
}

// streamStartedMsg is sent when the subprocess of stream id has started.
type streamStartedMsg struct {
	id      int
	lines   <-chan string
	errCh   <-chan error
	process *os.Process
}

// streamLinesMsg carries the lines of output stream id wrote since the
// last one.
type streamLinesMsg struct {
	id    int
	lines []string
}

// streamEndedMsg is sent when the subprocess of stream id exits.
type streamEndedMsg struct {
	id  int
	err error
}

// statusExpiredMsg clears status message id, unless a newer one replaced it.
type statusExpiredMsg struct {
	id int
}

// tickMsg triggers the periodic refresh.
type tickMsg struct{}

// clipboardMsg is sent after a clipboard copy attempt. fallback holds the
// text when no clipboard tool could take it.
type clipboardMsg struct {
	fallback string
}

// openedMsg is sent after trying to open a directory or editor.
type openedMsg struct {
	what string
	err  error
}
