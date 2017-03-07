package cmd

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"

	"github.com/drud/ddev/pkg/plugins/platform"
	"github.com/spf13/cobra"
)

// LocalDevWorkonCmd represents the list command
var LocalDevWorkonCmd = &cobra.Command{
	Use:   "workon",
	Short: "Set a site to work on",
	Long:  `If you select an app to workon you cant skip the activeApp and activeDeploy args.`,
	Run: func(cmd *cobra.Command, args []string) {
		var name string
		var env string
		var app string
		var answer int
		var files []os.FileInfo

		if activeApp == "" && activeDeploy == "" {
			fmt.Println("Enter a number to choose which app to work on:")
			files, _ = ioutil.ReadDir(path.Join(cfg.Workspace, plugin))
			files = platform.FilterNonAppDirs(files)

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
			name = files[answer-1].Name()
			parts := strings.Split(name, "-")
			env = parts[len(parts)-1]
			app = strings.TrimSuffix(name, "-"+env)
		} else {
			name = activeApp + "-" + activeDeploy
			env = activeDeploy
			app = activeApp
		}
		fmt.Println(name)
		cfg.ActiveApp = app
		cfg.ActiveDeploy = env

		err := cfg.WriteConfig(cfgFile)
		if err != nil {
			fmt.Println("Could not set config items.")
			log.Fatal(err)
		}

		fmt.Println("You are now working on", name)

	},
	PersistentPreRun: func(cmd *cobra.Command, args []string) {},
}

func init() {
	RootCmd.AddCommand(LocalDevWorkonCmd)
}
