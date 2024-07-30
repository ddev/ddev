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
		Action:       []string{"-p", ddevapp.RouterProjectName, "down"},
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
			Action:       []string{"-p", ddevapp.RouterProjectName, "down"},
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

	// Get a random port number in the dynamic port range
	startPort := rand.Intn(65535 - 49152 + 1)
	goodEndPort := startPort + 3
	badEndPort := startPort + 2

	// Listen in the first 3 ports
	l0, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(startPort))
	require.NoError(t, err)
	defer l0.Close()
	l1, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(startPort+1))
	require.NoError(t, err)
	defer l1.Close()
	l2, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(startPort+2))
	require.NoError(t, err)
	defer l2.Close()

	_, ok := ddevapp.FindAvailableRouterPort(startPort, badEndPort)
	assert.Exactly(false, ok)

	port, ok := ddevapp.FindAvailableRouterPort(startPort, goodEndPort)
	assert.Exactly(true, ok)
	assert.Exactly(startPort+3, port)
}

// Test that the app assigns an ephemeral port if the default one is not available.
func TestUseEphemeralPort(t *testing.T) {
	assert := asrt.New(t)

	targetHTTPPort, targetHTTPSPort := "28080", "28443"

	// This is copied from ddevapp_test.go
	site1 := testcommon.TestSite{
		Name:                          "TestEphemeralPort1", // Drupal D7
		SourceURL:                     "https://ftp.drupal.org/files/projects/drupal-7.90.tar.gz",
		ArchiveInternalExtractionPath: "drupal-7.90/",
		FilesTarballURL:               "https://github.com/ddev/ddev_test_tarballs/releases/download/v1.1/d7test-7.59.files.tar.gz",
		DBTarURL:                      "https://github.com/ddev/ddev_test_tarballs/releases/download/v1.1/d7test-7.87-db.tar.gz",
		FullSiteTarballURL:            "",
		Dir:                           testcommon.CreateTmpDir(t.Name() + "_1"),
		Docroot:                       "",
		Type:                          nodeps.AppTypeDrupal7,
		Safe200URIWithExpectation:     testcommon.URIWithExpect{URI: "/README.txt", Expect: "Drupal is an open source content management platform"},
		DynamicURI:                    testcommon.URIWithExpect{URI: "/node/1", Expect: "D7 test project, kittens edition"},
		FilesImageURI:                 "/sites/default/files/field/image/kittens-large.jpg",
		FullSiteArchiveExtPath:        "docroot/sites/default/files",
	}

	site2 := testcommon.TestSite{
		Name:                          "TestEphemeralPort2",
		SourceURL:                     "https://ftp.drupal.org/files/projects/drupal-8.9.20.tar.gz",
		ArchiveInternalExtractionPath: "drupal-8.9.20/",
		FilesTarballURL:               "https://github.com/ddev/ddev_test_tarballs/releases/download/v1.1/d8_umami.files.tar.gz",
		FilesZipballURL:               "https://github.com/ddev/ddev_test_tarballs/releases/download/v1.1/d8_umami.files.zip",
		DBTarURL:                      "https://github.com/ddev/ddev_test_tarballs/releases/download/v1.1/d8_umami.sql.tar.gz",
		DBZipURL:                      "https://github.com/ddev/ddev_test_tarballs/releases/download/v1.1/d8_umami.sql.zip",
		FullSiteTarballURL:            "",
		Dir:                           testcommon.CreateTmpDir(t.Name() + "_2"),
		Type:                          nodeps.AppTypeDrupal,
		Docroot:                       "",
		Safe200URIWithExpectation:     testcommon.URIWithExpect{URI: "/README.txt", Expect: "Drupal is an open source content management platform"},
		DynamicURI:                    testcommon.URIWithExpect{URI: "/node/2", Expect: "Vegan chocolate and nut brownies"},
		FilesImageURI:                 "/sites/default/files/vegan-chocolate-nut-brownies.jpg",
	}

	app, err := ddevapp.NewApp(site1.Dir, false)
	require.NoError(t, err)
	app2, err := ddevapp.NewApp(site2.Dir, false)
	require.NoError(t, err)

	// Configure both apps to use the same target ports. Keep original configured ports for undoing the configuration later.
	appHTTPPort, appHTTPSPort := app.RouterHTTPPort, app.RouterHTTPSPort
	app.RouterHTTPPort, app.RouterHTTPSPort = targetHTTPPort, targetHTTPSPort
	app2HTTPPort, app2HTTPSPort := app.RouterHTTPPort, app.RouterHTTPSPort
	app2.RouterHTTPPort, app2.RouterHTTPSPort = targetHTTPPort, targetHTTPSPort

	t.Cleanup(func() {
		err = app.Stop(true, false)
		assert.NoError(err)
		err = app2.Stop(true, false)
		assert.NoError(err)

		// Undo the configuration of the ports.
		app.RouterHTTPPort, app.RouterHTTPSPort = appHTTPPort, appHTTPSPort
		app2.RouterHTTPPort, app2.RouterHTTPSPort = app2HTTPPort, app2HTTPSPort

		err = app.WriteConfig()
		assert.NoError(err)
		err = app2.WriteConfig()
		assert.NoError(err)
		app.RemoveGlobalProjectInfo()
		app2.RemoveGlobalProjectInfo()

		// Finally reset the router configuration, so it does not interfere other tests.
		err = ddevapp.StartDdevRouter()
		require.NoError(t, err)

		router, err := ddevapp.FindDdevRouter()
		if router != nil && err == nil && router.State == "running" {
			err = dockerutil.RemoveContainer(nodeps.RouterContainer)
			assert.NoError(err)
		}
	})

	// Occupy target router ports:
	l0, err := net.Listen("tcp", "127.0.0.1:"+targetHTTPPort)
	require.NoError(t, err)
	defer l0.Close()
	l1, err := net.Listen("tcp", "127.0.0.1:"+targetHTTPSPort)
	require.NoError(t, err)
	defer l1.Close()

	// Find out which ephemeral ports the apps will use.
	ephemeralHTTPPort, ok := ddevapp.FindAvailableRouterPort(ddevapp.MinEphemeralHTTPPort, ddevapp.MaxEphemeralHTTPPort)
	assert.Exactly(true, ok)
	ephemeralHTTPSPort, ok := ddevapp.FindAvailableRouterPort(ddevapp.MinEphemeralHTTPSPort, ddevapp.MaxEphemeralHTTPSPort)
	assert.Exactly(true, ok)

	err = app.Start()
	require.NoError(t, err)

	// First app does not use the target ports, but the ephemeral ports.
	require.NotEqual(t, targetHTTPPort, app.GetRouterHTTPPort())
	require.NotEqual(t, targetHTTPSPort, app.GetRouterHTTPSPort())
	require.Equal(t, fmt.Sprint(ephemeralHTTPPort), app.GetRouterHTTPPort())
	require.Equal(t, fmt.Sprint(ephemeralHTTPSPort), app.GetRouterHTTPSPort())

	// Second app does not use either the target ports, but the ephemeral ports being used by first app.
	err = app2.Start()
	require.NoError(t, err)
	require.Equal(t, fmt.Sprint(ephemeralHTTPPort), app2.GetRouterHTTPPort())
	require.Equal(t, fmt.Sprint(ephemeralHTTPSPort), app2.GetRouterHTTPSPort())
}
