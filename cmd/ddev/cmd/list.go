package cmd

import (
	"time"

	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

var continuous bool

// DdevListCmd represents the list command
var DdevListCmd = &cobra.Command{
	Use:   "list",
	Short: "List projects",
	Long:  `List projects.`,
	Run: func(cmd *cobra.Command, args []string) {
		for {
			apps := ddevapp.GetApps()
			appDescs := make([]map[string]interface{}, 0)

			if len(apps) < 1 {
				output.UserOut.WithField("raw", appDescs).Println("There are no active ddev projects.")
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
	DdevListCmd.Flags().BoolVarP(&continuous, "continuous", "", false, "If set, project information will be emitted once per second")
	RootCmd.AddCommand(DdevListCmd)
}
