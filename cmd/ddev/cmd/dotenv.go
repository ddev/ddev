package cmd

import (
	"path/filepath"
	"strings"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// DotEnvCmd is the top-level "ddev dotenv" command
var DotEnvCmd = &cobra.Command{
	Use:   "dotenv [command]",
	Short: "Commands for managing the contents of .env files",
	Run: func(cmd *cobra.Command, _ []string) {
		err := cmd.Usage()
		util.CheckErr(err)
	},
}

// DotEnvGlobalCmd is the "ddev dotenv global" command
var DotEnvGlobalCmd = &cobra.Command{
	Use:   "global [command]",
	Short: "Commands for managing the contents of global .env files",
	Run: func(cmd *cobra.Command, _ []string) {
		err := cmd.Usage()
		util.CheckErr(err)
	},
}

// dotEnvFilePath resolves the file argument of a dotenv command against the
// project root, or against the global DDEV directory when app is nil.
func dotEnvFilePath(app *ddevapp.DdevApp, arg string) string {
	var baseDir, baseDirName string
	if app != nil {
		baseDir, baseDirName = app.GetAbsAppRoot(false), "project root"
	} else {
		baseDir, baseDirName = globalconfig.GetGlobalDdevDir(), "global DDEV directory"
	}

	envFile := arg
	if !filepath.IsAbs(envFile) {
		envFile = filepath.Join(baseDir, envFile)
	}

	// The file must stay within the base directory
	relPath, err := filepath.Rel(baseDir, envFile)
	if err != nil || !filepath.IsLocal(relPath) {
		util.Failed("The provided path %s is outside the %s %s", envFile, baseDirName, baseDir)
	}

	baseName := filepath.Base(envFile)
	if baseName != ".env" && !strings.HasPrefix(baseName, ".env.") {
		util.Failed("The file should have .env prefix")
	}
	return envFile
}

func init() {
	DotEnvCmd.AddCommand(DotEnvGlobalCmd)
	RootCmd.AddCommand(DotEnvCmd)
}
