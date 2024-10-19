package cmd

import (
	"fmt"
	"github.com/ddev/ddev/pkg/output"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// DotEnvSetCmd implements the "ddev dotenv set" command
var DotEnvSetCmd = &cobra.Command{
	Use:   "set [file]",
	Short: "Write values from the command line to a .env file",
	Long: `Create or update a .env file with values specified via long flags from the command line.
Flags in the format --env-key=value will be converted to environment variable names 
like ENV_KEY="value". The .env file should be named .env or .env.<servicename> or .env.<something>
All environment variables can be used and expanded in .ddev/docker-compose.*.yaml files.
Provide the path relative to the project root when specifying the file.`,
	Example: `ddev dotenv set .env --app-key=value
ddev dotenv set .ddev/.env --extra value --another-key "extra value"
ddev dotenv set .ddev/.env.redis --redis-tag 7-bookworm`,
	Args:    cobra.ExactArgs(1),
	Aliases: []string{"add"},
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
		// If this is an absolute path, make sure it's inside the approot
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

		envMap, envText, err := ddevapp.ReadProjectEnvFile(envFile)
		if err != nil && !os.IsNotExist(err) {
			util.Failed("Unable to read %s file: %v", envFile, err)
		}

		// Create a copy of the original envMap for comparison
		originalEnvMap := make(map[string]string, len(envMap))
		for k, v := range envMap {
			originalEnvMap[k] = v
		}

		// Get unknown flags and convert them to env variables
		envSlice, err := GetUnknownFlags(cmd)
		if err != nil {
			util.Failed("Error reading command flags: %v", err)
		}
		hasUnknownFlags := false
		changedEnvMap := make(map[string]string)
		for flag, value := range envSlice {
			if strings.HasPrefix(flag, "--") {
				envName := strings.ToUpper(strings.ReplaceAll(strings.TrimPrefix(flag, "--"), "-", "_"))
				envMap[envName] = value
				changedEnvMap[envName] = value
				hasUnknownFlags = true
			} else {
				util.Failed("The flag must be in long format, but received %s", flag)
			}
		}

		if !hasUnknownFlags {
			_ = cmd.Help()
			return
		}

		// Write only if there are changes
		if !reflect.DeepEqual(originalEnvMap, envMap) {
			if err := ddevapp.WriteProjectEnvFile(envFile, changedEnvMap, envText); err != nil {
				util.Failed("Error writing .env file: %v", err)
			}
		}

		// Sort before output, since the map order is not deterministic
		rawResultKeys := make([]string, 0, len(changedEnvMap))
		for k := range changedEnvMap {
			rawResultKeys = append(rawResultKeys, k)
		}
		sort.Strings(rawResultKeys)
		// Prepare the friendly message with formatted environment variables
		var formattedVars []string
		for _, k := range rawResultKeys {
			formattedVars = append(formattedVars, fmt.Sprintf(`%s="%v"`, k, strings.ReplaceAll(changedEnvMap[k], `"`, `\"`)))
		}
		friendlyMsg := fmt.Sprintf("Updated %s with:\n\n%s", envFile, strings.Join(formattedVars, "\n"))
		output.UserOut.WithField("raw", changedEnvMap).Print(friendlyMsg)
	},
}

func init() {
	DotEnvCmd.AddCommand(DotEnvSetCmd)
}
