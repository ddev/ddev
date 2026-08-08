package ddevapp_test

import (
	"fmt"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

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
	if nodeps.IsEnvFalse("DDEV_RUN_TEST_ANYWAY") && (dockerutil.IsLima() || dockerutil.IsColima() || dockerutil.IsRancherDesktop() || dockerutil.IsPodmanRootlessmacOS()) {
		// Intermittent failures in CI due apparently to https://github.com/lima-vm/lima/issues/2536
		// Expected port is not available, so it allocates another one.
		t.Skip("Lima/Colima/Rancher/Podman-rootless-macOS often allocate another port, so skip")
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

// isDockerHostPortRaceError detects a Docker-level host port bind collision.
//
// The db and web services publish some ports with no fixed host port
// (for example "127.0.0.1::3306"), so Docker picks a host port from the kernel's
// local port range (32768-60999 on Linux). Docker does not reserve that port, and
// under rootless Docker the bind happens later still, in RootlessKit's port
// manager. So the chosen port can be taken by an unrelated socket, or not yet
// released by a container that was just removed, by the time the bind runs.
//
// This is unrelated to DDEV's own ephemeral router port allocation; it is a
// property of the environment, so retrying is the appropriate response.
func isDockerHostPortRaceError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	// rootless Docker/Podman via RootlessKit, and rootful Docker, word this differently.
	return strings.Contains(msg, "address already in use") ||
		strings.Contains(msg, "port is already allocated")
}

// startRetryingHostPortRace calls app.Start(), retrying with a short backoff if it
// hits isDockerHostPortRaceError, since the port holder normally releases within a
// second or two and the next attempt draws a different random host port.
//
// It deliberately does not call app.Stop() between attempts: Stop() clears
// ddevapp.EphemeralRouterPortsAssigned, which would let a later project be handed
// an ephemeral router port already bound by the router for an earlier project.
func startRetryingHostPortRace(t *testing.T, app *ddevapp.DdevApp) error {
	const attempts = 3
	var err error
	for attempt := 1; attempt <= attempts; attempt++ {
		err = app.Start()
		if err == nil || !isDockerHostPortRaceError(err) {
			return err
		}
		if attempt < attempts {
			t.Logf("app.Start() attempt %d/%d hit a Docker host port collision, retrying: %v", attempt, attempts, err)
			time.Sleep(2 * time.Second)
		}
	}
	return err
}

// Test that the app assigns an ephemeral port if the default one is not available.
func TestUseEphemeralPort(t *testing.T) {
	if nodeps.IsEnvFalse("DDEV_RUN_TEST_ANYWAY") && (dockerutil.IsColima() || dockerutil.IsLima() || dockerutil.IsRancherDesktop()) {
		// Intermittent failures in CI due apparently to https://github.com/lima-vm/lima/issues/2536
		// Expected port is not available, so it allocates another one.
		t.Skip("Skipping on Lima/Colima/Rancher as ports don't seem to be released properly in a timely fashion")
	}
	if nodeps.IsAppleSilicon() && dockerutil.IsDockerDesktop() && nodeps.IsEnvFalse("DDEV_RUN_TEST_ANYWAY") {
		t.Skip("Skipping on Docker Desktop/Apple Silicon to ignore problems with 'connection reset by peer'")
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

	// Tracks ephemeral ports already handed out, so we can verify each project gets
	// its own. Maps port number to a description of what claimed it.
	assignedPorts := map[int]string{}

	for i, app := range apps {
		err := startRetryingHostPortRace(t, app)
		require.NoError(t, err)

		// Get a new copy of the app to make sure we have up-to-date port information
		app, err = ddevapp.NewApp(app.GetAppRoot(), true)
		require.NoError(t, err)

		// The project must not use the configured target ports, since those are occupied.
		require.NotEqual(t, targetHTTPPort, app.GetPrimaryRouterHTTPPort())
		require.NotEqual(t, targetHTTPSPort, app.GetPrimaryRouterHTTPSPort())

		// Don't predict the exact port numbers. Which ephemeral port a project lands on
		// depends on what else on the machine happens to hold ports in the range at that
		// moment, so all that matters is that each replacement port is in the ephemeral
		// range and is not one already given to another project. That the port actually
		// works is proven by the content checks below.
		for _, p := range []struct{ scheme, port string }{
			{"HTTP", app.GetPrimaryRouterHTTPPort()},
			{"HTTPS", app.GetPrimaryRouterHTTPSPort()},
		} {
			portNum, err := strconv.Atoi(p.port)
			require.NoError(t, err)
			require.GreaterOrEqual(t, portNum, ddevapp.MinEphemeralPort,
				"app %d (%s) %s port %d is below the ephemeral range", i, app.Name, p.scheme, portNum)
			require.LessOrEqual(t, portNum, ddevapp.MaxEphemeralPort,
				"app %d (%s) %s port %d is above the ephemeral range", i, app.Name, p.scheme, portNum)
			claimant := fmt.Sprintf("app %d (%s) %s", i, app.Name, p.scheme)
			require.NotContains(t, assignedPorts, portNum,
				"%s got port %d, which was already assigned to %s", claimant, portNum, assignedPorts[portNum])
			assignedPorts[portNum] = claimant
		}

		// Make sure that both http and https URLs have proper content
		testcommon.AssertLocalHTTPContent(t, app.GetHTTPURL(), testString,
			testcommon.WithMessagef("project should serve expected content over HTTP on its ephemeral port"),
			testcommon.WithTimeout(0),
		)
		require.Contains(t, app.GetHTTPURL(), app.GetHostname())
		if globalconfig.GetCAROOT() != "" {
			testcommon.AssertLocalHTTPContent(t, app.GetHTTPSURL(), testString,
				testcommon.WithMessagef("project should serve expected content over HTTPS on its ephemeral port"),
				testcommon.WithTimeout(0),
			)
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
	if nodeps.IsEnvFalse("DDEV_RUN_TEST_ANYWAY") && (dockerutil.IsColima() || dockerutil.IsLima() || dockerutil.IsRancherDesktop()) {
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

// TestRouterPortSubstitutionPersistsAcrossProjects tests that when a router port
// is substituted with an ephemeral port because the standard port was busy when
// the router was created, a later, different project still adopts the same
// substitute after the original conflict clears - recovered from the router's
// RouterPortSubstitutionsLabel - instead of expecting the router to bind the
// now-free standard port and forcing an unnecessary router recreation.
func TestRouterPortSubstitutionPersistsAcrossProjects(t *testing.T) {
	if os.Getenv("GOTEST_SHORT") != "" {
		t.Skip("Skipping because GOTEST_SHORT is set")
	}
	if nodeps.IsEnvFalse("DDEV_RUN_TEST_ANYWAY") && (dockerutil.IsColima() || dockerutil.IsLima() || dockerutil.IsRancherDesktop()) {
		t.Skip("Skipping on Lima/Colima/Rancher as ports don't seem to be released properly in a timely fashion")
	}
	// util.CaptureUserOut() below redirects output to an os.Pipe() that isn't
	// drained until getOutput() is called, i.e. only after app2.Start()
	// returns. With DDEV_DEBUG=true a full Start() can emit enough debug
	// output to fill that pipe before it returns, deadlocking the write on
	// Windows. Same root cause as the TestDdevLogs and
	// TestRouterNotRebuiltOnHostnameChange Windows skips.
	if nodeps.IsWindows() {
		t.Skip("Skipping on Windows because CaptureUserOut() can hang around a full app.Start()")
	}

	// Stop all projects and the router first so we can occupy the ports they would normally use
	ddevapp.PowerOff()
	ddevapp.EphemeralRouterPortsAssigned = make(map[int]bool)
	ddevapp.RouterPortEphemeralSubstitutions = make(map[string]string)

	targetHTTPPort, targetHTTPSPort := "39080", "39443"

	site1 := filepath.Join(testcommon.CreateTmpDir(t.Name() + "-project1"))
	_ = os.MkdirAll(site1, 0755)
	err := fileutil.TemplateStringToFile("Hello from project1", nil, filepath.Join(site1, "index.html"))
	require.NoError(t, err)
	app1, err := ddevapp.NewApp(site1, false)
	require.NoError(t, err)
	app1.Name = t.Name() + "-project1"
	app1.RouterHTTPPort, app1.RouterHTTPSPort = targetHTTPPort, targetHTTPSPort
	require.NoError(t, app1.WriteConfig())

	site2 := filepath.Join(testcommon.CreateTmpDir(t.Name() + "-project2"))
	_ = os.MkdirAll(site2, 0755)
	err = fileutil.TemplateStringToFile("Hello from project2", nil, filepath.Join(site2, "index.html"))
	require.NoError(t, err)
	app2, err := ddevapp.NewApp(site2, false)
	require.NoError(t, err)
	app2.Name = t.Name() + "-project2"
	app2.RouterHTTPPort, app2.RouterHTTPSPort = targetHTTPPort, targetHTTPSPort
	require.NoError(t, app2.WriteConfig())

	// Occupy the target ports so project1 is forced onto ephemeral substitutes
	var listeners []net.Listener
	for _, p := range []string{targetHTTPPort, targetHTTPSPort} {
		listener, err := net.Listen("tcp", "127.0.0.1:"+p)
		require.NoError(t, err)
		listeners = append(listeners, listener)
	}
	listenersClosed := false
	closeListeners := func() {
		if listenersClosed {
			return
		}
		for _, l := range listeners {
			_ = l.Close()
		}
		listenersClosed = true
	}

	t.Cleanup(func() {
		closeListeners()
		_ = app1.Stop(true, false)
		_ = app2.Stop(true, false)
		_ = os.RemoveAll(site1)
		_ = os.RemoveAll(site2)
		_ = dockerutil.RemoveContainer(nodeps.RouterContainer)
	})

	// Start project1 while the target ports are busy - it should be forced onto
	// ephemeral substitutes, and the router should record the substitution.
	require.NoError(t, app1.Start())

	app1, err = ddevapp.NewApp(app1.GetAppRoot(), true)
	require.NoError(t, err)
	substituteHTTPPort := app1.GetPrimaryRouterHTTPPort()
	substituteHTTPSPort := app1.GetPrimaryRouterHTTPSPort()
	require.NotEqual(t, targetHTTPPort, substituteHTTPPort, "HTTP port should be ephemeral")
	require.NotEqual(t, targetHTTPSPort, substituteHTTPSPort, "HTTPS port should be ephemeral")

	router, err := ddevapp.FindDdevRouter()
	require.NoError(t, err)
	originalRouterID := router.ID

	labelValue := router.Labels[ddevapp.RouterPortSubstitutionsLabel]
	require.Contains(t, labelValue, targetHTTPPort+"="+substituteHTTPPort,
		"router label should record the HTTP port substitution, got %q", labelValue)
	require.Contains(t, labelValue, targetHTTPSPort+"="+substituteHTTPSPort,
		"router label should record the HTTPS port substitution, got %q", labelValue)

	// Release the target ports: whatever was occupying them is gone now, so a
	// fresh negotiation would see them as free.
	closeListeners()

	// Simulate project2 starting from a brand-new ddev process: clear the
	// in-process substitution cache so the router's label is the only place
	// left the substitution could be recovered from.
	ddevapp.RouterPortEphemeralSubstitutions = make(map[string]string)

	reuseMessage := nodeps.RouterContainer + " already running, pushing new config"
	recreateMessage := "Starting " + nodeps.RouterContainer + ", pushing config"

	getOutput := util.CaptureUserOut()
	err = app2.Start()
	startOutput := getOutput()
	require.NoError(t, err)

	require.Contains(t, startOutput, reuseMessage,
		"project2 should reuse the running router, output: %s", startOutput)
	require.NotContains(t, startOutput, recreateMessage,
		"project2 should NOT recreate the router just because the standard port freed up, output: %s", startOutput)

	app2, err = ddevapp.NewApp(app2.GetAppRoot(), true)
	require.NoError(t, err)
	require.Equal(t, substituteHTTPPort, app2.GetPrimaryRouterHTTPPort(),
		"project2 should adopt the same ephemeral substitute recorded on the router, not the now-free standard port")
	require.Equal(t, substituteHTTPSPort, app2.GetPrimaryRouterHTTPSPort())

	router, err = ddevapp.FindDdevRouter()
	require.NoError(t, err)
	require.Equal(t, originalRouterID, router.ID,
		"router should not be recreated when a later project's proposed port frees up but was previously substituted")
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
	if nodeps.IsAppleSilicon() && dockerutil.IsDockerDesktop() && nodeps.IsEnvFalse("DDEV_RUN_TEST_ANYWAY") {
		t.Skip("Skipping on Docker Desktop/Apple Silicon to ignore problems with 'connection reset by peer'")
	}
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

	// The Traefik monitor port should ALWAYS be bound to 127.0.0.1,
	// never to all interfaces (0.0.0.0) or a remote Docker IP
	monitorPort := globalconfig.DdevGlobalConfig.TraefikMonitorPort

	// Expected format: "127.0.0.1:10999:10999"
	expectedBinding := "127.0.0.1:" + monitorPort + ":" + monitorPort

	assert.Contains(content, expectedBinding,
		"Traefik monitor port must always be bound to 127.0.0.1, not all interfaces")

	// Make sure it's NOT bound to all interfaces
	allInterfacesBinding := "- \"" + monitorPort + ":" + monitorPort + "\""
	assert.NotContains(content, allInterfacesBinding,
		"Traefik monitor port must NOT be bound to all interfaces (security risk)")

	// Test that the dashboard is accessible via localhost
	localhostDashboardURL := "http://127.0.0.1:" + monitorPort + "/api/overview"
	testcommon.AssertLocalHTTPContent(t, localhostDashboardURL, "",
		testcommon.WithMessagef("Traefik dashboard should be accessible via localhost at %s", localhostDashboardURL),
	)

	// Note: The dashboard may also be accessible via project hostnames on the monitor port
	// (e.g., http://project.ddev.site:10999) because the hostname resolves to localhost.
	// This is acceptable because the port is bound to localhost only, preventing external access.
	// The key security protection is that the port is NOT bound to 0.0.0.0, which would
	// expose it to the network.

	// Verify the port is NOT listening on all interfaces (0.0.0.0)
	// This is the key security check - the port should only be bound to localhost
	// We verify this by checking that the port binding in the router container
	// uses 127.0.0.1 and not 0.0.0.0
	router, err := ddevapp.FindDdevRouter()
	require.NoError(t, err)
	require.NotNil(t, router)

	// Check the actual port bindings on the router container
	foundMonitorPortBinding := false
	for _, port := range router.Ports {
		portStr := strconv.Itoa(int(port.PublicPort))
		if portStr == monitorPort {
			foundMonitorPortBinding = true
			// The IP should be 127.0.0.1, not 0.0.0.0 or a remote Docker IP
			actualIP := port.IP.String()
			assert.Equal("127.0.0.1", actualIP,
				"Traefik monitor port must be bound to 127.0.0.1, not all interfaces (0.0.0.0)")
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

// TestHostnamesMatch tests the HostnamesMatch function
func TestHostnamesMatch(t *testing.T) {
	tests := []struct {
		name              string
		existingHostnames []string
		neededHostnames   []string
		expected          bool
	}{
		{
			name:              "empty slices match",
			existingHostnames: []string{},
			neededHostnames:   []string{},
			expected:          true,
		},
		{
			name:              "same hostnames match",
			existingHostnames: []string{"test1.ddev.site", "test2.ddev.site"},
			neededHostnames:   []string{"test1.ddev.site", "test2.ddev.site"},
			expected:          true,
		},
		{
			name:              "same hostnames different order match",
			existingHostnames: []string{"test2.ddev.site", "test1.ddev.site"},
			neededHostnames:   []string{"test1.ddev.site", "test2.ddev.site"},
			expected:          true,
		},
		{
			name:              "different hostnames don't match",
			existingHostnames: []string{"test1.ddev.site", "test2.ddev.site"},
			neededHostnames:   []string{"test1.ddev.site", "test3.ddev.site"},
			expected:          false,
		},
		{
			name:              "missing needed hostname doesn't match",
			existingHostnames: []string{"test1.ddev.site", "test2.ddev.site"},
			neededHostnames:   []string{"test1.ddev.site", "test2.ddev.site", "test3.ddev.site"},
			expected:          false,
		},
		{
			name:              "router with extra hostnames still matches",
			existingHostnames: []string{"test1.ddev.site", "test2.ddev.site", "test3.ddev.site"},
			neededHostnames:   []string{"test1.ddev.site", "test2.ddev.site"},
			expected:          true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := ddevapp.HostnamesMatch(tc.existingHostnames, tc.neededHostnames)
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

// TestRouterNotRebuiltOnHostnameChange verifies that starting an additional
// project (which introduces a new network alias/hostname) and restarting a
// project update the router's network aliases in place rather than recreating
// the router container.
func TestRouterNotRebuiltOnHostnameChange(t *testing.T) {
	// Windows can't handle this test, util.CaptureUserOut() can deadlock around a full app.Start()/Restart(), see #8644
	if nodeps.IsEnvFalse("DDEV_RUN_TEST_ANYWAY") && nodeps.IsWindows() {
		t.Skip("Skipping TestRouterNotRebuiltOnHostnameChange on Windows")
	}

	// Start clean.
	ddevapp.PowerOff()

	project1Dir := testcommon.CreateTmpDir(t.Name() + "_project1")
	project2Dir := testcommon.CreateTmpDir(t.Name() + "_project2")

	app1, err := ddevapp.NewApp(project1Dir, true)
	require.NoError(t, err)
	app1.Name = t.Name() + "-project1"
	app1.Type = nodeps.AppTypePHP
	require.NoError(t, app1.WriteConfig())

	app2, err := ddevapp.NewApp(project2Dir, true)
	require.NoError(t, err)
	app2.Name = t.Name() + "-project2"
	app2.Type = nodeps.AppTypePHP
	require.NoError(t, app2.WriteConfig())

	t.Cleanup(func() {
		_ = app1.Stop(true, false)
		_ = app2.Stop(true, false)
		_ = os.RemoveAll(project1Dir)
		_ = os.RemoveAll(project2Dir)
		_ = dockerutil.RemoveContainer(nodeps.RouterContainer)
	})

	// The message the router prints when it reuses the running container versus
	// the message it prints when it (re)creates the container.
	reuseMessage := nodeps.RouterContainer + " already running, pushing new config"
	recreateMessage := "Starting " + nodeps.RouterContainer + ", pushing config"

	// First project start creates the router from scratch.
	require.NoError(t, app1.Start())

	router, err := ddevapp.FindDdevRouter()
	require.NoError(t, err)
	require.NotNil(t, router)
	originalRouterID := router.ID

	// Starting the second project adds a new hostname (network alias). The router
	// should be updated in place, not recreated.
	getOutput := util.CaptureUserOut()
	err = app2.Start()
	startOutput := getOutput()
	require.NoError(t, err)

	require.Contains(t, startOutput, reuseMessage,
		"second project start should reuse the running router, output: %s", startOutput)
	require.NotContains(t, startOutput, recreateMessage,
		"second project start should NOT recreate the router, output: %s", startOutput)

	router, err = ddevapp.FindDdevRouter()
	require.NoError(t, err)
	require.Equal(t, originalRouterID, router.ID,
		"router should not be recreated when a second project adds a new hostname")

	// The router's network aliases should now include both projects' hostnames,
	// proving the aliases were updated in place.
	aliases, err := dockerutil.GetRouterNetworkAliases(router.ID)
	require.NoError(t, err)
	require.Contains(t, aliases, app1.GetHostname(),
		"router aliases should include the first project hostname")
	require.Contains(t, aliases, app2.GetHostname(),
		"router aliases should include the second project hostname after in-place update")

	// Restarting a project must also reuse the running router.
	getOutput = util.CaptureUserOut()
	err = app2.Restart()
	restartOutput := getOutput()
	require.NoError(t, err)

	require.Contains(t, restartOutput, reuseMessage,
		"restart should reuse the running router, output: %s", restartOutput)
	require.NotContains(t, restartOutput, recreateMessage,
		"restart should NOT recreate the router, output: %s", restartOutput)

	router, err = ddevapp.FindDdevRouter()
	require.NoError(t, err)
	require.Equal(t, originalRouterID, router.ID,
		"router should not be recreated when a project restarts")
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
