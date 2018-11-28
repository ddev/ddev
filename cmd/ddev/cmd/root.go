package cmd

import (
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/getsentry/raven-go"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/updatecheck"
	"github.com/drud/ddev/pkg/util"
	"github.com/drud/ddev/pkg/version"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	updateInterval = time.Hour * 24 * 7 // One week interval between updates
	serviceType    string
	updateDocURL   = "https://ddev.readthedocs.io/en/stable/#installation"
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "ddev",
	Short: "A CLI for interacting with ddev.",
	Long:  "This Command Line Interface (CLI) gives you the ability to interact with the ddev to create a development environment.",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		ignores := []string{"version", "config", "hostname", "help", "auth-pantheon", "import-files"}
		command := strings.Join(os.Args[1:], " ")

		output.LogSetUp()

		// Skip docker validation for any command listed in "ignores"
		for _, k := range ignores {
			if strings.Contains(command, k) {
				return
			}
		}

		err := dockerutil.CheckDockerVersion(version.DockerVersionConstraint)
		if err != nil {
			if err.Error() == "no docker" {
				if os.Args[1] != "version" && os.Args[1] != "config" {
					util.Failed("Could not connect to docker. Please ensure Docker is installed and running.")
				}
			} else {
				util.Failed("The docker version currently installed does not meet ddev's requirements: %v", err)
			}
		}

		err = dockerutil.CheckDockerCompose(version.DockerComposeVersionConstraint)
		if err != nil {
			if err.Error() == "no docker-compose" {
				util.Failed("docker-compose does not appear to be installed.")
			} else {
				util.Failed("The docker-compose version currently installed does not meet ddev's requirements: %v", err)
			}
		}

		// Look for version change
		err = checkVersionAndOptIn()
		if err != nil {
			util.Failed(err.Error())
		}

		updateFile := filepath.Join(globalconfig.GetGlobalDdevDir(), ".update")

		// Do periodic detection of whether an update is available for ddev users.
		timeToCheckForUpdates, err := updatecheck.IsUpdateNeeded(updateFile, updateInterval)
		if err != nil {
			util.Warning("Could not perform update check: %v", err)
		}

		if timeToCheckForUpdates {
			// Recreate the updatefile with current time so we won't do this again soon.
			err = updatecheck.ResetUpdateTime(updateFile)
			if err != nil {
				util.Warning("Failed to update updatecheck file %s", updateFile)
				return // Do not continue as we'll end up with github api violations.
			}

			updateNeeded, updateURL, err := updatecheck.AvailableUpdates("drud", "ddev", version.DdevVersion)

			if err != nil {
				util.Warning("Could not check for updates. This is most often caused by a networking issue.")
				log.Debug(err)
				return
			}

			if updateNeeded {
				util.Warning("\n\nA new update is available! please visit %s to download the update.\nFor upgrade help see %s", updateURL, updateDocURL)
			}
		}

	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		// Do not report these comamnds
		ignores := map[string]bool{"list": true, "version": true, "help": true, "auth-pantheon": true}
		if _, ok := ignores[cmd.CalledAs()]; ok {
			return
		}
		if globalconfig.DdevGlobalConfig.InstrumentationOptIn && version.SentryDSN != "" {
			_ = raven.CaptureMessageAndWait("ddev "+cmd.CalledAs(), map[string]string{"level": "info"})
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

	// Ensure that the ~/.ddev exists
	_ = globalconfig.GetGlobalDdevDir()
	err := globalconfig.ReadGlobalConfig()
	if err != nil {
		util.Failed(err.Error())
	}
	setupSentry()
}

func setupSentry() {
	if version.SentryDSN == "" && globalconfig.DdevGlobalConfig.InstrumentationOptIn {
		output.UserOut.Warning("Instrumentation is opted in, but SentryDSN is not available.")
	}
}

// checkVersionAndOptIn() reads global config and checks to see if current version is different
// from the last saved version. If it is, prompt to request anon ddev usage stats
// and update the info.
func checkVersionAndOptIn() error {
	if version.COMMIT != globalconfig.DdevGlobalConfig.LastRunVersion {
		allowStats := util.Confirm("It looks like you have a new ddev release.\nMay we send anonymous ddev usage statistics and errors?")
		if allowStats {
			globalconfig.DdevGlobalConfig.InstrumentationOptIn = true
		}
		globalconfig.DdevGlobalConfig.LastRunVersion = version.COMMIT
		err := globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig)
		if err != nil {
			return err
		}
	}
	return nil
}
