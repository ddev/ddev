package cmd

import (
	"testing"

	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/testinteraction"
	asrt "github.com/stretchr/testify/assert"
)

// TestDdevStart runs `ddev start` on the test apps
func TestDdevStart(t *testing.T) {
	assert := asrt.New(t)

	// Make sure we have running sites.
	addSites()

	// Stop all sites.
	_, err := exec.RunCommand(DdevBin, []string{"stop", "--all"})
	assert.NoError(err)

	// Ensure all sites are started after ddev start --all.
	out, err := exec.RunCommand(DdevBin, []string{"start", "--all"})
	assert.NoError(err, "ddev start --all should succeed but failed, err: %v, output: %s", err, out)

	// Confirm all sites are running.
	for _, site := range DevTestSites {
		var app *ddevapp.DdevApp
		app, err = ddevapp.NewApp(site.Dir, ddevapp.DefaultProviderName)
		assert.NoError(err)

		// Ensure site interactivity
		interactor := testinteraction.NewInteractor(app)
		if interactor != nil {
			err = interactor.Configure()
			assert.NoError(err, "Configuration failed: %v", err)

			err = interactor.Install()
			assert.NoError(err, "Installation failed: %v", err)

			err = interactor.FindContentAtPath("/", app.GetName()) // fmt.Sprintf("%s.*", app.GetName()))
			assert.NoError(err, "Failed to find content at path: %v", err)

			err = interactor.Login()
			assert.NoError(err, "Admin login failed: %v", err)
		}

		assert.True(app.SiteStatus() == ddevapp.SiteRunning, "All sites should be running, but %s status: %s", app.GetName(), app.SiteStatus())
	}

	// Stop all sites.
	_, err = exec.RunCommand(DdevBin, []string{"stop", "--all"})
	assert.NoError(err)

	// Build start command startMultipleArgs
	startMultipleArgs := []string{"start"}
	for _, site := range DevTestSites {
		var app *ddevapp.DdevApp
		app, err = ddevapp.NewApp(site.Dir, ddevapp.DefaultProviderName)
		assert.NoError(err)

		startMultipleArgs = append(startMultipleArgs, app.GetName())
	}

	// Start multiple projects in one command
	out, err = exec.RunCommand(DdevBin, startMultipleArgs)
	assert.NoError(err, "ddev start with multiple project names should have succeeded, but failed, err: %v, output %s", err, out)

	// Confirm all sites are running
	for _, site := range DevTestSites {
		app, err := ddevapp.NewApp(site.Dir, ddevapp.DefaultProviderName)
		assert.NoError(err)

		assert.True(app.SiteStatus() == ddevapp.SiteRunning, "All sites should be running, but %s status: %s", app.GetName(), app.SiteStatus())
	}
}
