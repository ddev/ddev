package ddevapp_test

import (
	"fmt"
	"github.com/ddev/ddev/pkg/dockerutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/ddev/ddev/pkg/config/types"
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/ddev/ddev/pkg/util"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHardenedStart makes sure we can do a start and basic use with hardened images
func TestHardenedStart(t *testing.T) {
	if nodeps.IsAppleSilicon() || nodeps.IsWSL2() || dockerutil.IsRancherDesktop() {
		t.Skip("Skipping TestHardenedStart because of useless failures to connect on some platforms, no need to test hardened on arm64")
	}

	assert := asrt.New(t)
	app := &ddevapp.DdevApp{}

	origDir, _ := os.Getwd()

	testSite := 0
	// Prefer the drupal7 project, as it does ln -s into /usr/local/bin, possibly
	// requiring sudo, which isn't installed
	if len(TestSites) >= 3 {
		testSite = 2
	}
	site := TestSites[testSite]
	err := app.Init(site.Dir)
	assert.NoError(err)

	runTime := util.TimeTrackC(fmt.Sprintf("%s DdevStart", site.Name))

	ddevapp.PowerOff()

	t.Cleanup(func() {
		runTime()
		err = os.Chdir(origDir)
		assert.NoError(err)
		err = app.Stop(true, false)
		assert.NoError(err)
		err = os.RemoveAll(filepath.Join(app.AppRoot, "test.php"))
		assert.NoError(err)
		err = globalconfig.ReadGlobalConfig()
		require.NoError(t, err)
		globalconfig.DdevGlobalConfig.UseHardenedImages = false
		err = globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig)
		require.NoError(t, err)
		ddevapp.PowerOff()
	})

	err = globalconfig.ReadGlobalConfig()
	require.NoError(t, err)
	globalconfig.DdevGlobalConfig.UseHardenedImages = true
	err = globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig)
	require.NoError(t, err)

	// Create the simplest possible php file
	err = fileutil.TemplateStringToFile("<?php\necho \"hi there\";\n", nil, filepath.Join(app.AppRoot, app.Docroot, "test.php"))
	require.NoError(t, err)

	err = app.Init(site.Dir)
	require.NoError(t, err)

	app.PerformanceMode = types.PerformanceModeNone

	err = app.Start()
	require.NoError(t, err)

	testURL := app.GetPrimaryURL() + "/test.php"
	out, resp, err := testcommon.GetLocalHTTPResponse(t, testURL)
	assert.NoError(err, "Error getting response from %s: %v, out=%s, resp=%v", testURL, err, out, resp)
}
