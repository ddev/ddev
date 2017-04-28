package platform

import (
	"errors"
	"fmt"
	"path"
	"testing"

	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/drud/ddev/pkg/testcommon"
	"github.com/drud/ddev/pkg/util"
	"github.com/drud/drud-go/utils/dockerutil"
	"github.com/drud/drud-go/utils/system"
	"github.com/stretchr/testify/assert"
)

var (
	localDBContainerName  = "local-%s-db"
	localWebContainerName = "local-%s-web"
	TestSites             = []testcommon.TestSite{
		{
			Name:      "drupal8",
			SourceURL: "https://github.com/drud/drupal8/archive/v0.4.0.tar.gz",
			FileURL:   "https://github.com/drud/drupal8/releases/download/v0.4.0/files.tar.gz",
			DBURL:     "https://github.com/drud/drupal8/releases/download/v0.4.0/db.tar.gz",
		},
		{
			Name:      "wordpress",
			SourceURL: "https://github.com/drud/wordpress/archive/v0.3.0.tar.gz",
			FileURL:   "https://github.com/drud/wordpress/releases/download/v0.3.0/files.tar.gz",
			DBURL:     "https://github.com/drud/wordpress/releases/download/v0.3.0/db.tar.gz",
		},
		{
			Name:      "kickstart",
			SourceURL: "https://github.com/drud/drupal-kickstart/archive/v0.3.0.tar.gz",
			FileURL:   "https://github.com/drud/drupal-kickstart/releases/download/v0.3.0/files.tar.gz",
			DBURL:     "https://github.com/drud/drupal-kickstart/releases/download/v0.3.0/db.tar.gz",
		},
	}
)

const netName = "ddev_default"

func TestMain(m *testing.M) {
	for i := range TestSites {
		err := TestSites[i].Prepare()
		if err != nil {
			log.Fatalf("Prepare() failed on TestSite.Prepare(), err=%v", err)
		}
	}

	log.Debugln("Running tests.")
	testRun := m.Run()

	for i := range TestSites {
		TestSites[i].Cleanup()
	}

	os.Exit(testRun)
}

// ContainerCheck determines if a given container name exists and matches a given state
func ContainerCheck(checkName string, checkState string) (bool, error) {
	// ensure we have docker network
	client, _ := dockerutil.GetDockerClient()
	err := util.EnsureNetwork(client, netName)
	if err != nil {
		log.Fatal(err)
	}

	containers, err := util.GetDockerContainers(true)
	if err != nil {
		log.Fatal(err)
	}

	for _, container := range containers {
		name := util.ContainerName(container)
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
	err := util.EnsureNetwork(client, netName)
	if err != nil {
		log.Fatal(err)
	}

	assert := assert.New(t)
	app, err := GetPluginApp("local")
	assert.NoError(err)

	for _, site := range TestSites {
		webContainer := fmt.Sprintf(localWebContainerName, site.Name)
		dbContainer := fmt.Sprintf(localDBContainerName, site.Name)
		cleanup := site.Chdir()

		testcommon.ClearDockerEnv()
		err = app.Init(site.Dir)
		assert.NoError(err)

		err = app.Start()
		assert.NoError(err)

		err = app.Wait("web")
		assert.NoError(err)

		// ensure docker-compose.yaml exists inside .ddev site folder
		composeFile := system.FileExists(app.DockerComposeYAMLPath())
		assert.True(composeFile)

		check, err := ContainerCheck(webContainer, "running")
		assert.NoError(err)
		assert.True(check)

		check, err = ContainerCheck(dbContainer, "running")
		assert.NoError(err)
		assert.True(check)

		cleanup()
	}
}

// TestGetApps tests the GetApps function to ensure it accurately returns a list of running applications.
func TestGetApps(t *testing.T) {
	assert := assert.New(t)
	apps := GetApps()
	assert.Equal(len(apps["local"]), len(TestSites))

	for _, site := range TestSites {
		var found bool
		for _, siteInList := range apps["local"] {
			if site.Name == siteInList.GetName() {
				found = true
				break
			}
		}
		assert.True(found, "Found site %s in list", site.Name)
	}
}

// TestLocalImportDB tests the functionality that is called when "ddev import-db" is executed
func TestLocalImportDB(t *testing.T) {
	assert := assert.New(t)
	app, err := GetPluginApp("local")
	assert.NoError(err)

	for _, site := range TestSites {
		cleanup := site.Chdir()
		dbPath := path.Join(testcommon.CreateTmpDir("local-db"), "db.tar.gz")

		err := system.DownloadFile(dbPath, site.DBURL)
		assert.NoError(err)

		testcommon.ClearDockerEnv()
		err = app.Init(site.Dir)
		assert.NoError(err)

		err = app.ImportDB(dbPath)
		assert.NoError(err)

		err = os.Remove(dbPath)
		assert.NoError(err)

		cleanup()
	}
}

// TestLocalImportFiles tests the functionality that is called when "ddev import-files" is executed
func TestLocalImportFiles(t *testing.T) {
	assert := assert.New(t)
	app, err := GetPluginApp("local")
	assert.NoError(err)

	for _, site := range TestSites {
		cleanup := site.Chdir()
		filePath := path.Join(testcommon.CreateTmpDir("local-files"), "files.tar.gz")

		err := system.DownloadFile(filePath, site.FileURL)
		assert.NoError(err)

		testcommon.ClearDockerEnv()
		err = app.Init(site.Dir)
		assert.NoError(err)

		err = app.ImportFiles(filePath)
		assert.NoError(err)

		err = os.Remove(filePath)
		assert.NoError(err)

		cleanup()
	}
}

// TestLocalStop tests the functionality that is called when "ddev stop" is executed
func TestLocalStop(t *testing.T) {
	assert := assert.New(t)

	app, err := GetPluginApp("local")
	assert.NoError(err)

	for _, site := range TestSites {
		webContainer := fmt.Sprintf(localWebContainerName, site.Name)
		dbContainer := fmt.Sprintf(localDBContainerName, site.Name)
		cleanup := site.Chdir()

		testcommon.ClearDockerEnv()
		err := app.Init(site.Dir)
		assert.NoError(err)

		err = app.Stop()
		assert.NoError(err)

		check, err := ContainerCheck(webContainer, "exited")
		assert.NoError(err)
		assert.True(check)

		check, err = ContainerCheck(dbContainer, "exited")
		assert.NoError(err)
		assert.True(check)

		cleanup()
	}
}

// TestLocalRemove tests the functionality that is called when "ddev rm" is executed
func TestLocalRemove(t *testing.T) {
	assert := assert.New(t)

	app, err := GetPluginApp("local")
	assert.NoError(err)

	for _, site := range TestSites {
		webContainer := fmt.Sprintf(localWebContainerName, site.Name)
		dbContainer := fmt.Sprintf(localDBContainerName, site.Name)
		cleanup := site.Chdir()

		testcommon.ClearDockerEnv()
		err := app.Init(site.Dir)
		assert.NoError(err)

		// start the previously stopped containers -
		// stopped/removed have the same state
		err = app.Start()
		assert.NoError(err)

		err = app.Wait("web")
		assert.NoError(err)

		if err == nil {
			err = app.Down()
			assert.NoError(err)
		}

		check, err := ContainerCheck(webContainer, "running")
		assert.Error(err)
		assert.False(check)

		check, err = ContainerCheck(dbContainer, "running")
		assert.Error(err)
		assert.False(check)

		cleanup()
	}
}

// TestCleanupWithoutCompose
func TestCleanupWithoutCompose(t *testing.T) {
	assert := assert.New(t)
	site := TestSites[0]

	webContainer := fmt.Sprintf(localWebContainerName, site.Name)
	dbContainer := fmt.Sprintf(localDBContainerName, site.Name)
	revertDir := site.Chdir()
	app, err := GetPluginApp("local")
	assert.NoError(err)

	testcommon.ClearDockerEnv()
	err = app.Init(site.Dir)
	assert.NoError(err)

	// Start a site so we have something to cleanup
	err = app.Start()
	assert.NoError(err)

	err = app.Wait("web")
	assert.NoError(err)

	// Call the Cleanup command()
	err = Cleanup(app)
	assert.NoError(err)

	check, err := ContainerCheck(webContainer, "running")
	assert.Error(err)
	assert.False(check)

	check, err = ContainerCheck(dbContainer, "running")
	assert.Error(err)
	assert.False(check)

	revertDir()
}

// TestGetappsEmpty ensures that GetApps returns an empty list when no applications are running.
func TestGetAppsEmpty(t *testing.T) {
	assert := assert.New(t)
	apps := GetApps()
	assert.Equal(len(apps["local"]), 0)
}
