package cmd

import (
	"fmt"

	"strings"

	"github.com/drud/ddev/pkg/ddevapp"
)

// getRequestedProjects will collect and return the requested projects from command line arguments and flags.
func getRequestedProjects(names []string, all bool) ([]*ddevapp.DdevApp, error) {
	requestedProjects := make([]*ddevapp.DdevApp, 0)

	// If no project is specified, return the current project
	if len(names) == 0 && !all {
		project, err := ddevapp.GetActiveApp("")
		if err != nil {
			return nil, err
		}

		return append(requestedProjects, project), nil
	}

	allProjects := ddevapp.GetApps()

	// If all projects are requested, return here
	if all {
		return allProjects, nil
	}

	// Convert all projects slice into map indexed by project name to prevent duplication
	allProjectsMap := map[string]*ddevapp.DdevApp{}
	for _, project := range allProjects {
		allProjectsMap[project.Name] = project
	}

	// Select requested projects
	requestedProjectsMap := map[string]*ddevapp.DdevApp{}
	for _, name := range names {
		var exists bool
		if requestedProjectsMap[name], exists = allProjectsMap[name]; !exists {
			return nil, fmt.Errorf("could not find project %s", name)
		}
	}

	// Convert map back to slice
	for _, project := range requestedProjectsMap {
		requestedProjects = append(requestedProjects, project)
	}

	return requestedProjects, nil
}

// checkForMissingProjectFiles returns an error if the project's configuration or project root cannot be found
func checkForMissingProjectFiles(project *ddevapp.DdevApp) error {
	if strings.Contains(project.SiteStatus(), ddevapp.SiteConfigMissing) || strings.Contains(project.SiteStatus(), ddevapp.SiteDirMissing) {
		return fmt.Errorf("ddev can no longer find your project files at %s. If you would like to continue using ddev to manage this project please restore your files to that directory. If you would like to remove this site from ddev, you may run 'ddev remove %s'", project.GetAppRoot(), project.GetName())
	}

	return nil
}
