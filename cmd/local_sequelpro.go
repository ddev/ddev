package cmd

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"strconv"

	"github.com/drud/drud-go/drudapi"
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

		dclient, err := GetDockerClient()
		if err != nil {
			log.Fatal(err)
		}

		var index int
		var found bool
		containers, err := ListDockerContainers(dclient, nil)
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

		dbPort, err := GetDockerPublicPort(mysqlContainer, int64(3306))
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
			spTemplate,
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

var spTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>ContentFilters</key>
    <dict/>
    <key>auto_connect</key>
    <true/>
    <key>data</key>
    <dict>
        <key>connection</key>
        <dict>
            <key>database</key>
            <string>%s</string>
            <key>host</key>
            <string>%s</string>
            <key>name</key>
            <string>drud/%s</string>
            <key>password</key>
            <string>%s</string>
            <key>port</key>
            <integer>%s</integer>
            <key>rdbms_type</key>
            <string>mysql</string>
            <key>sslCACertFileLocation</key>
            <string></string>
            <key>sslCACertFileLocationEnabled</key>
            <integer>0</integer>
            <key>sslCertificateFileLocation</key>
            <string></string>
            <key>sslCertificateFileLocationEnabled</key>
            <integer>0</integer>
            <key>sslKeyFileLocation</key>
            <string></string>
            <key>sslKeyFileLocationEnabled</key>
            <integer>0</integer>
            <key>type</key>
            <string>SPTCPIPConnection</string>
            <key>useSSL</key>
            <integer>0</integer>
            <key>user</key>
            <string>%s</string>
        </dict>
    </dict>
    <key>encrypted</key>
    <false/>
    <key>format</key>
    <string>connection</string>
    <key>queryFavorites</key>
    <array/>
    <key>queryHistory</key>
    <array/>
    <key>rdbms_type</key>
    <string>mysql</string>
    <key>rdbms_version</key>
    <string>5.5.44</string>
    <key>version</key>
    <integer>1</integer>
</dict>
</plist>`
