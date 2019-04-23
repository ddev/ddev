package cmd

import (
	"fmt"
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/globalconfig"
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

	allDockerProjects := ddevapp.GetDockerProjects()

	// If all projects are requested, return here
	if all {
		return allDockerProjects, nil
	}

	// Convert all projects slice into map indexed by project name to prevent duplication
	allDockerProjectMap := map[string]*ddevapp.DdevApp{}
	for _, project := range allDockerProjects {
		allDockerProjectMap[project.Name] = project
	}

	// Select requested projects
	requestedProjectsMap := map[string]*ddevapp.DdevApp{}
	for _, name := range names {
		var exists bool
		// If the requested project name is found in the docker map, OK
		// If not, if we find it in the globl project list, OK
		// Otherwise, error.
		if requestedProjectsMap[name], exists = allDockerProjectMap[name]; !exists {
			if _, exists = globalconfig.DdevGlobalConfig.ProjectList[name]; exists {
				requestedProjectsMap[name] = &ddevapp.DdevApp{Name: name}
			} else {
				return nil, fmt.Errorf("could not find requested project %s", name)
			}
		}

	}

	// Convert map back to slice
	for _, project := range requestedProjectsMap {
		requestedProjects = append(requestedProjects, project)
	}

	return requestedProjects, nil
}
