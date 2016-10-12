package cmd

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"strconv"

	"github.com/drud/bootstrap/cli/local"

	"github.com/drud/drud-go/drudapi"
	"github.com/drud/drud-go/utils"
	"github.com/spf13/cobra"
)

var appName string

// SequelproCmd represents the sequelpro command
var SequelproCmd = &cobra.Command{
	Use:   "sequelpro",
	Short: "Easily connect local site to sequelpro",
	Long:  `A helper command for easily using sequelpro with a drud app that has been initialized locally.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 2 {
			log.Fatalln("app_name and deploy_name are expected as arguments.")
		}

		if appClient == "" {
			appClient = cfg.Client
		}

		al := &drudapi.ApplicationList{}
		drudclient.Query = fmt.Sprintf(`where={"name":"%s","client":"%s"}`, args[0], appClient)

		err := drudclient.Get(al)
		if err != nil {
			log.Fatal(err)
		}

		if len(al.Items) == 0 {
			log.Fatalln("No app found.")
		}

		app := al.Items[0]
		nameContainer := fmt.Sprintf("/%s-%s-db", app.AppID, args[1])

		dclient, err := utils.GetDockerClient()
		if err != nil {
			log.Fatal(err)
		}

		var index int
		var found bool
		containers, err := utils.ListDockerContainers(dclient, nil)
		for i, container := range containers {
			if container.Names[0] == nameContainer {
				index = i
				found = true
			}
		}
		if !found {
			log.Fatalln("no container found.")
		}

		mysqlContainer := containers[index]

		dbPort, err := utils.GetDockerPublicPort(mysqlContainer, int64(3306))
		if err != nil {
			log.Fatal(err)
		}

		basePath := path.Join(homedir, ".drud", appClient, args[0], args[1])
		tmpFilePath := path.Join(basePath, "sequelpro.spf")
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

		exec.Command("open", tmpFilePath).Run()
	},
}

func init() {
	SequelproCmd.Flags().StringVarP(&appClient, "client", "c", "", "Client name")
	LocalCmd.AddCommand(SequelproCmd)
	//RootCmd.AddCommand(SequelproCmd)
}
