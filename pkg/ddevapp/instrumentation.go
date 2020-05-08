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
	"github.com/drud/ddev/pkg/version"
	"gopkg.in/segmentio/analytics-go.v3"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var hashedHostID string

// ReportableEvents is the list of events that we choose to report specifically.
// Excludes non-ddev custom commands.
var ReportableEvents = map[string]bool{"auth": true, "composer": true, "config": true, "debug": true, "delete": true, "describe": true, "exec": true, "export-db": true, "import-db": true, "import-files": true, "launch": true, "list": true, "logs": true, "mysql": true, "pause": true, "poweroff": true, "pull": true, "restart": true, "restore-snapshot": true, "sequelpro": true, "share": true, "snapshot": true, "ssh": true, "start": true, "stop": true, "xdebug": true}

// GetInstrumentationUser normally gets just the hashed hostID but if
// an explicit user is provided in global_config.yaml that will be prepended.
func GetInstrumentationUser() string {
	return hashedHostID
}

// SetInstrumentationBaseTags sets the basic always-used tags for Segment
func SetInstrumentationBaseTags() {
	if globalconfig.DdevGlobalConfig.InstrumentationOptIn {
		dockerVersion, _ := version.GetDockerVersion()
		composeVersion, _ := version.GetDockerComposeVersion()
		isToolbox := nodeps.IsDockerToolbox()

		nodeps.InstrumentationTags["OS"] = runtime.GOOS
		if runtime.GOOS == "linux" {
			wslDistro := os.Getenv("WSL_DISTRO_NAME")
			if wslDistro != "" {
				nodeps.InstrumentationTags["isWSL"] = "true"
				nodeps.InstrumentationTags["wslDistro"] = wslDistro
			}
		}
		nodeps.InstrumentationTags["dockerVersion"] = dockerVersion
		nodeps.InstrumentationTags["dockerComposeVersion"] = composeVersion
		nodeps.InstrumentationTags["dockerToolbox"] = strconv.FormatBool(isToolbox)
		nodeps.InstrumentationTags["version"] = version.VERSION
		nodeps.InstrumentationTags["ServerHash"] = GetInstrumentationUser()
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
	ignoredProperties := []string{"approot", "hostname", "hostnames", "name", "router_status_log", "shortroot"}

	describeTags, _ := app.Describe()
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
		Context: &analytics.Context{App: analytics.AppInfo{Name: "ddev", Version: version.VERSION}, OS: analytics.OSInfo{Name: runtime.GOOS}, Locale: lang, Timezone: timezone},
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
		event = "customcommand"
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
		Context:    &analytics.Context{App: analytics.AppInfo{Name: "ddev", Version: version.VERSION}, OS: analytics.OSInfo{Name: runtime.GOOS}, Locale: lang, Timezone: timezone},
	})

	return err
}

// SendInstrumentationEvents does the actual send to segment
func SendInstrumentationEvents(event string) {

	if globalconfig.DdevGlobalConfig.InstrumentationOptIn && nodeps.IsInternetActive() {
		client := analytics.New(version.SegmentKey)

		err := SegmentUser(client, GetInstrumentationUser())
		if err != nil {
			output.UserOut.Debugf("error sending hashedHostID to segment: %v", err)
		}

		err = SegmentEvent(client, GetInstrumentationUser(), event)
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
