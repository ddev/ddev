package ddevapp_test

import (
	"fmt"
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/drud/ddev/pkg/util"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"runtime"
	"testing"
	"time"
)

// TestComposeV1 makes sure we can do basic actions with docker-compose v1
func TestComposeV1(t *testing.T) {
	if runtime.GOARCH != "amd64" {
		t.Skip("Skipping  because arm64 is not supported for docker-compose v1")
	}
	assert := asrt.New(t)
	app := &ddevapp.DdevApp{}

	// Make sure this leaves us in the original test directory
	origDir, _ := os.Getwd()

	site := TestSites[0]

	runTime := util.TimeTrack(time.Now(), fmt.Sprintf("%s DdevStart", site.Name))

	ddevapp.PowerOff()

	t.Cleanup(func() {
		runTime()
		err := os.Chdir(origDir)
		assert.NoError(err)
		err = app.Stop(true, false)
		assert.NoError(err)
		err = globalconfig.ReadGlobalConfig()
		require.NoError(t, err)
		globalconfig.DdevGlobalConfig.RequiredDockerComposeVersion = ""
		err = globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig)
		require.NoError(t, err)
	})

	err := globalconfig.ReadGlobalConfig()
	require.NoError(t, err)
	globalconfig.DdevGlobalConfig.RequiredDockerComposeVersion = "v1.29.2"
	err = globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig)
	require.NoError(t, err)
	err = app.Init(site.Dir)
	assert.NoError(err)

	err = app.Start()
	assert.NoError(err)

}
