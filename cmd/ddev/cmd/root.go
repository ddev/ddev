package cmd

import (
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
	updateDocURL   = "https://ddev.readthedocs.io/en/latest/#installation"
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "ddev",
	Short: "A CLI for interacting with ddev.",
	Long:  "This Command Line Interface (CLI) gives you the ability to interact with the ddev to create a development environment.",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		ignores := []string{"list", "version", "describe", "config", "hostname"}
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

		// Verify that the ~/.ddev exists
		userDdevDir := util.GetGlobalDdevDir()

		updateFile := filepath.Join(userDdevDir, ".update")

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

			// nolint: vetshadow
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
}
