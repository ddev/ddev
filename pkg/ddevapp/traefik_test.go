package ddevapp_test

import (
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/globalconfig"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

// TestTraefikSimple tests basic traefik router usage
func TestTraefikSimple(t *testing.T) {
	assert := asrt.New(t)

	// Make sure this leaves us in the original test directory
	origDir, _ := os.Getwd()

	site := TestSites[0] // 0 == wordpress
	app, err := ddevapp.NewApp(site.Dir, true)
	assert.NoError(err)

	ddevapp.PowerOff()
	origTraefik := globalconfig.DdevGlobalConfig.UseTraefik
	globalconfig.DdevGlobalConfig.UseTraefik = true
	err = globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig)
	require.NoError(t, err)

	t.Cleanup(func() {
		err = os.Chdir(origDir)
		assert.NoError(err)
		ddevapp.PowerOff()
		globalconfig.DdevGlobalConfig.UseTraefik = origTraefik
		err = globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig)
		assert.NoError(err)
	})
	err = app.Start()
	require.NoError(t, err)

	desc, err := app.Describe(false)
	assert.True(desc["use_traefik"].(bool))
}
