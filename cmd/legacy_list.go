package cmd

import (
	"fmt"
	"strings"

	"github.com/drud/bootstrap/cli/local"
	"github.com/fsouza/go-dockerclient"
	"github.com/gosuri/uitable"
	"github.com/spf13/cobra"
)

// listCmd represents the list command
var LegacyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List applications that exist locally",
	Long:  `List applications that exist locally.`,
	Run: func(cmd *cobra.Command, args []string) {

		client, _ := GetDockerClient()

		containers, _ := client.ListContainers(docker.ListContainersOptions{All: true})
		containers = local.FilterNonLegacy(containers)

		fmt.Printf("%v %v found.\n", len(containers), FormatPlural(len(containers), "container", "containers"))

		table := uitable.New()
		table.MaxColWidth = 200
		table.AddRow("NAME", "IMAGE", "STATUS", "MISC")

		for _, container := range containers {

			var miscField string
			for _, port := range container.Ports {
				if port.PublicPort != 0 {
					miscField = fmt.Sprintf("port: %d", port.PublicPort)
				}
			}

			table.AddRow(
				strings.Join(container.Names, ", "),
				container.Image,
				fmt.Sprintf("%s - %s", container.State, container.Status),
				miscField,
			)
		}
		fmt.Println(table)
	},
}

func init() {
	LegacyCmd.AddCommand(LegacyListCmd)

}
