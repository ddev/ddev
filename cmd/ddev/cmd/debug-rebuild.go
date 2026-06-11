package cmd

import (
	"fmt"
	"time"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/docker/compose/v5/cmd/display"
	"github.com/docker/compose/v5/pkg/api"
	"github.com/spf13/cobra"
)

var (
	buildAll bool
	service  string
)

// DebugRebuildCmd implements the ddev utility rebuild command
var DebugRebuildCmd = &cobra.Command{
	ValidArgsFunction: ddevapp.GetProjectNamesFunc("all", 1),
	Use:               "rebuild",
	Short:             "Rebuilds the project's Docker cache with verbose output and restarts the project or the specified service.",
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

		var services []string
		if !buildAll {
			services = []string{service}
		}

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
				// Only pull the base image for the service being rebuilt, not the whole project.
				serviceImages, findErr := app.FindServiceImages(services)
				if findErr != nil {
					util.Warning("Unable to find images for service %s: %v", service, findErr)
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
		} else {
			util.Success("Rebuilt %s service cache for %s in %s", service, app.Name, buildDuration)
		}

		// Restart the entire project only when changing "web",
		// since app.Start() includes a lot of extra logic
		// and just restarting the web service isn't enough here
		if buildAll || service == "web" {
			err = app.Restart()
			if err != nil {
				util.Failed("Failed to restart project: %v", err)
			}
			util.Success("Restarted %s", app.GetName())
			return
		}

		labels := map[string]string{
			"com.ddev.site-name":         app.GetName(),
			"com.docker.compose.oneoff":  "False",
			"com.docker.compose.service": service,
		}

		// Recreate the specified service using compose, if it is running
		if container, err := dockerutil.FindContainerByLabels(labels); err == nil && container != nil {
			output.UserOut.Printf("Recreating service %s...", service)

			// Narrow to the target service and drop its dependency edges so we don't disturb other services.
			recreateProject, selErr := buildProject.WithSelectedServices([]string{service}, types.IgnoreDependencies)
			if selErr != nil {
				util.Failed("Failed to select service %s: %v", service, selErr)
			}

			// Recreate rather than restart: a plain restart keeps the old container and image,
			// so the rebuilt image would never be applied, and a fresh container avoids
			// startup-delay healthchecks triggered by restarting a previously-healthy container.
			progress := display.ModeQuiet
			if globalconfig.DdevVerbose {
				progress = display.ModePlain
			}
			recreateTimeout := time.Duration(app.GetMaxContainerWaitTime()) * time.Second
			err = composeSvc.Up(composeCtx, recreateProject, api.UpOptions{
				Create: api.CreateOptions{
					Services: []string{service},
					Recreate: api.RecreateForce,
					Timeout:  &recreateTimeout,
					Build:    &api.BuildOptions{Progress: progress},
				},
				Start: api.StartOptions{
					Project:  recreateProject,
					Services: []string{service},
				},
			})
			if err != nil {
				util.Failed("Failed to recreate service %s: %v", service, err)
			}

			wait := output.StartWait(fmt.Sprintf("Waiting for containers to become ready: %v", []string{service}))
			err = app.Wait([]string{service})
			wait.Complete(err)
			if err != nil {
				util.Failed("Failed to wait for project container [%v] to become ready: %v", service, err)
			}
			if !ddevapp.IsRouterDisabled(app) {
				util.Debug("Starting %s if necessary...", nodeps.RouterContainer)
				err = ddevapp.StartDdevRouter()
				if err != nil {
					util.Failed("Failed to start %s: %v", nodeps.RouterContainer, err)
				}
			}
			util.Success("Recreated %s service for %s", service, app.GetName())
		}
	},
}

func init() {
	DebugCmd.AddCommand(DebugRebuildCmd)
	DebugRebuildCmd.Flags().BoolVarP(&buildAll, "all", "a", false, "Rebuild all services and restart the project")
	DebugRebuildCmd.Flags().Bool("cache", false, "Keep Docker cache")
	DebugRebuildCmd.Flags().StringVarP(&service, "service", "s", "web", "Rebuild the specified service and restart it")
	_ = DebugRebuildCmd.RegisterFlagCompletionFunc("service", ddevapp.GetServiceNamesFunc(false))
}
