package cmd

import (
	"os"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/stretchr/testify/require"
)

// TestDebugDeleteVolumeCmd checks that `ddev utility delete-volume`:
//   - rejects an unknown volume name and a shared/global volume name
//   - stops a running project and deletes its volume when given the volume's
//     short (docker-compose) name
//   - deletes a volume when given its full Docker name
func TestDebugDeleteVolumeCmd(t *testing.T) {
	if os.Getenv("GOTEST_SHORT") != "" {
		t.Skip("Skip because GOTEST_SHORT is set")
	}

	origDir, _ := os.Getwd()
	tmpdir := testcommon.CreateTmpDir(t.Name())

	err := os.Chdir(tmpdir)
	require.NoError(t, err)

	_, err = exec.RunHostCommand(DdevBin, "config", "--project-type=php", "--project-name=test-delete-volume")
	require.NoError(t, err)

	app, err := ddevapp.GetActiveApp("")
	require.NoError(t, err)

	expectedVolumeName := "ddev-" + app.Name + "_myvol"

	extraCompose := `services:
  myservice:
    image: busybox
    command: ["sleep", "infinity"]
    labels:
      com.ddev.site-name: ${DDEV_SITENAME}
    volumes:
      - myvol:/data
volumes:
  myvol:
    name: "` + expectedVolumeName + `"
`
	err = os.WriteFile(app.GetConfigPath("docker-compose.myvol.yaml"), []byte(extraCompose), 0644)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = app.Stop(true, false)
		_ = dockerutil.RemoveVolume(expectedVolumeName)
		_ = os.Chdir(origDir)
		_ = os.RemoveAll(tmpdir)
	})

	// Unknown volume name should fail and list the project's known volumes.
	out, err := exec.RunHostCommand(DdevBin, "utility", "delete-volume", "bogus-volume", "--yes")
	require.Error(t, err, "expected failure for unknown volume name, out='%s'", out)
	require.Contains(t, out, "No volume named")

	// A shared/global volume should be refused, not deleted.
	out, err = exec.RunHostCommand(DdevBin, "utility", "delete-volume", "ddev-global-cache", "--yes")
	require.Error(t, err, "expected failure for shared volume name, out='%s'", out)
	require.Contains(t, out, "shared volume")

	err = app.Start()
	require.NoError(t, err)
	require.True(t, dockerutil.VolumeExists(expectedVolumeName))

	status, _ := app.SiteStatus()
	require.Equal(t, ddevapp.SiteRunning, status)

	// Deleting by short name while the project is running must stop it first.
	out, err = exec.RunHostCommand(DdevBin, "utility", "delete-volume", "myvol", "--yes")
	require.NoError(t, err, "delete-volume by short name failed, out='%s'", out)
	require.Contains(t, out, expectedVolumeName)
	require.False(t, dockerutil.VolumeExists(expectedVolumeName))

	status, _ = app.SiteStatus()
	require.Equal(t, ddevapp.SiteStopped, status)

	// Deleting by full Docker name, with the project already stopped.
	_, err = dockerutil.CreateVolume(expectedVolumeName, "local", nil, nil)
	require.NoError(t, err)

	out, err = exec.RunHostCommand(DdevBin, "utility", "delete-volume", expectedVolumeName, "--yes")
	require.NoError(t, err, "delete-volume by full name failed, out='%s'", out)
	require.False(t, dockerutil.VolumeExists(expectedVolumeName))
}
