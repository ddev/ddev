package ddevapp_test

import (
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/netutil"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/ddev/ddev/pkg/util"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGlobalPortOverride tests global router_http_port and router_https_port
func TestGlobalPortOverride(t *testing.T) {
	if os.Getenv("DDEV_RUN_TEST_ANYWAY") != "true" && (dockerutil.IsLima() || dockerutil.IsColima() || dockerutil.IsRancherDesktop()) {
		// Intermittent failures in CI due apparently to https://github.com/lima-vm/lima/issues/2536
		// Expected port is not available, so it allocates another one.
		t.Skip("Lima and Colima often allocate another port, so skip")
	}
	assert := asrt.New(t)

	origGlobalHTTPPort := globalconfig.DdevGlobalConfig.RouterHTTPPort
	origGlobalHTTPSPort := globalconfig.DdevGlobalConfig.RouterHTTPSPort

	globalconfig.DdevGlobalConfig.RouterHTTPPort = "8555"
	globalconfig.DdevGlobalConfig.RouterHTTPSPort = "8556"
	err := globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig)
	require.NoError(t, err)

	site := TestSites[0]

	app, err := ddevapp.NewApp(site.Dir, false)
	require.NoError(t, err)
	t.Cleanup(func() {
		err = app.Stop(true, false)
		assert.NoError(err)
		globalconfig.DdevGlobalConfig.RouterHTTPPort = origGlobalHTTPPort
		globalconfig.DdevGlobalConfig.RouterHTTPSPort = origGlobalHTTPSPort
		err := globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig)
		assert.NoError(err)
	})

	util.Debug("Before app.Restart(): app.RouterHTTPPort=%s, app.RouterHTTPSPort=%s, app.GetRouterHTTPPort()=%s app.GetRouterHTTPSPort=%s", app.RouterHTTPPort, app.RouterHTTPSPort, app.GetPrimaryRouterHTTPPort(), app.GetPrimaryRouterHTTPSPort())
	err = app.Restart()
	util.Debug("After app.Restart(): app.RouterHTTPPort=%s, app.RouterHTTPSPort=%s, app.GetRouterHTTPPort()=%s app.GetRouterHTTPSPort=%s", app.RouterHTTPPort, app.RouterHTTPSPort, app.GetPrimaryRouterHTTPPort(), app.GetPrimaryRouterHTTPSPort())

	require.NoError(t, err)
	require.Equal(t, globalconfig.DdevGlobalConfig.RouterHTTPPort, app.GetPrimaryRouterHTTPPort())
	require.Equal(t, globalconfig.DdevGlobalConfig.RouterHTTPSPort, app.GetPrimaryRouterHTTPSPort())

	desc, err := app.Describe(false)
	require.NoError(t, err)
	require.Equal(t, globalconfig.DdevGlobalConfig.RouterHTTPPort, desc["router_http_port"])
	require.Equal(t, globalconfig.DdevGlobalConfig.RouterHTTPSPort, desc["router_https_port"])
}

// TestProjectPortOverride makes sure that the project-level
// router_http_port and router_https_port
// port overrides work correctly.
// It starts up three DDEV projects, looks to see if the config is set right,
// then tests to see that the right ports have been started up on the router.
func TestProjectPortOverride(t *testing.T) {
	assert := asrt.New(t)

	origDir, _ := os.Getwd()

	// Try some different combinations of ports.
	for i := 1; i < 3; i++ {
		testDir := testcommon.CreateTmpDir("TestProjectPortOverride")

		t.Cleanup(func() {
			err := os.Chdir(origDir)
			assert.NoError(err)
			_ = os.RemoveAll(testDir)
		})

		testcommon.ClearDockerEnv()
		app, err := ddevapp.NewApp(testDir, true)
		assert.NoError(err)
		app.RouterHTTPPort = strconv.Itoa(8080 + i)
		app.RouterHTTPSPort = strconv.Itoa(8443 + i)
		app.Name = "TestProjectPortOverride-" + strconv.Itoa(i)
		_ = app.Stop(true, false)
		app.Type = nodeps.AppTypePHP
		err = app.WriteConfig()
		assert.NoError(err)
		_, err = app.ReadConfig(false)
		assert.NoError(err)

		stringFound, err := fileutil.FgrepStringInFile(app.ConfigPath, "router_http_port: \""+app.RouterHTTPPort+"\"")
		assert.NoError(err)
		assert.True(stringFound)
		stringFound, err = fileutil.FgrepStringInFile(app.ConfigPath, "router_https_port: \""+app.RouterHTTPSPort+"\"")
		assert.NoError(err)
		assert.True(stringFound)

		err = app.StartAndWait(2)
		require.NoError(t, err)
		// defer the app.Stop() so we have a more diverse set of tests. If we brought
		// each down before testing the next that would be a more trivial test.
		// Don't worry about the possible error case as this is a test cleanup
		t.Cleanup(func() {
			err = app.Stop(true, false)
			assert.NoError(err)
		})

		assert.True(netutil.IsPortActive(app.RouterHTTPPort), "port "+app.RouterHTTPPort+" should be active")
		assert.True(netutil.IsPortActive(app.RouterHTTPSPort), "port "+app.RouterHTTPSPort+" should be active")
	}
}

// TestRouterConfigOverride tests that the ~/.ddev/.router-compose.yaml can be overridden
// with ~/.ddev/router-compose.*.yaml
func TestRouterConfigOverride(t *testing.T) {
	assert := asrt.New(t)
	origDir, _ := os.Getwd()
	extrasYamlName := `router-compose.extras.yaml`
	testDir := testcommon.CreateTmpDir(t.Name())
	_ = os.Chdir(testDir)
	extrasYaml := filepath.Join(globalconfig.GetGlobalDdevDir(), extrasYamlName)

	testcommon.ClearDockerEnv()

	// Remove the router so it gets recreated with the custom config
	err := ddevapp.RemoveRouterContainer()
	assert.NoError(err)

	app, err := ddevapp.NewApp(testDir, true)
	assert.NoError(err)
	err = app.WriteConfig()
	assert.NoError(err)
	err = fileutil.CopyFile(filepath.Join(origDir, "testdata", t.Name(), extrasYamlName), extrasYaml)
	assert.NoError(err)

	answer := fileutil.RandomFilenameBase()
	t.Setenv("ANSWER", answer)
	assert.NoError(err)
	t.Cleanup(func() {
		err = app.Stop(true, false)
		assert.NoError(err)
		err = os.Chdir(origDir)
		assert.NoError(err)
		_ = os.RemoveAll(testDir)
		_ = os.Remove(extrasYaml)
	})

	err = app.Start()
	assert.NoError(err)

	stdout, _, err := dockerutil.Exec("ddev-router", "bash -c 'echo ANSWER=${ANSWER}'", "")
	stdout = strings.Trim(stdout, "\r\n")
	assert.Equal("ANSWER="+answer, stdout)
}

// TestAllocateAvailablePortForRouter tests AllocateAvailablePortForRouter()
func TestAllocateAvailablePortForRouter(t *testing.T) {
	assert := asrt.New(t)

	localIP, _ := dockerutil.GetDockerIP()

	// Get a random port number in the dynamic port range
	startPort := ddevapp.MinEphemeralPort + rand.Intn(500)
	goodEndPort := startPort + 3
	badEndPort := startPort + 2

	// Listen in the first 3 ports
	l0, err := net.Listen("tcp", localIP+":"+strconv.Itoa(startPort))
	require.NoError(t, err)
	l1, err := net.Listen("tcp", localIP+":"+strconv.Itoa(startPort+1))
	require.NoError(t, err)
	l2, err := net.Listen("tcp", localIP+":"+strconv.Itoa(startPort+2))
	require.NoError(t, err)

	t.Cleanup(func() {
		for i, p := range []net.Listener{l0, l1, l2} {
			err = p.Close()
			assert.NoError(err, "failed to close listener %v", i)
		}
	})
	_, ok := ddevapp.AllocateAvailablePortForRouter(startPort, badEndPort)
	assert.Exactly(false, ok)

	port, ok := ddevapp.AllocateAvailablePortForRouter(startPort, goodEndPort)
	require.True(t, ok)
	require.Equal(t, startPort+3, port)
}

// Test that the app assigns an ephemeral port if the default one is not available.
func TestUseEphemeralPort(t *testing.T) {
	if os.Getenv("DDEV_RUN_TEST_ANYWAY") != "true" && (dockerutil.IsColima() || dockerutil.IsLima() || dockerutil.IsRancherDesktop()) {
		// Intermittent failures in CI due apparently to https://github.com/lima-vm/lima/issues/2536
		// Expected port is not available, so it allocates another one.
		t.Skip("Skipping on Lima/Colima/Rancher as ports don't seem to be released properly in a timely fashion")
	}

	// Stop all projects and the router first so we can occupy the ports they would normally use
	// Without this, leftover containers from other tests may have ports that interfere
	ddevapp.PowerOff()

	targetHTTPPort, targetHTTPSPort := "28080", "28443"
	const testString = "Hello from TestUseEphemeralPort"

	apps := []*ddevapp.DdevApp{}
	for _, s := range []string{"site1", "site2"} {
		site := filepath.Join(testcommon.CreateTmpDir(t.Name() + s))
		_ = os.MkdirAll(site, 0755)
		err := fileutil.TemplateStringToFile(testString, nil, filepath.Join(site, "index.html"))
		require.NoError(t, err)

		a, err := ddevapp.NewApp(site, false)
		require.NoError(t, err)
		err = a.WriteConfig()
		require.NoError(t, err)
		apps = append(apps, a)
		a.RouterHTTPPort, a.RouterHTTPSPort = targetHTTPPort, targetHTTPSPort
	}

	// Occupy target router ports so that app1 will be forced
	// to use the ephemeral ports
	for _, p := range []string{apps[0].GetPrimaryRouterHTTPPort(), apps[0].GetPrimaryRouterHTTPSPort(), apps[0].GetMailpitHTTPPort(), apps[0].GetMailpitHTTPSPort()} {
		listener, err := net.Listen("tcp", "127.0.0.1:"+p)
		require.NoError(t, err)
		t.Cleanup(func() {
			_ = listener.Close()
		})
	}
	t.Cleanup(func() {
		for _, a := range apps {
			_ = a.Stop(true, false)
			_ = os.RemoveAll(a.AppRoot)
		}

		// Stop the router, to prevent additional config from interfering with other tests.
		// We shouldn't have to do this when app.Stop() properly pushes new config to ddev-router
		_ = dockerutil.RemoveContainer(nodeps.RouterContainer)
	})

	for i, app := range apps {
		// Predict which ephemeral ports the apps will use by using guess from starting point
		// This is fragile, only works if no other projects are running and holding open the earlier ports
		expectedEphemeralHTTPPort := ddevapp.MinEphemeralPort + i*4
		expectedEphemeralHTTPSPort := ddevapp.MinEphemeralPort + i*4 + 1

		err := app.Start()
		require.NoError(t, err)

		// Get a new copy of the app to make sure we have up-to-date port information
		app, err = ddevapp.NewApp(app.GetAppRoot(), true)
		require.NoError(t, err)

		// app1 will not use the configured target ports, uses the assigned ephemeral ports.
		require.NotEqual(t, targetHTTPPort, app.GetPrimaryRouterHTTPPort())
		require.NotEqual(t, targetHTTPSPort, app.GetPrimaryRouterHTTPSPort())

		// Allow a margin of +2 for ephemeral port checks due to flakiness
		actualHTTPPort, err := strconv.Atoi(app.GetPrimaryRouterHTTPPort())
		require.NoError(t, err)
		require.Condition(t, func() bool {
			return actualHTTPPort >= expectedEphemeralHTTPPort && actualHTTPPort <= expectedEphemeralHTTPPort+2
		}, "HTTP port must be between %d and %d, got %d", expectedEphemeralHTTPPort, expectedEphemeralHTTPPort+2, actualHTTPPort)

		actualHTTPSPort, err := strconv.Atoi(app.GetPrimaryRouterHTTPSPort())
		require.NoError(t, err)
		require.Condition(t, func() bool {
			return actualHTTPSPort >= expectedEphemeralHTTPSPort && actualHTTPSPort <= expectedEphemeralHTTPSPort+2
		}, "HTTPS port must be between %d and %d, got %d", expectedEphemeralHTTPSPort, expectedEphemeralHTTPSPort+2, actualHTTPSPort)

		// Make sure that both http and https URLs have proper content
		_, _ = testcommon.EnsureLocalHTTPContent(t, app.GetHTTPURL(), testString, -1)
		require.Contains(t, app.GetHTTPURL(), app.GetHostname())
		if globalconfig.GetCAROOT() != "" {
			_, _ = testcommon.EnsureLocalHTTPContent(t, app.GetHTTPSURL(), testString, -1)
			require.Contains(t, app.GetHTTPSURL(), app.GetHostname())
		}
	}
}

// TestEphemeralPortsReusedOnRestart tests that ephemeral ports assigned to a project
// are reused when the project restarts, preventing unnecessary router recreation.
func TestEphemeralPortsReusedOnRestart(t *testing.T) {
	if os.Getenv("GOTEST_SHORT") != "" {
		t.Skip("Skipping because GOTEST_SHORT is set")
	}
	if os.Getenv("DDEV_RUN_TEST_ANYWAY") != "true" && (dockerutil.IsColima() || dockerutil.IsLima() || dockerutil.IsRancherDesktop()) {
		t.Skip("Skipping on Lima/Colima/Rancher as ports don't seem to be released properly in a timely fashion")
	}

	// Stop all projects and the router first so we can occupy the ports they would normally use
	ddevapp.PowerOff()
	// Clear ephemeral port assignments from previous tests
	ddevapp.EphemeralRouterPortsAssigned = make(map[int]bool)

	targetHTTPPort, targetHTTPSPort := "29080", "29443"

	site := filepath.Join(testcommon.CreateTmpDir(t.Name()))
	_ = os.MkdirAll(site, 0755)
	err := fileutil.TemplateStringToFile("Hello from TestEphemeralPortsReusedOnRestart", nil, filepath.Join(site, "index.html"))
	require.NoError(t, err)

	app, err := ddevapp.NewApp(site, false)
	require.NoError(t, err)
	app.RouterHTTPPort, app.RouterHTTPSPort = targetHTTPPort, targetHTTPSPort
	err = app.WriteConfig()
	require.NoError(t, err)

	// Occupy target router ports so that app will be forced to use ephemeral ports
	var listeners []net.Listener
	for _, p := range []string{targetHTTPPort, targetHTTPSPort} {
		listener, err := net.Listen("tcp", "127.0.0.1:"+p)
		require.NoError(t, err)
		listeners = append(listeners, listener)
	}
	t.Cleanup(func() {
		for _, l := range listeners {
			_ = l.Close()
		}
		_ = app.Stop(true, false)
		_ = os.RemoveAll(app.AppRoot)
		_ = dockerutil.RemoveContainer(nodeps.RouterContainer)
	})

	// Start the app - it should use ephemeral ports
	err = app.Start()
	require.NoError(t, err)

	// Get the ephemeral ports that were assigned
	app, err = ddevapp.NewApp(app.GetAppRoot(), true)
	require.NoError(t, err)
	firstHTTPPort := app.GetPrimaryRouterHTTPPort()
	firstHTTPSPort := app.GetPrimaryRouterHTTPSPort()

	// Make sure they're ephemeral ports (not the target ports)
	require.NotEqual(t, targetHTTPPort, firstHTTPPort, "HTTP port should be ephemeral")
	require.NotEqual(t, targetHTTPSPort, firstHTTPSPort, "HTTPS port should be ephemeral")

	// Get router container ID before restart
	router, err := ddevapp.FindDdevRouter()
	require.NoError(t, err)
	originalRouterID := router.ID

	// Clear ephemeral port assignments to simulate new process
	ddevapp.EphemeralRouterPortsAssigned = make(map[int]bool)

	// Restart the app - the ephemeral ports should be reused
	err = app.Restart()
	require.NoError(t, err)

	// Get the ports after restart
	app, err = ddevapp.NewApp(app.GetAppRoot(), true)
	require.NoError(t, err)
	secondHTTPPort := app.GetPrimaryRouterHTTPPort()
	secondHTTPSPort := app.GetPrimaryRouterHTTPSPort()

	// Verify the same ephemeral ports are used
	require.Equal(t, firstHTTPPort, secondHTTPPort, "HTTP ephemeral port should be reused on restart")
	require.Equal(t, firstHTTPSPort, secondHTTPSPort, "HTTPS ephemeral port should be reused on restart")

	// Verify the router was not recreated (same container ID)
	router, err = ddevapp.FindDdevRouter()
	require.NoError(t, err)
	require.Equal(t, originalRouterID, router.ID, "Router should not be recreated when ephemeral ports are reused")
}

// TestProcessExposePorts tests the ProcessExposePorts function for various input scenarios
func TestProcessExposePorts(t *testing.T) {
	type testCase struct {
		name          string
		exposePorts   []string
		initialPorts  []string
		expectedPorts []string
	}

	tests := []testCase{
		{
			name:          "Empty expose ports",
			exposePorts:   []string{},
			initialPorts:  []string{},
			expectedPorts: []string{},
		},
		{
			name:          "Single port format",
			exposePorts:   []string{"8080"},
			initialPorts:  []string{},
			expectedPorts: []string{"8080"},
		},
		{
			name:          "Port pair format",
			exposePorts:   []string{"8080:80"},
			initialPorts:  []string{},
			expectedPorts: []string{"8080"},
		},
		{
			name:          "Multiple ports",
			exposePorts:   []string{"8080", "9090:90", "3000"},
			initialPorts:  []string{},
			expectedPorts: []string{"8080", "9090", "3000"},
		},
		{
			name:          "Duplicate ports are not added",
			exposePorts:   []string{"8080", "8080:80"},
			initialPorts:  []string{},
			expectedPorts: []string{"8080"},
		},
		{
			name:          "Existing ports are preserved",
			exposePorts:   []string{"9090"},
			initialPorts:  []string{"8080"},
			expectedPorts: []string{"8080", "9090"},
		},
		{
			name:          "Port already exists in initial list",
			exposePorts:   []string{"8080"},
			initialPorts:  []string{"8080", "9090"},
			expectedPorts: []string{"8080", "9090"},
		},
		{
			name:          "Invalid port format is ignored",
			exposePorts:   []string{"invalid", "8080", "abc:def"},
			initialPorts:  []string{},
			expectedPorts: []string{"8080"},
		},
		{
			name:          "Port with letters is ignored",
			exposePorts:   []string{"80a0", "8080"},
			initialPorts:  []string{},
			expectedPorts: []string{"8080"},
		},
		{
			name:          "Port pair with invalid numbers is ignored",
			exposePorts:   []string{"80a0:80", "8080:8b0", "9090:90"},
			initialPorts:  []string{},
			expectedPorts: []string{"9090"},
		},
		{
			name:          "Empty strings are ignored",
			exposePorts:   []string{"", "8080", ""},
			initialPorts:  []string{},
			expectedPorts: []string{"8080"},
		},
		{
			name:          "Complex scenario with mixed formats",
			exposePorts:   []string{"8080", "9090:90", "invalid", "3000:3000", "8080:80"},
			initialPorts:  []string{"7070", "8080"},
			expectedPorts: []string{"7070", "8080", "9090", "3000"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := ddevapp.ProcessExposePorts(tc.exposePorts, tc.initialPorts)
			require.Equal(t, tc.expectedPorts, result)
		})
	}
}

// TestTraefikMonitorPortAlwaysLocalhost verifies that the Traefik monitor port
// is always bound to localhost, even when router_bind_all_interfaces=true
func TestTraefikMonitorPortAlwaysLocalhost(t *testing.T) {
	assert := asrt.New(t)

	origRouterBindAllInterfaces := globalconfig.DdevGlobalConfig.RouterBindAllInterfaces
	origRouter := globalconfig.DdevGlobalConfig.Router

	t.Cleanup(func() {
		globalconfig.DdevGlobalConfig.RouterBindAllInterfaces = origRouterBindAllInterfaces
		globalconfig.DdevGlobalConfig.Router = origRouter
		_ = globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig)
		ddevapp.PowerOff()
	})

	// Test with router_bind_all_interfaces=true (security concern)
	globalconfig.DdevGlobalConfig.RouterBindAllInterfaces = true
	globalconfig.DdevGlobalConfig.Router = "traefik"
	err := globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig)
	require.NoError(t, err)

	site := TestSites[0]
	app, err := ddevapp.NewApp(site.Dir, false)
	require.NoError(t, err)

	err = app.Start()
	require.NoError(t, err)

	// Read the generated router compose file
	routerComposeFile := ddevapp.RouterComposeYAMLPath()
	content, err := fileutil.ReadFileIntoString(routerComposeFile)
	require.NoError(t, err)

	// The Traefik monitor port should ALWAYS be bound to dockerIP (localhost),
	// never to all interfaces (0.0.0.0)
	dockerIP, _ := dockerutil.GetDockerIP()
	monitorPort := globalconfig.DdevGlobalConfig.TraefikMonitorPort

	// Expected format: "127.0.0.1:10999:10999" (or similar dockerIP)
	expectedBinding := dockerIP + ":" + monitorPort + ":" + monitorPort

	assert.Contains(content, expectedBinding,
		"Traefik monitor port must always be bound to localhost (%s), not all interfaces", dockerIP)

	// Make sure it's NOT bound to all interfaces
	allInterfacesBinding := "- \"" + monitorPort + ":" + monitorPort + "\""
	assert.NotContains(content, allInterfacesBinding,
		"Traefik monitor port must NOT be bound to all interfaces (security risk)")

	// Test that the dashboard is accessible via localhost
	localhostDashboardURL := "http://" + dockerIP + ":" + monitorPort + "/api/overview"
	_, err = testcommon.EnsureLocalHTTPContent(t, localhostDashboardURL, "")
	assert.NoError(err, "Traefik dashboard should be accessible via localhost at %s", localhostDashboardURL)

	// Note: The dashboard may also be accessible via project hostnames on the monitor port
	// (e.g., http://project.ddev.site:10999) because the hostname resolves to localhost.
	// This is acceptable because the port is bound to localhost only, preventing external access.
	// The key security protection is that the port is NOT bound to 0.0.0.0, which would
	// expose it to the network.

	// Verify the port is NOT listening on all interfaces (0.0.0.0)
	// This is the key security check - the port should only be bound to localhost
	// We verify this by checking that the port binding in the router container
	// uses dockerIP (127.0.0.1) and not 0.0.0.0
	router, err := ddevapp.FindDdevRouter()
	require.NoError(t, err)
	require.NotNil(t, router)

	// Check the actual port bindings on the router container
	foundMonitorPortBinding := false
	for _, port := range router.Ports {
		portStr := strconv.Itoa(int(port.PublicPort))
		if portStr == monitorPort {
			foundMonitorPortBinding = true
			// The IP should be 127.0.0.1 (or equivalent dockerIP), not 0.0.0.0
			// port.IP is a netip.Addr, so convert to string for comparison
			actualIP := port.IP.String()
			assert.Equal(dockerIP, actualIP,
				"Traefik monitor port must be bound to localhost (%s), not all interfaces (0.0.0.0)", dockerIP)
		}
	}
	assert.True(foundMonitorPortBinding, "Monitor port binding should be found in router container")
}

// TestAssignRouterPortsToGenericWebserverPorts ensures that RouterHTTPPort and RouterHTTPSPort
// are assigned correctly based on WebExtraExposedPorts for Generic webservers.
func TestAssignRouterPortsToGenericWebserverPorts(t *testing.T) {
	type testCase struct {
		name                 string
		webserverType        string
		webExtraExposedPorts []ddevapp.WebExposedPort
		expectedHTTPPort     string
		expectedHTTPSPort    string
	}

	tests := []testCase{
		{
			name:          "Generic webserver with valid ports",
			webserverType: nodeps.WebserverGeneric,
			webExtraExposedPorts: []ddevapp.WebExposedPort{
				{HTTPPort: 8080, HTTPSPort: 8443},
			},
			expectedHTTPPort:  "8080",
			expectedHTTPSPort: "8443",
		},
		{
			name:                 "Generic webserver with no extra ports",
			webserverType:        nodeps.WebserverGeneric,
			webExtraExposedPorts: []ddevapp.WebExposedPort{},
			expectedHTTPPort:     "",
			expectedHTTPSPort:    "",
		},
		{
			name:          "Non-Generic webserver should not assign ports",
			webserverType: nodeps.WebserverNginxFPM,
			webExtraExposedPorts: []ddevapp.WebExposedPort{
				{HTTPPort: 8080, HTTPSPort: 8443},
			},
			expectedHTTPPort:  "",
			expectedHTTPSPort: "",
		},
		{
			name:          "Generic webserver with multiple ports uses first",
			webserverType: nodeps.WebserverGeneric,
			webExtraExposedPorts: []ddevapp.WebExposedPort{
				{HTTPPort: 8000, HTTPSPort: 8443},
				{HTTPPort: 8081, HTTPSPort: 8444},
			},
			expectedHTTPPort:  "8000",
			expectedHTTPSPort: "8443",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			app := &ddevapp.DdevApp{
				WebserverType:        tc.webserverType,
				WebExtraExposedPorts: tc.webExtraExposedPorts,
			}

			ddevapp.AssignRouterPortsToGenericWebserverPorts(app)
			require.Equal(t, tc.expectedHTTPPort, app.RouterHTTPPort)
			require.Equal(t, tc.expectedHTTPSPort, app.RouterHTTPSPort)
		})
	}
}

// TestSortWebExtraExposedPorts verifies that WebExtraExposedPorts are sorted
// so the entry matching configured router ports comes first (index 0).
func TestSortWebExtraExposedPorts(t *testing.T) {
	// Save and restore global config
	origHTTPPort := globalconfig.DdevGlobalConfig.RouterHTTPPort
	origHTTPSPort := globalconfig.DdevGlobalConfig.RouterHTTPSPort
	t.Cleanup(func() {
		globalconfig.DdevGlobalConfig.RouterHTTPPort = origHTTPPort
		globalconfig.DdevGlobalConfig.RouterHTTPSPort = origHTTPSPort
	})

	type testCase struct {
		name                 string
		webExtraExposedPorts []ddevapp.WebExposedPort
		appHTTPPort          string // app.RouterHTTPPort
		appHTTPSPort         string // app.RouterHTTPSPort
		globalHTTPPort       string // global config
		globalHTTPSPort      string // global config
		expectedFirstName    string // expected Name of first entry after sort
	}

	tests := []testCase{
		{
			name:                 "Empty slice - no change",
			webExtraExposedPorts: []ddevapp.WebExposedPort{},
			expectedFirstName:    "",
		},
		{
			name: "Single entry - no change",
			webExtraExposedPorts: []ddevapp.WebExposedPort{
				{Name: "only", HTTPPort: 8080, HTTPSPort: 8443},
			},
			expectedFirstName: "only",
		},
		{
			name: "Standard ports (80/443) listed second - moves to first",
			webExtraExposedPorts: []ddevapp.WebExposedPort{
				{Name: "reverb", HTTPPort: 8081, HTTPSPort: 8080},
				{Name: "main", HTTPPort: 80, HTTPSPort: 443},
			},
			expectedFirstName: "main",
		},
		{
			name: "Standard ports listed first - stays first",
			webExtraExposedPorts: []ddevapp.WebExposedPort{
				{Name: "main", HTTPPort: 80, HTTPSPort: 443},
				{Name: "reverb", HTTPPort: 8081, HTTPSPort: 8080},
			},
			expectedFirstName: "main",
		},
		{
			name: "Partial match HTTP 80 only - moves to first",
			webExtraExposedPorts: []ddevapp.WebExposedPort{
				{Name: "other", HTTPPort: 8081, HTTPSPort: 8080},
				{Name: "partial", HTTPPort: 80, HTTPSPort: 8443},
			},
			expectedFirstName: "partial",
		},
		{
			name: "Full match beats partial match",
			webExtraExposedPorts: []ddevapp.WebExposedPort{
				{Name: "partial", HTTPPort: 80, HTTPSPort: 8080},
				{Name: "full", HTTPPort: 80, HTTPSPort: 443},
			},
			expectedFirstName: "full",
		},
		{
			name: "No matching ports - first entry stays first",
			webExtraExposedPorts: []ddevapp.WebExposedPort{
				{Name: "first", HTTPPort: 8081, HTTPSPort: 8080},
				{Name: "second", HTTPPort: 3000, HTTPSPort: 3443},
			},
			expectedFirstName: "first",
		},
		{
			name: "App config ports preferred over defaults",
			webExtraExposedPorts: []ddevapp.WebExposedPort{
				{Name: "default", HTTPPort: 80, HTTPSPort: 443},
				{Name: "custom", HTTPPort: 8080, HTTPSPort: 8443},
			},
			appHTTPPort:       "8080",
			appHTTPSPort:      "8443",
			expectedFirstName: "custom",
		},
		{
			name: "Global config ports preferred over defaults",
			webExtraExposedPorts: []ddevapp.WebExposedPort{
				{Name: "default", HTTPPort: 80, HTTPSPort: 443},
				{Name: "global", HTTPPort: 9080, HTTPSPort: 9443},
			},
			globalHTTPPort:    "9080",
			globalHTTPSPort:   "9443",
			expectedFirstName: "global",
		},
		{
			name: "App config takes priority over global config",
			webExtraExposedPorts: []ddevapp.WebExposedPort{
				{Name: "global", HTTPPort: 9080, HTTPSPort: 9443},
				{Name: "app", HTTPPort: 8080, HTTPSPort: 8443},
			},
			appHTTPPort:       "8080",
			appHTTPSPort:      "8443",
			globalHTTPPort:    "9080",
			globalHTTPSPort:   "9443",
			expectedFirstName: "app",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Set global config
			globalconfig.DdevGlobalConfig.RouterHTTPPort = tc.globalHTTPPort
			globalconfig.DdevGlobalConfig.RouterHTTPSPort = tc.globalHTTPSPort

			app := &ddevapp.DdevApp{
				WebExtraExposedPorts: tc.webExtraExposedPorts,
				RouterHTTPPort:       tc.appHTTPPort,
				RouterHTTPSPort:      tc.appHTTPSPort,
			}

			ddevapp.SortWebExtraExposedPorts(app)

			if tc.expectedFirstName == "" {
				require.Empty(t, app.WebExtraExposedPorts)
			} else {
				require.NotEmpty(t, app.WebExtraExposedPorts)
				require.Equal(t, tc.expectedFirstName, app.WebExtraExposedPorts[0].Name,
					"Expected entry '%s' to be first after sort", tc.expectedFirstName)
			}
		})
	}
}

// TestPortsMatch tests the PortsMatch function
func TestPortsMatch(t *testing.T) {
	tests := []struct {
		name          string
		existingPorts []string
		neededPorts   []string
		expected      bool
	}{
		{
			name:          "empty slices match",
			existingPorts: []string{},
			neededPorts:   []string{},
			expected:      true,
		},
		{
			name:          "same ports match",
			existingPorts: []string{"80", "443"},
			neededPorts:   []string{"80", "443"},
			expected:      true,
		},
		{
			name:          "same ports different order match",
			existingPorts: []string{"443", "80"},
			neededPorts:   []string{"80", "443"},
			expected:      true,
		},
		{
			name:          "different ports don't match",
			existingPorts: []string{"80", "443"},
			neededPorts:   []string{"80", "8443"},
			expected:      false,
		},
		{
			name:          "missing needed port doesn't match",
			existingPorts: []string{"80", "443"},
			neededPorts:   []string{"80", "443", "8080"},
			expected:      false,
		},
		{
			name:          "router with extra ports still matches",
			existingPorts: []string{"80", "443", "8080"},
			neededPorts:   []string{"80", "443"},
			expected:      true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := ddevapp.PortsMatch(tc.existingPorts, tc.neededPorts)
			require.Equal(t, tc.expected, result)
		})
	}
}

// TestRouterNotRebuiltWithExtraPorts verifies that when a project with extra ports
// is running and a simpler project starts, the router is not recreated.
// The router should only be recreated when NEW ports are needed, not when it has
// extra ports from other projects.
func TestRouterNotRebuiltWithExtraPorts(t *testing.T) {
	if dockerutil.IsRancherDesktop() {
		t.Skip("Rancher Desktop starts extra project with ephemeral ports, not default ones, causing test instability")
	}
	// Start clean
	ddevapp.PowerOff()

	// Create a temporary project with extra exposed ports
	extraPortsDir := testcommon.CreateTmpDir(t.Name() + "_extraports")
	t.Cleanup(func() {
		_ = os.RemoveAll(extraPortsDir)
	})

	extraPortsApp, err := ddevapp.NewApp(extraPortsDir, true)
	require.NoError(t, err)
	extraPortsApp.Name = t.Name() + "-extraports"
	extraPortsApp.Type = nodeps.AppTypePHP
	// Add extra exposed ports that will be unique to this project
	extraPortsApp.WebExtraExposedPorts = []ddevapp.WebExposedPort{
		{Name: "extra1", WebContainerPort: 3000, HTTPPort: 3080, HTTPSPort: 3443},
		{Name: "extra2", WebContainerPort: 4000, HTTPPort: 4080, HTTPSPort: 4443},
	}
	err = extraPortsApp.WriteConfig()
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = extraPortsApp.Stop(true, false)
	})

	// Start the extra ports project - this creates the router with extra ports
	err = extraPortsApp.Start()
	require.NoError(t, err)

	// Get the router's bound ports after starting the extra ports project
	router, err := ddevapp.FindDdevRouter()
	require.NoError(t, err)
	portsAfterExtraProject, err := dockerutil.GetBoundHostPorts(router.ID)
	require.NoError(t, err)

	// Verify the extra ports are bound
	require.Contains(t, portsAfterExtraProject, "3080", "Router should have extra HTTP port 3080")
	require.Contains(t, portsAfterExtraProject, "3443", "Router should have extra HTTPS port 3443")
	require.Contains(t, portsAfterExtraProject, "4080", "Router should have extra HTTP port 4080")
	require.Contains(t, portsAfterExtraProject, "4443", "Router should have extra HTTPS port 4443")

	// Now start a simpler project (TestSites[0]) that doesn't need those extra ports
	site := TestSites[0]
	simpleApp, err := ddevapp.NewApp(site.Dir, false)
	require.NoError(t, err)
	// Clear any extra ports from previous test runs
	simpleApp.WebExtraExposedPorts = nil
	err = simpleApp.WriteConfig()
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = simpleApp.Stop(true, false)
	})

	err = simpleApp.Start()
	require.NoError(t, err)

	// Get the router's bound ports after starting the simple project
	router, err = ddevapp.FindDdevRouter()
	require.NoError(t, err)
	portsAfterSimpleProject, err := dockerutil.GetBoundHostPorts(router.ID)
	require.NoError(t, err)

	// The router should still have the extra ports from the first project
	// This proves the router was not recreated when the simple project started
	require.Contains(t, portsAfterSimpleProject, "3080", "Router should still have extra HTTP port 3080 after starting simple project")
	require.Contains(t, portsAfterSimpleProject, "3443", "Router should still have extra HTTPS port 3443 after starting simple project")
	require.Contains(t, portsAfterSimpleProject, "4080", "Router should still have extra HTTP port 4080 after starting simple project")
	require.Contains(t, portsAfterSimpleProject, "4443", "Router should still have extra HTTPS port 4443 after starting simple project")

	// Verify the port lists are identical (router wasn't recreated)
	require.ElementsMatch(t, portsAfterExtraProject, portsAfterSimpleProject,
		"Router ports should be unchanged after starting simple project - router should not have been recreated")
}

// TestPausedProjectsExcludedFromRouter verifies that when a project is paused,
// its Traefik configuration is removed from the router when another project starts.
// This simulates the scenario where Docker restarts and projects end up paused,
// then only the started project should have its config in the router.
func TestPausedProjectsExcludedFromRouter(t *testing.T) {
	// Start clean
	ddevapp.PowerOff()

	// Create two temporary projects
	project1Dir := testcommon.CreateTmpDir(t.Name() + "_project1")
	project2Dir := testcommon.CreateTmpDir(t.Name() + "_project2")

	t.Cleanup(func() {
		_ = os.RemoveAll(project1Dir)
		_ = os.RemoveAll(project2Dir)
	})

	// Set up project 1
	app1, err := ddevapp.NewApp(project1Dir, true)
	require.NoError(t, err)
	app1.Name = t.Name() + "-project1"
	app1.Type = nodeps.AppTypePHP
	err = app1.WriteConfig()
	require.NoError(t, err)

	// Set up project 2
	app2, err := ddevapp.NewApp(project2Dir, true)
	require.NoError(t, err)
	app2.Name = t.Name() + "-project2"
	app2.Type = nodeps.AppTypePHP
	err = app2.WriteConfig()
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = app1.Stop(true, false)
		_ = app2.Stop(true, false)
		_ = dockerutil.RemoveContainer(nodeps.RouterContainer)
	})

	// Start both projects
	err = app1.Start()
	require.NoError(t, err)
	err = app2.Start()
	require.NoError(t, err)

	// Verify both project configs exist in the router
	configDir := "/mnt/ddev-global-cache/traefik/config"
	stdout, _, err := dockerutil.Exec(nodeps.RouterContainer, "ls "+configDir, "")
	require.NoError(t, err, "failed to list router config directory")

	require.Contains(t, stdout, app1.Name+"_merged.yaml",
		"Router should have config for project1 after both projects started")
	require.Contains(t, stdout, app2.Name+"_merged.yaml",
		"Router should have config for project2 after both projects started")

	// Pause project1 (simulates Docker restart leaving containers stopped)
	err = app1.Pause()
	require.NoError(t, err)

	// Verify project1 is paused
	status, _ := app1.SiteStatus()
	require.Equal(t, ddevapp.SitePaused, status, "Project1 should be paused")

	// Restart project2 - this triggers PushGlobalTraefikConfig which should
	// exclude the paused project1
	err = app2.Restart()
	require.NoError(t, err)

	// Verify only project2's config exists in the router now
	stdout, _, err = dockerutil.Exec(nodeps.RouterContainer, "ls "+configDir, "")
	require.NoError(t, err, "failed to list router config directory after restart")

	require.NotContains(t, stdout, app1.Name+"_merged.yaml",
		"Router should NOT have config for paused project1 after project2 restart")
	require.Contains(t, stdout, app2.Name+"_merged.yaml",
		"Router should still have config for running project2")

	// Verify project2 is still accessible
	status, _ = app2.SiteStatus()
	require.Equal(t, ddevapp.SiteRunning, status, "Project2 should be running")
}
