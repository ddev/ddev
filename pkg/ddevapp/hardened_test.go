package ddevapp_test

import (
	"fmt"
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/drud/ddev/pkg/testcommon"
	"github.com/drud/ddev/pkg/util"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"
)

// TestHardenedStart makes sure we can do a start and basic use with hardened images
func TestHardenedStart(t *testing.T) {
	assert := asrt.New(t)
	app := &ddevapp.DdevApp{}

	// Make sure this leaves us in the original test directory
	origDir, _ := os.Getwd()

	site := TestSites[0]

	runTime := util.TimeTrack(time.Now(), fmt.Sprintf("%s DdevStart", site.Name))

	_, err := exec.RunHostCommand(DdevBin, "poweroff")
	require.NoError(t, err)

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
	})

	err = globalconfig.ReadGlobalConfig()
	require.NoError(t, err)
	globalconfig.DdevGlobalConfig.UseHardenedImages = true
	err = globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig)
	require.NoError(t, err)
	err = app.Init(site.Dir)
	assert.NoError(err)

	// Create the simplest possible php file
	err = fileutil.TemplateStringToFile("<?php\necho \"hi there\";\n", nil, filepath.Join(app.AppRoot, "test.php"))
	require.NoError(t, err)

	err = app.Start()
	require.NoError(t, err)

	testURL := path.Join(app.GetPrimaryURL(), "test.php")
	out, resp, err := testcommon.GetLocalHTTPResponse(t, testURL)
	assert.NoError(err, "Error getting response from %s: %v, out=%s, resp=%v", testURL, err, out, resp)
}
