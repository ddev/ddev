package cmd

import (
	"bytes"
	"github.com/drud/ddev/pkg/styles"
	"os"
	"sort"

	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/util"
	"github.com/drud/ddev/pkg/version"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "print ddev version and component versions",
	Long:  `Display the version of this ddev binary and its components.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 0 {
			err := cmd.Usage()
			util.CheckErr(err)
			os.Exit(1)
		}

		v := version.GetVersionInfo()

		var out bytes.Buffer
		t := table.NewWriter()
		t.SetOutputMirror(&out)
		styles.SetGlobalTableStyle(t)
		t.AppendHeader(table.Row{"Item", "Value"})
		t.SetColumnConfigs([]table.ColumnConfig{{
			Name:     "Value",
			WidthMax: 70,
		},
		})

		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, label := range keys {
			if label != "build info" {
				t.AppendRow(table.Row{
					label, v[label],
				})
			}
		}
		t.Render()
		output.UserOut.WithField("raw", v).Println(out.String())
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
}
