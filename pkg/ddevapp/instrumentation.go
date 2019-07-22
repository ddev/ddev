package ddevapp

import (
	"fmt"
	"github.com/denisbrodbeck/machineid"
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/drud/ddev/pkg/nodeps"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/version"
	"github.com/getsentry/raven-go"
	"gopkg.in/segmentio/analytics-go.v3"
	"runtime"
	"strconv"
)

var hashedHostID string

// GetInstrumentationUser normally gets just the hashed hostID but if
// an explicit user is provided in global_config.yaml that will be prepended.
func GetInstrumentationUser() string {
	userID := hashedHostID
	if globalconfig.DdevGlobalConfig.InstrumentationUser != "" {
		userID = globalconfig.DdevGlobalConfig.InstrumentationUser + "_" + hashedHostID
	}
	return userID
}

// SetInstrumentationBaseTags sets the basic always-used tags for Sentry/Raven/Segment
func SetInstrumentationBaseTags() {
	if globalconfig.DdevGlobalConfig.InstrumentationOptIn {
		dockerVersion, _ := version.GetDockerVersion()
		composeVersion, _ := version.GetDockerComposeVersion()
		isToolbox := nodeps.IsDockerToolbox()

		raven.SetRelease("ddev@" + version.VERSION)

		nodeps.InstrumentationTags["OS"] = runtime.GOOS
		nodeps.InstrumentationTags["dockerVersion"] = dockerVersion
		nodeps.InstrumentationTags["dockerComposeVersion"] = composeVersion
		nodeps.InstrumentationTags["dockerToolbox"] = strconv.FormatBool(isToolbox)
		nodeps.InstrumentationTags["version"] = version.VERSION
		nodeps.InstrumentationTags["ServerHash"] = GetInstrumentationUser()

		// Add these tags to sentry/raven
		raven.SetTagsContext(nodeps.InstrumentationTags)
	}
}

// SetInstrumentationAppTags creates app-specific tags for Sentry/Raven/Segment
func (app *DdevApp) SetInstrumentationAppTags() {
	ignoredProperties := []string{"approot", "hostnames", "httpurl", "httpsurl", "mailhog_url", "name", "phpmyadmin_url", "router_status_log", "shortroot", "urls"}

	if globalconfig.DdevGlobalConfig.InstrumentationOptIn {
		describeTags, _ := app.Describe()
		for key, val := range describeTags {
			if !nodeps.ArrayContainsString(ignoredProperties, key) {
				nodeps.InstrumentationTags[key] = fmt.Sprintf("%v", val)
			}
		}
		raven.SetTagsContext(nodeps.InstrumentationTags)
	}
}

// SegmentUser does the enqueue of the Identify action, identifying the user
// Here we just use the hashed hostid as the user id
func SegmentUser(client analytics.Client, hashedID string) error {
	err := client.Enqueue(analytics.Identify{
		UserId: hashedID,
		Traits: analytics.NewTraits().
			Set("OS", runtime.GOOS).
			Set("ddev-version", version.VERSION),
	})

	if err != nil {
		return err
	}
	return nil
}

// SegmentEvent provides the event and traits that go with it.
func SegmentEvent(client analytics.Client, hashedID string, event string) error {

	properties := analytics.NewProperties()

	for key, val := range nodeps.InstrumentationTags {
		if val != "" {
			properties = properties.Set(key, val)
		}
	}

	err := client.Enqueue(analytics.Track{
		UserId:     hashedID,
		Event:      event,
		Properties: properties,
	})

	return err
}

// SendInstrumentationEvents does the actual send to sentry/segment
func SendInstrumentationEvents(event string) {

	if globalconfig.DdevGlobalConfig.InstrumentationOptIn && nodeps.IsInternetActive() {
		_ = raven.CaptureMessageAndWait("Usage: ddev "+event, map[string]string{"severity-level": "info", "report-type": "usage"})

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
