package ddevapp_test

import (
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/util"
	"github.com/ddev/ddev/pkg/versionconstants"
	"github.com/stretchr/testify/require"
)

func TestStopRemoveDataRemovesStaleUtilityContainers(t *testing.T) {
	// GlobalDdevLabels() has an empty com.ddev.site-name, like a real stale container.
	app, volumeName := newAppWithDBVolumeHeldBy(t, dockerutil.GlobalDdevLabels())

	require.NoError(t, app.Stop(true, false))

	require.False(t, dockerutil.VolumeExists(volumeName))
	stale, err := dockerutil.FindContainersByLabels(map[string]string{"com.ddev.test-holder": app.Name})
	require.NoError(t, err)
	require.Empty(t, stale)
}

func TestStopRemoveDataKeepsProjectListedWhenVolumeIsHeld(t *testing.T) {
	app, volumeName := newAppWithDBVolumeHeldBy(t, map[string]string{"com.ddev.platform": "not-ddev"})
	require.NoError(t, app.UpdateGlobalProjectList())
	t.Cleanup(func() {
		_ = globalconfig.RemoveProjectInfo(app.Name)
	})

	err := app.Stop(true, false)

	require.ErrorContains(t, err, volumeName)
	require.True(t, dockerutil.VolumeExists(volumeName))
	require.NotNil(t, globalconfig.GetProject(app.Name), "project must stay listed so ddev delete can be retried")
}

func newAppWithDBVolumeHeldBy(t *testing.T, labels map[string]string) (*ddevapp.DdevApp, string) {
	t.Helper()
	app, err := ddevapp.NewApp(t.TempDir(), true)
	require.NoError(t, err)
	app.Name = "tst-held-" + strings.ToLower(util.RandString(5))
	volumeName := app.GetMariaDBVolumeName()
	_, err = dockerutil.CreateVolume(volumeName, "local", map[string]string{}, nil)
	require.NoError(t, err)
	labels["com.ddev.test-holder"] = app.Name
	t.Cleanup(func() {
		_ = dockerutil.RemoveContainersByLabels(map[string]string{"com.ddev.test-holder": app.Name})
		_ = dockerutil.RemoveVolume(volumeName)
	})
	_, _, err = dockerutil.RunSimpleContainer(versionconstants.UtilitiesImage, "GetExistingDBType-"+app.Name, []string{"true"}, nil, nil, []string{volumeName + ":/var/lib/mysql"}, "0", false, false, labels, nil, nil)
	require.NoError(t, err)
	return app, volumeName
}
