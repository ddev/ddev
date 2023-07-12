package ddevapp_test

import (
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
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGlobalPortOverride tests global router_http_port and router_https_port
func TestGlobalPortOverride(t *testing.T) {
	assert := asrt.New(t)

	origGlobalHTTPPort := globalconfig.DdevGlobalConfig.RouterHTTPPort
	origGlobalHTTPSPort := globalconfig.DdevGlobalConfig.RouterHTTPSPort

	globalconfig.DdevGlobalConfig.RouterHTTPPort = "8553"
	globalconfig.DdevGlobalConfig.RouterHTTPSPort = "8554"

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
	err = app.Restart()
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
// It starts up 3 ddev projects, looks to see if the config is set right,
// then tests to see that the right ports have been started up on the router.
func TestProjectPortOverride(t *testing.T) {
	assert := asrt.New(t)

	origDir, _ := os.Getwd()

	// Try some different combinations of ports. The first (offset 0) will
	// share ports with already-started test sites.
	for i := 0; i < 3; i++ {
		testDir := testcommon.CreateTmpDir("TestProjectPortOverride")

		t.Cleanup(func() {
			err := os.Chdir(origDir)
			assert.NoError(err)
			err = os.RemoveAll(testDir)
			assert.NoError(err)
		})

		testcommon.ClearDockerEnv()

		app, err := ddevapp.NewApp(testDir, true)
		assert.NoError(err)
		app.RouterHTTPPort = strconv.Itoa(80 + i)
		// Note that we start with port 453 instead of 443 here because Windows
		// by default has port 445 occupied by NetBT (Netbios over TCP)
		// So the test will fail because of that.
		app.RouterHTTPSPort = strconv.Itoa(453 + i)
		app.Name = "TestProjectPortOverride-" + strconv.Itoa(i)
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

		// These ports will already be active if on the standard port, because
		// the TestMain has started stuff up on the standard ports.
		if i != 0 {
			assert.False(netutil.IsPortActive(app.RouterHTTPPort))
			assert.False(netutil.IsPortActive(app.RouterHTTPSPort))
		}

		err = app.Start()
		require.NoError(t, err)
		// defer the app.Stop() so we have a more diverse set of tests. If we brought
		// each down before testing the next that would be a more trivial test.
		// Don't worry about the possible error case as this is just a test cleanup
		t.Cleanup(func() {
			err = app.Stop(true, false)
			assert.NoError(err)
		})

		assert.True(netutil.IsPortActive(app.RouterHTTPPort), "port "+app.RouterHTTPPort+" should be active")
		assert.True(netutil.IsPortActive(app.RouterHTTPSPort), "port "+app.RouterHTTPSPort+" should be active")
	}
}

// Do a modest test of Lets Encrypt functionality
// This just checks to see that certbot ran and populated /etc/letsencrypt and
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
	_, _, err = dockerutil.ComposeCmd([]string{dest}, "-p", ddevapp.RouterProjectName, "down")
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
		_, _, err = dockerutil.ComposeCmd([]string{dest}, "-p", ddevapp.RouterProjectName, "down")
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
