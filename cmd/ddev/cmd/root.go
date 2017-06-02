package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/drud/ddev/pkg/plugins/platform"
	"github.com/drud/ddev/pkg/updatecheck"
	"github.com/drud/ddev/pkg/util"
	"github.com/drud/ddev/pkg/version"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	logLevel = log.WarnLevel
	plugin   = "local"
	// 1 week
	updateInterval = time.Hour * 24 * 7
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "ddev",
	Short: "A CLI for interacting with ddev.",
	Long:  "This Command Line Interface (CLI) gives you the ability to interact with the ddev to create a local development environment.",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		ignores := []string{"list", "version", "describe", "config"}
		skip := false
		command := strings.Join(os.Args, " ")

		for _, k := range ignores {
			if strings.Contains(command, " "+k) {
				skip = true
				break
			}
		}

		if !skip {
			_, err := platform.GetPluginApp(plugin)
			if err != nil {
				util.Failed("Plugin %s is not registered", plugin)
			}
		}

		usr, err := homedir.Dir()
		if err != nil {
			log.Fatal("Could not detect user's home directory: %v", err)
		}

		updateFile := filepath.Join(usr, ".ddev", ".update")
		// Do periodic detection of whether an update is available for ddev users.
		timeToCheckForUpdates, err := updatecheck.IsUpdateNeeded(updateFile, updateInterval)
		if err != nil {
			util.Warning("Could not perform update check: %v", err)
		}

		if timeToCheckForUpdates {
			updateNeeded, updateURL, err := updatecheck.AvailableUpdates("drud", "ddev", version.DdevVersion)

			if err != nil {
				util.Warning("Could not check for updates. this is most often caused by a networking issue.")
				log.Debug(err)
				return
			}

			if updateNeeded {
				util.Warning("\n\nA new update is available! please visit %s to download the update!\n\n", updateURL)
				err = updatecheck.ResetUpdateTime(updateFile)
				if err != nil {
					util.Warning("Could not reset automated update checking interval: %v", err)
				}
			}
		}

		err = util.CheckDockerVersion(version.DockerVersionConstraint)
		if err != nil {
			util.Failed("The docker version currently installed does not meet ddev's requirements: %v", err)
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
	drudDebug := os.Getenv("DRUD_DEBUG")
	if drudDebug != "" {
		logLevel = log.DebugLevel
	}

	log.SetLevel(logLevel)
}

// getActiveAppRoot returns the fully rooted directory of the active app, or an error
func getActiveAppRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("error determining the current directory: %s", err)
	}

	appRoot, err := platform.CheckForConf(cwd)
	if err != nil {
		return "", fmt.Errorf("unable to determine the application for this command. Have you run 'ddev config'? Error: %s", err)
	}

	return appRoot, nil
}

// getActiveApp returns the active platform.App based on the current working directory.
func getActiveApp() (platform.App, error) {
	app, err := platform.GetPluginApp(plugin)
	if err != nil {
		return app, err
	}
	activeAppRoot, err := getActiveAppRoot()
	if err != nil {
		return app, err
	}

	err = app.Init(activeAppRoot)
	return app, err
}
