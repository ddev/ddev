package ddevapp_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/testcommon"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNetworkAmbiguity tests the behavior and setup of docker networking.
// There should be no crosstalk between different projects
func TestNetworkAmbiguity(t *testing.T) {
	assert := asrt.New(t)

	origDir, _ := os.Getwd()

	projects := map[string]string{
		"proj1": testcommon.CreateTmpDir(t.Name() + "proj1-"),
		"proj2": testcommon.CreateTmpDir(t.Name() + "proj2-"),
	}
	var err error

	t.Cleanup(func() {
		err = os.Chdir(origDir)
		assert.NoError(err)
		for projName, projDir := range projects {
			app, err := ddevapp.GetActiveApp(projName)
			assert.NoError(err)
			err = app.Stop(true, false)
			assert.NoError(err)
			err = os.RemoveAll(projDir)
			assert.NoError(err)
		}
	})

	// Start a set of projects that contain a simple test container
	// Verify that test is ambiguous or not, with or without `links`
	// Use docker network inspect? Use getent hosts test
	for projName, projDir := range projects {
		// Clean up any existing name conflicts
		app, err := ddevapp.GetActiveApp(projName)
		if err == nil {
			err = app.Stop(true, false)
			assert.NoError(err)
		}
		// Create new app
		app, err = ddevapp.NewApp(projDir, false)
		assert.NoError(err)
		app.Type = nodeps.AppTypePHP
		app.Name = projName
		err = app.WriteConfig()
		assert.NoError(err)
		err = fileutil.CopyFile(filepath.Join(origDir, "testdata", t.Name(), "docker-compose.test.yaml"), app.GetConfigPath("docker-compose.test.yaml"))
		assert.NoError(err)
		err = app.Start()
		assert.NoError(err)
	}

	// With the improved two-network handling, the simple service names
	// are no longer ambiguious. We'll see just one entry for web and one for db
	// very ambiguous, but just one on web, because it has 'links'
	expectations := map[string]int{"web": 1, "db": 1}
	for projName := range projects {
		app, err := ddevapp.GetActiveApp(projName)
		assert.NoError(err)
		for c, expectation := range expectations {
			out, _, err := app.Exec(&ddevapp.ExecOpts{
				Service: c,
				Cmd:     "getent hosts test",
			})
			require.NoError(t, err)
			out = strings.Trim(out, "\r\n ")
			ips := strings.Split(out, "\n")
			assert.Len(ips, expectation)
		}
	}

}
