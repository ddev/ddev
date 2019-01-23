package ddevapp_test

import (
	"github.com/drud/ddev/pkg/netutil"
	"strconv"
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

		app, err := ddevapp.NewApp(testDir, ddevapp.ProviderDefault)
		assert.NoError(err)
		app.RouterHTTPPort = strconv.Itoa(80 + i)
		// Note that we start with port 453 instead of 443 here because Windows
		// by default has port 445 occupied by NetBT (Netbios over TCP)
		// So the test will fail because of that.
		app.RouterHTTPSPort = strconv.Itoa(453 + i)
		app.Name = "TestPortOverride-" + strconv.Itoa(i)
		app.Type = ddevapp.AppTypePHP
		err = app.WriteConfig()
		assert.NoError(err)
		err = app.ReadConfig()
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

		startErr := app.StartAndWaitForSync(5)
		// defer the app.Down so we have a more diverse set of tests. If we brought
		// each down before testing the next that would be a more trivial test.
		// Don't worry about the possible error case as this is just a test cleanup
		// nolint: errcheck
		defer app.Down(true, false)

		var logs string
		if startErr != nil {
			logs, err = ddevapp.GetErrLogsFromApp(app, err)
			assert.NoError(err)
			t.Fatalf("failed to app.StartAndWaitForSync(), err=%v logs=\n=========\n%s\n===========\n", startErr, logs)
		}
		err = app.Wait([]string{"web"})
		assert.NoError(err)
		assert.True(netutil.IsPortActive(app.RouterHTTPPort))
		assert.True(netutil.IsPortActive(app.RouterHTTPSPort))
	}

}
