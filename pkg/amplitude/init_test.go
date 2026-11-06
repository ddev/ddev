package amplitude

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/versionconstants"
	"github.com/ddev/ddev/third_party/ampli"
	"github.com/stretchr/testify/require"
)

// TestInitAmplitudeAfterGlobalConfig checks that an opted-in user's events
// are delivered even when InitAmplitude() runs before the global config is
// loaded, which disabled all instrumentation in v1.25.5.
func TestInitAmplitudeAfterGlobalConfig(t *testing.T) {
	var mu sync.Mutex
	var bodies []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		bodies = append(bodies, string(body))
		mu.Unlock()
		_, _ = w.Write([]byte(`{"code":200}`))
	}))
	defer server.Close()

	// A local server and a fake key, so nothing reaches Amplitude even where
	// CI or DDEV_NO_INSTRUMENTATION would otherwise disable instrumentation.
	t.Setenv("DDEV_XDG_CONFIG_HOME", t.TempDir())
	origKey, origNoInstrumentation := versionconstants.AmplitudeAPIKey, globalconfig.DdevNoInstrumentation
	versionconstants.AmplitudeAPIKey = "test-key"
	globalconfig.DdevNoInstrumentation = false
	t.Cleanup(func() {
		versionconstants.AmplitudeAPIKey, globalconfig.DdevNoInstrumentation = origKey, origNoInstrumentation
		resetForTest("")
	})
	resetForTest(server.URL)

	writeGlobalConfig := func(content string) {
		require.NoError(t, os.WriteFile(filepath.Join(globalconfig.GetGlobalDdevDir(), "global_config.yaml"), []byte(content), 0644))
	}
	writeGlobalConfig("instrumentation_opt_in: true\n")

	require.False(t, globalconfig.IsGlobalConfigLoaded(), "another test in this package loaded the global config first")
	InitAmplitude()
	require.False(t, ampli.Instance.InitializedAndEnabled())

	globalconfig.EnsureGlobalConfig()
	InitAmplitude()
	require.True(t, ampli.Instance.InitializedAndEnabled())

	ampli.Instance.Flush()
	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(bodies) > 0 && strings.Contains(strings.Join(bodies, ""), "$identify")
	}, 30*time.Second, 100*time.Millisecond, "the Identify event never reached the server")

	writeGlobalConfig("instrumentation_opt_in: false\n")
	globalconfig.EnsureGlobalConfig()
	resetForTest(server.URL)
	InitAmplitude()
	require.False(t, ampli.Instance.InitializedAndEnabled())

	writeGlobalConfig("instrumentation_opt_in: true\n")
	globalconfig.EnsureGlobalConfig()
	versionconstants.AmplitudeAPIKey = ""
	resetForTest(server.URL)
	InitAmplitude()
	require.False(t, ampli.Instance.InitializedAndEnabled())
}

// resetForTest clears the initialization state and sets the server URL.
func resetForTest(url string) {
	serverURL = url
	initialized = false
	identified = false
	ampli.Instance.Client = nil
	ampli.Instance.Disabled = false
}
