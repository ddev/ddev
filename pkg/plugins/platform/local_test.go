package platform

import (
	"fmt"
	"path/filepath"
	"sync"

	"testing"
	"time"

	"os"

	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/drud/ddev/pkg/testcommon"
	"github.com/drud/ddev/pkg/util"
	"github.com/drud/drud-go/utils/system"
	"github.com/stretchr/testify/assert"
)

var (
	TestSites = []testcommon.TestSite{
		{
			Name:      "TestMainPkgDrupal8",
			SourceURL: "https://github.com/drud/drupal8/archive/v0.5.0.tar.gz",
			FileURL:   "https://github.com/drud/drupal8/releases/download/v0.5.0/files.tar.gz",
			DBURL:     "https://github.com/drud/drupal8/releases/download/v0.5.0/db.tar.gz",
		},
		{
			Name:      "TestMainPkgWordpress",
			SourceURL: "https://github.com/drud/wordpress/archive/v0.4.0.tar.gz",
			FileURL:   "https://github.com/drud/wordpress/releases/download/v0.4.0/files.tar.gz",
			DBURL:     "https://github.com/drud/wordpress/releases/download/v0.4.0/db.tar.gz",
		},
		{
			Name:      "TestMainPkgDrupalKickstart",
			SourceURL: "https://github.com/drud/drupal-kickstart/archive/v0.4.0.tar.gz",
			FileURL:   "https://github.com/drud/drupal-kickstart/releases/download/v0.4.0/files.tar.gz",
			DBURL:     "https://github.com/drud/drupal-kickstart/releases/download/v0.4.0/db.tar.gz",
		},
	}
)

func TestMain(m *testing.M) {
	for i := range TestSites {
		err := TestSites[i].Prepare()
		if err != nil {
			log.Fatalf("Prepare() failed on TestSite.Prepare(), err=%v", err)
		}
	}

	// Add sites required by tests.
	addSites()

	log.Debugln("Running tests.")
	testRun := m.Run()

	removeSites()

	for i := range TestSites {
		TestSites[i].Cleanup()
	}

	os.Exit(testRun)
}

// addSites performs test setup by adding sites which are used during testing
func addSites() {
	// ensure we have docker network
	client := util.GetDockerClient()
	err := util.EnsureNetwork(client, util.NetName)
	if err != nil {
		log.Fatal(err)
	}

	// We need to ensure the router before we start other sites, so they don't conflict with each other.
	StartDockerRouter()

	// prep docker container for docker util tests
	testcommon.PrepDockerImages(util.GetDockerClient())
	var wg sync.WaitGroup
	wg.Add(len(TestSites))

	for i, site := range TestSites {
		time.Sleep(5000 * time.Millisecond)
		go func(i int, site testcommon.TestSite) {
			defer wg.Done()
			err := TestSites[i].Prepare()
			if err != nil {
				log.Fatalf("Prepare() failed on TestSite.Prepare(), err=%v", err)
			}

			app, err := GetPluginApp("local")
			util.CheckErr(err)

			cleanup := site.Chdir()

			testcommon.ClearDockerEnv()
			err = app.Init(site.Dir)
			util.CheckErr(err)
			err = app.Start()
			util.CheckErr(err)

			err = app.Wait("web")
			util.CheckErr(err)

			// ensure docker-compose.yaml exists inside .ddev site folder
			composeFile := system.FileExists(app.DockerComposeYAMLPath())
			if !composeFile {
				log.Fatalf("no docker-compose file exists at %s\n", app.DockerComposeYAMLPath())
			}

			for _, containerType := range [3]string{"web", "db", "dba"} {
				containerName, err := constructContainerName(containerType, app)
				util.CheckErr(err)
				check, err := testcommon.ContainerCheck(containerName, "running")
				util.CheckErr(err)
				if !check {
					log.Fatalf("Cotainer %s for site %s is not running", containerType, site.Name)
				}
			}

			cleanup()
		}(i, site)
	}

	wg.Wait()
}

// removeSites removes all sites used during testing.
func removeSites() {

	var wg sync.WaitGroup
	wg.Add(len(TestSites))

	// ensure we have docker network
	client := util.GetDockerClient()
	err := util.EnsureNetwork(client, util.NetName)
	if err != nil {
		log.Fatal(err)
	}

	for i, site := range TestSites {
		go func(i int, site testcommon.TestSite) {
			defer wg.Done()

			app, err := GetPluginApp("local")
			util.CheckErr(err)

			cleanup := site.Chdir()

			testcommon.ClearDockerEnv()
			err = app.Init(site.Dir)
			util.CheckErr(err)

			err = app.Down()
			util.CheckErr(err)

			for _, containerType := range [3]string{"web", "db", "dba"} {
				_, err := constructContainerName(containerType, app)
				util.CheckErr(err)
			}

			cleanup()

		}(i, site)

	}

	wg.Wait()

	for i := range TestSites {
		TestSites[i].Cleanup()
	}
}

// TestLocalStart tests the functionality that is called when "ddev start" is executed
func TestLocalStart(t *testing.T) {

	// ensure we have docker network
	client := util.GetDockerClient()
	err := util.EnsureNetwork(client, util.NetName)

	if err != nil {
		log.Fatal(err)
	}
	assert := assert.New(t)
	app, err := GetPluginApp("local")
	assert.NoError(err)

	// try to start a site of same name at different path
	another := TestSites[0]
	err = another.Prepare()
	if err != nil {
		assert.FailNow("Prepare() should have failed on TestSite.Prepare(), err=%v", err)
		return
	}

	err = app.Init(another.Dir)
	assert.Error(err)
	assert.Contains(err.Error(), fmt.Sprintf("container in running state already exists for %s that was created at %s", TestSites[0].Name, TestSites[0].Dir))
	another.Cleanup()

	containers, err := util.GetDockerContainers(true)
	if err != nil {
		log.Fatal(err)
	}

	for _, container := range containers {
		name := util.ContainerName(container)
		testcommon.ContainerCheck(name, "running")
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
		dbPath := filepath.Join(testcommon.CreateTmpDir("local-db"), "db.tar.gz")

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
		filePath := filepath.Join(testcommon.CreateTmpDir("local-files"), "files.tar.gz")

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

// TestLocalExec tests the execution of commands inside a docker container of a site.
func TestLocalExec(t *testing.T) {
	assert := assert.New(t)
	app, err := GetPluginApp("local")
	assert.NoError(err)

	for _, site := range TestSites {
		cleanup := site.Chdir()

		err := app.Init(site.Dir)
		assert.NoError(err)

		stdout := testcommon.CaptureStdOut()
		err = app.Exec("web", true, "pwd")
		assert.NoError(err)
		out := stdout()
		assert.Contains(out, "/var/www/html/docroot")

		stdout = testcommon.CaptureStdOut()
		switch app.GetType() {
		case "drupal7":
			fallthrough
		case "drupal8":
			err := app.Exec("web", true, "drush", "status")
			assert.NoError(err)
		case "wordpress":
			err = app.Exec("web", true, "wp", "--info")
			assert.NoError(err)
		default:
		}
		out = stdout()

		assert.Contains(string(out), "/etc/php/7.0/cli/php.ini")

		cleanup()

	}
}

// TestLocalLogs tests the container log output functionality.
func TestLocalLogs(t *testing.T) {
	assert := assert.New(t)

	app, err := GetPluginApp("local")
	assert.NoError(err)

	for _, site := range TestSites {
		cleanup := site.Chdir()

		err := app.Init(site.Dir)
		assert.NoError(err)

		stdout := testcommon.CaptureStdOut()
		err = app.Logs("web", false, false, "")
		assert.NoError(err)
		out := stdout()
		assert.Contains(out, "Server started")

		stdout = testcommon.CaptureStdOut()
		err = app.Logs("db", false, false, "")
		assert.NoError(err)
		out = stdout()
		assert.Contains(out, "Database initialized")

		stdout = testcommon.CaptureStdOut()
		err = app.Logs("db", false, false, "2")
		assert.NoError(err)
		out = stdout()
		assert.Contains(out, "MySQL init process done. Ready for start up.")
		assert.False(strings.Contains(out, "Database initialized"))

		cleanup()
	}
}

// TestLocalStop tests the functionality that is called when "ddev stop" is executed
func TestLocalStop(t *testing.T) {
	assert := assert.New(t)

	app, err := GetPluginApp("local")
	assert.NoError(err)

	for _, site := range TestSites {
		cleanup := site.Chdir()

		testcommon.ClearDockerEnv()
		err := app.Init(site.Dir)
		assert.NoError(err)

		err = app.Stop()
		assert.NoError(err)

		for _, containerType := range [3]string{"web", "db", "dba"} {
			containerName, err := constructContainerName(containerType, app)
			assert.NoError(err)
			check, err := testcommon.ContainerCheck(containerName, "exited")
			assert.NoError(err)
			assert.True(check, containerType, "container has exited")
		}

		cleanup()
	}
}

// TestLocalRemove tests the functionality that is called when "ddev rm" is executed
func TestLocalRemove(t *testing.T) {
	assert := assert.New(t)

	app, err := GetPluginApp("local")
	assert.NoError(err)

	for _, site := range TestSites {
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

		for _, containerType := range [3]string{"web", "db", "dba"} {
			_, err := constructContainerName(containerType, app)
			assert.Error(err, "Received error on containerName search: ", err)
		}

		cleanup()
	}
}

// TestCleanupWithoutCompose ensures app containers can be properly cleaned up without a docker-compose config file present.
func TestCleanupWithoutCompose(t *testing.T) {
	assert := assert.New(t)
	site := TestSites[0]

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

	for _, containerType := range [3]string{"web", "db", "dba"} {
		_, err := constructContainerName(containerType, app)
		assert.Error(err)
	}

	revertDir()
}

// TestGetappsEmpty ensures that GetApps returns an empty list when no applications are running.
func TestGetAppsEmpty(t *testing.T) {
	assert := assert.New(t)
	apps := GetApps()
	assert.Equal(len(apps["local"]), 0)
}

// TestRouterNotRunning ensures the router is shut down after all sites are stopped.
func TestRouterNotRunning(t *testing.T) {
	assert := assert.New(t)
	containers, err := util.GetDockerContainers(false)
	assert.NoError(err)

	for _, container := range containers {
		assert.NotEqual(util.ContainerName(container), "nginx-proxy", "Found nginx proxy running")
	}
}

// constructContainerName builds a container name given the type (web/db/dba) and the app
func constructContainerName(containerType string, app App) (string, error) {
	container, err := app.FindContainerByType(containerType)
	if err != nil {
		return "", err
	}
	name := util.ContainerName(container)
	return name, nil
}
