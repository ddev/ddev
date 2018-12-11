package ddevapp

import (
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/drud/ddev/pkg/ravenutils"
	"github.com/drud/ddev/pkg/util"
	"github.com/drud/ddev/pkg/version"
	"github.com/getsentry/raven-go"
	"os"
	"runtime"
	"strconv"
	"strings"
)

// Set the basic always-used tags for Sentry/Raven
func SetRavenBaseTags() {
	if globalconfig.DdevGlobalConfig.InstrumentationOptIn {
		dockerVersion, _ := dockerutil.GetDockerVersion()
		composeVersion, _ := dockerutil.GetDockerComposeVersion()
		isToolbox := util.IsDockerToolbox()

		raven.SetRelease("ddev@" + version.COMMIT)

		tags := map[string]string{
			"OS":                   runtime.GOOS,
			"dockerVersion":        dockerVersion,
			"dockerComposeVersion": composeVersion,
			"dockerToolbox":        strconv.FormatBool(isToolbox),
			"ddevCommand":          strings.Join(os.Args, " "),
		}
		ravenutils.AddRavenTags(tags)
		raven.SetTagsContext(tags)
	}
}

// Set app-specific tags for Sentry/Raven
func (app *DdevApp) SetRavenTags() {
	if globalconfig.DdevGlobalConfig.InstrumentationOptIn {
		describeTags, _ := app.Describe()
		tags := map[string]string{}
		for key, val := range describeTags {
			var tagVal string
			if valString, ok := val.(string); ok {
				tagVal = valString
			} else if valAry, ok := val.([]string); ok {
				tagVal = strings.Join(valAry, " ")
			}
			if tagVal != "" {
				tags[key] = tagVal
			}
		}
		ravenutils.AddRavenTags(tags)
		raven.SetTagsContext(tags)
	}
}
