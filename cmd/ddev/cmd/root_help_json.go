package cmd

import (
	"github.com/ddev/ddev/pkg/output"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Sets the help function of a command to one which respects the --json-output flag.
func setHelpFunc(command *cobra.Command) {
	originalHelpFunc := command.HelpFunc()
	command.SetHelpFunc(func(command *cobra.Command, strings []string) {
		if !output.JSONOutput {
			// Use the default help function for plain text output
			originalHelpFunc(command, strings)
		} else {
			// LogSetup already happened in init, but that's before flags were parsed.
			// It seems like the --help flag bypasses the second call of LogSetUp after parsing
			// flags, which means the output doesn't want to output JSON.
			output.LogSetUp()

			// JSON representation of a subcommand
			type jsonSubCommand struct {
				Name        string
				Description string
			}

			// JSON representation of a group of commands
			type jsonCommandGroup struct {
				Name     string
				Commands []*jsonSubCommand
			}

			// JSON representation of a CLI flag
			type jsonFlag struct {
				Name                string
				Shorthand           string
				Usage               string
				Default             string
				Deprecated          string
				Hidden              bool
				ShorthandDeprecated string
			}

			var jsonCommands []*jsonSubCommand
			var jsonCommandGroups []*jsonCommandGroup
			var jsonAdditionalCommands []*jsonSubCommand
			var jsonAdditionalHelpCommands []*jsonSubCommand
			var jsonFlags []*jsonFlag
			var jsonGlobalFlags []*jsonFlag

			// Build list of subcommands. Logic reflects the default "usage" template from cobra
			if len(command.Groups()) == 0 {
				// Direct subcommands, if there are no groups
				for _, subCmd := range command.Commands() {
					if subCmd.IsAvailableCommand() || subCmd.Name() == "help" {
						jsonCommands = append(jsonCommands, &jsonSubCommand{
							Name:        subCmd.Name(),
							Description: subCmd.Short,
						})
					}
				}
			} else {
				// Groups of subcommands
				for _, group := range command.Groups() {
					var jsonGroupCommands []*jsonSubCommand
					for _, subCmd := range command.Commands() {
						if subCmd.GroupID == group.ID || (subCmd.IsAvailableCommand() || subCmd.Name() == "help") {
							jsonGroupCommands = append(jsonGroupCommands, &jsonSubCommand{
								Name:        subCmd.Name(),
								Description: subCmd.Short,
							})
						}
					}
					jsonCommandGroups = append(jsonCommandGroups, &jsonCommandGroup{
						Name:     group.Title,
						Commands: jsonGroupCommands,
					})
				}
				// Subcommands that don't belong in a group
				if !command.AllChildCommandsHaveGroup() {
					for _, subCmd := range command.Commands() {
						if subCmd.GroupID == "" && (subCmd.IsAvailableCommand() || subCmd.Name() == "help") {
							jsonAdditionalCommands = append(jsonAdditionalCommands, &jsonSubCommand{
								Name:        subCmd.Name(),
								Description: subCmd.Short,
							})
						}
					}
				}
			}

			// Additional help commands
			for _, subCmd := range command.Commands() {
				if subCmd.IsAdditionalHelpTopicCommand() {
					jsonAdditionalHelpCommands = append(jsonAdditionalHelpCommands, &jsonSubCommand{
						Name:        subCmd.Name(),
						Description: subCmd.Short,
					})
				}
			}

			// Build list of all non-global flags
			command.LocalFlags().VisitAll(func(localFlag *pflag.Flag) {
				jsonFlags = append(jsonFlags, &jsonFlag{
					Name:                localFlag.Name,
					Shorthand:           localFlag.Shorthand,
					Usage:               localFlag.Usage,
					Default:             localFlag.DefValue,
					Deprecated:          localFlag.Deprecated,
					Hidden:              localFlag.Hidden,
					ShorthandDeprecated: localFlag.ShorthandDeprecated,
				})
			})

			// Build list of all global flags
			command.InheritedFlags().VisitAll(func(globalFlag *pflag.Flag) {
				jsonGlobalFlags = append(jsonGlobalFlags, &jsonFlag{
					Name:                globalFlag.Name,
					Shorthand:           globalFlag.Shorthand,
					Usage:               globalFlag.Usage,
					Default:             globalFlag.DefValue,
					Deprecated:          globalFlag.Deprecated,
					Hidden:              globalFlag.Hidden,
					ShorthandDeprecated: globalFlag.ShorthandDeprecated,
				})
			})

			// Build usage, reflecting what is rendered for the plain text output
			var fullUsage = command.UseLine()
			if command.HasAvailableSubCommands() {
				fullUsage = fullUsage + "\n" + command.CommandPath() + " [command]"
			}

			rawResult := make(map[string]interface{})
			rawResult["ShortDescription"] = command.Short
			rawResult["LongDescription"] = command.Long
			rawResult["Usage"] = fullUsage
			rawResult["Aliases"] = command.Aliases
			rawResult["Example"] = command.Example
			rawResult["Commands"] = jsonCommands
			rawResult["CommandGroups"] = jsonCommandGroups
			rawResult["AdditionalCommands"] = jsonAdditionalCommands
			rawResult["AdditionalHelpCommands"] = jsonAdditionalHelpCommands
			rawResult["Flags"] = jsonFlags
			rawResult["GlobalFlags"] = jsonGlobalFlags
			rawResult["Deprecated"] = command.Deprecated
			rawResult["Hidden"] = command.Hidden

			output.UserOut.WithField("raw", rawResult).Print()
		}
	})
}
