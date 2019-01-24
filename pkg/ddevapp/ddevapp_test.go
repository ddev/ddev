package ddevapp_test

import (
	"bufio"
	"fmt"
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"net"
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
			Type:                          ddevapp.AppTypeWordPress,
			Safe200URIWithExpectation:     testcommon.URIWithExpect{URI: "/readme.html", Expect: "Welcome. WordPress is a very special project to me."},
			DynamicURI:                    testcommon.URIWithExpect{URI: "/", Expect: "this post has a photo"},
			FilesImageURI:                 "/wp-content/uploads/2017/04/pexels-photo-265186-1024x683.jpeg",
		},
		{
			Name:                          "TestPkgDrupal8",
			SourceURL:                     "https://ftp.drupal.org/files/projects/drupal-8.6.1.tar.gz",
			ArchiveInternalExtractionPath: "drupal-8.6.1/",
			FilesTarballURL:               "https://github.com/drud/ddev_test_tarballs/releases/download/v1.1/drupal8_6_1_files.tar.gz",
			FilesZipballURL:               "https://github.com/drud/ddev_test_tarballs/releases/download/v1.0/drupal8_files.zip",
			DBTarURL:                      "https://github.com/drud/ddev_test_tarballs/releases/download/v1.1/drupal8_6_1_db.tar.gz",
			DBZipURL:                      "https://github.com/drud/ddev_test_tarballs/releases/download/v1.0/drupal8_db.zip",
			FullSiteTarballURL:            "",
			Type:                          ddevapp.AppTypeDrupal8,
			Docroot:                       "",
			Safe200URIWithExpectation:     testcommon.URIWithExpect{URI: "/README.txt", Expect: "Drupal is an open source content management platform"},
			DynamicURI:                    testcommon.URIWithExpect{URI: "/node/1", Expect: "this is a post with an image"},
			FilesImageURI:                 "/sites/default/files//2017-04/pexels-photo-265186.jpeg",
		},
		{
			Name:                          "TestPkgDrupal7", // Drupal Kickstart on D7
			SourceURL:                     "https://ftp.drupal.org/files/projects/drupal-7.61.tar.gz",
			ArchiveInternalExtractionPath: "drupal-7.61/",
			FilesTarballURL:               "https://github.com/drud/ddev_test_tarballs/releases/download/v1.1/d7test-7.59.files.tar.gz",
			DBTarURL:                      "https://github.com/drud/ddev_test_tarballs/releases/download/v1.1/d7test-7.59-db.tar.gz",
			FullSiteTarballURL:            "",
			Docroot:                       "",
			Type:                          ddevapp.AppTypeDrupal7,
			Safe200URIWithExpectation:     testcommon.URIWithExpect{URI: "/README.txt", Expect: "Drupal is an open source content management platform"},
			DynamicURI:                    testcommon.URIWithExpect{URI: "/node/1", Expect: "D7 test project, kittens edition"},
			FilesImageURI:                 "/sites/default/files/field/image/kittens-large.jpg",
			FullSiteArchiveExtPath:        "docroot/sites/default/files",
		},
		{
			Name:                          "TestPkgDrupal6",
			SourceURL:                     "https://ftp.drupal.org/files/projects/drupal-6.38.tar.gz",
			ArchiveInternalExtractionPath: "drupal-6.38/",
			DBTarURL:                      "https://github.com/drud/ddev_test_tarballs/releases/download/v1.1/drupal6.38_db.tar.gz",
			FullSiteTarballURL:            "",
			FilesTarballURL:               "https://github.com/drud/ddev_test_tarballs/releases/download/v1.1/drupal6_files.tar.gz",
			Docroot:                       "",
			Type:                          ddevapp.AppTypeDrupal6,
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
			Type:                          ddevapp.AppTypeBackdrop,
			Safe200URIWithExpectation:     testcommon.URIWithExpect{URI: "/README.md", Expect: "Backdrop is a full-featured content management system"},
			DynamicURI:                    testcommon.URIWithExpect{URI: "/posts/first-post-all-about-kittens", Expect: "Lots of kittens are a good thing"},
			FilesImageURI:                 "/files/styles/large/public/field/image/kittens-large.jpg",
		},
		{
			Name:                          "TestPkgTypo3",
			SourceURL:                     "https://github.com/drud/typo3-v9-test/archive/v0.2.2.tar.gz",
			ArchiveInternalExtractionPath: "typo3-v9-test-0.2.2/",
			DBTarURL:                      "https://github.com/drud/ddev_test_tarballs/releases/download/v1.1/typo3_v9.5_introduction_db.tar.gz",
			FilesTarballURL:               "https://github.com/drud/ddev_test_tarballs/releases/download/v1.1/typo3_v9.5_introduction_files.tar.gz",
			FullSiteTarballURL:            "",
			Docroot:                       "public",
			Type:                          ddevapp.AppTypeTYPO3,
			Safe200URIWithExpectation:     testcommon.URIWithExpect{URI: "/README.txt", Expect: "junk readme simply for reading"},
			DynamicURI:                    testcommon.URIWithExpect{URI: "/index.php?id=65", Expect: "Boxed Content"},
			FilesImageURI:                 "/fileadmin/introduction/images/streets/nikita-maru-70928.jpg",
		},
	}
	FullTestSites = TestSites
)

func TestMain(m *testing.M) {
	output.LogSetUp()

	// Ensure the ddev directory is created before tests run.
	_ = globalconfig.GetGlobalDdevDir()

	// Since this may be first time ddev has been used, we need the
	// ddev_default network available.
	dockerutil.EnsureDdevNetwork()

	// Avoid having sudo try to add to /etc/hosts.
	// This is normally done by Testsite.Prepare()
	_ = os.Setenv("DRUD_NONINTERACTIVE", "true")

	// Attempt to remove all running containers before starting a test.
	// If no projects are running, this will exit silently and without error.
	// If a system doesn't have `ddev` in its $PATH, this will emit a warning but will not fail the test.
	if _, err := exec.RunCommand("ddev", []string{"remove", "--all", "--stop-ssh-agent"}); err != nil {
		log.Warnf("Failed to remove all running projects: %v", err)
	}

	for _, volume := range []string{"ddev-composer-cache", "ddev-router-cert-cache", "ddev-ssh-agent_dot_ssh", "ddev-ssh-agent_socket_dir"} {
		err := dockerutil.RemoveVolume(volume)
		if err != nil && err.Error() != "no such volume" {
			log.Errorf("TestMain startup: Failed to delete volume %s: %v", volume, err)
		}
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

		testcommon.ClearDockerEnv()

		app := &ddevapp.DdevApp{}
		err = app.Init(TestSites[i].Dir)
		if err != nil {
			testRun = -1
			log.Errorf("TestMain startup: app.Init() failed on site %s in dir %s, err=%v", TestSites[i].Name, TestSites[i].Dir, err)
			continue
		}
		err = app.WriteConfig()
		if err != nil {
			testRun = -1
			log.Errorf("TestMain startup: app.WriteConfig() failed on site %s in dir %s, err=%v", TestSites[i].Name, TestSites[i].Dir, err)
			continue
		}
		// TODO: webcache PR will add other volumes that should be removed here.
		for _, volume := range []string{app.Name + "-mariadb"} {
			err = dockerutil.RemoveVolume(volume)
			if err != nil && err.Error() != "no such volume" {
				log.Errorf("TestMain startup: Failed to delete volume %s: %v", volume, err)
			}
		}

		switchDir()
	}

	if testRun == 0 {
		log.Debugln("Running tests.")
		testRun = m.Run()
	}

	for i, site := range TestSites {
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
		site.Cleanup()
	}

	os.Exit(testRun)
}

// TestDdevStart tests the functionality that is called when "ddev start" is executed
func TestDdevStart(t *testing.T) {
	assert := asrt.New(t)
	app := &ddevapp.DdevApp{}

	// Make sure this leaves us in the original test directory
	testDir, _ := os.Getwd()
	//nolint: errcheck
	defer os.Chdir(testDir)

	site := TestSites[0]
	switchDir := site.Chdir()
	runTime := testcommon.TimeTrack(time.Now(), fmt.Sprintf("%s DdevStart", site.Name))

	err := app.Init(site.Dir)
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
		assert.True(check, "Container check on %s failed", containerType)
	}

	err = app.Down(true, false)
	assert.NoError(err)
	runTime()
	switchDir()

	// Start up TestSites[0] again with a post-start hook
	// When run the first time, it should execute the hook, second time it should not
	err = os.Chdir(site.Dir)
	assert.NoError(err)
	err = app.Init(site.Dir)
	app.Commands = map[string][]ddevapp.Command{"post-start": {{Exec: "bash -c 'echo hello'"}}}

	assert.NoError(err)
	stdout := util.CaptureUserOut()
	err = app.Start()
	assert.NoError(err)
	out := stdout()
	assert.Contains(out, "Running exec command")
	assert.Contains(out, "hello\n")

	// When we run it again, it should not execute the post-start hook because the
	// container has already been created and does not need to be recreated.
	stdout = util.CaptureUserOut()
	err = app.Start()
	assert.NoError(err)
	out = stdout()
	assert.NotContains(out, "Running exec command")
	assert.NotContains(out, "hello\n")

	// try to start a site of same name at different path
	another := site
	err = another.Prepare()
	if err != nil {
		assert.FailNow("TestDdevStart: Prepare() failed on another.Prepare(), err=%v", err)
		return
	}

	badapp := &ddevapp.DdevApp{}

	err = badapp.Init(another.Dir)
	if err == nil {
		logs, logErr := app.CaptureLogs("web", false, "")
		require.Error(t, err, "did not receive err from badapp.Init, logErr=%v, logs:\n======================= logs from app webserver =================\n%s\n============ end logs =========\n", logErr, logs)
	}
	if err != nil {
		assert.Contains(err.Error(), fmt.Sprintf("a project (web container) in running state already exists for %s that was created at %s", TestSites[0].Name, TestSites[0].Dir))
	}

	// Try to start a site of same name at an equivalent but different path. It should work.
	tmpDir, err := testcommon.OsTempDir()
	assert.NoError(err)
	symlink := filepath.Join(tmpDir, fileutil.RandomFilenameBase())
	err = os.Symlink(app.AppRoot, symlink)
	assert.NoError(err)
	//nolint: errcheck
	defer os.Remove(symlink)
	symlinkApp := &ddevapp.DdevApp{}

	err = symlinkApp.Init(symlink)
	assert.NoError(err)

	// Make sure that GetActiveApp() also fails when trying to start app of duplicate name in current directory.
	switchDir = another.Chdir()
	_, err = ddevapp.GetActiveApp("")
	assert.Error(err)
	if err != nil {
		assert.Contains(err.Error(), fmt.Sprintf("a project (web container) in running state already exists for %s that was created at %s", TestSites[0].Name, TestSites[0].Dir))
	}
	testcommon.CleanupDir(another.Dir)
	switchDir()

	// Clean up site 0
	err = app.Down(true, false)
	assert.NoError(err)
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
		if err != nil && strings.Contains(err.Error(), "db container failed") {
			stdout := util.CaptureUserOut()
			err = app.Logs("db", false, false, "")
			assert.NoError(err)
			out := stdout()
			t.Logf("DB Logs after app.Start: \n%s\n=== END DB LOGS ===", out)
		}

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

		for _, hostname := range app.GetHostnames() {
			_, _ = testcommon.EnsureLocalHTTPContent(t, "http://"+hostname+site.Safe200URIWithExpectation.URI, site.Safe200URIWithExpectation.Expect)
			_, _ = testcommon.EnsureLocalHTTPContent(t, "https://"+hostname+site.Safe200URIWithExpectation.URI, site.Safe200URIWithExpectation.Expect)

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
	err = app.StartAndWaitForSync(0)
	//nolint: errcheck
	defer app.Down(true, false)
	require.NoError(t, err)

	opts := &ddevapp.ExecOpts{
		Service: "web",
		Cmd:     []string{"php", "--ri", "xdebug"},
	}
	stdout, _, err := app.Exec(opts)
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
	//nolint: errcheck
	defer app.Down(true, false)

	stdout, _, err = app.Exec(opts)
	assert.NoError(err)
	assert.Contains(stdout, "xdebug support => enabled")
	assert.Contains(stdout, "xdebug.remote_host => host.docker.internal => host.docker.internal")

	// Start a listener on port 9000 of localhost (where PHPStorm or whatever would listen)
	ln, err := net.Listen("tcp", ":9000")
	assert.NoError(err)
	// Curl to the project's index.php or anything else
	_, _, _ = testcommon.GetLocalHTTPResponse(t, app.GetHTTPURL(), 1)

	conn, err := ln.Accept()
	assert.NoError(err)

	// Grab the Xdebug connection start and look in it for "Xdebug"
	b := make([]byte, 650)
	_, err = bufio.NewReader(conn).Read(b)
	require.NoError(t, err)
	lineString := string(b)
	assert.Contains(lineString, "Xdebug")
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
	err = app.StartAndWaitForSync(0)
	//nolint: errcheck
	defer app.Down(true, false)
	require.NoError(t, err)

	// Test that mysql + .my.cnf works on web container
	_, _, err = app.Exec(&ddevapp.ExecOpts{
		Service: "web",
		Cmd:     []string{"bash", "-c", "mysql -e 'SELECT USER();' | grep 'db@'"},
	})
	assert.NoError(err)
	_, _, err = app.Exec(&ddevapp.ExecOpts{
		Service: "web",
		Cmd:     []string{"bash", "-c", "mysql -e 'SELECT DATABASE();' | grep 'db'"},
	})
	assert.NoError(err)

	// Test that mysql + .my.cnf works on db container
	_, _, err = app.Exec(&ddevapp.ExecOpts{
		Service: "db",
		Cmd:     []string{"bash", "-c", "mysql -e 'SELECT USER();' | grep 'root@localhost'"},
	})
	assert.NoError(err)
	_, _, err = app.Exec(&ddevapp.ExecOpts{
		Service: "db",
		Cmd:     []string{"bash", "-c", "mysql -e 'SELECT DATABASE();' | grep 'db'"},
	})
	assert.NoError(err)

	err = app.Down(true, false)
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
	assert.Equal(len(TestSites), len(apps))

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

	// Now shut down all sites as we expect them to be shut down.
	for _, site := range TestSites {
		testcommon.ClearDockerEnv()
		app := &ddevapp.DdevApp{}

		err := app.Init(site.Dir)
		assert.NoError(err)

		err = app.Down(true, false)
		assert.NoError(err)

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
		err = app.StartAndWaitForSync(2)
		assert.NoError(err)

		// Test simple db loads.
		for _, file := range []string{"users.sql", "users.mysql", "users.sql.gz", "users.mysql.gz", "users.sql.tar", "users.mysql.tar", "users.sql.tar.gz", "users.mysql.tar.gz", "users.sql.tgz", "users.mysql.tgz", "users.sql.zip", "users.mysql.zip"} {
			path := filepath.Join(testDir, "testdata", file)
			err = app.ImportDB(path, "", false)
			assert.NoError(err, "Failed to app.ImportDB path: %s err: %v", path, err)
			if err != nil {
				continue
			}

			// Test that a settings file has correct hash_salt format
			switch app.Type {
			case ddevapp.AppTypeDrupal7:
				drupalHashSalt, err := fileutil.FgrepStringInFile(app.SiteLocalSettingsPath, "$drupal_hash_salt")
				assert.NoError(err)
				assert.True(drupalHashSalt)
			case ddevapp.AppTypeDrupal8:
				settingsHashSalt, err := fileutil.FgrepStringInFile(app.SiteLocalSettingsPath, "settings['hash_salt']")
				assert.NoError(err)
				assert.True(settingsHashSalt)
			case ddevapp.AppTypeWordPress:
				hasAuthSalt, err := fileutil.FgrepStringInFile(app.SiteSettingsPath, "SECURE_AUTH_SALT")
				assert.NoError(err)
				assert.True(hasAuthSalt)
			}
		}

		if site.DBTarURL != "" {
			_, cachedArchive, err := testcommon.GetCachedArchive(site.Name, site.Name+"_siteTarArchive", "", site.DBTarURL)
			assert.NoError(err)
			err = app.ImportDB(cachedArchive, "", false)
			assert.NoError(err)

			out, _, err := app.Exec(&ddevapp.ExecOpts{
				Service: "db",
				Cmd:     []string{"mysql", "-e", "SHOW TABLES;"},
			})
			assert.NoError(err)

			assert.Contains(out, "Tables_in_db")
			assert.False(strings.Contains(out, "Empty set"))

			assert.NoError(err)
		}

		if site.DBZipURL != "" {
			_, cachedArchive, err := testcommon.GetCachedArchive(site.Name, site.Name+"_siteZipArchive", "", site.DBZipURL)

			assert.NoError(err)
			err = app.ImportDB(cachedArchive, "", false)
			assert.NoError(err)

			out, _, err := app.Exec(&ddevapp.ExecOpts{
				Service: "db",
				Cmd:     []string{"mysql", "-e", "SHOW TABLES;"},
			})
			assert.NoError(err)

			assert.Contains(out, "Tables_in_db")
			assert.False(strings.Contains(out, "Empty set"))
		}

		if site.FullSiteTarballURL != "" {
			_, cachedArchive, err := testcommon.GetCachedArchive(site.Name, site.Name+"_FullSiteTarballURL", "", site.FullSiteTarballURL)
			assert.NoError(err)

			err = app.ImportDB(cachedArchive, "data.sql", false)
			assert.NoError(err, "Failed to find data.sql at root of tarball %s", cachedArchive)
		}
		// We don't want all the projects running at once.
		err = app.Down(true, false)
		assert.NoError(err)

		runTime()
		switchDir()
	}
}

// TestDdevOldMariaDB tests db import/export/start with Mariadb 10.1
func TestDdevOldMariaDB(t *testing.T) {
	assert := asrt.New(t)
	app := &ddevapp.DdevApp{}
	testDir, _ := os.Getwd()

	site := TestSites[0]
	switchDir := site.Chdir()
	defer switchDir()
	runTime := testcommon.TimeTrack(time.Now(), fmt.Sprintf("%s DdevOldMariaDB", site.Name))

	testcommon.ClearDockerEnv()
	err := app.Init(site.Dir)
	assert.NoError(err)

	// Make sure there isn't an old db laying around
	_ = dockerutil.RemoveVolume(app.Name + "-mariadb")
	//nolint: errcheck
	defer dockerutil.RemoveVolume(app.Name + "-mariadb")

	app.MariaDBVersion = ddevapp.MariaDB101
	app.DBImage = version.GetDBImage(app.MariaDBVersion)
	startErr := app.StartAndWaitForSync(15)
	//nolint: errcheck
	defer app.Down(true, false)

	if startErr != nil {
		appLogs, err := ddevapp.GetErrLogsFromApp(app, startErr)
		assert.NoError(err)
		t.Fatalf("app.StartAndWaitForSync() failure %v; logs:\n=====\n%s\n=====\n", startErr, appLogs)
	}

	importPath := filepath.Join(testDir, "testdata", "users.sql")
	err = app.ImportDB(importPath, "", false)
	require.NoError(t, err)

	err = os.Mkdir("tmp", 0777)
	require.NoError(t, err)

	err = fileutil.PurgeDirectory("tmp")
	assert.NoError(err)

	// Test that we can export-db to a gzipped file
	err = app.ExportDB("tmp/users1.sql.gz", true)
	assert.NoError(err)

	// Validate contents
	err = archive.Ungzip("tmp/users1.sql.gz", "tmp")
	assert.NoError(err)
	stringFound, err := fileutil.FgrepStringInFile("tmp/users1.sql", "Table structure for table `users`")
	assert.NoError(err)
	assert.True(stringFound)

	err = fileutil.PurgeDirectory("tmp")
	assert.NoError(err)

	// Export to an ungzipped file and validate
	err = app.ExportDB("tmp/users2.sql", false)
	assert.NoError(err)

	// Validate contents
	stringFound, err = fileutil.FgrepStringInFile("tmp/users2.sql", "Table structure for table `users`")
	assert.NoError(err)
	assert.True(stringFound)

	err = fileutil.PurgeDirectory("tmp")
	assert.NoError(err)

	// Capture to stdout without gzip compression
	stdout := util.CaptureStdOut()
	err = app.ExportDB("", false)
	assert.NoError(err)
	out := stdout()
	assert.Contains(out, "Table structure for table `users`")

	snapshotName := fileutil.RandomFilenameBase()
	_, err = app.SnapshotDatabase(snapshotName)
	assert.NoError(err)
	err = app.RestoreSnapshot(snapshotName)
	assert.NoError(err)

	// Restore of a 10.2 snapshot should fail.
	// Attempt a restore with a pre-mariadb_10.2 snapshot. It should fail and give a link.
	newerSnapshotTarball, err := filepath.Abs(filepath.Join(testDir, "testdata", "restore_snapshot", "d7tester_test_1.snapshot_mariadb_10_2.tgz"))
	assert.NoError(err)

	err = archive.Untar(newerSnapshotTarball, filepath.Join(site.Dir, ".ddev", "db_snapshots"), "")
	assert.NoError(err)
	err = app.RestoreSnapshot("d7testerTest1")
	assert.Error(err)
	assert.Contains(err.Error(), "is not compatible")

	runTime()
}

// TestDdevExportDB tests the functionality that is called when "ddev export-db" is executed
func TestDdevExportDB(t *testing.T) {
	assert := asrt.New(t)
	app := &ddevapp.DdevApp{}
	testDir, _ := os.Getwd()

	site := TestSites[0]
	switchDir := site.Chdir()
	defer switchDir()

	runTime := testcommon.TimeTrack(time.Now(), fmt.Sprintf("%s DdevExportDB", site.Name))

	testcommon.ClearDockerEnv()
	err := app.Init(site.Dir)
	assert.NoError(err)
	err = app.StartAndWaitForSync(0)
	assert.NoError(err)
	//nolint: errcheck
	defer app.Down(true, false)
	importPath := filepath.Join(testDir, "testdata", "users.sql")
	err = app.ImportDB(importPath, "", false)
	require.NoError(t, err)

	_ = os.Mkdir("tmp", 0777)
	// Most likely reason for failure is it exists, so let that go
	err = fileutil.PurgeDirectory("tmp")
	assert.NoError(err)

	// Test that we can export-db to a gzipped file
	err = app.ExportDB("tmp/users1.sql.gz", true)
	assert.NoError(err)

	// Validate contents
	err = archive.Ungzip("tmp/users1.sql.gz", "tmp")
	assert.NoError(err)
	stringFound, err := fileutil.FgrepStringInFile("tmp/users1.sql", "Table structure for table `users`")
	assert.NoError(err)
	assert.True(stringFound)

	err = fileutil.PurgeDirectory("tmp")
	assert.NoError(err)

	// Export to an ungzipped file and validate
	err = app.ExportDB("tmp/users2.sql", false)
	assert.NoError(err)

	// Validate contents
	stringFound, err = fileutil.FgrepStringInFile("tmp/users2.sql", "Table structure for table `users`")
	assert.NoError(err)
	assert.True(stringFound)

	err = fileutil.PurgeDirectory("tmp")
	assert.NoError(err)

	// Capture to stdout without gzip compression
	stdout := util.CaptureStdOut()
	err = app.ExportDB("", false)
	assert.NoError(err)
	output := stdout()
	assert.Contains(output, "Table structure for table `users`")

	// Try it with capture to stdout, validate contents.
	runTime()
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

		// Get files before start, as syncing can start immediately.
		if site.FilesTarballURL != "" {
			_, tarballPath, err := testcommon.GetCachedArchive(site.Name, "local-tarballs-files", "", site.FilesTarballURL)
			assert.NoError(err)
			err = app.ImportFiles(tarballPath, "")
			assert.NoError(err)
		}

		// Running WriteConfig assures that settings.ddev.php gets written
		// so Drupal 8 won't try to set things unwriteable
		err = app.WriteConfig()
		assert.NoError(err)

		err = app.Start()
		assert.NoError(err)

		if site.DBTarURL != "" {
			_, cachedArchive, err := testcommon.GetCachedArchive(site.Name, site.Name+"_siteTarArchive", "", site.DBTarURL)
			assert.NoError(err)
			err = app.ImportDB(cachedArchive, "", false)
			assert.NoError(err)
		}

		startErr := app.StartAndWaitForSync(2)
		if startErr != nil {
			appLogs, getLogsErr := ddevapp.GetErrLogsFromApp(app, startErr)
			assert.NoError(getLogsErr)
			t.Fatalf("app.StartAndWaitForSync() failure err=%v; logs:\n=====\n%s\n=====\n", startErr, appLogs)
		}

		// Test static content.
		_, _ = testcommon.EnsureLocalHTTPContent(t, app.GetHTTPURL()+site.Safe200URIWithExpectation.URI, site.Safe200URIWithExpectation.Expect)
		// Test dynamic php + database content.
		rawurl := app.GetHTTPURL() + site.DynamicURI.URI
		body, resp, err := testcommon.GetLocalHTTPResponse(t, rawurl, 60)
		assert.NoError(err, "GetLocalHTTPResponse returned err on project=%s rawurl %s, resp=%v: %v", site.Name, rawurl, resp, err)
		if err != nil && strings.Contains(err.Error(), "container ") {
			logs, err := ddevapp.GetErrLogsFromApp(app, err)
			assert.NoError(err)
			t.Fatalf("Logs after GetLocalHTTPResponse: %s", logs)
		}
		assert.Contains(body, site.DynamicURI.Expect, "expected %s on project %s", site.DynamicURI.Expect, site.Name)

		// Load an image from the files section
		if site.FilesImageURI != "" {
			_, resp, err := testcommon.GetLocalHTTPResponse(t, app.GetHTTPURL()+site.FilesImageURI)
			assert.NoError(err, "failed ImageURI response on project %s: %v", site.Name, err)
			assert.Equal("image/jpeg", resp.Header["Content-Type"][0])
		}

		// We don't want all the projects running at once.
		err = app.Down(true, false)
		assert.NoError(err)

		runTime()
		switchDir()
	}
}

// TestDdevRestoreSnapshot tests creating a snapshot and reverting to it. This runs with Mariadb 10.2
func TestDdevRestoreSnapshot(t *testing.T) {
	assert := asrt.New(t)
	testDir, _ := os.Getwd()
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
		require.NoError(t, err)
	}

	switchDir := site.Chdir()
	testcommon.ClearDockerEnv()

	err = app.Init(site.Dir)
	require.NoError(t, err)

	// Try using php72 to avoid SIGBUS failures after restore.
	app.PHPVersion = ddevapp.PHP72

	// First do regular start, which is good enough to get us to an ImportDB()
	err = app.Start()
	require.NoError(t, err)

	err = app.ImportDB(d7testerTest1Dump, "", false)
	require.NoError(t, err, "Failed to app.ImportDB path: %s err: %v", d7testerTest1Dump, err)

	err = app.StartAndWaitForSync(2)
	require.NoError(t, err, "app.Start() failed on site %s, err=%v", site.Name, err)

	resp, ensureErr := testcommon.EnsureLocalHTTPContent(t, app.GetHTTPURL(), "d7 tester test 1 has 1 node", 45)
	assert.NoError(ensureErr)
	if ensureErr != nil && strings.Contains(ensureErr.Error(), "container failed") {
		logs, err := ddevapp.GetErrLogsFromApp(app, ensureErr)
		assert.NoError(err)
		t.Fatalf("container failed: logs:\n=======\n%s\n========\n", logs)
	}
	if ensureErr != nil && resp.StatusCode != 200 {
		logs, err := app.CaptureLogs("web", false, "")
		assert.NoError(err)
		t.Fatalf("EnsureLocalHTTPContent received %d. Resp=%v, web logs=\n========\n%s\n=========\n", resp.StatusCode, resp, logs)
	}

	// Make a snapshot of d7 tester test 1
	backupsDir := filepath.Join(app.GetConfigPath(""), "db_snapshots")
	snapshotName, err := app.SnapshotDatabase("d7testerTest1")
	assert.NoError(err)
	assert.EqualValues(snapshotName, "d7testerTest1")
	assert.True(fileutil.FileExists(filepath.Join(backupsDir, snapshotName, "xtrabackup_info")))

	err = app.ImportDB(d7testerTest2Dump, "", false)
	assert.NoError(err, "Failed to app.ImportDB path: %s err: %v", d7testerTest2Dump, err)
	_, _ = testcommon.EnsureLocalHTTPContent(t, app.GetHTTPURL(), "d7 tester test 2 has 2 nodes", 45)

	snapshotName, err = app.SnapshotDatabase("d7testerTest2")
	assert.NoError(err)
	assert.EqualValues(snapshotName, "d7testerTest2")
	assert.True(fileutil.FileExists(filepath.Join(backupsDir, snapshotName, "xtrabackup_info")))

	err = app.RestoreSnapshot("d7testerTest1")
	assert.NoError(err)
	_, _ = testcommon.EnsureLocalHTTPContent(t, app.GetHTTPURL(), "d7 tester test 1 has 1 node", 45)
	err = app.RestoreSnapshot("d7testerTest2")
	assert.NoError(err)

	body, resp, err := testcommon.GetLocalHTTPResponse(t, app.GetHTTPURL(), 45)
	assert.NoError(err, "GetLocalHTTPResponse returned err on rawurl %s: %v", app.GetHTTPURL(), err)
	assert.Contains(body, "d7 tester test 2 has 2 nodes")
	if err != nil {
		t.Logf("resp after timeout: %v", resp)
		stdout := util.CaptureUserOut()
		err = app.Logs("web", false, false, "")
		assert.NoError(err)
		out := stdout()
		t.Logf("web container logs after timeout: %s", out)
	}

	// Attempt a restore with a pre-mariadb_10.2 snapshot. It should fail and give a link.
	oldSnapshotTarball, err := filepath.Abs(filepath.Join(testDir, "testdata", "restore_snapshot", "d7tester_test_1.snapshot_mariadb_10_1.tgz"))
	assert.NoError(err)

	err = archive.Untar(oldSnapshotTarball, filepath.Join(site.Dir, ".ddev", "db_snapshots", "oldsnapshot"), "")
	assert.NoError(err)
	err = app.RestoreSnapshot("oldsnapshot")
	assert.Error(err)
	assert.Contains(err.Error(), "is not compatible")

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
	site := TestSites[0]
	switchDir := site.Chdir()
	runTime := testcommon.TimeTrack(time.Now(), fmt.Sprintf("%s TestWritableFilesDirectory", site.Name))

	testcommon.ClearDockerEnv()
	err := app.Init(site.Dir)
	assert.NoError(err)

	err = app.StartAndWaitForSync(0)
	assert.NoError(err)

	uploadDir := app.GetUploadDir()
	assert.NotEmpty(uploadDir)

	// Use exec to touch a file in the container and see what the result is. Make sure it comes out with ownership
	// making it writeable on the host.
	filename := fileutil.RandomFilenameBase()
	dirname := fileutil.RandomFilenameBase()
	// Use path.Join for items on th container (linux) and filepath.Join for items on the host.
	inContainerDir := path.Join(uploadDir, dirname)
	onHostDir := filepath.Join(app.Docroot, inContainerDir)

	// The container execution directory is dependent on the app type
	switch app.Type {
	case ddevapp.AppTypeWordPress, ddevapp.AppTypeTYPO3, ddevapp.AppTypePHP:
		inContainerDir = path.Join(app.Docroot, inContainerDir)
	}

	inContainerRelativePath := path.Join(inContainerDir, filename)
	onHostRelativePath := path.Join(onHostDir, filename)

	err = os.MkdirAll(onHostDir, 0775)
	assert.NoError(err)
	// Create a file in the directory to make sure it syncs
	f, err := os.OpenFile(filepath.Join(onHostDir, "junk.txt"), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0755)
	assert.NoError(err)
	_ = f.Close()
	ddevapp.WaitForSync(app, 5)

	_, _, createFileErr := app.Exec(&ddevapp.ExecOpts{
		Service: "web",
		Cmd:     []string{"sh", "-c", "echo 'content created inside container\n' >" + inContainerRelativePath},
	})
	assert.NoError(createFileErr)
	if app.WebcacheEnabled && createFileErr != nil {
		syncLogs, err := app.CaptureLogs("bgsync", false, "")
		assert.NoError(err)

		// ls -lR on host
		onHostList, err := exec.RunCommand("ls", []string{"-lR", filepath.Dir(uploadDir)})
		assert.NoError(err)

		// ls -lR in container
		inContainerList, _, err := app.Exec(&ddevapp.ExecOpts{
			Service: "web",
			Cmd:     []string{"ls", "-lR", filepath.Dir(uploadDir)},
		})
		assert.NoError(err)
		t.Fatalf("Unable to create file %s inside container; onHost ls=\n====\n%s\n====\ninContainer ls=\n======\n%s\n=====\nContainer Sync logs=\n=======\n%s\n===========\n", inContainerRelativePath, onHostList, inContainerList, syncLogs)
	}

	ddevapp.WaitForSync(app, 5)

	// Now try to append to the file on the host.
	// os.OpenFile() for append here fails if the file does not already exist.
	f, err = os.OpenFile(onHostRelativePath, os.O_APPEND|os.O_WRONLY, 0660)
	assert.NoError(err)
	_, err = f.WriteString("this addition to the file was added on the host side")
	assert.NoError(err)
	_ = f.Close()

	// Create a file on the host and see what the result is. Make sure we can not append/write to it in the container.
	filename = fileutil.RandomFilenameBase()
	dirname = fileutil.RandomFilenameBase()
	inContainerDir = path.Join(uploadDir, dirname)
	onHostDir = filepath.Join(app.Docroot, inContainerDir)
	// The container execution directory is dependent on the app type
	switch app.Type {
	case ddevapp.AppTypeWordPress, ddevapp.AppTypeTYPO3, ddevapp.AppTypePHP:
		inContainerDir = path.Join(app.Docroot, inContainerDir)
	}

	inContainerRelativePath = path.Join(inContainerDir, filename)
	onHostRelativePath = filepath.Join(onHostDir, filename)

	err = os.MkdirAll(onHostDir, 0775)
	assert.NoError(err)

	f, err = os.OpenFile(onHostRelativePath, os.O_CREATE|os.O_RDWR, 0660)
	assert.NoError(err)
	_, err = f.WriteString("this base content was inserted on the host side\n")
	assert.NoError(err)
	_ = f.Close()

	ddevapp.WaitForSync(app, 5)

	// if the file exists, add to it. We don't want to add if it's not already there.
	_, _, err = app.Exec(&ddevapp.ExecOpts{
		Service: "web",
		Cmd:     []string{"sh", "-c", "if [ -f " + inContainerRelativePath + " ]; then echo 'content added inside container\n' >>" + inContainerRelativePath + "; fi"},
	})
	assert.NoError(err)
	// grep the file for both the content added on host and that added in container.
	_, _, err = app.Exec(&ddevapp.ExecOpts{
		Service: "web",
		Cmd:     []string{"sh", "-c", "grep 'base content was inserted on the host' " + inContainerRelativePath + "&& grep 'content added inside container' " + inContainerRelativePath},
	})
	assert.NoError(err)

	err = app.Down(true, false)
	assert.NoError(err)

	runTime()
	switchDir()
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
		startErr := app.StartAndWaitForSync(0)
		if startErr != nil {
			logs, err := ddevapp.GetErrLogsFromApp(app, startErr)
			assert.NoError(err)
			t.Fatalf("app.StartAndWaitForSync() failed err=%v, logs from broken container:\n=======\n%s\n========\n", startErr, logs)
		}

		out, _, err := app.Exec(&ddevapp.ExecOpts{
			Service: "web",
			Cmd:     []string{"pwd"},
		})
		assert.NoError(err)
		assert.Contains(out, "/var/www/html")

		out, _, err = app.Exec(&ddevapp.ExecOpts{
			Service: "web",
			Dir:     "/usr/local",
			Cmd:     []string{"pwd"},
		})
		assert.NoError(err)
		assert.Contains(out, "/usr/local")

		_, _, err = app.Exec(&ddevapp.ExecOpts{
			Service: "db",
			Cmd:     []string{"mysql", "-e", "DROP DATABASE db;"},
		})
		assert.NoError(err)
		_, _, err = app.Exec(&ddevapp.ExecOpts{
			Service: "db",
			Cmd:     []string{"mysql", "information_schema", "-e", "CREATE DATABASE db;"},
		})
		assert.NoError(err)

		switch app.GetType() {
		case ddevapp.AppTypeDrupal6:
			fallthrough
		case ddevapp.AppTypeDrupal7:
			fallthrough
		case ddevapp.AppTypeDrupal8:
			out, _, err = app.Exec(&ddevapp.ExecOpts{
				Service: "web",
				Cmd:     []string{"drush", "status"},
			})
			assert.NoError(err)
			assert.Regexp("PHP configuration[ :]*/etc/php/[0-9].[0-9]/fpm/php.ini", out)
		case ddevapp.AppTypeWordPress:
			out, _, err = app.Exec(&ddevapp.ExecOpts{
				Service: "web",
				Cmd:     []string{"wp", "--info"},
			})
			assert.NoError(err)
			assert.Regexp("/etc/php.*/php.ini", out)
		}

		err = app.Down(true, false)
		assert.NoError(err)

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

	site := TestSites[0]
	switchDir := site.Chdir()
	runTime := testcommon.TimeTrack(time.Now(), fmt.Sprintf("%s DdevLogs", site.Name))

	err := app.Init(site.Dir)
	assert.NoError(err)

	startErr := app.StartAndWaitForSync(0)
	if startErr != nil {
		logs, err := ddevapp.GetErrLogsFromApp(app, startErr)
		assert.NoError(err)
		t.Fatalf("app.Start failed, err=%v, logs=\n========\n%s\n===========\n", startErr, logs)
	}

	stdout := util.CaptureUserOut()
	err = app.Logs("web", false, false, "")
	assert.NoError(err)
	out := stdout()
	assert.Contains(out, "Server started")

	stdout = util.CaptureUserOut()
	err = app.Logs("db", false, false, "")
	assert.NoError(err)
	out = stdout()
	assert.Contains(out, "MySQL init process done. Ready for start up.")

	// Test that we can get logs when project is stopped also
	err = app.Stop()
	assert.NoError(err)

	stdout = util.CaptureUserOut()
	err = app.Logs("web", false, false, "")
	assert.NoError(err)
	out = stdout()
	assert.Contains(out, "Server started")

	stdout = util.CaptureUserOut()
	err = app.Logs("db", false, false, "")
	assert.NoError(err)
	out = stdout()
	assert.Contains(out, "MySQL init process done. Ready for start up.")

	err = app.Down(true, false)
	assert.NoError(err)

	runTime()
	switchDir()
}

// TestProcessHooks tests execution of commands defined in config.yaml
func TestProcessHooks(t *testing.T) {
	assert := asrt.New(t)

	site := TestSites[0]
	cleanup := site.Chdir()
	runTime := testcommon.TimeTrack(time.Now(), fmt.Sprintf("%s ProcessHooks", site.Name))

	testcommon.ClearDockerEnv()
	app, err := ddevapp.NewApp(site.Dir, ddevapp.ProviderDefault)
	assert.NoError(err)
	err = app.StartAndWaitForSync(0)
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

	stdout := util.CaptureUserOut()
	err = app.ProcessHooks("hook-test")
	assert.NoError(err)

	// Ignore color in putput, can be different in different OS's
	out := vtclean.Clean(stdout(), false)

	assert.Contains(out, "hook-test exec command succeeded, output below ---\n/usr/local/bin/composer")
	assert.Contains(out, "--- Running host command: echo something ---\nRunning Command Command=echo something\nsomething")

	err = app.Down(true, false)
	assert.NoError(err)

	runTime()
	cleanup()
}

// TestDdevStop tests the functionality that is called when "ddev stop" is executed
func TestDdevStop(t *testing.T) {
	assert := asrt.New(t)

	app := &ddevapp.DdevApp{}

	site := TestSites[0]
	switchDir := site.Chdir()
	runTime := testcommon.TimeTrack(time.Now(), fmt.Sprintf("%s DdevStop", site.Name))

	testcommon.ClearDockerEnv()
	err := app.Init(site.Dir)
	assert.NoError(err)
	err = app.StartAndWaitForSync(0)
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
	err = app.Down(true, false)
	assert.NoError(err)

	runTime()
	switchDir()
}

// TestDdevStopMissingDirectory tests that the 'ddev stop' command works properly on sites with missing directories or ddev configs.
func TestDdevStopMissingDirectory(t *testing.T) {
	assert := asrt.New(t)

	site := TestSites[0]
	testcommon.ClearDockerEnv()
	app := &ddevapp.DdevApp{}
	err := app.Init(site.Dir)
	assert.NoError(err)

	startErr := app.StartAndWaitForSync(0)
	if startErr != nil {
		logs, err := ddevapp.GetErrLogsFromApp(app, startErr)
		assert.NoError(err)
		t.Fatalf("app.StartAndWaitForSync failed err=%v logs from broken container: \n=======\n%s\n========\n", startErr, logs)
	}

	tempPath := testcommon.CreateTmpDir("site-copy")
	siteCopyDest := filepath.Join(tempPath, "site")
	defer removeAllErrCheck(tempPath, assert)

	// Move the site directory to a temp location to mimic a missing directory.
	err = os.Rename(site.Dir, siteCopyDest)
	assert.NoError(err)

	err = app.Stop()
	assert.Error(err)
	assert.Contains(err.Error(), "If you would like to continue using ddev to manage this project please restore your files to that directory.")
	// Move the site directory back to its original location.
	err = os.Rename(siteCopyDest, site.Dir)
	assert.NoError(err)
	err = app.Down(true, false)
	assert.NoError(err)
}

// TestDdevDescribe tests that the describe command works properly on a running
// and also a stopped project.
func TestDdevDescribe(t *testing.T) {
	assert := asrt.New(t)
	app := &ddevapp.DdevApp{}

	site := TestSites[0]
	switchDir := site.Chdir()

	testcommon.ClearDockerEnv()
	err := app.Init(site.Dir)
	assert.NoError(err)

	err = app.StartAndWaitForSync(0)

	// If we have a problem starting, get the container logs and output.
	if err != nil {
		stdout := util.CaptureUserOut()
		logsErr := app.Logs("web", false, false, "")
		assert.NoError(logsErr)
		out := stdout()

		healthcheck, inspectErr := exec.RunCommandPipe("bash", []string{"-c", fmt.Sprintf("docker inspect ddev-%s-web|jq -r '.[0].State.Health.Log[-1]'", app.Name)})
		assert.NoError(inspectErr)

		assert.NoError(err, "app.Start(%s) failed: %v, \nweb container healthcheck='%s', \n=== web container logs=\n%s\n=== END web container logs ===", site.Name, err, healthcheck, out)
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
	err = app.Down(true, false)
	assert.NoError(err)
	switchDir()
}

// TestDdevDescribeMissingDirectory tests that the describe command works properly on sites with missing directories or ddev configs.
func TestDdevDescribeMissingDirectory(t *testing.T) {
	assert := asrt.New(t)
	site := TestSites[0]
	tempPath := testcommon.CreateTmpDir("site-copy")
	siteCopyDest := filepath.Join(tempPath, "site")
	defer removeAllErrCheck(tempPath, assert)

	app := &ddevapp.DdevApp{}
	err := app.Init(site.Dir)
	assert.NoError(err)
	startErr := app.StartAndWaitForSync(0)
	if startErr != nil {
		logs, err := ddevapp.GetErrLogsFromApp(app, startErr)
		assert.NoError(err)
		t.Fatalf("app.StartAndWaitForSync failed err=%v logs from broken container: \n=======\n%s\n========\n", startErr, logs)
	}
	// Move the site directory to a temp location to mimick a missing directory.
	err = os.Rename(site.Dir, siteCopyDest)
	assert.NoError(err)

	desc, err := app.Describe()
	assert.NoError(err)
	assert.Contains(desc["status"], ddevapp.SiteDirMissing, "Status did not include the phrase '%s' when describing a site with missing directories.", ddevapp.SiteDirMissing)
	// Move the site directory back to its original location.
	err = os.Rename(siteCopyDest, site.Dir)
	assert.NoError(err)
	err = app.Down(true, false)
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

		if app.SiteStatus() == ddevapp.SiteRunning || app.SiteStatus() == ddevapp.SiteStopped {
			err = app.Down(true, false)
			assert.NoError(err)
		}

		switchDir()
	}

	// Now start one site, it's hard to get router to behave without one site.
	site := TestSites[0]
	testcommon.ClearDockerEnv()

	err := app.Init(site.Dir)
	assert.NoError(err)
	startErr := app.StartAndWaitForSync(5)
	if startErr != nil {
		appLogs, getLogsErr := ddevapp.GetErrLogsFromApp(app, startErr)
		assert.NoError(getLogsErr)
		t.Fatalf("app.StartAndWaitForSync() failure; err=%v logs:\n=====\n%s\n=====\n", startErr, appLogs)
	}

	app, err = ddevapp.GetActiveApp(site.Name)
	if err != nil {
		t.Fatalf("Failed to GetActiveApp(%s), err:%v", site.Name, err)
	}
	startErr = app.StartAndWaitForSync(5)
	if startErr != nil {
		appLogs, getLogsErr := ddevapp.GetErrLogsFromApp(app, startErr)
		assert.NoError(getLogsErr)
		t.Fatalf("app.StartAndWaitForSync() failure err=%v logs:\n=====\n%s\n=====\n", startErr, appLogs)
	}

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
	err = app.Down(true, false)
	assert.NoError(err)
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

	startErr := app.StartAndWaitForSync(5)
	//nolint: errcheck
	defer app.Down(true, false)
	if startErr != nil {
		appLogs, getLogsErr := ddevapp.GetErrLogsFromApp(app, startErr)
		assert.NoError(getLogsErr)
		t.Fatalf("app.StartAndWaitForSync failure; err=%v, logs:\n=====\n%s\n=====\n", startErr, appLogs)
	}
	// Setup by creating temp directory and nesting a folder for our site.
	tempPath := testcommon.CreateTmpDir("site-copy")
	siteCopyDest := filepath.Join(tempPath, "site")

	//nolint: errcheck
	defer os.RemoveAll(tempPath)
	//nolint: errcheck
	defer revertDir()
	// Move the site directory back to its original location.
	//nolint: errcheck
	defer os.Rename(siteCopyDest, site.Dir)

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
	assert.Equal(0, len(apps), "Expected to find no apps but found %d apps=%v", len(apps), apps)
}

// TestRouterNotRunning ensures the router is shut down after all sites are stopped.
// This depends on TestGetAppsEmpty() having shut everything down.
func TestRouterNotRunning(t *testing.T) {
	assert := asrt.New(t)
	containers, err := dockerutil.GetDockerContainers(false)
	assert.NoError(err)

	for _, container := range containers {
		assert.NotEqual("ddev-router", dockerutil.ContainerName(container), "ddev-router was not supposed to be running but it was")
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

	app, err := ddevapp.NewApp(testDir, ddevapp.ProviderDefault)
	assert.NoError(err)
	app.Name = "junk"
	app.Type = ddevapp.AppTypeDrupal7
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

	app, err := ddevapp.NewApp(appDir, ddevapp.ProviderDefault)
	assert.NoError(err)
	app.Name = "proj"
	app.Type = ddevapp.AppTypePHP

	expectations := []URLRedirectExpectations{
		{"https", "/subdir", "/subdir/"},
		{"https", "/redir_abs.php", "/landed.php"},
		{"https", "/redir_relative.php", "/landed.php"},
		{"http", "/subdir", "/subdir/"},
		{"http", "/redir_abs.php", "/landed.php"},
		{"http", "/redir_relative.php", "/landed.php"},
	}

	for _, webserverType := range []string{ddevapp.WebserverNginxFPM, ddevapp.WebserverApacheFPM, ddevapp.WebserverApacheCGI} {
		app.WebserverType = webserverType
		err = app.WriteConfig()
		assert.NoError(err)

		// Do a start on the configured site.
		app, err = ddevapp.GetActiveApp("")
		assert.NoError(err)
		//nolint: errcheck
		defer app.Down(true, false)
		startErr := app.StartAndWaitForSync(30)
		if startErr != nil {
			appLogs, getLogsErr := ddevapp.GetErrLogsFromApp(app, startErr)
			assert.NoError(getLogsErr)
			// Get healthcheck status on bgsync container
			healthcheck, inspectErr := exec.RunCommandPipe("bash", []string{"-c", fmt.Sprintf("docker inspect ddev-%s-bgsync|jq -r '.[0].State.Health.Log[-1]'", app.Name)})
			assert.NoError(inspectErr)

			t.Fatalf("app.StartAndWaitForSync failure; err=%v \n===== container logs ===\n%s\n===== bgsync health info ===\n%s\n========\n", startErr, appLogs, healthcheck)
		}

		// Test for directory redirects under https and http
		for _, parts := range expectations {

			reqURL := parts.scheme + "://" + app.GetHostname() + parts.uri
			_, resp, err := testcommon.GetLocalHTTPResponse(t, reqURL)
			assert.Error(err)
			assert.NotNil(resp, "resp was nil for webserver_type=%s url=%s", webserverType, reqURL)
			if resp != nil {
				locHeader := resp.Header.Get("Location")

				expectedRedirect := parts.expectedRedirectURI
				// However, if we're hitting redir_abs.php (or apache hitting directory), the redirect will be the whole url.
				if strings.Contains(parts.uri, "redir_abs.php") || webserverType != ddevapp.WebserverNginxFPM {
					expectedRedirect = parts.scheme + "://" + app.GetHostname() + parts.expectedRedirectURI
				}
				// Except the php relative redirect is always relative.
				if strings.Contains(parts.uri, "redir_relative.php") {
					expectedRedirect = parts.expectedRedirectURI
				}

				assert.EqualValues(locHeader, expectedRedirect, "For webserver_type %s url %s expected redirect %s != actual %s", webserverType, reqURL, expectedRedirect, locHeader)
			}
		}
		err = app.Down(true, false)
		assert.NoError(err)
	}

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
	require.NotEmpty(t, files)
	require.Equal(t, files[0], filepath.Join(app.AppConfDir(), "docker-compose.yaml"))
	require.Equal(t, files[len(files)-1], filepath.Join(app.AppConfDir(), "docker-compose.override.yaml"))

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

		err = app.StartAndWaitForSync(0)
		require.NoError(t, err)

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
		require.NotEmpty(t, webContainer)

		dockerIP, err := dockerutil.GetDockerIP()
		require.NoError(t, err)

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

		err = app.Down(true, false)
		assert.NoError(err)

		runTime()
	}
}

// TestWebserverType checks that webserver_type:apache-cgi or apache-fpm does the right thing
func TestWebserverType(t *testing.T) {
	assert := asrt.New(t)

	for _, site := range TestSites {
		runTime := testcommon.TimeTrack(time.Now(), fmt.Sprintf("%s TestWebserverType", site.Name))

		app := new(ddevapp.DdevApp)

		err := app.Init(site.Dir)
		assert.NoError(err)

		// Copy our phpinfo into the docroot of testsite.
		pwd, err := os.Getwd()
		assert.NoError(err)
		err = fileutil.CopyFile(filepath.Join(pwd, "testdata", "servertype.php"), filepath.Join(app.AppRoot, app.Docroot, "servertype.php"))

		assert.NoError(err)
		for _, app.WebserverType = range []string{ddevapp.WebserverApacheFPM, ddevapp.WebserverApacheCGI, ddevapp.WebserverNginxFPM} {

			err = app.WriteConfig()
			assert.NoError(err)

			testcommon.ClearDockerEnv()

			startErr := app.StartAndWaitForSync(30)
			//nolint: errcheck
			defer app.Down(true, false)
			if startErr != nil {
				appLogs, getLogsErr := ddevapp.GetErrLogsFromApp(app, startErr)
				assert.NoError(getLogsErr)
				t.Fatalf("app.StartAndWaitForSync failure; err=%v, logs:\n=====\n%s\n=====\n", startErr, appLogs)
			}
			out, resp, err := testcommon.GetLocalHTTPResponse(t, app.GetWebContainerDirectURL()+"/servertype.php")
			assert.NoError(err)

			expectedServerType := "Apache/2"
			if app.WebserverType == ddevapp.WebserverNginxFPM {
				expectedServerType = "nginx"
			}
			require.NotEmpty(t, resp.Header["Server"][0])
			assert.Contains(resp.Header["Server"][0], expectedServerType, "Server header for project=%s, app.WebserverType=%s should be %s", app.Name, app.WebserverType, expectedServerType)
			assert.Contains(out, expectedServerType, "For app.WebserverType=%s phpinfo expected servertype.php to show %s", app.WebserverType, expectedServerType)
			err = app.Down(true, false)
			assert.NoError(err)
		}

		// Set the apptype back to whatever the default was so we don't break any following tests.
		testVar := os.Getenv("DDEV_TEST_WEBSERVER_TYPE")
		if testVar != "" {
			app.WebserverType = testVar
			err = app.WriteConfig()
			assert.NoError(err)
		}

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

	dataDir := filepath.Join(globalconfig.GetGlobalDdevDir(), app.Name, "mysql")

	// Remove any existing dataDir or migration backups
	if fileutil.FileExists(dataDir) {
		err = os.RemoveAll(dataDir)
		assert.NoError(err)
	}
	if fileutil.FileExists(dataDir + "_migrated.bak") {
		err = os.RemoveAll(dataDir + "_migrated.bak")
		assert.NoError(err)
	}

	// Untar the to-migrate db into old-style dataDir (~/.ddev/projectname/mysql)
	err = os.MkdirAll(dataDir, 0755)
	require.NoError(t, err)
	err = archive.Untar(dbMigrationTarball, dataDir, "")
	require.NoError(t, err)
	defer os.RemoveAll(dataDir)

	_, err = app.CreateSettingsFile()
	assert.NoError(err)

	// app.Start() will discover the mysql directory and migrate it to a snapshot.
	err = app.Start()
	assert.Error(err)
	assert.Contains(err.Error(), "it is not possible to migrate bind-mounted")

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
			_, _ = testcommon.EnsureLocalHTTPContent(t, app.GetHTTPURL()+site.Safe200URIWithExpectation.URI, site.Safe200URIWithExpectation.Expect)
			_, _ = testcommon.EnsureLocalHTTPContent(t, app.GetHTTPSURL()+site.Safe200URIWithExpectation.URI, site.Safe200URIWithExpectation.Expect)

			// Ensure that we can access the same URL from within the web container (via router)
			var out string
			out, _, err = app.Exec(&ddevapp.ExecOpts{
				Service: "web",
				Cmd:     []string{"curl", "-sk", app.GetHTTPURL() + site.Safe200URIWithExpectation.URI},
			})
			assert.NoError(err)
			assert.Contains(out, site.Safe200URIWithExpectation.Expect)

			out, _, err = app.Exec(&ddevapp.ExecOpts{
				Service: "web",
				Cmd:     []string{"curl", "-sk", app.GetHTTPSURL() + site.Safe200URIWithExpectation.URI},
			})
			assert.NoError(err)
			assert.Contains(out, site.Safe200URIWithExpectation.Expect)
		}

		// Set the ports back to the default was so we don't break any following tests.
		app.RouterHTTPSPort = "443"
		app.RouterHTTPPort = "80"
		err = app.WriteConfig()
		assert.NoError(err)
		err = app.Down(true, false)
		assert.NoError(err)

		runTime()
	}
}

// TestCaptureLogs checks that app.CaptureLogs() works
func TestCaptureLogs(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping TestCaptureLogs on windows, it sometimes hangs")
	}
	assert := asrt.New(t)

	site := TestSites[0]
	runTime := testcommon.TimeTrack(time.Now(), fmt.Sprintf("%s CaptureLogs", site.Name))

	app := new(ddevapp.DdevApp)

	err := app.Init(site.Dir)
	assert.NoError(err)
	err = app.Start()
	assert.NoError(err)

	logs, err := app.CaptureLogs("web", false, "100")
	assert.NoError(err)

	assert.Contains(logs, "INFO spawned")

	err = app.Down(true, false)
	assert.NoError(err)

	runTime()
}

// constructContainerName builds a container name given the type (web/db/dba) and the app
func constructContainerName(containerType string, app *ddevapp.DdevApp) (string, error) {
	container, err := app.FindContainerByType(containerType)
	if err != nil {
		return "", err
	}
	if container == nil {
		return "", fmt.Errorf("No container exists for containerType=%s app=%v", containerType, app)
	}
	name := dockerutil.ContainerName(*container)
	return name, nil
}

func removeAllErrCheck(path string, assert *asrt.Assertions) {
	err := os.RemoveAll(path)
	assert.NoError(err)
}
