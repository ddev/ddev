package ddevapp_test

import (
	"fmt"
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/nodeps"
	"github.com/drud/ddev/pkg/testcommon"
	"github.com/drud/ddev/pkg/util"
	asrt "github.com/stretchr/testify/assert"
	"strings"
	"testing"
	"time"
)

//TestSetInstrumentationAppTags checks to see that tags are properly set
//and tries to make sure no leakage happens with URLs or other
//tags that we don't want to see.
func TestSetInstrumentationAppTags(t *testing.T) {
	assert := asrt.New(t)

	site := TestSites[0]
	runTime := util.TimeTrack(time.Now(), fmt.Sprintf("%s %s", site.Name, t.Name()))

	testcommon.ClearDockerEnv()
	app := new(ddevapp.DdevApp)

	err := app.Init(site.Dir)
	assert.NoError(err)
	t.Cleanup(func() {
		_ = app.Stop(true, false)
	})
	app.SetInstrumentationAppTags()

	// Make sure that none of the "url" items in app.desc are being reported
	for k := range nodeps.InstrumentationTags {
		assert.NotContains(strings.ToLower(k), "url")
	}
	assert.NotEmpty(nodeps.InstrumentationTags["ProjectID"])
	runTime()
}
