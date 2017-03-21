package cmd

import (
	"strings"
	"testing"

	"github.com/drud/ddev/pkg/plugins/platform"
	"github.com/drud/drud-go/utils/system"
	"github.com/stretchr/testify/assert"
)

func TestDevList(t *testing.T) {
	if skipComposeTests {
		t.Skip("Compose tests being skipped.")
	}

	args := []string{"list"}
	out, err := system.RunCommand(DdevBin, args)
	assert.NoError(t, err)
	for _, v := range DevTestSites {
		cleanup := v.Chdir()

		app := platform.PluginMap[strings.ToLower(plugin)]
		app.Init()
		assert.Contains(t, string(out), v.Name)
		assert.Contains(t, string(out), app.URL())
		assert.Contains(t, string(out), app.GetType())

		cleanup()
	}

}
