package cmd

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/testcommon"

	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ddev/ddev/pkg/exec"
)

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
	_, err = exec.RunHostCommand(DdevBin, "start")
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

	out, err = exec.RunHostCommand(DdevBin, "xhgui", "status")
	require.NoError(t, err)
	require.Contains(t, out, "XHProf is enabled and capturing")
	require.Contains(t, out, fmt.Sprintf("XHGui service is running and you can access it at %s", app.GetXHGuiURL()))

	// Test to see if xhgui UI is working
	// Hit the site
	_, _, err = testcommon.GetLocalHTTPResponse(t, app.GetPrimaryURL(), 2)
	require.NoError(t, err, "failed to get http response from %s", app.GetPrimaryURL())
	// Give xhprof a moment to write the results; it may be asynchronous sometimes
	time.Sleep(2 * time.Second)

	// Now hit xhgui UI
	desc, err := app.Describe(true)
	require.NoError(t, err)
	require.NotNil(t, desc["xhgui_url"])
	require.NotNil(t, desc["xhgui_https_url"])
	xhguiURL := desc["xhgui_https_url"].(string)

	out, _, err = testcommon.GetLocalHTTPResponse(t, xhguiURL, 2)
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
	assert.NoError(t, err)
	assert.Contains(t, out, "Enabled xhgui")
	out, err = exec.RunHostCommand(DdevBin, "xhgui", "off")
	assert.NoError(t, err)
	assert.Contains(t, out, "Disabled xhgui")
}
