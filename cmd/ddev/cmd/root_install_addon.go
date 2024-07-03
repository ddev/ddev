package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// Prompts user to install add-on if the provided arg is unknown,
// and it's in the list of available official add-ons as ddev/ddev-<arg>
// For example, "ddev redis" will prompt user to install the "ddev/ddev-redis" addon.
func installAddonForUnknownCommand(command *cobra.Command, arg string) {
	if os.Getenv("DDEV_NONINTERACTIVE") != "" {
		return
	}
	// If there is an error here, it means that this arg is not a known command,
	// we can check if it matches an add-on name to install it.
	_, _, findErr := command.Find([]string{arg})
	if findErr == nil {
		return
	}
	// If there is no project, we can't install an add-on.
	app, err := ddevapp.GetActiveApp("")
	if err != nil {
		return
	}
	// Get a list of official add-ons.
	repos, err := ddevapp.ListAvailableAddons(true)
	if err != nil {
		return
	}
	// Check if the arg matches the add-on name as ddev/ddev-<arg>
	addonToInstall := ""
	for _, repo := range repos {
		if strings.EqualFold(arg, strings.TrimPrefix(*repo.Name, "ddev-")) {
			addonToInstall = *repo.FullName
			break
		}
	}
	if addonToInstall == "" {
		return
	}
	// If it's already installed, there's no reason to install it again.
	for _, manifest := range ddevapp.GetInstalledAddons(app) {
		if addonToInstall == manifest.Repository {
			return
		}
	}
	// Show the original error for unknown command
	command.PrintErrln(command.ErrPrefix(), findErr.Error())

	if yes := util.Confirm(fmt.Sprintf("\nDid you mean to install '%s' add-on?", addonToInstall)); !yes {
		os.Exit(1)
	}

	getArgs := []string{"get", addonToInstall}
	command.SetArgs(getArgs)
	output.UserOut.Printf("\nExecuting command: %v\n", append([]string{command.CommandPath()}, getArgs...))
	err = command.Execute()
	if err != nil {
		os.Exit(-1)
	}
	util.Success("\n%s has been installed and requires a restart.\n", addonToInstall)

	if yes := util.Confirm("Would you like to restart DDEV now?"); !yes {
		os.Exit(1)
	}

	output.UserOut.Printf("Restarting project %s...", app.GetName())
	err = app.Restart()
	if err != nil {
		util.Failed("Failed to restart %s: %v", app.GetName(), err)
	}
	os.Exit(0)
}
