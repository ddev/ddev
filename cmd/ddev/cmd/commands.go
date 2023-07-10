package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/util"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

const (
	CustomCommand        = "customCommand"
	BundledCustomCommand = "customCommand:bundled"
)

func IsUserDefinedCustomCommand(cmd *cobra.Command) bool {
	_, customCommand := cmd.Annotations[CustomCommand]
	_, bundledCustomCommand := cmd.Annotations[BundledCustomCommand]

	return customCommand && !bundledCustomCommand
}

// addCustomCommands looks for custom command scripts in
// ~/.ddev/commands/<servicename> etc. and
// .ddev/commands/<servicename> and .ddev/commands/host
// and if it finds them adds them to Cobra's commands.
func addCustomCommands(rootCmd *cobra.Command) error {
	app, err := ddevapp.GetActiveApp("")
	if err != nil {
		return nil
	}

	projectCommandPath := app.GetConfigPath("commands")
	// Make sure our target global command directory is empty
	copiedGlobalCommandPath := app.GetConfigPath(".global_commands")
	err = os.MkdirAll(copiedGlobalCommandPath, 0755)
	if err != nil {
		return err
	}
	commandsAdded := map[string]int{}
	for _, commandSet := range []string{projectCommandPath, copiedGlobalCommandPath} {
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
				windowsBashPath := util.FindBashPath()
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
					continue
				}

				// Any command we find will want to be executable on Linux
				_ = os.Chmod(onHostFullPath, 0755)
				if hasCR, _ := fileutil.FgrepStringInFile(onHostFullPath, "\r\n"); hasCR {
					util.Warning("command '%s' contains CRLF, please convert to Linux-style linefeeds with dos2unix or another tool, skipping %s", commandName, onHostFullPath)
					continue
				}

				directives := findDirectivesInScriptCommand(onHostFullPath)
				var description, usage, example, projectTypes, osTypes, hostBinaryExists, dbTypes string

				description = commandName
				if val, ok := directives["Description"]; ok {
					description = val
				}

				usage = commandName + " [flags] [args]"
				if val, ok := directives["Usage"]; ok {
					usage = val
				}

				if val, ok := directives["Example"]; ok {
					example = "  " + strings.ReplaceAll(val, `\n`, "\n  ")
				}

				// Init and import flags
				var flags Flags
				flags.Init(commandName, onHostFullPath)

				disableFlags := true
				if val, ok := directives["Flags"]; ok {
					disableFlags = false
					if err = flags.LoadFromJSON(val); err != nil {
						util.Warning("Error '%s', command '%s' contains an invalid flags definition '%s', skipping add flags of %s", err, commandName, val, onHostFullPath)
					}
				}

				// Import and handle ProjectTypes
				if val, ok := directives["ProjectTypes"]; ok {
					projectTypes = val
				}

				// Default is to exec with bash interpretation (not raw)
				execRaw := false
				if val, ok := directives["ExecRaw"]; ok {
					if val == "true" {
						execRaw = true
					}
				}

				// If ProjectTypes is specified and we aren't of that type, skip
				if projectTypes != "" && !strings.Contains(projectTypes, app.Type) {
					continue
				}

				// Import and handle OSTypes
				if val, ok := directives["OSTypes"]; ok {
					osTypes = val
				}

				// If OSTypes is specified and we aren't on one of the specified OSes, skip
				if osTypes != "" {
					if !strings.Contains(osTypes, runtime.GOOS) && !(strings.Contains(osTypes, "wsl2") && nodeps.IsWSL2()) {
						continue
					}
				}

				// Import and handle HostBinaryExists
				if val, ok := directives["HostBinaryExists"]; ok {
					hostBinaryExists = val
				}

				// If hostBinaryExists is specified it doesn't exist here, skip
				if hostBinaryExists != "" {
					binExists := false
					bins := strings.Split(hostBinaryExists, ",")
					for _, bin := range bins {
						if fileutil.FileExists(bin) {
							binExists = true
							break
						}
					}
					if !binExists {
						continue
					}
				}

				// Import and handle DBTypes
				if val, ok := directives["DBTypes"]; ok {
					dbTypes = val
				}

				// If DBTypes is specified and we aren't using that DBTypes
				if dbTypes != "" {
					if !strings.Contains(dbTypes, app.Database.Type) {
						continue
					}
				}

				// Create proper description suffix
				descSuffix := " (shell " + service + " container command)"
				if commandSet == copiedGlobalCommandPath {
					descSuffix = " (global shell " + service + " container command)"
				}

				// Initialize the new command
				commandToAdd := &cobra.Command{
					Use:                usage,
					Short:              description + descSuffix,
					Example:            example,
					DisableFlagParsing: disableFlags,
					FParseErrWhitelist: cobra.FParseErrWhitelist{
						UnknownFlags: true,
					},
				}

				// Add flags to command
				if err = flags.AssignToCommand(commandToAdd); err != nil {
					util.Warning("Error '%s' in the flags definition for command '%s', skipping %s", err, commandName, onHostFullPath)
					continue
				}

				// Execute the command matching the host working directory relative
				// to the app root.
				relative := false
				if val, ok := directives["HostWorkingDir"]; ok {
					if val == "true" {
						relative = true
					}
				}

				if service == "host" {
					commandToAdd.Run = makeHostCmd(app, onHostFullPath, commandName)
				} else {
					commandToAdd.Run = makeContainerCmd(app, inContainerFullPath, commandName, service, execRaw, relative)
				}

				if disableFlags {
					// Hide -h because we are disabling flags
					// Also hide --json-output for the same reason
					// @see https://github.com/spf13/cobra/issues/1328
					commandToAdd.InitDefaultHelpFlag()
					err = commandToAdd.Flags().MarkHidden("help")
					originalHelpFunc := commandToAdd.HelpFunc()
					if err == nil {
						commandToAdd.SetHelpFunc(func(command *cobra.Command, strings []string) {
							_ = command.Flags().MarkHidden("json-output")
							originalHelpFunc(command, strings)
						})
					}
				}

				// Mark custom command
				if commandToAdd.Annotations == nil {
					commandToAdd.Annotations = map[string]string{}
				}

				commandToAdd.Annotations[CustomCommand] = "true"
				if ddevapp.IsBundledCustomCommand(commandSet == copiedGlobalCommandPath, service, commandName) {
					commandToAdd.Annotations[BundledCustomCommand] = "true"
				}

				// Add the command and mark as added
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
		windowsBashPath = util.FindBashPath()
	}

	return func(cmd *cobra.Command, cobraArgs []string) {
		status, _ := app.SiteStatus()
		if status != ddevapp.SiteRunning {
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
func makeContainerCmd(app *ddevapp.DdevApp, fullPath, name, service string, execRaw bool, relative bool) func(*cobra.Command, []string) {
	s := service
	if s[0:1] == "." {
		s = s[1:]
	}
	return func(cmd *cobra.Command, args []string) {
		status, _ := app.SiteStatus()
		if status != ddevapp.SiteRunning {
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

		opts := &ddevapp.ExecOpts{
			Cmd:       fullPath + " " + strings.Join(osArgs, " "),
			Service:   s,
			Dir:       app.GetWorkingDir(s, ""),
			Tty:       isatty.IsTerminal(os.Stdin.Fd()),
			NoCapture: true,
		}
		if relative {
			opts.Dir = path.Join(opts.Dir, app.GetRelativeWorkingDirectory())
		}

		if execRaw {
			opts.RawCmd = append([]string{fullPath}, osArgs...)
		}
		_, _, err := app.Exec(opts)

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
			if parts[0] == "Example" {
				parts[1] = strings.Trim(parts[1], " ")
			} else {
				parts[1] = strings.Trim(parts[1], " \"'")
			}
			directives[parts[0]] = parts[1]
		}
	}

	if err := scanner.Err(); err != nil {
		return nil
	}

	return directives
}
