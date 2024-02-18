package cmd

import (
	"os"
	"strings"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/output"
	"github.com/spf13/pflag"
)

// This file is a.go because some code needs to be processed before any other
// files run their init() to avoid conflicts and ensure certain functionality
// is available.

func init() {
	// Set up logger
	output.LogSetUp()

	// Set up global config
	globalconfig.EnsureGlobalConfig()
	_ = os.Setenv("DOCKER_CLI_HINTS", "false")
	_ = os.Setenv("MUTAGEN_DATA_DIRECTORY", globalconfig.GetMutagenDataDirectory())

	// GetDockerClient should be called early to get DOCKER_HOST set
	_, _ = dockerutil.GetDockerClient()

	// Parse, and use the "--project" flag
	handleProjectFlag()
}

// handleProjectFlag parses the "--project" flag, and sets the current working
// directory to the named project, if any.
func handleProjectFlag() {
	// We need a new flag set so we can parse this early without affecting flags that belong to other commands.
	tempFlags := pflag.NewFlagSet("", pflag.ContinueOnError)
	// Don't throw errors for unknown flags - we don't know ANY of the flags at this stage except the ones
	// explicitly added below.
	tempFlags.ParseErrorsWhitelist = pflag.ParseErrorsWhitelist{UnknownFlags: true}
	// We need a dummy help flag to avoid pflag.ErrHelp
	tempFlags.BoolP("help", "h", false, "")
	tempFlags.String("project", "", "")

	args := os.Args[1:]

	// skip completion requests where we're trying to complete the project flag itself
	lastArgIndex := len(args) - 1
	if args[0] == "__complete" && (strings.HasPrefix(args[lastArgIndex], "--project=") || args[lastArgIndex-1] == "--project") {
		return
	}

	// Parse the flag
	err := tempFlags.Parse(args)
	if err != nil {
		output.UserErr.Fatal("Couldn't parse --project flag: ", err)
	}
	project, err := tempFlags.GetString("project")
	if err != nil {
		// This shouldn't ever happen - but go wants us to handle the error.
		output.UserErr.Fatal("Unexpected error getting value from --project flag: ", err)
	}

	if project != "" {
		// Find and change to the root dir for the project
		approot, err := ddevapp.GetActiveAppRoot(project)
		if err != nil {
			output.UserErr.Fatal("Couldn't find project: ", err)
		}
		err = os.Chdir(approot)
		if err != nil {
			output.UserErr.Fatal("Couldn't change working directory to "+approot+": ", err)
		}

		// Remove project flag from os.Args so it doesn't interfere with anything.
		// Completion gets a little stuffed up by it otherwise, and there could be other side-effects.
		for i, arg := range os.Args {
			if arg == "--project" || strings.HasPrefix(arg, "--project=") {
				split := strings.SplitN(arg, "=", 2)
				n := 2
				if len(split) == 2 {
					n = 1
				}
				os.Args = append(os.Args[:i], os.Args[i+n:]...)
				break
			}
		}
	}
}
