package cmd

import (
	"bufio"
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/util"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
	"strings"
)

// addCustomCommands looks for custom command scripts in
// .ddev/commands/<servicename> and .ddev/commands/host
// and if it finds them adds them to Cobra's commands.
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
		osArgs := []string{}
		if len(os.Args) > 2 {
			os.Args = os.Args[2:]
		}
		_ = os.Chdir(app.AppRoot)
		err := exec.RunInteractiveCommand(fullPath, osArgs)
		if err != nil {
			util.Failed("Failed to run %s %v: %v", name, strings.Join(osArgs, " "), err)
		}
	}
}

// makeContainerCmd creates the command which will app.Exec to a container command
func makeContainerCmd(app *ddevapp.DdevApp, fullPath, name string, service string) func(*cobra.Command, []string) {
	return func(cmd *cobra.Command, args []string) {
		app.DockerEnv()

		osArgs := []string{}
		if len(os.Args) > 2 {
			osArgs = os.Args[2:]
		}
		_, _, err := app.Exec(&ddevapp.ExecOpts{
			Cmd:       fullPath + " " + strings.Join(osArgs, " "),
			Service:   service,
			Dir:       app.WorkingDir[service],
			Tty:       isatty.IsTerminal(os.Stdin.Fd()),
			NoCapture: true,
		})

		if err != nil {
			util.Failed("Failed to run %s %v: %v", name, strings.Join(osArgs, " "), err)
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
