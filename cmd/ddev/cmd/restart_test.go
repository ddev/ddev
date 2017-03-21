package cmd

import (
	"fmt"
	"strings"
	"testing"

	"github.com/drud/ddev/pkg/plugins/platform"
	"github.com/drud/drud-go/utils/system"
	"github.com/stretchr/testify/assert"
)

// TestDevRm runs `drud legacy rm` on the test apps
func TestDevRm(t *testing.T) {
	if skipComposeTests {
		t.Skip("Compose tests being skipped.")
	}

	for _, site := range DevTestSites {
		cleanup := site.Chdir()

		args := []string{"restart"}
		out, err := system.RunCommand(DdevBin, args)
		assert.NoError(t, err)

		app := platform.PluginMap[strings.ToLower(plugin)]
		app.Init()
		format := fmt.Sprintf
		assert.Contains(t, string(out), format("Stopping %s-web ... done", app.ContainerName()))
		assert.Contains(t, string(out), format("Stopping %s-db ... done", app.ContainerName()))
		assert.Contains(t, string(out), format("Starting %s-web ... done", app.ContainerName()))
		assert.Contains(t, string(out), format("Starting %s-db ... done", app.ContainerName()))
		assert.Contains(t, string(out), "Your application can be reached at")
		assert.Contains(t, string(out), app.URL())
		cleanup()
	}
}
