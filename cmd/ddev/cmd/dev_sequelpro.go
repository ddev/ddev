package cmd

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"

	"github.com/drud/ddev/pkg/plugins/platform"
	"github.com/drud/drud-go/utils/dockerutil"
	"github.com/spf13/cobra"
)

// LocalDevSequelproCmd represents the sequelpro command
var LocalDevSequelproCmd = &cobra.Command{
	Use:   "sequelpro",
	Short: "Easily connect local site to sequelpro",
	Long:  `A helper command for easily using sequelpro with a drud app that has been initialized locally.`,
	Run: func(cmd *cobra.Command, args []string) {
		app := platform.PluginMap[strings.ToLower(plugin)]
		app.Init()

		nameContainer := fmt.Sprintf("%s-db", app.ContainerName())

		if !dockerutil.IsRunning(nameContainer) {
			Failed("App not running locally. Try `ddev start`.")
		}

		mysqlContainer, err := dockerutil.GetContainer(nameContainer)
		if err != nil {
			log.Fatal(err)
		}

		dbPort, err := dockerutil.GetDockerPublicPort(mysqlContainer, int64(3306))
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
			platform.SequelproTemplate,
			"data",                  //dbname
			"127.0.0.1",             //host
			mysqlContainer.Names[0], //container name
			"root",                  // dbpass
			strconv.FormatInt(dbPort, 10), // port
			"root", //dbuser
		))

		err = exec.Command("open", tmpFilePath).Run()
		if err != nil {
			log.Fatal(err)
		}

		Success("sequelpro command finished successfully!")

	},
}

func init() {
	RootCmd.AddCommand(LocalDevSequelproCmd)
}
