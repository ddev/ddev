package platform

import (
	"errors"
	"testing"

	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/drud/ddev/pkg/testcommon"
	"github.com/drud/drud-go/utils/dockerutil"
	"github.com/drud/drud-go/utils/system"
	docker "github.com/fsouza/go-dockerclient"
	"github.com/stretchr/testify/assert"
)

var (
	siteName             = "drupal8"
	TestDBContainerName  = "local-" + siteName + "-db"
	TestWebContainerName = "local-" + siteName + "-web"
	TestSite             = testcommon.TestSite{
		Name:        "drupal8",
		DownloadURL: "https://github.com/drud/drupal8/archive/v0.2.2.tar.gz",
	}
)

const netName = "ddev_default"

func TestMain(m *testing.M) {
	err := TestSite.Prepare()
	if err != nil {
		log.Fatalf("Prepare() failed on TestSite.Prepare(), err=%v", err)
	}
	defer TestSite.Chdir()()

	log.Debugln("Running tests.")
	testRun := m.Run()

	TestSite.Cleanup()

	os.Exit(testRun)
}

// ContainerCheck determines if a given container name exists and matches a given state
func ContainerCheck(checkName string, checkState string) (bool, error) {
	// ensure we have docker network
	client, _ := dockerutil.GetDockerClient()
	err := EnsureNetwork(client, netName)
	if err != nil {
		log.Fatal(err)
	}

	containers, err := client.ListContainers(docker.ListContainersOptions{All: true})
	if err != nil {
		log.Fatal(err)
	}

	for _, container := range containers {
		name := container.Names[0][1:]
		if name == checkName {
			if container.State == checkState {
				return true, nil
			}
			return false, errors.New("container " + name + " returned " + container.State)
		}
	}

	return false, errors.New("unable to find container " + checkName)
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
	app.Init(TestSite.Dir)

	err = app.Start()
	assert.NoError(err)

	_, err = app.Wait()
	assert.NoError(err)

	// ensure docker-compose.yaml exists inside .ddev site folder
	composeFile := system.FileExists(app.DockerComposeYAMLPath())
	assert.True(composeFile)

	check, err := ContainerCheck(TestWebContainerName, "running")
	assert.NoError(err)
	assert.True(check)

	check, err = ContainerCheck(TestDBContainerName, "running")
	assert.NoError(err)
	assert.True(check)
}

// TestLocalStop tests the functionality that is called when "ddev stop" is executed
func TestLocalStop(t *testing.T) {
	assert := assert.New(t)

	app := PluginMap["local"]
	app.Init(TestSite.Dir)

	err := app.Stop()
	assert.NoError(err)

	check, err := ContainerCheck(TestWebContainerName, "exited")
	assert.NoError(err)
	assert.True(check)

	check, err = ContainerCheck(TestDBContainerName, "exited")
	assert.NoError(err)
	assert.True(check)
}

// TestLocalRemove tests the functionality that is called when "ddev rm" is executed
func TestLocalRemove(t *testing.T) {
	assert := assert.New(t)

	app := PluginMap["local"]

	app.Init(TestSite.Dir)

	// start the previously stopped containers -
	// stopped/removed have the same state
	err := app.Start()
	assert.NoError(err)

	_, err = app.Wait()
	assert.NoError(err)

	if err == nil {
		err = app.Down()
		assert.NoError(err)
	}

	check, err := ContainerCheck(TestWebContainerName, "running")
	assert.Error(err)
	assert.False(check)

	check, err = ContainerCheck(TestDBContainerName, "running")
	assert.Error(err)
	assert.False(check)
}
