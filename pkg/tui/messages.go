package tui

import (
	"os"

	"github.com/ddev/ddev/pkg/ddevapp"
)

// ProjectInfo holds the display-relevant fields for a DDEV project.
type ProjectInfo struct {
	Name    string
	Status  string
	Type    string
	URL     string
	AppRoot string
}

// projectsLoadedMsg is sent when the project list has been fetched.
type projectsLoadedMsg struct {
	projects []ProjectInfo
	err      error
}

// operationFinishedMsg is sent when a subprocess operation completes.
type operationFinishedMsg struct {
	err error
}

// operationDetailFinishedMsg is sent when an operation finishes in detail view.
type operationDetailFinishedMsg struct {
	err error
}

// ServiceInfo holds the name and status of a container service.
type ServiceInfo struct {
	Name   string
	Status string
}

// ProjectDetail holds the full detail for a DDEV project (equivalent to `ddev describe`).
type ProjectDetail struct {
	Name            string
	Status          string
	Type            string
	PHPVersion      string
	WebserverType   string
	NodeJSVersion   string
	Docroot         string
	DatabaseType    string
	DatabaseVersion string
	XdebugEnabled   bool
	PerformanceMode string
	URLs            []string
	MailpitURL      string
	DBPublishedPort string
	Addons          []string
	Services        []ServiceInfo
	AppRoot         string
}

// projectDetailLoadedMsg is sent when project detail has been fetched.
type projectDetailLoadedMsg struct {
	detail ProjectDetail
	err    error
}

// logStreamStartedMsg is sent when the log streaming subprocess has started.
type logStreamStartedMsg struct {
	lines   <-chan string
	process *os.Process
}

// logLineMsg is sent for each new line of log output.
type logLineMsg struct {
	line string
}

// logStreamEndedMsg is sent when the log stream closes.
type logStreamEndedMsg struct{}

// operationStreamStartedMsg is sent when an operation stream subprocess has started.
type operationStreamStartedMsg struct {
	lines   <-chan string
	errCh   <-chan error
	process *os.Process
}

// operationStreamEndedMsg is sent when the operation stream finishes.
type operationStreamEndedMsg struct {
	err error
}

// operationAutoReturnMsg is sent after a delay to auto-return from a completed operation.
type operationAutoReturnMsg struct{}

// routerStatusMsg carries the router health status.
type routerStatusMsg struct {
	status string
}

// xdebugToggledMsg is sent after xdebug toggle completes.
type xdebugToggledMsg struct {
	err     error
	enabled bool
}

// clipboardMsg is sent after a clipboard copy attempt.
type clipboardMsg struct {
	err error
}

// extractProjectInfo converts a DdevApp to our lightweight ProjectInfo.
func extractProjectInfo(app *ddevapp.DdevApp) ProjectInfo {
	status, _ := app.SiteStatus()
	return ProjectInfo{
		Name:    app.Name,
		Status:  status,
		Type:    app.Type,
		URL:     app.GetPrimaryURL(),
		AppRoot: app.GetAppRoot(),
	}
}
