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
	if dockerutil.IsLima() || dockerutil.IsColima() {
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

	util.Debug("Before app.Restart(): app.RouterHTTPPort=%s, app.RouterHTTPSPort=%s, app.GetRouterHTTPPort()=%s app.GetRouterHTTPSPort=%s", app.RouterHTTPPort, app.RouterHTTPSPort, app.GetRouterHTTPPort(), app.GetRouterHTTPSPort())
	err = app.Restart()
	util.Debug("After app.Restart(): app.RouterHTTPPort=%s, app.RouterHTTPSPort=%s, app.GetRouterHTTPPort()=%s app.GetRouterHTTPSPort=%s", app.RouterHTTPPort, app.RouterHTTPSPort, app.GetRouterHTTPPort(), app.GetRouterHTTPSPort())

	require.NoError(t, err)
	require.Equal(t, globalconfig.DdevGlobalConfig.RouterHTTPPort, app.GetRouterHTTPPort())
	require.Equal(t, globalconfig.DdevGlobalConfig.RouterHTTPSPort, app.GetRouterHTTPSPort())

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
	if dockerutil.IsColima() || dockerutil.IsLima() || dockerutil.IsRancherDesktop() {
		// Intermittent failures in CI due apparently to https://github.com/lima-vm/lima/issues/2536
		// Expected port is not available, so it allocates another one.
		t.Skip("Skipping on Lima/Colima/Rancher as ports don't seem to be released properly in a timely fashion")
	}

	targetHTTPPort, targetHTTPSPort := "28080", "28443"
	const testString = "Hello from TestUseEphemeralPort"

	apps := []*ddevapp.DdevApp{}
	for _, s := range []string{"site1", "site2"} {
		site := filepath.Join(testcommon.CreateTmpDir(t.Name()), t.Name()+s)
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
	for _, p := range []string{apps[0].GetRouterHTTPPort(), apps[0].GetRouterHTTPSPort(), apps[0].GetMailpitHTTPPort(), apps[0].GetMailpitHTTPSPort()} {
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

		// app1 will not use the configured target ports, uses the assigned ephemeral ports.
		require.NotEqual(t, targetHTTPPort, app.GetRouterHTTPPort())
		require.NotEqual(t, targetHTTPSPort, app.GetRouterHTTPSPort())

		require.Equal(t, fmt.Sprint(expectedEphemeralHTTPPort), app.GetRouterHTTPPort())
		require.Equal(t, fmt.Sprint(expectedEphemeralHTTPSPort), app.GetRouterHTTPSPort())

		// Make sure that both http and https URLs have proper content
		_, _ = testcommon.EnsureLocalHTTPContent(t, app.GetHTTPURL(), testString, 0)
		if globalconfig.GetCAROOT() != "" {
			_, _ = testcommon.EnsureLocalHTTPContent(t, app.GetHTTPSURL(), testString, 0)
		}
	}

}
