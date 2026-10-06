package cmd

import (
	"slices"
	"strings"
	"time"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/docker/compose/v5/cmd/display"
	"github.com/docker/compose/v5/pkg/api"
	"github.com/spf13/cobra"
)

var (
	buildAll        bool
	rebuildServices []string
)

// DebugRebuildCmd implements the ddev utility rebuild command
var DebugRebuildCmd = &cobra.Command{
	ValidArgsFunction: ddevapp.GetProjectNamesFunc("all", 1),
	Use:               "rebuild",
	Short:             "Rebuilds the project's Docker cache with verbose output and restarts the project or the specified services.",
	Aliases:           []string{"refresh"},
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 1 {
			util.Failed("This command only takes one optional argument: project name")
		}

		projectName := ""
		if len(args) == 1 {
			projectName = args[0]
		}

		if cmd.Flags().Changed("all") && cmd.Flags().Changed("service") {
			util.Failed("--all flag cannot be used with --service flag")
		}

		_, err := dockerutil.DownloadDockerBuildxIfNeeded()
		if err != nil {
			util.Failed("Failed to download docker-buildx: %v", err)
		}

		app, err := ddevapp.GetActiveApp(projectName)
		if err != nil {
			util.Failed("Failed to get project: %v", err)
		}

		_ = app.DockerEnv()

		if err = app.WriteDockerComposeYAML(); err != nil {
			util.Failed("Failed to get compose-config: %v", err)
		}

		buildDurationStart := util.ElapsedDuration(time.Now())
		composeRenderedPath := app.DockerComposeFullRenderedYAMLPath()
		withoutCache := !cmd.Flags().Changed("cache")

		services := rebuildServices
		if buildAll {
			services = nil
		}
		serviceList := strings.Join(services, ", ")

		if withoutCache {
			output.UserOut.Printf("Rebuilding project images without Docker cache...")
			if buildAll {
				additionalImages, findErr := app.FindAllImages()
				if findErr != nil {
					util.Warning("Unable to find project images: %v", findErr)
				}
				if pullErr := ddevapp.PullBaseContainerImages(additionalImages, true); pullErr != nil {
					util.Warning("Unable to pull Docker images: %v", pullErr)
				}
			} else {
				// Only pull the base images for the services being rebuilt, not the whole project.
				serviceImages, findErr := app.FindServiceImages(services)
				if findErr != nil {
					util.Warning("Unable to find images for %s: %v", serviceList, findErr)
				}
				if pullErr := dockerutil.PullImages(serviceImages, true); pullErr != nil {
					util.Warning("Unable to pull Docker images: %v", pullErr)
				}
			}
		} else {
			output.UserOut.Printf("Rebuilding project images using Docker cache...")
		}

		buildProject, loadErr := dockerutil.LoadComposeProject([]string{composeRenderedPath}, api.ProjectLoadOptions{
			ProjectName: app.GetComposeProjectName(),
			Profiles:    []string{`*`},
		})
		if loadErr != nil {
			util.Failed("Failed to load compose project: %v", loadErr)
		}
		composeCtx, composeSvc, svcErr := dockerutil.NewComposeService()
		if svcErr != nil {
			util.Failed("Failed to create compose service: %v", svcErr)
		}
		err = composeSvc.Build(composeCtx, buildProject, api.BuildOptions{
			Progress: display.ModePlain,
			NoCache:  withoutCache,
			Services: services,
		})
		if err != nil {
			util.Failed("Failed to build project: %v", err)
		}

		buildDuration := util.FormatDuration(buildDurationStart())
		if buildAll {
			util.Success("Rebuilt %s cache in %s", app.Name, buildDuration)
			if err = app.Restart(); err != nil {
				util.Failed("Failed to restart project: %v", err)
			}
			util.Success("Restarted %s", app.GetName())
			return
		}
		util.Success("Rebuilt %s service cache for %s in %s", serviceList, app.Name, buildDuration)

		// Restart the entire project only when changing "web",
		// since app.Start() includes a lot of extra logic
		// and just restarting the web service isn't enough here
		if slices.Contains(services, "web") {
			if err = app.RestartKeepingProfiles(ddevapp.StartOptions{}); err != nil {
				util.Failed("Failed to restart project: %v", err)
			}
			util.Success("Restarted %s", app.GetName())
			return
		}

		// Recreate only the services that have a container, so a rebuild doesn't
		// start a profile service that wasn't started.
		var recreate []string
		for _, service := range services {
			if c, err := app.FindContainerByType(service); err == nil && c != nil {
				recreate = append(recreate, service)
			}
		}
		if len(recreate) > 0 {
			recreateList := strings.Join(recreate, ", ")
			output.UserOut.Printf("Recreating %s...", recreateList)
			if err = app.RecreateServices(recreate); err != nil {
				util.Failed("Failed to recreate %s: %v", recreateList, err)
			}
			util.Success("Recreated %s for %s", recreateList, app.GetName())
		}
	},
}

func registerDebugRebuildCmd() {
	DebugCmd.AddCommand(DebugRebuildCmd)
	DebugRebuildCmd.Flags().BoolVarP(&buildAll, "all", "a", false, "Rebuild all services and restart the project")
	DebugRebuildCmd.Flags().Bool("cache", false, "Keep Docker cache")
	DebugRebuildCmd.Flags().StringSliceVarP(&rebuildServices, "service", "s", []string{"web"}, "Rebuild these comma-separated services and restart them")
	_ = DebugRebuildCmd.RegisterFlagCompletionFunc("service", serviceListCompletionFunc(false))
}
