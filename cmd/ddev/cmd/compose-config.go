package cmd

import (
	"strings"

	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

var ComposeConfigCmd = &cobra.Command{
	Use:   "compose-config [project]",
	Short: "Prints the docker-compose configuration of the current project",
	Run: func(cmd *cobra.Command, args []string) {
		projectName := ""

		if len(args) > 1 {
			util.Failed("This command only takes one optional argument: project name")
		}

		if len(args) == 1 {
			projectName = args[0]
		}

		app, err := ddevapp.GetActiveApp(projectName)
		if err != nil {
			util.Failed("Failed to get compose-config: %v", err)
		}

		if err = app.WriteDockerComposeConfig(); err != nil {
			util.Failed("Failed to get compose-config: %v", err)
		}

		app.DockerEnv()
		files, err := app.ComposeFiles()
		if err != nil {
			util.Failed("Failed to get compose-config: %v", err)
		}

		out, _, err := dockerutil.ComposeCmd(files, "config")
		if err != nil {
			util.Failed("Failed to get compose-config: %v", err)
		}

		output.UserOut.Printf(strings.TrimSpace(out))
	},
}

func init() {
	ComposeConfigCmd.Hidden = true
	RootCmd.AddCommand(ComposeConfigCmd)
}
