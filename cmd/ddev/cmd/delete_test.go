package cmd

import (
	"fmt"
	"github.com/ddev/ddev/pkg/ddevapp"
	ddevImages "github.com/ddev/ddev/pkg/docker"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/globalconfig"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDeleteCmd ensures that `ddev delete` removes expected data
func TestDeleteCmd(t *testing.T) {
	assert := asrt.New(t)

	origDir, _ := os.Getwd()
	site := TestSites[0]
	err := os.Chdir(site.Dir)
	require.NoError(t, err)
	app, err := ddevapp.GetActiveApp("")
	require.NoError(t, err)

	t.Cleanup(func() {
		out, err := exec.RunHostCommand(DdevBin, "add-on", "remove", "busybox")
		assert.NoError(err, "output='%s'", out)
		out, err = exec.RunHostCommand(DdevBin, "delete", "-Oy", site.Name)
		assert.NoError(err, "output='%s'", out)
		// And register the project again in the global list for other tests
		out, err = exec.RunHostCommand(DdevBin, "config", "--auto")
		assert.NoError(err, "output='%s'", out)
		err = os.Chdir(origDir)
		assert.NoError(err)
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
	}
	if app.IsMutagenEnabled() {
		vols = append(vols, ddevapp.GetMutagenVolumeName(app))
	}
	if globalconfig.DdevGlobalConfig.NoBindMounts {
		vols = append(vols, app.Name+"-ddev-config")
	}
	for _, volName := range vols {
		assert.Contains(out, fmt.Sprintf("Volume %s for project %s was deleted", volName, app.Name))
		assert.False(dockerutil.VolumeExists(volName))
	}
	assert.Contains(out, "Deleting third-party persistent volume third-party-tmp-busybox-volume")

	// Check that images are deleted
	imgs := []string{
		"ddev-" + strings.ToLower(app.Name) + "-busybox:latest",
		app.GetDBImage() + "-" + app.Name + "-built",
		ddevImages.GetWebImage() + "-" + app.Name + "-built",
	}
	for _, img := range imgs {
		assert.Contains(out, fmt.Sprintf("Image %s for project %s was deleted", img, app.Name))
	}

	labels := map[string]string{
		"com.docker.compose.project": app.GetComposeProjectName(),
	}
	images, err := dockerutil.FindImagesByLabels(labels, false)
	require.NoError(t, err)
	assert.Len(images, 0)
}
