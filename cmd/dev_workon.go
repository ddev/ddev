package cmd

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"

	"github.com/drud/bootstrap/cli/local"
	"github.com/spf13/cobra"
)

// LegacyWorkonCmd represents the list command
var LocalDevWorkonCmd = &cobra.Command{
	Use:   "workon",
	Short: "Set a site to work on",
	Long:  `If you select an app to workon you cant skip the activeApp and activeDeploy args.`,
	Run: func(cmd *cobra.Command, args []string) {
		var parts []string
		var answer int
		var files []os.FileInfo

		if activeApp == "" && activeDeploy == "" {
			fmt.Println("Enter a number to choose which app to work on:")
			files, _ = ioutil.ReadDir(path.Join(homedir, ".drud", plugin))
			files = local.FilterNonAppDirs(files)

			fmt.Printf("%d: %s\n", 0, "Cancel")
			for i, f := range files {
				name := f.Name()
				c := i + 1
				fmt.Printf("%d: %s\n", c, name)
			}

			fmt.Scanf("%d", &answer)
			if answer == 0 {
				os.Exit(0)
			}
			if answer >= len(files)+1 {
				Failed("You must choose one of the numbers listed above.")
			}
			parts = strings.Split(files[answer-1].Name(), "-")
		} else {
			parts = []string{activeApp, activeDeploy}
		}
		fmt.Println(parts)
		cfg.ActiveApp = parts[0]
		cfg.ActiveDeploy = parts[1]

		err := cfg.WriteConfig(drudconf)
		if err != nil {
			fmt.Println("Could not set config items.")
			log.Fatal(err)
		}

		fmt.Println("You are now working on", strings.Join(parts, "-"))

	},
	PersistentPreRun: func(cmd *cobra.Command, args []string) {},
}

func init() {
	LocalDevCmd.AddCommand(LocalDevWorkonCmd)
}
