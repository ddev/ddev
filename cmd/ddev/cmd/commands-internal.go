package cmd

import (
	"bytes"
	"io/ioutil"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

// addInternalCommands adds internal hidden commands e.g. used for
// documentation and debug reasons.
func addInternalCommands(command *cobra.Command) error {
	command.AddCommand(&cobra.Command{
		Use:    "create-example-custom-command",
		Short:  "Creates an example custom command",
		Hidden: true,
		Run: func(cmd *cobra.Command, args []string) {
			// Create an example custom command definition
			demoCmdDef := createDemoCommandDefinition()

			var yamlDemoCmdDef []byte
			if yamlDemoCmdDef, err := yaml.Marshal(demoCmdDef); err != nil {
				return
			}

			yamlDemoCmdDef = bytes.TrimRight(bytes.ReplaceAll(yamlDemoCmdDef, []byte("\n"), []byte("\n### ")), "### ")

			// Create an example custom command script
			var scriptDemoCmd []byte
			scriptDemoCmd = append(scriptDemoCmd, []byte("#!/bin/bash\n\n"))
			scriptDemoCmd = append(scriptDemoCmd, yamlDemoCmdDef)
			scriptDemoCmd = append(scriptDemoCmd, []byte("\n\necho \"$@\""))

			err = ioutil.WriteFile(appcopy.ConfigPath, scriptDemoCmd, 0644)
			if err != nil {
				return err
			}
		},
	})
}

func createDemoCommandDefinition() *commandDefinition {
	return &commandDefinition{
		Use:        "Use",
		Aliases:    []string{"Alias 1", "Alias 2", "Alias 3"},
		SuggestFor: []string{"SuggestFor 1", "SuggestFor 2", "SuggestFor 3"},
		Short:      "Short",
		Long:       "Long",
		Example:    "Example",
		ValidArgs:  []string{"ValidArgs 1", "ValidArgs 2", "ValidArgs 3"},
		ArgAliases: []string{"ArgAliases 1", "ArgAliases 2", "ArgAliases 3"},
		Flags: FlagsDefinition{
			Flag{
				Name:        "Name",
				Shorthand:   "Shorthand",
				Usage:       "Usage",
				Type:        "Type",
				DefValue:    "DefValue",
				NoOptDefVal: "NoOptDefVal",
				Annotations: annotationsValue{
					cobra.BashCompFilenameExt:     []string{"json", "yaml", "yml"},
					cobra.BashCompFilenameExt:     []string{},
					cobra.BashCompCustom:          []string{"handler1", "handler2", "handler3"},
					cobra.BashCompCustom:          []string{},
					cobra.BashCompOneRequiredFlag: []string{},
					cobra.BashCompSubdirsInDir:    []string{"dir"},
					cobra.BashCompSubdirsInDir:    []string{},
				},
			},
		},
		Deprecated: "Deprecated",
		Hidden:     false,
		Annotations: map[string]string{
			cobra.BashCompFilenameExt:  []string{"json", "yaml", "yml"},
			cobra.BashCompFilenameExt:  []string{},
			cobra.BashCompCustom:       []string{"handler1", "handler2", "handler3"},
			cobra.BashCompCustom:       []string{},
			cobra.BashCompSubdirsInDir: []string{"dir"},
			cobra.BashCompSubdirsInDir: []string{},
		},
		/*
			SilenceErrors bool
			SilenceUsage bool
			DisableFlagParsing bool
			DisableAutoGenTag bool
			DisableFlagsInUseLine bool
			DisableSuggestions bool
			SuggestionsMinimumDistance int
		*/
	}
}
