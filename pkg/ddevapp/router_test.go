package ddevapp_test

import (
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/drud/ddev/pkg/netutil"
	"github.com/drud/ddev/pkg/nodeps"
	"github.com/stretchr/testify/require"
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

		app, err := ddevapp.NewApp(testDir, true, nodeps.ProviderDefault)
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

	site := TestSites[0]
	switchDir := site.Chdir()
	defer switchDir()

	// Force router stop so it will start up with Lets Encrypt mount
	dest := ddevapp.RouterComposeYAMLPath()
	_, _, err = dockerutil.ComposeCmd([]string{dest}, "-p", ddevapp.RouterProjectName, "down")
	assert.NoError(err)

	err = dockerutil.RemoveVolume("ddev-router-letsencrypt")
	assert.NoError(err)

	app, err := ddevapp.NewApp(site.Dir, false, "")
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

	stdout, _, err := dockerutil.Exec(container.ID, "df -T /etc/letsencrypt  | awk 'NR==2 {print $7;}'")
	assert.NoError(err)
	stdout = strings.Trim(stdout, "\n")

	assert.Equal("/etc/letsencrypt", stdout)

	_, _, err = dockerutil.Exec(container.ID, "test -f /etc/letsencrypt/options-ssl-nginx.conf")
	assert.NoError(err)
}
