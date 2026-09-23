package ddevapp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"

	composeTypes "github.com/compose-spec/compose-go/v2/types"
	"github.com/containerd/platforms"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/distribution/reference"
	"github.com/docker/compose/v5/pkg/api"
	"github.com/moby/buildkit/client/llb/sourceresolver"
	"github.com/moby/buildkit/frontend/dockerfile/dockerfile2llb"
	digest "github.com/opencontainers/go-digest"
	ocispecs "github.com/opencontainers/image-spec/specs-go/v1"
)

// describeServiceImage names a service's image for `ddev describe`: its build
// base images when it has any, "scratch" for a build that starts from nothing,
// otherwise its image without the "-built" suffix.
func (app *DdevApp) describeServiceImage(serviceName, image string) string {
	if baseImages := app.buildBaseImages(serviceName); baseImages != nil {
		if len(baseImages) == 0 {
			return "scratch"
		}
		return strings.Join(baseImages, ", ")
	}
	return strings.TrimSuffix(image, fmt.Sprintf("-%s-built", app.Name))
}

// buildBaseImages returns resolveBuildBaseImages for one of the project's
// services, asking the Docker daemon for its platform only for a build service.
func (app *DdevApp) buildBaseImages(serviceName string) []string {
	if app.ComposeYaml == nil || app.ComposeYaml.Services[serviceName].Build == nil {
		return nil
	}
	return resolveBuildBaseImages(app.ComposeYaml, serviceName, dockerDaemonPlatform())
}

// resolveBuildBaseImages returns the images a compose service's `build:`
// section pulls, read from its Dockerfile instead of guessed from its `image:`
// tag. BuildKit's own Dockerfile frontend does the resolution, so the result
// matches a real build. It returns nil when there is no build section or the
// Dockerfile can't be read or parsed, and an empty slice for `FROM scratch`.
// daemon is the Docker daemon's platform, which BuildKit builds on.
func resolveBuildBaseImages(project *composeTypes.Project, serviceName string, daemon ocispecs.Platform) []string {
	service := project.Services[serviceName]
	if service.Build == nil {
		return nil
	}
	content, err := readDockerfile(service.Build)
	if err != nil {
		return nil
	}
	targets, err := buildTargetPlatforms(project, service, daemon)
	if err != nil {
		return nil
	}

	recorder := &baseImageRecorder{images: []string{}, contexts: service.Build.AdditionalContexts, services: serviceImages(project)}
	for _, target := range targets {
		_, err := dockerfile2llb.Dockerfile2LLB(context.Background(), content, dockerfile2llb.ConvertOpt{
			BuildArgs:      service.Build.Args.ToMapping(),
			Target:         service.Build.Target,
			BuildPlatforms: []ocispecs.Platform{daemon},
			TargetPlatform: &target,
			MetaResolver:   recorder,
		})
		if err != nil {
			return nil
		}
	}
	slices.Sort(recorder.images)
	return slices.Compact(recorder.images)
}

// buildTargetPlatforms returns the platforms compose builds a service for, in
// compose's order: `build.platforms`, then DOCKER_DEFAULT_PLATFORM, then the
// service's `platform:`, then the daemon's own.
func buildTargetPlatforms(project *composeTypes.Project, service composeTypes.ServiceConfig, daemon ocispecs.Platform) ([]ocispecs.Platform, error) {
	names := []string(service.Build.Platforms)
	if len(names) == 0 {
		if platform := project.Environment["DOCKER_DEFAULT_PLATFORM"]; platform != "" {
			names = []string{platform}
		} else if service.Platform != "" {
			names = []string{service.Platform}
		}
	}
	if len(names) == 0 {
		return []ocispecs.Platform{daemon}, nil
	}
	return platforms.ParseAll(names)
}

// dockerDaemonPlatform returns the Docker daemon's platform, which differs
// from the host's on macOS and Windows, under Rosetta, and with a remote
// DOCKER_HOST.
func dockerDaemonPlatform() ocispecs.Platform {
	platform := ocispecs.Platform{OS: "linux", Architecture: runtime.GOARCH}
	if version, err := dockerutil.GetServerVersion(); err == nil && version.Os != "" && version.Arch != "" {
		platform.OS, platform.Architecture = version.Os, version.Arch
	}
	return platforms.Normalize(platform)
}

// serviceImages maps each of the project's services to its image, which an
// `additional_contexts` entry of "service:<name>" builds from.
func serviceImages(project *composeTypes.Project) map[string]string {
	images := map[string]string{}
	for name, service := range project.Services {
		service.Name = name
		if image, err := familiarImage(api.GetImageNameOrDefault(service, project.Name)); err == nil {
			images[name] = image
		}
	}
	return images
}

// builtImages returns the tags the project's build services produce, which
// another service can build from but no registry has.
func builtImages(project *composeTypes.Project) map[string]bool {
	images := map[string]bool{}
	all := serviceImages(project)
	for name, service := range project.Services {
		if image, ok := all[name]; ok && service.Build != nil {
			images[image] = true
		}
	}
	return images
}

// familiarImage returns an image reference in its short form with a tag, so
// "alpine", "alpine:latest" and "docker.io/library/alpine" compare equal.
func familiarImage(ref string) (string, error) {
	named, err := reference.ParseNormalizedNamed(ref)
	if err != nil {
		return "", err
	}
	return reference.FamiliarString(reference.TagNameOnly(named)), nil
}

// readDockerfile returns a build's inline Dockerfile, or reads the one it
// names, which compose resolves against the build context.
func readDockerfile(build *composeTypes.BuildConfig) ([]byte, error) {
	if build.DockerfileInline != "" {
		return []byte(build.DockerfileInline), nil
	}
	dockerfile := build.Dockerfile
	if !filepath.IsAbs(dockerfile) {
		dockerfile = filepath.Join(build.Context, dockerfile)
	}
	return os.ReadFile(dockerfile)
}

// baseImageRecorder stands in for BuildKit's registry lookup: it records each
// image the Dockerfile needs and answers with an empty config, so nothing is
// fetched. BuildKit calls it from several goroutines at once.
type baseImageRecorder struct {
	mu       sync.Mutex
	images   []string
	contexts composeTypes.Mapping
	services map[string]string
}

func (r *baseImageRecorder) ResolveImageConfig(_ context.Context, ref string, _ sourceresolver.Opt) (string, digest.Digest, []byte, error) {
	image, err := familiarImage(ref)
	if err != nil {
		return "", "", nil, err
	}
	// A compose `additional_contexts` entry replaces the image of that name,
	// matched without the ":latest" BuildKit adds to an untagged name, with
	// another service's image, a docker-image:// one, or a local directory.
	if source, ok := r.contexts[strings.TrimSuffix(image, ":latest")]; ok {
		if name, ok := strings.CutPrefix(source, "service:"); ok {
			image = r.services[name]
		} else if source, ok = strings.CutPrefix(source, "docker-image://"); ok {
			if image, err = familiarImage(source); err != nil {
				return "", "", nil, err
			}
		} else {
			image = ""
		}
	}
	if image == "" {
		return ref, "", []byte("{}"), nil
	}
	r.mu.Lock()
	r.images = append(r.images, image)
	r.mu.Unlock()
	return ref, "", []byte("{}"), nil
}
