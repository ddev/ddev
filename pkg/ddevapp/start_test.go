package ddevapp_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/stretchr/testify/require"
)

// TestDdevApp_StartOptionalProfile makes sure that we can start an optional service appropriately
func TestDdevApp_StartOptionalProfile(t *testing.T) {

	origDir, _ := os.Getwd()
	site := TestSites[0]

	app, err := ddevapp.NewApp(site.Dir, false)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = app.Stop(true, false)
		// Remove the added docker-compose.busybox.yaml
		_ = os.RemoveAll(filepath.Join(app.GetConfigPath("docker-compose.busybox.yaml")))
	})

	// Add extra service that is in the "optional" profile
	err = fileutil.CopyFile(filepath.Join(origDir, "testdata", t.Name(), "docker-compose.busybox.yaml"), app.GetConfigPath("docker-compose.busybox.yaml"))
	require.NoError(t, err)

	err = app.Start()
	require.NoError(t, err)

	// Make sure the busybox service didn't get started
	container, err := ddevapp.GetContainer(app, "busybox")
	require.Error(t, err)
	require.Nil(t, container)

	// Now StartOptionalProfiles() and make sure the service is there
	err = app.StartOptionalProfiles([]string{"busybox"})
	require.NoError(t, err)
	container, err = ddevapp.GetContainer(app, "busybox")
	require.NoError(t, err)
	require.NotNil(t, container)
}
