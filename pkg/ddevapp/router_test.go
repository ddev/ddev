package ddevapp_test

import (
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/drud/ddev/pkg/netutil"
	"github.com/drud/ddev/pkg/nodeps"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/testcommon"
	asrt "github.com/stretchr/testify/assert"
)

// TestPortOverride makes sure that the router_http_port and router_https_port
// config.yaml overrides work correctly.
// It starts up 3 ddev projects, looks to see if the config is set right,
// then tests to see that the right ports have been started up on the router.
func TestPortOverride(t *testing.T) {
	assert := asrt.New(t)

	// Try some different combinations of ports. The first (offset 0) will
	// share ports with already-started test sites.
	for i := 0; i < 3; i++ {
		testDir := testcommon.CreateTmpDir("TestPortOverride")

		// testcommon.Chdir()() and CleanupDir() checks their own errors (and exit)
		defer testcommon.CleanupDir(testDir)
		defer testcommon.Chdir(testDir)()

		testcommon.ClearDockerEnv()

		app, err := ddevapp.NewApp(testDir, true)
		assert.NoError(err)
		app.RouterHTTPPort = strconv.Itoa(80 + i)
		// Note that we start with port 453 instead of 443 here because Windows
		// by default has port 445 occupied by NetBT (Netbios over TCP)
		// So the test will fail because of that.
		app.RouterHTTPSPort = strconv.Itoa(453 + i)
		app.Name = "TestPortOverride-" + strconv.Itoa(i)
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

		startErr := app.StartAndWait(5)
		// defer the app.Stop so we have a more diverse set of tests. If we brought
		// each down before testing the next that would be a more trivial test.
		// Don't worry about the possible error case as this is just a test cleanup
		// nolint: errcheck
		defer app.Stop(true, false)

		if startErr != nil {
			t.Logf("Failed app.StartAndWait: %v", startErr)
			out, err := exec.RunCommand(DdevBin, []string{"list"})
			assert.NoError(err)
			t.Logf("\n=========== output of ddev list ==========\n%s\n============\n", out)
			out, err = exec.RunCommand("docker", []string{"logs", "ddev-router"})
			assert.NoError(err)
			t.Logf("\n=========== output of docker logs ddev-router ==========\n%s\n============\n", out)

			logsFromApp, err := ddevapp.GetErrLogsFromApp(app, startErr)
			assert.NoError(err)
			t.Logf("\n================== logsFromApp ====================\n%s\n", logsFromApp)

			t.Fatalf("failed to app.StartAndWait(), err=%v", startErr)
		}
		err = app.Wait([]string{"web"})
		assert.NoError(err)
		assert.True(netutil.IsPortActive(app.RouterHTTPPort))
		assert.True(netutil.IsPortActive(app.RouterHTTPSPort))
	}

}

// Do a modest test of Lets Encrypt functionality
// This just checks to see that certbot ran and populated /etc/letsencrypt and
// that /etc/letsencrypt is mounted on volume.
func TestLetsEncrypt(t *testing.T) {
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

	stdout, _, err := dockerutil.Exec(container.ID, "df -T /etc/letsencrypt  | awk 'NR==2 {print $7;}'")
	assert.NoError(err)
	stdout = strings.Trim(stdout, "\n")

	assert.Equal("/etc/letsencrypt", stdout)

	_, _, err = dockerutil.Exec(container.ID, "test -f /etc/letsencrypt/options-ssl-nginx.conf")
	assert.NoError(err)
}

// TestRouterConfigOverride tests that the ~/.ddev/.router-compose.yaml can be overridden
// with ~/.ddev/router-compose.*.yaml
func TestRouterConfigOverride(t *testing.T) {
	assert := asrt.New(t)
	pwd, _ := os.Getwd()
	testDir := testcommon.CreateTmpDir(t.Name())
	_ = os.Chdir(testDir)
	overrideYaml := filepath.Join(globalconfig.GetGlobalDdevDir(), "router-compose.override.yaml")

	testcommon.ClearDockerEnv()

	app, err := ddevapp.NewApp(testDir, true)
	assert.NoError(err)
	err = app.WriteConfig()
	assert.NoError(err)
	err = fileutil.CopyFile(filepath.Join(pwd, "testdata", t.Name(), "router-compose.override.yaml"), overrideYaml)
	assert.NoError(err)

	answer := fileutil.RandomFilenameBase()
	os.Setenv("ANSWER", answer)
	assert.NoError(err)
	t.Cleanup(func() {
		err = app.Stop(true, false)
		assert.NoError(err)
		err = os.Chdir(pwd)
		assert.NoError(err)
		_ = os.RemoveAll(testDir)
		err = os.Remove(overrideYaml)
		assert.NoError(err)
	})

	err = app.Start()
	assert.NoError(err)

	stdout, _, err := dockerutil.Exec("ddev-router", "bash -c 'echo $ANSWER'")
	assert.Equal(answer+"\n", stdout)
}

// TestDisableHTTP2 tests we can enable or disable http2
func TestDisableHTTP2(t *testing.T) {
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
	out, err := exec.RunCommand("bash", []string{"-c", "curl -k -s -I " + app.GetPrimaryURL() + "| head -1"})
	assert.NoError(err, "failed to curl, err=%v out=%v", err, out)
	assert.Equal("HTTP/2 200 \r\n", out)

	// Now turn it off and verify
	globalconfig.DdevGlobalConfig.DisableHTTP2 = true
	err = app.Start()
	assert.NoError(err)

	out, err = exec.RunCommand("bash", []string{"-c", "curl -k -s -I " + app.GetPrimaryURL() + "| head -1"})
	assert.Equal("HTTP/1.1 200 OK\r\n", out)

}
