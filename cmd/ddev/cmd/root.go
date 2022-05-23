package cmd

import (
	"fmt"
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/updatecheck"
	"github.com/drud/ddev/pkg/util"
	"github.com/drud/ddev/pkg/versionconstants"
	"github.com/rogpeppe/go-internal/semver"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/segmentio/analytics-go.v3"
	"os"
	"path/filepath"

	"time"
)

var (
	updateInterval     = time.Hour * 24 * 7 // One week interval between updates
	serviceType        string
	updateDocURL       = "https://ddev.readthedocs.io/en/stable/#installation"
	instrumentationApp *ddevapp.DdevApp
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "ddev",
	Short: "DDEV-Local local development environment",
	Long: `Create and maintain a local web development environment.
Docs: https://ddev.readthedocs.io
Support: https://ddev.readthedocs.io/en/stable/#support`,
	Version: versionconstants.DdevVersion,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		command := os.Args[1]

		// LogSetup() has already been done, but now needs to be done
		// again *after* --json flag is parsed.
		output.LogSetUp()

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

			updateNeeded, updateVersion, updateURL, err := updatecheck.AvailableUpdates("drud", "ddev", versionconstants.DdevVersion)

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
		// Do not report these commands
		ignores := map[string]bool{"auth": true, "exec": true, "help": true, "hostname": true, "list": true, "ssh": true, "version": true}
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
			runTime := util.TimeTrack(time.Now(), "Instrumentation")
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
			runTime()
		}
	},
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	// bind flags to viper config values...allows override by flag
	viper.AutomaticEnv() // read in environment variables that match

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
	if err != nil && len(os.Args) > 1 && os.Args[1] != "--version" {
		util.Failed("Could not connect to a docker provider. Please start or install a docker provider.\nFor installation help go to: https://ddev.readthedocs.io/en/latest/users/docker_installation/")
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
		allowStats := util.Confirm("It looks like you have a new ddev release.\nMay we send anonymous ddev usage statistics and errors?\nTo know what we will see please take a look at\nhttps://ddev.readthedocs.io/en/stable/users/cli-usage/#opt-in-usage-information\nPermission to beam up?")
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
		globalconfig.DdevGlobalConfig.LastStartedVersion = versionconstants.DdevVersion
		err := globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig)
		if err != nil {
			return err
		}

		okPoweroff := util.Confirm("It looks like you have a new DDEV version. During an upgrade it's important to `ddev poweroff`. May I do `ddev poweroff` before continuing? This does no harm and loses no data.")
		if okPoweroff {
			ddevapp.PowerOff()
		}
		return nil
	}

	return nil
}
