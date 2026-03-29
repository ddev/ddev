package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	ddevImages "github.com/ddev/ddev/pkg/docker"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/stretchr/testify/require"
)

// TestDeleteCmd ensures that `ddev delete` removes expected data
func TestDeleteCmd(t *testing.T) {
	origDir, _ := os.Getwd()
	site := TestSites[0]
	err := os.Chdir(site.Dir)
	require.NoError(t, err)
	app, err := ddevapp.GetActiveApp("")
	require.NoError(t, err)
	t.Setenv("NO_COLOR", "true")

	t.Cleanup(func() {
		_, _ = exec.RunHostCommand(DdevBin, "add-on", "remove", "busybox")
		_, _ = exec.RunHostCommand(DdevBin, "delete", "-Oy", site.Name)
		// And register the project again in the global list for other tests
		out, err := exec.RunHostCommand(DdevBin, "config", "--auto")
		require.NoError(t, err, "output='%s'", out)
		_ = os.Chdir(origDir)
	})

	out, err := exec.RunHostCommand(DdevBin, "add-on", "get", filepath.Join(origDir, "testdata", t.Name(), "busybox"))
	require.NoError(t, err, "failed to add-on get, out=%s, err=%v", out, err)

	out, err = exec.RunHostCommand(DdevBin, "start", "-y")
	require.NoError(t, err, "failed to start, out=%s, err=%v", out, err)

	out, err = exec.RunHostCommand(DdevBin, "delete", "-Oy", app.Name)
	require.NoError(t, err, "failed to delete, out=%s, err=%v", out, err)

	// Check that volumes are deleted
	vols := []string{
		"third-party-tmp-busybox-volume",
	}
	if app.Database.Type == nodeps.Postgres {
		vols = append(vols, app.GetPostgresVolumeName())
	} else {
		vols = append(vols, app.GetMariaDBVolumeName())
	}
	if app.IsMutagenEnabled() {
		vols = append(vols, ddevapp.GetMutagenVolumeName(app))
	}

	for _, volName := range vols {
		require.Contains(t, out, fmt.Sprintf("Volume %s for project %s was deleted", volName, app.Name))
		require.False(t, dockerutil.VolumeExists(volName))
	}

	// Check that images are deleted
	imgs := []string{
		"ddev-" + strings.ToLower(app.Name) + "-busybox:latest",
		app.GetDBImage() + "-" + app.Name + "-built",
		ddevImages.GetWebImage() + "-" + app.Name + "-built",
	}
	for _, img := range imgs {
		require.Contains(t, out, fmt.Sprintf("%s for project %s was deleted", img, app.Name))
	}

	labels := map[string]string{
		"com.docker.compose.project": app.GetComposeProjectName(),
	}
	images, err := dockerutil.FindImagesByLabels(labels, false)
	require.NoError(t, err)
	require.Len(t, images, 0)
}

// TestOmitSnapshotOnDeleteGlobal tests the omit_snapshot_on_delete global config option
// and its interaction with the --omit-snapshot flag on 'ddev delete'.
func TestOmitSnapshotOnDeleteGlobal(t *testing.T) {
	origDir, _ := os.Getwd()
	site := TestSites[0]
	err := os.Chdir(site.Dir)
	require.NoError(t, err)
	t.Setenv("NO_COLOR", "true")

	// Create temporary XDG_CONFIG_HOME for isolated global config testing
	tmpXdgConfigHomeDir := testcommon.CopyGlobalDdevDir(t)

	t.Cleanup(func() {
		_ = os.Chdir(origDir)
		testcommon.ResetGlobalDdevDir(t, tmpXdgConfigHomeDir)
		// Ensure the project is started again for other tests
		out, err := exec.RunHostCommand(DdevBin, "config", "--auto")
		require.NoError(t, err, "output='%s'", out)
		out, err = exec.RunHostCommand(DdevBin, "start", "-y")
		require.NoError(t, err, "output='%s'", out)
	})

	tests := []struct {
		name                 string
		omitSnapshotOnDelete bool
		omitSnapshotFlag     string // "", "true", or "false"
		expectSnapshot       bool
	}{
		{
			name:                 "global_false_no_flag",
			omitSnapshotOnDelete: false,
			omitSnapshotFlag:     "",
			expectSnapshot:       true,
		},
		{
			name:                 "global_false_flag_true",
			omitSnapshotOnDelete: false,
			omitSnapshotFlag:     "true",
			expectSnapshot:       false,
		},
		{
			name:                 "global_true_no_flag",
			omitSnapshotOnDelete: true,
			omitSnapshotFlag:     "",
			expectSnapshot:       false,
		},
		{
			name:                 "global_true_flag_false",
			omitSnapshotOnDelete: true,
			omitSnapshotFlag:     "false",
			expectSnapshot:       true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Set up the global config for this test case
			globalconfig.EnsureGlobalConfig()
			globalconfig.DdevGlobalConfig.OmitSnapshotOnDelete = tc.omitSnapshotOnDelete
			err := globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig)
			require.NoError(t, err)

			// Ensure the project is started so snapshot can be taken if needed
			out, err := exec.RunHostCommand(DdevBin, "start", "-y")
			require.NoError(t, err, "failed to start, out=%s", out)

			// Clean up any pre-existing snapshots so we can detect new ones
			app, err := ddevapp.GetActiveApp("")
			require.NoError(t, err)
			snapshotsBefore, err := app.ListSnapshotNames()
			require.NoError(t, err)

			// Build the delete command
			args := []string{"delete", "--yes"}
			if tc.omitSnapshotFlag != "" {
				args = append(args, "--omit-snapshot="+tc.omitSnapshotFlag)
			}
			args = append(args, site.Name)

			out, err = exec.RunHostCommand(DdevBin, args...)
			require.NoError(t, err, "failed to delete, out=%s", out)

			if tc.expectSnapshot {
				require.Contains(t, out, "Creating database snapshot")
			} else {
				require.NotContains(t, out, "Creating database snapshot")
			}

			// Verify by checking actual snapshot count
			// Re-configure and start to check snapshots (project was deleted)
			outConfig, err := exec.RunHostCommand(DdevBin, "config", "--auto")
			require.NoError(t, err, "output='%s'", outConfig)
			outStart, err := exec.RunHostCommand(DdevBin, "start", "-y")
			require.NoError(t, err, "output='%s'", outStart)

			app, err = ddevapp.GetActiveApp("")
			require.NoError(t, err)
			snapshotsAfter, err := app.ListSnapshotNames()
			require.NoError(t, err)

			if tc.expectSnapshot {
				require.Greater(t, len(snapshotsAfter), len(snapshotsBefore), "expected a new snapshot to be created")
			} else {
				require.Equal(t, len(snapshotsBefore), len(snapshotsAfter), "expected no new snapshot to be created")
			}
		})
	}
}
