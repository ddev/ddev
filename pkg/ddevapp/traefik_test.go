package ddevapp_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

// waitForTraefikRouterMiddleware polls Traefik API until the specified router has the expected middleware
func waitForTraefikRouterMiddleware(t *testing.T, routerName string, expectedMiddleware string, timeout time.Duration) error {
	dockerIP, err := dockerutil.GetDockerIP()
	if err != nil {
		return fmt.Errorf("failed to get Docker IP: %v", err)
	}

	monitorPort := globalconfig.DdevGlobalConfig.TraefikMonitorPort
	routerURL := fmt.Sprintf("http://%s:%s/api/http/routers/%s@file", dockerIP, monitorPort, routerName)
	t.Logf("Polling Traefik API: %s", routerURL)

	deadline := time.Now().Add(timeout)
	attempt := 0
	for time.Now().Before(deadline) {
		attempt++
		resp, err := http.Get(routerURL)
		if err != nil {
			t.Logf("Attempt %d: Failed to query API: %v", attempt, err)
			time.Sleep(200 * time.Millisecond)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			t.Logf("Attempt %d: Failed to read response: %v", attempt, err)
			time.Sleep(200 * time.Millisecond)
			continue
		}

		if resp.StatusCode == 404 {
			t.Logf("Attempt %d: Router not found (404), response: %s", attempt, string(body))
			time.Sleep(200 * time.Millisecond)
			continue
		}

		var router struct {
			Middlewares []string `json:"middlewares"`
			Status      string   `json:"status"`
		}

		err = json.Unmarshal(body, &router)
		if err != nil {
			t.Logf("Attempt %d: Failed to parse JSON: %v, body: %s", attempt, err, string(body))
			time.Sleep(200 * time.Millisecond)
			continue
		}

		t.Logf("Attempt %d: Router %s has middlewares: %v (expecting %s)", attempt, routerName, router.Middlewares, expectedMiddleware)

		// Check if router has the expected middleware
		// Traefik appends the provider name (e.g., "@file") to middleware names in the API
		for _, mw := range router.Middlewares {
			if mw == expectedMiddleware || mw == expectedMiddleware+"@file" {
				t.Logf("SUCCESS: Found expected middleware %s on router %s", mw, routerName)
				return nil
			}
		}

		time.Sleep(200 * time.Millisecond)
	}

	return fmt.Errorf("timeout waiting for middleware %s on router %s after %d attempts", expectedMiddleware, routerName, attempt)
}

// TestTraefikCustomProjectConfig tests project-level Traefik customization including:
// - Custom Traefik config without #ddev-generated signature
// - Additional certificate files in .ddev/traefik/certs/
func TestTraefikCustomProjectConfig(t *testing.T) {
	origDir, _ := os.Getwd()

	site := TestSites[0]
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
	})

	// Start project to generate default traefik config
	err = app.Start()
	require.NoError(t, err)

	// Stop to modify the config
	err = app.Stop(false, false)
	require.NoError(t, err)

	// Create custom project traefik config WITHOUT #ddev-generated
	projectTraefikConfigDir := app.GetConfigPath("traefik/config")
	projectTraefikConfigFile := filepath.Join(projectTraefikConfigDir, app.Name+".yaml")

	// Read generated config and modify it (remove #ddev-generated signature)
	generatedConfig, err := fileutil.ReadFileIntoString(projectTraefikConfigFile)
	require.NoError(t, err)

	// Remove #ddev-generated and add a custom comment
	customConfig := strings.Replace(generatedConfig, "#ddev-generated", "# Custom Traefik config for testing", 1)

	// Enable the redirectHttps middleware on the HTTP router by uncommenting it
	customConfig = strings.Replace(customConfig, "# middlewares:", "middlewares:", 1)
	customConfig = strings.Replace(customConfig, "#   - \""+app.Name+"-redirectHttps\"", "  - \""+app.Name+"-redirectHttps\"", 1)

	// Verify the middleware was uncommented
	require.Contains(t, customConfig, "middlewares:",
		"Config should contain uncommented middlewares")
	require.Contains(t, customConfig, "- \""+app.Name+"-redirectHttps\"",
		"Config should contain uncommented redirectHttps middleware")

	err = os.WriteFile(projectTraefikConfigFile, []byte(customConfig), 0644)
	require.NoError(t, err)

	// Add extra certificate files to .ddev/traefik/certs/
	projectCertsDir := app.GetConfigPath("traefik/certs")
	err = os.MkdirAll(projectCertsDir, 0755)
	require.NoError(t, err)

	// Create dummy certificate files for testing
	customCertContent := "-----BEGIN CERTIFICATE-----\nTEST CERT CONTENT\n-----END CERTIFICATE-----\n"
	customKeyContent := "-----BEGIN PRIVATE KEY-----\nTEST KEY CONTENT\n-----END PRIVATE KEY-----\n"

	testCerts := map[string]string{
		"custom-ca.crt":     customCertContent,
		"custom-ca.key":     customKeyContent,
		"mtls-client.crt":   customCertContent,
		"mtls-client.key":   customKeyContent,
		"extra-service.crt": customCertContent,
		"extra-service.key": customKeyContent,
	}

	for filename, content := range testCerts {
		certPath := filepath.Join(projectCertsDir, filename)
		err = os.WriteFile(certPath, []byte(content), 0644)
		require.NoError(t, err)
	}

	// DEBUG: Verify config file content before restart
	configBeforeRestart, err := fileutil.ReadFileIntoString(projectTraefikConfigFile)
	require.NoError(t, err)
	configSnippet := configBeforeRestart
	if len(configSnippet) > 500 {
		configSnippet = configSnippet[:500]
	}
	t.Logf("Config file on host before restart (first 500 chars):\n%s", configSnippet)
	hasMiddlewareInHost := strings.Contains(configBeforeRestart, "middlewares:") &&
		strings.Contains(configBeforeRestart, app.Name+"-redirectHttps")
	t.Logf("Host config has middleware directive: %v", hasMiddlewareInHost)

	// Restart the project
	err = app.Start()
	require.NoError(t, err)

	// DEBUG: Check what's actually in the router volume after restart
	configInVolume := "/mnt/ddev-global-cache/traefik/config/" + app.Name + ".yaml"
	stdout, stderr, err := dockerutil.Exec("ddev-router", "cat "+configInVolume, "")
	if err != nil {
		t.Logf("Failed to read config from router volume: %v, stderr: %s", err, stderr)
	} else {
		volumeSnippet := stdout
		if len(volumeSnippet) > 500 {
			volumeSnippet = volumeSnippet[:500]
		}
		t.Logf("Config file in router volume (first 500 chars):\n%s", volumeSnippet)
		hasMiddlewareInVolume := strings.Contains(stdout, "middlewares:") &&
			strings.Contains(stdout, app.Name+"-redirectHttps")
		t.Logf("Volume config has middleware directive: %v", hasMiddlewareInVolume)

		if hasMiddlewareInHost != hasMiddlewareInVolume {
			t.Logf("WARNING: Config mismatch! Host has middleware=%v but volume has middleware=%v",
				hasMiddlewareInHost, hasMiddlewareInVolume)
		}
	}

	// DEBUG: Check Traefik logs for file provider reload messages
	logs, _, _ := dockerutil.Exec("ddev-router", `grep -i "configuration" /tmp/traefik-stderr.txt 2>/dev/null | tail -20 || echo "No logs found"`, "")
	t.Logf("Recent Traefik configuration logs:\n%s", logs)

	// Wait for Traefik to load the custom config with middleware
	httpRouterName := app.Name + "-web-80-http"
	expectedMiddleware := app.Name + "-redirectHttps"
	err = waitForTraefikRouterMiddleware(t, httpRouterName, expectedMiddleware, 10*time.Second)

	// If it failed, dump more diagnostic info
	if err != nil {
		t.Logf("DIAGNOSTIC: Middleware not found, gathering more info...")

		// List all routers
		allRoutersURL := fmt.Sprintf("http://127.0.0.1:%s/api/http/routers", globalconfig.DdevGlobalConfig.TraefikMonitorPort)
		resp, _ := http.Get(allRoutersURL)
		if resp != nil {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			routersSnippet := string(body)
			if len(routersSnippet) > 2000 {
				routersSnippet = routersSnippet[:2000]
			}
			t.Logf("All routers in Traefik:\n%s", routersSnippet)
		}

		// Check overview for errors
		overviewURL := fmt.Sprintf("http://127.0.0.1:%s/api/overview", globalconfig.DdevGlobalConfig.TraefikMonitorPort)
		resp2, _ := http.Get(overviewURL)
		if resp2 != nil {
			body2, _ := io.ReadAll(resp2.Body)
			resp2.Body.Close()
			t.Logf("Traefik overview:\n%s", string(body2))
		}
	}

	require.NoError(t, err, "Traefik should load the redirectHttps middleware")

	// Verify custom config is NOT overwritten (still has our custom comment)
	configAfterStart, err := fileutil.ReadFileIntoString(projectTraefikConfigFile)
	require.NoError(t, err)
	require.Contains(t, configAfterStart, "# Custom Traefik config for testing",
		"Custom config should be preserved after restart")
	require.NotContains(t, configAfterStart, "#ddev-generated",
		"Custom config should not have #ddev-generated after restart")

	// Verify all custom cert files are present in the router volume
	certsDir := "/mnt/ddev-global-cache/traefik/certs"
	stdout, _, err = dockerutil.Exec("ddev-router", "ls "+certsDir, "")
	require.NoError(t, err, "failed to list router certs directory")

	for filename := range testCerts {
		require.Contains(t, stdout, filename,
			"Router certs directory should contain custom cert file: "+filename)
	}

	// Verify the custom cert files have the expected content
	for filename, expectedContent := range testCerts {
		stdout, _, err := dockerutil.Exec("ddev-router", "cat "+filepath.Join(certsDir, filename), "")
		require.NoError(t, err, "failed to read cert file "+filename)
		require.Equal(t, expectedContent, stdout,
			"Cert file "+filename+" should have the expected content")
	}

	// Verify project still works correctly with custom config
	_, err = testcommon.EnsureLocalHTTPContent(t, app.GetPrimaryURL()+site.Safe200URIWithExpectation.URI, site.Safe200URIWithExpectation.Expect)
	require.NoError(t, err, "Project should still be accessible with custom traefik config")

	// Verify the HTTP->HTTPS redirect middleware is working
	// Get the HTTP URL (not HTTPS)
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
