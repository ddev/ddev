package ddevapp

import (
	"os"
	"path/filepath"
	"testing"

	composeTypes "github.com/compose-spec/compose-go/v2/types"
	ocispecs "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/require"
)

// testDaemon stands in for the Docker daemon's platform, so results don't
// depend on the host running the tests.
var testDaemon = ocispecs.Platform{OS: "linux", Architecture: "amd64"}

// resolveOne resolves the base images of the only service in a project.
func resolveOne(service composeTypes.ServiceConfig, environment composeTypes.Mapping) []string {
	project := &composeTypes.Project{Name: "ddev-acme", Environment: environment, Services: composeTypes.Services{"svc": service}}
	return resolveBuildBaseImages(project, "svc", testDaemon)
}

// TestResolveBuildBaseImages checks that a build service's base images come
// from its Dockerfile, resolved the way BuildKit resolves them.
func TestResolveBuildBaseImages(t *testing.T) {
	for _, tc := range []struct {
		name       string
		dockerfile string
		args       []string
		target     string
		expected   []string
	}{
		{"ARG default", "ARG BASE=debian\nFROM $BASE\n", nil, "", []string{"debian:latest"}},
		{"build arg overrides a default used by a later ARG", "ARG A=debian\nARG B=${A}:12\nFROM $B\n", []string{"A=ubuntu"}, "", []string{"ubuntu:12"}},
		{"reachable stages and COPY --from images, deduplicated", "FROM golang:1.25 AS builder\nFROM alpine:3.20 AS unused\nFROM golang:1.25\nCOPY --from=builder /a /a\nCOPY --from=busybox:1 /b /b\n", nil, "", []string{"busybox:1", "golang:1.25"}},
		{"build target skips later stages", "FROM debian:12 AS base\nFROM alpine:3.20\n", nil, "base", []string{"debian:12"}},
		{"RUN --mount=from an image", "FROM alpine:3.20\nRUN --mount=from=busybox:1,target=/bb true\n", nil, "", []string{"alpine:3.20", "busybox:1"}},
		{"scratch has nothing to pull", "FROM scratch\n", nil, "", []string{}},
		{"scratch with COPY --from an image", "FROM scratch\nCOPY --from=busybox:1 /bin/busybox /busybox\n", nil, "", []string{"busybox:1"}},
		{"stage picked by TARGETARCH", "FROM alpine:3.20 AS base-amd64\nFROM alpine:3.19 AS base-arm64\nFROM base-${TARGETARCH}\n", nil, "", []string{"alpine:3.20"}},
		{"TARGETOS is the daemon's, not the host's", "FROM alpine:3.20 AS base-linux\nFROM base-${TARGETOS}\n", nil, "", []string{"alpine:3.20"}},
		{"unparseable Dockerfile", "RUN true\n", nil, "", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			service := composeTypes.ServiceConfig{Build: &composeTypes.BuildConfig{
				DockerfileInline: tc.dockerfile,
				Args:             composeTypes.NewMappingWithEquals(tc.args),
				Target:           tc.target,
			}}
			require.Equal(t, tc.expected, resolveOne(service, nil))
		})
	}

	t.Run("platforms in compose's order", func(t *testing.T) {
		service := composeTypes.ServiceConfig{Build: &composeTypes.BuildConfig{
			DockerfileInline: "FROM tool:$BUILDARCH AS tool\nFROM golang:1.25-$TARGETARCH\nCOPY --from=tool /t /t\n",
		}}
		require.Equal(t, []string{"golang:1.25-amd64", "tool:amd64"}, resolveOne(service, nil))
		service.Platform = "linux/arm64"
		require.Equal(t, []string{"golang:1.25-arm64", "tool:amd64"}, resolveOne(service, nil))
		service.Platform = "linux/amd64"
		require.Equal(t, []string{"golang:1.25-arm64", "tool:amd64"}, resolveOne(service, composeTypes.Mapping{"DOCKER_DEFAULT_PLATFORM": "linux/arm64"}))
		service.Build.Platforms = []string{"linux/amd64", "linux/arm64"}
		require.Equal(t, []string{"golang:1.25-amd64", "golang:1.25-arm64", "tool:amd64"}, resolveOne(service, composeTypes.Mapping{"DOCKER_DEFAULT_PLATFORM": "linux/arm64"}))
	})

	t.Run("additional contexts", func(t *testing.T) {
		service := composeTypes.ServiceConfig{Build: &composeTypes.BuildConfig{
			DockerfileInline:   "FROM base\nCOPY --from=pinned /a /a\nCOPY --from=localdir /b /b\n",
			AdditionalContexts: composeTypes.Mapping{"base": "docker-image://alpine:3.20", "pinned": "docker-image://busybox", "localdir": "./files"},
		}}
		require.Equal(t, []string{"alpine:3.20", "busybox:latest"}, resolveOne(service, nil))
	})

	t.Run("Dockerfile on disk", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "Dockerfile.custom"), []byte("FROM debian:12\n"), 0644))
		service := composeTypes.ServiceConfig{Build: &composeTypes.BuildConfig{Context: dir, Dockerfile: "Dockerfile.custom"}}
		require.Equal(t, []string{"debian:12"}, resolveOne(service, nil))
		service.Build.Dockerfile = "missing"
		require.Nil(t, resolveOne(service, nil))

		outside := filepath.Join(t.TempDir(), "Dockerfile")
		require.NoError(t, os.WriteFile(outside, []byte("FROM debian:12-slim\n"), 0644))
		service.Build.Dockerfile = outside
		require.Equal(t, []string{"debian:12-slim"}, resolveOne(service, nil))
	})

	t.Run("no build section", func(t *testing.T) {
		require.Nil(t, resolveOne(composeTypes.ServiceConfig{Image: "debian"}, nil))
	})
}

// TestBuildBaseImageCallers checks that FindAllImages and describeServiceImage,
// which both `ddev describe` sites use, report a build service by its base
// image whatever its image tag is, per #8832.
func TestBuildBaseImageCallers(t *testing.T) {
	remote := &composeTypes.BuildConfig{Context: "https://github.com/example/example.git", Dockerfile: "Dockerfile"}
	app := &DdevApp{Name: "acme", ComposeYaml: &composeTypes.Project{
		Name: "ddev-acme",
		Services: composeTypes.Services{
			"pi": {
				Image: "ddev-pi-acme-built",
				Build: &composeTypes.BuildConfig{DockerfileInline: "FROM ubuntu:24.04\n"},
			},
			// Built from other services' images, which exist only locally.
			"chained": {
				Image: "acme-chained-local-tag",
				Build: &composeTypes.BuildConfig{DockerfileInline: "FROM ddev-pi-acme-built\nCOPY --from=ddev-acme-tool /t /t\n"},
			},
			"context": {
				Image: "acme-context-local-tag",
				Build: &composeTypes.BuildConfig{DockerfileInline: "FROM base\n", AdditionalContexts: composeTypes.Mapping{"base": "service:pi"}},
			},
			"tool":    {Build: &composeTypes.BuildConfig{DockerfileInline: "FROM alpine:3.20\n"}},
			"scratch": {Image: "acme-scratch-local-tag", Build: &composeTypes.BuildConfig{DockerfileInline: "FROM scratch\n"}},
			// A remote context can't be read, so only a "-built" tag is trimmed.
			"remote":       {Image: "acme-remote-local-tag", Build: remote},
			"remote-built": {Image: "debian:12-acme-built", Build: remote},
			"db":           {Image: "mariadb:11.8-acme-built"},
		},
	}}
	images, err := app.FindAllImages()
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"ubuntu:24.04", "alpine:3.20", "debian:12", "mariadb:11.8"}, images)
	require.Equal(t, "ubuntu:24.04", app.describeServiceImage("pi", "ddev-pi-acme-built"))
	require.Equal(t, "ddev-acme-tool:latest, ddev-pi-acme-built:latest", app.describeServiceImage("chained", "acme-chained-local-tag"))
	require.Equal(t, "ddev-pi-acme-built:latest", app.describeServiceImage("context", "acme-context-local-tag"))
	require.Equal(t, "scratch", app.describeServiceImage("scratch", "acme-scratch-local-tag"))
	require.Equal(t, "mariadb:11.8", app.describeServiceImage("db", "mariadb:11.8-acme-built"))
}
