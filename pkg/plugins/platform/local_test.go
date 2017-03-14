package platform

import (
	"testing"

	"os"

	"github.com/drud/ddev/pkg/version"
	"github.com/drud/drud-go/utils/system"
	"github.com/stretchr/testify/assert"
)

func TestInit(t *testing.T) {
	system.RunCommand("git", []string{"clone", "git@github.com:newmediadenver/drud-d8.git"})

	os.Chdir("drud-d8")

	assert := assert.New(t)

	app := PluginMap["local"]

	opts := AppOptions{
		Name:        "drud-d8",
		WebImage:    version.WebImg,
		WebImageTag: version.WebTag,
		DbImage:     version.DBImg,
		DbImageTag:  version.DBTag,
		SkipYAML:    false,
	}

	app.Init(opts)
	err := app.Start()
	assert.NoError(err)
}
