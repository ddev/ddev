package cmd

import (
	"os"
	"os/user"
	"strings"

	"github.com/drud/ddev/pkg/local"
	drudfiles "github.com/drud/drud-go/files"

	log "github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	timestampFormat = "20060102150405"
	updateFile      = ".drud-update"
)

var (
	cfg                *local.Config
	usr                *user.User
	pwd                string
	cfgFile            string
	bucket             string                   // aws s3 bucket used with file storage functionality
	region             = "us-west-2"            // region where the s3 bucket can be found
	creds              *credentials.Credentials // s3 related credentials
	awsID              string
	awsSecret          string
	dbFile             string
	homedir            string // current user's home directory
	fileService        *drudfiles.FileService
	clientCreateAccess bool
	filesAccess        bool
	drudAccess         bool
	bucketName         = "nmdarchive"
	forceDelete        bool
	logLevel           = log.WarnLevel
	plugin             = ""
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "ddev",
	Short: "A CLI for interacting with DRUD.",
	Long:  "This Command Line Interface (CLI) gives you the ability to interact with the DRUD platform to manage applications, create a local development environment, or deploy an application to production. DRUD also provides utilities for securely uploading files and secrets associated with applications.",
	Run: func(cmd *cobra.Command, args []string) {
		// fmt.Println(`Use "drud --help" for more information about this tool.`)
	},
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		ignores := []string{"list", "config", "version", "update"}
		skip := false
		command := strings.Join(os.Args, " ")

		for _, k := range ignores {
			if strings.Contains(command, " "+k) {
				skip = true
				break
			}
		}

		if !skip {
			parseLegacyArgs(args)
			plugin = strings.ToLower(plugin)
			if _, ok := local.PluginMap[plugin]; !ok {
				Failed("Plugin %s is not registered", plugin)
			}
		}
	},
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cfgFile = ParseConfigFlag()

	// sets up config path and defaults
	PrepConf()
	// bind flags to viper config values...allows override by flag
	//viper.BindPFlag("vault_host", RootCmd.PersistentFlags().Lookup("vault_host"))
	viper.SetConfigFile(cfgFile)
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		log.Fatalf("Fatal error config file: %s \n", err)
	}

	cfg, err = local.GetConfig()
	if err != nil {
		log.Fatal(err)
	}

	if err := RootCmd.Execute(); err != nil {
		os.Exit(-1)
	}

}

func init() {
	cobra.OnInitialize(initConfig)
	RootCmd.PersistentFlags().StringVarP(&plugin, "plugin", "p", "legacy", "Choose which plugin to use")
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "$HOME/drud.yaml", "yaml config file")
	cfgFile = ParseConfigFlag()
	_, err := local.GetConfig()
	if err != nil {
		log.Fatal(err)
	}

	SetHomedir()

	drudDebug := os.Getenv("DRUD_DEBUG")
	if drudDebug != "" {
		logLevel = log.InfoLevel
	}

	log.SetLevel(logLevel)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {}

func parseLegacyArgs(args []string) {
	activeApp = cfg.ActiveApp
	activeDeploy = cfg.ActiveDeploy
	appClient = cfg.Client

	if len(args) > 1 {
		if args[0] != "" {
			activeApp = args[0]
		}

		if args[1] != "" {
			activeDeploy = args[1]
		}
	}
	if activeApp == "" {
		log.Fatalln("No app name found. app_name and deploy_name are expected as arguments.")
	}
	if activeDeploy == "" {
		log.Fatalln("No deploy name found. app_name and deploy_name are expected as arguments.")
	}
}
