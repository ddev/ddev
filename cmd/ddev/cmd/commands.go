package cmd

import (
	"bufio"
	"fmt"
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/util"
	"github.com/gobuffalo/packr/v2"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
	"io/ioutil"
	"os"
	osexec "os/exec"
	"path"
	"path/filepath"
	"runtime"
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
		if runtime.GOOS == "windows" {
			windowsBashPath := findWindowsBashPath()
			if windowsBashPath == "" {
				fmt.Println("Unable to find bash.exe in PATH, not loading custom commands")
				return nil
			}
		}

		for _, commandName := range commandFiles {
			if strings.HasSuffix(commandName, ".example") || strings.HasPrefix(commandName, "README") {
				continue
			}
			// Use path.Join() for the inContainerFullPath because it's about the path in the container, not on the
			// host; a Windows path is not useful here.
			inContainerFullPath := path.Join("/mnt/ddev_config/commands", service, commandName)
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
	var windowsBashPath = ""
	if runtime.GOOS == "windows" {
		windowsBashPath = findWindowsBashPath()
	}

	return func(cmd *cobra.Command, cobraArgs []string) {
		if app.SiteStatus() != ddevapp.SiteRunning {
			err := app.Start()
			if err != nil {
				util.Failed("Failed to start project for custom command: %v", err)
			}
		}
		app.DockerEnv()

		osArgs := []string{}
		if len(os.Args) > 2 {
			osArgs = os.Args[2:]
		}
		var err error
		// Load environment variables that may be useful for script.
		app.DockerEnv()
		if runtime.GOOS == "windows" {
			// Sadly, not sure how to have a bash interpreter without this.
			args := []string{fullPath}
			args = append(args, osArgs...)
			err = exec.RunInteractiveCommand(windowsBashPath, args)
		} else {
			err = exec.RunInteractiveCommand(fullPath, osArgs)
		}
		if err != nil {
			util.Failed("Failed to run %s %v; error=%v", name, strings.Join(osArgs, " "), err)
		}
	}
}

// makeContainerCmd creates the command which will app.Exec to a container command
func makeContainerCmd(app *ddevapp.DdevApp, fullPath, name string, service string) func(*cobra.Command, []string) {
	return func(cmd *cobra.Command, args []string) {
		if app.SiteStatus() != ddevapp.SiteRunning {
			err := app.Start()
			if err != nil {
				util.Failed("Failed to start project for custom command: %v", err)
			}
		}
		app.DockerEnv()

		osArgs := []string{}
		if len(os.Args) > 2 {
			osArgs = os.Args[2:]
		}
		_, _, err := app.Exec(&ddevapp.ExecOpts{
			Cmd:       fullPath + " " + strings.Join(osArgs, " "),
			Service:   service,
			Dir:       app.GetWorkingDir(service, ""),
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

// populateExamplesAndCommands grabs packr2 assets
// When the items in the assets directory are changed, the packr2 command
// must be run again in this directory (cmd/ddev/cmd) to update the saved
// embedded files.
// "make packr2" can be used to update the packr2 cache.
func populateExamplesAndCommands() error {
	app, err := ddevapp.GetActiveApp("")
	if err != nil {
		return nil
	}
	box := packr.New("customcommands", "./dotddev_assets")

	list := box.List()
	for _, file := range list {
		localPath := app.GetConfigPath(file)
		if !fileutil.FileExists(localPath) {
			content, err := box.Find(file)
			if err != nil {
				return err
			}
			err = os.MkdirAll(filepath.Dir(localPath), 0755)
			if err != nil {
				return err
			}
			err = ioutil.WriteFile(localPath, content, 0755)
			if err != nil {
				return err
			}

		}
	}
	return nil
}

// On Windows we'll need the path to bash to execute anything.
// Returns empty string if not found, path if found
func findWindowsBashPath() string {
	windowsBashPath, err := osexec.LookPath(`C:\Program Files\Git\bin\bash.exe`)
	if err != nil {
		// This one could come back with the WSL bash, in which case we may have some trouble.
		windowsBashPath, err = osexec.LookPath("bash.exe")
		if err != nil {
			fmt.Println("Not loading custom commands; bash is not in PATH")
			return ""
		}
	}
	return windowsBashPath
}
