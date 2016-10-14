package cmd

import (
	"fmt"
	"log"
	"runtime"
	"sync"

	"github.com/drud/bootstrap/cli/local"
	"github.com/spf13/cobra"
)

var appType string
var scaffold bool

// LegacyAddCmd represents the add command
var LegacyAddCmd = &cobra.Command{
	Use:   "add [app_name] [environment_name]",
	Short: "Add an existing application to your local development environment",
	Long: `Add an existing application to your local dev environment.  This involves
	downloading of containers, media, and databases.`,
	Run: func(cmd *cobra.Command, args []string) {

		var err error

		// limit logical processors to 3
		runtime.GOMAXPROCS(3)
		// set up wait group

		var wg sync.WaitGroup
		wg.Add(3)

		app := local.LegacyApp{
			Name:        activeApp,
			Environment: activeDeploy,
			AppType:     appType,
			Template:    local.LegacyComposeTemplate,
		}

		if !app.DatabagExists() {
			log.Fatalln("No legacy site by that name.")
		}

		go func() {
			defer wg.Done()

			fmt.Println("Creating docker-compose config.")
			err := local.WriteLocalAppYAML(app)
			if err != nil {
				log.Fatalln(err)
			}
		}()

		go func() {
			defer wg.Done()

			fmt.Println("Getting source code.")
			err := local.CloneSource(app)
			if err != nil {
				log.Fatalln(err)
			}
		}()

		go func() {
			defer wg.Done()

			fmt.Println("Getting Resources.")
			err := app.GetResources()
			if err != nil {
				log.Fatalln(err)
			}
		}()

		wg.Wait()

		err = app.UnpackResources()
		if err != nil {
			log.Fatalln(err)
		}

		var siteURL string
		if !scaffold {
			err = app.Start()
			if err != nil {
				log.Fatalln(err)
			}

			err = app.Config()
			if err != nil {
				log.Fatalln(err)
			}

			fmt.Println("Waiting for site readiness. This may take a couple minutes...")
			siteURL, err = app.Wait()
			if err != nil {
				log.Fatalln(err)
			}
		}

		fmt.Println("Successfully added", activeApp, activeDeploy)
		if siteURL != "" {
			fmt.Println("Your application can be reached at:", siteURL)
		}

	},
}

func init() {
	LegacyAddCmd.Flags().StringVarP(&appType, "type", "t", "drupal", "Type of application ('drupal' or 'wp')")
	LegacyAddCmd.Flags().BoolVarP(&scaffold, "scaffold", "s", false, "Add the app but don't run or config it.")
	LegacyCmd.AddCommand(LegacyAddCmd)
}
