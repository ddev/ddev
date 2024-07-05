package ddevapp_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/globalconfig/types"
	"github.com/ddev/ddev/pkg/testcommon"
	copy2 "github.com/otiai10/copy"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// TestTraefikSimple tests basic Traefik router usage
func TestTraefikSimple(t *testing.T) {
	assert := asrt.New(t)

	// Make sure this leaves us in the original test directory
	origDir, _ := os.Getwd()

	site := TestSites[0] // 0 == wordpress
	app, err := ddevapp.NewApp(site.Dir, true)
	assert.NoError(err)

	ddevapp.PowerOff()
	origRouter := globalconfig.DdevGlobalConfig.Router
	globalconfig.DdevGlobalConfig.Router = types.RouterTypeTraefik
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
		globalconfig.DdevGlobalConfig.Router = origRouter
		err = globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig)
		assert.NoError(err)
	})

	app.AdditionalHostnames = []string{"one", "two", "*.wild"}
	app.AdditionalFQDNs = []string{"onefullurl.ddev.site", "twofullurl.ddev.site", "*.wild.fqdn"}
	err = app.WriteConfig()
	require.NoError(t, err)
	err = app.StartAndWait(5)
	require.NoError(t, err)

	err = app.MutagenSyncFlush()
	require.NoError(t, err, "failed to flush Mutagen sync")

	desc, err := app.Describe(false)
	assert.Equal(desc["router"].(string), types.RouterTypeTraefik)

	// Test reachabiliity in each of the hostnames
	httpURLs, _, allURLs := app.GetAllURLs()

	// If no mkcert trusted https, use only the httpURLs
	// This is especially the case for Colima
	if globalconfig.GetCAROOT() == "" {
		allURLs = httpURLs
	}

	for _, u := range allURLs {
		// Use something here for wildcard
		u = strings.Replace(u, `*`, `somewildcard`, 1)
		_, err = testcommon.EnsureLocalHTTPContent(t, u+site.Safe200URIWithExpectation.URI, site.Safe200URIWithExpectation.Expect)
		assert.NoError(err, "failed EnsureLocalHTTPContent() %s: %v", u+site.Safe200URIWithExpectation.URI, err)
	}
}

// TestTraefikVirtualHost tests Traefik with an extra VIRTUAL_HOST
func TestTraefikVirtualHost(t *testing.T) {
	assert := asrt.New(t)

	// Make sure this leaves us in the original test directory
	origDir, _ := os.Getwd()

	site := TestSites[0] // 0 == wordpress
	app, err := ddevapp.NewApp(site.Dir, true)
	assert.NoError(err)

	ddevapp.PowerOff()
	origRouter := globalconfig.DdevGlobalConfig.Router
	globalconfig.DdevGlobalConfig.Router = "traefik"
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
		globalconfig.DdevGlobalConfig.Router = origRouter
		err = globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig)
		assert.NoError(err)
	})

	err = fileutil.CopyFile(filepath.Join(origDir, "testdata", t.Name(), "docker-compose.extra.yaml"), app.GetConfigPath("docker-compose.extra.yaml"))
	require.NoError(t, err)

	err = app.StartAndWait(5)
	require.NoError(t, err)

	desc, err := app.Describe(false)
	assert.Equal(types.RouterTypeTraefik, desc["router"].(string))

	// Test reachabiliity in each of the hostnames
	httpURLs, _, allURLs := app.GetAllURLs()

	// If no mkcert trusted https, use only the httpURLs
	// This is especially the case for Colima
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

// TestTraefikStaticConfig tests static config usage and merging
func TestTraefikStaticConfig(t *testing.T) {
	origDir, _ := os.Getwd()
	globalTraefikDir := filepath.Join(globalconfig.GetGlobalDdevDir(), "traefik")
	staticConfigFinalPath := filepath.Join(globalTraefikDir, ".static_config.yaml")

	site := TestSites[0] // 0 == wordpress
	app, err := ddevapp.NewApp(site.Dir, true)
	require.NoError(t, err)

	testData := filepath.Join(origDir, "testdata", t.Name())

	err = app.Start()
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = app.Stop(true, false)
		ddevapp.PowerOff()
	})

	testCases := []struct {
		content string
		dir     string
	}{
		{"logChange", "logChange"},
		{"extraPlugin", "extraPlugin"},
	}
	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			testSourceDir := filepath.Join(testData, tc.dir)
			traefikGlobalConfigDir := filepath.Join(globalconfig.GetGlobalDdevDir(), "traefik")

			err = copy2.Copy(testSourceDir, traefikGlobalConfigDir)
			require.NoError(t, err)

			// Remove any static_config.*.yaml we have added
			t.Cleanup(func() {
				files, _ := filepath.Glob(filepath.Join(testSourceDir, "static_config.*.yaml"))
				for _, fileToRemove := range files {
					f := filepath.Base(fileToRemove)
					err = os.Remove(filepath.Join(traefikGlobalConfigDir, f))
					require.NoError(t, err)
				}
				err = os.Remove(filepath.Join(traefikGlobalConfigDir, "expectation.yaml"))
				require.NoError(t, err)
				err = ddevapp.PushGlobalTraefikConfig()
				require.NoError(t, err)
			})

			// Unmarshal the loaded result expectation so it will look the same as merged (without comments, etc)
			var tmpMap map[string]interface{}
			expectedResultString, err := fileutil.ReadFileIntoString(filepath.Join(testSourceDir, "expectation.yaml"))
			require.NoError(t, err)
			err = yaml.Unmarshal([]byte(expectedResultString), &tmpMap)
			require.NoError(t, err)
			unmarshalledExpectationString, err := yaml.Marshal(tmpMap)
			require.NoError(t, err)

			// Generate and push config
			err = ddevapp.PushGlobalTraefikConfig()
			require.NoError(t, err)
			// Now read result config and compare
			renderedStaticConfig, err := fileutil.ReadFileIntoString(staticConfigFinalPath)
			require.NoError(t, err)
			require.Equal(t, string(unmarshalledExpectationString), renderedStaticConfig)
		})
	}
}
