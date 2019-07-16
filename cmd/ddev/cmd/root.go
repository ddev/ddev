package cmd

import (
	"bufio"
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/drud/ddev/pkg/nodeps"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/updatecheck"
	"github.com/drud/ddev/pkg/util"
	"github.com/drud/ddev/pkg/version"
	"github.com/getsentry/raven-go"
	"github.com/mattn/go-isatty"
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
	Use:   "ddev",
	Short: "DDEV-Local local development environment",
	Long: `Create and maintain a local web development environment.
Docs: https://ddev.readthedocs.io
Support: https://ddev.readthedocs.io/en/stable/#support`,
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

		if timeToCheckForUpdates && nodeps.IsInternetActive() {
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

		uString := strings.Join(fullCommand, " ")
		event := ""
		if len(fullCommand) > 1 {
			event = fullCommand[1]
		}

		instrumentationNotSetUpWarning()
		if globalconfig.DdevGlobalConfig.InstrumentationOptIn && version.SentryDSN != "" && nodeps.IsInternetActive() && len(fullCommand) > 1 {
			_ = raven.CaptureMessageAndWait("Usage: "+uString, map[string]string{"severity-level": "info", "report-type": "usage"})
			ddevapp.SendInstrumentationEvents(event)
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

	err := addCustomCommands(RootCmd)
	if err != nil {
		util.Warning("Adding custom commands failed: %v", err)
	}

	// Prevent running as root for most cases
	// We really don't want ~/.ddev to have root ownership, breaks things.
	if os.Geteuid() == 0 && len(os.Args) > 1 && os.Args[1] != "hostname" {
		output.UserOut.Fatal("ddev is not designed to be run with root privileges, please run as normal user and without sudo")
	}

	err = globalconfig.ReadGlobalConfig()
	if err != nil {
		util.Failed("Failed to read global config file %s: %v", globalconfig.GetGlobalConfigPath(), err)
	}
}

func instrumentationNotSetUpWarning() {
	if version.SentryDSN == "" && globalconfig.DdevGlobalConfig.InstrumentationOptIn {
		output.UserOut.Warning("Instrumentation is opted in, but SentryDSN is not available.")
	}
	if version.SegmentKey == "" && globalconfig.DdevGlobalConfig.InstrumentationOptIn {
		output.UserOut.Warning("Instrumentation is opted in, but SegmentKey is not available.")
	}
}

func addCustomCommands(rootCmd *cobra.Command) error {
	app, err := ddevapp.GetActiveApp("")
	if err != nil {
		return nil
	}

	topCommandPath := app.GetConfigPath("commands")
	if !fileutil.FileExists(topCommandPath) || !fileutil.IsDirectory(topCommandPath) {
		return nil
	}
	commandDirs, _ := fileutil.ListFilesInDir(topCommandPath)
	for _, service := range commandDirs {
		if !fileutil.IsDirectory(filepath.Join(topCommandPath, service)) {
			continue
		}
		commandFiles, err := fileutil.ListFilesInDir(filepath.Join(topCommandPath, service))
		if err != nil {
			return err
		}

		for _, commandName := range commandFiles {
			if strings.HasSuffix(commandName, ".example") {
				continue
			}
			inContainerFullPath := filepath.Join("/mnt/ddev_config/commands", service, commandName)
			onHostFullPath := filepath.Join(topCommandPath, service, commandName)
			description := findDirectiveInScript(onHostFullPath, "## Description")
			if description == "" {
				description = commandName
			}
			usage := findDirectiveInScript(onHostFullPath, "## Usage")
			if usage == "" {
				usage = commandName + " [flags] [args]"
			}
			example := findDirectiveInScript(onHostFullPath, "## Example")
			commandToAdd := &cobra.Command{
				Use:     usage,
				Short:   description + " (custom " + service + " container command)",
				Example: example,
				FParseErrWhitelist: cobra.FParseErrWhitelist{
					UnknownFlags: true,
				},
			}

			if service == "host" {
				commandToAdd.Run = makeHostCmd(app, filepath.Join(topCommandPath, service, commandName), commandName)
			} else {
				commandToAdd.Run = makeContainerCmd(app, inContainerFullPath, commandName, service)
			}
			rootCmd.AddCommand(commandToAdd)
		}
	}

	return nil
}

// makeHostCmd creates a command which will run on the host
func makeHostCmd(app *ddevapp.DdevApp, fullPath, name string) func(*cobra.Command, []string) {
	return func(cmd *cobra.Command, args []string) {
		args = []string{}
		if len(os.Args) > 2 {
			args = os.Args[2:]
		}
		_ = os.Chdir(app.AppRoot)
		err := exec.RunInteractiveCommand(fullPath, args)
		if err != nil {
			util.Failed("Failed to run %s %v: %v", name, strings.Join(args, " "), err)
		}
	}
}

// makeContainerCmd creates the command which will app.Exec to a container command
func makeContainerCmd(app *ddevapp.DdevApp, fullPath, name string, service string) func(*cobra.Command, []string) {
	return func(cmd *cobra.Command, args []string) {
		app.DockerEnv()

		args = []string{}
		if len(os.Args) > 2 {
			args = os.Args[2:]
		}
		_, _, err := app.Exec(&ddevapp.ExecOpts{
			Cmd:       fullPath + " " + strings.Join(args, " "),
			Service:   service,
			Dir:       app.WorkingDir[service],
			Tty:       isatty.IsTerminal(os.Stdin.Fd()),
			NoCapture: true,
		})

		if err != nil {
			util.Failed("Failed to run %s %v: %v", name, strings.Join(args, " "), err)
		}
	}
}

// findDirectiveInScript() looks for the named directive and returns the string following colon and spaces
func findDirectiveInScript(script string, directive string) string {
	f, err := os.Open(script)
	if err != nil {
		util.Failed("Failed to open %s: %v", script, err)
	}

	// nolint errcheck
	defer f.Close()

	// Splits on newlines by default.
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, directive) && strings.Contains(line, ":") {
			parts := strings.Split(line, ":")
			return strings.Trim(parts[1], " ")
		}
	}

	if err := scanner.Err(); err != nil {
		return ""
	}

	return ""
}

// checkDdevVersionAndOptInInstrumentation() reads global config and checks to see if current version is different
// from the last saved version. If it is, prompt to request anon ddev usage stats
// and update the info.
func checkDdevVersionAndOptInInstrumentation() error {
	if !output.JSONOutput && version.COMMIT != globalconfig.DdevGlobalConfig.LastUsedVersion && globalconfig.DdevGlobalConfig.InstrumentationOptIn == false && !globalconfig.DdevNoSentry {
		allowStats := util.Confirm("It looks like you have a new ddev release.\nMay we send anonymous ddev usage statistics and errors?\nTo know what we will see please take a look at\nhttps://ddev.readthedocs.io/en/latest/users/cli-usage/#opt-in-usage-information\nPermission to beam up?")
		if allowStats {
			globalconfig.DdevGlobalConfig.InstrumentationOptIn = true
		}
		globalconfig.DdevGlobalConfig.LastUsedVersion = version.VERSION
		err := globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig)
		if err != nil {
			return err
		}
	}
	return nil
}
