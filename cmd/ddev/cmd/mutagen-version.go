package cmd

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/styles"
	"github.com/ddev/ddev/pkg/versionconstants"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

// MutagenVersionCmd implements the ddev mutagen version command
var MutagenVersionCmd = &cobra.Command{
	Use:     "version",
	Short:   "Display the version of the Mutagen binary and the location of its components.",
	Example: `"ddev mutagen version"`,
	Run: func(_ *cobra.Command, _ []string) {

		v := make(map[string]string)
		v["version"] = versionconstants.RequiredMutagenVersion
		v["binary"] = globalconfig.GetMutagenPath()
		v["enabled"] = fmt.Sprintf("%t", globalconfig.DdevGlobalConfig.IsMutagenEnabled())
		v["MUTAGEN_DATA_DIRECTORY"] = globalconfig.GetMutagenDataDirectory()

		var out bytes.Buffer
		t := table.NewWriter()
		t.SetOutputMirror(&out)
		styles.SetGlobalTableStyle(t, true)

		t.AppendHeader(table.Row{"Item", "Value"})

		// Ensure "version" is always the first row
		t.AppendRow(table.Row{"version", v["version"]})

		keys := make([]string, 0, len(v))
		for k := range v {
			if k != "version" {
				keys = append(keys, k)
			}
		}
		sort.Strings(keys)

		for _, label := range keys {
			t.AppendRow(table.Row{
				label, v[label],
			})
		}
		t.Render()
		output.UserOut.WithField("raw", v).Println(out.String())
	},
}

func init() {
	MutagenCmd.AddCommand(MutagenVersionCmd)
}
