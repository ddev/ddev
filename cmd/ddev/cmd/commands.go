package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
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
	// Custom commands are shell scripts - so we can't use them on windows without bash.
	if runtime.GOOS == "windows" {
		windowsBashPath := util.FindBashPath()
		if windowsBashPath == "" {
			fmt.Println("Unable to find bash.exe in PATH, not loading custom commands")
			return nil
		}
	}

	// Keep a map so we don't add multiple commands with the same name.
	commandsAdded := map[string]int{}

	app, err := ddevapp.GetActiveApp("")
	// If we're not running ddev inside a project directory, we should still add any host commands that can run without one.
	if err != nil {
		globalHostCommandPath := filepath.Join(globalconfig.GetGlobalDdevDir(), "commands", "host")
		commandFiles, err := fileutil.ListFilesInDir(globalHostCommandPath)
		if err != nil {
			return err
		}
		err = addCustomCommandsFromDir(rootCmd, nil, globalHostCommandPath, commandFiles, true, commandsAdded)
		if err != nil {
			return err
		}
		return nil
	}

	projectCommandPath := app.GetConfigPath("commands")
	// Make sure our target global command directory is empty
	copiedGlobalCommandPath := app.GetConfigPath(".global_commands")
	err = os.MkdirAll(copiedGlobalCommandPath, 0755)
	if err != nil {
		return err
	}

	for _, commandSet := range []string{projectCommandPath, copiedGlobalCommandPath} {
		commandDirs, err := fileutil.ListFilesInDirFullPath(commandSet)
		if err != nil {
			return err
		}
		for _, serviceDirOnHost := range commandDirs {
			// If the item isn't a directory, skip it.
			if !fileutil.IsDirectory(serviceDirOnHost) {
				continue
			}
			commandFiles, err := fileutil.ListFilesInDir(serviceDirOnHost)
			if err != nil {
				return err
			}
			err = addCustomCommandsFromDir(rootCmd, app, serviceDirOnHost, commandFiles, commandSet == copiedGlobalCommandPath, commandsAdded)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// addCustomCommandsFromDir adds the custom commands from inside a given directory
func addCustomCommandsFromDir(rootCmd *cobra.Command, app *ddevapp.DdevApp, serviceDirOnHost string, commandFiles []string, isGlobalSet bool, commandsAdded map[string]int) error {
	service := filepath.Base(serviceDirOnHost)
	var err error

	for _, commandName := range commandFiles {
		// Use path.Join() for the inContainerFullPath because it's about the path in the container, not on the
		// host; a Windows path is not useful here.
		inContainerFullPath := path.Join("/mnt/ddev_config", filepath.Base(filepath.Dir(serviceDirOnHost)), service, commandName)
		onHostFullPath := filepath.Join(serviceDirOnHost, commandName)

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
			util.Warning("Command '%s' contains CRLF, please convert to Linux-style linefeeds with dos2unix or another tool, skipping %s", commandName, onHostFullPath)
			continue
		}

		directives := findDirectivesInScriptCommand(onHostFullPath)
		var description, usage, example, projectTypes, osTypes, hostBinaryExists, dbTypes string

		// Skip host commands that need a project if we aren't in a project directory.
		if service == "host" && app == nil {
			if val, ok := directives["CanRunGlobally"]; !ok || val != "true" {
				continue
			}
		}

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

		autocompleteTerms := []string{}
		if val, ok := directives["AutocompleteTerms"]; ok {
			if err = json.Unmarshal([]byte(val), &autocompleteTerms); err != nil {
				util.Warning("Error '%s', command '%s' contains an invalid autocomplete args definition '%s', skipping adding terms", err, commandName, val)
			}
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

		// Default is to exec with Bash interpretation (not raw)
		execRaw := false
		if val, ok := directives["ExecRaw"]; ok {
			if val == "true" {
				execRaw = true
			}
		}

		// If ProjectTypes is specified and we aren't of that type, skip
		if projectTypes != "" && (app == nil || !strings.Contains(projectTypes, app.Type)) {
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
		if dbTypes != "" && app != nil {
			if !strings.Contains(dbTypes, app.Database.Type) {
				continue
			}
		}

		// Create proper description suffix
		descSuffix := " (shell " + service + " container command)"
		if isGlobalSet {
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
			ValidArgs: autocompleteTerms,
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

		autocompletePathOnHost := filepath.Join(serviceDirOnHost, "autocomplete", commandName)
		if service == "host" {
			commandToAdd.Run = makeHostCmd(app, onHostFullPath, commandName)
			if fileutil.FileExists(autocompletePathOnHost) {
				// Make sure autocomplete script can be executed
				_ = os.Chmod(autocompletePathOnHost, 0755)
				if hasCR, _ := fileutil.FgrepStringInFile(autocompletePathOnHost, "\r\n"); hasCR {
					util.Warning("Command '%s' contains CRLF, please convert to Linux-style linefeeds with dos2unix or another tool, skipping %s", commandName, onHostFullPath)
					continue
				}
				// Add autocomplete script
				commandToAdd.ValidArgsFunction = makeHostCompletionFunc(autocompletePathOnHost, commandToAdd)
			}
		} else {
			commandToAdd.Run = makeContainerCmd(app, inContainerFullPath, commandName, service, execRaw, relative)
			if fileutil.FileExists(autocompletePathOnHost) {
				// Make sure autocomplete script can be executed
				_ = os.Chmod(autocompletePathOnHost, 0755)
				if hasCR, _ := fileutil.FgrepStringInFile(autocompletePathOnHost, "\r\n"); hasCR {
					util.Warning("Command '%s' contains CRLF, please convert to Linux-style linefeeds with dos2unix or another tool, skipping %s", commandName, onHostFullPath)
					continue
				}
				// Add autocomplete script
				autocompletePathInContainer := path.Join("/mnt/ddev_config", filepath.Base(filepath.Dir(serviceDirOnHost)), service, "autocomplete", commandName)
				commandToAdd.ValidArgsFunction = makeContainerCompletionFunc(autocompletePathInContainer, service, app, commandToAdd)
			}
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
		if ddevapp.IsBundledCustomCommand(isGlobalSet, service, commandName) {
			commandToAdd.Annotations[BundledCustomCommand] = "true"
		}

		// Add the command and mark as added
		rootCmd.AddCommand(commandToAdd)
		commandsAdded[commandName] = 1
	}
	return nil
}

func makeHostCompletionFunc(autocompletePathOnHost string, commandToAdd *cobra.Command) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(_ *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		// Add quotes to an empty item, so it gets passed as an empty string to the script
		if toComplete == "" {
			toComplete = "''"
		}
		args = append(args, toComplete)
		args = append([]string{commandToAdd.Name()}, args...)

		result, err := exec.RunCommand(autocompletePathOnHost, args)
		if err != nil {
			cobra.CompDebugln("error: "+err.Error(), true)
			return nil, cobra.ShellCompDirectiveDefault
		}

		// Turn result (which was separated by line breaks) into an array and return it to cobra to deal with
		return strings.Split(strings.TrimSpace(result), "\n"), cobra.ShellCompDirectiveDefault
	}
}

func makeContainerCompletionFunc(autocompletePathInContainer string, service string, app *ddevapp.DdevApp, commandToAdd *cobra.Command) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(_ *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		// Add quotes to an empty item, so it gets passed as an empty string to the script
		if toComplete == "" {
			toComplete = "''"
		}
		args = append(args, toComplete)
		compWords := commandToAdd.Name() + " " + strings.Join(args, " ")

		// Prepare docker exec command
		opts := &ddevapp.ExecOpts{
			Cmd:       autocompletePathInContainer + " " + compWords,
			Service:   service,
			Dir:       app.GetWorkingDir(service, ""),
			Tty:       false,
			NoCapture: false,
		}

		// Execute completion in docker container
		result, stderr, err := app.Exec(opts)
		if err != nil {
			cobra.CompDebugln("error: "+stderr+","+err.Error(), true)
			return nil, cobra.ShellCompDirectiveDefault
		}

		// Turn result (which was separated by line breaks) into an array and return it to cobra to deal with
		return strings.Split(strings.TrimSpace(result), "\n"), cobra.ShellCompDirectiveDefault
	}
}

// makeHostCmd creates a command which will run on the host
func makeHostCmd(app *ddevapp.DdevApp, fullPath, name string) func(*cobra.Command, []string) {
	var windowsBashPath = ""
	if runtime.GOOS == "windows" {
		windowsBashPath = util.FindBashPath()
	}

	return func(_ *cobra.Command, _ []string) {
		if app != nil {
			status, _ := app.SiteStatus()
			app.DockerEnv()
			_ = os.Setenv("DDEV_PROJECT_STATUS", status)
		} else {
			_ = os.Setenv("DDEV_PROJECT_STATUS", "")
		}

		osArgs := []string{}
		if len(os.Args) > 2 {
			osArgs = os.Args[2:]
		}
		var err error
		// Load environment variables that may be useful for script.
		if app != nil {
			app.DockerEnv()
		}
		if runtime.GOOS == "windows" {
			// Sadly, not sure how to have a Bash interpreter without this.
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
	return func(_ *cobra.Command, _ []string) {
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
