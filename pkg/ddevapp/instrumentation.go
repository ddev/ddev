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
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/ddev/ddev/pkg/version"
	"github.com/ddev/ddev/pkg/versionconstants"
	"github.com/denisbrodbeck/machineid"
	"gopkg.in/segmentio/analytics-go.v3"
)

var hashedHostID string

// SegmentNoopLogger defines a no-op logger to prevent Segment log messages from being emitted
type SegmentNoopLogger struct{}

func (n *SegmentNoopLogger) Logf(_ string, _ ...interface{})   {}
func (n *SegmentNoopLogger) Errorf(_ string, _ ...interface{}) {}

// ReportableEvents is the list of events that we choose to report specifically.
// Excludes non-ddev custom commands.
var ReportableEvents = map[string]bool{"start": true}

// GetInstrumentationUser normally gets just the hashed hostID but if
// an explicit user is provided in global_config.yaml that will be prepended.
func GetInstrumentationUser() string {
	return hashedHostID
}

// SetInstrumentationBaseTags sets the basic always-used tags for Segment
func SetInstrumentationBaseTags() {
	defer util.TimeTrack()()

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

// SetInstrumentationAppTags creates app-specific tags for Segment
func (app *DdevApp) SetInstrumentationAppTags() {
	defer util.TimeTrack()()

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

// SegmentUser does the enqueue of the Identify action, identifying the user
// Here we just use the hashed hostid as the user id
func SegmentUser(client analytics.Client, hashedID string) error {
	timezone, _ := time.Now().In(time.Local).Zone()
	lang := os.Getenv("LANG")
	err := client.Enqueue(analytics.Identify{
		UserId:  hashedID,
		Context: &analytics.Context{App: analytics.AppInfo{Name: "ddev", Version: versionconstants.DdevVersion}, OS: analytics.OSInfo{Name: runtime.GOOS}, Locale: lang, Timezone: timezone},
		Traits:  analytics.Traits{"instrumentation_user": globalconfig.DdevGlobalConfig.InstrumentationUser},
	})

	if err != nil {
		return err
	}
	return nil
}

// SegmentEvent provides the event and traits that go with it.
func SegmentEvent(client analytics.Client, hashedID string, event string) error {
	if _, ok := ReportableEvents[event]; !ok {
		// There's no need to waste people's time on custom commands.
		return nil
	}
	properties := analytics.NewProperties()

	for key, val := range nodeps.InstrumentationTags {
		if val != "" {
			properties = properties.Set(key, val)
		}
	}
	timezone, _ := time.Now().In(time.Local).Zone()
	lang := os.Getenv("LANG")
	err := client.Enqueue(analytics.Track{
		UserId:     hashedID,
		Event:      event,
		Properties: properties,
		Context:    &analytics.Context{App: analytics.AppInfo{Name: "ddev", Version: versionconstants.DdevVersion}, OS: analytics.OSInfo{Name: runtime.GOOS}, Locale: lang, Timezone: timezone},
	})

	return err
}

// SendInstrumentationEvents does the actual send to segment
func SendInstrumentationEvents(event string) {
	defer util.TimeTrack()()

	if globalconfig.DdevGlobalConfig.InstrumentationOptIn && globalconfig.IsInternetActive() {
		client, _ := analytics.NewWithConfig(versionconstants.SegmentKey, analytics.Config{
			Logger: &SegmentNoopLogger{},
		})

		err := SegmentEvent(client, GetInstrumentationUser(), event)
		if err != nil {
			output.UserOut.Debugf("error sending event to segment: %v", err)
		}
		err = client.Close()
		if err != nil {
			output.UserOut.Debugf("segment analytics client.close() failed: %v", err)
		}
	}
}

func init() {
	hashedHostID, _ = machineid.ProtectedID("ddev")
}
