package cmd

import (
	"encoding/json"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// The array structure for the flags, the json from the annotation is
// unmarshaled into this structure. For more information see also
// github.com/spf13/pflag/flag
type FlagsDefinition []struct {
	Name        string              // name as it appears on command line
	Shorthand   string              // one-letter abbreviated flag
	Usage       string              // help message
	Type        string              // type, defaults to bool
	DefValue    string              // default value (as text); for usage message
	NoOptDefVal string              // default value (as text); if the flag is on the command line without any options
	Annotations map[string][]string // used by cobra.Command bash autocomple code
}

type Flags struct {
	Definition FlagsDefinition
}

// Converts the defs provided by the custom command as json into the flags
// structure.
func (f *Flags) assign(defs string) error {
	if err := json.Unmarshal([]byte(defs), &f.Definition); err != nil {
		return err
	}

	return nil
}

// Iterates the flags, makes a simple verification and assigns it to the
// provided command.
func (f *Flags) addToCommand(command *cobra.Command, onHostFullPath, commandName string) error {
	for _, flag := range f.Definition {
		// Check usage is defined
		if flag.Usage == "" {
			util.Warning("No usage defined for flag '%s' of command '%s', skipping add flag defined in %s", flag.Name, commandName, onHostFullPath)
			continue
		}

		// Check flag does not already exist
		if command.Flags().Lookup(flag.Name) != nil {
			util.Warning("Flag '%s' already defined for command '%s', skipping add flag defined in %s", flag.Name, commandName, onHostFullPath)
			continue
		}

		// Check shorthand is one letter only
		if len(flag.Shorthand) > 1 {
			util.Warning("Shorthand '%s' with more than one ASCII character defined for command '%s', skipping add flag defined in %s", flag.Shorthand, commandName, onHostFullPath)
			continue
		}

		// Check shorthand does not already exist
		if command.Flags().ShorthandLookup(flag.Shorthand) != nil {
			util.Warning("Shorthand '%s' already defined for command '%s', skipping add flag defined in %s", flag.Shorthand, commandName, onHostFullPath)
			continue
		}

		// Add flag to command
		command.Flags().BoolP(flag.Name, flag.Shorthand, false, flag.Usage)
	}

	return nil
}
