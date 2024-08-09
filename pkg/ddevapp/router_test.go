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
	"github.com/ddev/ddev/pkg/exec"
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
		t.Skip("Lima and Colima allocate another port, so skip")
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

// Do a modest test of Lets Encrypt functionality
// This checks to see that Certbot ran and populated /etc/letsencrypt and
// that /etc/letsencrypt is mounted on volume.
func TestLetsEncrypt(t *testing.T) {
	if globalconfig.DdevGlobalConfig.IsTraefikRouter() {
		t.Skip("Skipping because router=traefik set and not yet supported")
	}
	assert := asrt.New(t)

	savedGlobalconfig := globalconfig.DdevGlobalConfig

	globalconfig.DdevGlobalConfig.UseLetsEncrypt = true
	globalconfig.DdevGlobalConfig.LetsEncryptEmail = "nobody@example.com"
	globalconfig.DdevGlobalConfig.RouterBindAllInterfaces = true
	err := globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig)
	require.NoError(t, err)

	site := TestSites[0]
	switchDir := site.Chdir()
	defer switchDir()

	// Force router stop so it will start up with Lets Encrypt mount
	dest := ddevapp.RouterComposeYAMLPath()
	_, _, err = dockerutil.ComposeCmd(&dockerutil.ComposeCmdOpts{
		ComposeFiles: []string{dest},
		Action:       []string{"-p", ddevapp.RouterComposeProjectName, "down"},
	})
	assert.NoError(err)

	err = dockerutil.RemoveVolume("ddev-router-letsencrypt")
	assert.NoError(err)

	app, err := ddevapp.NewApp(site.Dir, false)
	assert.NoError(err)
	err = app.Start()
	assert.NoError(err)

	t.Cleanup(func() {
		globalconfig.DdevGlobalConfig = savedGlobalconfig
		err = globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig)
		assert.NoError(err)
		_, _, err = dockerutil.ComposeCmd(&dockerutil.ComposeCmdOpts{
			ComposeFiles: []string{dest},
			Action:       []string{"-p", ddevapp.RouterComposeProjectName, "down"},
		})
		assert.NoError(err)
		err = app.Stop(true, false)
		assert.NoError(err)
		err = dockerutil.RemoveVolume("ddev-router-letsencrypt")
		assert.NoError(err)
	})

	container, err := dockerutil.FindContainerByName("ddev-router")
	require.NoError(t, err)
	require.NotNil(t, container)

	stdout, _, err := dockerutil.Exec(container.ID, "df -T /etc/letsencrypt  | awk 'NR==2 {print $7;}'", "")
	assert.NoError(err)
	stdout = strings.Trim(stdout, "\r\n")

	assert.Equal("/etc/letsencrypt", stdout)

	_, _, err = dockerutil.Exec(container.ID, "test -f /etc/letsencrypt/options-ssl-nginx.conf", "")
	assert.NoError(err)
}

// TestRouterConfigOverride tests that the ~/.ddev/.router-compose.yaml can be overridden
// with ~/.ddev/router-compose.*.yaml
func TestRouterConfigOverride(t *testing.T) {
	assert := asrt.New(t)
	origDir, _ := os.Getwd()
	testDir := testcommon.CreateTmpDir(t.Name())
	_ = os.Chdir(testDir)
	overrideYaml := filepath.Join(globalconfig.GetGlobalDdevDir(), "router-compose.override.yaml")

	testcommon.ClearDockerEnv()

	app, err := ddevapp.NewApp(testDir, true)
	assert.NoError(err)
	err = app.WriteConfig()
	assert.NoError(err)
	err = fileutil.CopyFile(filepath.Join(origDir, "testdata", t.Name(), "router-compose.override.yaml"), overrideYaml)
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
		_ = os.Remove(overrideYaml)
	})

	err = app.Start()
	assert.NoError(err)

	stdout, _, err := dockerutil.Exec("ddev-router", "bash -c 'echo $ANSWER'", "")
	assert.Equal(answer+"\n", stdout)
}

// TestDisableHTTP2 tests we can enable or disable http2
func TestDisableHTTP2(t *testing.T) {
	if nodeps.IsAppleSilicon() {
		t.Skip("Skipping on mac M1 to ignore problems with 'connection reset by peer'")
	}
	if globalconfig.GetCAROOT() == "" {
		t.Skip("Skipping because mkcert/http not enabled")
	}
	if globalconfig.DdevGlobalConfig.IsTraefikRouter() {
		t.Skip("Skipping because router=traefik doesn't have feature to turn off http/2")
	}

	assert := asrt.New(t)
	pwd, _ := os.Getwd()
	testDir := testcommon.CreateTmpDir(t.Name())
	_ = os.Chdir(testDir)
	err := os.WriteFile("index.html", []byte("hello from the test"), 0644)
	assert.NoError(err)
	testcommon.ClearDockerEnv()

	_, err = exec.RunCommand(DdevBin, []string{"poweroff"})
	require.NoError(t, err)

	app, err := ddevapp.NewApp(testDir, true)
	assert.NoError(err)
	err = app.WriteConfig()
	assert.NoError(err)

	t.Cleanup(func() {
		globalconfig.DdevGlobalConfig.DisableHTTP2 = false
		err = app.Stop(true, false)
		assert.NoError(err)
		err = os.Chdir(pwd)
		assert.NoError(err)
		_ = os.RemoveAll(testDir)
		assert.NoError(err)
	})

	err = app.Start()
	assert.NoError(err)

	// Verify that http2 is on by default
	out, err := exec.RunCommand("bash", []string{"-c", "curl -k -s -L -I " + app.GetPrimaryURL() + "| head -1"})
	assert.NoError(err, "failed to curl, err=%v out=%v", err, out)
	assert.Equal("HTTP/2 200 \r\n", out)

	// Now turn it off and verify
	globalconfig.DdevGlobalConfig.DisableHTTP2 = true
	err = app.Start()
	assert.NoError(err)

	out, err = exec.RunCommand("bash", []string{"-c", "curl -k -s -L -I " + app.GetPrimaryURL() + "| head -1"})
	assert.NoError(err, "failed to curl, err=%v out=%v", err, out)
	assert.Equal("HTTP/1.1 200 OK\r\n", out)

}

// Test the function FindEphemeralPort
func TestFindEphemeralPort(t *testing.T) {
	assert := asrt.New(t)

	localIP, _ := dockerutil.GetDockerIP()

	// Get a random port number in the dynamic port range
	startPort := 49152 + rand.Intn(65535-3-49152+1)
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
	if dockerutil.IsColima() || dockerutil.IsLima() {
		// Intermittent failures in CI due apparently to https://github.com/lima-vm/lima/issues/2536
		// Expected port is not available, so it allocates another one.
		t.Skip("Skipping on Lima/Colima as ports don't seem to be released properly in a timely fashion")
	}
	assert := asrt.New(t)

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
	targetHTTPListener, err := net.Listen("tcp", "127.0.0.1:"+targetHTTPPort)
	require.NoError(t, err)
	targetHTTPSListener, err := net.Listen("tcp", "127.0.0.1:"+targetHTTPSPort)
	require.NoError(t, err)

	t.Cleanup(func() {
		err = targetHTTPListener.Close()
		assert.NoError(err, "failed to close targetHTTPListener")
		err = targetHTTPSListener.Close()
		assert.NoError(err, "failed to close targetHTTPSListener")
		for _, a := range apps {
			err = a.Stop(true, false)
			assert.NoError(err)
			_ = os.RemoveAll(a.AppRoot)
		}

		// Stop the router, to prevent additional config from interfering with other tests.
		// We shouldn't have to do this when app.Stop() properly pushes new config to ddev-router
		_ = dockerutil.RemoveContainer(nodeps.RouterContainer)
		// TODO: Verify after stop that ddev-router has forgotten all about the extra ports
	})

	for i, app := range apps {
		// Predict which ephemeral ports the apps will use by using guess from starting point
		expectedEphemeralHTTPPort := ddevapp.MinEphemeralPort + i*2
		expectedEphemeralHTTPSPort := ddevapp.MinEphemeralPort + i*2 + 1

		err = app.Start()
		require.NoError(t, err)

		// app1 will not use the target ports, but the uses the discovered ephemeral ports.
		require.NotEqual(t, targetHTTPPort, app.GetRouterHTTPPort())
		require.NotEqual(t, targetHTTPSPort, app.GetRouterHTTPSPort())

		require.Equal(t, fmt.Sprint(expectedEphemeralHTTPPort), app.GetRouterHTTPPort())
		require.Equal(t, fmt.Sprint(expectedEphemeralHTTPSPort), app.GetRouterHTTPSPort())

		// Make sure that both http and https URLs have proper content
		_, err = testcommon.EnsureLocalHTTPContent(t, app.GetHTTPURL(), testString, 0)
		_, err = testcommon.EnsureLocalHTTPContent(t, app.GetHTTPSURL(), testString, 0)
	}

}
