package ddevapp

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"time"

	composeTypes "github.com/compose-spec/compose-go/v2/types"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/output"
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
		if !dockerutil.IsAppleContainer() {
			return svc.Build(ctx, project, opts)
		}
		// socktainer loses one of two image tags loaded at once, so a parallel
		// build can leave a service on its old image; build one at a time.
		for _, name := range appleContainerBuildServices(project, opts.Services) {
			one := opts
			one.Services = []string{name}
			if err := svc.Build(ctx, project, one); err != nil {
				return err
			}
		}
		return nil
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

// appleContainerBuildServices returns the services composeBuild builds one at
// a time on Apple Container: those named, or else every enabled service with a
// build section, sorted.
func appleContainerBuildServices(project *composeTypes.Project, named []string) []string {
	if len(named) > 0 {
		return named
	}
	var names []string
	for name, service := range project.Services {
		if service.Build != nil {
			names = append(names, name)
		}
	}
	slices.Sort(names)
	return names
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
	if !output.JSONOutput {
		// End the line the build's progress dots left open.
		_, _ = fmt.Fprintln(output.UserOut.Out)
	}
	util.Warning(`Unable to build project images while offline, using the ones from the last successful build.
Dockerfile changes take effect on the next online 'ddev start'.
See https://docs.ddev.com/en/stable/users/usage/offline/ for info.`)
	return false, nil
}

// composeUp runs docker-compose up for create.Services, or for all the
// project's enabled services when that's empty. It copies the mkcert root CA
// into web between create and start, so web's start.sh finds it in place.
func (app *DdevApp) composeUp(project *composeTypes.Project, create api.CreateOptions) error {
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
	if err = upSvc.Create(upCtx, project, create); err != nil {
		return err
	}
	if err = app.copyCAROOTIntoWeb(); err != nil {
		util.Warning("%v", err)
	}
	return upSvc.Start(upCtx, project.Name, api.StartOptions{Project: project, Services: create.Services})
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
	exists, err := dockerutil.ImageExistsLocally(builtImageName(project, name, service))
	return err == nil && exists
}

// builtImageName returns the tag of the image compose builds for a service.
func builtImageName(project *composeTypes.Project, name string, service composeTypes.ServiceConfig) string {
	service.Name = name
	return api.GetImageNameOrDefault(service, project.Name)
}

// isBuiltImage reports whether a service in the project builds image.
func isBuiltImage(project *composeTypes.Project, image string) bool {
	if project == nil || image == "" {
		return false
	}
	for name, service := range project.Services {
		if service.Build != nil && builtImageName(project, name, service) == image {
			return true
		}
	}
	return false
}

// buildTagCollisions returns, by tag, the services that build different images
// under the same tag, so they all run whichever was built last. Services with
// the same build config share an image on purpose.
func buildTagCollisions(project *composeTypes.Project) map[string][]string {
	builders := map[string][]string{}
	for _, name := range slices.Sorted(maps.Keys(project.Services)) {
		if service := project.Services[name]; service.Build != nil {
			image := builtImageName(project, name, service)
			builders[image] = append(builders[image], name)
		}
	}
	maps.DeleteFunc(builders, func(_ string, names []string) bool {
		first := project.Services[names[0]].Build
		return !slices.ContainsFunc(names[1:], func(name string) bool {
			return !reflect.DeepEqual(project.Services[name].Build, first)
		})
	})
	return builders
}
