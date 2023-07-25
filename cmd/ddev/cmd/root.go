package cmd

import (
	"fmt"
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
	"gopkg.in/segmentio/analytics-go.v3"
)

var (
	updateInterval     = time.Hour * 24 * 7 // One week interval between updates
	serviceType        string
	updateDocURL       = "https://ddev.readthedocs.io/en/stable/users/install/"
	instrumentationApp *ddevapp.DdevApp
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "ddev",
	Short: "DDEV local development environment",
	Long: `Create and maintain a local web development environment.
Docs: https://ddev.readthedocs.io
Support: https://ddev.readthedocs.io/en/stable/users/support`,
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
			cmdCopy.Use = "custom-command"
			argsCopy = []string{}
		}

		// We don't want to send to amplitude if using --json-output
		// That captures an enormous number of PhpStorm running the
		// ddev describe -j over and over again.
		if !output.JSONOutput {
			amplitude.TrackCommand(&cmdCopy, argsCopy)
		}

		// Skip docker and other validation for most commands
		if command != "start" && command != "restart" {
			return
		}

		err := dockerutil.CheckDockerVersion(dockerutil.DockerVersionConstraint)
		if err != nil {
			if err.Error() == "no docker" {
				if os.Args[1] != "version" {
					util.Failed("Could not connect to docker. Please ensure Docker is installed and running.")
				}
			} else {
				util.Warning("The docker version currently installed does not seem to meet ddev's requirements: %v", err)
			}
		}

		updateFile := filepath.Join(globalconfig.GetGlobalDdevDir(), ".update")

		// Do periodic detection of whether an update is available for ddev users.
		timeToCheckForUpdates, err := updatecheck.IsUpdateNeeded(updateFile, updateInterval)
		if err != nil {
			util.Warning("Could not perform update check: %v", err)
		}

		if timeToCheckForUpdates && globalconfig.IsInternetActive() {
			// Recreate the updatefile with current time so we won't do this again soon.
			err = updatecheck.ResetUpdateTime(updateFile)
			if err != nil {
				util.Warning("Failed to update updatecheck file %s", updateFile)
				return // Do not continue as we'll end up with github api violations.
			}

			updateNeeded, updateVersion, updateURL, err := updatecheck.AvailableUpdates("ddev", "ddev", versionconstants.DdevVersion)

			if err != nil {
				util.Warning("Could not check for updates. This is most often caused by a networking issue.")
				return
			}

			if updateNeeded {
				output.UserOut.Printf(util.ColorizeText(fmt.Sprintf("\n\nUpgraded DDEV %s is available!\nPlease visit %s to get the upgrade.\nFor upgrade help see %s\n\n", updateVersion, updateURL, updateDocURL), "green"))
			}
		}
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		// TODO remove once it's activated directly in ddevapp
		if instrumentationApp == nil {
			app, err := ddevapp.NewApp("", false)
			if err == nil {
				instrumentationApp = app
			}
		}

		// We don't need to track when used with --json-output
		// picks up enormous number of automated ddev describe
		if instrumentationApp != nil && !output.JSONOutput {
			instrumentationApp.TrackProject()
		}

		// TODO: remove when we decide to stop reporting to Segment.
		// All code to "end TODO remove once Amplitude" will be removed
		// Do not report these commands
		ignores := map[string]bool{"describe": true, "auth": true, "blackfire": false, "clean": true, "composer": true, "debug": true, "delete": true, "drush": true, "exec": true, "export-db": true, "get": true, "help": true, "hostname": true, "import-db": true, "import-files": true, "list": true, "logs": true, "mutagen": true, "mysql": true, "npm": true, "nvm": true, "php": true, "poweroff": true, "pull": true, "push": true, "service": true, "share": true, "snapshot": true, "ssh": true, "stop": true, "version": true, "xdebug": true, "xhprof": true, "yarn": true}

		if _, ok := ignores[cmd.CalledAs()]; ok {
			return
		}
		instrumentationNotSetUpWarning()

		// All this nonsense is to capture the official usage we used for this command.
		// Unfortunately cobra doesn't seem to provide this easily.
		// We use the first word of Use: to get it.
		cmdCopy := cmd
		var fullCommand = make([]string, 0)
		fullCommand = append(fullCommand, util.GetFirstWord(cmdCopy.Use))
		for cmdCopy.HasParent() {
			fullCommand = append(fullCommand, util.GetFirstWord(cmdCopy.Parent().Use))
			cmdCopy = cmdCopy.Parent()
		}
		for i := 0; i < len(fullCommand)/2; i++ {
			j := len(fullCommand) - i - 1
			fullCommand[i], fullCommand[j] = fullCommand[j], fullCommand[i]
		}

		event := ""
		if len(fullCommand) > 1 {
			event = fullCommand[1]
		}

		if globalconfig.DdevGlobalConfig.InstrumentationOptIn && versionconstants.SegmentKey != "" && globalconfig.IsInternetActive() && len(fullCommand) > 1 {
			defer util.TimeTrackC("Instrumentation")()
			// Try to get default instrumentationApp from current directory if not already set
			if instrumentationApp == nil {
				app, err := ddevapp.NewApp("", false)
				if err == nil {
					instrumentationApp = app
				}
			}
			// If it has been set, provide the tags, otherwise no app tags
			if instrumentationApp != nil {
				instrumentationApp.SetInstrumentationAppTags()
			}
			ddevapp.SetInstrumentationBaseTags()
			ddevapp.SendInstrumentationEvents(event)
		}
		// end TODO remove once Amplitude has verified with an alpha release.
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

	output.LogSetUp()

	// Determine if Docker is running by getting the version.
	// This helps to prevent a user from seeing the Cobra error: "Error: unknown command "<custom command>" for ddev"
	_, err := dockerutil.GetDockerVersion()
	// ddev --version may be called without docker available.
	if err != nil && len(os.Args) > 1 && os.Args[1] != "--version" && os.Args[1] != "hostname" {
		util.Failed("Could not connect to a docker provider. Please start or install a docker provider.\nFor install help go to: https://ddev.readthedocs.io/en/latest/users/install/")
	}

	// Populate custom/script commands so they're visible
	// We really don't want ~/.ddev or .ddev/homeadditions or .ddev/.globalcommands to have root ownership, breaks things.
	if os.Geteuid() == 0 {
		util.Warning("Not populating custom commands or hostadditions because running with root privileges")
	} else {
		err := ddevapp.PopulateExamplesCommandsHomeadditions("")
		if err != nil {
			util.Warning("populateExamplesAndCommands() failed: %v", err)
		}

		err = addCustomCommands(RootCmd)
		if err != nil {
			util.Warning("Adding custom/shell commands failed: %v", err)
		}
	}
}

func instrumentationNotSetUpWarning() {
	if !output.JSONOutput && versionconstants.SegmentKey == "" && globalconfig.DdevGlobalConfig.InstrumentationOptIn {
		util.Warning("Instrumentation is opted in, but SegmentKey is not available. This usually means you have a locally-built ddev binary or one from a PR build. It's not an error. Please report it if you're using an official release build.")
	}
}

// checkDdevVersionAndOptInInstrumentation() reads global config and checks to see if current version is different
// from the last saved version. If it is, prompt to request anon ddev usage stats
// and update the info.
func checkDdevVersionAndOptInInstrumentation(skipConfirmation bool) error {
	if !output.JSONOutput && semver.Compare(versionconstants.DdevVersion, globalconfig.DdevGlobalConfig.LastStartedVersion) > 0 && globalconfig.DdevGlobalConfig.InstrumentationOptIn == false && !globalconfig.DdevNoInstrumentation && !skipConfirmation {
		allowStats := util.Confirm("It looks like you have a new ddev release.\nMay we send anonymous ddev usage statistics and errors?\nTo know what we will see please take a look at\nhttps://ddev.readthedocs.io/en/latest/users/usage/diagnostics/#opt-in-usage-information\nPermission to beam up?")
		if allowStats {
			globalconfig.DdevGlobalConfig.InstrumentationOptIn = true
			client, _ := analytics.NewWithConfig(versionconstants.SegmentKey, analytics.Config{
				Logger: &ddevapp.SegmentNoopLogger{},
			})
			defer func() {
				_ = client.Close()
			}()

			err := ddevapp.SegmentUser(client, ddevapp.GetInstrumentationUser())
			if err != nil {
				output.UserOut.Debugf("error in SegmentUser: %v", err)
			}
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
			util.Debug("Terminating all mutagen sync sessions")
			ddevapp.TerminateAllMutagenSync()
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
