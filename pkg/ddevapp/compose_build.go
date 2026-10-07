package ddevapp

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	composeTypes "github.com/compose-spec/compose-go/v2/types"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/util"
	"github.com/docker/compose/v5/cmd/display"
	"github.com/docker/compose/v5/pkg/api"
)

// loadRenderedProject loads the rendered compose file with the given profiles,
// following the same rules as StartOptions.Profiles.
func (app *DdevApp) loadRenderedProject(profiles []string) (*composeTypes.Project, error) {
	return dockerutil.LoadComposeProject([]string{app.DockerComposeFullRenderedYAMLPath()}, api.ProjectLoadOptions{
		ProjectName: app.GetComposeProjectName(),
		Profiles:    profiles,
	})
}

// composeBuild executes docker-compose build for the project's enabled
// services, or for opts.Services when set.
func (app *DdevApp) composeBuild(project *composeTypes.Project, opts api.BuildOptions) error {
	goCtx, _, err := dockerutil.GetDockerClient()
	if err != nil {
		return fmt.Errorf("docker-compose build failed: %v", err)
	}

	util.Debug("Executing docker-compose build -f %s", app.DockerComposeFullRenderedYAMLPath())

	ctx, cancel := context.WithTimeout(goCtx, time.Hour)
	defer cancel()

	stopDots := util.ShowDots()

	opts.Progress = display.ModePlain
	out, stderr, err := dockerutil.CaptureOutput(func(svc api.Compose) error {
		return svc.Build(ctx, project, opts)
	})

	stopDots()

	timedOut := errors.Is(ctx.Err(), context.DeadlineExceeded)
	cancel()

	if timedOut {
		return fmt.Errorf("docker-compose build timed out after 1 hour: %v", err)
	}

	if err != nil {
		return fmt.Errorf("docker-compose build failed: %v, output='%s', stderr='%s'", err, out, stderr)
	}

	if globalconfig.DdevVerbose && out != "" {
		util.Debug("docker-compose build output:\n%s\n\n", out)
	}

	return nil
}

// buildProjectImages runs composeBuild. Offline, BuildKit can fail to resolve
// a FROM image that an earlier build used, so a failed build falls back to the
// images from the last successful build; built is false when that happens.
func (app *DdevApp) buildProjectImages(project *composeTypes.Project, opts api.BuildOptions) (built bool, err error) {
	err = app.composeBuild(project, opts)
	if err == nil {
		return true, nil
	}
	if dockerutil.IsRegistryReachable() || !builtImagesExist(project, opts.Services) {
		return false, err
	}
	util.Debug("Unable to build project images: %v", err)
	util.Warning(`Unable to build project images while offline, using the ones from the last successful build.
Dockerfile changes take effect on the next online 'ddev start'.
See https://docs.ddev.com/en/stable/users/usage/offline/ for info.`)
	return false, nil
}

// composeUp runs docker-compose up for create.Services, or for all the
// project's enabled services when that's empty.
func composeUp(project *composeTypes.Project, create api.CreateOptions) error {
	upCtx, upSvc, err := dockerutil.NewComposeService()
	if err != nil {
		return err
	}
	progress := display.ModeQuiet
	if globalconfig.DdevVerbose {
		progress = display.ModePlain
	}
	create.Build = &api.BuildOptions{Progress: progress}
	create.RemoveOrphans = true
	return upSvc.Up(upCtx, project, api.UpOptions{
		Create: create,
		Start:  api.StartOptions{Project: project, Services: create.Services},
	})
}

// builtProfileServices returns the profile services with build: whose images
// exist locally, which a --no-cache start also rebuilds so a profile started
// later doesn't run a stale image. Profile images never built stay unbuilt, so
// unused profiles don't slow down the start.
func builtProfileServices(project *composeTypes.Project) []string {
	var services []string
	for name, service := range project.DisabledServices {
		if service.Build != nil && imageExists(project, name, service) {
			services = append(services, name)
		}
	}
	return services
}

// builtImagesExist reports whether every enabled service with a build section,
// or each of the named services, already has its image locally, so the
// project can start without building.
func builtImagesExist(project *composeTypes.Project, services []string) bool {
	for name, service := range project.Services {
		if service.Build == nil || (len(services) > 0 && !slices.Contains(services, name)) {
			continue
		}
		if !imageExists(project, name, service) {
			return false
		}
	}
	return true
}

func imageExists(project *composeTypes.Project, name string, service composeTypes.ServiceConfig) bool {
	service.Name = name
	exists, err := dockerutil.ImageExistsLocally(api.GetImageNameOrDefault(service, project.Name))
	return err == nil && exists
}
