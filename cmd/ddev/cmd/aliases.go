package cmd

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/ddev/ddev/pkg/styles"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

// AliasesCmd implements the command to list all command aliases
var AliasesCmd = &cobra.Command{
	Use:     "aliases",
	Short:   "Shows all aliases for each command in the current context (global or project).",
	Example: `ddev aliases`,
	Args:    cobra.NoArgs,
	Run: func(_ *cobra.Command, _ []string) {
		var out bytes.Buffer
		t := table.NewWriter()
		t.SetOutputMirror(&out)

		// Use simple output
		t.SetStyle(styles.GetTableStyle("default"))

		// Append table header
		t.AppendHeader(table.Row{"Command", "Aliases"})

		// Collect aliases for all commands
		commandAliases := make(map[string][]string)
		collectAliases(RootCmd, commandAliases)

		// Get a sorted list of command paths
		sortedCommands := make([]string, 0, len(commandAliases))
		for cmdPath := range commandAliases {
			sortedCommands = append(sortedCommands, cmdPath)
		}
		sort.Strings(sortedCommands)

		// Append rows to the table in sorted order
		for _, cmdPath := range sortedCommands {
			t.AppendRows([]table.Row{
				{cmdPath, strings.Join(commandAliases[cmdPath], ", ")},
			})
		}

		// Render the table
		t.Render()
		fmt.Println(out.String())
	},
}

// collectAliases collects aliases for all commands in the app
func collectAliases(cmd *cobra.Command, commandAliases map[string][]string) {
	if len(cmd.Aliases) > 0 {
		fullAliases := make([]string, len(cmd.Aliases))
		for i, alias := range cmd.Aliases {
			fullAliases[i] = strings.TrimSpace(buildFullCommandPath(cmd, alias))
		}
		commandAliases[strings.TrimSpace(trimProgramName(cmd.CommandPath()))] = fullAliases
	}
	for _, subCmd := range cmd.Commands() {
		collectAliases(subCmd, commandAliases)
	}
}

// buildFullCommandPath constructs the full command path using the alias, excluding the program name
func buildFullCommandPath(cmd *cobra.Command, alias string) string {
	return trimProgramName(cmd.Parent().CommandPath()) + " " + alias
}

// trimProgramName removes the program name from the command path
func trimProgramName(commandPath string) string {
	parts := strings.SplitN(commandPath, " ", 2)
	if len(parts) > 1 {
		return parts[1]
	}
	return ""
}

func init() {
	RootCmd.AddCommand(AliasesCmd)
}
