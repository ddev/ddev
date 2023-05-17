package ddevapp_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/ddev/ddev/pkg/util"
	asrt "github.com/stretchr/testify/assert"
)

// TestSetInstrumentationAppTags checks to see that tags are properly set
// and tries to make sure no leakage happens with URLs or other
// tags that we don't want to see.
func TestSetInstrumentationAppTags(t *testing.T) {
	assert := asrt.New(t)

	site := TestSites[0]
	runTime := util.TimeTrackC(fmt.Sprintf("%s %s", site.Name, t.Name()))

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

	for _, unwanted := range []string{"approot", "hostname", "hostnames", "name", "router_status_log", "shortroot"} {
		assert.Empty(nodeps.InstrumentationTags[unwanted])
	}

	// Make sure that expected attributes come through
	for _, wanted := range []string{"database_type", "nfs_mount_enabled", "ProjectID", "php_version", "router_http_port", "router_https_port", "router_status", "ssh_agent_status", "status", "type", "webimg", "webserver_type"} {
		assert.NotEmpty(nodeps.InstrumentationTags[wanted], "tag '%s' was not found and it should have been", wanted)
	}
	runTime()
}
