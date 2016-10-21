package cmd

import (
	"fmt"
	"log"
	"runtime"
	"sync"

	"github.com/drud/bootstrap/cli/local"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var scaffold bool

// LegacyAddCmd represents the add command
var LegacyAddCmd = &cobra.Command{
	Use:   "add [app_name] [environment_name]",
	Short: "Add an existing application to your local development environment",
	Long: `Add an existing application to your local dev environment.  This involves
	downloading of containers, media, and databases.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		netName := "drud_default"

		client, err := local.GetDockerClient()
		if err != nil {
			log.Fatal(err)
		}

		err = EnsureNetwork(client, netName)
		if err != nil {
			log.Fatal(err)
		}

	},
	Run: func(cmd *cobra.Command, args []string) {

		var err error

		// limit logical processors to 3
		runtime.GOMAXPROCS(2)
		// set up wait group

		var wg sync.WaitGroup
		wg.Add(2)

		app := local.LegacyApp{
			Name:        activeApp,
			Environment: activeDeploy,
			Template:    local.LegacyComposeTemplate,
		}

		if !app.DatabagExists() {
			log.Println(err)
			Failed("No legacy site by that name.")
		}

		err = PrepLocalSiteDirs(app.AbsPath())
		if err != nil {
			log.Println(err)
			Failed("Failed to unpack application resources.")
		}

		go func() {
			defer wg.Done()

			fmt.Println("Getting source code.")

			err := local.CloneSource(app)
			if err != nil {
				log.Println(err)
				Failed("Failed to clone the project repository.")
			}
		}()

		go func() {
			defer wg.Done()

			fmt.Println("Getting Resources.")
			err := app.GetResources()
			if err != nil {
				log.Println(err)
				Failed("Failed to retrieve application resources.")
			}
		}()

		wg.Wait()

		err = app.UnpackResources()
		if err != nil {
			log.Println(err)
			Failed("Failed to unpack application resources.")
		}

<<<<<<< HEAD
		err = app.SetType()
		if err != nil {
			log.Println(err)
			Failed("Failed to determine app type.")
		}

		fmt.Println("Creating docker-compose config.")
		err = local.WriteLocalAppYAML(app)
		if err != nil {
			log.Println(err)
			Failed("Failed to create docker-compose.yaml.")
		}

=======
>>>>>>> master
		var siteURL string
		if !scaffold {
			err = app.Start()
			if err != nil {
				log.Println(err)
				Failed("Failed to start application.")
			}

			err = app.Config()
			if err != nil {
				log.Println(err)
				Failed("Failed to configure application.")
			}

			fmt.Println("Waiting for site readiness. This may take a couple minutes...")
			siteURL, err = app.Wait()
			if err != nil {
				log.Println(err)
				Failed("Site never became ready")
			}
		}

		color.Cyan("Successfully added %s-%s", activeApp, activeDeploy)
		if siteURL != "" {
			color.Cyan("Your application can be reached at: %s", siteURL)
		}

	},
}

func init() {
	LegacyAddCmd.Flags().BoolVarP(&scaffold, "scaffold", "s", false, "Add the app but don't run or config it.")
	LegacyCmd.AddCommand(LegacyAddCmd)
}
