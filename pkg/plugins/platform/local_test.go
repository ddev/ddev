package platform

import (
	"fmt"
	"log"
	"path"
	"testing"

	"os"

	"github.com/drud/ddev/pkg/version"
	"github.com/drud/drud-go/utils/dockerutil"
	"github.com/drud/drud-go/utils/system"
	docker "github.com/fsouza/go-dockerclient"
	"github.com/stretchr/testify/assert"
)

var (
	tmp                  = os.TempDir()
	TestSite             = "drupal8"
	TestSiteVer          = "0.1.0"
	TestDir              = path.Join(tmp, TestSite+"-"+TestSiteVer)
	TestDBContainerName  = "local-" + TestSite + "-db"
	TestWebContainerName = "local-" + TestSite + "-web"
)

const netName = "drud_default"

func TestMain(m *testing.M) {
	os.Setenv("DRUD_NONINTERACTIVE", "true")
	PrepareTest()

	fmt.Println("Running tests.")
	os.Exit(m.Run())
}

func PrepareTest() {
	os.Mkdir(TestDir, 0755)
	archive := path.Join(path.Dir(TestDir), "testsite.tar.gz")

	// if !system.FileExists(archive) {
	system.DownloadFile(archive, "https://github.com/drud/"+TestSite+"/archive/v"+TestSiteVer+".tar.gz")
	// }

	system.RunCommand("tar",
		[]string{
			"-xzf",
			archive,
			"-C",
			tmp,
		})
}

func CleanupTest() {
	os.RemoveAll(TestDir)
}

// TestLocalStart tests the functionality that is called when "ddev start" is executed
func TestLocalStart(t *testing.T) {
	err := os.Chdir(TestDir)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("running from %s", TestDir)

	// ensure we have docker network
	client, _ := dockerutil.GetDockerClient()
	err = EnsureNetwork(client, netName)
	if err != nil {
		log.Fatal(err)
	}

	assert := assert.New(t)

	app := PluginMap["local"]

	opts := AppOptions{
		Name:        TestSite,
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
		Name: TestSite,
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
		Name:        TestSite,
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

	CleanupTest()
	os.Unsetenv("DRUD_NONINTERACTIVE")
}
