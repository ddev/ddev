package cmd

import (
	"fmt"
	"testing"

	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/plugins/platform"
	"github.com/stretchr/testify/assert"
)

// TestDevRestart runs `drud legacy restart` on the test apps
func TestDevRestart(t *testing.T) {
	assert := assert.New(t)
	containerPrefix := "ddev"
	for _, site := range DevTestSites {
		cleanup := site.Chdir()

		args := []string{"restart"}
		out, err := exec.RunCommand(DdevBin, args)
		assert.NoError(err)

		app, err := platform.GetActiveApp("")
		if err != nil {
			assert.Fail("Could not find an active ddev configuration: %v", err)
		}

		format := fmt.Sprintf
		assert.Contains(string(out), format("Stopping %s-%s-web", containerPrefix, app.GetName()))
		assert.Contains(string(out), format("Stopping %s-%s-db", containerPrefix, app.GetName()))
		assert.Contains(string(out), format("Stopping %s-%s-dba", containerPrefix, app.GetName()))
		assert.Contains(string(out), format("Starting %s-%s-web", containerPrefix, app.GetName()))
		assert.Contains(string(out), format("Starting %s-%s-db", containerPrefix, app.GetName()))
		assert.Contains(string(out), format("Starting %s-%s-dba", containerPrefix, app.GetName()))
		assert.Contains(string(out), "Your application can be reached at")
		assert.Contains(string(out), app.URL())

		cleanup()
	}
}
