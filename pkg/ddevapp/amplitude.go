package ddevapp

import (
	"fmt"
	"strings"

	"github.com/ddev/ddev/pkg/amplitude"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/util"
	"github.com/ddev/ddev/third_party/ampli"
	"github.com/denisbrodbeck/machineid"
)

// TrackProject collects and tracks information about the project for instrumentation.
func (app *DdevApp) TrackProject() {
	runTime := util.TimeTrack()
	defer runTime()

	// Early exit if instrumentation is disabled.
	if ampli.Instance.Disabled {
		return
	}

	ignoredProperties := []string{"approot", "hostname", "hostnames", "name", "router_status_log", "shortroot"}
	properties := map[string]string{}

	describeTags, _ := app.Describe(false)
	for key, val := range describeTags {
		// Make sure none of the "URL" attributes or the ignoredProperties comes through
		if strings.Contains(strings.ToLower(key), "url") || nodeps.ArrayContainsString(ignoredProperties, key) {
			continue
		}
		properties[key] = fmt.Sprintf("%v", val)
	}

	builder := ampli.Project.Builder().
		Id(app.ProtectedID()).
		Properties(properties)

	ampli.Instance.Project("", builder.Build(), amplitude.GetEventOptions())
}

// ProtectedID returns the unique hash value for the project.
func (app *DdevApp) ProtectedID() string {
	appID, _ := machineid.ProtectedID("ddev" + app.Name)
	return appID
}
