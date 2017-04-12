package cmd

import (
	"fmt"
	"testing"

	"github.com/drud/drud-go/utils/system"
	"github.com/stretchr/testify/assert"
)

// TestDevRm runs `drud legacy rm` on the test apps
func TestDevRm(t *testing.T) {
	assert := assert.New(t)
	for _, site := range DevTestSites {
		_ = site.Chdir()

		args := []string{"rm"}
		out, err := system.RunCommand(DdevBin, args)
		assert.NoError(err)

		app, err := getActiveApp()
		if err != nil {
			assert.Fail("Could not find an active ddev configuration: %v", err)
		}
		format := fmt.Sprintf
		assert.Contains(string(out), format("Stopping %s-web ... done", app.ContainerName()))
		assert.Contains(string(out), format("Stopping %s-db ... done", app.ContainerName()))
		assert.Contains(string(out), format("Stopping %s-dba ... done", app.ContainerName()))
		assert.Contains(string(out), format("Removing %s-web ... done", app.ContainerName()))
		assert.Contains(string(out), format("Removing %s-db ... done", app.ContainerName()))
		assert.Contains(string(out), format("Removing %s-dba ... done", app.ContainerName()))
	}
}
