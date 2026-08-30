package cmd

import (
	"fmt"
	"strings"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// DotEnvGetCmd implements the "ddev dotenv get" command
var DotEnvGetCmd = &cobra.Command{
	Use:   "get [file]",
	Short: "Get the value of an environment variable from a .env file",
	Long: `Retrieve the value of an environment variable specified via a long flag from a .env file.
Provide the path relative to the project root when specifying the file.`,
	Example: `ddev dotenv get .env --app-key
ddev dotenv get .ddev/.env.web --env-key
ddev dotenv get .ddev/.env.redis --redis-tag
ddev dotenv get .ddev/.env.web.local --api-key`,
	Args: cobra.ExactArgs(1),
	FParseErrWhitelist: cobra.FParseErrWhitelist{
		UnknownFlags: true,
	},
	Run: func(cmd *cobra.Command, args []string) {
		app, err := ddevapp.GetActiveApp("")
		if err != nil {
			util.Failed(err.Error())
		}
		dotEnvGet(cmd, args[0], app)
	},
}

// DotEnvGlobalGetCmd implements the "ddev dotenv global get" command
var DotEnvGlobalGetCmd = &cobra.Command{
	Use:   "get [file]",
	Short: "Get the value of an environment variable from a global .env file",
	Long: `Retrieve the value of an environment variable specified via a long flag from a global .env file.
Name the file as .ddev/<file>, the same way as in a project; it is read from the global DDEV directory.`,
	Example: `ddev dotenv global get .ddev/.env.web --api-url
ddev dotenv global get .ddev/.env.web.local --api-key`,
	Args: cobra.ExactArgs(1),
	FParseErrWhitelist: cobra.FParseErrWhitelist{
		UnknownFlags: true,
	},
	Run: func(cmd *cobra.Command, args []string) {
		dotEnvGet(cmd, args[0], nil)
	},
}

// dotEnvGet prints one environment variable from an env file, reading it from
// the global DDEV directory when app is nil.
func dotEnvGet(cmd *cobra.Command, arg string, app *ddevapp.DdevApp) {
	envFile := dotEnvFilePath(app, arg)

	// Read the .env file
	envMap, _, err := ddevapp.ReadProjectEnvFile(envFile)
	if err != nil {
		util.Failed("Unable to read %s file: %v", envFile, err)
	}

	// Get unknown flags and ensure only one flag is passed
	envFlags, err := GetUnknownFlags(cmd)
	if err != nil {
		util.Failed("Error reading command flags: %v", err)
	}
	if len(envFlags) < 1 {
		_ = cmd.Help()
		return
	}
	if len(envFlags) != 1 {
		util.Failed("Only one environment variable can be retrieved at a time.")
	}

	var flag string
	for f := range envFlags {
		flag = f
	}

	if !strings.HasPrefix(flag, "--") {
		util.Failed("The flag must be in long format, but received %s", flag)
	}

	// Extract the environment variable name
	envName := strings.ToUpper(strings.ReplaceAll(strings.TrimPrefix(flag, "--"), "-", "_"))
	if val, exists := envMap[envName]; exists {
		// Show a raw, unescaped value without double quotes.
		// See https://stackoverflow.com/questions/50054666/golang-not-escape-a-string-variable
		output.UserOut.Print(strings.Trim(fmt.Sprintf("%#v", val), `"`))
	} else {
		util.Failed("The environment variable '%s' not found in %s", envName, envFile)
	}
}

func registerDotenvGetCmd() {
	DotEnvCmd.AddCommand(DotEnvGetCmd)
	DotEnvGlobalCmd.AddCommand(DotEnvGlobalGetCmd)
}
