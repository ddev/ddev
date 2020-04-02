package cmd

import (
	"github.com/drud/ddev/pkg/fileutil"
	"strings"

	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// DebugComposeConfigCmd implements the ddev debug compose-config command
var DebugComposeConfigCmd = &cobra.Command{
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

		app.DockerEnv()
		if err = app.WriteDockerComposeYAML(); err != nil {
			util.Failed("Failed to get compose-config: %v", err)
		}

		out, err := fileutil.ReadFileIntoString(app.DockerComposeFullRenderedYAMLPath())
		if err != nil {
			util.Failed("unable to read rendered file %s: %v", app.DockerComposeFullRenderedYAMLPath(), err)
		}
		output.UserOut.Print(strings.TrimSpace(out))
	},
}

func init() {
	DebugCmd.AddCommand(DebugComposeConfigCmd)
}
