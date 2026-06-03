package testcommon

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewRequestConfig verifies option resolution and defaults. No Docker needed.
func TestNewRequestConfig(t *testing.T) {
	// Defaults.
	c := newRequestConfig()
	require.Equal(t, defaultHTTPTimeout, c.timeout)
	require.Equal(t, defaultHTTPTick, c.tick)
	require.Equal(t, defaultHTTPAttempts, c.maxAttempts)
	require.Equal(t, defaultHTTPStatus, c.expectStatus)
	require.False(t, c.backoff)
	require.Empty(t, c.alsoContains)

	// WithMaxAttempts sets total attempts; values below 1 clamp to 1.
	require.Equal(t, 5, newRequestConfig(WithMaxAttempts(5)).maxAttempts)
	require.Equal(t, 1, newRequestConfig(WithMaxAttempts(0)).maxAttempts)
	require.Equal(t, 1, newRequestConfig(WithMaxAttempts(-3)).maxAttempts)

	// WithTimeout clamps negatives to 0 (disabled).
	require.Equal(t, 5*time.Second, newRequestConfig(WithTimeout(5*time.Second)).timeout)
	require.Equal(t, time.Duration(0), newRequestConfig(WithTimeout(-1)).timeout)

	// WithExpectStatus overrides the success status.
	require.Equal(t, http.StatusFound, newRequestConfig(WithExpectStatus(http.StatusFound)).expectStatus)

	// WithBackoff enables backoff and sets the starting tick (positive only).
	bc := newRequestConfig(WithBackoff(500 * time.Millisecond))
	require.True(t, bc.backoff)
	require.Equal(t, 500*time.Millisecond, bc.tick)
	require.Equal(t, defaultHTTPTick, newRequestConfig(WithBackoff(0)).tick)

	// WithAlsoContains accumulates across calls.
	require.Equal(t, []string{"a", "b", "c"},
		newRequestConfig(WithAlsoContains("a", "b"), WithAlsoContains("c")).alsoContains)

	// WithMessagef stores the format and args.
	mc := newRequestConfig(WithMessagef("x %d", 7))
	require.Equal(t, "x %d", mc.msgFormat)
	require.Equal(t, []any{7}, mc.msgArgs)
}

// TestLocalDockerAddress checks that localDockerAddress points the URL at the
// Docker IP while keeping the original port and path, and returns the original
// hostname (used for the Host header and TLS server name).
func TestLocalDockerAddress(t *testing.T) {
	// A URL with a port keeps it.
	addr, host, err := localDockerAddress("https://example.ddev.site:8142/run")
	require.NoError(t, err)
	require.Equal(t, "example.ddev.site", host)
	require.True(t, strings.HasPrefix(addr, "https://"), "scheme must be preserved, got %s", addr)
	require.Contains(t, addr, ":8142", "port must be preserved, got %s", addr)
	require.True(t, strings.HasSuffix(addr, "/run"), "path must be preserved, got %s", addr)
	require.NotContains(t, addr, "example.ddev.site", "host should be replaced by the Docker IP, got %s", addr)

	// A URL without a port must not gain one.
	addr, host, err = localDockerAddress("http://example.ddev.site/")
	require.NoError(t, err)
	require.Equal(t, "example.ddev.site", host)
	require.True(t, strings.HasPrefix(addr, "http://"), "scheme must be preserved, got %s", addr)
	require.True(t, strings.HasSuffix(addr, "/"), "path must be preserved, got %s", addr)
	require.NotContains(t, addr, "example.ddev.site", "host should be replaced by the Docker IP, got %s", addr)
	u, err := url.Parse(addr)
	require.NoError(t, err)
	require.Empty(t, u.Port(), "no port should be added, got %s", addr)
}

// TestDescribeHTTPFailure verifies the failure message layout. No Docker needed.
func TestDescribeHTTPFailure(t *testing.T) {
	resp := &LocalHTTPResponse{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"text/html"}}}
	msg := describeHTTPFailure("https://x.ddev.site:8142", resp, "the body", nil, []string{"Recent runs"}, "want a run")
	require.Contains(t, msg, "Fail: want a run")
	require.Contains(t, msg, "GET: https://x.ddev.site:8142")
	require.Contains(t, msg, "Status: 200")
	require.Contains(t, msg, `Error: body does not contain ["Recent runs"]`)
	require.Contains(t, msg, "Content-Type", "response headers should be in the message")
	require.Contains(t, msg, `Body: "the body"`)

	// A transport error and an empty caller message fall back sensibly.
	msg = describeHTTPFailure("https://x.ddev.site", nil, "", fmt.Errorf("dial tcp: boom"), nil, "")
	require.Contains(t, msg, "Fail: HTTP request failed")
	require.Contains(t, msg, "Status: 0")
	require.Contains(t, msg, "Error: dial tcp: boom")
}

// TestGetLocalHTTPResponse brings up a project and exercises GetLocalHTTPResponse,
// AssertLocalHTTPContent, and RequireLocalHTTPContent against it. Needs Docker.
func TestGetLocalHTTPResponse(t *testing.T) {
	if nodeps.IsEnvFalse("DDEV_RUN_TEST_ANYWAY") && (nodeps.IsWindows() || dockerutil.IsColima() || dockerutil.IsLima() || dockerutil.IsRancherDesktop() || nodeps.IsWSL2MirroredMode()) {
		t.Skip("Skipping on Windows/Colima/Lima/Rancher/WSL2-mirrored as it always seems to fail")
	}
	ensureDdevBin()

	// We have to get globalconfig read so CA is known and installed.
	err := globalconfig.ReadGlobalConfig()
	require.NoError(t, err)

	assert := asrt.New(t)

	origDir, _ := os.Getwd()

	dockerutil.EnsureDdevNetwork()

	// It's not ideal to copy/paste this archive around, but we don't actually care about the contents
	// of the archive for this test, only that it exists and can be extracted. This should (knock on wood)
	//not need to be updated over time.
	site := TestSites[0]
	site.Name = t.Name()

	_, _ = exec.RunCommand(DdevBin, []string{"stop", "-RO", site.Name})

	err = site.Prepare()
	require.NoError(t, err, "Prepare() failed on TestSite.Prepare() site=%s, err=%v", site.Name, err)

	app := &ddevapp.DdevApp{}
	err = app.Init(site.Dir)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = os.Chdir(origDir)
		_ = app.Stop(true, false)

		app.RouterHTTPSPort = ""
		app.RouterHTTPPort = ""
		_ = app.WriteConfig()

		site.Cleanup()
	})

	for _, pair := range []PortPair{{"8000", "8043"}, {"8080", "8443"}} {
		ClearDockerEnv()
		app.RouterHTTPPort = pair.HTTPPort
		app.RouterHTTPSPort = pair.HTTPSPort
		err = app.WriteConfig()
		assert.NoError(err)

		startErr := app.StartAndWait(5)
		assert.NoError(startErr, "app.StartAndWait failed for port pair %v", pair)
		if startErr != nil {
			logs, health, _ := ddevapp.GetErrLogsFromApp(app, startErr)
			t.Fatalf("healthcheck:\n%s\n\nlogs from broken container:\n=======\n%s\n========\n", health, logs)
		}

		// Exercise all three helpers (getter + non-fatal + fatal) against the live
		// site. The plain-HTTP URL always works; the HTTPS one needs mkcert, so
		// check it only when that's set up.
		checkURL := func(reqURL string) {
			out, _, getErr := GetLocalHTTPResponse(t, reqURL)
			assert.NoError(getErr)
			assert.Contains(out, site.Safe200URIWithExpectation.Expect)
			AssertLocalHTTPContent(t, reqURL, site.Safe200URIWithExpectation.Expect,
				WithMessagef("safe URL should serve the expected static content"))
			RequireLocalHTTPContent(t, reqURL, site.Safe200URIWithExpectation.Expect,
				WithMessagef("safe URL should serve the expected static content"))
		}

		checkURL(app.GetHTTPURL() + site.Safe200URIWithExpectation.URI)
		if globalconfig.GetCAROOT() != "" {
			checkURL(app.GetHTTPSURL() + site.Safe200URIWithExpectation.URI)
		}
	}
}
