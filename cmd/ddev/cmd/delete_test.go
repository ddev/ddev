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
		app.GetMariaDBVolumeName(),
		app.GetPostgresVolumeName(),
		"third-party-tmp-busybox-volume",
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
		require.Contains(t, out, fmt.Sprintf("Image %s for project %s was deleted", img, app.Name))
	}

	labels := map[string]string{
		"com.docker.compose.project": app.GetComposeProjectName(),
	}
	images, err := dockerutil.FindImagesByLabels(labels, false)
	require.NoError(t, err)
	require.Len(t, images, 0)
}
