package cmd

import (
	"github.com/drud/bootstrap/cli/local"
	"github.com/drud/bootstrap/cli/utils"
	"github.com/fsouza/go-dockerclient"
	"github.com/spf13/cobra"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List applications that exist locally",
	Long:  `List applications that exist locally.`,
	Run: func(cmd *cobra.Command, args []string) {

		client, _ := utils.GetDockerClient()

		containers, _ := client.ListContainers(docker.ListContainersOptions{All: true})
		containers = local.FilterNonDrud(containers)

		local.RenderContainerList(containers)
	},
}

func init() {
	LocalCmd.AddCommand(listCmd)

}
