package ddevapp_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/fileutil"
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
