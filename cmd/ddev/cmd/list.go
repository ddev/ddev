package cmd

import (
	"time"

	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// continuous, if set, makes list continuously output
var continuous bool

// showAll, if set, shows non-running projects in addition to running/paused
var showAll bool

// DdevListCmd represents the list command
var DdevListCmd = &cobra.Command{
	Use:   "list",
	Short: "List projects",
	Long:  `List projects. Shows active projects by default, includes stopped projects with --all`,
	Run: func(cmd *cobra.Command, args []string) {
		for {
			apps, err := ddevapp.GetProjects(!showAll)
			if err != nil {
				util.Failed("failed getting GetProjects: %v", err)
			}
			appDescs := make([]map[string]interface{}, 0)

			if len(apps) < 1 {
				output.UserOut.WithField("raw", appDescs).Println("No ddev projects were found.")
			} else {
				table := ddevapp.CreateAppTable()
				for _, app := range apps {
					desc, err := app.Describe()
					if err != nil {
						util.Error("Failed to describe project %s: %v", app.GetName(), err)
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
	DdevListCmd.Flags().BoolVarP(&showAll, "all", "a", false, "If set, all projects will be displayed, even stopped projects.")
	DdevListCmd.Flags().BoolVarP(&continuous, "continuous", "", false, "If set, project information will be emitted once per second")
	RootCmd.AddCommand(DdevListCmd)
}
