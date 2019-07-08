package ddevapp

import (
	"github.com/drud/ddev/pkg/nodeps"
	"github.com/drud/ddev/pkg/version"
	"gopkg.in/segmentio/analytics-go.v3"
	"runtime"
	"strings"
)

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
func SegmentEvent(client analytics.Client, hashedID string, event string, project *DdevApp) error {

	dockerVersion, _ := version.GetDockerVersion()
	composeVersion, _ := version.GetDockerComposeVersion()
	isToolbox := nodeps.IsDockerToolbox()
	ignoredProperties := []string{"approot", "hostnames", "httpurl", "httpsurl", "mailhog_url", "name", "phpmyadmin_url", "router_status_log", "shortroot", "urls"}

	describeTags, _ := project.Describe()
	properties := analytics.NewProperties().Set("dockerVersion", dockerVersion).Set("dockerComposeVersion", composeVersion).Set("isDockerToolbox", isToolbox)

	for key, val := range describeTags {
		var tagVal string
		if !nodeps.ArrayContainsString(ignoredProperties, key) {
			if valString, ok := val.(string); ok {
				tagVal = valString
			} else if valAry, ok := val.([]string); ok {
				tagVal = strings.Join(valAry, " ")
			}
			if tagVal != "" {
				properties = properties.Set(key, tagVal)
			}
		}
	}

	err := client.Enqueue(analytics.Track{
		UserId:     hashedID,
		Event:      event,
		Properties: properties,
	})

	return err
}
