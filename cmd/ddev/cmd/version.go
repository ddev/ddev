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
		out := handleVersionCommand()
		output.UserOut.Println(out)
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
}

// handleVersionCommand does the testable work of the version command.
func handleVersionCommand() *uitable.Table {
	table := uitable.New()
	table.MaxColWidth = 200

	table.AddRow("cli:", version.DdevVersion)
	table.AddRow("web:", version.WebImg+":"+version.WebTag)
	table.AddRow("db:", version.DBImg+":"+version.DBTag)
	table.AddRow("dba:", version.DBAImg+":"+version.DBATag)
	table.AddRow("router:", version.RouterImage+":"+version.RouterTag)
	table.AddRow("commit:", version.COMMIT)
	table.AddRow("domain:", version.DDevTLD)
	table.AddRow("build info:", version.BUILDINFO)

	return table
}
