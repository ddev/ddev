package platform

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"os"

	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/testcommon"
	"github.com/drud/ddev/pkg/util"
	"github.com/stretchr/testify/assert"
)

var (
	TestSites = []testcommon.TestSite{
		{
			Name:                          "TestMainPkgDrupal8",
			SourceURL:                     "https://github.com/drud/drupal8/archive/v0.6.0.tar.gz",
			ArchiveInternalExtractionPath: "drupal8-0.6.0/",
			FilesTarballURL:               "https://github.com/drud/drupal8/releases/download/v0.6.0/files.tar.gz",
			FilesZipballURL:               "https://github.com/drud/drupal8/releases/download/v0.6.0/files.zip",
			DBTarURL:                      "https://github.com/drud/drupal8/releases/download/v0.6.0/db.tar.gz",
			DBZipURL:                      "https://github.com/drud/drupal8/releases/download/v0.6.0/db.zip",
			FullSiteTarballURL:            "https://github.com/drud/drupal8/releases/download/v0.6.0/site.tar.gz",
		},
		{
			Name:                          "TestMainPkgWordpress",
			SourceURL:                     "https://github.com/drud/wordpress/archive/v0.4.0.tar.gz",
			ArchiveInternalExtractionPath: "wordpress-0.4.0/",
			FilesTarballURL:               "https://github.com/drud/wordpress/releases/download/v0.4.0/files.tar.gz",
			DBTarURL:                      "https://github.com/drud/wordpress/releases/download/v0.4.0/db.tar.gz",
		},
		{
			Name:                          "TestMainPkgDrupalKickstart",
			SourceURL:                     "https://github.com/drud/drupal-kickstart/archive/v0.4.0.tar.gz",
			ArchiveInternalExtractionPath: "drupal-kickstart-0.4.0/",
			FilesTarballURL:               "https://github.com/drud/drupal-kickstart/releases/download/v0.4.0/files.tar.gz",
			DBTarURL:                      "https://github.com/drud/drupal-kickstart/releases/download/v0.4.0/db.tar.gz",
			FullSiteTarballURL:            "https://github.com/drud/drupal-kickstart/releases/download/v0.4.0/site.tar.gz",
		},
	}
)

func TestMain(m *testing.M) {

	if len(GetApps()) > 0 {
		log.Fatalf("Local plugin tests require no sites running. You have %v site(s) running.", len(GetApps()))
	}

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

// TestLocalSetup reduces the TestSite list on shorter test runs.
func TestLocalSetup(t *testing.T) {
	// Allow tests to run in "short" mode, which will only test a single site. This keeps test runtimes low.
	// We would much prefer to do this in TestMain, but the Short() flag is not yet available at that point.
	if testing.Short() {
		TestSites = []testcommon.TestSite{TestSites[0]}
	}
}

// TestLocalStart tests the functionality that is called when "ddev start" is executed
func TestLocalStart(t *testing.T) {

	// ensure we have docker network
	client := dockerutil.GetDockerClient()
	err := dockerutil.EnsureNetwork(client, dockerutil.NetName)
	if err != nil {
		log.Fatal(err)
	}

	assert := assert.New(t)
	app, err := GetPluginApp("local")
	assert.NoError(err)

	for _, site := range TestSites {
		cleanup := site.Chdir()
		runTime := testcommon.TimeTrack(time.Now(), fmt.Sprintf("%s LocalStart", site.Name))

		testcommon.ClearDockerEnv()
		err = app.Init(site.Dir)
		assert.NoError(err)

		err = app.Start()
		assert.NoError(err)

		// ensure docker-compose.yaml exists inside .ddev site folder
		composeFile := fileutil.FileExists(app.DockerComposeYAMLPath())
		assert.True(composeFile)

		for _, containerType := range [3]string{"web", "db", "dba"} {
			containerName, err := constructContainerName(containerType, app)
			assert.NoError(err)
			check, err := testcommon.ContainerCheck(containerName, "running")
			assert.NoError(err)
			assert.True(check, containerType, "container is running")
		}

		runTime()
		cleanup()
	}

	// try to start a site of same name at different path
	another := TestSites[0]
	err = another.Prepare()
	if err != nil {
		assert.FailNow("TestLocalStart: Prepare() failed on another.Prepare(), err=%v", err)
		return
	}

	err = app.Init(another.Dir)
	assert.Error(err)
	assert.Contains(err.Error(), fmt.Sprintf("container in running state already exists for %s that was created at %s", TestSites[0].Name, TestSites[0].Dir))
	another.Cleanup()
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
	testDir, _ := os.Getwd()

	for _, site := range TestSites {
		cleanup := site.Chdir()
		runTime := testcommon.TimeTrack(time.Now(), fmt.Sprintf("%s LocalImportDB", site.Name))

		testcommon.ClearDockerEnv()
		err = app.Init(site.Dir)
		assert.NoError(err)

		// Test simple db loads.
		for _, file := range []string{"users.sql", "users.sql.gz", "users.sql.tar", "users.sql.tar.gz", "users.sql.tgz", "users.sql.zip"} {
			path := filepath.Join(testDir, "testdata", file)
			err = app.ImportDB(path, "")
			assert.NoError(err, "Failed to app.ImportDB path: %s err: %v", path, err)
		}

		if site.DBTarURL != "" {
			err = app.Exec("db", true, "mysql", "-e", "DROP DATABASE db;")
			assert.NoError(err)
			err = app.Exec("db", true, "mysql", "information_schema", "-e", "CREATE DATABASE db;")
			assert.NoError(err)

			dbPath := filepath.Join(testcommon.CreateTmpDir("local-db"), "db.tar.gz")
			err := util.DownloadFile(dbPath, site.DBTarURL)
			assert.NoError(err)
			err = app.ImportDB(dbPath, "")
			assert.NoError(err)

			stdout := testcommon.CaptureStdOut()
			err = app.Exec("db", true, "mysql", "-e", "SHOW TABLES;")
			assert.NoError(err)
			out := stdout()

			assert.Contains(string(out), "Tables_in_db")
			assert.False(strings.Contains(string(out), "Empty set"))

			err = os.Remove(dbPath)
			assert.NoError(err)
		}

		if site.DBZipURL != "" {
			err = app.Exec("db", true, "mysql", "-e", "DROP DATABASE db;")
			assert.NoError(err)
			err = app.Exec("db", true, "mysql", "information_schema", "-e", "CREATE DATABASE db;")
			assert.NoError(err)

			dbZipPath := filepath.Join(testcommon.CreateTmpDir("local-db-zip"), "db.zip")
			err = util.DownloadFile(dbZipPath, site.DBZipURL)
			assert.NoError(err)
			err = app.ImportDB(dbZipPath, "")
			assert.NoError(err)

			stdout := testcommon.CaptureStdOut()
			err = app.Exec("db", true, "mysql", "-e", "SHOW TABLES;")
			assert.NoError(err)
			out := stdout()

			assert.Contains(string(out), "Tables_in_db")
			assert.False(strings.Contains(string(out), "Empty set"))

			err = os.Remove(dbZipPath)
			assert.NoError(err)
		}

		if site.FullSiteTarballURL != "" {
			err = app.Exec("db", true, "mysql", "-e", "DROP DATABASE db;")
			assert.NoError(err)
			err = app.Exec("db", true, "mysql", "information_schema", "-e", "CREATE DATABASE db;")
			assert.NoError(err)

			siteTarPath := filepath.Join(testcommon.CreateTmpDir("local-site-tar"), "site.tar.gz")
			err = util.DownloadFile(siteTarPath, site.FullSiteTarballURL)
			assert.NoError(err)
			err = app.ImportDB(siteTarPath, "data.sql")
			assert.NoError(err)
			err = os.Remove(siteTarPath)
			assert.NoError(err)
		}

		runTime()
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
		runTime := testcommon.TimeTrack(time.Now(), fmt.Sprintf("%s LocalImportFiles", site.Name))

		testcommon.ClearDockerEnv()
		err = app.Init(site.Dir)
		assert.NoError(err)

		if site.FilesTarballURL != "" {
			filePath := filepath.Join(testcommon.CreateTmpDir("local-tarball-files"), "files.tar.gz")
			err := util.DownloadFile(filePath, site.FilesTarballURL)
			assert.NoError(err)
			err = app.ImportFiles(filePath, "")
			assert.NoError(err)
			err = os.Remove(filePath)
			assert.NoError(err)
		}

		if site.FilesZipballURL != "" {
			filePath := filepath.Join(testcommon.CreateTmpDir("local-zipball-files"), "files.zip")
			err := util.DownloadFile(filePath, site.FilesZipballURL)
			assert.NoError(err)
			err = app.ImportFiles(filePath, "")
			assert.NoError(err)
			err = os.Remove(filePath)
			assert.NoError(err)
		}

		if site.FullSiteTarballURL != "" {
			siteTarPath := filepath.Join(testcommon.CreateTmpDir("local-site-tar"), "site.tar.gz")
			err = util.DownloadFile(siteTarPath, site.FullSiteTarballURL)
			assert.NoError(err)
			err = app.ImportFiles(siteTarPath, "docroot/sites/default/files")
			assert.NoError(err)
			err = os.Remove(siteTarPath)
			assert.NoError(err)
		}

		runTime()
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
		runTime := testcommon.TimeTrack(time.Now(), fmt.Sprintf("%s LocalExec", site.Name))

		err := app.Init(site.Dir)
		assert.NoError(err)

		stdout := testcommon.CaptureStdOut()
		err = app.Exec("web", true, "pwd")
		assert.NoError(err)
		out := stdout()
		assert.Contains(out, "/var/www/html")

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

		runTime()
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
		runTime := testcommon.TimeTrack(time.Now(), fmt.Sprintf("%s LocalLogs", site.Name))

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

		runTime()
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
		runTime := testcommon.TimeTrack(time.Now(), fmt.Sprintf("%s LocalStop", site.Name))

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

		runTime()
		cleanup()
	}
}

// TestDescribeStopped tests that the describe command works properly on a stopped site.
func TestDescribeStopped(t *testing.T) {
	assert := assert.New(t)
	app, err := GetPluginApp("local")
	assert.NoError(err)

	for _, site := range TestSites {
		cleanup := site.Chdir()

		testcommon.ClearDockerEnv()
		err := app.Init(site.Dir)
		assert.NoError(err)

		out, err := app.Describe()
		assert.NoError(err)

		assert.Contains(out, SiteStopped, "Output did not include the word stopped when describing a stopped site.")
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

		runTime := testcommon.TimeTrack(time.Now(), fmt.Sprintf("%s LocalRemove", site.Name))
		err = app.Down()
		assert.NoError(err)

		for _, containerType := range [3]string{"web", "db", "dba"} {
			_, err := constructContainerName(containerType, app)
			assert.Error(err, "Received error on containerName search: ", err)
		}

		runTime()
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
	containers, err := dockerutil.GetDockerContainers(false)
	assert.NoError(err)

	for _, container := range containers {
		assert.NotEqual(dockerutil.ContainerName(container), "ddev-router", "Failed to find ddev-router container running")
	}
}

// constructContainerName builds a container name given the type (web/db/dba) and the app
func constructContainerName(containerType string, app App) (string, error) {
	container, err := app.FindContainerByType(containerType)
	if err != nil {
		return "", err
	}
	name := dockerutil.ContainerName(container)
	return name, nil
}
