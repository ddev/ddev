package cmd

import (
	"fmt"
	"log"
	"path"

	"github.com/drud/ddev/pkg/plugins/platform"
	"github.com/drud/drud-go/utils/dockerutil"
	"github.com/spf13/cobra"
)

// ImportCmd represents the add command
var ImportCmd = &cobra.Command{
	Use:   "import",
	Short: "Import an existing site to the local dev environment",
	Long:  `Import the database and file assets of an existing site into the local development environment.`,
	PreRun: func(cmd *cobra.Command, args []string) {

		client, err := platform.GetDockerClient()
		if err != nil {
			log.Fatal(err)
		}

		err = EnsureNetwork(client, netName)
		if err != nil {
			log.Fatal(err)
		}

	},
	Run: func(cmd *cobra.Command, args []string) {
		app, err := getActiveApp()
		if err != nil {
			log.Fatalf("Could not find an active ddev configuration, have you ran 'ddev config'?: %v", err)
		}

		err = app.GetResources()
		if err != nil {
			Failed("Failed to gather resources for %s: %s", app.GetName(), err)
		}

		err = app.UnpackResources()
		if err != nil {
			Failed("Failed to unpack resources for %s: %s", app.GetName(), err)
		}

		err = app.Config()
		if err != nil {
			Failed("Failed to configure %s: %s.", app.GetName(), err)
		}

		nameContainer := fmt.Sprintf("%s-db", app.ContainerName())
		if !dockerutil.IsRunning(nameContainer) || !platform.ComposeFileExists(app) {
			Failed("This application is not currently running. Run `ddev start` to start the environment.")
		}

		cmdArgs := []string{
			"-f", path.Join(app.AbsPath(), ".ddev", "docker-compose.yaml"),
			"exec",
			"-T", nameContainer,
			"./import.sh",
		}

		err = dockerutil.DockerCompose(cmdArgs...)
		if err != nil {
			Failed("Could not execute command: %s", err)
		}

		siteURL, err := app.Wait()
		if err != nil {
			Failed("%s did not return a 200 status before timeout. %s", app.GetName(), err)
		}

		Success("Successfully imported %s", app.GetName())
		if siteURL != "" {
			Success("Your application can be reached at: %s", siteURL)
		}

	},
}

func init() {
	RootCmd.AddCommand(ImportCmd)
}
