package cmd

import (
	//"os"
	"time"

	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

var continuous bool

// DevListCmd represents the list command
var DevListCmd = &cobra.Command{
	Use:   "list",
	Short: "List applications",
	Long:  `List applications.`,
	Run: func(cmd *cobra.Command, args []string) {
		apps := ddevapp.GetApps()
		var appDescs []map[string]interface{}

		for {

			if len(apps) < 1 {
				output.UserOut.Println("There are no running ddev applications.")
			} else {
				table := ddevapp.CreateAppTable()
				for _, app := range apps {
					desc, err := app.Describe()
					if err != nil {
						util.Failed("Failed to describe site %s: %v", app.GetName(), err)
					}
					appDescs = append(appDescs, desc)
					ddevapp.RenderAppRow(table, desc)
				}
				output.UserOut.WithField("raw", appDescs).Print(table.String() + "\n" + ddevapp.RenderRouterStatus())
			}

			if !continuous {
				break
			}

			time.Sleep(time.Second)
		}

	},
}

func init() {
	DevListCmd.Flags().BoolVarP(&continuous, "continuous", "", false, "If set, site information will be emitted once per second")
	RootCmd.AddCommand(DevListCmd)
}
