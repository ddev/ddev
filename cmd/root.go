package cmd

import (
	"fmt"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/drud/drud-go/drudapi"
	"github.com/drud/drud-go/secrets"
	"github.com/hashicorp/vault/api"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	timestampFormat = "20060102150405"
	updateFile      = ".drud-update"
	tokenFile       = ".drud-sanctuary-token"
	cliVersion      = "0.2.1"
	drudapiVersion  = "v0.1"
)

var (
	cfg                *Config
	usr                *user.User
	pwd                string
	cfgFile            string
	drudconf           string           //absolute path to cfg file
	drudclient         *drudapi.Request //client for interacting with drud api
	workdir            string
	bucket             string                   // aws s3 bucket used with file storage functionality
	region             string                   // region where the s3 bucket can be found
	creds              *credentials.Credentials // s3 related credentials
	svc                *s3.S3
	awsID              string
	vaultAddress       string // stores the vault host address
	awsSecret          string
	dbFile             string
	isDev              bool   // isDev stores boolean value to allow special functionality for devs
	homedir            string // current user's home directory
	gitToken           string
	clientCreateAccess bool
	filesAccess        bool
	drudAccess         bool
	bucketName         = "drudcli-drud-files-bucket"
	forceDelete        bool
	vaultToken         string
	vault              api.Logical // part of the vault go api
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "drud",
	Short: "A CLI for interacting with DRUD.",
	Long:  "This Command Line Interface (CLI) gives you the ability to interact with the DRUD platform to manage applications, create a local development environment, or deploy an application to production. DRUD also provides utilities for securely uploading files and secrets associated with applications.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(`Use "drud --help" for more information about this tool.`)
	},
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {

	//RootCmd.RemoveCommand(SecretCmd)
	SetHomedir()
	tokenLocation := filepath.Join(homedir, tokenFile)
	if len(os.Args) == 2 && os.Args[1] != "--hrlp" {
		if _, err := os.Stat(tokenLocation); os.IsNotExist(err) {
			RootCmd.Help()
			os.Exit(0)
		}
	}

	SetConf()

	// create a blank drud config if one does not exist
	if _, err := os.Stat(drudconf); os.IsNotExist(err) {
		f, ferr := os.Create(drudconf)
		if ferr != nil {
			log.Fatal(ferr)
		}
		defer f.Close()
		//default drud.yaml contents
		f.WriteString(`Client: 1fee
Dev: false
DrudHost: drudapi.genesis.drud.io
Protocol: https
VaultHost: https://sanctuary.drud.io:8200
Version: v0.1
`)

	}

	// prepopulate tokenfile so i dont have to check for its existence everywhere
	if _, err := os.Stat(tokenLocation); os.IsNotExist(err) {
		f, ferr := os.Create(tokenLocation)
		if ferr != nil {
			log.Fatal(ferr)
		}
		defer f.Close()
		//default drud.yaml contents
		f.WriteString("placeholder")
	}

	// bind flags to viper config values...allows override by flag
	//viper.BindPFlag("vault_host", RootCmd.PersistentFlags().Lookup("vault_host"))
	viper.BindPFlag("dev", RootCmd.PersistentFlags().Lookup("dev"))
	viper.SetConfigFile(drudconf)
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		log.Fatalf("Fatal error config file: %s \n", err)
	}

	cfg, err = GetConfig()
	if err != nil {
		log.Fatal(err)
	}

	gitToken = os.Getenv("GITHUB_TOKEN")

	// load the vault token from disk and use it to get policy information
	// if permission is denied send the user through `drud auth github` and then try again
	if len(os.Args) >= 2 && os.Args[1] != "auth" {
		for i := 0; i < 2; i++ {
			vaultToken = secrets.ConfigVault(tokenLocation, cfg.VaultHost)
			vault = secrets.GetVault()
			err = EnableAvailablePackages()
			if err != nil {
				if strings.Contains(err.Error(), "permission denied") && i == 0 {
					githubCmd.Run(RootCmd, []string{})
					continue
				}
				log.Fatal(err)
			}
			break
		}
	}

	if err := RootCmd.Execute(); err != nil {
		os.Exit(-1)
	}

}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports Persistent Flags, which, if defined here,
	// will be global for your application.

	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/drud.yaml)")
	RootCmd.PersistentFlags().StringVar(&vaultAddress, "vault_address", "https://sanctuary.drud.io:8200", "Vault Address")
	RootCmd.PersistentFlags().BoolVarP(&isDev, "dev", "", false, "Enable Dev mode")
	// files related
	RootCmd.PersistentFlags().StringVar(&bucket, "bucket", "nmdarchive", "name of S3 bucket to work with")
	RootCmd.PersistentFlags().StringVar(&region, "region", "us-west-2", "region your bucket is in")
	RootCmd.PersistentFlags().StringVar(&workdir, "workdir", "", "directory other than cwd we are running from")
	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	RootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {

	if drudAccess {
		// auth with vault token if available
		eveCreds := &drudapi.Credentials{}
		if vaultToken != "" {
			eveCreds.Token = vaultToken
		} else {
			eveCreds.AdminToken = gitToken
		}

		// drud api client
		drudclient = &drudapi.Request{
			Host: cfg.EveHost(),
			Auth: eveCreds,
		}
	}

	if filesAccess {

		sobj := secrets.Secret{
			Path: "secret/shared/services/awscfg",
		}

		err := sobj.Read()
		if err != nil {
			log.Fatal(err)
		}

		awsID = sobj.Data["accesskey"].(string)
		awsSecret = sobj.Data["secretkey"].(string)
		os.Setenv("AWS_ACCESS_KEY_ID", awsID)
		os.Setenv("AWS_SECRET_ACCESS_KEY", awsSecret)

		svc = s3.New(session.New(&aws.Config{Region: aws.String(region)}))
	}

}
