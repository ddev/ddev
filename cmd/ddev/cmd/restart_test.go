package cmd

import (
	"fmt"
	"sync"
	"testing"

	"github.com/drud/drud-go/utils/system"
	"github.com/stretchr/testify/assert"
)

// TestDevRestart runs `drud legacy restart` on the test apps
func TestDevRestart(t *testing.T) {
	assert := assert.New(t)
	var wg sync.WaitGroup
	wg.Add(len(DevTestSites))
	for _, site := range DevTestSites {
		go func() {
			cleanup := site.Chdir()

			args := []string{"restart"}
			out, err := system.RunCommand(DdevBin, args)
			assert.NoError(err)

			app, err := getActiveApp()
			if err != nil {
				assert.Fail("Could not find an active ddev configuration: %v", err)
			}

			format := fmt.Sprintf
			assert.Contains(string(out), format("Stopping %s-web", app.ContainerName()))
			assert.Contains(string(out), format("Stopping %s-db", app.ContainerName()))
			assert.Contains(string(out), format("Stopping %s-dba", app.ContainerName()))
			assert.Contains(string(out), format("Starting %s-web", app.ContainerName()))
			assert.Contains(string(out), format("Starting %s-db", app.ContainerName()))
			assert.Contains(string(out), format("Starting %s-dba", app.ContainerName()))
			assert.Contains(string(out), "Your application can be reached at")
			assert.Contains(string(out), app.URL())

			cleanup()
		}()
	}
	wg.Wait()
}
