package cmd

import (
	"fmt"
	"os"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/manifoldco/promptui"
	"github.com/mattn/go-isatty"
)

// configureProjectHere handles `ddev start` in a directory with no project:
// it writes a config if the directory looks like a web project and the user
// agrees, or autoConfig is set, and otherwise exits explaining what to do.
// skipConfirmation means don't ask, which never configures a project.
func configureProjectHere(autoConfig bool, skipConfirmation bool) {
	cwd, err := os.Getwd()
	if err != nil {
		util.Failed("Could not determine the current directory: %v", err)
	}
	noProject := fmt.Sprintf("No DDEV project is configured in %s", cwd)

	if outer := ddevapp.RegisteredProjectAbove(cwd); outer != "" {
		util.Failed("%s, but it is inside the DDEV project in %s, which has no .ddev/config.yaml.\nRestore that project's .ddev directory, or remove it from 'ddev list' with 'ddev stop --unlist'.", noProject, outer)
	}
	// NewApp, under NewAutoConfigApp, refuses disallowed locations like the home directory.
	app, err := ddevapp.NewAutoConfigApp(cwd)
	if err != nil {
		util.Failed("%s, and one can't be configured here: %v", noProject, err)
	}
	if !app.LooksLikeWebProject() {
		if entries, err := os.ReadDir(cwd); err == nil && len(entries) == 0 {
			util.Failed("%s, and the directory is empty.\nTo start a new project, see https://docs.ddev.com/en/stable/users/quickstart/ or run 'ddev config'.", noProject)
		}
		util.Failed("%s, and it doesn't look like a web project.\nIf it is one, run 'ddev config' to configure it.", noProject)
	}

	problem := app.AutoConfigProblem()
	canAsk := globalconfig.IsInteractive() && isatty.IsTerminal(os.Stdin.Fd()) && !output.JSONOutput && !skipConfirmation
	if !autoConfig && !canAsk {
		util.Failed("%s.\nIt looks like a '%s' project. Run 'ddev start --auto-config' to configure it with DDEV's defaults and start it, or 'ddev config' to choose the settings.", noProject, app.Type)
	}

	output.UserOut.Printf("%s.\nIt looks like a '%s' project. DDEV can configure it as:\n\n%s\n", noProject, app.Type, app.AutoConfigSummary())
	if problem != "" {
		util.Warning("%s", problem)
		if autoConfig {
			util.Failed("Not configuring the project automatically. Run 'ddev config' to choose the settings.")
		}
	}

	prompt := !autoConfig && chooseCustomConfig(problem != "")
	runConfig(ConfigCommand, nil, getConfigApp(), prompt)
}

// chooseCustomConfig asks how to configure the project, returning true for
// the 'ddev config' questions and false for the configuration shown, and
// exits if the user cancels.
func chooseCustomConfig(onlyCustom bool) bool {
	const (
		useShown  = "Yes, configure as shown and start"
		customize = "Customize (answer the 'ddev config' questions), then start"
		cancel    = "Cancel"
	)
	items := []string{useShown, customize, cancel}
	if onlyCustom {
		items = []string{customize, cancel}
	}
	sel := promptui.Select{
		Label: "Configure and start this project",
		Items: items,
	}
	_, choice, err := sel.Run()
	if err != nil || choice == cancel {
		util.Failed("Cancelled. Nothing was configured or started.")
	}
	return choice == customize
}
