package cmd

import (
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

var (
	tail      string
	follow    bool
	timestamp bool
)

// DdevLogsCmd contains the "ddev logs" command
var DdevLogsCmd = &cobra.Command{
	Use:   "logs [projectname]",
	Short: "Get the logs from your running services.",
	Long:  `Uses 'docker logs' to display stdout from the running services.`,
	Example: `ddev logs
ddev logs -f
ddev logs -s db
ddev logs -s db [projectname]`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 1 {
			util.Failed("Too many arguments provided. Please use 'ddev logs' or 'ddev logs [projectname]'")
		}

		projects, err := getRequestedProjects(args, false)
		if err != nil {
			util.Failed("GetRequestedProjects() failed:  %v", err)
		}
		project := projects[0]

		err = project.Logs(serviceType, follow, timestamp, tail)
		if err != nil {
			util.Failed("Failed to retrieve logs for %s: %v", project.GetName(), err)
		}
	},
}

func init() {
	DdevLogsCmd.Flags().BoolVarP(&follow, "follow", "f", false, "Follow the logs in real time.")
	DdevLogsCmd.Flags().BoolVarP(&timestamp, "time", "t", false, "Add timestamps to logs")
	DdevLogsCmd.Flags().StringVarP(&serviceType, "service", "s", "web", "Defines the service to retrieve logs from. [e.g. web, db]")
	DdevLogsCmd.Flags().StringVarP(&tail, "tail", "", "", "How many lines to show")
	RootCmd.AddCommand(DdevLogsCmd)

}
