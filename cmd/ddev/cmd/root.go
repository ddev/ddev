package cmd

import (
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/updatecheck"
	"github.com/drud/ddev/pkg/util"
	"github.com/drud/ddev/pkg/version"
	"github.com/getsentry/raven-go"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	updateInterval = time.Hour * 24 * 7 // One week interval between updates
	serviceType    string
	updateDocURL   = "https://ddev.readthedocs.io/en/stable/#installation"
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:     "ddev",
	Short:   "DDEV-Local local development environment",
	Long:    "Create and maintain a local web development environment.",
	Version: version.DdevVersion,
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
		ignores := map[string]bool{"list": true, "version": true, "help": true, "auth-pantheon": true, "hostname": true}
		if _, ok := ignores[cmd.CalledAs()]; ok {
			return
		}
		sentryNotSetupWarning()

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
		uString := "Usage:"
		for i := len(fullCommand) - 1; i >= 0; i = i - 1 {
			uString = uString + " " + fullCommand[i]
		}

		if globalconfig.DdevGlobalConfig.InstrumentationOptIn && version.SentryDSN != "" {
			_ = raven.CaptureMessageAndWait(uString, map[string]string{"severity-level": "info", "report-type": "usage"})
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

	// Prevent running as root for most cases
	// We really don't want ~/.ddev to have root ownership, breaks things.
	if os.Geteuid() == 0 && len(os.Args) > 1 && os.Args[1] != "hostname" {
		output.UserOut.Fatal("ddev is not designed to be run with root privileges, please run as normal user and without sudo")
	}

	err := globalconfig.ReadGlobalConfig()
	if err != nil {
		util.Failed("Failed to read global config file %s: %v", globalconfig.GetGlobalConfigPath(), err)
	}
}

func sentryNotSetupWarning() {
	if version.SentryDSN == "" && globalconfig.DdevGlobalConfig.InstrumentationOptIn {
		output.UserOut.Warning("Instrumentation is opted in, but SentryDSN is not available.")
	}
}

// checkDdevVersionAndOptInSentry() reads global config and checks to see if current version is different
// from the last saved version. If it is, prompt to request anon ddev usage stats
// and update the info.
func checkDdevVersionAndOptInSentry() error {
	if !output.JSONOutput && version.COMMIT != globalconfig.DdevGlobalConfig.LastUsedVersion && globalconfig.DdevGlobalConfig.InstrumentationOptIn == false && !globalconfig.DdevNoSentry {
		allowStats := util.Confirm("It looks like you have a new ddev release.\nMay we send anonymous ddev usage statistics and errors?\nTo know what we will see please take a look at\nhttps://ddev.readthedocs.io/en/latest/users/cli-usage/#opt-in-usage-information\nPermission to beam up?")
		if allowStats {
			globalconfig.DdevGlobalConfig.InstrumentationOptIn = true
		}
		globalconfig.DdevGlobalConfig.LastUsedVersion = version.COMMIT
		err := globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig)
		if err != nil {
			return err
		}
	}
	return nil
}
