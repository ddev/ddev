package cmd

import (
	"fmt"
	"testing"

	"github.com/drud/drud-go/utils/system"
	"github.com/stretchr/testify/assert"
)

// TestDevRestart runs `drud legacy restart` on the test apps
func TestDevRestart(t *testing.T) {
	assert := assert.New(t)
	for _, site := range DevTestSites {
		cleanup := site.Chdir()

		args := []string{"restart"}
		out, err := system.RunCommand(DdevBin, args)
		assert.NoError(err)

		app, err := getActiveApp()
		if err != nil {
			assert.Fail("Could not find an active ddev configuration: %v", err)
		}

		format := fmt.Sprintf
		assert.Contains(string(out), format("Stopping %s-%s-web", plugin, app.GetName()))
		assert.Contains(string(out), format("Stopping %s-%s-db", plugin, app.GetName()))
		assert.Contains(string(out), format("Stopping %s-%s-dba", plugin, app.GetName()))
		assert.Contains(string(out), format("Starting %s-%s-web", plugin, app.GetName()))
		assert.Contains(string(out), format("Starting %s-%s-db", plugin, app.GetName()))
		assert.Contains(string(out), format("Starting %s-%s-dba", plugin, app.GetName()))
		assert.Contains(string(out), "Your application can be reached at")
		assert.Contains(string(out), app.URL())

		cleanup()
	}
}
