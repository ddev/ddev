package cmd

import (
	"os"
	"path/filepath"
	"time"

	"github.com/ddev/ddev/pkg/amplitude"
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/updatecheck"
	"github.com/ddev/ddev/pkg/util"
	"github.com/ddev/ddev/pkg/versionconstants"
	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"
)

var (
	updateInterval     = time.Hour * 4 // Four-hour interval between updates
	serviceType        string
	updateDocURL       = "https://ddev.readthedocs.io/en/stable/users/install/ddev-upgrade/"
	instrumentationApp *ddevapp.DdevApp
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "ddev",
	Short: "DDEV local development environment",
	Long: `Create and maintain a local web development environment.
Docs: https://ddev.readthedocs.io
Support: https://ddev.readthedocs.io/en/stable/users/support/`,
	Version: versionconstants.DdevVersion,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		command := os.Args[1]

		// LogSetup() has already been done, but now needs to be done
		// again *after* --json flag is parsed.
		output.LogSetUp()

		// Anonymize user defined custom commands.
		cmdCopy := *cmd
		argsCopy := args
		if IsUserDefinedCustomCommand(&cmdCopy) {
			cmdCopy = cobra.Command{Use: "custom-command"}
			argsCopy = []string{}
		}

		// We don't want to send to amplitude if using --json-output
		// That captures an enormous number of PhpStorm running the
		// ddev describe -j over and over again.
		// And we don't want to send __complete commands,
		// that are called each time you press <TAB>.
		if !output.JSONOutput && cmdCopy.Name() != cobra.ShellCompRequestCmd {
			amplitude.TrackCommand(&cmdCopy, argsCopy)
		}

		// Skip Docker and other validation for most commands
		if command != "start" && command != "restart" {
			return
		}

		err := dockerutil.CheckDockerVersion(dockerutil.DockerVersionConstraint)
		if err != nil {
			if err.Error() == "no docker" {
				if os.Args[1] != "version" {
					util.Failed("Could not connect to Docker. Please ensure Docker is installed and running.")
				}
			} else {
				util.Warning("The Docker version currently installed does not seem to meet DDEV's requirements: %v", err)
			}
		}

		updateFile := filepath.Join(globalconfig.GetGlobalDdevDir(), ".update")

		// Do periodic detection of whether an update is available for DDEV users.
		timeToCheckForUpdates, err := updatecheck.IsUpdateNeeded(updateFile, updateInterval)
		if err != nil {
			util.Warning("Could not perform update check: %v", err)
		}

		if timeToCheckForUpdates && globalconfig.IsInternetActive() {
			// Recreate the updatefile with current time so we won't do this again soon.
			err = updatecheck.ResetUpdateTime(updateFile)
			if err != nil {
				util.Warning("Failed to update updatecheck file %s", updateFile)
				return // Do not continue as we'll end up with GitHub API violations.
			}

			updateNeeded, updateVersion, updateURL, err := updatecheck.AvailableUpdates("ddev", "ddev", versionconstants.DdevVersion)

			if err != nil {
				util.Warning("Could not check for updates. This is most often caused by a networking issue.")
				return
			}

			if updateNeeded {
				util.Warning("\nUpgraded DDEV %s is available!\nPlease visit %s\nto get the upgrade.\nFor upgrade help see\n%s\n", updateVersion, updateURL, updateDocURL)
			}
		}
	},
	PersistentPostRun: func(_ *cobra.Command, _ []string) {
		if instrumentationApp == nil {
			app, err := ddevapp.GetActiveApp("")
			if err == nil {
				instrumentationApp = app
			}
		}

		// We don't need to track when used with --json-output
		// picks up enormous number of automated ddev describe
		if instrumentationApp != nil && !output.JSONOutput {
			instrumentationApp.TrackProject()
		}
	},
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		os.Exit(-1)
	}
}

func init() {

	RootCmd.PersistentFlags().BoolVarP(&output.JSONOutput, "json-output", "j", false, "If true, user-oriented output will be in JSON format.")
	RootCmd.PersistentFlags().BoolVarP(&ddevapp.SkipHooks, "skip-hooks", "", false, "If true, any hook normally run by the command will be skipped.")

	output.LogSetUp()

	// Determine if Docker is running by getting the version.
	// This helps to prevent a user from seeing the Cobra error: "Error: unknown command "<custom command>" for ddev"
	_, err := dockerutil.GetDockerVersion()
	// ddev --version may be called without Docker available.
	if err != nil && len(os.Args) > 1 && os.Args[1] != "--version" && os.Args[1] != "hostname" {
		util.Failed("Could not connect to a Docker provider. Please start or install a Docker provider.\nFor install help go to: https://ddev.readthedocs.io/en/stable/users/install/docker-installation/")
	}

	// Populate custom/script commands so they're visible.
	// We really don't want ~/.ddev or .ddev/homeadditions to have root ownership, breaks things.
	if os.Geteuid() != 0 {
		err := ddevapp.PopulateExamplesCommandsHomeadditions("")
		if err != nil {
			util.Warning("populateExamplesAndCommands() failed: %v", err)
		}

		err = addCustomCommands(RootCmd)
		if err != nil {
			util.Warning("Adding custom/shell commands failed: %v", err)
		}
	}

	setHelpFunc(RootCmd)
}

// checkDdevVersionAndOptInInstrumentation() reads global config and checks to see if current version is different
// from the last saved version. If it is, prompt to request anon DDEV usage stats
// and update the info.
func checkDdevVersionAndOptInInstrumentation(skipConfirmation bool) error {
	if !output.JSONOutput && semver.Compare(versionconstants.DdevVersion, globalconfig.DdevGlobalConfig.LastStartedVersion) > 0 && globalconfig.DdevGlobalConfig.InstrumentationOptIn == false && !globalconfig.DdevNoInstrumentation && !skipConfirmation {
		allowStats := util.Confirm("It looks like you have a new DDEV release.\nMay we send anonymous DDEV usage statistics and errors?\nTo know what we will see please take a look at\nhttps://ddev.readthedocs.io/en/stable/users/usage/diagnostics/#opt-in-usage-information\nPermission to beam up?")
		if allowStats {
			globalconfig.DdevGlobalConfig.InstrumentationOptIn = true
		}
	}

	if globalconfig.DdevGlobalConfig.LastStartedVersion != versionconstants.DdevVersion && !skipConfirmation {

		// If they have a new version (but not first-timer) then prompt to poweroff
		if globalconfig.DdevGlobalConfig.LastStartedVersion != "v0.0" {
			output.UserOut.Print("You seem to have a new DDEV version.")
			okPoweroff := util.Confirm("During an upgrade it's important to `ddev poweroff`.\nMay I do `ddev poweroff` before continuing?\nThis does no harm and loses no data.")
			if okPoweroff {
				ddevapp.PowerOff()
			}
		}

		// If they have a new version write the new version into last-started
		globalconfig.DdevGlobalConfig.LastStartedVersion = versionconstants.DdevVersion
		err := globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig)
		if err != nil {
			return err
		}

		return nil
	}

	return nil
}
