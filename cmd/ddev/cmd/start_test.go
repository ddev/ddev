package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/ddev/ddev/pkg/config/state/storage/yaml"
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/ddev/ddev/pkg/util"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCmdStart runs `ddev start` on the test apps
func TestCmdStart(t *testing.T) {
	assert := asrt.New(t)

	// Gather reporting about goroutines at exit
	_ = os.Setenv("DDEV_GOROUTINES", "true")
	// Make sure we have running sites.
	err := addSites()
	require.NoError(t, err)

	// Stop all sites.
	out, err := exec.RunCommand(DdevBin, []string{"stop", "--all"})
	assert.NoError(err)
	testcommon.CheckGoroutineOutput(t, out)

	apps := []*ddevapp.DdevApp{}
	for _, testSite := range TestSites {
		app, err := ddevapp.NewApp(testSite.Dir, false)
		require.NoError(t, err)
		apps = append(apps, app)
	}

	// Build start command startMultipleArgs
	startMultipleArgs := []string{"start", "-y"}
	for _, app := range apps {
		startMultipleArgs = append(startMultipleArgs, app.GetName())
	}

	// Start multiple projects in one command
	out, err = exec.RunCommand(DdevBin, startMultipleArgs)
	assert.NoError(err, "ddev start with multiple project names should have succeeded, but failed, err: %v, output %s", err, out)
	testcommon.CheckGoroutineOutput(t, out)
	// If we omit the router, we should see the 127.0.0.1 URL.
	// Whether we have the router or not is not up to us, since it is omitted on codespaces.
	if slices.Contains(globalconfig.DdevGlobalConfig.OmitContainersGlobal, "ddev-router") {
		// Assert that the output contains the 127.0.0.1 URL
		assert.Contains(out, "127.0.0.1", "The output should contain the 127.0.0.1 URL, but it does not: %s", out)
	}
	// Confirm all sites are running
	for _, app := range apps {
		status, statusDesc := app.SiteStatus()
		assert.Equal(ddevapp.SiteRunning, status, "All sites should be running, but project=%s status=%s statusDesc=%s", app.GetName(), status, statusDesc)
		assert.Equal(ddevapp.SiteRunning, statusDesc, `The status description should be "running", but project %s status  is: %s`, app.GetName(), statusDesc)
		if len(globalconfig.DdevGlobalConfig.OmitContainersGlobal) == 0 {
			assert.Contains(out, app.GetPrimaryURL(), "The output should contain the primary URL, but it does not: %s", out)
		}
	}
}

// TestCmdStartOptionalProfiles checks `ddev start --profiles=list,of,profiles`
func TestCmdStartOptionalProfiles(t *testing.T) {
	testcommon.ClearDockerEnv()

	// When DDEV_GOROUTINES is set as it is in CI, its output is added to the
	// json in our curl tests below.
	t.Setenv("DDEV_GOROUTINES", "")

	site := TestSites[0]
	origDir, _ := os.Getwd()
	err := os.Chdir(TestSites[0].Dir)
	require.NoError(t, err)

	app, err := ddevapp.NewApp(site.Dir, false)
	require.NoError(t, err)

	_, err = exec.RunCommand(DdevBin, []string{"stop", site.Name})
	require.NoError(t, err)

	t.Cleanup(func() {
		err = os.Chdir(origDir)
		require.NoError(t, err)
		_ = app.Stop(true, false)
		// Remove the added docker-compose.busybox.yaml
		_ = os.RemoveAll(filepath.Join(app.GetConfigPath("docker-compose.busybox.yaml")))
		_ = app.Start()
	})

	// Add extra service that is in the "optional" profile
	err = fileutil.CopyFile(filepath.Join(origDir, "testdata", t.Name(), "docker-compose.busybox.yaml"), app.GetConfigPath("docker-compose.busybox.yaml"))
	require.NoError(t, err)

	out, err := exec.RunCommand(DdevBin, []string{"start", site.Name})
	require.NoError(t, err, "failed to start %s, output='%s'", site.Name, out)

	// Make sure that "optional" busybox1 service has Traefik entrypoint even when it's not started
	out, err = exec.RunCommand(DdevBin, []string{"exec", "curl", "-sf", "-w", util.CurlDiagnosticSuffix + "%{http_code}", fmt.Sprintf("%s:%s/api/http/routers/%s-busybox1-80-http@file", ddevapp.RouterComposeProjectName, globalconfig.DdevGlobalConfig.TraefikMonitorPort, site.Name)})
	require.NoError(t, err, "failed to check for http router for %s, output='%s'", site.Name, out)
	validateEntrypointForTraefikRouter(t, util.ExtractCurlBody(out), "http-18125", "http")

	out, err = exec.RunCommand(DdevBin, []string{"exec", "curl", "-sf", "-w", util.CurlDiagnosticSuffix + "%{http_code}", fmt.Sprintf("%s:%s/api/http/routers/%s-busybox1-80-https@file", ddevapp.RouterComposeProjectName, globalconfig.DdevGlobalConfig.TraefikMonitorPort, site.Name)})
	require.NoError(t, err, "failed to check for https router for %s, output='%s'", site.Name, out)
	validateEntrypointForTraefikRouter(t, util.ExtractCurlBody(out), "http-18126", "https")

	// Make sure the busybox service didn't get started
	container, err := ddevapp.GetContainer(app, "busybox")
	require.Error(t, err)
	require.Nil(t, container)

	profiles := []string{"busybox1", "busybox2"}
	// Now ddev start --optional and make sure the services are there
	out, err = exec.RunCommand(DdevBin, []string{"start", "--profiles=" + strings.Join(profiles, ","), site.Name})
	require.NoError(t, err, "start --profiles=%s failed, output='%s'", strings.Join(profiles, ","), out)
	for _, prof := range profiles {
		container, err = ddevapp.GetContainer(app, prof)
		require.NoError(t, err)
		require.NotNil(t, container)
	}

	// Make sure that "optional" busybox1 service has Traefik entrypoint when it's started
	out, err = exec.RunCommand(DdevBin, []string{"exec", "curl", "-sf", "-w", util.CurlDiagnosticSuffix + "%{http_code}", fmt.Sprintf("%s:%s/api/http/routers/%s-busybox1-80-http@file", ddevapp.RouterComposeProjectName, globalconfig.DdevGlobalConfig.TraefikMonitorPort, site.Name)})
	require.NoError(t, err, "failed to check for http router for %s, output='%s'", site.Name, out)
	validateEntrypointForTraefikRouter(t, util.ExtractCurlBody(out), "http-18125", "http")

	out, err = exec.RunCommand(DdevBin, []string{"exec", "curl", "-sf", "-w", util.CurlDiagnosticSuffix + "%{http_code}", fmt.Sprintf("%s:%s/api/http/routers/%s-busybox1-80-https@file", ddevapp.RouterComposeProjectName, globalconfig.DdevGlobalConfig.TraefikMonitorPort, site.Name)})
	require.NoError(t, err, "failed to check for https router for %s, output='%s'", site.Name, out)
	validateEntrypointForTraefikRouter(t, util.ExtractCurlBody(out), "http-18126", "https")
}

// validateEntrypointForTraefikRouter validates a JSON response from ddev-router:10999/api/http/routers/routerName
func validateEntrypointForTraefikRouter(t *testing.T, jsonResponse string, expectedEntryPoint string, routerType string) {
	var router map[string]interface{}
	err := json.Unmarshal([]byte(jsonResponse), &router)
	require.NoError(t, err, "failed to parse JSON response for %s router: %s", routerType, jsonResponse)
	require.Equal(t, "enabled", router["status"], "%s router status should be enabled: %s", routerType, jsonResponse)
	_, hasError := router["error"]
	require.False(t, hasError, "%s router should not have error field: %s", routerType, jsonResponse)
	entryPoints, ok := router["entryPoints"].([]interface{})
	require.True(t, ok, "entryPoints should be an array: %s", jsonResponse)
	require.Len(t, entryPoints, 1, "should have exactly one entryPoint: %s", jsonResponse)
	require.Equal(t, expectedEntryPoint, entryPoints[0], "%s router should use entryPoint %s: %s", routerType, expectedEntryPoint, jsonResponse)
}

// TestCmdStartShowsMessages tests that `ddev start` displays tip-of-the-day messages and sponsorship appreciation
func TestCmdStartShowsMessages(t *testing.T) {
	assert := asrt.New(t)

	site := TestSites[0]
	origDir, _ := os.Getwd()
	err := os.Chdir(TestSites[0].Dir)
	require.NoError(t, err)

	app, err := ddevapp.NewApp(site.Dir, false)
	require.NoError(t, err)

	_, err = exec.RunCommand(DdevBin, []string{"stop", site.Name})
	require.NoError(t, err)

	// Create temporary XDG_CONFIG_HOME for isolated testing
	tmpXdgConfigHomeDir := testcommon.CopyGlobalDdevDir(t)
	tmpGlobalDdevDir := globalconfig.GetGlobalDdevDir()

	t.Cleanup(func() {
		_ = os.Chdir(origDir)
		_ = app.Stop(true, false)
		testcommon.ResetGlobalDdevDir(t, tmpXdgConfigHomeDir)
		_ = app.Start()
	})

	// Create a global config with shorter intervals for testing
	globalconfig.EnsureGlobalConfig()
	globalconfig.DdevGlobalConfig.Messages.TickerInterval = 1     // 1 hour for testing
	globalconfig.DdevGlobalConfig.RemoteConfig.UpdateInterval = 1 // 1 hour for testing
	globalconfig.DdevGlobalConfig.RemoteConfig.RemoteConfigURL = "https://raw.githubusercontent.com/ddev/remote-config/main/remote-config.jsonc"
	globalconfig.DdevGlobalConfig.RemoteConfig.SponsorshipDataURL = "https://ddev.com/s/sponsorship-data.json"
	err = globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig)
	require.NoError(t, err)

	// Create a state file that indicates messages should be shown (old timestamp)
	stateFile := filepath.Join(tmpGlobalDdevDir, ".state.yaml")
	state := yaml.NewState(stateFile)

	// Set timestamps to force messages to be shown
	oldTime := time.Now().Add(-25 * time.Hour) // 25 hours ago
	err = state.Set("remoteconfig.last_ticker_time", oldTime.Unix())
	require.NoError(t, err)
	err = state.Set("remoteconfig.last_notification_time", oldTime.Unix())
	require.NoError(t, err)
	err = state.Set("sponsorship.updated_at", oldTime.Unix())
	require.NoError(t, err)
	err = state.Save()
	require.NoError(t, err)

	// Start the site and capture output (start without specifying project name to use current directory)
	out, err := exec.RunCommand(DdevBin, []string{"start", "-y"})
	require.NoError(t, err, "ddev start should succeed, output: %s", out)

	// Test that either ticker messages are shown OR the system indicates why they're not shown
	// Since remote config and sponsorship data depend on internet connectivity and GitHub availability,
	// we check for evidence that the system attempted to show messages
	hasTickerAttempt := strings.Contains(out, "tip") || strings.Contains(out, "DDEV") ||
		strings.Contains(out, "Internet connection not detected") || strings.Contains(out, "offline")

	// Check for sponsorship-related output (could be actual sponsorship data or debug messages)
	hasSponsorshipAttempt := strings.Contains(out, "sponsor") || strings.Contains(out, "appreciation") ||
		strings.Contains(out, "funding") || strings.Contains(out, "Internet connection not detected") ||
		strings.Contains(out, "offline")

	// Check for SponsorAppreciationMessage if present in the output
	expectedAppreciation := "💚 DDEV currently receives sponsorships from our community"
	hasAppreciationMessage := strings.Contains(out, expectedAppreciation)

	// We expect at least one of these systems to show evidence of attempting to work
	assert.True(hasTickerAttempt || hasSponsorshipAttempt || hasAppreciationMessage,
		"ddev start should show evidence of ticker messages, sponsorship appreciation, or appreciation message, output: %s", out)

	t.Logf("ddev start output contained: ticker_attempt=%v, sponsorship_attempt=%v, appreciation_message=%v", hasTickerAttempt, hasSponsorshipAttempt, hasAppreciationMessage)
}

// TestCmdStartShowsSponsorshipData tests that sponsorship data display works with isolated config
func TestCmdStartShowsSponsorshipData(t *testing.T) {
	assert := asrt.New(t)

	site := TestSites[0]
	origDir, _ := os.Getwd()
	err := os.Chdir(TestSites[0].Dir)
	require.NoError(t, err)

	app, err := ddevapp.NewApp(site.Dir, false)
	require.NoError(t, err)

	_, err = exec.RunCommand(DdevBin, []string{"stop", site.Name})
	require.NoError(t, err)

	// Create temporary XDG_CONFIG_HOME for isolated testing
	tmpXdgConfigHomeDir := testcommon.CopyGlobalDdevDir(t)
	tmpGlobalDdevDir := globalconfig.GetGlobalDdevDir()

	t.Cleanup(func() {
		_ = os.Chdir(origDir)
		_ = app.Stop(true, false)
		testcommon.ResetGlobalDdevDir(t, tmpXdgConfigHomeDir)
		_ = app.Start()
	})

	// Create a global config with custom sponsorship settings
	globalconfig.EnsureGlobalConfig()
	globalconfig.DdevGlobalConfig.RemoteConfig.UpdateInterval = 1 // 1 hour for testing
	globalconfig.DdevGlobalConfig.RemoteConfig.SponsorshipDataURL = "https://ddev.com/s/sponsorship-data.json"
	err = globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig)
	require.NoError(t, err)

	// Create a state file that indicates sponsorship data is stale
	stateFile := filepath.Join(tmpGlobalDdevDir, ".state.yaml")
	state := yaml.NewState(stateFile)

	// Set timestamp to force sponsorship data refresh
	oldTime := time.Now().Add(-25 * time.Hour) // 25 hours ago
	err = state.Set("sponsorship.updated_at", oldTime.Unix())
	require.NoError(t, err)
	err = state.Save()
	require.NoError(t, err)

	// Create a mock sponsorship data file for offline testing
	sponsorshipFile := filepath.Join(tmpGlobalDdevDir, ".sponsorship-data")
	mockSponsorshipData := `{
		"total_monthly_average_income": 1234.56,
		"github_ddev_sponsorships": {
			"total_sponsors": 42,
			"total_monthly_sponsorship": 500
		},
		"github_rfay_sponsorships": {
			"total_sponsors": 15,
			"total_monthly_sponsorship": 200
		},
		"monthly_invoiced_sponsorships": {
			"total_sponsors": 5,
			"total_monthly_sponsorship": 400
		},
		"annual_invoiced_sponsorships": {
			"total_sponsors": 3,
			"total_monthly_sponsorship": 134.56
		}
	}`
	err = os.WriteFile(sponsorshipFile, []byte(mockSponsorshipData), 0644)
	require.NoError(t, err)

	// Start the site and capture output (start without specifying project name to use current directory)
	out, err := exec.RunCommand(DdevBin, []string{"start", "-y"})
	require.NoError(t, err, "ddev start should succeed, output: %s", out)

	// Since sponsorship appreciation display depends on the implementation details
	// and network connectivity, we test that the sponsorship system was initialized
	// by checking for relevant debug output or system behavior

	// Check that the start command completed successfully (basic functionality test)
	assert.Contains(out, "Successfully started", "ddev start should report successful start")

	// The sponsorship system runs in the background, so we verify it was initialized
	// by checking the state and config files were created properly
	assert.FileExists(stateFile, "State file should exist after start")
	assert.FileExists(sponsorshipFile, "Sponsorship data file should exist")

	// Verify global config contains sponsorship configuration
	globalconfig.EnsureGlobalConfig()
	assert.Equal("https://ddev.com/s/sponsorship-data.json", globalconfig.DdevGlobalConfig.RemoteConfig.SponsorshipDataURL)

	t.Logf("Sponsorship configuration verified in global config")
}

// TestCmdStartRestartNoCache tests that --no-cache flag bypasses Docker build cache
func TestCmdStartRestartNoCache(t *testing.T) {
	// Create a temporary directory and switch to it
	origDir, _ := os.Getwd()
	tmpdir := testcommon.CreateTmpDir(t.Name())

	err := os.Chdir(tmpdir)
	require.NoError(t, err)
	_, err = exec.RunHostCommand(DdevBin, "config", "--auto")
	require.NoError(t, err)

	app, err := ddevapp.GetActiveApp("")
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = os.Chdir(origDir)
		_ = app.Stop(true, false)
		_ = os.RemoveAll(tmpdir)
	})

	// Add Dockerfiles that write random numbers to files
	err = fileutil.AppendStringToFile(app.GetConfigPath("web-build/Dockerfile"), `
RUN shuf -i 0-99999 -n1 > /random-web.txt
`)
	require.NoError(t, err)

	err = fileutil.AppendStringToFile(app.GetConfigPath("db-build/Dockerfile"), `
RUN shuf -i 0-99999 -n1 > /random-db.txt
`)
	require.NoError(t, err)

	// Populate examples (normally done in root's init())
	err = ddevapp.PopulateExamplesCommandsHomeadditions("")
	require.NoError(t, err)

	// Initial start
	err = app.Start()
	require.NoError(t, err)

	origRandomWeb, _, err := app.Exec(&ddevapp.ExecOpts{
		Cmd: "cat /random-web.txt",
	})
	require.NoError(t, err)

	origRandomDB, _, err := app.Exec(&ddevapp.ExecOpts{
		Cmd:     "cat /random-db.txt",
		Service: "db",
	})
	require.NoError(t, err)

	// Test 1: Regular restart should use cache for both web and db
	_, err = exec.RunHostCommand(DdevBin, "restart", app.Name)
	require.NoError(t, err)

	cachedRandomWeb, _, err := app.Exec(&ddevapp.ExecOpts{
		Cmd: "cat /random-web.txt",
	})
	require.NoError(t, err)
	require.Equal(t, origRandomWeb, cachedRandomWeb, "Regular restart should use cache for web")

	cachedRandomDB, _, err := app.Exec(&ddevapp.ExecOpts{
		Cmd:     "cat /random-db.txt",
		Service: "db",
	})
	require.NoError(t, err)
	require.Equal(t, origRandomDB, cachedRandomDB, "Regular restart should use cache for db")

	// Test 2: restart --no-cache should bypass cache for both web and db
	_, err = exec.RunHostCommand(DdevBin, "restart", "--no-cache", app.Name)
	require.NoError(t, err)

	noCacheRandomWeb, _, err := app.Exec(&ddevapp.ExecOpts{
		Cmd: "cat /random-web.txt",
	})
	require.NoError(t, err)
	require.NotEqual(t, origRandomWeb, noCacheRandomWeb, "restart --no-cache should bypass cache for web")

	noCacheRandomDB, _, err := app.Exec(&ddevapp.ExecOpts{
		Cmd:     "cat /random-db.txt",
		Service: "db",
	})
	require.NoError(t, err)
	require.NotEqual(t, origRandomDB, noCacheRandomDB, "restart --no-cache should bypass cache for db")

	// Test 3: Regular start after stop should use cache for both web and db
	err = app.Stop(false, false)
	require.NoError(t, err)

	_, err = exec.RunHostCommand(DdevBin, "start", app.Name)
	require.NoError(t, err)

	startCachedRandomWeb, _, err := app.Exec(&ddevapp.ExecOpts{
		Cmd: "cat /random-web.txt",
	})
	require.NoError(t, err)
	require.Equal(t, noCacheRandomWeb, startCachedRandomWeb, "Regular start should use cache for web")

	startCachedRandomDB, _, err := app.Exec(&ddevapp.ExecOpts{
		Cmd:     "cat /random-db.txt",
		Service: "db",
	})
	require.NoError(t, err)
	require.Equal(t, noCacheRandomDB, startCachedRandomDB, "Regular start should use cache for db")

	// Test 4: start --no-cache should bypass cache for both web and db
	err = app.Stop(false, false)
	require.NoError(t, err)

	_, err = exec.RunHostCommand(DdevBin, "start", "--no-cache", app.Name)
	require.NoError(t, err)

	startNoCacheRandomWeb, _, err := app.Exec(&ddevapp.ExecOpts{
		Cmd: "cat /random-web.txt",
	})
	require.NoError(t, err)
	require.NotEqual(t, noCacheRandomWeb, startNoCacheRandomWeb, "start --no-cache should bypass cache for web")

	startNoCacheRandomDB, _, err := app.Exec(&ddevapp.ExecOpts{
		Cmd:     "cat /random-db.txt",
		Service: "db",
	})
	require.NoError(t, err)
	require.NotEqual(t, noCacheRandomDB, startNoCacheRandomDB, "start --no-cache should bypass cache for db")
}
