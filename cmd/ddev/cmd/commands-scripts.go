package cmd

import (
	"path/filepath"

	"github.com/spf13/cobra"
)

// commandScript
type commandScript struct {
	// Path and name to the script file
	filename string

	// Local cache vars
	name        string
	servicePath string
	service     string
	cmdTypePath string
	cmdType     string
}

// Internal types to make methods possible.
type (
	cmdUse                        string
	cmdAliases                    []string
	cmdSuggestFor                 []string
	cmdShort                      string
	cmdLong                       string
	cmdExample                    string
	cmdValidArgs                  []string
	cmdArgAliases                 []string
	cmdDeprecated                 string
	cmdHidden                     bool
	cmdAnnotations                map[string]string
	cmdSilenceErrors              bool
	cmdSilenceUsage               bool
	cmdDisableFlagParsing         bool
	cmdDisableAutoGenTag          bool
	cmdDisableFlagsInUseLine      bool
	cmdDisableSuggestions         bool
	cmdSuggestionsMinimumDistance int
)

// commandDefinition represents the definition extracted from the annotations
// of the custom command script. For more information about the fields see also
// github.com/spf13/cobra.
type commandDefinition struct {
	Use                        cmdUse
	Aliases                    cmdAliases
	SuggestFor                 cmdSuggestFor
	Short                      cmdShort
	Long                       cmdLong
	Example                    cmdExample
	ValidArgs                  cmdValidArgs
	ArgAliases                 cmdArgAliases
	Flags                      flagsDefinition
	Deprecated                 cmdDeprecated
	Hidden                     cmdHidden
	Annotations                cmdAnnotations
	SilenceErrors              cmdSilenceErrors
	SilenceUsage               cmdSilenceUsage
	DisableFlagParsing         cmdDisableFlagParsing
	DisableAutoGenTag          cmdDisableAutoGenTag
	DisableFlagsInUseLine      cmdDisableFlagsInUseLine
	DisableSuggestions         cmdDisableSuggestions
	SuggestionsMinimumDistance cmdSuggestionsMinimumDistance
}

func createCustomCommandScript(filename string) *commandScript {
	return &commandScript{filename: filename}
}

func (s *commandScript) Name() string {
	if s.name == "" {
		s.name = filepath.Base(s.filename)
	}

	return s.name
}

func (s *commandScript) ServicePath() string {
	if s.servicePath == "" {
		s.servicePath = filepath.Dir(s.Name)
	}

	return s.servicePath
}

func (s *commandScript) Service() string {
	if s.service == "" {
		s.service = filepath.Base(s.ServicePath())
	}

	return s.service
}

func (s *commandScript) TypePath() string {
	if s.cmdTypePath == "" {
		s.cmdTypePath = filepath.Dir(filepath.Dir(s.Name))
	}

	return s.cmdTypePath
}

func (s *commandScript) Type() string {
	if s.cmdType == "" {
		s.cmdType = filepath.Base(s.TypePath())
	}

	return s.cmdType
}

func (s *commandScript) Ignore() bool {
}

func (s *commandScript) Validate() error {
}

func (s *commandScript) addToCommand(command *cobra.Command, commandsAdded *commandsAdded) error {
	s.Ignore()

	/*
		// Use path.Join() for the inContainerFullPath because it'serviceDirOnHost about the path in the container, not on the
		// host; a Windows path is not useful here.
		inContainerFullPath := path.Join("/mnt/ddev_config", filepath.Base(commandSet), service, commandName)
		onHostFullPath := filepath.Join(commandSet, service, commandName)

		if strings.HasSuffix(commandName, ".example") || strings.HasPrefix(commandName, "README") || strings.HasPrefix(commandName, ".") || fileutil.IsDirectory(onHostFullPath) {
			continue
		}

		// If command has already been added, we won't work with it again.
		if _, found := commandsAdded[commandName]; found {
			util.Warning("not adding command %s (%s) because it was already added to project %s", commandName, onHostFullPath, app.Name)
			continue
		}

		// Any command we find will want to be executable on Linux
		_ = os.Chmod(onHostFullPath, 0755)
		if hasCR, _ := fileutil.FgrepStringInFile(onHostFullPath, "\r\n"); hasCR {
			util.Warning("command '%s' contains CRLF, please convert to Linux-style linefeeds with dos2unix or another tool, skipping %s", commandName, onHostFullPath)
			continue
		}

		directives := findDirectivesInScriptCommand(onHostFullPath)
		var description, usage, example, projectTypes, osTypes, hostBinaryExists string

		description = commandName
		if val, ok := directives["Description"]; ok {
			description = val
		}

		if val, ok := directives["Usage"]; ok {
			usage = val
		}

		if val, ok := directives["Example"]; ok {
			example = "  " + strings.ReplaceAll(val, `\n`, "\n  ")
		}

		// Init and import flags
		var flags Flags
		flags.Init(commandName, onHostFullPath)

		if val, ok := directives["Flags"]; ok {
			if err = flags.LoadFromJSON(val); err != nil {
				util.Warning("Error '%s', command '%s' contains an invalid flags definition '%s', skipping add flags of %s", err, commandName, val, onHostFullPath)
			}
		}

		// Init and import sub commands
		var subCommands SubCommands
		subCommands.Init(commandName, onHostFullPath)

		if val, ok := directives["Commands"]; ok {
			if err = subCommands.LoadFromJSON(val); err != nil {
				util.Warning("Error '%s', command '%s' contains an invalid flags definition '%s', skipping add flags of %s", err, commandName, val, onHostFullPath)
			}
		}

		// Import and handle ProjectTypes
		if val, ok := directives["ProjectTypes"]; ok {
			projectTypes = val
		}

		// If ProjectTypes is specified and we aren't of that type, skip
		if projectTypes != "" && !strings.Contains(projectTypes, app.Type) {
			continue
		}

		// Import and handle OSTypes
		if val, ok := directives["OSTypes"]; ok {
			osTypes = val
		}

		// If OSTypes is specified and we aren't this isn't a specified OS, skip
		if osTypes != "" && !strings.Contains(osTypes, runtime.GOOS) {
			continue
		}

		// Import and handle HostBinaryExists
		if val, ok := directives["HostBinaryExists"]; ok {
			hostBinaryExists = val
		}

		// If hostBinaryExists is specified it doesn't exist here, skip
		if hostBinaryExists != "" && !fileutil.FileExists(hostBinaryExists) {
			continue
		}

		// Create proper description suffix
		descSuffix := " (shell " + service + " container command)"
		if commandSet == targetGlobalCommandPath {
			descSuffix = " (global shell " + service + " container command)"
		}

		// Initialize the new command
		commandToAdd := &cobra.Command{
			Use:     usage,
			Short:   description + descSuffix,
			Example: example,
			FParseErrWhitelist: cobra.FParseErrWhitelist{
				UnknownFlags: true,
			},
		}

		// Add flags to command
		if err = flags.AssignToCommand(commandToAdd); err != nil {
			util.Warning("Error '%s' in the flags definition for command '%s', skipping %s", err, commandName, onHostFullPath)
			continue
		}

		// Add sub commands to command
		if err = flags.AssignToCommand(commandToAdd); err != nil {
			util.Warning("Error '%s' in the flags definition for command '%s', skipping %s", err, commandName, onHostFullPath)
			continue
		}

		if service == "host" {
			commandToAdd.Run = makeHostCmd(app, onHostFullPath, commandName)
		} else {
			commandToAdd.Run = makeContainerCmd(app, inContainerFullPath, commandName, service)
		}

		// Add the command and mark as added
		c.AddCommand(commandToAdd)
		commandsAdded[commandName] = true
	*/
}
