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

		args := []string{"rm"}
		out, err := system.RunCommand(DdevBin, args)
		assert.NoError(t, err)

		app := platform.PluginMap[strings.ToLower(plugin)]
		app.Init()
		format := fmt.Sprintf
		assert.Contains(t, string(out), format("Stopping %s-web ... done", app.ContainerName()))
		assert.Contains(t, string(out), format("Stopping %s-db ... done", app.ContainerName()))
		assert.Contains(t, string(out), format("Removing %s-web ... done", app.ContainerName()))
		assert.Contains(t, string(out), format("Removing %s-db ... done", app.ContainerName()))

		cleanup()
	}
}
