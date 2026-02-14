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
