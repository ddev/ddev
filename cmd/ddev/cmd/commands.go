package cmd

import (
	"bufio"
	"fmt"
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/drud/ddev/pkg/util"
	"github.com/gobuffalo/packr/v2"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

// addCustomCommands looks for custom command scripts in
// ~/.ddev/commands/<servicename> etc. and
// .ddev/commands/<servicename> and .ddev/commands/host
// and if it finds them adds them to Cobra's commands.
func addCustomCommands(rootCmd *cobra.Command) error {
	app, err := ddevapp.GetActiveApp("")
	if err != nil {
		return nil
	}

	sourceGlobalCommandPath := filepath.Join(globalconfig.GetGlobalDdevDir(), "commands")
	err = os.MkdirAll(sourceGlobalCommandPath, 0755)
	if err != nil {
		return nil
	}

	projectCommandPath := app.GetConfigPath("commands")
	// Make sure our target global command directory is empty
	targetGlobalCommandPath := app.GetConfigPath(".global_commands")
	_ = os.RemoveAll(targetGlobalCommandPath)

	err = fileutil.CopyDir(sourceGlobalCommandPath, targetGlobalCommandPath)
	if err != nil {
		return err
	}

	if !fileutil.FileExists(projectCommandPath) || !fileutil.IsDirectory(projectCommandPath) {
		return nil
	}

	commandsAdded := map[string]int{}
	for _, commandSet := range []string{projectCommandPath, targetGlobalCommandPath} {
		commandDirs, err := fileutil.ListFilesInDirFullPath(commandSet)
		if err != nil {
			return err
		}
		for _, serviceDirOnHost := range commandDirs {
			service := filepath.Base(serviceDirOnHost)

			// If the item isn't actually a directory, just skip it.
			if !fileutil.IsDirectory(serviceDirOnHost) {
				continue
			}
			commandFiles, err := fileutil.ListFilesInDir(serviceDirOnHost)
			if err != nil {
				return err
			}
			if runtime.GOOS == "windows" {
				windowsBashPath := util.FindWindowsBashPath()
				if windowsBashPath == "" {
					fmt.Println("Unable to find bash.exe in PATH, not loading custom commands")
					return nil
				}
			}

			for _, commandName := range commandFiles {
				// Use path.Join() for the inContainerFullPath because it'serviceDirOnHost about the path in the container, not on the
				// host; a Windows path is not useful here.
				inContainerFullPath := path.Join("/mnt/ddev_config", filepath.Base(commandSet), service, commandName)
				onHostFullPath := filepath.Join(commandSet, service, commandName)

				if strings.HasSuffix(commandName, ".example") || strings.HasPrefix(commandName, "README") || strings.HasPrefix(commandName, ".") || fileutil.IsDirectory(onHostFullPath) {
					continue
				}

				// If command has already been added, we won't work with it again.
				if _, ok := commandsAdded[commandName]; ok {
					util.Warning("not adding custom command %s because it was already added", onHostFullPath)
					continue
				}

				// Any command we find will want to be executable on Linux
				_ = os.Chmod(onHostFullPath, 0755)
				if hasCR, _ := fileutil.FgrepStringInFile(onHostFullPath, "\r\n"); hasCR {
					util.Warning("command '%s' contains CRLF, please convert to Linux-style linefeeds with dos2unix or another tool, skipping %s", commandName, onHostFullPath)
					continue
				}
				description := findDirectiveInScript(onHostFullPath, "## Description")
				if description == "" {
					description = commandName
				}
				usage := findDirectiveInScript(onHostFullPath, "## Usage")
				if usage == "" {
					usage = commandName + " [flags] [args]"
				}
				example := findDirectiveInScript(onHostFullPath, "## Example")
				descSuffix := " (shell " + service + " container command)"
				if serviceDirOnHost[0:1] == "." {
					descSuffix = " (global shell " + service + " container command)"
				}
				commandToAdd := &cobra.Command{
					Use:     usage,
					Short:   description + descSuffix,
					Example: example,
					FParseErrWhitelist: cobra.FParseErrWhitelist{
						UnknownFlags: true,
					},
				}

				if service == "host" {
					commandToAdd.Run = makeHostCmd(app, onHostFullPath, commandName)
				} else {
					commandToAdd.Run = makeContainerCmd(app, inContainerFullPath, commandName, service)
				}
				rootCmd.AddCommand(commandToAdd)

				commandsAdded[commandName] = 1
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
	s := service
	if s[0:1] == "." {
		s = s[1:]
	}
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
			Service:   s,
			Dir:       app.GetWorkingDir(s, ""),
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
