package cmd

import (
	"fmt"
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/globalconfig"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

// TestDebugRouterNginxConfigCmd ensures the debug router-nginx-config prints the nginx config to stdout
func TestDebugRouterNginxConfigCmd(t *testing.T) {
	if globalconfig.DdevGlobalConfig.IsTraefikRouter() {
		t.Skip("Skipping when UseTrafik is set, as it's invalid")
	}
	assert := asrt.New(t)

	site := TestSites[0]
	cleanup := site.Chdir()

	app, err := ddevapp.NewApp(site.Dir, false)
	assert.NoError(err)

	t.Cleanup(func() {
		// Make sure all databases are back to default empty
		cleanup()
	})

	err = app.Start()
	require.NoError(t, err)

	// Test default invocation
	out, err := exec.RunHostCommand(DdevBin, "debug", "router-nginx-config")
	assert.NoError(err)
	assert.Contains(
		out,
		fmt.Sprintf("# Container=ddev-%s-web", app.Name),
		"Cannot find generated config of wordpress test site in generated router nginx config",
	)

	assert.Contains(
		out,
		"proxy_set_header",
		"Cannot find nginx related proxy settings in output",
	)
}
