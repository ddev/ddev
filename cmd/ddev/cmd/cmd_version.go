package cmd

import (
	"bytes"
	"os"
	"sort"

	"github.com/ddev/ddev/pkg/amplitude"
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/styles"

	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/ddev/ddev/pkg/version"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print DDEV version and component versions",
	Long:  `Display the version of this DDEV binary and its components.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 0 {
			err := cmd.Usage()
			util.CheckErr(err)
			os.Exit(1)
		}

		_, err := dockerutil.DownloadDockerComposeIfNeeded()
		if err != nil {
			util.Failed("Failed to check for and download docker-compose: %v", err)
		}

		v := version.GetVersionInfo()

		if cmd.Flags().Changed("match-constraint") {
			versionConstraint := cmd.Flag("match-constraint").Value.String()
			err := ddevapp.CheckDdevVersionConstraint(versionConstraint, "", "")
			if err != nil {
				util.Failed(err.Error())
			}
		}

		var out bytes.Buffer
		t := table.NewWriter()
		t.SetOutputMirror(&out)

		// Use simplest possible output
		s := styles.GetTableStyle("default")
		s.Options.SeparateRows = false
		s.Options.SeparateFooter = false
		s.Options.SeparateColumns = false
		s.Options.SeparateHeader = false
		s.Options.DrawBorder = false
		t.SetStyle(s)

		t.AppendHeader(table.Row{"Item", "Value"})

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
		amplitude.CheckSetUp()
	},
}

func init() {
	versionCmd.Flags().String("match-constraint", "", `Check if the currently installed ddev matches the specified version constraint.`)
	RootCmd.AddCommand(versionCmd)
}
