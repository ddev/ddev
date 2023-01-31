package ddevapp_test

import (
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/drud/ddev/pkg/testcommon"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"strings"
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
	origConfig := *app

	t.Cleanup(func() {
		err = os.Chdir(origDir)
		assert.NoError(err)
		err = app.Stop(true, false)
		assert.NoError(err)
		ddevapp.PowerOff()
		err = origConfig.WriteConfig()
		assert.NoError(err)
		globalconfig.DdevGlobalConfig.UseTraefik = origTraefik
		err = globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig)
		assert.NoError(err)
	})

	app.AdditionalHostnames = []string{"one", "two", "*.wild"}
	app.AdditionalFQDNs = []string{"onefullurl.ddev.site", "twofullurl.ddev.site", "*.wild.fqdn"}
	err = app.WriteConfig()
	require.NoError(t, err)
	err = app.StartAndWait(5)
	require.NoError(t, err)

	desc, err := app.Describe(false)
	assert.True(desc["use_traefik"].(bool))

	// Test reachabiliity in each of the hostnames
	httpURLs, httpsURLs, allURLs := app.GetAllURLs()

	// If no mkcert trusted https, use only the httpURLs
	// This is especially the case for colima
	if globalconfig.GetCAROOT() == "" {
		allURLs = httpURLs
	}

	for _, u := range allURLs {
		// Use something here for wildcard
		u = strings.Replace(u, `*`, `somewildcard`, 1)
		_, err = testcommon.EnsureLocalHTTPContent(t, u+site.Safe200URIWithExpectation.URI, site.Safe200URIWithExpectation.Expect)
		assert.NoError(err, "failed EnsureLocalHTTPContent() %s: %v", u+site.Safe200URIWithExpectation.URI, err)
	}

	// Test Reachability to PhpMyAdmin, which uses different technique
	_, _ = testcommon.EnsureLocalHTTPContent(t, httpURLs[0]+":8036", "phpMyAdmin")
	if globalconfig.DdevGlobalConfig.MkcertCARoot != "" {
		_, _ = testcommon.EnsureLocalHTTPContent(t, httpsURLs[0]+":8037", "phpMyAdmin")
	}
}

// TestTraefikVirtualHost tests traefik with an extra VIRTUAL_HOST
func TestTraefikVirtualHost(t *testing.T) {
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
	origConfig := *app

	t.Cleanup(func() {
		err = os.Chdir(origDir)
		assert.NoError(err)
		err = os.RemoveAll(app.GetConfigPath(`docker-compose.extra.yaml`))
		assert.NoError(err)
		err = app.Stop(true, false)
		assert.NoError(err)
		ddevapp.PowerOff()
		err = origConfig.WriteConfig()
		assert.NoError(err)
		globalconfig.DdevGlobalConfig.UseTraefik = origTraefik
		err = globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig)
		assert.NoError(err)
	})

	err = fileutil.CopyFile(filepath.Join(origDir, "testdata", t.Name(), "docker-compose.extra.yaml"), app.GetConfigPath("docker-compose.extra.yaml"))
	require.NoError(t, err)

	err = app.StartAndWait(5)
	require.NoError(t, err)

	desc, err := app.Describe(false)
	assert.True(desc["use_traefik"].(bool))

	// Test reachabiliity in each of the hostnames
	httpURLs, _, allURLs := app.GetAllURLs()

	// If no mkcert trusted https, use only the httpURLs
	// This is especially the case for colima
	if globalconfig.GetCAROOT() == "" {
		allURLs = httpURLs
	}

	for _, u := range allURLs {
		// Use something here for wildcard
		u = strings.Replace(u, `*`, `somewildcard`, 1)
		_, err = testcommon.EnsureLocalHTTPContent(t, u+site.Safe200URIWithExpectation.URI, site.Safe200URIWithExpectation.Expect)
		assert.NoError(err, "failed EnsureLocalHTTPContent() %s: %v", u+site.Safe200URIWithExpectation.URI, err)
	}

	// Test Reachability to nginx special VIRTUAL_HOST
	_, _ = testcommon.EnsureLocalHTTPContent(t, "http://extra.ddev.site", "Welcome to nginx")
	if globalconfig.DdevGlobalConfig.MkcertCARoot != "" {
		_, _ = testcommon.EnsureLocalHTTPContent(t, "https://extra.ddev.site", "Welcome to nginx")
	}
}
