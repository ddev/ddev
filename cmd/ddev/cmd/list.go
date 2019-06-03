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

// activeOnly, if set, shows only running projects
var activeOnly bool

// continuousSleepTime is time to sleep between reads with --continuous
var continuousSleepTime = 1

// DdevListCmd represents the list command
var DdevListCmd = &cobra.Command{
	Use:   "list",
	Short: "List projects",
	Long:  `List projects. Shows active projects by default, includes stopped projects with --all`,
	Run: func(cmd *cobra.Command, args []string) {
		for {
			apps, err := ddevapp.GetProjects(activeOnly)
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

			time.Sleep(time.Duration(continuousSleepTime) * time.Second)
		}
	},
}

func init() {
	DdevListCmd.Flags().BoolVarP(&activeOnly, "active-only", "A", false, "If set, only currently active projects will be displayed.")
	DdevListCmd.Flags().BoolVarP(&continuous, "continuous", "", false, "If set, project information will be emitted until the command is stopped.")
	DdevListCmd.Flags().IntVarP(&continuousSleepTime, "continuous-sleep-interval", "I", 1, "Time in seconds between ddev list --continous output lists.")

	RootCmd.AddCommand(DdevListCmd)
}
