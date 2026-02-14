package ddevapp_test

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/globalconfig/types"
	"github.com/ddev/ddev/pkg/testcommon"
	copy2 "github.com/otiai10/copy"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

// TestTraefikSimple tests basic Traefik router usage
func TestTraefikSimple(t *testing.T) {
	if os.Getenv("DDEV_RUN_TEST_ANYWAY") != "true" && (dockerutil.IsColima() || dockerutil.IsLima() || dockerutil.IsRancherDesktop()) {
		// Intermittent failures in CI due apparently to https://github.com/lima-vm/lima/issues/2536
		// Expected port is not available, so it allocates another one.
		t.Skip("Skipping on Colima/Lima/Rancher because they don't predictably return ports")
	}

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

	// Verify default_config.yaml exists in the router volume
	stdout, _, err := dockerutil.Exec("ddev-router", "cat /mnt/ddev-global-cache/traefik/config/default_config.yaml", "")
	require.NoError(t, err, "default_config.yaml should exist in router volume")
	require.Contains(t, stdout, "defaultCertificate", "default_config.yaml should contain default certificate configuration")

	// Verify default certificates exist in the router volume (if not using Let's Encrypt)
	if !globalconfig.DdevGlobalConfig.UseLetsEncrypt && globalconfig.GetCAROOT() != "" {
		stdout, _, err = dockerutil.Exec("ddev-router", "ls -la /mnt/ddev-global-cache/traefik/certs/", "")
		require.NoError(t, err, "should be able to list certs directory")
		require.Contains(t, stdout, "default_cert.crt", "default_cert.crt should exist in router volume")
		require.Contains(t, stdout, "default_key.key", "default_key.key should exist in router volume")
	}

	// Test reachability in each of the hostnames
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
	if globalconfig.GetCAROOT() != "" {
		_, _ = testcommon.EnsureLocalHTTPContent(t, "https://extra.ddev.site", "Welcome to nginx")
	}
}

// TestTraefikStaticConfig tests static config usage and merging
func TestTraefikStaticConfig(t *testing.T) {
	if os.Getenv("DDEV_RUN_TEST_ANYWAY") != "true" && (dockerutil.IsColima() || dockerutil.IsLima() || dockerutil.IsRancherDesktop()) {
		// Intermittent failures in CI due apparently to https://github.com/lima-vm/lima/issues/2536
		// Expected port is not available, so it allocates another one.
		t.Skip("Skipping on Colima/Lima/Rancher because they don't predictably release ports")
	}
	origDir, _ := os.Getwd()
	globalTraefikDir := filepath.Join(globalconfig.GetGlobalDdevDir(), "traefik")
	staticConfigFinalPath := filepath.Join(globalTraefikDir, ".static_config.yaml")

	site := TestSites[0] // 0 == wordpress
	app, err := ddevapp.NewApp(site.Dir, true)
	require.NoError(t, err)

	testData := filepath.Join(origDir, "testdata", t.Name())

	err = app.Start()
	require.NoError(t, err)

	activeApps := ddevapp.GetActiveProjects()

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
		t.Run(tc.content, func(t *testing.T) {
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
				err = ddevapp.PushGlobalTraefikConfig(activeApps)
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
			err = ddevapp.PushGlobalTraefikConfig(activeApps)
			require.NoError(t, err)
			// Now read result config and compare
			renderedStaticConfig, err := fileutil.ReadFileIntoString(staticConfigFinalPath)
			require.NoError(t, err)
			require.Equal(t, string(unmarshalledExpectationString), renderedStaticConfig)
		})
	}
}

// TestCustomGlobalConfig tests that custom Traefik config from
// ~/.ddev/traefik/custom-global-config/ is pushed to the router and loaded by Traefik
func TestCustomGlobalConfig(t *testing.T) {
	origDir, _ := os.Getwd()
	testDataDir := filepath.Join(origDir, "testdata", t.Name())

	site := TestSites[0]
	app, err := ddevapp.NewApp(site.Dir, true)
	require.NoError(t, err)

	// Set up paths
	customGlobalConfigDir := filepath.Join(globalconfig.GetGlobalDdevDir(), "traefik", "custom-global-config")
	testMiddlewareName := "test-custom-middleware"
	testConfigFile := filepath.Join(customGlobalConfigDir, testMiddlewareName+".yaml")
	testHeaderName := "X-Test-Custom-Global"
	testHeaderValue := "ddev-test-value"

	// Create the custom-global-config directory if it doesn't exist
	err = os.MkdirAll(customGlobalConfigDir, 0755)
	require.NoError(t, err)

	// Copy the middleware config from testdata to custom-global-config
	err = fileutil.CopyFile(filepath.Join(testDataDir, "test-custom-middleware.yaml"), testConfigFile)
	require.NoError(t, err)

	// Create project-level traefik config that replaces the ddev-generated one
	// and adds the test middleware to routers
	projectTraefikConfigDir := app.GetConfigPath("traefik/config")
	err = os.MkdirAll(projectTraefikConfigDir, 0755)
	require.NoError(t, err)
	projectTraefikConfigFile := filepath.Join(projectTraefikConfigDir, app.Name+".yaml")

	// Read the template and replace APPNAME_LOWERCASE and APPNAME_ORIGCASE with actual app name
	projectConfigTemplate, err := fileutil.ReadFileIntoString(filepath.Join(testDataDir, "project-traefik-config.yaml"))
	require.NoError(t, err)
	projectTraefikConfig := strings.ReplaceAll(projectConfigTemplate, "APPNAME_LOWERCASE", strings.ToLower(app.Name))
	projectTraefikConfig = strings.ReplaceAll(projectTraefikConfig, "APPNAME_ORIGCASE", app.Name)

	err = os.WriteFile(projectTraefikConfigFile, []byte(projectTraefikConfig), 0644)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = os.Chdir(origDir)
		_ = app.Stop(true, false)
		_ = os.Remove(testConfigFile)
		_ = os.Remove(projectTraefikConfigFile)
		ddevapp.PowerOff()
	})

	// Start the project and push the custom config to the router
	err = app.Start()
	require.NoError(t, err)

	// Verify the config file exists in the router's config directory
	configDir := "/mnt/ddev-global-cache/traefik/config"
	stdout, _, err := dockerutil.Exec("ddev-router", "ls "+configDir, "")
	require.NoError(t, err, "failed to list router config directory")
	require.Contains(t, stdout, testMiddlewareName+".yaml",
		"Router config directory should contain the custom global config file")

	// Verify the middleware is loaded by Traefik via its API
	dockerIP, _ := dockerutil.GetDockerIP()
	monitorPort := globalconfig.DdevGlobalConfig.TraefikMonitorPort
	middlewaresURL := "http://" + dockerIP + ":" + monitorPort + "/api/http/middlewares"

	// Query Traefik API for middlewares - the custom middleware should be listed
	body, resp, err := testcommon.GetLocalHTTPResponse(t, middlewaresURL, 30)
	require.NoError(t, err, "Failed to query Traefik API for middlewares")
	require.Equal(t, 200, resp.StatusCode, "Traefik API should return 200")
	require.Contains(t, body, testMiddlewareName,
		"Traefik API should show the custom middleware is loaded")

	// Verify the middleware is actually applied by checking the response header
	// Use a simple HTTP client - we only care about headers, not status code
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Returning any error stops redirects.
			// http.ErrUseLastResponse is the canonical choice.
			return http.ErrUseLastResponse
		},
	}

	httpResp, err := client.Get(app.GetPrimaryURL())
	require.NoError(t, err)
	defer httpResp.Body.Close()
	require.Equal(t, testHeaderValue, httpResp.Header.Get(testHeaderName),
		"Response should contain the custom header from global middleware")
}

// TestMergeTraefikProjectConfig tests that multiple project traefik config files are properly merged
// and that the merged configuration works correctly with HTTP to HTTPS redirect
func TestMergeTraefikProjectConfig(t *testing.T) {
	origDir, _ := os.Getwd()

	site := TestSites[0] // 0 == wordpress
	app, err := ddevapp.NewApp(site.Dir, true)
	require.NoError(t, err)

	origRouter := globalconfig.DdevGlobalConfig.Router
	globalconfig.DdevGlobalConfig.Router = types.RouterTypeTraefik
	err = globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = os.Chdir(origDir)
		_ = app.Stop(true, false)
		ddevapp.PowerOff()
		globalconfig.DdevGlobalConfig.Router = origRouter
		_ = globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig)

		// Clean up extra traefik config file
		extraConfigFile := filepath.Join(app.GetConfigPath("traefik/config"), "redirect-https.yaml")
		_ = os.Remove(extraConfigFile)
	})

	// Start the app first to generate base traefik config
	err = app.Start()
	require.NoError(t, err)

	// Stop to add the extra config file
	err = app.Stop(false, false)
	require.NoError(t, err)

	// Create an extra traefik config file that enables HTTP to HTTPS redirect
	projectTraefikConfigDir := app.GetConfigPath("traefik/config")
	extraConfigFile := filepath.Join(projectTraefikConfigDir, "redirect-https.yaml")
	extraConfig := `# Extra config to enable HTTP to HTTPS redirect
http:
  routers:
    ` + app.Name + `-web-80-http:
      middlewares:
        - "` + app.Name + `-redirectHttps"
`
	err = os.WriteFile(extraConfigFile, []byte(extraConfig), 0644)
	require.NoError(t, err)

	// Start again to pick up the new config
	err = app.Start()
	require.NoError(t, err)

	// Check that the merged config file exists in global traefik config
	globalTraefikConfigDir := filepath.Join(globalconfig.GetGlobalDdevDir(), "traefik", "config")
	mergedConfigFile := filepath.Join(globalTraefikConfigDir, app.Name+"_merged.yaml")
	require.FileExists(t, mergedConfigFile, "Merged config file should exist in global traefik config")

	// Read and verify the merged config contains the middleware reference
	mergedConfigContent, err := fileutil.ReadFileIntoString(mergedConfigFile)
	require.NoError(t, err)
	require.Contains(t, mergedConfigContent, app.Name+"-redirectHttps", "Merged config should contain redirectHttps middleware reference")
	require.Contains(t, mergedConfigContent, "middlewares:", "Merged config should contain middlewares section")

	// Verify project still works correctly with custom config
	_, err = testcommon.EnsureLocalHTTPContent(t, app.GetPrimaryURL()+site.Safe200URIWithExpectation.URI, site.Safe200URIWithExpectation.Expect)
	require.NoError(t, err, "Project should still be accessible with merged traefik config")

	// Test that HTTP to HTTPS redirect actually works
	// Get the HTTP URLs
	httpURLs, _, _ := app.GetAllURLs()
	require.NotEmpty(t, httpURLs, "Should have at least one HTTP URL")

	// Create client that doesn't follow redirects so we can check the redirect response
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// Make request to HTTP URL
	resp, err := client.Get(httpURLs[0] + site.Safe200URIWithExpectation.URI)
	require.NoError(t, err, "HTTP request should succeed")
	defer resp.Body.Close()

	// Should get a redirect (301 or 308)
	require.True(t, resp.StatusCode == 301 || resp.StatusCode == 308,
		"HTTP request should return redirect status (got %d)", resp.StatusCode)

	// Redirect location should be HTTPS version of the URL
	location := resp.Header.Get("Location")
	require.NotEmpty(t, location, "Redirect should have Location header")
	require.True(t, strings.HasPrefix(location, "https://"),
		"Redirect location should be HTTPS (got %s)", location)
}

// TestCustomProjectTraefikConfig tests that custom project-level Traefik configuration
// (after removing #ddev-generated) is properly deployed and affects behavior
func TestCustomProjectTraefikConfig(t *testing.T) {
	//if dockerutil.IsRancherDesktop() {
	//	t.Skip("Skipping on Rancher Desktop because it seems to be too slow to pick up fsnotify on changed traefik config files")
	//}
	origDir, _ := os.Getwd()
	site := TestSites[0]
	app, err := ddevapp.NewApp(site.Dir, true)
	require.NoError(t, err)

	if app.CanUseHTTPOnly() {
		t.Skip("Skipping because HTTPS is not available")
	}

	projectTraefikConfigDir := app.GetConfigPath("traefik/config")

	t.Cleanup(func() {
		_ = os.Chdir(origDir)
		_ = app.Stop(true, false)
		// Remove the custom config directory so it will be regenerated
		_ = os.RemoveAll(projectTraefikConfigDir)
		ddevapp.PowerOff()
	})

	// We need a clean set of ports for this test because we're doing a specific alteration
	// of the traefik config that won't work if the port changes, so avoid ephemeral port use
	ddevapp.PowerOff()

	// Start the project to generate initial Traefik config
	err = app.Start()
	require.NoError(t, err)

	// Skip if ephemeral ports are in use due to port conflicts
	// This happens on Lima-based providers when ports 80/443 are already in use
	httpPort := app.GetPrimaryRouterHTTPPort()
	httpsPort := app.GetPrimaryRouterHTTPSPort()
	if httpPort != "80" || httpsPort != "443" {
		t.Skipf("Skipping because non-standard ports are in use (HTTP=%s, HTTPS=%s) - likely due to port conflicts; this breaks the assumption in the altered traefik config", httpPort, httpsPort)
	}

	// Path to the project's Traefik config file
	projectTraefikConfigFile := filepath.Join(projectTraefikConfigDir, app.Name+".yaml")

	// Verify the config file was generated
	require.FileExists(t, projectTraefikConfigFile, "Project Traefik config should be generated")

	// Read the generated config
	configContent, err := fileutil.ReadFileIntoString(projectTraefikConfigFile)
	require.NoError(t, err)

	// Remove the #ddev-generated line so DDEV won't overwrite it
	configContent = strings.Replace(configContent, "#ddev-generated\n", "", 1)

	// Enable the HTTP→HTTPS redirect middleware by uncommenting it
	// The default config has these lines commented out:
	//      # middlewares:
	//      #   - "{{ $.App.Name }}-redirectHttps"
	// We need to find the HTTP router and uncomment the middleware
	configContent = strings.ReplaceAll(configContent, "      # middlewares:\n      #   - \""+app.Name+"-redirectHttps\"",
		"      middlewares:\n        - \""+app.Name+"-redirectHttps\"")

	// Write the modified config back
	err = os.WriteFile(projectTraefikConfigFile, []byte(configContent), 0644)
	require.NoError(t, err)

	// Restart the project to push the custom config
	err = app.Restart()
	require.NoError(t, err)

	// Verify the custom config exists in the global traefik config directory
	globalTraefikConfigDir := filepath.Join(globalconfig.GetGlobalDdevDir(), "traefik", "config")
	globalProjectConfigFile := filepath.Join(globalTraefikConfigDir, app.Name+"_merged.yaml")
	require.FileExists(t, globalProjectConfigFile,
		"Custom project config should be copied to global traefik config directory")

	// Verify the content was copied correctly
	globalConfigContent, err := fileutil.ReadFileIntoString(globalProjectConfigFile)
	require.NoError(t, err)
	require.Equal(t, configContent, globalConfigContent,
		"Global config should match the custom project config")

	// Verify the config exists in the router's volume
	configDir := "/mnt/ddev-global-cache/traefik/config"
	stdout, _, err := dockerutil.Exec("ddev-router", "ls "+configDir, "")
	require.NoError(t, err, "Failed to list router config directory")
	require.Contains(t, stdout, app.Name+"_merged.yaml",
		"Router config directory should contain the project config file")

	// Verify the config content in the router volume
	stdout, _, err = dockerutil.Exec("ddev-router", "cat "+configDir+"/"+app.Name+"_merged.yaml", "")
	require.NoError(t, err, "Failed to read project config from router volume")
	require.Contains(t, stdout, "middlewares:",
		"Router config should contain the enabled middlewares section")
	require.Contains(t, stdout, app.Name+"-redirectHttps",
		"Router config should reference the redirect middleware")

	// Most important: Verify the HTTP→HTTPS redirect actually works
	// Create an HTTP client that doesn't follow redirects
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// Make an HTTP request to the project
	httpURL := strings.Replace(app.GetPrimaryURL(), "https://", "http://", 1)
	httpResp, err := client.Get(httpURL) //nolint:noctx
	require.NoError(t, err)
	require.NotNil(t, httpResp)
	defer func() {
		_ = httpResp.Body.Close()
	}()

	// Verify we get a redirect response
	require.True(t, httpResp.StatusCode == http.StatusMovedPermanently || httpResp.StatusCode == http.StatusFound,
		"HTTP request should return redirect status (301 or 302), got %d", httpResp.StatusCode)

	// Verify the Location header points to HTTPS
	location := httpResp.Header.Get("Location")
	require.NotEmpty(t, location, "Redirect response should have Location header")
	require.True(t, strings.HasPrefix(location, "https://"),
		"Redirect location should use https://, got: %s", location)
}

// TestTraefikMultipleCerts tests that multiple certificate files from
// .ddev/traefik/certs/ are copied to the router container
func TestTraefikMultipleCerts(t *testing.T) {
	origDir, _ := os.Getwd()

	site := TestSites[0]
	app, err := ddevapp.NewApp(site.Dir, true)
	require.NoError(t, err)

	// Set up paths
	projectCertsDir := app.GetConfigPath("traefik/certs")
	err = os.MkdirAll(projectCertsDir, 0755)
	require.NoError(t, err)

	// Create multiple test certificate files
	testCerts := map[string]string{
		"ca.crt":         "test CA certificate content",
		"custom-ca.crt":  "custom CA certificate content",
		"client.crt":     "client certificate content",
		"client.key":     "client key content",
		"additional.crt": "additional certificate content",
		"additional.key": "additional key content",
	}

	for certName, content := range testCerts {
		certPath := filepath.Join(projectCertsDir, certName)
		err = os.WriteFile(certPath, []byte(content), 0644)
		require.NoError(t, err)
	}

	t.Cleanup(func() {
		_ = os.Chdir(origDir)
		_ = app.Stop(true, false)
		_ = os.RemoveAll(projectCertsDir)
		ddevapp.PowerOff()
	})

	// Start the project - this will trigger PushGlobalTraefikConfig
	err = app.Start()
	require.NoError(t, err)

	// Verify all certificate files exist in the router's certs directory
	certsDir := "/mnt/ddev-global-cache/traefik/certs"
	for certName := range testCerts {
		stdout, _, err := dockerutil.Exec("ddev-router", "cat "+certsDir+"/"+certName, "")
		require.NoError(t, err, "certificate %s should exist in router volume", certName)
		require.Contains(t, stdout, testCerts[certName],
			"certificate %s should contain expected content", certName)
	}
}
