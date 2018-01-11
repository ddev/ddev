package cmd

import (
	"os"

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
		for label, value := range v {
			versionOutput.AddRow(label, value)
		}

		output.UserOut.WithField("raw", v).Println(versionOutput)
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
}
