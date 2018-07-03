package cmd

import (
	"testing"

	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/exec"
	asrt "github.com/stretchr/testify/assert"
)

// TestDevRemove runs `ddev rm` on the test apps
func TestDdevStop(t *testing.T) {
	assert := asrt.New(t)

	// Make sure we have running sites.
	addSites()

	for _, site := range DevTestSites {
		cleanup := site.Chdir()

		out, err := exec.RunCommand(DdevBin, []string{"stop"})
		assert.NoError(err, "ddev stop should succeed but failed, err: %v, output: %s", err, out)
		assert.Contains(out, "has been stopped")

		apps := ddevapp.GetApps()
		for _, app := range apps {
			if app.GetName() != site.Name {
				continue
			}

			assert.True(app.SiteStatus() == ddevapp.SiteStopped)
		}

		cleanup()
	}

	// Re-create running sites.
	addSites()
	out, err := exec.RunCommand(DdevBin, []string{"stop", "--all"})
	assert.NoError(err, "ddev remove --all should succeed but failed, err: %v, output: %s", err, out)

	// Confirm all sites are stopped.
	apps := ddevapp.GetApps()
	for _, app := range apps {
		assert.True(app.SiteStatus() == ddevapp.SiteStopped, "All sites should be stopped, but %s status: %s", app.GetName(), app.SiteStatus())
	}

	// Now put the sites back together so other tests can use them.
	addSites()
}
