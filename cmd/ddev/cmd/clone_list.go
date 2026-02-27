package cmd

import (
	"bytes"
	"fmt"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/styles"
	"github.com/ddev/ddev/pkg/util"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

// CloneListCmd implements the "ddev clone list" command.
var CloneListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all clones of the current project",
	Long: `List all clones associated with the current project.

Can be run from the source project or any of its clones.`,
	Example: `ddev clone list`,
	Run: func(_ *cobra.Command, _ []string) {
		app, err := ddevapp.GetActiveApp("")
		if err != nil {
			util.Failed("Unable to get active project: %v", err)
		}

		clones, err := ddevapp.CloneList(app)
		if err != nil {
			util.Failed("Failed to list clones: %v", err)
		}

		sourceProjectName := ddevapp.GetSourceProjectNameExported(app)

		if len(clones) == 0 {
			output.UserOut.Printf("No clones found for project '%s'.", sourceProjectName)
			output.UserOut.Printf("Create one with: ddev clone create <clone-name>")
			return
		}

		var out bytes.Buffer
		t := table.NewWriter()
		t.SetOutputMirror(&out)
		styles.SetGlobalTableStyle(t, false)

		columns := table.Row{"Clone Name", "Path", "Branch", "Status"}
		if !globalconfig.DdevGlobalConfig.SimpleFormatting {
			var colConfig []table.ColumnConfig
			for _, col := range columns {
				colConfig = append(colConfig, table.ColumnConfig{
					Name: fmt.Sprint(col),
				})
			}
			t.SetColumnConfigs(colConfig)
		}
		t.AppendHeader(columns)

		for _, clone := range clones {
			displayName := clone.CloneName
			if clone.Current {
				displayName = "* " + displayName
			}
			t.AppendRow(table.Row{displayName, clone.WorktreePath, clone.Branch, clone.Status})
		}

		t.Render()
		output.UserOut.WithField("raw", clones).Println(out.String())
	},
}

func init() {
	CloneCmd.AddCommand(CloneListCmd)
}
