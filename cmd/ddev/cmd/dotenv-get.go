package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// DotEnvGetCmd implements the "ddev dotenv get" command
var DotEnvGetCmd = &cobra.Command{
	Use:   "get [file]",
	Short: "Get the value of an environment variable from a .env file",
	Long: `Retrieve the value of an environment variable specified via a long flag from a .env file.
The .env file should be named .env or .env.<servicename> or .env.<something>
Provide the path relative to the project root when specifying the file.`,
	Example: `ddev dotenv get .env --app-key
ddev dotenv get .ddev/.env --env-key
ddev dotenv get .ddev/.env.redis --redis-tag`,
	Args: cobra.ExactArgs(1),
	FParseErrWhitelist: cobra.FParseErrWhitelist{
		UnknownFlags: true,
	},
	Run: func(cmd *cobra.Command, args []string) {
		app, err := ddevapp.GetActiveApp("")
		if err != nil {
			util.Failed(err.Error())
		}

		// Get the .env file from the approot
		envFile := filepath.Join(app.GetAbsAppRoot(false), args[0])

		// Validate absolute paths
		if filepath.IsAbs(args[0]) {
			envFile = args[0]
			relPath, err := filepath.Rel(app.GetAbsAppRoot(false), envFile)
			if err != nil || strings.HasPrefix(relPath, "..") {
				util.Failed("The provided path %s is outside the project root %s", envFile, app.GetAbsAppRoot(false))
			}
		}

		baseName := filepath.Base(envFile)
		if baseName != ".env" && !strings.HasPrefix(baseName, ".env.") {
			util.Failed("The file should have .env prefix")
		}

		// Read the .env file
		envMap, _, err := ddevapp.ReadProjectEnvFile(envFile)
		if err != nil {
			util.Failed("Unable to read %s file: %v", envFile, err)
		}

		// Get unknown flags and ensure only one flag is passed
		envFlags := GetUnknownFlags(cmd)
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
			util.Failed("The flag must be in a long format.")
		}

		// Extract the environment variable name
		envName := strings.ToUpper(strings.ReplaceAll(strings.TrimPrefix(flag, "--"), "-", "_"))
		if val, exists := envMap[envName]; exists {
			// Show a raw, unescaped value without double quotes.
			// See https://stackoverflow.com/questions/50054666/golang-not-escape-a-string-variable
			fmt.Println(strings.Trim(fmt.Sprintf("%#v", val), `"`))
		} else {
			util.Failed("The environment variable '%s' not found in %s", envName, envFile)
		}
	},
}

func init() {
	DotEnvCmd.AddCommand(DotEnvGetCmd)
}
