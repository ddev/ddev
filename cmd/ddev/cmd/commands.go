package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/util"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type annotations = struct {
	// @TODO maybe some should not be omit empty
	// @TODO maybe allow or even enforce capital letter starts
	// @TODO check for any missing annotations and make sure we parse and use them
	Aliases           []string `yaml:"aliases,omitempty"`
	AutocompleteTerms []string `yaml:"autocompleteTerms,omitempty"`
	CanRunGlobally    bool     `yaml:"canRunGlobally,omitempty"`
	DbTypes           []string `yaml:"dbTypes,omitempty"`
	Description       string   `yaml:"description,omitempty"`
	DisableFlags      bool     `yaml:"-"`
	Example           string   `yaml:"example,omitempty"`
	ExecRaw           bool     `yaml:"execRaw,omitempty"`
	Flags             Flags    `yaml:"flags,omitempty"`
	HostBinaryExists  string   `yaml:"hostBinaryExists,omitempty"`
	HostWorkingDir    bool     `yaml:"hostWorkingDir,omitempty"`
	MutagenSync       bool     `yaml:"mutagenSynx,omitempty"`
	OsTypes           []string `yaml:"osTypes,omitempty"`
	ProjectTypes      []string `yaml:"projectTypes,omitempty"`
	Usage             string   `yaml:"usage,omitempty"`
}

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
	globalCommandPath := filepath.Join(globalconfig.GetGlobalDdevDir(), "commands")
	err = os.MkdirAll(globalCommandPath, 0755)
	if err != nil {
		return err
	}

	for _, commandSet := range []string{projectCommandPath, globalCommandPath} {
		// If the item isn't a directory, skip it.
		if !fileutil.IsDirectory(commandSet) {
			continue
		}
		commandDirs, err := fileutil.ListFilesInDirFullPath(commandSet, false)
		if err != nil {
			return err
		}
		for _, serviceDirOnHost := range commandDirs {
			// If the item isn't a directory, skip it.
			if !fileutil.IsDirectory(serviceDirOnHost) {
				continue
			}
			// Skip hidden directories as well.
			if strings.HasPrefix(filepath.Base(serviceDirOnHost), ".") {
				continue
			}
			commandFiles, err := fileutil.ListFilesInDir(serviceDirOnHost)
			if err != nil {
				return err
			}
			err = addCustomCommandsFromDir(rootCmd, app, serviceDirOnHost, commandFiles, commandSet == globalCommandPath, commandsAdded)
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

	for _, commandName := range commandFiles {
		onHostFullPath := filepath.Join(serviceDirOnHost, commandName)

		if strings.HasSuffix(commandName, ".example") || strings.HasPrefix(commandName, "README") || strings.HasPrefix(commandName, ".") || fileutil.IsDirectory(onHostFullPath) {
			continue
		}

		// If command has already been added, we won't work with it again.
		if _, ok := commandsAdded[commandName]; ok {
			continue
		}

		// Any command we find will want to be executable on Linux
		// Note that this only affects host commands and project-level commands.
		// Global container commands are made executable on `ddev start` instead.
		_ = util.Chmod(onHostFullPath, 0755)
		if hasCR, _ := fileutil.FgrepStringInFile(onHostFullPath, "\r\n"); hasCR {
			util.Warning("Command '%s' contains CRLF, please convert to Linux-style linefeeds with dos2unix or another tool, skipping %s", commandName, onHostFullPath)
			continue
		}

		// Prepare an annotations struct with sensible default values
		var annotations = annotations{
			AutocompleteTerms: []string{},
			CanRunGlobally:    false,
			Description:       commandName,
			DisableFlags:      true,
			ExecRaw:           false,
			HostWorkingDir:    false,
			MutagenSync:       false,
			Usage:             commandName + " [flags] [args]",
		}

		scriptContent, err := fileutil.ReadFileIntoString(onHostFullPath)
		if err != nil {
			util.Warning("Could not read script file %s for command %s: %v", onHostFullPath, commandName, err)
			continue
		}

		// Check for and try to parse YAML annotations if a valid block is present
		re := regexp.MustCompile(`(?s)<<'###YAML_ANNOTATIONS'\n(.*)\n###YAML_ANNOTATIONS`)
		match := re.FindStringSubmatch(scriptContent)
		// A valid block will have two items in the match - the full match, and then just the YAML portion.
		if len(match) == 2 {
			err = yaml.Unmarshal([]byte(match[1]), &annotations)
			if err != nil {
				util.Warning("Couldn't read YAML annotations in script file %s for command %s: %v", onHostFullPath, commandName, err)
				continue
			}
		} else {
			// If we didn't find a valid YAML_ANNOTATIONS block, try parsing legacy annotations
			directives := findDirectivesInScriptCommand(onHostFullPath)
			parseLegacyDirectives(onHostFullPath, commandName, directives, &annotations)
		}

		// Skip host commands that need a project if we aren't in a project directory.
		if service == "host" && app == nil {
			if annotations.CanRunGlobally {
				if isCustomCommandInArgs(commandName) {
					util.Warning("Command '%s' cannot be used outside the project directory.", commandName)
				}
				continue
			}
		}

		// @TODO keep the warning logic here instead of in the two parse methods
		// @TODO will mean we need to remove those aliases from the annotation
		// var aliases []string
		// if val, ok := directives["Aliases"]; ok {
		// 	for _, alias := range strings.Split(val, ",") {
		// 		alias = strings.TrimSpace(alias)
		// 		if foundCmd, _, err := rootCmd.Find([]string{alias}); err != nil {
		// 			aliases = append(aliases, alias)
		// 		} else {
		// 			util.Warning("Command '%s' cannot have alias '%s' that is already in use by command '%s', skipping it", commandName, alias, foundCmd.Name())
		// 		}
		// 	}
		// }

		// If ProjectTypes is specified and we aren't of that type, skip
		if len(annotations.ProjectTypes) > 0 && (app == nil || !slices.Contains(annotations.ProjectTypes, app.Type)) {
			if app != nil && isCustomCommandInArgs(commandName) {
				suggestedCommands := annotations.ProjectTypes
				for i, projectType := range suggestedCommands {
					suggestedCommands[i] = fmt.Sprintf("ddev config --project-type=%s", projectType)
				}
				suggestedCommand, _ := util.ArrayToReadableOutput(suggestedCommands)
				util.Warning("Command '%s' is not available for the '%s' project type.\nIf you intend to use '%s', change the project type to one of the supported types %s", commandName, app.Type, commandName, suggestedCommand)
			}
			continue
		}

		// If OSTypes is specified and we aren't on one of the specified OSes, skip
		if len(annotations.OsTypes) > 0 {
			if !slices.Contains(annotations.OsTypes, runtime.GOOS) && !(slices.Contains(annotations.OsTypes, "wsl2") && nodeps.IsWSL2()) {
				if isCustomCommandInArgs(commandName) {
					util.Warning("Command '%s' cannot be used with your OS.", commandName)
				}
				continue
			}
		}

		// If hostBinaryExists is specified it doesn't exist here, skip
		if annotations.HostBinaryExists != "" {
			binExists := false
			bins := strings.Split(annotations.HostBinaryExists, ",")
			for _, bin := range bins {
				if fileutil.FileExists(bin) {
					binExists = true
					break
				}
			}
			if !binExists {
				if isCustomCommandInArgs(commandName) {
					suggestedBinaries, _ := util.ArrayToReadableOutput(bins)
					util.Warning("Command '%s' cannot be used, because the binary is not found at %s", commandName, suggestedBinaries)
				}
				continue
			}
		}

		// If DBTypes is specified and we aren't using that DBTypes
		if len(annotations.DbTypes) > 0 && app != nil {
			if !slices.Contains(annotations.DbTypes, app.Database.Type) {
				if isCustomCommandInArgs(commandName) {
					util.Warning("Command '%s' is not available for the '%s' database type.", commandName, app.Database.Type)
				}
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
			Use:                annotations.Usage,
			Short:              annotations.Description + descSuffix,
			Example:            annotations.Example,
			Aliases:            annotations.Aliases,
			DisableFlagParsing: annotations.DisableFlags,
			FParseErrWhitelist: cobra.FParseErrWhitelist{
				UnknownFlags: true,
			},
			ValidArgs: annotations.AutocompleteTerms,
		}

		// Add flags to command
		if err = annotations.Flags.AssignToCommand(commandToAdd); err != nil {
			util.Warning("Error '%s' in the flags definition for command '%s', skipping %s", err, commandName, onHostFullPath)
			continue
		}

		autocompletePathOnHost := filepath.Join(serviceDirOnHost, "autocomplete", commandName)
		if service == "host" {
			commandToAdd.Run = makeHostCmd(app, onHostFullPath, commandName, annotations.MutagenSync)
			if fileutil.FileExists(autocompletePathOnHost) {
				// Make sure autocomplete script can be executed
				_ = util.Chmod(autocompletePathOnHost, 0755)
				if hasCR, _ := fileutil.FgrepStringInFile(autocompletePathOnHost, "\r\n"); hasCR {
					util.Warning("Command '%s' contains CRLF, please convert to Linux-style linefeeds with dos2unix or another tool, skipping %s", commandName, onHostFullPath)
					continue
				}
				// Add autocomplete script
				commandToAdd.ValidArgsFunction = makeHostCompletionFunc(autocompletePathOnHost, commandToAdd)
			}
		} else {
			// Use path.Join() for the container path because it's about the path in the container, not on the
			// host; a Windows path is not useful here.
			containerBasePath := path.Join("/mnt/ddev_config", filepath.Base(filepath.Dir(serviceDirOnHost)), service)
			if strings.HasPrefix(serviceDirOnHost, globalconfig.GetGlobalDdevDir()) {
				containerBasePath = path.Join("/mnt/ddev-global-cache/global-commands/", service)
			}
			inContainerFullPath := path.Join(containerBasePath, commandName)
			commandToAdd.Run = makeContainerCmd(app, inContainerFullPath, commandName, service, annotations.ExecRaw, annotations.HostWorkingDir, annotations.MutagenSync)
			if fileutil.FileExists(autocompletePathOnHost) {
				// Make sure autocomplete script can be executed
				_ = util.Chmod(autocompletePathOnHost, 0755)
				if hasCR, _ := fileutil.FgrepStringInFile(autocompletePathOnHost, "\r\n"); hasCR {
					util.Warning("Autocomplete script for command '%s' contains CRLF, please convert to Linux-style linefeeds with dos2unix or another tool, skipping %s", commandName, autocompletePathOnHost)
					continue
				}
				// Add autocomplete script
				autocompletePathInContainer := path.Join(containerBasePath, "autocomplete", commandName)
				commandToAdd.ValidArgsFunction = makeContainerCompletionFunc(autocompletePathInContainer, service, app, commandToAdd)
			}
		}

		if annotations.DisableFlags {
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

// parseLegacyDirectives parses the legacy `## SomeAnnotation: someValue` style directives for custom commands.
func parseLegacyDirectives(scriptPath string, commandName string, directives map[string]string, annotations *annotations) {
	if val, ok := directives["Aliases"]; ok {
		for _, alias := range strings.Split(val, ",") {
			annotations.Aliases = append(annotations.Aliases, strings.TrimSpace(alias))
		}
	}

	if val, ok := directives["AutocompleteTerms"]; ok {
		if err := json.Unmarshal([]byte(val), &annotations.AutocompleteTerms); err != nil {
			util.Warning("Error '%s', command '%s' contains an invalid autocomplete args definition '%s', skipping adding terms", err, commandName, val)
		}
	}

	if val, ok := directives["CanRunGlobally"]; !ok || val != "true" {
		annotations.CanRunGlobally = true
	}

	if val, ok := directives["DBTypes"]; ok {
		for _, dbType := range strings.Split(val, ",") {
			annotations.DbTypes = append(annotations.DbTypes, strings.TrimSpace(dbType))
		}
	}

	if val, ok := directives["Description"]; ok {
		annotations.Description = val
	}

	if val, ok := directives["Example"]; ok {
		annotations.Example = "  " + strings.ReplaceAll(val, `\n`, "\n  ")
	}

	if val, ok := directives["ExecRaw"]; ok {
		if val == "true" {
			annotations.ExecRaw = true
		}
	}

	// Init and import flags
	annotations.Flags.Init(commandName, scriptPath) // @TODO do I need this in the YAML one?
	if val, ok := directives["Flags"]; ok {
		annotations.DisableFlags = false
		if err := annotations.Flags.LoadFromJSON(val); err != nil { // @TODO do I need to do something like this for the YAML one?
			util.Warning("Error '%s', command '%s' contains an invalid flags definition '%s', skipping add flags of %s", err, commandName, val, scriptPath)
		}
	}

	if val, ok := directives["HostBinaryExists"]; ok {
		annotations.HostBinaryExists = val
	}

	if val, ok := directives["HostWorkingDir"]; ok {
		if val == "true" {
			annotations.HostWorkingDir = true
		}
	}

	if val, ok := directives["MutagenSync"]; ok {
		if val == "true" {
			annotations.MutagenSync = true
		}
	}

	if val, ok := directives["OSTypes"]; ok {
		for _, osType := range strings.Split(val, ",") {
			annotations.OsTypes = append(annotations.OsTypes, strings.TrimSpace(osType))
		}
	}

	if val, ok := directives["ProjectTypes"]; ok {
		for _, projectType := range strings.Split(val, ",") {
			annotations.ProjectTypes = append(annotations.ProjectTypes, strings.TrimSpace(projectType))
		}
	}

	if val, ok := directives["Usage"]; ok {
		annotations.Usage = val
	}
}

// isCustomCommandInArgs checks if the command is the first arg passed to the "ddev" command.
func isCustomCommandInArgs(commandName string) bool {
	return len(os.Args) > 1 && os.Args[1] == commandName
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
func makeHostCmd(app *ddevapp.DdevApp, fullPath, name string, mutagenSync bool) func(*cobra.Command, []string) {
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

		runMutagenSync(app, mutagenSync)

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

		runMutagenSync(app, mutagenSync)
	}
}

// makeContainerCmd creates the command which will app.Exec to a container command
func makeContainerCmd(app *ddevapp.DdevApp, fullPath, name, service string, execRaw bool, relative bool, mutagenSync bool) func(*cobra.Command, []string) {
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

		if service == "web" {
			runMutagenSync(app, mutagenSync)
		}

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

		if service == "web" {
			runMutagenSync(app, mutagenSync)
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

func runMutagenSync(app *ddevapp.DdevApp, mutagenSync bool) {
	if mutagenSync && app.IsMutagenEnabled() {
		if status, _ := app.SiteStatus(); status == ddevapp.SiteRunning {
			err := app.MutagenSyncFlush()
			if err != nil {
				util.Warning("Could not flush Mutagen: %v", err)
			}
		}
	}
}
