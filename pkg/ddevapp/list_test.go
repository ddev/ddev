package ddevapp_test

import (
	"bytes"
	"os"
	"regexp"
	"runtime"
	"testing"

	"github.com/ddev/ddev/pkg/config/types"
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/testcommon"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestListWithoutDir prevents regression where ddev list panics if one of the
// sites found is missing a directory
func TestListWithoutDir(t *testing.T) {
	// Can't run with mutagen because we actually delete the alpha
	if runtime.GOOS == "windows" || nodeps.PerformanceModeDefault == types.PerformanceModeMutagen {
		t.Skip("Skipping because unreliable on Windows and can't be used with mutagen")
	}
	// Set up tests and give ourselves a working directory.
	assert := asrt.New(t)
	testcommon.ClearDockerEnv()
	origDir, _ := os.Getwd()

	// startCount is the count of apps at the start of this adventure
	apps := ddevapp.GetActiveProjects()
	startCount := len(apps)

	testDir := testcommon.CreateTmpDir(t.Name())
	defer testcommon.CleanupDir(testDir)

	err := os.MkdirAll(testDir+"/sites/default", 0777)
	assert.NoError(err)
	err = os.Chdir(testDir)
	assert.NoError(err)

	app, err := ddevapp.NewApp(testDir, true)
	app.Name = t.Name()
	_ = globalconfig.RemoveProjectInfo(app.Name)

	assert.NoError(err)
	_ = app.Stop(true, false)
	err = os.Chdir(testDir)
	require.NoError(t, err)

	t.Cleanup(func() {
		err = os.Chdir(origDir)
		assert.NoError(err)
		_ = globalconfig.RemoveProjectInfo(app.Name)
		err = app.Stop(true, false)
		assert.NoError(err)
		err = os.RemoveAll(testDir)
		assert.NoError(err)
		globalconfig.DdevGlobalConfig.SimpleFormatting = false
		_ = globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig)
	})
	globalconfig.DdevGlobalConfig.SimpleFormatting = true
	_ = globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig)
	app.Name = t.Name()
	app.Type = nodeps.AppTypeDrupal7
	err = app.WriteConfig()
	assert.NoError(err)

	// Do a start on the configured site.
	app, err = ddevapp.GetActiveApp("")
	assert.NoError(err)
	err = app.Start()
	assert.NoError(err)

	err = os.Chdir(origDir)
	assert.NoError(err)

	err = os.RemoveAll(testDir)
	assert.NoError(err)

	apps = ddevapp.GetActiveProjects()

	assert.EqualValues(len(apps), startCount+1)

	// Make a whole table and make sure our app directory missing shows up.
	// This could be done otherwise, but we'd have to go find the site in the
	// array first.
	out := bytes.Buffer{}
	table := ddevapp.CreateAppTable(&out, true)
	for _, site := range apps {
		desc, err := site.Describe(false)
		if err != nil {
			t.Fatalf("Failed to describe site %s: %v", site.GetName(), err)
		}

		ddevapp.RenderAppRow(table, desc)
	}
	table.Render()
	assert.Regexp(regexp.MustCompile("(?s)"+ddevapp.SiteDirMissing+".*"+testDir), out.String())

	err = app.Stop(true, false)
	assert.NoError(err)
}

// TestDdevList tests the ddevapp.List() functionality
// It's only here for profiling at this point.
func TestDdevList(_ *testing.T) {
	ddevapp.List(ddevapp.ListCommandSettings{
		ActiveOnly:          true,
		Continuous:          false,
		WrapTableText:       true,
		ContinuousSleepTime: 1,
		TypeFilter:          "",
	})
}
