package ddevapp_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
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
	profiles := []string{"busybox1", "busybox2"}
	err = app.StartOptionalProfiles(profiles)
	require.NoError(t, err)
	for _, prof := range profiles {
		container, err = ddevapp.GetContainer(app, prof)
		require.NoError(t, err)
		require.NotNil(t, container)
	}
}
