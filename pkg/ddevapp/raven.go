package ddevapp

import (
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/util"
	"github.com/drud/ddev/pkg/version"
	"github.com/getsentry/raven-go"
	"os"
	"runtime"
	"strconv"
	"strings"
)

func SetRavenBaseTags() {
	dockerVersion, _ := dockerutil.GetDockerVersion()
	composeVersion, _ := dockerutil.GetDockerComposeVersion()
	isToolbox := util.IsDockerToolbox()

	raven.SetRelease("ddev@" +version.COMMIT)

	tags := map[string]string{
		"OS":             runtime.GOOS,
		"dockerVersion":  dockerVersion,
		"composeVersion": composeVersion,
		"dockerToolbox":  strconv.FormatBool(isToolbox),
		"ddevCommand": strings.Join(os.Args, " "),
	}
	raven.SetTagsContext(tags)
}


func (app *DdevApp) SetRavenTags() {
	describeTags, _ := app.Describe()
	tags := map[string]string{}
	for key, val := range describeTags {
		var tagVal string
		if valString, ok := val.(string); ok  {
			tagVal = valString
		} else if valAry, ok := val.([]string); ok {
			tagVal = strings.Join(valAry, " ")
		}
		if (tagVal != "") {
			tags[key] = tagVal
		}
	}
	raven.SetTagsContext(tags)

}