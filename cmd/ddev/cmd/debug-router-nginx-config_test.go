package cmd

import (
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/exec"
	asrt "github.com/stretchr/testify/assert"
	"testing"
)

// TestDebugRouterNginxConfigCmd ensures the debug router-nginx-config prints the nginx config to stdout
func TestDebugRouterNginxConfigCmd(t *testing.T) {
	assert := asrt.New(t)
	site := TestSites[0]
	cleanup := site.Chdir()
	defer cleanup()

	_, err := ddevapp.GetActiveApp(site.Name)
	assert.NoError(err)

	// Test default invocation
	out, err := exec.RunHostCommand(DdevBin, "debug", "router-nginx-config")
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
