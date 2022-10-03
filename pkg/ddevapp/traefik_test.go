package ddevapp_test

import (
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/drud/ddev/pkg/testcommon"
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
		app.AdditionalHostnames = nil
		err = app.WriteConfig()
		assert.NoError(err)
		globalconfig.DdevGlobalConfig.UseTraefik = origTraefik
		err = globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig)
		assert.NoError(err)
	})

	app.AdditionalHostnames = []string{"one", "two"}
	app.AdditionalFQDNs = []string{"onefullurl.ddev.site", "twofullurl.ddev.site"}
	err = app.Start()
	require.NoError(t, err)

	desc, err := app.Describe(false)
	assert.True(desc["use_traefik"].(bool))

	// Test reachabiliity in each of the hostnames
	httpURLs, httpsURLs, allURLs := app.GetAllURLs()
	for _, u := range allURLs {
		_, _ = testcommon.EnsureLocalHTTPContent(t, u+site.Safe200URIWithExpectation.URI, site.Safe200URIWithExpectation.Expect)
	}

	// Test Reachability to PhpMyAdmin, which uses different technique
	_, _ = testcommon.EnsureLocalHTTPContent(t, httpURLs[0]+":8036", "phpMyAdmin")
	_, _ = testcommon.EnsureLocalHTTPContent(t, httpsURLs[0]+":8037", "phpMyAdmin")

}
