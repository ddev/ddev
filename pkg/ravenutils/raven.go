package ravenutils

// RavenTags provides exposed access to tags for Sentry
var RavenTags = make(map[string]string)

// AddRavenTags adds a set of tags to the exposed RavenTags so it can be used later
// by logrus_sentry.NewAsyncWithTagsSentryHook
func AddRavenTags(tags map[string]string) {
	for k, v := range tags {
		RavenTags[k] = v
	}
}
