package cmd

import (
	"slices"
	"strings"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

var restartAll bool

// RestartCmd rebuilds an apps settings
var RestartCmd = &cobra.Command{
	ValidArgsFunction: ddevapp.GetProjectNamesFunc("all", 0),
	Use:               "restart [projects]",
	Short:             "Restart a project or several projects.",
	Long: `Stops named projects and then starts them back up again.
With --service, only the named services get new containers, which also brings
back a service that is unhealthy or has crashed, and the rest of the project
keeps running. Restarting web restarts the whole project, with the optional
services that were started.`,
	Example: `ddev restart
ddev restart <project1> <project2>
ddev restart --all
ddev restart --profiles=busybox --no-cache
ddev restart --service=db
ddev restart -s solr,redis --no-cache
ddev restart --reset-database`,
	PreRun: func(_ *cobra.Command, _ []string) {
		dockerutil.EnsureDdevNetwork()
	},
	Run: func(cmd *cobra.Command, args []string) {
		projects, err := getRequestedProjects(args, restartAll)
		if err != nil {
			util.Failed("Failed to get project(s): %v", err)
		}
		if len(projects) > 0 {
			instrumentationApp = projects[0]
		}

		// Look for version change and opt-in to instrumentation if it has changed.
		// Do it here rather than waiting for app.Start() so the poweroff prompt
		// comes before any of the restart output.
		ddevapp.RunUpgradeCheck()

		noCache, _ := cmd.Flags().GetBool("no-cache")
		profilesFlag, _ := cmd.Flags().GetString("profiles")
		services, _ := cmd.Flags().GetStringSlice("service")
		serviceList := strings.Join(services, ", ")
		seedSnapshot, _ := cmd.Flags().GetString("seed-snapshot")
		resetDatabase, _ := cmd.Flags().GetBool("reset-database")
		omitSnapshot, _ := cmd.Flags().GetBool("omit-snapshot")
		skipConfirmation, _ := cmd.Flags().GetBool("skip-confirmation")
		checkResetDatabaseFlags(resetDatabase, omitSnapshot, restartAll)

		for _, app := range projects {
			app.SeedSnapshot = seedSnapshot

			if resetDatabase {
				if err := resetProjectDatabase(app, omitSnapshot, skipConfirmation); err != nil {
					util.Failed("Failed to reset the database of %s: %v", app.GetName(), err)
				}
			}

			if len(services) > 0 {
				output.UserOut.Printf("Restarting %s in project %s...", serviceList, app.GetName())
				if err = app.RestartServices(services, ddevapp.StartOptions{NoCache: noCache}); err != nil {
					util.Failed("Failed to restart %s in %s: %v", serviceList, app.GetName(), err)
				}
				util.Success("Restarted %s in %s", serviceList, app.GetName())
				if slices.Contains(services, "web") {
					emitReachProjectMessage(app)
				}
				continue
			}

			output.UserOut.Printf("Restarting project %s...", app.GetName())
			err = app.RestartWith(ddevapp.StartOptions{Profiles: splitProfiles(profilesFlag), NoCache: noCache})
			if err != nil {
				util.Failed("Failed to restart %s: %v", app.GetName(), err)
			}

			util.Success("Restarted %s", app.GetName())
			emitReachProjectMessage(app)
		}
	},
}

// serviceListCompletionFunc completes a comma-separated list of the
// project's services, only those with a container when existingOnly is set.
func serviceListCompletionFunc(existingOnly bool) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	serviceNames := ddevapp.GetServiceNamesFunc(existingOnly)
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		names, _ := serviceNames(cmd, args, toComplete)
		return configCompletionFuncWithCommas(names)(cmd, args, toComplete)
	}
}

func registerRestartCmd() {
	RestartCmd.Flags().BoolP("skip-confirmation", "y", false, "Skip any confirmation steps")
	RestartCmd.Flags().BoolP("no-cache", "", false, "Rebuild custom Docker image layers without cache")
	RestartCmd.Flags().BoolVarP(&restartAll, "all", "a", false, "Restart all projects")
	RestartCmd.Flags().String("profiles", "", "Start optional comma-separated docker compose profiles")
	RestartCmd.Flags().StringSliceP("service", "s", nil, "Restart only these comma-separated services")
	_ = RestartCmd.RegisterFlagCompletionFunc("service", serviceListCompletionFunc(true))
	addResetDatabaseFlags(RestartCmd)
	for _, flag := range []string{"all", "profiles", "reset-database", "seed-snapshot"} {
		RestartCmd.MarkFlagsMutuallyExclusive("service", flag)
	}
	RootCmd.AddCommand(RestartCmd)
}
