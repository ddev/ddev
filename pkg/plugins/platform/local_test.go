package platform_test

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"os"

	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/plugins/platform"
	"github.com/drud/ddev/pkg/testcommon"
	"github.com/drud/ddev/pkg/util"
	docker "github.com/fsouza/go-dockerclient"
	"github.com/stretchr/testify/assert"
)

var (
	TestSites = []testcommon.TestSite{
		{
			Name:                          "TestMainPkgWordpress",
			SourceURL:                     "https://github.com/drud/wordpress/archive/v0.4.0.tar.gz",
			ArchiveInternalExtractionPath: "wordpress-0.4.0/",
			FilesTarballURL:               "https://github.com/drud/wordpress/releases/download/v0.4.0/files.tar.gz",
			DBTarURL:                      "https://github.com/drud/wordpress/releases/download/v0.4.0/db.tar.gz",
			DocrootBase:                   "htdocs",
		},
		{
			Name:                          "TestMainPkgDrupal8",
			SourceURL:                     "https://github.com/drud/drupal8/archive/v0.6.0.tar.gz",
			ArchiveInternalExtractionPath: "drupal8-0.6.0/",
			FilesTarballURL:               "https://github.com/drud/drupal8/releases/download/v0.6.0/files.tar.gz",
			FilesZipballURL:               "https://github.com/drud/drupal8/releases/download/v0.6.0/files.zip",
			DBTarURL:                      "https://github.com/drud/drupal8/releases/download/v0.6.0/db.tar.gz",
			DBZipURL:                      "https://github.com/drud/drupal8/releases/download/v0.6.0/db.zip",
			FullSiteTarballURL:            "https://github.com/drud/drupal8/releases/download/v0.6.0/site.tar.gz",
			DocrootBase:                   "docroot",
		},
		{
			Name:                          "TestMainPkgDrupalKickstart",
			SourceURL:                     "https://github.com/drud/drupal-kickstart/archive/v0.4.0.tar.gz",
			ArchiveInternalExtractionPath: "drupal-kickstart-0.4.0/",
			FilesTarballURL:               "https://github.com/drud/drupal-kickstart/releases/download/v0.4.0/files.tar.gz",
			DBTarURL:                      "https://github.com/drud/drupal-kickstart/releases/download/v0.4.0/db.tar.gz",
			FullSiteTarballURL:            "https://github.com/drud/drupal-kickstart/releases/download/v0.4.0/site.tar.gz",
			DocrootBase:                   "docroot",
		},
	}
)

func TestMain(m *testing.M) {
	// ensure we have docker network
	client := dockerutil.GetDockerClient()
	err := dockerutil.EnsureNetwork(client, dockerutil.NetName)
	if err != nil {
		log.Fatal(err)
	}

	if len(platform.GetApps()) > 0 {
		log.Fatalf("Local plugin tests require no sites running. You have %v site(s) running.", len(platform.GetApps()))
	}

	if os.Getenv("GOTEST_SHORT") != "" {
		TestSites = []testcommon.TestSite{TestSites[0]}
	}

	for i := range TestSites {
		err := TestSites[i].Prepare()
		if err != nil {
			log.Fatalf("Prepare() failed on TestSite.Prepare() site=%s, err=%v", TestSites[i].Name, err)
		}

		switchDir := TestSites[i].Chdir()
		runTime := testcommon.TimeTrack(time.Now(), fmt.Sprintf("%s Start", TestSites[i].Name))

		testcommon.ClearDockerEnv()

		app, err := platform.GetPluginApp("local")
		if err != nil {
			log.Fatalf("TestMain startup: Platform.GetPluginApp(local) failed for site %s, err=%s", TestSites[i].Name, err)
		}

		err = app.Init(TestSites[i].Dir)
		if err != nil {
			log.Fatalf("TestMain startup: app.Init() failed on site %s in dir %s, err=%v", TestSites[i].Name, TestSites[i].Dir, err)
		}

		err = app.Start()
		if err != nil {
			log.Fatalf("TestMain startup: app.Start() failed on site %s, err=%v", TestSites[i].Name, err)
		}

		runTime()
		switchDir()
	}

	log.Debugln("Running tests.")
	testRun := m.Run()

	for i, site := range TestSites {
		runTime := testcommon.TimeTrack(time.Now(), fmt.Sprintf("%s Remove", site.Name))

		testcommon.ClearDockerEnv()

		app, err := platform.GetPluginApp("local")
		if err != nil {
			log.Fatalf("TestMain shutdown: Platform.GetPluginApp(local) failed for site %s, err=%s", TestSites[i].Name, err)
		}

		err = app.Init(site.Dir)
		if err != nil {
			log.Fatalf("TestMain shutdown: app.Init() failed on site %s in dir %s, err=%v", TestSites[i].Name, TestSites[i].Dir, err)
		}

		err = app.Down(true)
		if err != nil {
			log.Fatalf("TestMain startup: app.Down() failed on site %s, err=%v", TestSites[i].Name, err)
		}

		runTime()
		site.Cleanup()
	}

	os.Exit(testRun)
}

// TestLocalStart tests the functionality that is called when "ddev start" is executed
func TestLocalStart(t *testing.T) {
	assert := assert.New(t)
	app, err := platform.GetPluginApp("local")
	assert.NoError(err)

	for _, site := range TestSites {
		switchDir := site.Chdir()
		runTime := testcommon.TimeTrack(time.Now(), fmt.Sprintf("%s LocalStart", site.Name))

		err = app.Init(site.Dir)
		assert.NoError(err)

		// ensure docker-compose.yaml exists inside .ddev site folder
		composeFile := fileutil.FileExists(app.DockerComposeYAMLPath())
		assert.True(composeFile)

		for _, containerType := range [3]string{"web", "db", "dba"} {
			containerName, err := constructContainerName(containerType, app)
			assert.NoError(err)
			check, err := testcommon.ContainerCheck(containerName, "running")
			assert.NoError(err)
			assert.True(check, "Container check on %s failed", containerType)
		}

		runTime()
		switchDir()
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
	testcommon.CleanupDir(another.Dir)
}

// TestGetApps tests the GetApps function to ensure it accurately returns a list of running applications.
func TestGetApps(t *testing.T) {
	assert := assert.New(t)
	apps := platform.GetApps()
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
	app, err := platform.GetPluginApp("local")
	assert.NoError(err)
	testDir, _ := os.Getwd()

	for _, site := range TestSites {
		switchDir := site.Chdir()
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
			_, cachedArchive, err := testcommon.GetCachedArchive(site.Name, site.Name+"_siteTarArchive", "", site.DBTarURL)
			assert.NoError(err)
			err = app.ImportDB(cachedArchive, "")
			assert.NoError(err)

			stdout := testcommon.CaptureStdOut()
			err = app.Exec("db", true, "mysql", "-e", "SHOW TABLES;")
			assert.NoError(err)
			out := stdout()

			assert.Contains(string(out), "Tables_in_db")
			assert.False(strings.Contains(string(out), "Empty set"))

			assert.NoError(err)
		}

		if site.DBZipURL != "" {
			_, cachedArchive, err := testcommon.GetCachedArchive(site.Name, site.Name+"_siteZipArchive", "", site.DBZipURL)

			assert.NoError(err)
			err = app.ImportDB(cachedArchive, "")
			assert.NoError(err)

			stdout := testcommon.CaptureStdOut()
			err = app.Exec("db", true, "mysql", "-e", "SHOW TABLES;")
			assert.NoError(err)
			out := stdout()

			assert.Contains(string(out), "Tables_in_db")
			assert.False(strings.Contains(string(out), "Empty set"))
		}

		if site.FullSiteTarballURL != "" {
			_, cachedArchive, err := testcommon.GetCachedArchive(site.Name, site.Name+"_FullSiteTarballURL", "", site.FullSiteTarballURL)
			assert.NoError(err)

			err = app.ImportDB(cachedArchive, "data.sql")
			assert.NoError(err, "Failed to find data.sql at root of tarball %s", cachedArchive)
		}

		runTime()
		switchDir()
	}
}

// TestLocalImportFiles tests the functionality that is called when "ddev import-files" is executed
func TestLocalImportFiles(t *testing.T) {
	assert := assert.New(t)
	app, err := platform.GetPluginApp("local")
	assert.NoError(err)

	for _, site := range TestSites {
		switchDir := site.Chdir()
		runTime := testcommon.TimeTrack(time.Now(), fmt.Sprintf("%s LocalImportFiles", site.Name))

		testcommon.ClearDockerEnv()
		err = app.Init(site.Dir)
		assert.NoError(err)

		if site.FilesTarballURL != "" {
			_, tarballPath, err := testcommon.GetCachedArchive(site.Name, "local-tarballs-files", "", site.FilesTarballURL)
			assert.NoError(err)
			err = app.ImportFiles(tarballPath, "")
			assert.NoError(err)
		}

		if site.FilesZipballURL != "" {
			_, zipballPath, err := testcommon.GetCachedArchive(site.Name, "local-zipballs-files", "", site.FilesZipballURL)
			assert.NoError(err)
			err = app.ImportFiles(zipballPath, "")
			assert.NoError(err)
		}

		if site.FullSiteTarballURL != "" {
			_, siteTarPath, err := testcommon.GetCachedArchive(site.Name, "local-site-tar", "", site.FullSiteTarballURL)
			assert.NoError(err)
			err = app.ImportFiles(siteTarPath, "docroot/sites/default/files")
			assert.NoError(err)
		}

		runTime()
		switchDir()
	}
}

// TestLocalExec tests the execution of commands inside a docker container of a site.
func TestLocalExec(t *testing.T) {
	assert := assert.New(t)
	app, err := platform.GetPluginApp("local")
	assert.NoError(err)

	for _, site := range TestSites {
		switchDir := site.Chdir()
		runTime := testcommon.TimeTrack(time.Now(), fmt.Sprintf("%s LocalExec", site.Name))

		err := app.Init(site.Dir)
		assert.NoError(err)

		stdout := testcommon.CaptureStdOut()
		err = app.Exec("web", true, "pwd")
		assert.NoError(err)
		out := stdout()
		assert.Contains(out, "/var/www/html")

		err = app.Exec("db", true, "mysql", "-e", "DROP DATABASE db;")
		assert.NoError(err)
		err = app.Exec("db", true, "mysql", "information_schema", "-e", "CREATE DATABASE db;")
		assert.NoError(err)

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
		switchDir()

	}
}

// TestLocalLogs tests the container log output functionality.
func TestLocalLogs(t *testing.T) {
	assert := assert.New(t)

	app, err := platform.GetPluginApp("local")
	assert.NoError(err)

	for _, site := range TestSites {
		switchDir := site.Chdir()
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

		runTime()
		switchDir()
	}
}

// TestProcessHooks tests execution of commands defined in config
func TestProcessHooks(t *testing.T) {
	assert := assert.New(t)

	for _, site := range TestSites {
		cleanup := site.Chdir()
		runTime := testcommon.TimeTrack(time.Now(), fmt.Sprintf("%s ProcessHooks", site.Name))

		testcommon.ClearDockerEnv()
		conf, err := ddevapp.NewConfig(site.Dir, ddevapp.DDevDefaultPlatform)
		assert.NoError(err)

		conf.Commands = map[string][]ddevapp.Command{
			"hook-test": []ddevapp.Command{
				ddevapp.Command{
					Exec: "pwd",
				},
				ddevapp.Command{
					ExecHost: "pwd",
				},
			},
		}

		l := &platform.LocalApp{
			AppConfig: conf,
		}

		stdout := testcommon.CaptureStdOut()
		err = l.ProcessHooks("hook-test")
		assert.NoError(err)
		out := stdout()

		assert.Contains(out, "--- Running exec command: pwd ---")
		assert.Contains(out, "--- Running host command: pwd ---")

		runTime()
		cleanup()

	}
}

// TestLocalStop tests the functionality that is called when "ddev stop" is executed
func TestLocalStop(t *testing.T) {
	assert := assert.New(t)

	app, err := platform.GetPluginApp("local")
	assert.NoError(err)

	for _, site := range TestSites {
		switchDir := site.Chdir()
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
		switchDir()
	}
}

// TestDescribeStopped tests that the describe command works properly on a stopped site.
func TestDescribeStopped(t *testing.T) {
	assert := assert.New(t)
	app, err := platform.GetPluginApp("local")
	assert.NoError(err)

	for _, site := range TestSites {
		switchDir := site.Chdir()

		testcommon.ClearDockerEnv()
		err := app.Init(site.Dir)
		assert.NoError(err)

		err = app.Stop()
		assert.NoError(err)

		out, err := app.Describe()
		assert.NoError(err)

		assert.Contains(out, platform.SiteStopped, "Output did not include the word stopped when describing a stopped site.")
		switchDir()
	}
}

// TestCleanupWithoutCompose ensures app containers can be properly cleaned up without a docker-compose config file present.
func TestCleanupWithoutCompose(t *testing.T) {
	assert := assert.New(t)
	site := TestSites[0]

	revertDir := site.Chdir()
	app, err := platform.GetPluginApp("local")
	assert.NoError(err)

	testcommon.ClearDockerEnv()
	err = app.Init(site.Dir)
	assert.NoError(err)

	// Ensure we have a site started so we have something to cleanup
	err = app.Start()
	assert.NoError(err)

	// Call the Cleanup command()
	err = platform.Cleanup(app)
	assert.NoError(err)

	for _, containerType := range [3]string{"web", "db", "dba"} {
		_, err := constructContainerName(containerType, app)
		assert.Error(err)
	}

	// Ensure there are no volumes associated with this project
	client := dockerutil.GetDockerClient()
	volumes, err := client.ListVolumes(docker.ListVolumesOptions{})
	assert.NoError(err)
	for _, volume := range volumes {
		assert.False(volume.Labels["com.docker.compose.project"] == "ddev"+strings.ToLower(app.GetName()))
	}

	// Cleanup the global site database dirs. This does the work instead of running site.Cleanup()
	// because site.Cleanup() removes site directories that we'll need in other tests.
	dir := filepath.Join(util.GetGlobalDdevDir(), site.Name)
	err = os.RemoveAll(dir)
	assert.NoError(err)

	revertDir()
}

// TestGetappsEmpty ensures that GetApps returns an empty list when no applications are running.
func TestGetAppsEmpty(t *testing.T) {
	assert := assert.New(t)

	// Ensure test sites are removed
	for _, site := range TestSites {
		app, err := platform.GetPluginApp("local")
		assert.NoError(err)

		switchDir := site.Chdir()

		testcommon.ClearDockerEnv()
		err = app.Init(site.Dir)
		assert.NoError(err)

		err = app.Down(true)
		assert.NoError(err)

		switchDir()
	}

	apps := platform.GetApps()
	assert.Equal(len(apps["local"]), 0, "Expected to find no apps but found %d apps=%v", len(apps["local"]), apps["local"])
}

// TestRouterNotRunning ensures the router is shut down after all sites are stopped.
func TestRouterNotRunning(t *testing.T) {
	assert := assert.New(t)
	containers, err := dockerutil.GetDockerContainers(false)
	assert.NoError(err)

	for _, container := range containers {
		assert.NotEqual(dockerutil.ContainerName(container), "ddev-router", "ddev-router was not supposed to be running but it was")
	}
}

// constructContainerName builds a container name given the type (web/db/dba) and the app
func constructContainerName(containerType string, app platform.App) (string, error) {
	container, err := app.FindContainerByType(containerType)
	if err != nil {
		return "", err
	}
	name := dockerutil.ContainerName(container)
	return name, nil
}
