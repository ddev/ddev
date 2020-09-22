package cmd

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/drud/ddev/pkg/util"
	"github.com/gobuffalo/packr/v2"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

// Custom local types
type commandType string
type commandTypes []commandType
type commandsAdded map[string]bool

// addCustomCommands looks for custom command scripts in
// ~/.ddev/commands/<servicename> etc. and
// .ddev/commands/<servicename> and .ddev/commands/host
// and if it finds them adds them to Cobra's commands.
func addCustomCommands(command *cobra.Command) error {
	// Make sure we are running in a project context
	if app, err := ddevapp.GetActiveApp(""); err != nil {
		return nil
	}

	//
	addInternalCommands(command)

	// commandTypes will hold all valid command types, currently global or local
	// project commands
	var commandTypes commandTypes

	// Prepare custom commands
	if projectCommandPath, err := app.prepareCustomCommands(); err != nil {
		util.Warning("Preparing custom commands failed: %v", err)
	} else {
		commandTypes = append(commandTypes, projectCommandPath)
	}

	// Prepare global custom commands
	if projectGlobalCommandPath, err := app.prepareGlobalCustomCommands(); err != nil {
		util.Warning("Preparing global custom commands failed: %v", err)
	} else {
		commandTypes = append(commandTypes, projectGlobalCommandPath)
	}

	// Check if there are any command types or early return
	if len(commandTypes) == 0 {
		return fmt.Errorf("Unable to find custom commands")
	}

	// Remember added commands to avoid overwriting by later commands and
	// to show an information about the added commands to the user.
	var commandsAdded commandsAdded

	if err = commandTypes.addToCommand(c, &commandsAdded); err == nil {
		util.Info("%d custom %s added", len(commandsAdded), utils.FormatPlural(len(commandsAdded), "command", "commands"))
	}

	return err
}

func prepareCustomCommands(app *ddevapp.DdevApp) (string, error) {
	// Get the custom commands path for this project
	path := app.GetConfigPath("commands")

	if !fileutil.FileExists(path) || !fileutil.IsDirectory(path) {
		return nil, fmt.Errorf("Custom command directory '%s' does not exist or is not a directory", path)
	}

	return path, nil
}

func prepareGlobalCustomCommands(app *ddevapp.DdevApp) (string, error) {
	// Get the global custom commands path for this project
	path := app.GetConfigPath(".global_commands")

	if err := copyGlobalCustomCommands(path); err != nil {
		return nil, err
	}

	return path, nil
}

func copyGlobalCustomCommands(targetPath string) error {
	// Calculate the source path of the global custom commands
	sourcePath := filepath.Join(globalconfig.GetGlobalDdevDir(), "commands")

	// Make sure the source path exists
	if err := os.MkdirAll(sourcePath, 0755); err != nil {
		return err
	}

	// Make sure our target global command directory is empty
	if err = os.RemoveAll(targetPath); err != nil {
		return err
	}

	// Copy the global commands to the project target
	if err = fileutil.CopyDir(sourcePath, targetPath); err != nil {
		return err
	}

	return nil
}

func (t *commandTypes) addToCommand(command *cobra.Command, commandsAdded *commandsAdded) error {
	for _, commandType := range t {
		// Get service direcotries
		if serviceDirs, err := fileutil.ListDirectoriesWithFullPath(commandType, true); err != nil {
			util.Warning("Failed to list directories of '%s': %v", commandType, err)
			continue
		}

		// Process service directories
		for _, serviceDir := range serviceDirs {
			// Extract the service name from the directory
			service := filepath.Base(serviceDir)

			// On Windows check for the existence of bash or early return, todo: change see last PR
			if runtime.GOOS == "windows" {
				windowsBashPath := util.FindWindowsBashPath()
				if windowsBashPath == "" {
					return fmt.Errorf("Unable to find bash.exe in PATH, not loading custom commands")
				}
			}

			if serviceFiles, err := fileutil.ListFilesInDir(serviceDir, true); err != nil {
				util.Warning("Failed to list files of '%s': %v", serviceDir, err)
				continue
			}

			for _, serviceFile := range serviceFiles {
				script := createCustomCommandScript(serviceFile)

				if err = s.addToCommand(command, commandsAdded); err != nil {
					util.Warning("Failed to add script '%s': %v", serviceFile, err)
				}
			}
		}
	}

	return nil
}

// makeHostCmd creates a command which will run on the host
func makeHostCmd(app *ddevapp.DdevApp, fullPath, name string) func(*cobra.Command, []string) {
	var windowsBashPath = ""
	if runtime.GOOS == "windows" {
		windowsBashPath = util.FindWindowsBashPath()
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

// findDirectivesInScriptCommand() Returns a map of directives and their contents
// found in the named script
func findDirectivesInScriptCommand(script string) map[string]string {
	f, err := os.Open(script)
	if err != nil {
		util.Failed("Failed to open %s: %v", script, err)
	}

	// nolint errcheck
	defer f.Close()

	var directives = make(map[string]string)

	// Splits on newlines by default.
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "## ") && strings.Contains(line, ":") {
			line = strings.Replace(line, "## ", "", 1)
			parts := strings.SplitN(line, ":", 2)
			parts[1] = strings.Trim(parts[1], " ")
			directives[parts[0]] = parts[1]
		}
	}

	if err := scanner.Err(); err != nil {
		return nil
	}

	return directives
}

// populateExamplesCommandsHomeadditions grabs packr2 assets
// When the items in the assets directory are changed, the packr2 command
// must be run again in this directory (cmd/ddev/cmd) to update the saved
// embedded files.
// "make packr2" can be used to update the packr2 cache.
func populateExamplesCommandsHomeadditions() error {
	app, err := ddevapp.GetActiveApp("")
	if err != nil {
		return nil
	}
	box := packr.New("customcommands", "./dotddev_assets")

	list := box.List()
	for _, file := range list {
		localPath := app.GetConfigPath(file)
		sigFound, err := fileutil.FgrepStringInFile(localPath, ddevapp.DdevFileSignature)
		if sigFound || err != nil {
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

	// This brings in both the commands and the homeadditions files
	box = packr.New("global_dotddev", "./global_dotddev_assets")

	list = box.List()
	globalDdevDir := globalconfig.GetGlobalDdevDir()
	for _, file := range list {
		localPath := filepath.Join(globalDdevDir, file)
		sigFound, err := fileutil.FgrepStringInFile(localPath, ddevapp.DdevFileSignature)
		if sigFound || err != nil {
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
