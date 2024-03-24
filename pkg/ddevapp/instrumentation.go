package ddevapp

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/version"
	"github.com/ddev/ddev/pkg/versionconstants"
	"github.com/denisbrodbeck/machineid"
)

var hashedHostID string

// ReportableEvents is the list of events that we choose to report specifically.
// Excludes non-ddev custom commands.
var ReportableEvents = map[string]bool{"start": true}

// GetInstrumentationUser normally gets the hashed hostID but if an explicit
// user is provided in global_config.yaml that will be prepended.
func GetInstrumentationUser() string {
	return hashedHostID
}

// SetInstrumentationBaseTags sets the basic always-used tags for telemetry
func SetInstrumentationBaseTags() {
	// defer util.TimeTrack()()

	if globalconfig.DdevGlobalConfig.InstrumentationOptIn {
		dockerVersion, _ := dockerutil.GetDockerVersion()
		dockerPlaform, _ := version.GetDockerPlatform()
		timezone, _ := time.Now().In(time.Local).Zone()
		lang := os.Getenv("LANG")

		nodeps.InstrumentationTags["OS"] = runtime.GOOS
		nodeps.InstrumentationTags["architecture"] = runtime.GOARCH
		wslDistro := nodeps.GetWSLDistro()
		if wslDistro != "" {
			nodeps.InstrumentationTags["isWSL"] = "true"
			nodeps.InstrumentationTags["wslDistro"] = wslDistro
			nodeps.InstrumentationTags["OS"] = "wsl2"
		}
		nodeps.InstrumentationTags["dockerVersion"] = dockerVersion
		nodeps.InstrumentationTags["dockerPlatform"] = dockerPlaform
		nodeps.InstrumentationTags["dockerToolbox"] = strconv.FormatBool(false)
		nodeps.InstrumentationTags["version"] = versionconstants.DdevVersion
		nodeps.InstrumentationTags["ServerHash"] = GetInstrumentationUser()
		nodeps.InstrumentationTags["timezone"] = timezone
		nodeps.InstrumentationTags["language"] = lang
	}
}

// SetInstrumentationAppTags creates app-specific tags for telemetry
func (app *DdevApp) SetInstrumentationAppTags() {
	// defer util.TimeTrack()()

	ignoredProperties := []string{"approot", "hostname", "hostnames", "name", "router_status_log", "shortroot"}

	describeTags, _ := app.Describe(false)
	for key, val := range describeTags {
		// Make sure none of the "URL" attributes or the ignoredProperties comes through
		if strings.Contains(strings.ToLower(key), "url") || nodeps.ArrayContainsString(ignoredProperties, key) {
			continue
		}
		nodeps.InstrumentationTags[key] = fmt.Sprintf("%v", val)
	}
	nodeps.InstrumentationTags["ProjectID"] = app.ProtectedID()
}

func init() {
	hashedHostID, _ = machineid.ProtectedID("ddev")
}
