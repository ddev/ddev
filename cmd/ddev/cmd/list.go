package cmd

import (
	"os"

	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/plugins/platform"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// DevListCmd represents the list command
var DevListCmd = &cobra.Command{
	Use:   "list",
	Short: "List applications that exist locally",
	Long:  `List applications that exist locally.`,
	Run: func(cmd *cobra.Command, args []string) {
		sites := platform.GetApps()
		var appDescs []map[string]interface{}

		if len(sites) < 1 {
			output.UserOut.Println("There are no running ddev applications.")
			os.Exit(0)
		}

		table := platform.CreateAppTable()
		for _, site := range sites {
			desc, err := site.Describe()
			if err != nil {
				util.Failed("Failed to describe site %s: %v", site.GetName(), err)
			}
			appDescs = append(appDescs, desc)
			platform.RenderAppRow(table, desc)
		}

		output.UserOut.WithField("raw", appDescs).Print(table.String() + "\n" + platform.RenderRouterStatus())
	},
}

func init() {
	RootCmd.AddCommand(DevListCmd)
}
