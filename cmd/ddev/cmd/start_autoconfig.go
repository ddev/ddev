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
// agrees, and otherwise exits explaining what to do.
// skipConfirmation means don't ask, which never configures a project.
func configureProjectHere(skipConfirmation bool) {
	cwd, err := os.Getwd()
	if err != nil {
		util.Failed("Could not determine the current directory: %v", err)
	}
	noProject := fmt.Sprintf("No DDEV project is configured in %s", cwd)

	if outer := ddevapp.RegisteredProjectAbove(cwd); outer != "" {
		util.Failed("%s, but it is inside the DDEV project in %s, which has no .ddev/config.yaml.\nRestore that project's .ddev directory, or remove it from 'ddev list' with 'ddev stop --unlist'.", noProject, outer)
	}
	app := getConfigApp()
	if err := ddevapp.HasAllowedLocation(app); err != nil {
		util.Failed("%s, and one can't be configured here: %v", noProject, err)
	}
	// Some apptype detectors look in the docroot; runConfig discovers it the same way.
	app.Docroot = ddevapp.DiscoverDefaultDocroot(app)
	if !app.LooksLikeWebProject() {
		if entries, err := os.ReadDir(cwd); err == nil && len(entries) == 0 {
			util.Failed("%s, and the directory is empty.\nTo start a new project, see https://docs.ddev.com/en/stable/users/quickstart/ or run 'ddev config'.", noProject)
		}
		util.Failed("%s, and it doesn't look like a web project.\nIf it is one, run 'ddev config' to configure it.", noProject)
	}

	canAsk := globalconfig.IsInteractive() && isatty.IsTerminal(os.Stdin.Fd()) && !output.JSONOutput && !skipConfirmation
	if !canAsk {
		util.Failed("%s.\nIt looks like a web project. Run 'ddev config --auto && ddev start' to configure it with DDEV's defaults and start it, or 'ddev config' to choose the settings.", noProject)
	}

	runConfig(ConfigCommand, nil, app, false, func(app *ddevapp.DdevApp) error {
		output.UserOut.Printf("%s.\nIt looks like a '%s' project. DDEV can configure it as:\n\n%s\n", noProject, app.Type, app.AutoConfigSummary())
		problem := app.AutoConfigProblem()
		if problem != "" {
			util.Warning("%s", problem)
		}
		if chooseCustomConfig(problem != "") {
			return errCustomizeConfig
		}
		return nil
	})
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
