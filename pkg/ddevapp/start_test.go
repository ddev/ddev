package ddevapp_test

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/util"
	"github.com/ddev/ddev/pkg/versionconstants"
	"github.com/moby/moby/client"
	"github.com/stretchr/testify/require"
)

// TestDdevApp_StartOptionalProfiles makes sure that we can start an optional service appropriately
func TestDdevApp_StartOptionalProfiles(t *testing.T) {
	origDir, _ := os.Getwd()
	site := TestSites[0]

	app, err := ddevapp.NewApp(site.Dir, false)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = app.Stop(true, false)
		// Remove the added docker-compose.busybox.yaml
		_ = os.RemoveAll(filepath.Join(app.GetConfigPath("docker-compose.busybox.yaml")))
	})

	// Add extra services with named profiles
	err = fileutil.CopyFile(filepath.Join(origDir, "testdata", t.Name(), "docker-compose.busybox.yaml"), app.GetConfigPath("docker-compose.busybox.yaml"))
	require.NoError(t, err)

	err = app.Start()
	require.NoError(t, err)

	// Make sure the busybox services didn't get started
	container, err := ddevapp.GetContainer(app, "busybox1")
	require.Error(t, err)
	require.Nil(t, container)

	container, err = ddevapp.GetContainer(app, "busybox2")
	require.Error(t, err)
	require.Nil(t, container)

	// Now StartOptionalProfiles() and make sure the service is there
	profiles := []string{"busybox-first", "busybox-second"}
	err = app.StartOptionalProfiles(profiles)
	require.NoError(t, err)
	// Map profile names to service names for verification
	profileToService := map[string]string{
		"busybox-first":  "busybox1",
		"busybox-second": "busybox2",
	}
	for _, prof := range profiles {
		serviceName := profileToService[prof]
		container, err = ddevapp.GetContainer(app, serviceName)
		require.NoError(t, err)
		require.NotNil(t, container)
	}
}

// TestStartWithExitedProfileContainer makes sure that a container left behind by a
// profile which is not active does not fail `ddev start`.
// See https://github.com/ddev/ddev/issues/8830
func TestStartWithExitedProfileContainer(t *testing.T) {
	origDir, _ := os.Getwd()
	site := TestSites[0]

	app, err := ddevapp.NewApp(site.Dir, false)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = app.Stop(true, false)
		_ = os.RemoveAll(app.GetConfigPath("docker-compose.busybox.yaml"))
	})

	err = fileutil.CopyFile(filepath.Join(origDir, "testdata", t.Name(), "docker-compose.busybox.yaml"), app.GetConfigPath("docker-compose.busybox.yaml"))
	require.NoError(t, err)

	err = app.Start()
	require.NoError(t, err)

	err = app.StartOptionalProfiles([]string{"busybox-first"})
	require.NoError(t, err)

	// Leave every project container "exited", as a restart of the Docker provider
	// does. The whole project must be down: Compose prunes the profile's leftover
	// container itself while the rest of the project is still up.
	ctx, apiClient, err := dockerutil.GetDockerClient()
	require.NoError(t, err)
	projectContainers, err := dockerutil.FindContainersByLabels(map[string]string{"com.ddev.site-name": app.Name})
	require.NoError(t, err)
	require.NotEmpty(t, projectContainers)
	timeout := 30
	for _, c := range projectContainers {
		_, err = apiClient.ContainerStop(ctx, c.ID, client.ContainerStopOptions{Timeout: &timeout})
		require.NoError(t, err)
	}
	busyboxContainer, err := dockerutil.FindContainerByName(fmt.Sprintf("ddev-%s-busybox1", app.Name))
	require.NoError(t, err)
	require.NotNil(t, busyboxContainer)
	require.Equal(t, "exited", string(busyboxContainer.State))

	// A start without the profile must not wait on, or fail because of, that container.
	err = app.Start()
	require.NoError(t, err)
}

// TestStartProfileImages checks when start and restart build, pull, and
// rebuild the images of profile-gated services.
// See https://github.com/ddev/ddev/issues/8817
func TestStartProfileImages(t *testing.T) {
	origDir, _ := os.Getwd()
	site := TestSites[0]

	app, err := ddevapp.NewApp(site.Dir, false)
	require.NoError(t, err)
	require.NoFileExists(t, app.GetConfigPath(".env"))

	xImage := app.GetComposeProjectName() + "-x1:latest"
	x2Image := app.GetComposeProjectName() + "-x2:latest"
	yImage := app.GetComposeProjectName() + "-y1:latest"
	zImage := "busybox:1.36.1-musl"
	xDockerfile := app.GetConfigPath("profile-x1/Dockerfile")
	removeImages := func() {
		for _, image := range []string{xImage, x2Image, yImage, zImage} {
			_ = dockerutil.RemoveImage(image)
		}
	}
	t.Cleanup(func() {
		_ = app.Stop(true, false)
		_ = os.RemoveAll(app.GetConfigPath("docker-compose.profile-images.yaml"))
		_ = os.RemoveAll(app.GetConfigPath("profile-x1"))
		_ = os.RemoveAll(app.GetConfigPath("profile-y1"))
		_ = os.RemoveAll(app.GetConfigPath(".env"))
		removeImages()
	})
	removeImages()

	err = fileutil.CopyFile(filepath.Join(origDir, "testdata", t.Name(), "docker-compose.profile-images.yaml"), app.GetConfigPath("docker-compose.profile-images.yaml"))
	require.NoError(t, err)
	for _, dir := range []string{"profile-x1", "profile-y1"} {
		err = fileutil.CopyDir(filepath.Join(origDir, "testdata", t.Name(), dir), app.GetConfigPath(dir))
		require.NoError(t, err)
	}

	readRandom := func(image string) string {
		_, out, err := dockerutil.RunSimpleContainer(image, "read-random-"+util.RandString(6), []string{"cat", "/random.txt"}, nil, nil, nil, "", true, false, nil, nil, nil)
		require.NoError(t, err)
		return strings.TrimSpace(out)
	}
	containerID := func(service string) string {
		c, err := ddevapp.GetContainer(app, service)
		require.NoError(t, err)
		return c.ID
	}
	requireNoImage := func(image string) {
		exists, err := dockerutil.ImageExistsLocally(image)
		require.NoError(t, err)
		require.False(t, exists, "image %s should not exist", image)
	}

	err = app.Start()
	require.NoError(t, err)
	requireNoImage(xImage)
	requireNoImage(x2Image)
	requireNoImage(yImage)
	requireNoImage(zImage)

	err = app.Stop(false, false)
	require.NoError(t, err)
	err = app.StartWith(ddevapp.StartOptions{Profiles: []string{"x"}})
	require.NoError(t, err)
	firstRandom := readRandom(xImage)
	x1ID := containerID("x1")

	err = fileutil.AppendStringToFile(xDockerfile, "RUN touch /changed.txt\n")
	require.NoError(t, err)
	err = fileutil.ReplaceStringInFile("inline-1", "inline-2", app.GetConfigPath("docker-compose.profile-images.yaml"), app.GetConfigPath("docker-compose.profile-images.yaml"))
	require.NoError(t, err)
	err = app.StartWith(ddevapp.StartOptions{Profiles: []string{"x"}})
	require.NoError(t, err)
	_, _, err = app.Exec(&ddevapp.ExecOpts{Service: "x1", Cmd: "ls /changed.txt"})
	require.NoError(t, err)
	out, _, err := app.Exec(&ddevapp.ExecOpts{Service: "x2", Cmd: "cat /inline.txt"})
	require.NoError(t, err)
	require.Equal(t, "inline-2", strings.TrimSpace(out))
	require.NotEqual(t, x1ID, containerID("x1"))
	require.Equal(t, firstRandom, readRandom(xImage))

	webID := containerID("web")
	err = app.StartWith(ddevapp.StartOptions{Profiles: []string{"z"}})
	require.NoError(t, err)
	_, err = ddevapp.GetContainer(app, "z1")
	require.NoError(t, err)
	require.Equal(t, webID, containerID("web"))

	err = app.RestartWith(ddevapp.StartOptions{NoCache: true})
	require.NoError(t, err)
	_, err = ddevapp.GetContainer(app, "x1")
	require.Error(t, err)
	require.NotEqual(t, firstRandom, readRandom(xImage))
	requireNoImage(yImage)

	err = os.WriteFile(app.GetConfigPath(".env"), []byte("COMPOSE_PROFILES=y\n"), 0644)
	require.NoError(t, err)
	err = app.Restart()
	require.NoError(t, err)
	_, err = ddevapp.GetContainer(app, "y1")
	require.NoError(t, err)
}

// TestPlatformOverride makes sure that a project which overrides the web
// service platform (e.g. `platform: linux/amd64` on an arm64 host) actually
// builds and runs the web image for the requested architecture.
// See https://github.com/ddev/ddev/issues/8578.
// It only runs on macOS arm64 + OrbStack, where cross-platform emulation is
// available and exercised in CI (Buildkite).
func TestPlatformOverride(t *testing.T) {
	if runtime.GOOS != "darwin" || runtime.GOARCH != "arm64" || !dockerutil.IsOrbStack() {
		t.Skip("Skipping TestPlatformOverride; only runs on macOS arm64 with OrbStack")
	}

	origDir, _ := os.Getwd()
	site := TestSites[0]

	app, err := ddevapp.NewApp(site.Dir, false)
	require.NoError(t, err)

	overridePath := app.GetConfigPath("docker-compose.amd64.yaml")
	t.Cleanup(func() {
		_ = app.Stop(true, false)
		_ = os.RemoveAll(overridePath)
	})

	err = fileutil.CopyFile(filepath.Join(origDir, "testdata", t.Name(), "docker-compose.amd64.yaml"), overridePath)
	require.NoError(t, err)

	err = app.Start()
	require.NoError(t, err)

	// The web container must be the amd64 (x86_64) architecture requested by the override,
	// not the arm64 host architecture.
	out, _, err := app.Exec(&ddevapp.ExecOpts{
		Cmd: "uname -m",
	})
	require.NoError(t, err)
	require.Equal(t, "x86_64", strings.TrimSpace(out))
}

// TestStartOfflineWithBuiltImages checks that a failed build stops the start
// while a registry is reachable, and falls back to the last built images when not.
func TestStartOfflineWithBuiltImages(t *testing.T) {
	origDir, _ := os.Getwd()
	site := TestSites[0]

	app, err := ddevapp.NewApp(site.Dir, false)
	require.NoError(t, err)

	origUtilitiesImage := versionconstants.UtilitiesImage
	origInternetChecked := globalconfig.IsInternetActiveAlreadyChecked
	origInternetResult := globalconfig.IsInternetActiveResult
	t.Cleanup(func() {
		versionconstants.UtilitiesImage = origUtilitiesImage
		globalconfig.IsInternetActiveAlreadyChecked = origInternetChecked
		globalconfig.IsInternetActiveResult = origInternetResult
		_ = app.Stop(true, false)
		_ = os.RemoveAll(app.GetConfigPath("docker-compose.offline-build.yaml"))
		_ = os.RemoveAll(app.GetConfigPath("offline-build"))
		_ = dockerutil.RemoveImage(app.GetComposeProjectName() + "-offline-build")
	})

	err = fileutil.CopyFile(filepath.Join(origDir, "testdata", t.Name(), "docker-compose.offline-build.yaml"), app.GetConfigPath("docker-compose.offline-build.yaml"))
	require.NoError(t, err)
	err = fileutil.CopyDir(filepath.Join(origDir, "testdata", t.Name(), "offline-build"), app.GetConfigPath("offline-build"))
	require.NoError(t, err)

	err = app.Start()
	require.NoError(t, err)

	// offline.invalid never resolves, so the rebuild fails the way it does offline.
	err = os.WriteFile(app.GetConfigPath("offline-build/Dockerfile"), []byte("FROM offline.invalid/busybox:stable\n"), 0644)
	require.NoError(t, err)

	err = app.Restart()
	require.Error(t, err)
	require.Contains(t, err.Error(), "offline.invalid")

	// A locally built image has no registry digest, so the registry check falls
	// back to the DNS check, which is faked here as offline. The never-built
	// offline-profile service must not block the fallback.
	versionconstants.UtilitiesImage = app.GetComposeProjectName() + "-offline-build:latest"
	globalconfig.IsInternetActiveAlreadyChecked = true
	globalconfig.IsInternetActiveResult = false
	err = app.Restart()
	require.NoError(t, err)

	_, _, err = app.Exec(&ddevapp.ExecOpts{
		Service: "offline-build",
		Cmd:     "ls /tmp/added-by-offline-build.txt",
	})
	require.NoError(t, err)
}

// TestBuildServiceImageTags checks that two services building from one base
// image keep their own images.
func TestBuildServiceImageTags(t *testing.T) {
	site := TestSites[0]
	app, err := ddevapp.NewApp(site.Dir, false)
	require.NoError(t, err)

	composeFile := app.GetConfigPath("docker-compose.build-tags.yaml")
	taggedImage := "busybox:1.36-" + app.Name + "-tagged-built"
	t.Cleanup(func() {
		_ = os.Remove(composeFile)
		_ = app.Stop(true, false)
		_ = dockerutil.RemoveImage(taggedImage)
		_ = dockerutil.RemoveImage(app.GetComposeProjectName() + "-no-image")
	})

	compose := `services:
  tagged:
    container_name: ddev-${DDEV_SITENAME}-tagged
    image: ${BUILD_TAGS_BASE:-busybox:1.36}-${DDEV_SITENAME}-tagged-built
    build:
      dockerfile_inline: |
        ARG BASE_IMAGE=scratch
        FROM $${BASE_IMAGE}
        RUN echo tagged > /marker.txt
      args:
        BASE_IMAGE: ${BUILD_TAGS_BASE:-busybox:1.36}
    command: sleep infinity
    init: true
  no-image:
    container_name: ddev-${DDEV_SITENAME}-no-image
    build:
      dockerfile_inline: |
        FROM busybox:1.36
        RUN echo no-image > /marker.txt
    command: sleep infinity
    init: true
    x-ddev:
      pull-images:
        - busybox:1.36
`
	err = os.WriteFile(composeFile, []byte(compose), 0644)
	require.NoError(t, err)

	restoreErr := util.CaptureUserErr()
	err = app.Start()
	stderr := restoreErr()
	require.NoError(t, err)
	require.NotContains(t, stderr, "Unable to pull")

	desc, err := app.Describe(false)
	require.NoError(t, err)
	services := desc["services"].(map[string]map[string]any)
	for service, image := range map[string]string{"tagged": "busybox:1.36", "no-image": app.GetComposeProjectName() + "-no-image"} {
		out, _, err := app.Exec(&ddevapp.ExecOpts{Service: service, Cmd: "cat /marker.txt"})
		require.NoError(t, err)
		require.Equal(t, service, strings.TrimSpace(out))
		require.Equal(t, image, services[service]["image"])
	}
}
