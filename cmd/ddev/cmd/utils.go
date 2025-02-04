package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"os"
	"regexp"
	"strings"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/globalconfig"
)

// getRequestedProjects will collect and return the requested projects from command line arguments and flags.
func getRequestedProjects(names []string, all bool) ([]*ddevapp.DdevApp, error) {
	return getRequestedProjectsExtended(names, all, false)
}

// getRequestedProjectsExtended will collect and return the requested projects from command line arguments and flags.
// If withNonExisting is true, it will return project stubs even if they don't exist.
func getRequestedProjectsExtended(names []string, all bool, withNonExisting bool) ([]*ddevapp.DdevApp, error) {
	requestedProjects := make([]*ddevapp.DdevApp, 0)

	// If no project is specified, return the current project
	if len(names) == 0 && !all {
		project, err := ddevapp.GetActiveApp("")
		if err != nil {
			return nil, err
		}

		return append(requestedProjects, project), nil
	}

	allProjects, err := ddevapp.GetProjects(false)
	if err != nil {
		return nil, err
	}

	// If all projects are requested, return here
	if all {
		for _, project := range allProjects {
			err = project.ValidateConfig()
			if err != nil {
				return nil, err
			}
		}

		return allProjects, nil
	}

	// Convert all projects slice into map indexed by project name to prevent duplication
	allProjectMap := map[string]*ddevapp.DdevApp{}
	for _, project := range allProjects {
		allProjectMap[project.Name] = project
	}

	// Select requested projects
	requestedProjectsMap := map[string]*ddevapp.DdevApp{}
	for _, name := range names {
		var exists bool
		// If the requested project name is found in the Docker map, OK
		// If not, if we find it in the global project list, (if it has approot)
		// Otherwise, error.
		if requestedProjectsMap[name], exists = allProjectMap[name]; !exists {
			p := globalconfig.GetProject(name)
			if p != nil && p.AppRoot != "" {
				requestedProjectsMap[name] = &ddevapp.DdevApp{Name: name, AppRoot: p.AppRoot}
			} else if withNonExisting {
				requestedProjectsMap[name] = &ddevapp.DdevApp{Name: name}
			} else {
				return nil, fmt.Errorf("could not find requested project '%s', you may need to use \"ddev start\" to add it to the project catalog", name)
			}
		}
	}

	// Convert map back to slice
	for _, project := range requestedProjectsMap {
		err = project.ValidateConfig()
		if err != nil {
			return nil, err
		}

		requestedProjects = append(requestedProjects, project)
	}

	return requestedProjects, nil
}

// GetUnknownFlags returns a map of unknown flags (short and long) passed to cmd and their values.
// If there is no value passed to the flag, it will be an empty string.
// Works only with this config in Cobra: FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true}
// If Cobra implements handling of unknown flags, this function can be removed/refactored.
// See https://github.com/spf13/cobra/issues/739
func GetUnknownFlags(cmd *cobra.Command) (map[string]string, error) {
	unknownFlags := make(map[string]string)
	if len(os.Args) < 1 {
		return unknownFlags, nil
	}

	// Known flags tracking
	knownFlags := make(map[string]bool)
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		knownFlags["--"+f.Name] = true
		if f.Shorthand != "" {
			knownFlags["-"+f.Shorthand] = true
		}
	})
	// Match only:
	// 1. short lowercase flags, such as `-a`
	// 2. short lowercase flags with a value, such as `-a=anything`
	// 3. long lowercase flags, such as `--flag-123-name`
	// 4. long lowercase flags with a value, such as `--flag-123-name=anything`
	flagRegex := regexp.MustCompile(`^-[a-z]$|^-[a-z]=.*|^--[a-z][a-z0-9-]*$|^--[a-z][a-z0-9-]*=.*`)

	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		arg := args[i]
		// Skip if the value is a known flag
		if knownFlags[arg] {
			continue
		}

		if !flagRegex.MatchString(arg) {
			// Fail if the value is not a valid flag
			if strings.HasPrefix(arg, "-") {
				return unknownFlags, fmt.Errorf("the flag must consist of lowercase letters, numbers, and hyphens, and it must start with a letter, but received %s", arg)
			}
			// Skip if the value is not a flag, but an argument
			continue
		}

		// Handle `--flag=value` or `-f=value` case
		if strings.Contains(arg, "=") {
			parts := strings.SplitN(arg, "=", 2)
			unknownFlags[parts[0]] = parts[1]
		} else if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
			// Handle `--flag value` or `-f value` case
			unknownFlags[arg] = args[i+1]
			i++ // Skip the value
		} else {
			// Handle flags without value
			unknownFlags[arg] = ""
		}
	}
	return unknownFlags, nil
}
