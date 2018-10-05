package ddevapp_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/drud/ddev/pkg/archive"
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/testcommon"
	"github.com/drud/ddev/pkg/util"
	"github.com/drud/ddev/pkg/version"

	"github.com/fsouza/go-dockerclient"
	"github.com/google/uuid"
	"github.com/lunixbochs/vtclean"
	log "github.com/sirupsen/logrus"
	asrt "github.com/stretchr/testify/assert"
)

var (
	TestSites = []testcommon.TestSite{
		{
			Name:                          "TestPkgWordpress",
			SourceURL:                     "https://github.com/drud/wordpress/archive/v0.4.0.tar.gz",
			ArchiveInternalExtractionPath: "wordpress-0.4.0/",
			FilesTarballURL:               "https://github.com/drud/ddev_test_tarballs/releases/download/v1.0/wordpress_files.tar.gz",
			DBTarURL:                      "https://github.com/drud/ddev_test_tarballs/releases/download/v1.0/wordpress_db.tar.gz",
			Docroot:                       "htdocs",
			Type:                          "wordpress",
			Safe200URIWithExpectation:     testcommon.URIWithExpect{URI: "/readme.html", Expect: "Welcome. WordPress is a very special project to me."},
			DynamicURI:                    testcommon.URIWithExpect{URI: "/", Expect: "this post has a photo"},
			FilesImageURI:                 "/wp-content/uploads/2017/04/pexels-photo-265186-1024x683.jpeg",
		},
		{
			Name:                          "TestPkgDrupal8",
			SourceURL:                     "https://ftp.drupal.org/files/projects/drupal-8.5.3.tar.gz",
			ArchiveInternalExtractionPath: "drupal-8.5.3/",
			FilesTarballURL:               "https://github.com/drud/ddev_test_tarballs/releases/download/v1.0/drupal8_files.tar.gz",
			FilesZipballURL:               "https://github.com/drud/ddev_test_tarballs/releases/download/v1.0/drupal8_files.zip",
			DBTarURL:                      "https://github.com/drud/ddev_test_tarballs/releases/download/v1.0/drupal8_db.tar.gz",
			DBZipURL:                      "https://github.com/drud/ddev_test_tarballs/releases/download/v1.0/drupal8_db.zip",
			FullSiteTarballURL:            "",
			Type:                          "drupal8",
			Docroot:                       "",
			Safe200URIWithExpectation:     testcommon.URIWithExpect{URI: "/README.txt", Expect: "Drupal is an open source content management platform"},
			DynamicURI:                    testcommon.URIWithExpect{URI: "/node/1", Expect: "this is a post with an image"},
			FilesImageURI:                 "/sites/default/files//2017-04/pexels-photo-265186.jpeg",
		},
		{
			Name:                          "TestPkgDrupal7", // Drupal Kickstart on D7
			SourceURL:                     "https://github.com/drud/drupal-kickstart/archive/v0.4.0.tar.gz",
			ArchiveInternalExtractionPath: "drupal-kickstart-0.4.0/",
			FilesTarballURL:               "https://github.com/drud/drupal-kickstart/releases/download/v0.4.0/files.tar.gz",
			DBTarURL:                      "https://github.com/drud/drupal-kickstart/releases/download/v0.4.0/db.tar.gz",
			FullSiteTarballURL:            "https://github.com/drud/drupal-kickstart/releases/download/v0.4.0/site.tar.gz",
			Docroot:                       "docroot",
			Type:                          "drupal7",
			Safe200URIWithExpectation:     testcommon.URIWithExpect{URI: "/README.txt", Expect: "Drupal is an open source content management platform"},
			DynamicURI:                    testcommon.URIWithExpect{URI: "/drinks/guy-mug", Expect: "The Guy&quot; Mug"},
			FilesImageURI:                 "/sites/default/files/mug-2v1.jpg",
			FullSiteArchiveExtPath:        "docroot/sites/default/files",
		},
		{
			Name:                          "TestPkgDrupal6",
			SourceURL:                     "https://ftp.drupal.org/files/projects/drupal-6.38.tar.gz",
			ArchiveInternalExtractionPath: "drupal-6.38/",
			DBTarURL:                      "https://github.com/drud/ddev_test_tarballs/releases/download/v1.1/drupal6_db.tar.gz",
			FullSiteTarballURL:            "",
			FilesTarballURL:               "https://github.com/drud/ddev_test_tarballs/releases/download/v1.1/drupal6_files.tar.gz",
			Docroot:                       "",
			Type:                          "drupal6",
			Safe200URIWithExpectation:     testcommon.URIWithExpect{URI: "/CHANGELOG.txt", Expect: "Drupal 6.38, 2016-02-24"},
			DynamicURI:                    testcommon.URIWithExpect{URI: "/node/2", Expect: "This is a story. The story is somewhat shaky"},
			FilesImageURI:                 "/sites/default/files/garland_logo.jpg",
		},
		{
			Name:                          "TestPkgBackdrop",
			SourceURL:                     "https://github.com/backdrop/backdrop/archive/1.11.0.tar.gz",
			ArchiveInternalExtractionPath: "backdrop-1.11.0/",
			DBTarURL:                      "https://github.com/drud/ddev_test_tarballs/releases/download/v1.1/backdrop_db.11.0.tar.gz",
			FilesTarballURL:               "https://github.com/drud/ddev_test_tarballs/releases/download/v1.1/backdrop_files.11.0.tar.gz",
			FullSiteTarballURL:            "",
			Docroot:                       "",
			Type:                          "backdrop",
			Safe200URIWithExpectation:     testcommon.URIWithExpect{URI: "/README.md", Expect: "Backdrop is a full-featured content management system"},
			DynamicURI:                    testcommon.URIWithExpect{URI: "/posts/first-post-all-about-kittens", Expect: "Lots of kittens are a good thing"},
			FilesImageURI:                 "/files/styles/large/public/field/image/kittens-large.jpg",
		},
		{
			Name:                          "TestPkgTypo3",
			SourceURL:                     "https://typo3.azureedge.net/typo3/8.7.10/typo3_src-8.7.10.tar.gz",
			ArchiveInternalExtractionPath: "typo3_src-8.7.10/",
			DBTarURL:                      "https://github.com/drud/ddev_test_tarballs/releases/download/v1.0/typo3git_db.tar.gz",
			FullSiteTarballURL:            "",
			Docroot:                       "",
			Type:                          "typo3",
			Safe200URIWithExpectation:     testcommon.URIWithExpect{URI: "/INSTALL.md", Expect: "TYPO3 is an open source PHP based web content management"},
			DynamicURI:                    testcommon.URIWithExpect{URI: "/someting", Expect: "Haven't got this working yet"},
		},
	}
	FullTestSites = TestSites
)

func TestMain(m *testing.M) {
	output.LogSetUp()

	// Ensure the ddev directory is created before tests run.
	_ = util.GetGlobalDdevDir()

	// Since this may be first time ddev has been used, we need the
	// ddev_default network available.
	dockerutil.EnsureDdevNetwork()

	// Avoid having sudo try to add to /etc/hosts.
	// This is normally done by Testsite.Prepare()
	_ = os.Setenv("DRUD_NONINTERACTIVE", "true")

	// Attempt to remove all running containers before starting a test.
	// If no projects are running, this will exit silently and without error.
	// If a system doesn't have `ddev` in its $PATH, this will emit a warning but will not fail the test.
	if _, err := exec.RunCommand("ddev", []string{"remove", "--all"}); err != nil {
		log.Warnf("Failed to remove all running projects: %v", err)
	}

	count := len(ddevapp.GetApps())
	if count > 0 {
		log.Fatalf("ddevapp tests require no projects running. You have %v project(s) running.", count)
	}

	// If GOTEST_SHORT is an integer, then use it as index for a single usage
	// in the array. Any value can be used, it will default to just using the
	// first site in the array.
	gotestShort := os.Getenv("GOTEST_SHORT")
	if gotestShort != "" {
		useSite := 0
		if site, err := strconv.Atoi(gotestShort); err == nil && site >= 0 && site < len(TestSites) {
			useSite = site
		}
		TestSites = []testcommon.TestSite{TestSites[useSite]}
	}

	// testRun is the exit result we'll provide.
	// Start with a clean exit result, it will be changed if we have trouble.
	testRun := 0
	for i := range TestSites {
		err := TestSites[i].Prepare()
		if err != nil {
			log.Fatalf("Prepare() failed on TestSite.Prepare() site=%s, err=%v", TestSites[i].Name, err)
		}

		switchDir := TestSites[i].Chdir()
		runTime := testcommon.TimeTrack(time.Now(), fmt.Sprintf("%s Start", TestSites[i].Name))

		testcommon.ClearDockerEnv()

		app := &ddevapp.DdevApp{}

		err = app.Init(TestSites[i].Dir)
		if err != nil {
			testRun = -1
			log.Errorf("TestMain startup: app.Init() failed on site %s in dir %s, err=%v", TestSites[i].Name, TestSites[i].Dir, err)
			continue
		}

		runTime()
		switchDir()
	}

	if testRun == 0 {
		log.Debugln("Running tests.")
		testRun = m.Run()
	}

	for i, site := range TestSites {
		runTime := testcommon.TimeTrack(time.Now(), fmt.Sprintf("%s Remove", site.Name))

		testcommon.ClearDockerEnv()

		app := &ddevapp.DdevApp{}

		err := app.Init(site.Dir)
		if err != nil {
			log.Fatalf("TestMain shutdown: app.Init() failed on site %s in dir %s, err=%v", TestSites[i].Name, TestSites[i].Dir, err)
		}

		if app.SiteStatus() != ddevapp.SiteNotFound {
			err = app.Down(true, false)
			if err != nil {
				log.Fatalf("TestMain shutdown: app.Down() failed on site %s, err=%v", TestSites[i].Name, err)
			}
		}

		runTime()
		site.Cleanup()
	}

	os.Exit(testRun)
}

// TestDdevStart tests the functionality that is called when "ddev start" is executed
func TestDdevStart(t *testing.T) {
	assert := asrt.New(t)
	app := &ddevapp.DdevApp{}

	for _, site := range TestSites {
		switchDir := site.Chdir()
		runTime := testcommon.TimeTrack(time.Now(), fmt.Sprintf("%s DdevStart", site.Name))

		err := app.Init(site.Dir)
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

		err = app.Down(true, false)
		assert.NoError(err)
		runTime()
		switchDir()
	}

	// try to start a site of same name at different path
	another := TestSites[0]
	err := another.Prepare()
	if err != nil {
		assert.FailNow("TestDdevStart: Prepare() failed on another.Prepare(), err=%v", err)
		return
	}

	err = app.Init(another.Dir)
	assert.Error(err)
	if err != nil {
		assert.Contains(err.Error(), fmt.Sprintf("a project (web container) in running state already exists for %s that was created at %s", TestSites[0].Name, TestSites[0].Dir))
	}

	// Make sure that GetActiveApp() also fails when trying to start app of duplicate name in current directory.
	switchDir := another.Chdir()
	_, err = ddevapp.GetActiveApp("")
	assert.Error(err)
	if err != nil {
		assert.Contains(err.Error(), fmt.Sprintf("a project (web container) in running state already exists for %s that was created at %s", TestSites[0].Name, TestSites[0].Dir))
	}
	testcommon.CleanupDir(another.Dir)
	switchDir()
}

// TestDdevStartMultipleHostnames tests start with multiple hostnames
func TestDdevStartMultipleHostnames(t *testing.T) {
	assert := asrt.New(t)
	app := &ddevapp.DdevApp{}

	for _, site := range TestSites {
		runTime := testcommon.TimeTrack(time.Now(), fmt.Sprintf("%s DdevStartMultipleHostnames", site.Name))
		testcommon.ClearDockerEnv()

		err := app.Init(site.Dir)
		assert.NoError(err)

		// site.Name is explicitly added because if not removed in GetHostNames() it will cause ddev-router failure
		// "a" is repeated for the same reason; a user error of this type should not cause a failure; GetHostNames()
		// should uniqueify them.
		app.AdditionalHostnames = []string{"sub1." + site.Name, "sub2." + site.Name, "subname.sub3." + site.Name, site.Name, site.Name, site.Name}

		// sub1.<sitename>.ddev.local and sitename.ddev.local are deliberately included to prove they don't
		// cause ddev-router failures"
		app.AdditionalFQDNs = []string{"one.example.com", "two.example.com", "a.one.example.com", site.Name + "." + version.DDevTLD, "sub1." + site.Name + version.DDevTLD}

		err = app.WriteConfig()
		assert.NoError(err)

		err = app.Start()
		assert.NoError(err)

		// ensure docker-compose.yaml exists inside .ddev site folder
		composeFile := fileutil.FileExists(app.DockerComposeYAMLPath())
		assert.True(composeFile)

		for _, containerType := range [3]string{"web", "db", "dba"} {
			//nolint: vetshadow
			containerName, err := constructContainerName(containerType, app)
			assert.NoError(err)
			check, err := testcommon.ContainerCheck(containerName, "running")
			assert.NoError(err)
			assert.True(check, "Container check on %s failed", containerType)
		}

		for _, hostname := range app.GetHostnames() {
			testcommon.EnsureLocalHTTPContent(t, "http://"+hostname+site.Safe200URIWithExpectation.URI, site.Safe200URIWithExpectation.Expect)
			testcommon.EnsureLocalHTTPContent(t, "https://"+hostname+site.Safe200URIWithExpectation.URI, site.Safe200URIWithExpectation.Expect)

		}

		// Multiple projects can't run at the same time with the fqdns, so we need to clean
		// up these for tests that run later.
		app.AdditionalFQDNs = []string{}
		app.AdditionalHostnames = []string{}
		err = app.WriteConfig()
		assert.NoError(err)

		err = app.Down(true, false)
		assert.NoError(err)

		runTime()
	}
}

// TestDdevXdebugEnabled tests running with xdebug_enabled = true, etc.
func TestDdevXdebugEnabled(t *testing.T) {
	assert := asrt.New(t)
	app := &ddevapp.DdevApp{}

	site := TestSites[0]
	runTime := testcommon.TimeTrack(time.Now(), fmt.Sprintf("%s DdevXdebugEnabled", site.Name))

	err := app.Init(site.Dir)
	assert.NoError(err)

	// Run with xdebug_enabled: false
	testcommon.ClearDockerEnv()
	app.XdebugEnabled = false
	err = app.WriteConfig()
	assert.NoError(err)
	err = app.Start()
	assert.NoError(err)

	stdout, _, err := app.Exec("web", "php", "--ri", "xdebug")
	assert.Error(err)
	assert.Contains(stdout, "Extension 'xdebug' not present")

	// Run with xdebug_enabled: true
	//err = app.Stop()
	testcommon.ClearDockerEnv()
	app.XdebugEnabled = true
	err = app.WriteConfig()
	assert.NoError(err)
	err = app.Start()
	assert.NoError(err)
	stdout, _, err = app.Exec("web", "php", "--ri", "xdebug")
	assert.NoError(err)
	assert.Contains(stdout, "xdebug support => enabled")
	assert.Contains(stdout, "xdebug.remote_host => host.docker.internal => host.docker.internal")

	err = app.Stop()
	assert.NoError(err)

	runTime()

}

// TestDdevMysqlWorks tests that mysql client can be run in both containers.
func TestDdevMysqlWorks(t *testing.T) {
	assert := asrt.New(t)
	app := &ddevapp.DdevApp{}

	site := TestSites[0]
	runTime := testcommon.TimeTrack(time.Now(), fmt.Sprintf("%s DdevMysqlWorks", site.Name))

	err := app.Init(site.Dir)
	assert.NoError(err)

	testcommon.ClearDockerEnv()
	err = app.Start()
	assert.NoError(err)

	// Test that mysql + .my.cnf works on web container
	_, _, err = app.Exec("web", "bash", "-c", "mysql -e 'SELECT USER();' | grep 'db@'")
	assert.NoError(err)
	_, _, err = app.Exec("web", "bash", "-c", "mysql -e 'SELECT DATABASE();' | grep 'db'")
	assert.NoError(err)

	// Test that mysql + .my.cnf works on db container
	_, _, err = app.Exec("db", "bash", "-c", "mysql -e 'SELECT USER();' | grep 'root@localhost'")
	assert.NoError(err)
	_, _, err = app.Exec("db", "bash", "-c", "mysql -e 'SELECT DATABASE();' | grep 'db'")
	assert.NoError(err)

	err = app.Stop()
	assert.NoError(err)

	runTime()

}

// TestStartWithoutDdev makes sure we don't have a regression where lack of .ddev
// causes a panic.
func TestStartWithoutDdevConfig(t *testing.T) {
	// Set up tests and give ourselves a working directory.
	assert := asrt.New(t)
	testDir := testcommon.CreateTmpDir("TestStartWithoutDdevConfig")

	// testcommon.Chdir()() and CleanupDir() check their own errors (and exit)
	defer testcommon.CleanupDir(testDir)
	defer testcommon.Chdir(testDir)()

	err := os.MkdirAll(testDir+"/sites/default", 0777)
	assert.NoError(err)
	err = os.Chdir(testDir)
	assert.NoError(err)

	_, err = ddevapp.GetActiveApp("")
	assert.Error(err)
	if err != nil {
		assert.Contains(err.Error(), "Could not find a project")
	}
}

// TestGetApps tests the GetApps function to ensure it accurately returns a list of running applications.
func TestGetApps(t *testing.T) {
	assert := asrt.New(t)

	// Start the apps.
	for _, site := range TestSites {
		testcommon.ClearDockerEnv()
		app := &ddevapp.DdevApp{}

		err := app.Init(site.Dir)
		assert.NoError(err)

		err = app.Start()
		assert.NoError(err)
	}

	apps := ddevapp.GetApps()
	assert.Equal(len(apps), len(TestSites))

	for _, testSite := range TestSites {
		var found bool
		for _, app := range apps {
			if testSite.Name == app.GetName() {
				found = true
				break
			}
		}
		assert.True(found, "Found testSite %s in list", testSite.Name)
	}
}

// TestDdevImportDB tests the functionality that is called when "ddev import-db" is executed
func TestDdevImportDB(t *testing.T) {
	assert := asrt.New(t)
	app := &ddevapp.DdevApp{}
	testDir, _ := os.Getwd()

	for _, site := range TestSites {
		switchDir := site.Chdir()
		runTime := testcommon.TimeTrack(time.Now(), fmt.Sprintf("%s DdevImportDB", site.Name))

		testcommon.ClearDockerEnv()
		err := app.Init(site.Dir)
		assert.NoError(err)

		// Test simple db loads.
		for _, file := range []string{"users.sql", "users.mysql", "users.sql.gz", "users.mysql.gz", "users.sql.tar", "users.mysql.tar", "users.sql.tar.gz", "users.mysql.tar.gz", "users.sql.tgz", "users.mysql.tgz", "users.sql.zip", "users.mysql.zip"} {
			path := filepath.Join(testDir, "testdata", file)
			err = app.ImportDB(path, "")
			assert.NoError(err, "Failed to app.ImportDB path: %s err: %v", path, err)

			// Test that a settings file has correct hash_salt format
			switch app.Type {
			case "drupal7":
				// nolint: vetshadow
				drupalHashSalt, err := fileutil.FgrepStringInFile(app.SiteLocalSettingsPath, "$drupal_hash_salt")
				assert.NoError(err)
				assert.True(drupalHashSalt)
			case "drupal8":
				// nolint: vetshadow
				settingsHashSalt, err := fileutil.FgrepStringInFile(app.SiteLocalSettingsPath, "settings['hash_salt']")
				assert.NoError(err)
				assert.True(settingsHashSalt)
			case "wordpress":
				// nolint: vetshadow
				hasAuthSalt, err := fileutil.FgrepStringInFile(app.SiteSettingsPath, "SECURE_AUTH_SALT")
				assert.NoError(err)
				assert.True(hasAuthSalt)
			}

		}

		if site.DBTarURL != "" {
			// nolint: vetshadow
			_, cachedArchive, err := testcommon.GetCachedArchive(site.Name, site.Name+"_siteTarArchive", "", site.DBTarURL)
			assert.NoError(err)
			err = app.ImportDB(cachedArchive, "")
			assert.NoError(err)

			out, _, err := app.Exec("db", "mysql", "-e", "SHOW TABLES;")
			assert.NoError(err)

			assert.Contains(out, "Tables_in_db")
			assert.False(strings.Contains(out, "Empty set"))

			assert.NoError(err)
		}

		if site.DBZipURL != "" {
			// nolint: vetshadow
			_, cachedArchive, err := testcommon.GetCachedArchive(site.Name, site.Name+"_siteZipArchive", "", site.DBZipURL)

			assert.NoError(err)
			err = app.ImportDB(cachedArchive, "")
			assert.NoError(err)

			out, _, err := app.Exec("db", "mysql", "-e", "SHOW TABLES;")
			assert.NoError(err)

			assert.Contains(out, "Tables_in_db")
			assert.False(strings.Contains(out, "Empty set"))
		}

		if site.FullSiteTarballURL != "" {
			// nolint: vetshadow
			_, cachedArchive, err := testcommon.GetCachedArchive(site.Name, site.Name+"_FullSiteTarballURL", "", site.FullSiteTarballURL)
			assert.NoError(err)

			err = app.ImportDB(cachedArchive, "data.sql")
			assert.NoError(err, "Failed to find data.sql at root of tarball %s", cachedArchive)
		}
		// We don't want all the projects running at once.
		err = app.Stop()
		assert.NoError(err)

		runTime()
		switchDir()
	}
}

// TestDdevFullSiteSetup tests a full import-db and import-files and then looks to see if
// we have a spot-test success hit on a URL
func TestDdevFullSiteSetup(t *testing.T) {
	assert := asrt.New(t)
	app := &ddevapp.DdevApp{}

	for _, site := range TestSites {
		switchDir := site.Chdir()
		runTime := testcommon.TimeTrack(time.Now(), fmt.Sprintf("%s DdevFullSiteSetup", site.Name))

		testcommon.ClearDockerEnv()
		err := app.Init(site.Dir)
		assert.NoError(err)

		err = app.Start()
		assert.NoError(err)

		if site.DBTarURL != "" {
			// nolint: vetshadow
			_, cachedArchive, err := testcommon.GetCachedArchive(site.Name, site.Name+"_siteTarArchive", "", site.DBTarURL)
			assert.NoError(err)
			err = app.ImportDB(cachedArchive, "")
			assert.NoError(err)
		}
		if site.FilesTarballURL != "" {
			// nolint: vetshadow
			_, tarballPath, err := testcommon.GetCachedArchive(site.Name, "local-tarballs-files", "", site.FilesTarballURL)
			assert.NoError(err)
			err = app.ImportFiles(tarballPath, "")
			assert.NoError(err)
		}

		// Make sure that the loaded site at least does something.
		testcommon.EnsureLocalHTTPContent(t, app.GetHTTPURL()+site.DynamicURI.URI, site.DynamicURI.Expect)

		// Load an image from the files section
		if site.FilesImageURI != "" {
			_, resp, err := testcommon.GetLocalHTTPResponse(t, app.GetHTTPURL()+site.FilesImageURI)
			assert.NoError(err)
			assert.Equal(resp.Header["Content-Type"][0], "image/jpeg")
		}

		// We don't want all the projects running at once.
		err = app.Down(true, false)
		assert.NoError(err)

		runTime()
		switchDir()
	}
}

// TestDdevRestoreSnapshot tests creating a snapshot and reverting to it
func TestDdevRestoreSnapshot(t *testing.T) {
	assert := asrt.New(t)
	app := &ddevapp.DdevApp{}

	runTime := testcommon.TimeTrack(time.Now(), fmt.Sprintf("TestDdevRestoreSnapshot"))

	d7testerTest1Dump, err := filepath.Abs(filepath.Join("testdata", "restore_snapshot", "d7tester_test_1.sql.gz"))
	assert.NoError(err)
	d7testerTest2Dump, err := filepath.Abs(filepath.Join("testdata", "restore_snapshot", "d7tester_test_2.sql.gz"))
	assert.NoError(err)

	// Use d7 only for this test, the key thing is the database interaction
	site := FullTestSites[2]
	// If running this with GOTEST_SHORT we have to create the directory, tarball etc.
	if site.Dir == "" || !fileutil.FileExists(site.Dir) {
		err = site.Prepare()
		if err != nil {
			t.Fatalf("Prepare() failed on TestSite.Prepare() site=%s, err=%v", site.Name, err)
		}
	}

	switchDir := site.Chdir()
	testcommon.ClearDockerEnv()

	err = app.Init(site.Dir)
	if err != nil {
		if app.SiteStatus() != ddevapp.SiteRunning {
			t.Fatalf("app.Init() failed on site %s in dir %s, err=%v", site.Name, site.Dir, err)
		}
	}

	err = app.Start()
	if err != nil {
		t.Fatalf("TestMain startup: app.Start() failed on site %s, err=%v", site.Name, err)
	}

	err = app.ImportDB(d7testerTest1Dump, "")
	assert.NoError(err, "Failed to app.ImportDB path: %s err: %v", d7testerTest1Dump, err)
	testcommon.EnsureLocalHTTPContent(t, app.GetHTTPURL(), "d7 tester test 1 has 1 node")

	// Make a snapshot of d7 tester test 1
	backupsDir := filepath.Join(app.GetConfigPath(""), "db_snapshots")
	snapshotName, err := app.SnapshotDatabase("d7testerTest1")
	assert.NoError(err)
	assert.EqualValues(snapshotName, "d7testerTest1")
	assert.True(fileutil.FileExists(filepath.Join(backupsDir, snapshotName, "xtrabackup_info")))

	err = app.ImportDB(d7testerTest2Dump, "")
	assert.NoError(err, "Failed to app.ImportDB path: %s err: %v", d7testerTest2Dump, err)
	testcommon.EnsureLocalHTTPContent(t, app.GetHTTPURL(), "d7 tester test 2 has 2 nodes")

	snapshotName, err = app.SnapshotDatabase("d7testerTest2")
	assert.NoError(err)
	assert.EqualValues(snapshotName, "d7testerTest2")
	assert.True(fileutil.FileExists(filepath.Join(backupsDir, snapshotName, "xtrabackup_info")))

	err = app.RestoreSnapshot("d7testerTest1")
	assert.NoError(err)
	testcommon.EnsureLocalHTTPContent(t, app.GetHTTPURL(), "d7 tester test 1 has 1 node")
	err = app.RestoreSnapshot("d7testerTest2")
	assert.NoError(err)
	//testcommon.EnsureLocalHTTPContent(t, app.GetHTTPURL(), "d7 tester test 2 has 2 nodes")

	body, resp, err := testcommon.GetLocalHTTPResponse(t, app.GetHTTPURL())
	assert.NoError(err, "GetLocalHTTPResponse returned err on rawurl %s: %v", app.GetHTTPURL(), err)
	assert.Contains(body, "d7 tester test 2 has 2 nodes")
	if err != nil {
		t.Logf("resp after timeout: %v", resp)
		stdout := testcommon.CaptureUserOut()
		err = app.Logs("web", false, false, "")
		assert.NoError(err)
		out := stdout()
		t.Logf("web container logs after timeout: %s", out)
	}

	err = app.Down(true, false)
	assert.NoError(err)

	// TODO: Check behavior of ddev rm with snapshot, see if it has right stuff in it.

	runTime()
	switchDir()
}

// TestWriteableFilesDirectory tests to make sure that files created on host are writable on container
// and files ceated in container are correct user on host.
func TestWriteableFilesDirectory(t *testing.T) {
	assert := asrt.New(t)
	app := &ddevapp.DdevApp{}

	for _, site := range TestSites {
		switchDir := site.Chdir()
		runTime := testcommon.TimeTrack(time.Now(), fmt.Sprintf("%s DdevImportDB", site.Name))

		testcommon.ClearDockerEnv()
		err := app.Init(site.Dir)
		assert.NoError(err)

		err = app.Start()
		assert.NoError(err)

		uploadDir := app.GetUploadDir()
		if uploadDir != "" {

			// Use exec to touch a file in the container and see what the result is. Make sure it comes out with ownership
			// making it writeable on the host.
			filename := fileutil.RandomFilenameBase()
			dirname := fileutil.RandomFilenameBase()
			// Use path.Join for items on th container (linux) and filepath.Join for items on the host.
			inContainerDir := path.Join(uploadDir, dirname)
			onHostDir := filepath.Join(app.Docroot, inContainerDir)
			inContainerRelativePath := path.Join(inContainerDir, filename)
			onHostRelativePath := path.Join(onHostDir, filename)

			err = os.MkdirAll(onHostDir, 0775)
			assert.NoError(err)
			_, _, err = app.Exec("web", "sh", "-c", "echo 'content created inside container\n' >"+inContainerRelativePath)
			assert.NoError(err)

			// Now try to append to the file on the host.
			// os.OpenFile() for append here fails if the file does not already exist.
			f, err := os.OpenFile(onHostRelativePath, os.O_APPEND|os.O_WRONLY, 0660)
			assert.NoError(err)
			_, err = f.WriteString("this addition to the file was added on the host side")
			assert.NoError(err)
			_ = f.Close()

			// Create a file on the host and see what the result is. Make sure we can not append/write to it in the container.
			filename = fileutil.RandomFilenameBase()
			dirname = fileutil.RandomFilenameBase()
			inContainerDir = path.Join(uploadDir, dirname)
			onHostDir = filepath.Join(app.Docroot, inContainerDir)
			inContainerRelativePath = path.Join(inContainerDir, filename)
			onHostRelativePath = filepath.Join(onHostDir, filename)

			err = os.MkdirAll(onHostDir, 0775)
			assert.NoError(err)

			f, err = os.OpenFile(onHostRelativePath, os.O_CREATE|os.O_RDWR, 0660)
			assert.NoError(err)
			_, err = f.WriteString("this base content was inserted on the host side\n")
			assert.NoError(err)
			_ = f.Close()
			// if the file exists, add to it. We don't want to add if it's not already there.
			_, _, err = app.Exec("web", "sh", "-c", "if [ -f "+inContainerRelativePath+" ]; then echo 'content added inside container\n' >>"+inContainerRelativePath+"; fi")
			assert.NoError(err)
			// grep the file for both the content added on host and that added in container.
			_, _, err = app.Exec("web", "sh", "-c", "grep 'base content was inserted on the host' "+inContainerRelativePath+"&& grep 'content added inside container' "+inContainerRelativePath)
			assert.NoError(err)
		}

		runTime()
		switchDir()
	}
}

// TestDdevImportFilesDir tests that "ddev import-files" can successfully import non-archive directories
func TestDdevImportFilesDir(t *testing.T) {
	assert := asrt.New(t)
	app := &ddevapp.DdevApp{}

	// Create a dummy directory to test non-archive imports
	importDir, err := ioutil.TempDir("", t.Name())
	assert.NoError(err)
	fileNames := make([]string, 0)
	for i := 0; i < 5; i++ {
		fileName := uuid.New().String()
		fileNames = append(fileNames, fileName)

		fullPath := filepath.Join(importDir, fileName)
		err = ioutil.WriteFile(fullPath, []byte(fileName), 0644)
		assert.NoError(err)
	}

	for _, site := range TestSites {
		switchDir := site.Chdir()
		runTime := testcommon.TimeTrack(time.Now(), fmt.Sprintf("%s %s", site.Name, t.Name()))

		testcommon.ClearDockerEnv()
		err = app.Init(site.Dir)
		assert.NoError(err)

		// Function under test
		err = app.ImportFiles(importDir, "")
		assert.NoError(err, "Importing a directory returned an error:", err)

		// Confirm contents of destination dir after import
		absUploadDir := filepath.Join(app.AppRoot, app.Docroot, app.GetUploadDir())
		uploadedFiles, err := ioutil.ReadDir(absUploadDir)
		assert.NoError(err)

		uploadedFilesMap := map[string]bool{}
		for _, uploadedFile := range uploadedFiles {
			uploadedFilesMap[filepath.Base(uploadedFile.Name())] = true
		}

		for _, expectedFile := range fileNames {
			assert.True(uploadedFilesMap[expectedFile], "Expected file %s not found for site: %s", expectedFile, site.Name)
		}

		runTime()
		switchDir()
	}
}

// TestDdevImportFiles tests the functionality that is called when "ddev import-files" is executed
func TestDdevImportFiles(t *testing.T) {
	assert := asrt.New(t)
	app := &ddevapp.DdevApp{}

	for _, site := range TestSites {
		switchDir := site.Chdir()
		runTime := testcommon.TimeTrack(time.Now(), fmt.Sprintf("%s %s", site.Name, t.Name()))

		testcommon.ClearDockerEnv()
		err := app.Init(site.Dir)
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

		if site.FullSiteTarballURL != "" && site.FullSiteArchiveExtPath != "" {
			_, siteTarPath, err := testcommon.GetCachedArchive(site.Name, "local-site-tar", "", site.FullSiteTarballURL)
			assert.NoError(err)
			err = app.ImportFiles(siteTarPath, site.FullSiteArchiveExtPath)
			assert.NoError(err)
		}

		runTime()
		switchDir()
	}
}

// TestDdevImportFilesCustomUploadDir ensures that files are imported to a custom upload directory when requested
func TestDdevImportFilesCustomUploadDir(t *testing.T) {
	assert := asrt.New(t)
	app := &ddevapp.DdevApp{}

	for _, site := range TestSites {
		switchDir := site.Chdir()
		runTime := testcommon.TimeTrack(time.Now(), fmt.Sprintf("%s %s", site.Name, t.Name()))

		testcommon.ClearDockerEnv()
		err := app.Init(site.Dir)
		assert.NoError(err)

		// Set custom upload dir
		app.UploadDir = "my/upload/dir"
		absUploadDir := filepath.Join(app.AppRoot, app.Docroot, app.UploadDir)
		err = os.MkdirAll(absUploadDir, 0755)
		assert.NoError(err)

		if site.FilesTarballURL != "" {
			_, tarballPath, err := testcommon.GetCachedArchive(site.Name, "local-tarballs-files", "", site.FilesTarballURL)
			assert.NoError(err)
			err = app.ImportFiles(tarballPath, "")
			assert.NoError(err)

			// Ensure upload dir isn't empty
			fileInfoSlice, err := ioutil.ReadDir(absUploadDir)
			assert.NoError(err)
			assert.NotEmpty(fileInfoSlice)
		}

		if site.FilesZipballURL != "" {
			_, zipballPath, err := testcommon.GetCachedArchive(site.Name, "local-zipballs-files", "", site.FilesZipballURL)
			assert.NoError(err)
			err = app.ImportFiles(zipballPath, "")
			assert.NoError(err)

			// Ensure upload dir isn't empty
			fileInfoSlice, err := ioutil.ReadDir(absUploadDir)
			assert.NoError(err)
			assert.NotEmpty(fileInfoSlice)
		}

		if site.FullSiteTarballURL != "" && site.FullSiteArchiveExtPath != "" {
			_, siteTarPath, err := testcommon.GetCachedArchive(site.Name, "local-site-tar", "", site.FullSiteTarballURL)
			assert.NoError(err)
			err = app.ImportFiles(siteTarPath, site.FullSiteArchiveExtPath)
			assert.NoError(err)

			// Ensure upload dir isn't empty
			fileInfoSlice, err := ioutil.ReadDir(absUploadDir)
			assert.NoError(err)
			assert.NotEmpty(fileInfoSlice)
		}

		runTime()
		switchDir()
	}
}

// TestDdevExec tests the execution of commands inside a docker container of a site.
func TestDdevExec(t *testing.T) {
	assert := asrt.New(t)
	app := &ddevapp.DdevApp{}

	for _, site := range TestSites {
		switchDir := site.Chdir()
		runTime := testcommon.TimeTrack(time.Now(), fmt.Sprintf("%s DdevExec", site.Name))

		err := app.Init(site.Dir)
		assert.NoError(err)

		out, _, err := app.Exec("web", "pwd")
		assert.NoError(err)
		assert.Contains(out, "/var/www/html")

		_, _, err = app.Exec("db", "mysql", "-e", "DROP DATABASE db;")
		assert.NoError(err)
		_, _, err = app.Exec("db", "mysql", "information_schema", "-e", "CREATE DATABASE db;")
		assert.NoError(err)

		switch app.GetType() {
		case "drupal6":
			fallthrough
		case "drupal7":
			fallthrough
		case "drupal8":
			out, _, err = app.Exec("web", "drush", "status")
			assert.NoError(err)
			assert.Regexp("PHP configuration[ :]*/etc/php/[0-9].[0-9]/fpm/php.ini", out)
		case "wordpress":
			out, _, err = app.Exec("web", "wp", "--info")
			assert.NoError(err)
			assert.Regexp("/etc/php.*/php.ini", out)
		}

		runTime()
		switchDir()

	}
}

// TestDdevLogs tests the container log output functionality.
func TestDdevLogs(t *testing.T) {
	assert := asrt.New(t)

	// Skip test because on Windows because the CaptureUserOut() hangs, at least
	// sometimes.
	if runtime.GOOS == "windows" {
		t.Skip("Skipping test TestDdevLogs on Windows")
	}

	app := &ddevapp.DdevApp{}

	for _, site := range TestSites {
		switchDir := site.Chdir()
		runTime := testcommon.TimeTrack(time.Now(), fmt.Sprintf("%s DdevLogs", site.Name))

		err := app.Init(site.Dir)
		assert.NoError(err)

		stdout := testcommon.CaptureUserOut()
		err = app.Logs("web", false, false, "")
		assert.NoError(err)
		out := stdout()
		assert.Contains(out, "Server started")

		stdout = testcommon.CaptureUserOut()
		err = app.Logs("db", false, false, "")
		assert.NoError(err)
		out = stdout()
		assert.Contains(out, "MySQL init process done. Ready for start up.")

		// Test that we can get logs when project is stopped also
		err = app.Stop()
		assert.NoError(err)

		stdout = testcommon.CaptureUserOut()
		err = app.Logs("web", false, false, "")
		assert.NoError(err)
		out = stdout()
		assert.Contains(out, "Server started")

		stdout = testcommon.CaptureUserOut()
		err = app.Logs("db", false, false, "")
		assert.NoError(err)
		out = stdout()
		assert.Contains(out, "MySQL init process done. Ready for start up.")

		err = app.Start()
		assert.NoError(err)

		runTime()
		switchDir()
	}
}

// TestProcessHooks tests execution of commands defined in config.yaml
func TestProcessHooks(t *testing.T) {
	assert := asrt.New(t)

	for _, site := range TestSites {
		cleanup := site.Chdir()
		runTime := testcommon.TimeTrack(time.Now(), fmt.Sprintf("%s ProcessHooks", site.Name))

		testcommon.ClearDockerEnv()
		app, err := ddevapp.NewApp(site.Dir, ddevapp.DefaultProviderName)
		assert.NoError(err)

		// Note that any ExecHost commands must be able to run on Windows.
		// echo and pwd are things that work pretty much the same in both places.
		app.Commands = map[string][]ddevapp.Command{
			"hook-test": {
				{
					Exec: "ls /usr/local/bin/composer",
				},
				{
					ExecHost: "echo something",
				},
			},
		}

		stdout := testcommon.CaptureUserOut()
		err = app.ProcessHooks("hook-test")
		assert.NoError(err)

		// Ignore color in putput, can be different in different OS's
		out := vtclean.Clean(stdout(), false)

		assert.Contains(out, "hook-test exec command succeeded, output below ---\n/usr/local/bin/composer")
		assert.Contains(out, "--- Running host command: echo something ---\nRunning Command Command=echo something\nsomething")

		runTime()
		cleanup()

	}
}

// TestDdevStop tests the functionality that is called when "ddev stop" is executed
func TestDdevStop(t *testing.T) {
	assert := asrt.New(t)

	app := &ddevapp.DdevApp{}

	for _, site := range TestSites {
		switchDir := site.Chdir()
		runTime := testcommon.TimeTrack(time.Now(), fmt.Sprintf("%s DdevStop", site.Name))

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

// TestDdevStopMissingDirectory tests that the 'ddev stop' command works properly on sites with missing directories or ddev configs.
func TestDdevStopMissingDirectory(t *testing.T) {
	assert := asrt.New(t)

	site := TestSites[0]
	testcommon.ClearDockerEnv()
	app := &ddevapp.DdevApp{}
	err := app.Init(site.Dir)
	assert.NoError(err)

	// Restart the site since it was stopped in the previous test.
	if app.SiteStatus() != ddevapp.SiteRunning {
		err = app.Start()
		assert.NoError(err)
	}

	tempPath := testcommon.CreateTmpDir("site-copy")
	siteCopyDest := filepath.Join(tempPath, "site")
	defer removeAllErrCheck(tempPath, assert)

	// Move the site directory to a temp location to mimic a missing directory.
	err = os.Rename(site.Dir, siteCopyDest)
	assert.NoError(err)

	out := app.Stop()
	assert.Error(out)
	assert.Contains(out.Error(), "If you would like to continue using ddev to manage this project please restore your files to that directory.")
	// Move the site directory back to its original location.
	err = os.Rename(siteCopyDest, site.Dir)
	assert.NoError(err)

}

// TestDescribe tests that the describe command works properly on a running
// and also a stopped project.
func TestDescribe(t *testing.T) {
	assert := asrt.New(t)
	app := &ddevapp.DdevApp{}

	for _, site := range TestSites {
		switchDir := site.Chdir()

		testcommon.ClearDockerEnv()
		err := app.Init(site.Dir)
		assert.NoError(err)

		// It should already be running, but start does no harm.
		err = app.Start()

		// If we have a problem starting, get the container logs and output.
		if err != nil {
			stdout := testcommon.CaptureUserOut()
			logsErr := app.Logs("web", false, false, "")
			assert.NoError(logsErr)
			out := stdout()
			assert.NoError(err, "app.Start(%s) failed: %v, web container logs=%s", site.Name, err, out)
		}

		desc, err := app.Describe()
		assert.NoError(err)
		assert.EqualValues(ddevapp.SiteRunning, desc["status"], "")
		assert.EqualValues(app.GetName(), desc["name"])
		assert.EqualValues(ddevapp.RenderHomeRootedDir(app.GetAppRoot()), desc["shortroot"])
		assert.EqualValues(app.GetAppRoot(), desc["approot"])
		assert.EqualValues(app.GetPhpVersion(), desc["php_version"])

		// Now stop it and test behavior.
		err = app.Stop()
		assert.NoError(err)

		desc, err = app.Describe()
		assert.NoError(err)
		assert.EqualValues(ddevapp.SiteStopped, desc["status"])
		switchDir()
	}
}

// TestDescribeMissingDirectory tests that the describe command works properly on sites with missing directories or ddev configs.
func TestDescribeMissingDirectory(t *testing.T) {
	assert := asrt.New(t)
	site := TestSites[0]
	tempPath := testcommon.CreateTmpDir("site-copy")
	siteCopyDest := filepath.Join(tempPath, "site")
	defer removeAllErrCheck(tempPath, assert)

	app := &ddevapp.DdevApp{}
	err := app.Init(site.Dir)
	assert.NoError(err)

	// Move the site directory to a temp location to mimick a missing directory.
	err = os.Rename(site.Dir, siteCopyDest)
	assert.NoError(err)

	desc, err := app.Describe()
	assert.NoError(err)
	assert.Contains(desc["status"], ddevapp.SiteDirMissing, "Status did not include the phrase '%s' when describing a site with missing directories.", ddevapp.SiteDirMissing)
	// Move the site directory back to its original location.
	err = os.Rename(siteCopyDest, site.Dir)
	assert.NoError(err)
}

// TestRouterPortsCheck makes sure that we can detect if the ports are available before starting the router.
func TestRouterPortsCheck(t *testing.T) {
	assert := asrt.New(t)

	// First, stop any sites that might be running
	app := &ddevapp.DdevApp{}

	// Stop all sites, which should get the router out of there.
	for _, site := range TestSites {
		switchDir := site.Chdir()

		testcommon.ClearDockerEnv()
		err := app.Init(site.Dir)
		assert.NoError(err)

		err = app.Stop()
		assert.NoError(err)

		switchDir()
	}

	// Now start one site, it's hard to get router to behave without one site.
	site := TestSites[0]
	testcommon.ClearDockerEnv()

	app, err := ddevapp.GetActiveApp(site.Name)
	if err != nil {
		t.Fatalf("Failed to GetActiveApp(%s), err:%v", site.Name, err)
	}
	err = app.Start()
	assert.NoError(err, "app.Start(%s) failed, err: %v", app.GetName(), err)

	// Stop the router using code from StopRouterIfNoContainers().
	// StopRouterIfNoContainers can't be used here because it checks to see if containers are running
	// and doesn't do its job as a result.
	dest := ddevapp.RouterComposeYAMLPath()
	_, _, err = dockerutil.ComposeCmd([]string{dest}, "-p", ddevapp.RouterProjectName, "down", "-v")
	assert.NoError(err, "Failed to stop router using docker-compose, err=%v", err)

	// Occupy port 80 using docker busybox trick, then see if we can start router.
	// This is done with docker so that we don't have to use explicit sudo
	containerID, err := exec.RunCommand("sh", []string{"-c", "docker run -d -p80:80 --rm busybox:latest sleep 100 2>/dev/null"})
	if err != nil {
		t.Fatalf("Failed to run docker command to occupy port 80, err=%v output=%v", err, containerID)
	}
	containerID = strings.TrimSpace(containerID)

	// Now try to start the router. It should fail because the port is occupied.
	err = ddevapp.StartDdevRouter()
	assert.Error(err, "Failure: router started even though port 80 was occupied")

	// Remove our dummy busybox docker container.
	out, err := exec.RunCommand("docker", []string{"rm", "-f", containerID})
	assert.NoError(err, "Failed to docker rm the port-occupier container, err=%v output=%v", err, out)
}

// TestCleanupWithoutCompose ensures app containers can be properly cleaned up without a docker-compose config file present.
func TestCleanupWithoutCompose(t *testing.T) {
	assert := asrt.New(t)

	// Skip test because we can't rename folders while they're in use if running on Windows.
	if runtime.GOOS == "windows" {
		t.Skip("Skipping test TestCleanupWithoutCompose on Windows")
	}

	site := TestSites[0]

	revertDir := site.Chdir()
	app := &ddevapp.DdevApp{}

	testcommon.ClearDockerEnv()
	err := app.Init(site.Dir)
	assert.NoError(err)

	// Ensure we have a site started so we have something to cleanup
	err = app.Start()
	assert.NoError(err)
	// Setup by creating temp directory and nesting a folder for our site.
	tempPath := testcommon.CreateTmpDir("site-copy")
	siteCopyDest := filepath.Join(tempPath, "site")
	defer removeAllErrCheck(tempPath, assert)

	// Move site directory to a temp directory to mimick a missing directory.
	err = os.Rename(site.Dir, siteCopyDest)
	assert.NoError(err)

	// Call the Down command()
	// Notice that we set the removeData parameter to true.
	// This gives us added test coverage over sites with missing directories
	// by ensuring any associated database files get cleaned up as well.
	err = app.Down(true, false)
	assert.NoError(err)

	for _, containerType := range [3]string{"web", "db", "dba"} {
		// nolint: vetshadow
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
	// Move the site directory back to its original location.
	err = os.Rename(siteCopyDest, site.Dir)
	assert.NoError(err)
}

// TestGetappsEmpty ensures that GetApps returns an empty list when no applications are running.
func TestGetAppsEmpty(t *testing.T) {
	assert := asrt.New(t)

	// Ensure test sites are removed
	for _, site := range TestSites {
		app := &ddevapp.DdevApp{}

		switchDir := site.Chdir()

		testcommon.ClearDockerEnv()
		err := app.Init(site.Dir)
		assert.NoError(err)

		if app.SiteStatus() != ddevapp.SiteNotFound {
			err = app.Down(true, false)
			assert.NoError(err)
		}
		switchDir()
	}

	apps := ddevapp.GetApps()
	assert.Equal(len(apps), 0, "Expected to find no apps but found %d apps=%v", len(apps), apps)
}

// TestRouterNotRunning ensures the router is shut down after all sites are stopped.
func TestRouterNotRunning(t *testing.T) {
	assert := asrt.New(t)
	containers, err := dockerutil.GetDockerContainers(false)
	assert.NoError(err)

	for _, container := range containers {
		assert.NotEqual(dockerutil.ContainerName(container), "ddev-router", "ddev-router was not supposed to be running but it was")
	}
}

// TestListWithoutDir prevents regression where ddev list panics if one of the
// sites found is missing a directory
func TestListWithoutDir(t *testing.T) {
	// Set up tests and give ourselves a working directory.
	assert := asrt.New(t)
	testcommon.ClearDockerEnv()
	packageDir, _ := os.Getwd()

	// startCount is the count of apps at the start of this adventure
	apps := ddevapp.GetApps()
	startCount := len(apps)

	testDir := testcommon.CreateTmpDir("TestStartWithoutDdevConfig")
	defer testcommon.CleanupDir(testDir)

	err := os.MkdirAll(testDir+"/sites/default", 0777)
	assert.NoError(err)
	err = os.Chdir(testDir)
	assert.NoError(err)

	app, err := ddevapp.NewApp(testDir, ddevapp.DefaultProviderName)
	assert.NoError(err)
	app.Name = "junk"
	app.Type = "drupal7"
	err = app.WriteConfig()
	assert.NoError(err)

	// Do a start on the configured site.
	app, err = ddevapp.GetActiveApp("")
	assert.NoError(err)
	err = app.Start()
	assert.NoError(err)

	// Make sure we move out of the directory for Windows' sake
	garbageDir := testcommon.CreateTmpDir("RestingHere")
	defer testcommon.CleanupDir(garbageDir)

	err = os.Chdir(garbageDir)
	assert.NoError(err)

	testcommon.CleanupDir(testDir)

	apps = ddevapp.GetApps()

	assert.EqualValues(len(apps), startCount+1)

	// Make a whole table and make sure our app directory missing shows up.
	// This could be done otherwise, but we'd have to go find the site in the
	// array first.
	table := ddevapp.CreateAppTable()
	for _, site := range apps {
		// nolint: vetshadow
		desc, err := site.Describe()
		if err != nil {
			t.Fatalf("Failed to describe site %s: %v", site.GetName(), err)
		}

		ddevapp.RenderAppRow(table, desc)
	}

	// testDir on Windows has backslashes in it, resulting in invalid regexp
	// Remove them and use ., which is good enough.
	testDirSafe := strings.Replace(testDir, "\\", ".", -1)
	assert.Regexp(regexp.MustCompile("(?s)"+ddevapp.SiteDirMissing+".*"+testDirSafe), table.String())

	err = app.Down(true, false)
	assert.NoError(err)

	// Change back to package dir. Lots of things will have to be cleaned up
	// in defers, and for windows we have to not be sitting in them.
	err = os.Chdir(packageDir)
	assert.NoError(err)
}

type URLRedirectExpectations struct {
	scheme              string
	uri                 string
	expectedRedirectURI string
}

// TestHttpsRedirection tests to make sure that webserver and php redirect to correct
// scheme (http or https).
func TestHttpsRedirection(t *testing.T) {
	// Set up tests and give ourselves a working directory.
	assert := asrt.New(t)
	testcommon.ClearDockerEnv()
	packageDir, _ := os.Getwd()

	testDir := testcommon.CreateTmpDir("TestHttpsRedirection")
	defer testcommon.CleanupDir(testDir)
	appDir := filepath.Join(testDir, "proj")
	err := fileutil.CopyDir(filepath.Join(packageDir, "testdata", "TestHttpsRedirection"), appDir)
	assert.NoError(err)
	err = os.Chdir(appDir)
	assert.NoError(err)

	app, err := ddevapp.NewApp(appDir, ddevapp.DefaultProviderName)
	assert.NoError(err)
	app.Name = "proj"
	app.Type = "php"

	expectations := []URLRedirectExpectations{
		{"https", "/subdir", "/subdir/"},
		{"https", "/redir_abs.php", "/landed.php"},
		{"https", "/redir_relative.php", "/landed.php"},
		{"http", "/subdir", "/subdir/"},
		{"http", "/redir_abs.php", "/landed.php"},
		{"http", "/redir_relative.php", "/landed.php"},
	}

	for _, webserverType := range []string{"nginx-fpm", "apache-fpm", "apache-cgi"} {
		app.WebserverType = webserverType
		err = app.WriteConfig()
		assert.NoError(err)

		// Do a start on the configured site.
		app, err = ddevapp.GetActiveApp("")
		assert.NoError(err)
		err = app.Start()
		assert.NoError(err)

		// Test for directory redirects under https and http
		for _, parts := range expectations {

			reqURL := parts.scheme + "://" + app.GetHostname() + parts.uri
			// nolint: vetshadow
			_, resp, err := testcommon.GetLocalHTTPResponse(t, reqURL)
			assert.Error(err)
			assert.NotNil(resp, "resp was nil for webserver_type=%s url=%s", webserverType, reqURL)
			if resp != nil {
				locHeader := resp.Header.Get("Location")

				expectedRedirect := parts.expectedRedirectURI
				// However, if we're hitting redir_abs.php (or apache hitting directory), the redirect will be the whole url.
				if strings.Contains(parts.uri, "redir_abs.php") || webserverType != "nginx-fpm" {
					expectedRedirect = parts.scheme + "://" + app.GetHostname() + parts.expectedRedirectURI
				}
				// Except the php relative redirect is always relative.
				if strings.Contains(parts.uri, "redir_relative.php") {
					expectedRedirect = parts.expectedRedirectURI
				}

				assert.EqualValues(locHeader, expectedRedirect, "For webserver_type %s url %s expected redirect %s != actual %s", webserverType, reqURL, expectedRedirect, locHeader)
			}
		}
	}

	err = app.Down(true, false)
	assert.NoError(err)

	// Change back to package dir. Lots of things will have to be cleaned up
	// in defers, and for windows we have to not be sitting in them.
	err = os.Chdir(packageDir)
	assert.NoError(err)
}

// TestMultipleComposeFiles checks to see if a set of docker-compose files gets
// properly loaded in the right order, with docker-compose.yaml first and
// with docker-compose.override.yaml last.
func TestMultipleComposeFiles(t *testing.T) {
	// Set up tests and give ourselves a working directory.
	assert := asrt.New(t)

	// Make sure that valid yaml files get properly loaded in the proper order
	app, err := ddevapp.NewApp("./testdata/testMultipleComposeFiles", "")
	assert.NoError(err)

	files, err := app.ComposeFiles()
	assert.NoError(err)
	assert.True(files[0] == filepath.Join(app.AppConfDir(), "docker-compose.yaml"))
	assert.True(files[len(files)-1] == filepath.Join(app.AppConfDir(), "docker-compose.override.yaml"))

	// Make sure that some docker-compose.yml and docker-compose.yaml conflict gets noted properly
	app, err = ddevapp.NewApp("./testdata/testConflictingYamlYml", "")
	assert.NoError(err)

	_, err = app.ComposeFiles()
	assert.Error(err)
	if err != nil {
		assert.Contains(err.Error(), "there are more than one docker-compose.y*l")
	}

	// Make sure that some docker-compose.override.yml and docker-compose.override.yaml conflict gets noted properly
	app, err = ddevapp.NewApp("./testdata/testConflictingOverrideYaml", "")
	assert.NoError(err)

	_, err = app.ComposeFiles()
	assert.Error(err)
	if err != nil {
		assert.Contains(err.Error(), "there are more than one docker-compose.override.y*l")
	}

	// Make sure the error gets pointed out of there's no main docker-compose.yaml
	app, err = ddevapp.NewApp("./testdata/testNoDockerCompose", "")
	assert.NoError(err)

	_, err = app.ComposeFiles()
	assert.Error(err)
	if err != nil {
		assert.Contains(err.Error(), "failed to find a docker-compose.yml or docker-compose.yaml")
	}

	// Catch if we have no docker files at all.
	// This should also fail if the docker-compose.yaml.bak gets loaded.
	app, err = ddevapp.NewApp("./testdata/testNoDockerFilesAtAll", "")
	assert.NoError(err)

	_, err = app.ComposeFiles()
	assert.Error(err)
	if err != nil {
		assert.Contains(err.Error(), "failed to load any docker-compose.*y*l files")
	}
}

// TestGetAllURLs ensures the GetAllURLs function returns the expected number of URLs,
// and that one of them is the direct web container address.
func TestGetAllURLs(t *testing.T) {
	assert := asrt.New(t)

	for _, site := range TestSites {
		runTime := testcommon.TimeTrack(time.Now(), fmt.Sprintf("%s GetAllURLs", site.Name))

		testcommon.ClearDockerEnv()
		app := new(ddevapp.DdevApp)

		err := app.Init(site.Dir)
		assert.NoError(err)

		// Add some additional hostnames
		app.AdditionalHostnames = []string{
			fmt.Sprintf("sub1.%s", site.Name),
			fmt.Sprintf("sub2.%s", site.Name),
			fmt.Sprintf("sub3.%s", site.Name),
		}

		err = app.WriteConfig()
		assert.NoError(err)

		err = app.Start()
		assert.NoError(err)

		urls := app.GetAllURLs()

		// Convert URLs to map[string]bool
		urlMap := make(map[string]bool)
		for _, u := range urls {
			urlMap[u] = true
		}

		// We expect two URLs for each hostname (http/https) and one direct web container address.
		expectedNumUrls := (2 * len(app.GetHostnames())) + 1
		assert.Equal(len(urlMap), expectedNumUrls, "Unexpected number of URLs returned: %d", len(urlMap))

		// Ensure urlMap contains direct address of the web container
		webContainer, err := app.FindContainerByType("web")
		assert.NoError(err)

		dockerIP, err := dockerutil.GetDockerIP()
		assert.NoError(err)

		// Find HTTP port of web container
		var port docker.APIPort
		for _, p := range webContainer.Ports {
			if p.PrivatePort == 80 {
				port = p
				break
			}
		}

		expectedDirectAddress := fmt.Sprintf("http://%s:%d", dockerIP, port.PublicPort)
		exists := urlMap[expectedDirectAddress]

		assert.True(exists, "URL list for app: %s does not contain direct web container address: %s", app.Name, expectedDirectAddress)

		// Multiple projects can't run at the same time with the fqdns, so we need to clean
		// up these for tests that run later.
		app.AdditionalFQDNs = []string{}
		app.AdditionalHostnames = []string{}
		err = app.WriteConfig()
		assert.NoError(err)

		err = app.Stop()
		assert.NoError(err)

		runTime()
	}
}

// TestWebserverType checks that webserver_type:apache-cgi or apache-fpm does the right thing
func TestWebserverType(t *testing.T) {
	assert := asrt.New(t)

	for _, site := range TestSites {
		runTime := testcommon.TimeTrack(time.Now(), fmt.Sprintf("%s GetAllURLs", site.Name))

		app := new(ddevapp.DdevApp)

		err := app.Init(site.Dir)
		assert.NoError(err)

		// Copy our phpinfo into the docroot of testsite.
		pwd, err := os.Getwd()
		assert.NoError(err)
		err = fileutil.CopyFile(filepath.Join(pwd, "testdata", "phpinfo.php"), filepath.Join(app.AppRoot, app.Docroot, "phpinfo.php"))

		assert.NoError(err)
		for _, app.WebserverType = range []string{"apache-fpm", "apache-cgi", "nginx-fpm"} {

			err = app.WriteConfig()
			assert.NoError(err)

			testcommon.ClearDockerEnv()

			err = app.Start()
			assert.NoError(err)

			// nolint: vetshadow
			out, resp, err := testcommon.GetLocalHTTPResponse(t, app.GetWebContainerDirectURL()+"/phpinfo.php")
			assert.NoError(err)

			expectedServerType := "Apache/2"
			if app.WebserverType == "nginx-fpm" {
				expectedServerType = "nginx"
			}
			assert.Contains(resp.Header["Server"][0], expectedServerType, "Server header for project=%s, app.WebserverType=%s should be %s", app.Name, app.WebserverType, expectedServerType)
			assert.Contains(out, "$_SERVER['SERVER_SOFTWARE']</td><td class=\"v\">"+expectedServerType, "For app.WebserverType=%s expected $_SERVER['SERVER_SOFTWARE'] to show %s", app.WebserverType, expectedServerType)

		}

		// Set the apptype back to whatever the default was so we don't break any following tests.
		testVar := os.Getenv("DDEV_TEST_WEBSERVER_TYPE")
		if testVar != "" {
			app.WebserverType = testVar
			err = app.WriteConfig()
			assert.NoError(err)
		}
		err = app.Stop()
		assert.NoError(err)

		runTime()
	}
}

// TestDbMigration tests migration from bind-mounted db to volume-mounted db
// This should be important around the time of its release, 2018-08-02 or so, but should be increasingly
// irrelevant after that and can eventually be removed.
func TestDbMigration(t *testing.T) {
	assert := asrt.New(t)
	runTime := testcommon.TimeTrack(time.Now(), fmt.Sprintf("TestDbMigration"))

	app := &ddevapp.DdevApp{}
	dbMigrationTarball, err := filepath.Abs(filepath.Join("testdata", "db_migration", "d7_to_migrate.tgz"))
	assert.NoError(err)

	// Use d7 only for this test
	site := FullTestSites[2]

	// If running this with GOTEST_SHORT we have to create the directory, tarball etc.
	if site.Dir == "" || !fileutil.FileExists(site.Dir) {
		err = site.Prepare()
		if err != nil {
			t.Fatalf("Prepare() failed on TestSite.Prepare() site=%s, err=%v", site.Name, err)
		}
	}

	switchDir := site.Chdir()
	testcommon.ClearDockerEnv()

	err = app.Init(site.Dir)
	assert.NoError(err)

	dataDir := filepath.Join(util.GetGlobalDdevDir(), app.Name, "mysql")

	// Remove any existing dataDir or migration backups
	if fileutil.FileExists(dataDir) {
		err = os.RemoveAll(dataDir)
		assert.NoError(err)
	}
	if fileutil.FileExists(dataDir + "_migrated.bak") {
		err = os.RemoveAll(dataDir + "_migrated.bak")
		assert.NoError(err)
	}

	// Start it to make sure we can do a removeData
	err = app.Start()
	assert.NoError(err)

	// removeData removes the volume so we have a completely clean start
	err = app.Down(true, false)
	assert.NoError(err)

	// Untar the to-migrate db into old-style dataDir (~/.ddev/projectname/mysql)
	err = os.MkdirAll(dataDir, 0755)
	assert.NoError(err)
	err = archive.Untar(dbMigrationTarball, dataDir, "")
	assert.NoError(err)

	_, err = app.CreateSettingsFile()
	assert.NoError(err)

	// app.Start() will discover the mysql directory and migrate it to a snapshot.
	err = app.Start()
	if err != nil {
		t.Fatalf("TestMain startup: app.Start() failed on site %s, err=%v", site.Name, err)
	}

	// Check to see expected migrated data.
	testcommon.EnsureLocalHTTPContent(t, app.GetHTTPURL(), "This d7 site is ready for migration to docker-volume")

	assert.True(fileutil.FileExists(dataDir + "_migrated.bak"))
	assert.False(fileutil.FileExists(dataDir))

	err = app.Down(true, false)
	assert.NoError(err)

	runTime()
	switchDir()
}

// TestInternalAndExternalAccessToURL checks we can access content from host and from inside container by URL (with port)
func TestInternalAndExternalAccessToURL(t *testing.T) {
	assert := asrt.New(t)

	for _, site := range TestSites {
		runTime := testcommon.TimeTrack(time.Now(), fmt.Sprintf("%s TestInternalAndExternalAccessToURL", site.Name))

		app := new(ddevapp.DdevApp)

		err := app.Init(site.Dir)
		assert.NoError(err)

		//nolint: vet
		for _, pair := range []testcommon.PortPair{{"80", "443"}, {"8080", "8443"}} {
			testcommon.ClearDockerEnv()
			app.RouterHTTPPort = pair.HTTPPort
			app.RouterHTTPSPort = pair.HTTPSPort
			err = app.WriteConfig()
			assert.NoError(err)

			if app.SiteStatus() == ddevapp.SiteStopped || app.SiteStatus() == ddevapp.SiteRunning {
				err = app.Down(true, false)
				assert.NoError(err)
			}
			err = app.Start()
			assert.NoError(err)

			// Ensure that we can access from the host even with extra port specifications.
			testcommon.EnsureLocalHTTPContent(t, app.GetHTTPURL()+site.Safe200URIWithExpectation.URI, site.Safe200URIWithExpectation.Expect)
			testcommon.EnsureLocalHTTPContent(t, app.GetHTTPSURL()+site.Safe200URIWithExpectation.URI, site.Safe200URIWithExpectation.Expect)

			// Ensure that we can access the same URL from within the web container (via router)
			var out string
			out, _, err = app.Exec("web", "curl", "-sk", app.GetHTTPURL()+site.Safe200URIWithExpectation.URI)
			assert.NoError(err)
			assert.Contains(out, site.Safe200URIWithExpectation.Expect)

			out, _, err = app.Exec("web", "curl", "-sk", app.GetHTTPSURL()+site.Safe200URIWithExpectation.URI)
			assert.NoError(err)
			assert.Contains(out, site.Safe200URIWithExpectation.Expect)
		}

		// Set the ports back to the default was so we don't break any following tests.
		app.RouterHTTPSPort = "443"
		app.RouterHTTPPort = "80"
		err = app.WriteConfig()
		assert.NoError(err)
		err = app.Stop()
		assert.NoError(err)

		runTime()
	}
}

// constructContainerName builds a container name given the type (web/db/dba) and the app
func constructContainerName(containerType string, app *ddevapp.DdevApp) (string, error) {
	container, err := app.FindContainerByType(containerType)
	if err != nil {
		return "", err
	}
	name := dockerutil.ContainerName(container)
	return name, nil
}

func removeAllErrCheck(path string, assert *asrt.Assertions) {
	err := os.RemoveAll(path)
	assert.NoError(err)
}
