package ddevapp_test

import (
	"fmt"
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/drud/ddev/pkg/nodeps"
	"github.com/drud/ddev/pkg/testcommon"
	"github.com/drud/ddev/pkg/util"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestHardenedStart makes sure we can do a start and basic use with hardened images
func TestHardenedStart(t *testing.T) {
	if nodeps.IsMacM1() {
		t.Skip("Skipping TestHardenedStart on Mac M1 because of useless Docker Desktop failures to connect")
	}

	assert := asrt.New(t)
	app := &ddevapp.DdevApp{}

	origDir, _ := os.Getwd()

	site := TestSites[0]
	err := app.Init(site.Dir)
	assert.NoError(err)
	if app.IsMutagenEnabled() {
		t.Skip("Skipping test because mutagen is enabled")
	}

	runTime := util.TimeTrack(time.Now(), fmt.Sprintf("%s DdevStart", site.Name))

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
	err = fileutil.TemplateStringToFile("<?php\necho \"hi there\";\n", nil, filepath.Join(app.AppRoot, "test.php"))
	require.NoError(t, err)

	err = app.Start()
	require.NoError(t, err)

	testURL := app.GetPrimaryURL() + "/test.php"
	out, resp, err := testcommon.GetLocalHTTPResponse(t, testURL)
	assert.NoError(err, "Error getting response from %s: %v, out=%s, resp=%v", testURL, err, out, resp)
}
