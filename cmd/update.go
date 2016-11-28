package cmd

import (
	"fmt"
	"net/http"

	"github.com/fatih/color"
	"github.com/inconshreveable/go-update"
	"github.com/spf13/cobra"
)

func doUpdate(url string) error {
	// request the new file
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	err = update.Apply(resp.Body, update.Options{})
	if err != nil {
		fmt.Println(err)
		if rerr := update.RollbackError(err); rerr != nil {
			fmt.Printf("Failed to rollback from bad update: %v\n", rerr)
		}
	}
	return err
}

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update DRUD cli tool",
	Long:  `Update DRUD tool to latest release.`,
	Run: func(cmd *cobra.Command, args []string) {

		color.Yellow("Updating DRUD binary ...")

		err := doUpdate("https://storage.googleapis.com/drud/drud")
		if err != nil {
			Failed("Could not update DRUD: %s", err)
		} else {
			Success("You are now running the latest version of DRUD.")
		}
	},
}

func init() {
	RootCmd.AddCommand(updateCmd)
}
