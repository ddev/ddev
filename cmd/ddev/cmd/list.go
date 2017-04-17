package cmd

import (
	"strings"

	"github.com/drud/ddev/pkg/plugins/platform"
	"github.com/drud/ddev/pkg/util"
	"github.com/drud/drud-go/utils/dockerutil"
	"github.com/fsouza/go-dockerclient"
	"github.com/spf13/cobra"
)

// DevListCmd represents the list command
var DevListCmd = &cobra.Command{
	Use:   "list",
	Short: "List applications that exist locally",
	Long:  `List applications that exist locally.`,
	Run: func(cmd *cobra.Command, args []string) {

		var prefixes []string

		for _, instance := range platform.PluginMap {
			prefixes = append(prefixes, instance.ContainerPrefix())
		}

		client, _ := dockerutil.GetDockerClient()
		containers, _ := client.ListContainers(docker.ListContainersOptions{All: true})

		containers = func(containers []docker.APIContainers) []docker.APIContainers {
			var vsf []docker.APIContainers
			for _, c := range containers {
				for _, p := range prefixes {
					container := c.Names[0][1:]
					if !strings.HasPrefix(container, p) {
						continue
					}
					vsf = append(vsf, c)
				}
			}
			return vsf
		}(containers)

		err := platform.SiteList(containers)
		util.CheckErr(err)
	},
}

func init() {
	RootCmd.AddCommand(DevListCmd)
}
