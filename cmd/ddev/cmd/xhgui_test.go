package cmd

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/stretchr/testify/require"
)

// logHTTPFailureDiagnostics logs detailed diagnostic information when an HTTP request fails
// This helps understand whether failures are due to routing, container readiness, or application issues
func logHTTPFailureDiagnostics(t *testing.T, url string, resp *http.Response, body string, err error, app *ddevapp.DdevApp, serviceName string) {
	t.Logf("Failed to access %s at %s: %v", serviceName, url, err)

	if resp != nil {
		t.Logf("Response status: %d", resp.StatusCode)
		t.Logf("Response headers:")
		for k, v := range resp.Header {
			t.Logf("  %s: %v", k, v)
		}
		// Key headers to check:
		// - "Server" will show "nginx" if from service container, or identify the router
		// - "X-Powered-By" will show "PHP/x.x.x" if PHP-FPM processed the request
	}

	if len(body) > 0 {
		maxLen := 500
		if len(body) < maxLen {
			maxLen = len(body)
		}
		t.Logf("Response body (first %d chars): %s", maxLen, body[:maxLen])
	}

	// Try accessing the service container internally to see if the issue is routing or the container itself
	internalOut, _, internalErr := app.Exec(&ddevapp.ExecOpts{
		Service: serviceName,
		Cmd:     "curl -I 127.0.0.1 2>&1",
	})
	t.Logf("Internal container curl to 127.0.0.1 in %s (err=%v):\n%s", serviceName, internalErr, internalOut)

	// Check container health
	containerName := ddevapp.GetContainerName(app, serviceName)
	container, findErr := dockerutil.FindContainerByName(containerName)
	if findErr == nil && container != nil {
		health, healthLog := dockerutil.GetContainerHealth(container)
		t.Logf("%s container health: %s", serviceName, health)
		if healthLog != "" {
			t.Logf("Health log: %s", healthLog)
		}
	} else {
		t.Logf("Could not find %s container: %v", serviceName, findErr)
	}
}

// TestCmdXHGui tests the `ddev xhgui` command
func TestCmdXHGui(t *testing.T) {
	globalconfig.DdevVerbose = true

	origDir, _ := os.Getwd()
	v := TestSites[0]

	err := os.Chdir(v.Dir)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = exec.RunHostCommand(DdevBin, "xhprof", "off")
		_, err = exec.RunHostCommand(DdevBin, "config", "global", "xhprof-mode-reset")
		_, _ = exec.RunHostCommand(DdevBin, "restart")
		require.NoError(t, err)

		_ = os.Chdir(origDir)
	})

	_, err = exec.RunHostCommand(DdevBin, "config", "global", "--xhprof-mode=xhgui")
	require.NoError(t, err)
	_, err = exec.RunHostCommand(DdevBin, "restart")
	require.NoError(t, err)

	out, err := exec.RunHostCommand(DdevBin, "xhgui", "status")
	require.NoError(t, err)
	require.Contains(t, out, "XHProf is disabled.")
	require.Contains(t, out, "XHGui is disabled.")

	out, err = exec.RunHostCommand(DdevBin, "xhgui", "on")
	require.NoError(t, err)
	require.Contains(t, out, "Started optional compose profiles 'xhgui'")

	app, err := ddevapp.GetActiveApp("")
	require.NoError(t, err)

	//TODO: php8.5: Remove exclusion when xdebug lands in PHP8.5
	if app.PHPVersion == nodeps.PHP85 {
		t.Skip("Skipping tests for PHP8.5 until xdebug lands in PHP8.5")
	}

	out, err = exec.RunHostCommand(DdevBin, "xhgui", "status")
	require.NoError(t, err)
	require.Contains(t, out, "XHProf is enabled and capturing")
	require.Contains(t, out, fmt.Sprintf("XHGui service is running and you can access it at %s", app.GetXHGuiURL()))

	// Use more retries as it may fail on macOS at first
	opts := testcommon.HTTPRequestOpts{
		TimeoutSeconds: 2,
	}

	// Test to see if xhgui UI is working
	// Hit the site
	_, _, err = testcommon.GetLocalHTTPResponseWithBackoff(t, app.GetPrimaryURL(), 5, 500*time.Millisecond, opts)
	require.NoError(t, err, "failed to get http response from %s", app.GetPrimaryURL())
	// Give xhprof a moment to write the results; it may be asynchronous sometimes
	time.Sleep(2 * time.Second)

	// Now hit xhgui UI
	desc, err := app.Describe(true)
	require.NoError(t, err)
	require.NotNil(t, desc["xhgui_url"])
	require.NotNil(t, desc["xhgui_https_url"])
	xhguiURL := desc["xhgui_https_url"].(string)

	// Increase retries to handle transient 403/timing issues
	// Using longer backoff: 1s, 2s, 4s, 8s, 16s, 32s, 64s = ~127s total
	out, resp, err := testcommon.GetLocalHTTPResponseWithBackoff(t, xhguiURL, 10, 1*time.Second, opts)
	if err != nil {
		logHTTPFailureDiagnostics(t, xhguiURL, resp, out, err, app, "xhgui")
	}
	require.NoError(t, err)
	// Output should contain at least one run
	require.Contains(t, out, strings.ToLower(app.GetHostname()))
	require.Contains(t, out, "Recent runs")

	_, err = exec.RunHostCommand(DdevBin, "xhgui", "off")
	require.NoError(t, err)
	out, err = exec.RunHostCommand(DdevBin, "xhgui", "status")
	require.NoError(t, err)
	require.Contains(t, out, "XHProf is disabled")
	out, err = exec.RunHostCommand(DdevBin, "xhgui", "on")
	require.NoError(t, err)
	require.Contains(t, out, "Enabled xhgui")
	out, err = exec.RunHostCommand(DdevBin, "xhgui", "off")
	require.NoError(t, err)
	require.Contains(t, out, "Disabled xhgui")
}
