package ddevapp

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/denisbrodbeck/machineid"
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/drud/ddev/pkg/nodeps"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/util"
	"github.com/drud/ddev/pkg/version"
	"gopkg.in/segmentio/analytics-go.v3"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var hashedHostID string

// SegmentNoopLogger defines a no-op logger to prevent Segment log messages from being emitted
type SegmentNoopLogger struct{}

func (n *SegmentNoopLogger) Logf(format string, args ...interface{})   {}
func (n *SegmentNoopLogger) Errorf(format string, args ...interface{}) {}

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
	runTime := util.TimeTrack(time.Now(), "SetInstrumentationBaseTags")
	defer runTime()

	if globalconfig.DdevGlobalConfig.InstrumentationOptIn {
		dockerVersion, _ := version.GetDockerVersion()

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
		nodeps.InstrumentationTags["dockerToolbox"] = strconv.FormatBool(false)
		nodeps.InstrumentationTags["version"] = version.DdevVersion
		nodeps.InstrumentationTags["ServerHash"] = GetInstrumentationUser()
		nodeps.InstrumentationTags["timezone"] = timezone
		nodeps.InstrumentationTags["language"] = lang
	}
}

// getProjectHash combines the machine ID and project name and then
// hashes the result, so we can end up with a unique project id
func getProjectHash(projectName string) string {
	ph := hmac.New(sha256.New, []byte(GetInstrumentationUser()+projectName))
	_, _ = ph.Write([]byte("phash"))
	return hex.EncodeToString(ph.Sum(nil))
}

// SetInstrumentationAppTags creates app-specific tags for Segment
func (app *DdevApp) SetInstrumentationAppTags() {
	runTime := util.TimeTrack(time.Now(), "SetInstrumentationAppTags")
	defer runTime()

	ignoredProperties := []string{"approot", "hostname", "hostnames", "name", "router_status_log", "shortroot"}

	describeTags, _ := app.Describe(false)
	for key, val := range describeTags {
		// Make sure none of the "URL" attributes or the ignoredProperties comes through
		if strings.Contains(strings.ToLower(key), "url") || nodeps.ArrayContainsString(ignoredProperties, key) {
			continue
		}
		nodeps.InstrumentationTags[key] = fmt.Sprintf("%v", val)
	}
	nodeps.InstrumentationTags["ProjectID"] = getProjectHash(app.Name)
}

// SegmentUser does the enqueue of the Identify action, identifying the user
// Here we just use the hashed hostid as the user id
func SegmentUser(client analytics.Client, hashedID string) error {
	timezone, _ := time.Now().In(time.Local).Zone()
	lang := os.Getenv("LANG")
	err := client.Enqueue(analytics.Identify{
		UserId:  hashedID,
		Context: &analytics.Context{App: analytics.AppInfo{Name: "ddev", Version: version.DdevVersion}, OS: analytics.OSInfo{Name: runtime.GOOS}, Locale: lang, Timezone: timezone},
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
		Context:    &analytics.Context{App: analytics.AppInfo{Name: "ddev", Version: version.DdevVersion}, OS: analytics.OSInfo{Name: runtime.GOOS}, Locale: lang, Timezone: timezone},
	})

	return err
}

// SendInstrumentationEvents does the actual send to segment
func SendInstrumentationEvents(event string) {
	runTime := util.TimeTrack(time.Now(), "SendInstrumentationEvents")
	defer runTime()

	if globalconfig.DdevGlobalConfig.InstrumentationOptIn && globalconfig.IsInternetActive() {
		client, _ := analytics.NewWithConfig(version.SegmentKey, analytics.Config{
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
