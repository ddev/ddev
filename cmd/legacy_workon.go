package cmd

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"

	"github.com/spf13/cobra"
)

// LegacyWorkonCmd represents the list command
var LegacyWorkonCmd = &cobra.Command{
	Use:   "workon",
	Short: "Set a site to work on",
	Long:  `If you select an app to workon you cant skip the acticeApp and activeDeploy args.`,
	Run: func(cmd *cobra.Command, args []string) {
		var parts []string
		var answer int
		var files []os.FileInfo

		if activeApp == "" && activeDeploy == "" {
			fmt.Println("Enter a number to choose which app to work on:")
			files, _ = ioutil.ReadDir(path.Join(homedir, ".drud", "legacy"))
			for i, f := range files {
				if f.Name()[0:1] != "." && strings.Contains(f.Name(), "-") {
					fmt.Printf("%d: %s\n", i, f.Name())
				}
			}

			fmt.Scanf("%d", &answer)
			if answer >= len(files) {
				fmt.Println("You must choose a number listed above.")
				log.Fatal("Number entered was not valid")
			}

			parts = strings.Split(files[answer].Name(), "-")
		} else {
			parts = []string{activeApp, activeDeploy}
		}
		cfg.ActiveApp = parts[0]
		cfg.ActiveDeploy = parts[1]

		err := cfg.WriteConfig(drudconf)
		if err != nil {
			fmt.Println("Could not set config items.")
			log.Fatal(err)
		}

		fmt.Println("You are now working on", files[answer].Name())

	},
	PersistentPreRun: func(cmd *cobra.Command, args []string) {},
}

func init() {
	LegacyCmd.AddCommand(LegacyWorkonCmd)
}
