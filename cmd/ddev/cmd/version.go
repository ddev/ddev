package cmd

import (
	"os"
	"sort"

	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/util"
	"github.com/drud/ddev/pkg/version"
	"github.com/gosuri/uitable"
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

		versionOutput := uitable.New()

		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, label := range keys {
			if label != "build info" {
				versionOutput.AddRow(label, v[label])
			}
		}

		output.UserOut.WithField("raw", v).Println(versionOutput)
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
}
