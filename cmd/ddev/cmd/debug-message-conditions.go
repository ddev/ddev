package cmd

import (
	"github.com/ddev/ddev/pkg/config/remoteconfig"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/styles"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

// DebugMessageConditionsCmd implements the ddev debug message-conditions command
var DebugMessageConditionsCmd = &cobra.Command{
	Use:   "message-conditions",
	Short: "Show message conditions of this version of ddev",
	Run: func(cmd *cobra.Command, args []string) {
		conditions := remoteconfig.ListConditions()

		t := table.NewWriter()
		styles.SetGlobalTableStyle(t)

		t.AppendHeader(table.Row{"Condition", "Description"})

		for condition, description := range conditions {
			t.AppendRow(table.Row{condition, description})
		}

		t.SortBy([]table.SortBy{
			{Name: "Condition", Mode: table.Asc},
		})

		output.UserOut.WithField("raw", conditions).Print("Supported conditions for messages are:\n\n", t.Render())
	},
	Hidden: true,
}

func init() {
	DebugCmd.AddCommand(DebugMessageConditionsCmd)
}
