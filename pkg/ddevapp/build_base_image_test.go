package ddevapp

import (
	"os"
	"path/filepath"
	"testing"

	composeTypes "github.com/compose-spec/compose-go/v2/types"
	"github.com/stretchr/testify/require"
)

// TestResolveBuildBaseImages covers the Dockerfile/dockerfile_inline resolution
// that replaced deriving the base image from the service's "-built" image tag.
func TestResolveBuildBaseImages(t *testing.T) {
	t.Run("no build section returns nil", func(t *testing.T) {
		require.Nil(t, resolveBuildBaseImages(composeTypes.ServiceConfig{Image: "foo"}))
	})

	t.Run("dockerfile_inline with ARG default and compose override", func(t *testing.T) {
		service := composeTypes.ServiceConfig{
			Build: &composeTypes.BuildConfig{
				DockerfileInline: "ARG YOUR_DOCKER_IMAGE=\"scratch\"\nFROM ${YOUR_DOCKER_IMAGE}\nRUN true\n",
				Args:             composeTypes.NewMappingWithEquals([]string{"YOUR_DOCKER_IMAGE=ubuntu:24.04"}),
			},
		}
		require.Equal(t, []string{"ubuntu:24.04"}, resolveBuildBaseImages(service))
	})

	t.Run("dockerfile_inline falls back to the ARG default when compose doesn't override", func(t *testing.T) {
		service := composeTypes.ServiceConfig{
			Build: &composeTypes.BuildConfig{
				DockerfileInline: "ARG YOUR_DOCKER_IMAGE=\"scratch\"\nFROM ${YOUR_DOCKER_IMAGE}\n",
			},
		}
		require.Equal(t, []string{"scratch"}, resolveBuildBaseImages(service))
	})

	t.Run("dockerfile_inline resolves an argument supplied from the project environment", func(t *testing.T) {
		service := composeTypes.ServiceConfig{
			Build: &composeTypes.BuildConfig{
				DockerfileInline: "ARG BASE_IMAGE\nFROM ${BASE_IMAGE}\n",
				Args:             composeTypes.NewMappingWithEquals([]string{"BASE_IMAGE"}),
			},
		}
		environment := composeTypes.Mapping{"BASE_IMAGE": "alpine:3.20"}
		require.Equal(t, []string{"alpine:3.20"}, resolveBuildBaseImagesWithEnvironment(service, environment))
	})

	t.Run("multi-stage build excludes internal stage references", func(t *testing.T) {
		service := composeTypes.ServiceConfig{
			Build: &composeTypes.BuildConfig{
				DockerfileInline: "FROM golang:1.25 AS builder\nRUN go build -o /app\nFROM alpine:3.20\nCOPY --from=builder /app /app\n",
			},
		}
		require.Equal(t, []string{"golang:1.25", "alpine:3.20"}, resolveBuildBaseImages(service))
	})

	t.Run("multi-stage build retains an image whose name is used by a later stage", func(t *testing.T) {
		service := composeTypes.ServiceConfig{
			Build: &composeTypes.BuildConfig{
				DockerfileInline: "FROM alpine\nFROM busybox AS alpine\n",
			},
		}
		require.Equal(t, []string{"alpine", "busybox"}, resolveBuildBaseImages(service))
	})

	t.Run("Dockerfile read from build context on disk", func(t *testing.T) {
		dir := t.TempDir()
		err := os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM debian\n"), 0644)
		require.NoError(t, err)
		service := composeTypes.ServiceConfig{
			Build: &composeTypes.BuildConfig{Context: dir},
		}
		require.Equal(t, []string{"debian"}, resolveBuildBaseImages(service))
	})

	t.Run("Dockerfile on disk with ARG default and compose override", func(t *testing.T) {
		dir := t.TempDir()
		err := os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("ARG BASE_IMAGE=\"scratch\"\nFROM ${BASE_IMAGE}\n"), 0644)
		require.NoError(t, err)
		service := composeTypes.ServiceConfig{
			Build: &composeTypes.BuildConfig{
				Context: dir,
				Args:    composeTypes.NewMappingWithEquals([]string{"BASE_IMAGE=ubuntu:24.04"}),
			},
		}
		require.Equal(t, []string{"ubuntu:24.04"}, resolveBuildBaseImages(service))
	})

	t.Run("Dockerfile on disk with a custom dockerfile filename", func(t *testing.T) {
		dir := t.TempDir()
		err := os.WriteFile(filepath.Join(dir, "Dockerfile.custom"), []byte("FROM golang:1.25 AS builder\nRUN go build -o /app\nFROM alpine:3.20\nCOPY --from=builder /app /app\n"), 0644)
		require.NoError(t, err)
		service := composeTypes.ServiceConfig{
			Build: &composeTypes.BuildConfig{Context: dir, Dockerfile: "Dockerfile.custom"},
		}
		require.Equal(t, []string{"golang:1.25", "alpine:3.20"}, resolveBuildBaseImages(service))
	})

	t.Run("unreadable context falls back to nil", func(t *testing.T) {
		service := composeTypes.ServiceConfig{
			Build: &composeTypes.BuildConfig{Context: filepath.Join(t.TempDir(), "does-not-exist")},
		}
		require.Nil(t, resolveBuildBaseImages(service))
	})

	t.Run("unparseable Dockerfile falls back to nil", func(t *testing.T) {
		service := composeTypes.ServiceConfig{
			Build: &composeTypes.BuildConfig{DockerfileInline: "RUN echo no FROM at all\n"},
		}
		require.Nil(t, resolveBuildBaseImages(service))
	})
}

func TestDescribeImageForService(t *testing.T) {
	t.Run("prefers the resolved build base image", func(t *testing.T) {
		service := composeTypes.ServiceConfig{
			Build: &composeTypes.BuildConfig{DockerfileInline: "FROM ubuntu:24.04\n"},
		}
		require.Equal(t, "ubuntu:24.04", describeImageForService(service, "ubuntu:24.04-myproject-svc1-built", "myproject"))
	})

	t.Run("joins multiple resolved base images", func(t *testing.T) {
		service := composeTypes.ServiceConfig{
			Build: &composeTypes.BuildConfig{DockerfileInline: "FROM golang:1.25 AS builder\nFROM alpine:3.20\n"},
		}
		require.Equal(t, "golang:1.25, alpine:3.20", describeImageForService(service, "whatever", "myproject"))
	})

	t.Run("falls back to trimming the tag suffix when there's no build section", func(t *testing.T) {
		service := composeTypes.ServiceConfig{Image: "ubuntu:24.04-myproject-built"}
		require.Equal(t, "ubuntu:24.04", describeImageForService(service, "ubuntu:24.04-myproject-built", "myproject"))
	})
}

func TestFindServiceImagesResolvesBuildArgsFromProjectEnvironment(t *testing.T) {
	service := composeTypes.ServiceConfig{
		Image: "local-build-tag",
		Build: &composeTypes.BuildConfig{
			DockerfileInline: "ARG BASE_IMAGE\nFROM ${BASE_IMAGE}\n",
			Args:             composeTypes.NewMappingWithEquals([]string{"BASE_IMAGE"}),
		},
	}
	app := &DdevApp{
		ComposeYaml: &composeTypes.Project{
			Environment: composeTypes.Mapping{"BASE_IMAGE": "alpine:3.20"},
			Services:    composeTypes.Services{"custom": service},
		},
	}
	images, err := app.FindAllImages()
	require.NoError(t, err)
	require.Equal(t, []string{"alpine:3.20"}, images)
}
