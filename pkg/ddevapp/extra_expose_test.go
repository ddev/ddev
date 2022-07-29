package ddevapp_test

import (
	"fmt"
	. "github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/testcommon"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
)

// TestExtraPortExpose tests exposing additional ports with web_extra_exposed_ports.
func TestExtraPortExpose(t *testing.T) {
	assert := asrt.New(t)

	testDir := TestSites[0].Dir
	app, err := NewApp(testDir, true)
	assert.NoError(err)

	t.Cleanup(func() {
		err = app.Stop(true, false)
		assert.NoError(err)
	})

	err = os.WriteFile(filepath.Join(app.AppRoot, "testfile.html"), []byte(`this is a test`), 0755)
	require.NoError(t, err)

	app.WebExtraExposedPorts = []WebExposedPort{
		{Name: "first", WebContainerPort: 3000, HTTPPort: 2999, HTTPSPort: 3000},
		{Name: "second", WebContainerPort: 4000, HTTPPort: 3999, HTTPSPort: 4000},
	}
	err = app.Start()
	require.NoError(t, err)

	// Run php built-in webserver on 3000 and 4000
	_, _, err = app.Exec(&ExecOpts{
		Cmd: "nohup php -S 0.0.0.0:3000 &",
	})
	_, _, err = app.Exec(&ExecOpts{
		Cmd: "nohup php -S 0.0.0.0:4000 &",
	})

	for _, p := range []string{"3000", "4000"} {
		out, resp, err := testcommon.GetLocalHTTPResponse(t, fmt.Sprintf("%s:%s/testfile.html", app.GetPrimaryURL(), p))
		require.NoError(t, err, "failed to get hit port %s, out=%s, resp=%v err=%v", p, out, resp, err)
		assert.Contains(out, "this is a test")
	}
}
