package cmd

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/fsouza/go-dockerclient"
	"github.com/gosuri/uitable"
	"github.com/spf13/cobra"
)

// FilterNonDrud takes a list of containers and returns the ones most likely related to DRUD
func FilterNonDrud(vs []docker.APIContainers) []docker.APIContainers {
	var vsf []docker.APIContainers
	for _, v := range vs {
		clientName := strings.Split(v.Names[0][1:], "-")[0]
		if _, err := os.Stat(path.Join(homedir, ".drud", clientName)); os.IsNotExist(err) {
			continue
		}
		vsf = append(vsf, v)
	}
	return vsf
}

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List applications that exist locally",
	Long:  `List applications that exist locally.`,
	Run: func(cmd *cobra.Command, args []string) {

		client, _ := GetDockerClient()

		containers, _ := client.ListContainers(docker.ListContainersOptions{All: true})
		containers = FilterNonDrud(containers)

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
	LocalCmd.AddCommand(listCmd)

}
