package cmd

import (
	"os"
	"path"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/drud/ddev/pkg/plugins/platform"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	timestampFormat = "20060102150405"
	updateFile      = ".drud-update"
)

var (
	logLevel  = log.WarnLevel
	plugin    = "local"
	activeApp string
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "ddev",
	Short: "A CLI for interacting with ddev.",
	Long:  "This Command Line Interface (CLI) gives you the ability to interact with the ddev to create a local development environment.",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		ignores := []string{"list", "version"}
		skip := false
		command := strings.Join(os.Args, " ")

		for _, k := range ignores {
			if strings.Contains(command, " "+k) {
				skip = true
				break
			}
		}

		if !skip {
			setActiveApp()
			plugin = strings.ToLower(plugin)
			if _, ok := platform.PluginMap[plugin]; !ok {
				Failed("Plugin %s is not registered", plugin)
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
	drudDebug := os.Getenv("DRUD_DEBUG")
	if drudDebug != "" {
		logLevel = log.DebugLevel
	}

	log.SetLevel(logLevel)
}

func setActiveApp() {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Error determining the current directory: %s", err)
	}

	app, err := platform.CheckForConf(cwd)
	if err != nil {
		log.Fatalf("Unable to determine the application for this command. Have you run 'ddev config'? Error: %s", err)
	}

	activeApp = path.Base(app)
}
