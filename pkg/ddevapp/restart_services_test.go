package ddevapp_test

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/moby/moby/client"
	"github.com/stretchr/testify/require"
)

// TestRestartServices checks that RestartServices gives only the named
// started services new containers, an exited one included, refuses one that
// wasn't started or a db the volume doesn't match, leaves a dependency's
// container alone, applies compose file and Dockerfile edits, rebuilds with
// NoCache, and keeps the profile services running through the full restart
// for web.
// See https://github.com/ddev/ddev/issues/7904
func TestRestartServices(t *testing.T) {
	site := TestSites[0]

	app, err := ddevapp.NewApp(site.Dir, false)
	require.NoError(t, err)

	composePath := app.GetConfigPath("docker-compose.restart-services.yaml")
	writeCompose := func(value string) {
		err := os.WriteFile(composePath, fmt.Appendf(nil, `services:
  busybox1:
    build:
      context: .
      dockerfile_inline: |
        FROM busybox:stable
        RUN cat /proc/sys/kernel/random/uuid > /built
        ENV RESTART_BUILT=%[1]s
    init: true
    command: tail -f /dev/null
    environment:
      - RESTART_TEST=%[1]s
    depends_on:
      - busybox2
    profiles:
      - busybox
    container_name: ddev-${DDEV_SITENAME}-busybox1
  busybox2:
    image: busybox:stable
    init: true
    command: tail -f /dev/null
    profiles:
      - busybox
    container_name: ddev-${DDEV_SITENAME}-busybox2
`, value), 0644)
		require.NoError(t, err)
	}
	containerID := func(service string) string {
		c, err := app.FindContainerByType(service)
		require.NoError(t, err)
		require.NotNil(t, c, "no %s container", service)
		return c.ID
	}
	busyboxExec := func(cmd ...string) string {
		out, _, err := app.Exec(&ddevapp.ExecOpts{Service: "busybox1", RawCmd: cmd})
		require.NoError(t, err)
		return strings.TrimSpace(out)
	}
	busyboxEnv := func() string {
		return busyboxExec("sh", "-c", `echo "$RESTART_TEST $RESTART_BUILT"`)
	}
	busyboxBuilt := func() string {
		return busyboxExec("cat", "/built")
	}

	t.Cleanup(func() {
		_ = app.Stop(true, false)
		_ = os.RemoveAll(composePath)
		_ = dockerutil.RemoveImage(app.GetComposeProjectName() + "-busybox1:latest")
	})
	writeCompose("first")

	err = app.Stop(false, false)
	require.NoError(t, err)
	err = app.RestartServices([]string{"db"}, ddevapp.StartOptions{})
	require.ErrorContains(t, err, "service db is not running, use 'ddev start' to start it")

	err = app.Start()
	require.NoError(t, err)
	webID := containerID("web")
	dbID := containerID("db")

	origDB := app.Database
	app.Database = ddevapp.DatabaseDesc{Type: nodeps.MySQL, Version: nodeps.MySQL80}
	require.NotEqual(t, origDB, app.Database)
	err = app.RestartServices([]string{"db"}, ddevapp.StartOptions{})
	app.Database = origDB
	require.ErrorContains(t, err, "is configured for database mysql:8.0")
	require.Equal(t, dbID, containerID("db"))

	err = app.RestartServices([]string{"db"}, ddevapp.StartOptions{})
	require.NoError(t, err)
	require.NotEqual(t, dbID, containerID("db"))
	require.Equal(t, webID, containerID("web"))
	dbID = containerID("db")

	// A crashed service leaves an exited container behind.
	ctx, apiClient, err := dockerutil.GetDockerClient()
	require.NoError(t, err)
	timeout := 30
	_, err = apiClient.ContainerStop(ctx, dbID, client.ContainerStopOptions{Timeout: &timeout})
	require.NoError(t, err)
	err = app.RestartServices([]string{"db"}, ddevapp.StartOptions{})
	require.NoError(t, err)
	require.NotEqual(t, dbID, containerID("db"))
	dbID = containerID("db")

	err = app.RestartServices([]string{"db", "busybox1"}, ddevapp.StartOptions{})
	require.ErrorContains(t, err, "service busybox1 is not running, use 'ddev start --profiles=busybox'")
	require.Equal(t, dbID, containerID("db"))

	err = app.StartOptionalProfiles([]string{"busybox"})
	require.NoError(t, err)
	webID = containerID("web")
	busybox2ID := containerID("busybox2")
	err = app.RestartServices([]string{"db", "busybox1"}, ddevapp.StartOptions{})
	require.NoError(t, err)
	require.NotEqual(t, dbID, containerID("db"))
	require.Equal(t, busybox2ID, containerID("busybox2"))
	require.Equal(t, "first first", busyboxEnv())
	built := busyboxBuilt()

	writeCompose("second")
	err = app.RestartServices([]string{"busybox1"}, ddevapp.StartOptions{})
	require.NoError(t, err)
	require.Equal(t, "second second", busyboxEnv())
	require.Equal(t, webID, containerID("web"))
	require.Equal(t, busybox2ID, containerID("busybox2"))
	require.Equal(t, built, busyboxBuilt())

	err = app.RestartServices([]string{"busybox1"}, ddevapp.StartOptions{NoCache: true})
	require.NoError(t, err)
	require.NotEqual(t, built, busyboxBuilt())
	require.Equal(t, webID, containerID("web"))

	err = app.RestartServices([]string{"nonexistent"}, ddevapp.StartOptions{})
	require.Error(t, err)
	require.Equal(t, webID, containerID("web"))

	busyboxID := containerID("busybox1")
	err = app.RestartServices([]string{"web"}, ddevapp.StartOptions{})
	require.NoError(t, err)
	require.NotEqual(t, webID, containerID("web"))
	require.NotEqual(t, busyboxID, containerID("busybox1"))
	require.NotEqual(t, busybox2ID, containerID("busybox2"))
}
