package cmd

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/go-github/github"
	"github.com/inconshreveable/go-update"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
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

func writeUpdateTime() error {
	timeFile, err := os.Create(filepath.Join(homedir, updateFile))
	if err != nil {
		return err
	}
	defer timeFile.Close()

	timeFile.WriteString(time.Now().Format(time.RFC3339))

	return nil
}

func isUpdateTime() (isTime bool, err error) {

	// if updateFile does not exist then create it with an old timestamp
	if _, err = os.Stat(filepath.Join(homedir, updateFile)); os.IsNotExist(err) {
		f, ferr := os.Create(filepath.Join(homedir, updateFile))
		if ferr != nil {
			log.Fatal(ferr)
		}
		defer f.Close()

		f.WriteString(`2016-05-20T08:35:46-06:00`)

	}

	fileBytes, err := ioutil.ReadFile(filepath.Join(homedir, updateFile))
	if err != nil {
		return
	}

	timeObj, err := time.Parse(time.RFC3339, strings.TrimSpace(string(fileBytes)))
	if err != nil {
		return
	}

	hoursSinceUpdate := time.Now().Sub(timeObj).Hours()

	if hoursSinceUpdate >= 24.0 {
		isTime = true
	}
	return
}

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update DRUD cli tool",
	Long:  `Update DRUD tool to latest release.`,
	Run: func(cmd *cobra.Command, args []string) {

		if isDev {
			fmt.Println("Developer mode enabled. Skipping updates.")
		} else {
			fmt.Println("update called")
			ts := oauth2.StaticTokenSource(
				&oauth2.Token{AccessToken: os.Getenv("GITHUB_TOKEN")},
			)
			tc := oauth2.NewClient(oauth2.NoContext, ts)

			client := github.NewClient(tc)

			rel, _, err := client.Repositories.GetLatestRelease("drud", "drud-bin")
			if err != nil {
				log.Fatal(err)
			}

			var assetID int
			for _, v := range rel.Assets {
				if *v.Name == "drud" {
					assetID = *v.ID
				}
			}

			rc, redirectURL, err := client.Repositories.DownloadReleaseAsset("drud", "drud-bin", assetID)
			if err != nil {
				log.Fatal(err)
			}

			// rc should be nil and redirectURL should not...but according ot docs
			// the reverse could be true
			if rc != nil {
				log.Fatalln("Something went wrong with update.")
			}

			err = doUpdate(redirectURL)
			if err != nil {
				log.Fatal(err)
			}

			err = writeUpdateTime()
			if err != nil {
				log.Fatal(err)
			}
		}
	},
}

func init() {
	RootCmd.AddCommand(updateCmd)
}
