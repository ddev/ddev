package dockerutil

import "github.com/ddev/ddev/pkg/versionconstants"

// GlobalDdevLabels returns labels applied to all DDEV-managed Docker resources.
// Resources tied to a specific project should add project-specific labels on top
// via ddevapp.GetDdevLabels.
func GlobalDdevLabels() map[string]string {
	return map[string]string{
		// Marks the resource as DDEV-managed; used by poweroff to find and remove
		// all DDEV networks (FindNetworksWithLabel) and by GetActiveProjects to
		// filter web containers.
		"com.ddev.platform": "ddev",
		// Project name; empty for global resources. Used by poweroff to remove
		// straggling containers (FindContainersByLabels) and by router to list
		// containers belonging to a site.
		"com.ddev.site-name": "",
		// Stamps the DDEV web image tag on the resource. Not actively used yet,
		// but intended to allow pruning stale resources or triggering rebuilds
		// when switching to a different DDEV version.
		"com.ddev.webtag": versionconstants.WebTag,
	}
}
