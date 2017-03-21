package cmd

import (
	"fmt"
	"strings"
	"testing"

	"github.com/drud/ddev/pkg/plugins/platform"
	"github.com/drud/drud-go/utils/system"
	"github.com/stretchr/testify/assert"
)

// TestDevRestart runs `drud legacy restart` on the test apps
func TestDevRestart(t *testing.T) {
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
		assert.Contains(t, string(out), format("Stopping %s-web", app.ContainerName()))
		assert.Contains(t, string(out), format("Stopping %s-db", app.ContainerName()))
		assert.Contains(t, string(out), format("Starting %s-web", app.ContainerName()))
		assert.Contains(t, string(out), format("Starting %s-db", app.ContainerName()))
		assert.Contains(t, string(out), "Your application can be reached at")
		assert.Contains(t, string(out), app.URL())
		cleanup()
	}
}
