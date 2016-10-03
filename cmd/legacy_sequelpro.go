package cmd

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"strconv"

	"github.com/drud/bootstrap/cli/local"
	"github.com/drud/bootstrap/cli/utils"
	"github.com/spf13/cobra"
)

// LegacySequelproCmd represents the sequelpro command
var LegacySequelproCmd = &cobra.Command{
	Use:   "sequelpro",
	Short: "Easily connect local site to sequelpro",
	Long:  `A helper command for easily using sequelpro with a drud app that has been initialized locally.`,
	Run: func(cmd *cobra.Command, args []string) {
		if activeApp == "" {
			log.Fatalln("Must set app flag to dentoe which app you want to work with.")
		}

		app := local.LegacyApp{
			Name:        activeApp,
			Environment: activeDeploy,
		}

		nameContainer := fmt.Sprintf("%s-db", app.ContainerName())

		if !utils.IsRunning(nameContainer) {
			log.Fatal("App not running locally. Try `drud legacy add`.")
		}

		mysqlContainer, err := utils.GetContainer(nameContainer)
		if err != nil {
			log.Fatal(err)
		}

		dbPort, err := utils.GetDockerPublicPort(mysqlContainer, int64(3306))
		if err != nil {
			log.Fatal(err)
		}

		tmpFilePath := path.Join(app.AbsPath(), "sequelpro.spf")
		tmpFile, err := os.Create(tmpFilePath)
		if err != nil {
			log.Fatalln(err)
		}
		defer tmpFile.Close()

		tmpFile.WriteString(fmt.Sprintf(
			local.SequelproTemplate,
			"data",                  //dbname
			"127.0.0.1",             //host
			mysqlContainer.Names[0], //container name
			"root",                  // dbpass
			strconv.FormatInt(dbPort, 10), // port
			"root", //dbuser
		))

		if scaffold != true {
			exec.Command("open", tmpFilePath).Run()
		}

		fmt.Println("sequelpro command finished successfully!")

	},
}

func init() {
	LegacySequelproCmd.Flags().BoolVarP(&scaffold, "scaffold", "s", false, "Add the app but don't run or config it.")
	LegacyCmd.AddCommand(LegacySequelproCmd)
	//RootCmd.AddCommand(SequelproCmd)
}
