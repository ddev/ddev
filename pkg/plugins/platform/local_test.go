package platform

import (
	"fmt"
	"log"
	"path"
	"testing"

	"os"

	"github.com/drud/ddev/pkg/testcommon"
	"github.com/drud/ddev/pkg/version"
	"github.com/drud/drud-go/utils/dockerutil"
	"github.com/drud/drud-go/utils/system"
	docker "github.com/fsouza/go-dockerclient"
	"github.com/stretchr/testify/assert"
)

var (
	siteName             = "drupal8"
	TestDBContainerName  = "local-" + siteName + "-db"
	TestWebContainerName = "local-" + siteName + "-web"
	TestSite             = []string{"drupal8", "https://github.com/drud/drupal8/archive/v0.1.0.tar.gz", "drupal8-0.1.0"}
	TestDir              = path.Join(os.TempDir(), TestSite[2])
)

const netName = "drud_default"

func TestMain(m *testing.M) {
	testcommon.PrepareTest(TestSite)

	fmt.Println("Running tests.")
	testRun := m.Run()

	testcommon.CleanupTest(TestSite)

	os.Exit(testRun)
}

// TestLocalStart tests the functionality that is called when "ddev start" is executed
func TestLocalStart(t *testing.T) {

	// ensure we have docker network
	client, _ := dockerutil.GetDockerClient()
	err := EnsureNetwork(client, netName)
	if err != nil {
		log.Fatal(err)
	}

	assert := assert.New(t)

	app := PluginMap["local"]

	opts := AppOptions{
		Name:        siteName,
		WebImage:    version.WebImg,
		WebImageTag: version.WebTag,
		DbImage:     version.DBImg,
		DbImageTag:  version.DBTag,
	}

	app.Init(opts)

	err = app.Start()
	assert.NoError(err)

	// ensure docker-compose.yaml exists inside .ddev site folder
	composeFile := system.FileExists(path.Join(TestDir, ".ddev", "docker-compose.yaml"))
	assert.True(composeFile)

	// ensure we have running containers for this site
	containers, _ := client.ListContainers(docker.ListContainersOptions{All: true})

	dbContainer := ""
	webContainer := ""

	for _, container := range containers {
		name := container.Names[0][1:]
		if name == TestDBContainerName && container.State == "running" {
			dbContainer = name
		}
		if name == TestWebContainerName && container.State == "running" {
			webContainer = name
		}
	}

	assert.Equal(TestDBContainerName, dbContainer)
	assert.Equal(TestWebContainerName, webContainer)
}

// TestLocalStop tests the functionality that is called when "ddev stop" is executed
func TestLocalStop(t *testing.T) {
	assert := assert.New(t)

	app := PluginMap["local"]
	opts := AppOptions{
		Name: siteName,
	}
	app.SetOpts(opts)

	err := app.Stop()
	assert.NoError(err)

	// ensure we have stopped containers for this site
	client, _ := dockerutil.GetDockerClient()
	err = EnsureNetwork(client, netName)
	if err != nil {
		log.Fatal(err)
	}
	containers, _ := client.ListContainers(docker.ListContainersOptions{All: true})

	var active bool

	for _, container := range containers {
		name := container.Names[0][1:]
		if name == TestDBContainerName && container.State == "running" {
			active = true
		}
		if name == TestWebContainerName && container.State == "running" {
			active = true
		}
	}

	assert.False(active)
}

// TestLocalRemove tests the functionality that is called when "ddev rm" is executed
func TestLocalRemove(t *testing.T) {
	assert := assert.New(t)

	app := PluginMap["local"]
	opts := AppOptions{
		Name:        siteName,
		WebImage:    version.WebImg,
		WebImageTag: version.WebTag,
		DbImage:     version.DBImg,
		DbImageTag:  version.DBTag,
	}

	app.Init(opts)

	// start the previously stopped containers -
	// stopped/removed have the same state
	err := app.Start()
	assert.NoError(err)

	if err == nil {
		err = app.Down()
		assert.NoError(err)
	}

	// ensure we have stopped containers for this site
	client, _ := dockerutil.GetDockerClient()
	err = EnsureNetwork(client, netName)
	if err != nil {
		log.Fatal(err)
	}
	containers, _ := client.ListContainers(docker.ListContainersOptions{All: true})

	var active bool

	for _, container := range containers {
		name := container.Names[0][1:]
		if name == TestDBContainerName && container.State == "running" {
			active = true
		}
		if name == TestWebContainerName && container.State == "running" {
			active = true
		}
	}

	assert.False(active)
}
