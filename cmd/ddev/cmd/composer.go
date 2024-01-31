package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// ComposerCmd handles ddev composer
var ComposerCmd = &cobra.Command{
	DisableFlagParsing: true,
	Use:                "composer [command]",
	Short:              "Executes a Composer command within the web container",
	Long: `Executes a Composer command at the Composer root in the web container. Generally,
any Composer command can be forwarded to the container context by prepending
the command with 'ddev'.`,
	Example: `ddev composer install
ddev composer require <package>
ddev composer outdated --minor-only
ddev composer create drupal/recommended-project`,
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		app, err := ddevapp.GetActiveApp("")
		// If there's no active app or the app isn't running, we can't get completions from composer
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		status, _ := app.SiteStatus()
		if status != ddevapp.SiteRunning {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		// Get the host shell, so the composer completion is compatible
		shell, err := exec.GetHostShell()
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		// Try to get the web service container
		containerName := ddevapp.GetContainerName(app, "web")
		container, err := dockerutil.FindContainerByName(containerName)
		if err != nil || container == nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		// Prepare the -c and -i arguments for symfony's __complete command
		current := fmt.Sprintf("%d", len(args)+1)
		input := []string{}
		for _, val := range args {
			input = append(input, "-i"+val)
		}
		if toComplete != "" {
			input = append(input, "-i"+toComplete)
		}

		// Try to get completion from composer in the container
		stdout, _, err := app.Exec(&ddevapp.ExecOpts{
			Service: "web",
			Dir:     app.GetComposerRoot(true, true),
			RawCmd:  append([]string{"composer", "_complete", "-S2.6.6", "-n", "-s" + shell, "-c" + current, "-icomposer"}, input...),
			Tty:     false,
			// Prevent Composer from debugging when Xdebug is enabled
			Env: []string{"XDEBUG_MODE=off"},
		})
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		// Exclude create-project, which will already be provided in completion by cobra
		// and isn't supported by ddev anyway
		completion := []string{}
		for _, val := range strings.Split(stdout, "\n") {
			if val != "create-project" {
				completion = append(completion, val)
			}
		}

		return completion, cobra.ShellCompDirectiveNoFileComp
	},
	Run: func(cmd *cobra.Command, args []string) {
		app, err := ddevapp.GetActiveApp("")
		if err != nil {
			util.Failed(err.Error())
		}

		status, _ := app.SiteStatus()
		if status != ddevapp.SiteRunning {
			if err = app.Start(); err != nil {
				util.Failed("Failed to start %s: %v", app.Name, err)
			}
		}

		stdout, stderr, err := app.Composer(args)
		if err != nil {
			util.Failed("Composer %v failed, %v. stderr=%v", args, err, stderr)
		}
		_, _ = fmt.Fprint(os.Stderr, stderr)
		_, _ = fmt.Fprint(os.Stdout, stdout)
	},
}

func init() {
	ComposerCmd.InitDefaultHelpFlag()
	err := ComposerCmd.Flags().MarkHidden("help")
	originalHelpFunc := ComposerCmd.HelpFunc()
	if err == nil {
		ComposerCmd.SetHelpFunc(func(command *cobra.Command, strings []string) {
			if command == ComposerCmd {
				_ = command.Flags().MarkHidden("json-output")
			}
			originalHelpFunc(command, strings)
		})
	}
	RootCmd.AddCommand(ComposerCmd)
}
