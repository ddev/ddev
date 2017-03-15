package platform

import (
	"path"
	"testing"

	"os"

	"github.com/drud/ddev/pkg/version"
	"github.com/drud/drud-go/utils/dockerutil"
	"github.com/drud/drud-go/utils/system"
	docker "github.com/fsouza/go-dockerclient"
	"github.com/stretchr/testify/assert"
)

// TestStart tests the functionality that is called when "ddev start" is executed
func TestStart(t *testing.T) {
	assert := assert.New(t)
	testSite := "drupal8"
	testDir := path.Join(os.TempDir(), testSite)

	os.Setenv("DRUD_NONINTERACTIVE", "true")

	system.RunCommand("git", []string{"clone", "--depth", "1", "https://github.com/drud/drupal8.git", testDir})
	os.Chdir(testDir)

	app := PluginMap["local"]

	opts := AppOptions{
		Name:        testSite,
		WebImage:    version.WebImg,
		WebImageTag: version.WebTag,
		DbImage:     version.DBImg,
		DbImageTag:  version.DBTag,
	}

	app.Init(opts)

	err := app.Start()
	assert.NoError(err)

	// ensure docker-compose.yaml exists inside .ddev site folder
	composeFile := system.FileExists(path.Join(testDir, ".ddev", "docker-compose.yaml"))
	assert.True(composeFile)

	// ensure we have containers for this site
	client, _ := dockerutil.GetDockerClient()
	containers, _ := client.ListContainers(docker.ListContainersOptions{All: true})

	dbContainerName := "local-" + testSite + "-db"
	webContainerName := "local-" + testSite + "-web"
	dbContainer := ""
	webContainer := ""

	for _, container := range containers {
		name := container.Names[0][1:]
		if name == dbContainerName {
			dbContainer = name
		}
		if name == webContainerName {
			webContainer = name
		}
	}

	assert.Equal(dbContainerName, dbContainer)
	assert.Equal(webContainerName, webContainer)
}
