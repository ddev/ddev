package ddevapp_test

import (
	"fmt"
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/nodeps"
	"github.com/drud/ddev/pkg/testcommon"
	"github.com/drud/ddev/pkg/util"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"strings"
	"testing"
	"time"
)

// TestNodeJSVersions whether we can configure nodejs versions
func TestNodeJSVersions(t *testing.T) {
	assert := asrt.New(t)

	site := TestSites[0]
	origDir, _ := os.Getwd()
	_ = os.Chdir(site.Dir)

	runTime := util.TimeTrack(time.Now(), t.Name())

	testcommon.ClearDockerEnv()
	app, err := ddevapp.NewApp(site.Dir, true)
	assert.NoError(err)
	t.Cleanup(func() {
		runTime()
		err = os.Chdir(origDir)
		assert.NoError(err)
		err = app.Stop(true, false)
		assert.NoError(err)
	})

	err = app.Start()
	require.NoError(t, err)
	// As of v1.19.0, the nvm_dir doesn't get cleaned up on delete,
	//so on a machine where this test has run before this will fail, as nvm has been set up
	_, _, err = app.Exec(&ddevapp.ExecOpts{
		Cmd: fmt.Sprintf("rm -rf /mnt/ddev-global-cache/nvm_dir/%s-web", site.Name),
	})
	assert.NoError(err)

	for _, v := range nodeps.GetValidNodeVersions() {
		app.NodeJSVersion = v
		err = app.Restart()
		assert.NoError(err)
		out, _, err := app.Exec(&ddevapp.ExecOpts{
			Cmd: "node --version",
		})
		assert.NoError(err)
		assert.True(strings.HasPrefix(out, "v"+v), "Expected node version to start with '%s', but got %s", v, out)
	}

	out, _, err := app.Exec(&ddevapp.ExecOpts{
		Cmd: `bash -ic "nvm install 6 && node --version"`,
	})
	assert.NoError(err)
	assert.Contains(out, "Now using node v6")

	out, err = exec.RunHostCommand(DdevBin, "nvm", "install", "8")
	assert.NoError(err)
	assert.Contains(out, "Now using node v8")
}
