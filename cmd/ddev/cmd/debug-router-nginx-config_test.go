package cmd

import (
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/exec"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

// TestDebugRouterNginxConfigCmd ensures the debug router-nginx-config prints the nginx config to stdout
func TestDebugRouterNginxConfigCmd(t *testing.T) {
	assert := asrt.New(t)

	site := TestSites[0]
	cleanup := site.Chdir()

	app, err := ddevapp.NewApp(site.Dir, false)
	assert.NoError(err)

	t.Cleanup(func() {
		// Make sure all databases are back to default empty
		_ = app.Stop(true, false)
		cleanup()
	})

	err = app.Start()
	require.NoError(t, err)

	// Test default invocation
	args := []string{"debug", "router-nginx-config"}
	out, err := exec.RunCommand(DdevBin, args)
	assert.NoError(err)
	assert.Contains(
		out,
		"# Container=ddev-TestCmdWordpress-web",
		"Cannot find generated config of wordpress test site in generated router nginx config",
	)

	assert.Contains(
		out,
		"proxy_set_header",
		"Cannot find nginx related proxy settings in outout",
	)
}
