package cmd

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/stretchr/testify/require"
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

	out, err = exec.RunHostCommand(DdevBin, "xhgui", "status")
	require.NoError(t, err)
	require.Contains(t, out, "XHProf is enabled and capturing")
	require.Contains(t, out, fmt.Sprintf("XHGui service is running and you can access it at %s", app.GetXHGuiURL()))

	// Test to see if xhgui UI is working
	// Hit the site
	_, _, err = testcommon.GetLocalHTTPResponse(t, app.GetPrimaryURL(),
		testcommon.WithTimeout(2*time.Second),
		testcommon.WithMaxAttempts(5),
		testcommon.WithBackoff(500*time.Millisecond),
	)
	require.NoError(t, err, "failed to get http response from %s", app.GetPrimaryURL())

	// Now hit xhgui UI
	desc, err := app.Describe(true)
	require.NoError(t, err)
	require.NotNil(t, desc["xhgui_url"])
	require.NotNil(t, desc["xhgui_https_url"])
	xhguiURL := desc["xhgui_https_url"].(string)

	// Output should contain at least one run
	testcommon.RequireLocalHTTPContent(t, xhguiURL, strings.ToLower(app.GetHostname()),
		testcommon.WithAlsoContains("Recent runs"),
		testcommon.WithMessagef("xhgui should list a profiling run for this project"),
		testcommon.WithTimeout(2*time.Second),
		testcommon.WithMaxAttempts(5),
		testcommon.WithBackoff(500*time.Millisecond),
	)

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
