package tui

import "github.com/ddev/ddev/pkg/ddevapp"

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

// logsLoadedMsg is sent when container logs have been fetched.
type logsLoadedMsg struct {
	logs    string
	service string
	err     error
}

// routerStatusMsg carries the router health status.
type routerStatusMsg struct {
	status string
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
