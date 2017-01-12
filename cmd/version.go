package cmd

import (
	"fmt"

	"github.com/drud/bootstrap/cli/version"
	"github.com/gosuri/uitable"
	"github.com/spf13/cobra"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "print DRUD version",
	Long:  `Display the version of this DRUD binary.`,
	Run: func(cmd *cobra.Command, args []string) {
		table := uitable.New()
		table.MaxColWidth = 200

		table.AddRow("cli:", version.VERSION)
		table.AddRow("nginx:", version.NGINX)
		table.AddRow("mysql:", version.MYSQL)

		fmt.Println(table)
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
}
