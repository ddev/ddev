package ddevapp_test

import (
	"fmt"
	"github.com/ddev/ddev/pkg/dockerutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/testcommon"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExtraPortExpose tests exposing additional ports with web_extra_exposed_ports.
// It also tests web_extra_daemons
func TestExtraPortExpose(t *testing.T) {
	if dockerutil.IsColima() || dockerutil.IsLima() {
		t.Skip("skipping on Lima/Colima because of unpredictable behavior, unable to connect")
	}
	assert := asrt.New(t)

	site := TestSites[0]

	testcommon.ClearDockerEnv()
	app := new(ddevapp.DdevApp)
	err := app.Init(site.Dir)
	assert.NoError(err)

	t.Cleanup(func() {
		err = app.Stop(true, false)
		assert.NoError(err)
		_ = os.RemoveAll(filepath.Join(app.AppRoot, "testfile1.html"))
	})

	err = os.WriteFile(filepath.Join(app.AppRoot, "testfile.html"), []byte(`this is test1 in root`), 0755)
	require.NoError(t, err)
	err = os.MkdirAll(filepath.Join(app.AppRoot, "sub"), 0777)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(app.AppRoot, "sub", "testfile.html"), []byte(`this is test2 in root/sub`), 0755)
	require.NoError(t, err)

	app.WebExtraExposedPorts = []ddevapp.WebExposedPort{
		{Name: "first", WebContainerPort: 3000, HTTPPort: 2999, HTTPSPort: 3000},
		{Name: "second", WebContainerPort: 4000, HTTPPort: 3999, HTTPSPort: 4000},
	}
	app.WebExtraDaemons = []ddevapp.WebExtraDaemon{
		{Name: "FirstDaemon", Command: "php -S 0.0.0.0:3000", Directory: "/var/www/html"},
		{Name: "SecondDaemon", Command: "php -S 0.0.0.0:4000", Directory: "/var/www/html/sub"},
	}
	err = app.Start()
	if err != nil {
		logs, logErr := exec.RunCommand("docker", []string{"logs", "ddev-" + app.Name + "-web"})
		t.Fatalf("app failed to start: %v, logErr=%v logs=%v", err, logErr, logs)
	}

	// Careful with portsToTest because https ports won't work on GitHub Actions Colima tests (although they work fine on normal Mac)
	portsToTest := []string{"3000", "4000"}
	if app.CanUseHTTPOnly() {
		portsToTest = []string{"2999", "3999"}
	}

	for i, p := range portsToTest {
		url := fmt.Sprintf("%s:%s/testfile.html", app.GetPrimaryURL(), p)
		out, resp, err := testcommon.GetLocalHTTPResponse(t, url)
		require.NoError(t, err, "failed to get hit url %s, out=%s, resp=%v err=%v", url, out, resp, err)
		require.Contains(t, out, fmt.Sprintf("this is test%d", i+1))
	}
}
