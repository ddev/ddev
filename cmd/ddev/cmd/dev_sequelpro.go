package cmd

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"

	"github.com/drud/ddev/pkg/local"
	"github.com/drud/drud-go/utils/dockerutil"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// LocalDevSequelproCmd represents the sequelpro command
var LocalDevSequelproCmd = &cobra.Command{
	Use:   "sequelpro [app_name] [environment_name]",
	Short: "Easily connect local site to sequelpro",
	Long:  `A helper command for easily using sequelpro with a drud app that has been initialized locally.`,
	Run: func(cmd *cobra.Command, args []string) {
		app := local.PluginMap[strings.ToLower(plugin)]

		opts := local.AppOptions{
			Name:        activeApp,
			Environment: activeDeploy,
		}
		app.SetOpts(opts)

		nameContainer := fmt.Sprintf("%s-db", app.ContainerName())

		if !dockerutil.IsRunning(nameContainer) {
			Failed("App not running locally. Try `drud legacy add`.")
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

		color.Cyan("sequelpro command finished successfully!")

	},
}

func init() {
	LocalDevSequelproCmd.Flags().BoolVarP(&scaffold, "scaffold", "s", false, "Add the app but don't run or config it.")
	RootCmd.AddCommand(LocalDevSequelproCmd)
	//RootCmd.AddCommand(SequelproCmd)
}
