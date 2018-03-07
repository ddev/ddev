package cmd

import (
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
	Short: "List projects",
	Long:  `List projects.`,
	Run: func(cmd *cobra.Command, args []string) {
		for {
			apps := ddevapp.GetApps()
			var appDescs []map[string]interface{}

			if len(apps) < 1 {
				output.UserOut.Println("There are no running ddev projects.")
			} else {
				table := ddevapp.CreateAppTable()
				for _, app := range apps {
					desc, err := app.Describe()
					if err != nil {
						util.Failed("Failed to describe project %s: %v", app.GetName(), err)
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
	DevListCmd.Flags().BoolVarP(&continuous, "continuous", "", false, "If set, project information will be emitted once per second")
	RootCmd.AddCommand(DevListCmd)
}
