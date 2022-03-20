package ddevapp_test

import (
	"bufio"
	"fmt"
	"net"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/drud/ddev/pkg/nodeps"
	"github.com/drud/ddev/pkg/version"

	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/stretchr/testify/require"

	"github.com/drud/ddev/pkg/archive"
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/testcommon"
	"github.com/drud/ddev/pkg/util"
	docker "github.com/fsouza/go-dockerclient"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	asrt "github.com/stretchr/testify/assert"
)

var (
	DdevBin   = "ddev"
	TestSites = []testcommon.TestSite{
		// 0: wordpress
		{
			Name:                          "TestPkgWordpress",
			SourceURL:                     "https://wordpress.org/wordpress-5.8.2.tar.gz",
			ArchiveInternalExtractionPath: "wordpress/",
			FilesTarballURL:               "https://github.com/drud/ddev_test_tarballs/releases/download/v1.1/wordpress5.8.2_files.tar.gz",
			DBTarURL:                      "https://github.com/drud/ddev_test_tarballs/releases/download/v1.1/wordpress5.8.2_db.sql.tar.gz",
			Docroot:                       "",
			Type:                          nodeps.AppTypeWordPress,
			Safe200URIWithExpectation:     testcommon.URIWithExpect{URI: "/readme.html", Expect: "Welcome. WordPress is a very special project to me."},
			DynamicURI:                    testcommon.URIWithExpect{URI: "/", Expect: "this post has a photo"},
			FilesImageURI:                 "/wp-content/uploads/2021/12/DSCF0436-randy-and-nancy-with-traditional-wedding-out-fit-2048x1536.jpg",
		},
		// 1: drupal8
		{
			Name:                          "TestPkgDrupal8",
			SourceURL:                     "https://ftp.drupal.org/files/projects/drupal-8.9.13.tar.gz",
			ArchiveInternalExtractionPath: "drupal-8.9.13/",
			FilesTarballURL:               "https://github.com/drud/ddev_test_tarballs/releases/download/v1.1/d8_umami.files.tar.gz",
			FilesZipballURL:               "https://github.com/drud/ddev_test_tarballs/releases/download/v1.1/d8_umami.files.zip",
			DBTarURL:                      "https://github.com/drud/ddev_test_tarballs/releases/download/v1.1/d8_umami.sql.tar.gz",
			DBZipURL:                      "https://github.com/drud/ddev_test_tarballs/releases/download/v1.1/d8_umami.sql.zip",
			FullSiteTarballURL:            "",
			Type:                          nodeps.AppTypeDrupal8,
			Docroot:                       "",
			Safe200URIWithExpectation:     testcommon.URIWithExpect{URI: "/README.txt", Expect: "Drupal is an open source content management platform"},
			DynamicURI:                    testcommon.URIWithExpect{URI: "/node/2", Expect: "Vegan chocolate and nut brownies"},
			FilesImageURI:                 "/sites/default/files/vegan-chocolate-nut-brownies.jpg",
		},
		// 2: drupal7
		{
			Name:                          "TestPkgDrupal7", // Drupal D7
			SourceURL:                     "https://ftp.drupal.org/files/projects/drupal-7.87.tar.gz",
			ArchiveInternalExtractionPath: "drupal-7.87/",
			FilesTarballURL:               "https://github.com/drud/ddev_test_tarballs/releases/download/v1.1/d7test-7.59.files.tar.gz",
			DBTarURL:                      "https://github.com/drud/ddev_test_tarballs/releases/download/v1.1/d7test-7.87-db.tar.gz",
			FullSiteTarballURL:            "",
			Docroot:                       "",
			Type:                          nodeps.AppTypeDrupal7,
			Safe200URIWithExpectation:     testcommon.URIWithExpect{URI: "/README.txt", Expect: "Drupal is an open source content management platform"},
			DynamicURI:                    testcommon.URIWithExpect{URI: "/node/1", Expect: "D7 test project, kittens edition"},
			FilesImageURI:                 "/sites/default/files/field/image/kittens-large.jpg",
			FullSiteArchiveExtPath:        "docroot/sites/default/files",
		},
		// 3: drupal6
		{
			Name:                          "TestPkgDrupal6",
			SourceURL:                     "https://ftp.drupal.org/files/projects/drupal-6.38.tar.gz",
			ArchiveInternalExtractionPath: "drupal-6.38/",
			DBTarURL:                      "https://github.com/drud/ddev_test_tarballs/releases/download/v1.1/drupal6.38_db.tar.gz",
			FullSiteTarballURL:            "",
			FilesTarballURL:               "https://github.com/drud/ddev_test_tarballs/releases/download/v1.1/drupal6_files.tar.gz",
			Docroot:                       "",
			Type:                          nodeps.AppTypeDrupal6,
			Safe200URIWithExpectation:     testcommon.URIWithExpect{URI: "/CHANGELOG.txt", Expect: "Drupal 6.38, 2016-02-24"},
			DynamicURI:                    testcommon.URIWithExpect{URI: "/node/2", Expect: "This is a story. The story is somewhat shaky"},
			FilesImageURI:                 "/sites/default/files/garland_logo.jpg",
		},
		// 4: backdrop
		{
			Name:                          "TestPkgBackdrop",
			SourceURL:                     "https://github.com/backdrop/backdrop/archive/1.11.0.tar.gz",
			ArchiveInternalExtractionPath: "backdrop-1.11.0/",
			DBTarURL:                      "https://github.com/drud/ddev_test_tarballs/releases/download/v1.1/backdrop_db.11.0.tar.gz",
			FilesTarballURL:               "https://github.com/drud/ddev_test_tarballs/releases/download/v1.1/backdrop_files.11.0.tar.gz",
			FullSiteTarballURL:            "",
			Docroot:                       "",
			Type:                          nodeps.AppTypeBackdrop,
			Safe200URIWithExpectation:     testcommon.URIWithExpect{URI: "/README.md", Expect: "Backdrop is a full-featured content management system"},
			DynamicURI:                    testcommon.URIWithExpect{URI: "/posts/first-post-all-about-kittens", Expect: "Lots of kittens are a good thing"},
			FilesImageURI:                 "/files/styles/large/public/field/image/kittens-large.jpg",
		},
		// 5: typo3
		{
			Name:                          "TestPkgTypo3",
			SourceURL:                     "https://github.com/drud/typo3-v9-test/archive/v0.2.2.tar.gz",
			ArchiveInternalExtractionPath: "typo3-v9-test-0.2.2/",
			DBTarURL:                      "https://github.com/drud/ddev_test_tarballs/releases/download/v1.1/typo3_v9.5_introduction_db.tar.gz",
			FilesTarballURL:               "https://github.com/drud/ddev_test_tarballs/releases/download/v1.1/typo3_v9.5_introduction_files.tar.gz",
			FullSiteTarballURL:            "",
			Docroot:                       "public",
			Type:                          nodeps.AppTypeTYPO3,
			Safe200URIWithExpectation:     testcommon.URIWithExpect{URI: "/README.txt", Expect: "junk readme simply for reading"},
			DynamicURI:                    testcommon.URIWithExpect{URI: "/index.php?id=65", Expect: "Boxed Content"},
			FilesImageURI:                 "/fileadmin/introduction/images/streets/nikita-maru-70928.jpg",
		},
		// 6: magento1
		{
			Name:                          "testpkgmagento",
			SourceURL:                     "https://github.com/OpenMage/magento-lts/archive/refs/tags/v20.0.13.tar.gz",
			ArchiveInternalExtractionPath: "magento-lts-20.0.13/",
			DBTarURL:                      "https://github.com/drud/ddev_test_tarballs/releases/download/v1.1/TestPkgMagento_db_secure_url.tar.gz",
			FilesTarballURL:               "https://github.com/drud/ddev_test_tarballs/releases/download/v1.1/magento_upload_files.tgz",
			FullSiteTarballURL:            "",
			Docroot:                       "",
			Type:                          nodeps.AppTypeMagento,
			Safe200URIWithExpectation:     testcommon.URIWithExpect{URI: "/LICENSE.txt", Expect: `Open Software License ("OSL")`},
			DynamicURI:                    testcommon.URIWithExpect{URI: "/", Expect: "This is a demo store"},
			FilesImageURI:                 "/media/wrapping/Chrysanthemum.jpg",
		},
		// 7: magento2
		// Note that testpkgmagento2 code is enormous and makes this really, really slow.
		{
			Name: "testpkgmagento2",
			// echo "This is a junk" >pub/junk.txt && tar -czf .tarballs/testpkgmagento2_code_no_media.magento2.4_try_4.tgz --exclude=.ddev --exclude=var --exclude=pub/media --exclude=.tarballs --exclude=app/etc/env.php .
			SourceURL:                     "https://github.com/drud/ddev_test_tarballs/releases/download/v1.1/testpkgmagento2_code_no_media.magento2.4_try_4.tgz",
			ArchiveInternalExtractionPath: "",
			// ddev export-db --gzip=false --file=.tarballs/db.sql && tar -czf .tarballs/testpkgmagento2.magento2.4.db_try_4.tgz -C .tarballs db.sql
			DBTarURL: "https://github.com/drud/ddev_test_tarballs/releases/download/v1.1/testpkgmagento2.magento2.4.db_try_4.tgz",
			// tar -czf .tarballs/testpkgmagento2_files.magento2.4_try_4.tgz -C pub/media .
			FilesTarballURL:           "https://github.com/drud/ddev_test_tarballs/releases/download/v1.1/testpkgmagento2_files.magento2.4_try_4.tgz",
			FullSiteTarballURL:        "",
			Docroot:                   "pub",
			Type:                      nodeps.AppTypeMagento2,
			Safe200URIWithExpectation: testcommon.URIWithExpect{URI: "/junk.txt", Expect: `This is a junk`},
			DynamicURI:                testcommon.URIWithExpect{URI: "/index.php/unicycle.html", Expect: "Unicycle"},
			FilesImageURI:             "/media/catalog/product/r/a/randy_4th_of_july_unicycle.jpg",
		},
		// 8: drupal9
		{
			Name:                          "TestPkgDrupal9",
			SourceURL:                     "https://ftp.drupal.org/files/projects/drupal-9.1.4.tar.gz",
			ArchiveInternalExtractionPath: "drupal-9.1.4/",
			FilesTarballURL:               "https://github.com/drud/ddev_test_tarballs/releases/download/v1.1/d9_umami_files.tgz",
			FilesZipballURL:               "https://github.com/drud/ddev_test_tarballs/releases/download/v1.1/d9_umami_files.zip",
			DBTarURL:                      "https://github.com/drud/ddev_test_tarballs/releases/download/v1.1/d9_umami_sql.tar.gz",
			DBZipURL:                      "https://github.com/drud/ddev_test_tarballs/releases/download/v1.1/d9_umami.sql.zip",
			FullSiteTarballURL:            "",
			Type:                          nodeps.AppTypeDrupal9,
			Docroot:                       "",
			Safe200URIWithExpectation:     testcommon.URIWithExpect{URI: "/README.txt", Expect: "Drupal is an open source content management platform"},
			DynamicURI:                    testcommon.URIWithExpect{URI: "/node/1", Expect: "Deep mediterranean quiche"},
			FilesImageURI:                 "/sites/default/files/mediterranean-quiche-umami.jpg",
		},
		// 9: laravel
		{
			Name:                          "TestPkgLaravel",
			SourceURL:                     "https://github.com/drud/ddev_test_tarballs/releases/download/v1.1/ddev-lumen-testapp.tar.gz",
			ArchiveInternalExtractionPath: "ddev-lumen-testapp/",
			FilesTarballURL:               "",
			FilesZipballURL:               "",
			DBTarURL:                      "https://github.com/drud/ddev_test_tarballs/releases/download/v1.1/ddev-lumen-testapp_sql.tar.gz",
			DBZipURL:                      "https://github.com/drud/ddev_test_tarballs/releases/download/v1.1/ddev-lumen-testapp_sql.zip",
			FullSiteTarballURL:            "",
			Type:                          nodeps.AppTypeLaravel,
			Docroot:                       "public",
			Safe200URIWithExpectation:     testcommon.URIWithExpect{URI: "/", Expect: "Laravel Components"},
			DynamicURI:                    testcommon.URIWithExpect{URI: "/api/status-code/200", Expect: "indicates that the request has succeeded."},
			FilesImageURI:                 "/images/200.jpg",
		},
		// 10: shopware6
		{
			Name:                          "testpkgshopware6",
			SourceURL:                     "https://github.com/drud/ddev_test_tarballs/releases/download/v1.1/shopware6_code.tgz",
			ArchiveInternalExtractionPath: "",
			FilesTarballURL:               "https://github.com/drud/ddev_test_tarballs/releases/download/v1.1/shopware6_files.tgz",
			DBTarURL:                      "https://github.com/drud/ddev_test_tarballs/releases/download/v1.1/shopware6_db.tgz",
			FullSiteTarballURL:            "",
			Type:                          nodeps.AppTypeShopware6,
			Docroot:                       "public",
			Safe200URIWithExpectation:     testcommon.URIWithExpect{URI: "/maintenance.html", Expect: "Our website is currently undergoing maintenance"},
			DynamicURI:                    testcommon.URIWithExpect{URI: "/Main-product-with-properties/SWDEMO10007.1", Expect: "Main product with properties"},
			FilesImageURI:                 "/media/2f/b0/e2/1603218072/hemd_600x600.jpg",
		},
		// 11: php
		{
			Name:                          "TestPkgPHP",
			SourceURL:                     "https://github.com/drud/ddev-test-php-repo/archive/refs/tags/v1.1.0.tar.gz",
			ArchiveInternalExtractionPath: "ddev-test-php-repo-1.1.0/",
			FullSiteTarballURL:            "",
			FilesTarballURL:               "https://github.com/drud/ddev_test_tarballs/releases/download/v1.1/drupal6_files.tar.gz",
			Docroot:                       "",
			Type:                          nodeps.AppTypePHP,
			Safe200URIWithExpectation:     testcommon.URIWithExpect{URI: "/README.txt", Expect: "This is a simple readme."},
			DynamicURI:                    testcommon.URIWithExpect{URI: "/index.php", Expect: "This program makes use of the Zend Scripting Language Engine"},
		},
		// 12: drupal10
		{
			Name:                          "TestPkgDrupal10",
			SourceURL:                     "https://github.com/drud/ddev_test_tarballs/releases/download/v1.1/d10_code.tgz",
			ArchiveInternalExtractionPath: "",
			FilesTarballURL:               "https://github.com/drud/ddev_test_tarballs/releases/download/v1.1/d10_files.tgz",
			DBTarURL:                      "https://github.com/drud/ddev_test_tarballs/releases/download/v1.1/d10.sql.tar.gz",
			FullSiteTarballURL:            "",
			Type:                          nodeps.AppTypeDrupal10,
			Docroot:                       "",
			Safe200URIWithExpectation:     testcommon.URIWithExpect{URI: "/README.md", Expect: "Drupal is an open source content management platform"},
			DynamicURI:                    testcommon.URIWithExpect{URI: "/", Expect: "Welcome to d10"},
		},
	}

	FullTestSites = TestSites
)

func init() {
	// Make sets DDEV_BINARY_FULLPATH when building the executable
	if os.Getenv("DDEV_BINARY_FULLPATH") != "" {
		DdevBin = os.Getenv("DDEV_BINARY_FULLPATH")
	}
}

func TestMain(m *testing.M) {
	output.LogSetUp()

	// Since this may be first time ddev has been used, we need the
	// ddev network available.
	dockerutil.EnsureDdevNetwork()

	// Avoid having sudo try to add to /etc/hosts.
	// This is normally done by Testsite.Prepare()
	_ = os.Setenv("DDEV_NONINTERACTIVE", "true")

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
	err := globalconfig.ReadGlobalConfig()
	if err != nil {
		log.Fatalf("could not read globalconfig: %v", err)
	}

	for i, site := range TestSites {
		app := &ddevapp.DdevApp{Name: site.Name}
		_ = app.Stop(true, false)
		_ = globalconfig.RemoveProjectInfo(site.Name)

		err := TestSites[i].Prepare()
		if err != nil {
			log.Fatalf("Prepare() failed on TestSite.Prepare() site=%s, err=%v", TestSites[i].Name, err)
		}

		switchDir := TestSites[i].Chdir()

		testcommon.ClearDockerEnv()

		app = &ddevapp.DdevApp{}
		err = app.Init(TestSites[i].Dir)
		if err != nil {
			testRun = -1
			log.Errorf("TestMain startup: app.Init() failed on site %s in dir %s, err=%v", TestSites[i].Name, TestSites[i].Dir, err)
			continue
		}
		err = ddevapp.PrepDdevDirectory(app.AppRoot)
		if err != nil {
			testRun = -1
			log.Errorf("TestMain startup: ddevapp.PrepDdevDirectory() failed on site %s in dir %s, err=%v", TestSites[i].Name, TestSites[i].Dir, err)
			continue
		}
		// Use ddev binary here because just app.WriteConfig() doesn't
		// populate the project .ddev
		err = ddevapp.PopulateExamplesCommandsHomeadditions(app.Name)
		if err != nil {
			testRun = -1
			log.Errorf("TestMain startup: ddevapp.PopulateExamplesCommandsHomeadditions() failed on site %s in dir %s, err=%v", TestSites[i].Name, TestSites[i].Dir, err)
			continue
		}
		err = app.WriteConfig()
		if err != nil {
			testRun = -1
			log.Errorf("TestMain startup: app.WriteConfig() failed on site %s in dir %s, err=%v", TestSites[i].Name, TestSites[i].Dir, err)
			continue
		}
		// Pre-download any images we may need, just to get them out of the way so they don't clutter tests
		_, err = exec.RunHostCommand("sh", "-c", fmt.Sprintf("%s debug download-images >/dev/null", DdevBin))
		if err != nil {
			log.Warnf("TestMain startup: failed to ddev debug download-images, site %s in dir %s: %v", TestSites[i].Name, TestSites[i].Dir, err)
		}

		for _, volume := range []string{app.Name + "-mariadb"} {
			err = dockerutil.RemoveVolume(volume)
			if err != nil {
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

		if app.SiteStatus() != ddevapp.SiteStopped {
			err = app.Stop(true, false)
			if err != nil {
				log.Fatalf("TestMain shutdown: app.Stop() failed on site %s, err=%v", TestSites[i].Name, err)
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
	defer switchDir()

	runTime := util.TimeTrack(time.Now(), fmt.Sprintf("%s DdevStart", site.Name))

	err := app.Init(site.Dir)
	assert.NoError(err)

	err = app.Start()
	assert.NoError(err)

	t.Cleanup(func() {
		err = app.Stop(true, false)
		assert.NoError(err)
		assert.False(dockerutil.NetworkExists("ddev-" + app.Name + "_default"))
	})

	// Make sure the -built docker image exists before stop
	webBuilt := version.GetWebImage() + "-" + site.Name + "-built"
	dbBuilt := version.GetWebImage() + "-" + site.Name + "-built"
	exists, err := dockerutil.ImageExistsLocally(webBuilt)
	assert.NoError(err)
	assert.True(exists)

	// ensure .ddev/.ddev-docker-compose* exists inside .ddev site folder
	composeFile := fileutil.FileExists(app.DockerComposeYAMLPath())
	assert.True(composeFile)

	for _, containerType := range [3]string{"web", "db", "dba"} {
		containerName, err := constructContainerName(containerType, app)
		assert.NoError(err)
		check, err := testcommon.ContainerCheck(containerName, "running")
		assert.NoError(err)
		assert.True(check, "Container check on %s failed", containerType)
	}

	err = app.Stop(true, false)
	assert.NoError(err)

	// Make sure the -built docker images do not exist after stop with removeData
	for _, imageName := range []string{webBuilt, dbBuilt} {
		exists, err = dockerutil.ImageExistsLocally(imageName)
		assert.NoError(err)
		assert.False(exists, "image %s should not have existed but still exists (while testing %s)", imageName, app.Name)
	}

	runTime()
	switchDir()

	// Start up TestSites[0] again with a post-start hook
	// When run the first time, it should execute the hook, second time it should not
	err = os.Chdir(site.Dir)
	assert.NoError(err)
	err = app.Init(site.Dir)
	app.Hooks = map[string][]ddevapp.YAMLTask{"post-start": {{"exec": "echo hello"}}}

	assert.NoError(err)
	stdoutFunc, err := util.CaptureOutputToFile()
	assert.NoError(err)
	promptOutFunc := util.CaptureUserOut()
	err = app.Start()
	assert.NoError(err)
	t.Cleanup(func() {
		err = app.Stop(true, false)
		assert.NoError(err)
		assert.False(dockerutil.NetworkExists("ddev-" + app.Name + "_default"))
	})
	out := stdoutFunc()
	UOut := promptOutFunc()
	assert.Contains(UOut, "Running task: Exec command 'echo hello' in container/service 'web'")
	assert.Contains(out, "hello\n")

	// try to start a site of same name at different path
	another := site
	tmpDir := testcommon.CreateTmpDir("another")
	copyDir := filepath.Join(tmpDir, "copy")
	err = fileutil.CopyDir(site.Dir, copyDir)
	assert.NoError(err)
	another.Dir = copyDir

	t.Cleanup(func() {
		err = os.RemoveAll(copyDir)
		assert.NoError(err)
	})

	badapp := &ddevapp.DdevApp{}

	err = badapp.Init(copyDir)
	//nolint: errcheck
	defer badapp.Stop(true, false)
	if err == nil {
		logs, logErr := app.CaptureLogs("web", false, "")
		require.Error(t, err, "did not receive err from badapp.Init, logErr=%v, logs:\n======================= logs from app webserver =================\n%s\n============ end logs =========\n", logErr, logs)
	}
	if err != nil {
		assert.Contains(err.Error(), fmt.Sprintf("a project (web container) in running state already exists for %s that was created at %s", TestSites[0].Name, TestSites[0].Dir))
	}

	// Try to start a site of same name at an equivalent but different path. It should work.
	tmpDir, err = testcommon.OsTempDir()
	assert.NoError(err)
	symlink := filepath.Join(tmpDir, fileutil.RandomFilenameBase())
	err = os.Symlink(app.AppRoot, symlink)
	assert.NoError(err)

	t.Cleanup(func() {
		err = os.Remove(symlink)
		assert.NoError(err)
	})

	symlinkApp := &ddevapp.DdevApp{}

	err = symlinkApp.Init(symlink)
	assert.NoError(err)
	//nolint: errcheck
	defer symlinkApp.Stop(true, false)
	// Make sure that GetActiveApp() also fails when trying to start app of duplicate name in current directory.
	switchDir = another.Chdir()
	defer switchDir()

	_, err = ddevapp.GetActiveApp("")
	assert.Error(err)
	if err != nil {
		assert.Contains(err.Error(), fmt.Sprintf("a project (web container) in running state already exists for %s that was created at %s", TestSites[0].Name, TestSites[0].Dir))
	}
	testcommon.CleanupDir(another.Dir)
}

// TestDdevStartMultipleHostnames tests start with multiple hostnames
func TestDdevStartMultipleHostnames(t *testing.T) {
	if nodeps.IsMacM1() {
		t.Skip("Skipping on mac M1 to ignore problems with 'connection reset by peer'")
	}

	assert := asrt.New(t)
	app := &ddevapp.DdevApp{}

	for _, site := range TestSites {
		runTime := util.TimeTrack(time.Now(), fmt.Sprintf("%s DdevStartMultipleHostnames", site.Name))
		testcommon.ClearDockerEnv()

		err := app.Init(site.Dir)
		assert.NoError(err)

		// site.Name is explicitly added because if not removed in GetHostNames() it will cause ddev-router failure
		// "a" is repeated for the same reason; a user error of this type should not cause a failure; GetHostNames()
		// should uniqueify them.
		app.AdditionalHostnames = []string{"sub1." + site.Name, "sub2." + site.Name, "subname.sub3." + site.Name, site.Name, site.Name, site.Name}

		// sub1.<sitename>.ddev.site and sitename.ddev.site are deliberately included to prove they don't
		// cause ddev-router failures"
		// Note that these AdditionalFQDNs require sudo privileges, which the test runners
		// don't typically have.
		app.AdditionalFQDNs = []string{"one.example.com", "two.example.com", "a.one.example.com", site.Name + "." + app.ProjectTLD, "sub1." + site.Name + "." + app.ProjectTLD}

		err = app.WriteConfig()
		assert.NoError(err)

		err = app.StartAndWait(5)
		assert.NoError(err)
		if err != nil && strings.Contains(err.Error(), "db container failed") {
			container, err := app.FindContainerByType("db")
			assert.NoError(err)
			out, err := exec.RunHostCommand("docker", "logs", container.Names[0])
			assert.NoError(err)
			t.Logf("DB Logs after app.Start: \n%s\n== END DB LOGS ==", out)
		}

		// ensure .ddev/docker-compose*.yaml exists inside .ddev site folder
		composeFile := fileutil.FileExists(app.DockerComposeYAMLPath())
		assert.True(composeFile)

		for _, containerType := range [3]string{"web", "db", "dba"} {
			containerName, err := constructContainerName(containerType, app)
			assert.NoError(err)
			check, err := testcommon.ContainerCheck(containerName, "running")
			assert.NoError(err)
			assert.True(check, "Container check on %s failed", containerType)
		}

		httpURLs, _, urls := app.GetAllURLs()
		if globalconfig.GetCAROOT() == "" {
			urls = httpURLs
		}
		t.Logf("Testing these URLs: %v", urls)
		for _, url := range urls {
			_, _ = testcommon.EnsureLocalHTTPContent(t, url+site.Safe200URIWithExpectation.URI, site.Safe200URIWithExpectation.Expect)
		}

		out, err := exec.RunHostCommand(DdevBin, "list")
		assert.NoError(err)
		t.Logf("=========== output of ddev list ==========\n%s\n============", out)

		// Multiple projects can't run at the same time with the fqdns, so we need to clean
		// up these for tests that run later.
		app.AdditionalFQDNs = []string{}
		app.AdditionalHostnames = []string{}
		err = app.WriteConfig()
		assert.NoError(err)

		err = app.Stop(true, false)
		assert.NoError(err)

		runTime()
	}
}

// TestDdevStartUnmanagedSettings start and config with disable_settings_management
func TestDdevStartUnmanagedSettings(t *testing.T) {
	if nodeps.MutagenEnabledDefault || globalconfig.DdevGlobalConfig.MutagenEnabledGlobal || nodeps.NoBindMountsDefault {
		t.Skip("Skipping with mutagen because conflict on settings files")
	}

	assert := asrt.New(t)

	// Make sure this leaves us in the original test directory
	origDir, _ := os.Getwd()

	// Use Drupal9 as it is a good target for composer failures
	site := FullTestSites[8]
	// We will create directory from scratch, as we'll be removing files and changing it.

	app := &ddevapp.DdevApp{Name: site.Name}
	_ = app.Stop(true, false)

	_ = globalconfig.RemoveProjectInfo(site.Name)
	err := site.Prepare()
	require.NoError(t, err)

	err = app.Init(site.Dir)
	assert.NoError(err)

	t.Cleanup(func() {
		err = os.Chdir(origDir)
		assert.NoError(err)
		_ = app.Stop(true, false)
		err = os.RemoveAll(site.Dir)
		assert.NoError(err)
	})

	err = os.Chdir(app.AppRoot)
	assert.NoError(err)

	// Previous tests may have left settings files
	_ = os.Remove(app.SiteSettingsPath)
	_ = os.Remove(app.SiteDdevSettingsFile)

	app.DisableSettingsManagement = true
	err = app.WriteConfig()
	assert.NoError(err)

	// After config, they should still not exist, because we had DisableSettingsManagement
	assert.False(fileutil.FileExists(app.SiteSettingsPath))
	assert.False(fileutil.FileExists(app.SiteDdevSettingsFile))

	err = app.Start()
	assert.NoError(err)

	// After start, they should still not exist, because we had DisableSettingsManagement
	assert.False(fileutil.FileExists(app.SiteSettingsPath))
	assert.False(fileutil.FileExists(app.SiteDdevSettingsFile))

	err = app.Stop(false, false)
	assert.NoError(err)

	app.DisableSettingsManagement = false
	err = app.WriteConfig()
	assert.NoError(err)
	_, err = app.CreateSettingsFile()
	assert.NoError(err)

	err = app.Start()
	assert.NoError(err)

	// Now with DisableSettingsManagement=false, both should exist after config/settings creation
	assert.FileExists(app.SiteSettingsPath)
	assert.FileExists(app.SiteDdevSettingsFile)

	err = os.Remove(filepath.Join(app.SiteSettingsPath))
	assert.NoError(err)
	err = os.Remove(filepath.Join(app.SiteDdevSettingsFile))
	assert.NoError(err)
	// Flush to prevent conflict with mutagen holding file below
	err = app.MutagenSyncFlush()
	assert.NoError(err)

	assert.False(fileutil.FileExists(app.SiteSettingsPath))
	assert.False(fileutil.FileExists(app.SiteDdevSettingsFile))

	err = app.Start()
	assert.NoError(err)

	// Now with DisableSettingsManagement=false, start should have created both
	assert.FileExists(app.SiteSettingsPath)
	assert.FileExists(app.SiteDdevSettingsFile)

}

// TestDdevNoProjectMount tests running without the app file mount.
func TestDdevNoProjectMount(t *testing.T) {
	if nodeps.MutagenEnabledDefault == true || nodeps.NoBindMountsDefault == true {
		t.Skip("Skipping because this doesn't make sense with mutagen or NoBindMounts")
	}
	assert := asrt.New(t)
	app := &ddevapp.DdevApp{}

	// Make sure this leaves us in the original test directory
	testDir, _ := os.Getwd()
	//nolint: errcheck
	defer os.Chdir(testDir)

	site := TestSites[0]
	switchDir := site.Chdir()
	defer switchDir()

	runTime := util.TimeTrack(time.Now(), fmt.Sprintf("%s %s", t.Name(), site.Name))
	defer runTime()

	err := app.Init(site.Dir)
	assert.NoError(err)

	app.NoProjectMount = true
	err = app.WriteConfig()
	assert.NoError(err)

	defer func() {
		err = app.Stop(true, false)
		assert.NoError(err)
		app.NoProjectMount = false
		err = app.WriteConfig()
		assert.NoError(err)
	}()

	err = app.Restart()
	assert.NoError(err)

	stdout, _, err := app.Exec(&ddevapp.ExecOpts{
		Service: "web",
		Dir:     "/var/www/html",
		Cmd:     `findmnt -T /var/www/html | awk '$1 != "TARGET" {printf $1}'`,
	})
	assert.NoError(err)
	assert.NotEqual("/var/www/html", stdout)
}

// TestDdevXdebugEnabled tests running with xdebug_enabled = true, etc.
func TestDdevXdebugEnabled(t *testing.T) {
	// 2021-02: I've been unable to make this test work on WSL2, even though it's easy to demonstrate
	// that it works using PhpStorm, etc. The go listener here doesn't seem to listen on all interfaces.
	// If you get golang listening, then enter the web container and try to connect to the port golang
	// is listening on, it can't connect. However, if you use netcat to listen on the wsl2 side and then
	// connect to it from inside the container, it connects fine.
	if nodeps.IsWSL2() || dockerutil.IsColima() {
		t.Skip("Skipping on WSL2/Colima because this test doesn't work although manual testing works")
	}
	assert := asrt.New(t)

	origDir, _ := os.Getwd()

	app := &ddevapp.DdevApp{}
	testcommon.ClearDockerEnv()

	// On macOS we want to just listen on localhost port, to not trigger
	// firewall block. On other systems, just listen on all interfaces
	listenPort := ":9003"
	if runtime.GOOS == "darwin" {
		listenPort = "127.0.0.1:9003"
	}

	projDir := testcommon.CreateTmpDir(t.Name())
	app, err := ddevapp.NewApp(projDir, false)
	require.NoError(t, err)
	err = app.WriteConfig()
	require.NoError(t, err)

	// Create the simplest possible php file
	err = fileutil.TemplateStringToFile("<?php\necho \"hi there\";\n", nil, filepath.Join(app.AppRoot, "index.php"))
	require.NoError(t, err)

	t.Cleanup(func() {
		err = app.Stop(true, false)
		assert.NoError(err)
		err := os.Chdir(origDir)
		assert.NoError(err)
		err = os.RemoveAll(projDir)
		assert.NoError(err)
	})
	runTime := util.TimeTrack(time.Now(), fmt.Sprintf("%s %s", app.Name, t.Name()))

	_ = os.Chdir(app.AppRoot)

	testcommon.ClearDockerEnv()

	// Most of the time there's no reason to do all versions of PHP
	phpKeys := []string{}
	exclusions := []string{"5.6", "7.0", "7.1", "7.2"}
	for k := range nodeps.ValidPHPVersions {
		if os.Getenv("GOTEST_SHORT") != "" && !nodeps.ArrayContainsString(exclusions, k) {
			phpKeys = append(phpKeys, k)
		}
	}
	sort.Strings(phpKeys)

	for _, v := range phpKeys {
		app.PHPVersion = v
		t.Logf("Beginning XDebug checks with XDebug php%s\n", v)

		err = app.Restart()
		require.NoError(t, err)

		opts := &ddevapp.ExecOpts{
			Service: "web",
			Cmd:     "php --ri xdebug",
		}
		stdout, _, err := app.Exec(opts)
		assert.Error(err)
		assert.Contains(stdout, "Extension 'xdebug' not present")

		// Run with xdebug enabled
		_, _, err = app.Exec(&ddevapp.ExecOpts{
			Cmd: "enable_xdebug",
		})
		assert.NoError(err)

		stdout, _, err = app.Exec(opts)
		assert.NoError(err)

		if err != nil {
			t.Errorf("Aborting xdebug check for php%s: %v", v, err)
			continue
		}
		// PHP 7.2 through 8.1 gets xdebug 3.0+
		if nodeps.ArrayContainsString([]string{nodeps.PHP72, nodeps.PHP73, nodeps.PHP74, nodeps.PHP80, nodeps.PHP81}, app.PHPVersion) {
			assert.Contains(stdout, "xdebug.mode => debug,develop => debug,develop", "xdebug is not enabled for %s", v)
			assert.Contains(stdout, "xdebug.client_host => host.docker.internal => host.docker.internal")
		} else {
			assert.Contains(stdout, "xdebug support => enabled", "xdebug is not enabled for %s", v)
			assert.Contains(stdout, "xdebug.remote_host => host.docker.internal => host.docker.internal")
		}

		// Start a listener on port 9003 of localhost (where PHPStorm or whatever would listen)
		listener, err := net.Listen("tcp", listenPort)
		require.NoError(t, err)
		time.Sleep(time.Second * 1)

		acceptListenDone := make(chan bool, 1)
		defer close(acceptListenDone)

		go func() {
			time.Sleep(time.Second)
			t.Logf("Curling to port 9003 with xdebug enabled, PHP version=%s time=%v", v, time.Now())
			// Curl to the project's index.php or anything else
			out, resp, err := testcommon.GetLocalHTTPResponse(t, app.GetWebContainerDirectHTTPURL(), 12)
			if err != nil {
				t.Logf("time=%v got resp %v output %s: %v", time.Now(), resp, out, err)
				if resp != nil {
					t.Logf("resp code=%v", resp.StatusCode)
				}
			}
		}()

		// Accept is blocking, no way to timeout, so use
		// goroutine instead.

		go func() {
			t.Logf("Attempting accept of port 9003 with xdebug enabled, PHP version=%s time=%v", v, time.Now())

			// Accept the listen on 9003 coming in from in-container php-xdebug
			conn, err := listener.Accept()
			assert.NoError(err)
			if err == nil {
				t.Logf("Completed accept of port 9003 with xdebug enabled, PHP version=%s, time=%v\n", v, time.Now())
			} else {
				t.Logf("Failed accept on port 9003, err=%v", err)
				acceptListenDone <- true
				return
			}
			// Grab the Xdebug connection start and look in it for "Xdebug"
			b := make([]byte, 650)
			_, err = bufio.NewReader(conn).Read(b)
			assert.NoError(err)
			lineString := string(b)
			assert.Contains(lineString, "Xdebug")
			assert.Contains(lineString, `xdebug:language_version="`+v)
			acceptListenDone <- true
		}()

		select {
		case <-acceptListenDone:
			fmt.Printf("Read from acceptListenDone at %v\n", time.Now())
		case <-time.After(time.Second * 11):
			t.Fatalf("Timed out waiting for accept/listen at %v, PHP version %v\n", time.Now(), v)
		}
	}
	runTime()
}

// TestDdevXhprofEnabled tests running with xhprof_enabled = true, etc.
func TestDdevXhprofEnabled(t *testing.T) {
	if nodeps.IsMacM1() || dockerutil.IsColima() {
		t.Skip("Skipping on mac M1 to ignore problems with 'connection reset by peer'")
	}

	assert := asrt.New(t)
	origDir, _ := os.Getwd()

	testcommon.ClearDockerEnv()

	projDir := testcommon.CreateTmpDir(t.Name())
	app, err := ddevapp.NewApp(projDir, false)
	require.NoError(t, err)
	err = app.WriteConfig()
	require.NoError(t, err)

	// Create the simplest possible php file
	err = fileutil.TemplateStringToFile("<?php\nphpinfo();\n", nil, filepath.Join(app.AppRoot, "index.php"))
	require.NoError(t, err)

	runTime := util.TimeTrack(time.Now(), fmt.Sprintf("%s %s", app.Name, t.Name()))

	// Does not work with php5.6 anyway (SEGV), for resource conservation
	// skip older unsupported versions
	phpKeys := []string{}
	exclusions := []string{"5.6", "7.0", "7.1"}
	for k := range nodeps.ValidPHPVersions {
		if !nodeps.ArrayContainsString(exclusions, k) {
			phpKeys = append(phpKeys, k)
		}
	}
	sort.Strings(phpKeys)

	err = app.Init(app.AppRoot)
	require.NoError(t, err)

	t.Cleanup(func() {
		err = app.Stop(true, false)
		assert.NoError(err)
		err = os.Chdir(origDir)
		assert.NoError(err)
		err = os.RemoveAll(projDir)
		assert.NoError(err)
	})

	webserverKeys := make([]string, 0, len(nodeps.ValidWebserverTypes))
	for k := range nodeps.ValidWebserverTypes {
		webserverKeys = append(webserverKeys, k)
	}

	for _, webserverKey := range webserverKeys {
		app.WebserverType = webserverKey

		for _, v := range phpKeys {
			t.Logf("Beginning XHProf checks with XHProf webserver_type=%s php%s\n", webserverKey, v)
			fmt.Printf("Attempting XHProf checks with XHProf PHP%s\n", v)
			app.PHPVersion = v

			err = app.Restart()
			require.NoError(t, err)

			stdout, _, err := app.Exec(&ddevapp.ExecOpts{
				Service: "web",
				Cmd:     fmt.Sprintf("php --ri xhprof"),
			})
			assert.Error(err)
			assert.Contains(stdout, "Extension 'xhprof' not present")

			// Run with xhprof enabled
			_, _, err = app.Exec(&ddevapp.ExecOpts{
				Cmd: fmt.Sprintf("enable_xhprof"),
			})
			assert.NoError(err)

			stdout, _, err = app.Exec(&ddevapp.ExecOpts{
				Service: "web",
				Cmd:     fmt.Sprintf("php --ri xhprof"),
			})
			assert.NoError(err)
			if err != nil {
				t.Errorf("Aborting xhprof check for php%s: %v", v, err)
				continue
			}
			assert.Contains(stdout, "xhprof.output_dir", "xhprof is not enabled for %s", v)

			// Dummy hit to avoid M1 "connection reset by peer"
			_, _, _ = testcommon.GetLocalHTTPResponse(t, app.GetPrimaryURL(), 1)
			out, _, err := testcommon.GetLocalHTTPResponse(t, app.GetPrimaryURL(), 1)
			assert.NoError(err, "Failed to get base URL webserver_type=%s, php_version=%s", webserverKey, v)
			assert.Contains(out, "module_xhprof")

			out, _, err = testcommon.GetLocalHTTPResponse(t, app.GetPrimaryURL()+"/xhprof/", 1)
			assert.NoError(err)
			// Output should contain at least one run
			assert.Contains(out, ".ddev.xhprof</a><small>")

			// Disable all to avoid confusion
			_, _, err = app.Exec(&ddevapp.ExecOpts{
				Cmd: fmt.Sprintf("disable_xhprof && rm -rf /tmp/xhprof"),
			})
			assert.NoError(err)
		}
	}
	runTime()
}

// TestDdevMysqlWorks tests that mysql client can be run in both containers.
func TestDdevMysqlWorks(t *testing.T) {
	assert := asrt.New(t)
	app := &ddevapp.DdevApp{}

	site := TestSites[0]
	runTime := util.TimeTrack(time.Now(), fmt.Sprintf("%s DdevMysqlWorks", site.Name))

	err := app.Init(site.Dir)
	assert.NoError(err)

	testcommon.ClearDockerEnv()
	err = app.StartAndWait(0)
	//nolint: errcheck
	defer app.Stop(true, false)
	require.NoError(t, err)

	// Test that mysql + .my.cnf works on web container
	_, _, err = app.Exec(&ddevapp.ExecOpts{
		Service: "web",
		Cmd:     "mysql -e 'SELECT USER();' | grep 'db@'",
	})
	assert.NoError(err)
	_, _, err = app.Exec(&ddevapp.ExecOpts{
		Service: "web",
		Cmd:     "mysql -e 'SELECT DATABASE();' | grep 'db'",
	})
	assert.NoError(err)

	// Test that mysql + .my.cnf works on db container
	_, _, err = app.Exec(&ddevapp.ExecOpts{
		Service: "db",
		Cmd:     "mysql -e 'SELECT USER();' | grep 'root@localhost'",
	})
	assert.NoError(err)
	_, _, err = app.Exec(&ddevapp.ExecOpts{
		Service: "db",
		Cmd:     "mysql -e 'SELECT DATABASE();' | grep 'db'",
	})
	assert.NoError(err)

	err = app.Stop(true, false)
	assert.NoError(err)

	runTime()

}

// TestStartWithoutDdev makes sure we don't have a regression where lack of .ddev
// causes a panic.
func TestStartWithoutDdevConfig(t *testing.T) {
	// Set up tests and give ourselves a working directory.
	assert := asrt.New(t)
	testDir := testcommon.CreateTmpDir(t.Name())

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
		assert.Contains(err.Error(), "could not find a project")
	}
}

// TestGetApps tests the GetActiveProjects function to ensure it accurately returns a list of running applications.
func TestGetApps(t *testing.T) {
	assert := asrt.New(t)

	// Start the apps.
	for _, site := range TestSites {
		testcommon.ClearDockerEnv()
		app := &ddevapp.DdevApp{}

		err := app.Init(site.Dir)
		if err != nil {
			assert.NoError(err, "unable to app.Init %s: %v", app.Name, err)
			continue
		}

		err = app.Start()
		if err != nil {
			assert.NoError(err, "unable to app.start %s: %v", app.Name, err)
			continue
		}
	}

	apps := ddevapp.GetActiveProjects()

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
		if err != nil {
			assert.NoError(err, "unable to app.Init %s: %v", app.Name, err)
			continue
		}

		err = app.Stop(true, false)
		if err != nil {
			assert.NoError(err, "unable to app.Stop %s: %v", app.Name, err)
			continue
		}
	}
}

// TestDdevImportDB tests the functionality that is called when "ddev import-db" is executed
func TestDdevImportDB(t *testing.T) {
	assert := asrt.New(t)
	app := &ddevapp.DdevApp{}
	origDir, _ := os.Getwd()

	site := TestSites[0]

	runTime := util.TimeTrack(time.Now(), fmt.Sprintf("%s %s", site.Name, t.Name()))

	testcommon.ClearDockerEnv()
	err := app.Init(site.Dir)
	assert.NoError(err)

	t.Cleanup(func() {
		runTime()
		err = app.Stop(true, false)
		assert.NoError(err)
		app.Hooks = nil
		app.Database.Type = nodeps.MariaDB
		app.Database.Version = nodeps.MariaDBDefaultVersion
		err = app.WriteConfig()
		assert.NoError(err)
		_ = os.Remove(filepath.Join(app.AppRoot, "hello-pre-import-db-"+app.Name))
		_ = os.Remove(filepath.Join(app.AppRoot, "hello-post-import-db-"+app.Name))

	})

	c := make(map[string]string)

	for _, dbType := range []string{nodeps.MariaDB, nodeps.Postgres} {
		err = app.Stop(true, false)
		require.NoError(t, err)

		app.Database = ddevapp.DatabaseDesc{
			Type:    dbType,
			Version: nodeps.MariaDBDefaultVersion,
		}
		if dbType == nodeps.Postgres {
			app.Database.Version = nodeps.PostgresDefaultVersion
		}
		err = app.WriteConfig()
		assert.NoError(err)

		err = app.Start()
		require.NoError(t, err)

		c[nodeps.MariaDB] = "mysql -N -e 'DROP DATABASE IF EXISTS test;'"
		c[nodeps.Postgres] = fmt.Sprintf(`echo "SELECT 'DROP DATABASE test' WHERE EXISTS (SELECT FROM pg_database WHERE datname = 'test')\gexec" | psql -v ON_ERROR_STOP=1 -d postgres`)
		out, stderr, err := app.Exec(&ddevapp.ExecOpts{
			Service: "db",
			Cmd:     c[dbType],
		})
		assert.NoError(err, "out=%s, stderr=%s", out, stderr)

		app.Hooks = map[string][]ddevapp.YAMLTask{"post-import-db": {{"exec-host": "touch hello-post-import-db-" + app.Name}}, "pre-import-db": {{"exec-host": "touch hello-pre-import-db-" + app.Name}}}

		// Test simple db loads.
		for _, file := range []string{"users.sql", "users.mysql", "users.sql.gz", "users.sql.bz2", "users.sql.xz", "users.mysql.gz", "users.mysql.bz2", "users.mysql.xz", "users.sql.tar", "users.mysql.tar", "users.sql.tar.gz", "users.sql.tar.bz2", "users.sql.tar.xz", "users.mysql.tar.gz", "users.mysql.tar.xz", "users.sql.tgz", "users.mysql.tgz", "users.sql.zip", "users.mysql.zip", "users_with_USE_statement.sql"} {
			p := filepath.Join(origDir, "testdata", t.Name(), dbType, file)
			if !fileutil.FileExists(p) {
				t.Logf("skipping %s because it does not exist", p)
				continue
			}
			err = app.ImportDB(p, "", false, false, "db")
			assert.NoError(err, "Failed to app.ImportDB path: %s err: %v", p, err)
			if err != nil {
				continue
			}

			c[nodeps.MariaDB] = `set -eu -o pipefail; mysql -N -e 'SHOW TABLES;' | cat`
			c[nodeps.Postgres] = `set -eu -o pipefail; psql -t -v ON_ERROR_STOP=1 db -c '\dt' |awk -F' *\| *' '{ if (NF>2) print $2 }' `
			// There should be exactly the one "users" table for each of these files
			out, stderr, err := app.Exec(&ddevapp.ExecOpts{
				Service: "db",
				Cmd:     c[dbType],
			})
			assert.NoError(err)
			assert.Equal("users\n", out, "Failed to find users table for file %s, stdout='%s', stderr='%s'", file, out, stderr)

			c[nodeps.MariaDB] = `set -eu -o pipefail; mysql -N -e 'SHOW DATABASES;' | egrep -v "^(information_schema|performance_schema|mysql)$"`
			c[nodeps.Postgres] = `set -eu -o pipefail; psql -t -c "SELECT datname FROM pg_database;" | egrep -v "template?|postgres"`

			// Verify that no extra database was created
			out, stderr, err = app.Exec(&ddevapp.ExecOpts{
				Service: "db",
				Cmd:     c[dbType],
			})
			assert.NoError(err)
			out = strings.Trim(out, " \n")
			assert.Equal("db", out, "found extra database, file=%s dbType=%s dbVersion=%s out=%s, stderr=%s", file, dbType, app.Database.Version, out, stderr)
		}

		// Test that a settings file has correct hash_salt format
		switch app.Type {
		case nodeps.AppTypeDrupal7:
			drupalHashSalt, err := fileutil.FgrepStringInFile(app.SiteDdevSettingsFile, "$drupal_hash_salt")
			assert.NoError(err)
			assert.True(drupalHashSalt)
		case nodeps.AppTypeDrupal9:
			settingsHashSalt, err := fileutil.FgrepStringInFile(app.SiteDdevSettingsFile, "settings['hash_salt']")
			assert.NoError(err)
			assert.True(settingsHashSalt)
		case nodeps.AppTypeWordPress:
			hasAuthSalt, err := fileutil.FgrepStringInFile(app.SiteSettingsPath, "SECURE_AUTH_SALT")
			assert.NoError(err)
			assert.True(hasAuthSalt)
		}
		err = app.Stop(true, false)
		require.NoError(t, err)
	}

	for _, dbType := range []string{nodeps.MariaDB, nodeps.Postgres} {
		err = app.Stop(true, false)
		app.Database = ddevapp.DatabaseDesc{Type: dbType, Version: nodeps.MariaDBDefaultVersion}
		if dbType == nodeps.Postgres {
			app.Database.Version = nodeps.PostgresDefaultVersion
		}
		err = app.WriteConfig()
		require.NoError(t, err)
		err = app.Start()
		require.NoError(t, err)

		for _, db := range []string{"db", "extradb"} {
			// Import from stdin, make sure that works
			inputFile := filepath.Join(origDir, "testdata", t.Name(), dbType, "stdintable.sql")
			f, err := os.Open(inputFile)
			require.NoError(t, err)
			// nolint: errcheck
			defer f.Close()
			savedStdin := os.Stdin
			os.Stdin = f
			err = app.ImportDB("", "", false, false, db)
			os.Stdin = savedStdin
			assert.NoError(err)

			c[nodeps.MariaDB] = fmt.Sprintf(`mysql -N %s -e "SHOW DATABASES LIKE '%s'; SELECT COUNT(*) from stdintable"`, db, db)
			c[nodeps.Postgres] = fmt.Sprintf(`psql -t -d %s -c "SELECT datname FROM pg_database WHERE datname='%s' ;" && psql -t -d %s -c "SELECT COUNT(*) from stdintable"`, db, db, db)

			out, stderr, err := app.Exec(&ddevapp.ExecOpts{
				Service: "db",
				Cmd:     c[dbType],
			})
			assert.NoError(err, "db=%s dbType=%s stdout=%s, stderr=%s", db, dbType, out, stderr)
			out = strings.ReplaceAll(out, "\n", "")
			out = strings.ReplaceAll(out, " ", "")
			out = strings.Trim(out, " \n")
			assert.Equal(fmt.Sprintf("%s2", db), out)

			// Import 2-user users.sql into users table
			path := filepath.Join(origDir, "testdata", t.Name(), dbType, "users.sql")
			err = app.ImportDB(path, "", false, false, db)
			assert.NoError(err)
			c[nodeps.MariaDB] = fmt.Sprintf(`echo "SELECT * FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_SCHEMA = '%s';" | mysql -N %s`, db, db)
			c[nodeps.Postgres] = fmt.Sprintf(`bash -c "echo '\dt' | psql -t -d %s | awk 'NF > 1'"`, db)
			out, stderr, err = app.Exec(&ddevapp.ExecOpts{
				Service: "db",
				Cmd:     c[dbType],
			})
			assert.NoError(err, "db=%s, dbType=%s, out=%s, stderr=%s", db, dbType, out, stderr)
			out = strings.Trim(out, "\n")
			lines := strings.Split(out, "\n")
			assert.Len(lines, 1)

			// Import 1-user sql (users_just_one table) and make sure only one table is left there
			path = filepath.Join(origDir, "testdata", t.Name(), dbType, "oneuser.sql")
			err = app.ImportDB(path, "", false, false, db)
			assert.NoError(err)
			out, _, err = app.Exec(&ddevapp.ExecOpts{
				Service: "db",
				Cmd:     c[dbType],
			})
			assert.NoError(err)
			out = strings.Trim(out, "\n")
			lines = strings.Split(out, "\n")
			assert.Len(lines, 1)
			assert.Contains(out, "users_just_one")

			// This import-on-top-of-existing-file can't work on postgres
			if dbType != nodeps.Postgres {
				// Import 2-user users.sql again, but with nodrop=true
				// We should end up with 2 tables now
				path = filepath.Join(origDir, "testdata", t.Name(), dbType, "users.sql")
				err = app.ImportDB(path, "", false, true, db)
				assert.NoError(err)
				out, _, err = app.Exec(&ddevapp.ExecOpts{
					Service: "db",
					Cmd:     c[dbType],
				})
				assert.NoError(err)
				out = strings.Trim(out, "\n")
				lines = strings.Split(out, "\n")
				assert.Len(lines, 2)
			}
		}
	}
	app.Database = ddevapp.DatabaseDesc{Type: nodeps.MariaDB, Version: nodeps.MariaDBDefaultVersion}
	err = app.WriteConfig()
	require.NoError(t, err)
	err = app.Start()
	require.NoError(t, err)

	// Test database that has SQL DDL in the content to make sure nothing gets corrupted.
	// Make sure database "test" does not exist initially
	dbType := nodeps.MariaDB
	c[nodeps.MariaDB] = "mysql -N -e 'DROP DATABASE IF EXISTS test;'"
	c[nodeps.Postgres] = fmt.Sprintf(`echo "SELECT 'DROP DATABASE test' WHERE EXISTS (SELECT FROM pg_database WHERE datname = 'test')\gexec" | psql -v ON_ERROR_STOP=1 -d postgres`)
	out, stderr, err := app.Exec(&ddevapp.ExecOpts{
		Service: "db",
		Cmd:     c[dbType],
	})
	assert.NoError(err, "out=%s, stderr=%s", out, stderr)

	_, _, err = app.Exec(&ddevapp.ExecOpts{Service: "db", Cmd: "mysql -N -e 'DROP TABLE IF EXISTS wp_posts;'"})
	require.NoError(t, err)
	file := "posts_with_ddl_content.sql"
	path := filepath.Join(origDir, "testdata", t.Name(), "mariadb", file)
	err = app.ImportDB(path, "", false, false, "db")
	assert.NoError(err, "Failed to app.ImportDB path: %s err: %v", path, err)
	checkImportDbImports(t, app)

	// Now test the same when importing from stdin, same file
	_, _, err = app.Exec(&ddevapp.ExecOpts{Service: "db", Cmd: "mysql -N -e 'DROP TABLE wp_posts;'"})
	require.NoError(t, err)
	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()
	oldStdin := os.Stdin
	t.Cleanup(func() {
		os.Stdin = oldStdin
	})
	os.Stdin = f
	err = app.ImportDB("", "", false, false, "db")
	assert.NoError(err, "Failed to app.ImportDB path: %s err: %v", path, err)
	os.Stdin = oldStdin
	checkImportDbImports(t, app)

	// Verify that the count of tables is exactly what it should be, that nothing was lost in the
	// import due to embedded DDL statements.
	out, stderr, err = app.Exec(&ddevapp.ExecOpts{
		Service: "db",
		Cmd:     `mysql -N -e 'SELECT COUNT(*) FROM wp_posts;'`,
	})
	assert.NoError(err, "out=%s, stderr=%s", out, stderr)
	assert.Equal("180\n", out)

	// Now check standard archive imports
	if site.DBTarURL != "" {
		_, cachedArchive, err := testcommon.GetCachedArchive(site.Name, site.Name+"_siteTarArchive", "", site.DBTarURL)
		require.NoError(t, err)
		_ = os.Remove(filepath.Join(app.AppRoot, "hello-pre-import-db-"+app.Name))
		_ = os.Remove(filepath.Join(app.AppRoot, "hello-post-import-db-"+app.Name))
		assert.NoFileExists(filepath.Join(app.AppRoot, "hello-pre-import-db-"+app.Name))
		assert.NoFileExists(filepath.Join(app.AppRoot, "hello-post-import-db-"+app.Name))

		err = app.ImportDB(cachedArchive, "", false, false, "db")
		assert.NoError(err)
		assert.FileExists(filepath.Join(app.AppRoot, "hello-pre-import-db-"+app.Name))
		assert.FileExists(filepath.Join(app.AppRoot, "hello-post-import-db-"+app.Name))
		_ = os.Remove(filepath.Join(app.AppRoot, "hello-pre-import-db-"+app.Name))
		_ = os.Remove(filepath.Join(app.AppRoot, "hello-post-import-db-"+app.Name))
	}

	if site.DBZipURL != "" {
		_, cachedArchive, err := testcommon.GetCachedArchive(site.Name, site.Name+"_siteZipArchive", "", site.DBZipURL)

		require.NoError(t, err)
		err = app.ImportDB(cachedArchive, "", false, false, "db")
		assert.NoError(err)
	}

	if site.FullSiteTarballURL != "" {
		_, cachedArchive, err := testcommon.GetCachedArchive(site.Name, site.Name+"_FullSiteTarballURL", "", site.FullSiteTarballURL)
		require.NoError(t, err)

		err = app.ImportDB(cachedArchive, "data.sql", false, false, "db")
		assert.NoError(err, "Failed to find data.sql at root of tarball %s", cachedArchive)
	}

}

func checkImportDbImports(t *testing.T, app *ddevapp.DdevApp) {
	assert := asrt.New(t)

	// There should be exactly the one wp_posts table for this file
	out, _, err := app.Exec(&ddevapp.ExecOpts{
		Service: "db",
		Cmd:     "mysql -N -e 'SHOW TABLES;' | cat",
	})
	assert.NoError(err)
	assert.Equal("wp_posts\n", out)

	// Verify that no additional database was created (this one has a CREATE DATABASE statement)
	out, _, err = app.Exec(&ddevapp.ExecOpts{
		Service: "db",
		Cmd:     `mysql -N -e 'SHOW DATABASES;' | egrep -v "^(information_schema|performance_schema|mysql)$"`,
	})
	assert.NoError(err)
	assert.Equal("db\n", out)
}

// TestDdevAllDatabases tests db import/export/snapshot/restore/start with supported database versions
func TestDdevAllDatabases(t *testing.T) {
	assert := asrt.New(t)

	dbVersions := nodeps.GetValidDatabaseVersions()
	// Bug: postgres 9 doesn't work with snapshot restore, see https://github.com/drud/ddev/issues/3583
	dbVersions = nodeps.RemoveItemFromSlice(dbVersions, "postgres:9")
	//Use a smaller list if GOTEST_SHORT
	if os.Getenv("GOTEST_SHORT") != "" {
		dbVersions = []string{"postgres:14", "mariadb:10.2", "mariadb:10.3", "mysql:8.0", "mysql:5.7"}
		t.Logf("Using limited set of database servers because GOTEST_SHORT is set (%v)", dbVersions)
	}

	app := &ddevapp.DdevApp{}
	origDir, _ := os.Getwd()

	site := TestSites[0]
	runTime := util.TimeTrack(time.Now(), fmt.Sprintf("%s %s", site.Name, t.Name()))

	testcommon.ClearDockerEnv()
	err := app.Init(site.Dir)
	assert.NoError(err)

	err = app.Stop(true, false)
	require.NoError(t, err)

	// Make sure there isn't an old db laying around
	_ = dockerutil.RemoveVolume(app.Name + "-mariadb")
	t.Cleanup(func() {
		err = app.Stop(true, false)
		assert.NoError(err)
		err = os.RemoveAll(app.GetConfigPath("db_snapshots"))
		assert.NoError(err)

		// Make sure we leave the config.yaml in expected state
		app.Database.Type = nodeps.MariaDB
		app.Database.Version = nodeps.MariaDBDefaultVersion
		err = app.WriteConfig()
		assert.NoError(err)
		err = os.Chdir(origDir)
		assert.NoError(err)
	})

	for _, dbTypeVersion := range dbVersions {
		t.Logf("testing db server functionality of %s", dbTypeVersion)
		parts := strings.Split(dbTypeVersion, ":")
		dbType := parts[0]
		dbVersion := parts[1]
		require.Len(t, parts, 2)

		err = app.Stop(true, false)
		require.NoError(t, err)
		app.Database.Type = dbType
		app.Database.Version = dbVersion
		err = app.WriteConfig()
		require.NoError(t, err)

		startErr := app.Start()
		if startErr != nil {
			assert.NoError(startErr, "failed to start %s:%s", dbType, dbVersion)
			err = app.Stop(true, false)
			assert.NoError(err)
			t.Errorf("Continuing/skippping %s due to app.Start() failure %v", dbVersion, startErr)
			continue
		}

		// The db_mariadb_version.txt file does not exist on postgres
		if dbType != nodeps.Postgres {
			// Make sure the version of db running matches expected
			containerDBVersion, _, _ := app.Exec(&ddevapp.ExecOpts{
				Service: "db",
				Cmd:     "cat /var/lib/mysql/db_mariadb_version.txt",
			})
			assert.Equal(dbType+"_"+dbVersion, strings.Trim(containerDBVersion, "\n\r "))
		}

		importPath := filepath.Join(origDir, "testdata", t.Name(), dbType, "users.sql")
		err = app.ImportDB(importPath, "", false, false, "db")
		if err != nil {
			assert.NoError(err, "failed to import %s on %s:%s", importPath, dbType, dbVersion)
			continue
		}

		_ = os.Mkdir("tmp", 0777)
		err = fileutil.PurgeDirectory("tmp")
		assert.NoError(err)

		// Test that we can export-db to a gzipped file
		err = app.ExportDB("tmp/users1.sql.gz", "gzip", "db")
		assert.NoError(err)

		// Validate contents
		err = archive.Ungzip("tmp/users1.sql.gz", "tmp")
		assert.NoError(err)
		stringFound, err := fileutil.GrepStringInFile("tmp/users1.sql", "CREATE TABLE.*users")
		assert.NoError(err)
		assert.True(stringFound)

		err = app.MutagenSyncFlush()
		assert.NoError(err)
		err = fileutil.PurgeDirectory("tmp")
		assert.NoError(err)

		// Export to an ungzipped file and validate
		err = app.ExportDB("tmp/users2.sql", "", "db")
		assert.NoError(err)

		err = app.MutagenSyncFlush()
		assert.NoError(err)

		// Validate contents
		stringFound, err = fileutil.GrepStringInFile("tmp/users2.sql", "CREATE TABLE.*users")
		assert.NoError(err)
		assert.True(stringFound)

		err = fileutil.PurgeDirectory("tmp")
		assert.NoError(err)

		// Capture to stdout without gzip compression
		// CaptureStdOut() doesn't work on Windows.
		if runtime.GOOS != "windows" {
			stdout := util.CaptureStdOut()
			err = app.ExportDB("", "", "db")
			assert.NoError(err)
			out := stdout()
			assert.Regexp(regexp.MustCompilePOSIX("CREATE TABLE.*users"), out)
		}

		snapshotName := dbType + "_" + dbVersion + "_" + fileutil.RandomFilenameBase()
		fullSnapshotName, err := app.Snapshot(snapshotName)
		if err != nil {
			assert.NoError(err, "could not create snapshot %s for %s: %v output=%v", snapshotName, dbTypeVersion, err, fullSnapshotName)
			continue
		}

		snapshotPath, err := ddevapp.GetSnapshotFileFromName(fullSnapshotName, app)
		assert.NoError(err)
		fi, err := os.Stat(app.GetConfigPath(filepath.Join("db_snapshots", snapshotPath)))
		require.NoError(t, err)
		// make sure there's something in the snapshot
		assert.Greater(fi.Size(), int64(1000), "snapshot %s for %s may be empty: %v", fi.Name(), dbTypeVersion, fi)

		// Delete the user in the database so we can later verify snapshot restore
		c := map[string]string{
			nodeps.MySQL:    fmt.Sprintf(`echo "DELETE FROM users;" | mysql`),
			nodeps.MariaDB:  fmt.Sprintf(`echo "DELETE FROM users;" | mysql`),
			nodeps.Postgres: fmt.Sprintf(`echo "DELETE FROM users;" | psql`),
		}
		_, _, err = app.Exec(&ddevapp.ExecOpts{
			Service: "db",
			Cmd:     c[dbType],
		})
		assert.NoError(err)

		err = app.RestoreSnapshot(fullSnapshotName)
		assert.NoError(err, "could not restore snapshot %s for %s: %v", fullSnapshotName, dbTypeVersion, err)
		if err != nil {
			_ = app.Stop(true, false)
			continue
		}

		if dbType != nodeps.Postgres {
			// Make sure the version of db running matches expected
			containerDBVersion, _, err := app.Exec(&ddevapp.ExecOpts{
				Service: "db",
				Cmd:     "cat /var/lib/mysql/db_mariadb_version.txt",
			})
			assert.NoError(err)
			assert.Equal(dbType+"_"+dbVersion, strings.Trim(containerDBVersion, "\n\r "))
		}

		c = map[string]string{
			nodeps.MySQL:    `echo "SELECT COUNT(*) FROM users;" | mysql -N`,
			nodeps.MariaDB:  `echo "SELECT COUNT(*) FROM users;" | mysql -N`,
			nodeps.Postgres: `echo "SELECT COUNT(*) FROM users;" | psql -t`,
		}
		out, _, err := app.Exec(&ddevapp.ExecOpts{
			Service: "db",
			Cmd:     c[dbType],
		})
		assert.NoError(err)
		out = strings.Trim(out, "\n\r ")
		assert.Equal("2", out)

		err = app.Stop(true, false)
		assert.NoError(err)
	}
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

	runTime := util.TimeTrack(time.Now(), fmt.Sprintf("%s DdevExportDB", site.Name))

	testcommon.ClearDockerEnv()
	err := app.Init(site.Dir)
	assert.NoError(err)

	t.Cleanup(func() {
		err = app.Stop(true, false)
		assert.NoError(err)
		app.Database = ddevapp.DatabaseDesc{Type: nodeps.MariaDB, Version: nodeps.MariaDBDefaultVersion}
		err = app.WriteConfig()
		assert.NoError(err)
	})

	for _, dbType := range []string{nodeps.MariaDB, nodeps.Postgres} {
		err = app.Stop(true, false)
		require.NoError(t, err)
		app.Database = ddevapp.DatabaseDesc{
			Type:    dbType,
			Version: nodeps.MariaDBDefaultVersion,
		}
		if dbType == nodeps.Postgres {
			app.Database.Version = nodeps.PostgresDefaultVersion
		}
		err = app.WriteConfig()
		require.NoError(t, err)
		err = app.Start()
		require.NoError(t, err)

		importPath := filepath.Join(testDir, "testdata", t.Name(), dbType, "users.sql")
		err = app.ImportDB(importPath, "", false, false, "db")
		require.NoError(t, err)

		_ = os.Mkdir("tmp", 0777)
		// Most likely reason for failure is it exists, so let that go
		err = fileutil.PurgeDirectory("tmp")
		assert.NoError(err)

		// Test that we can export-db to a plain file. This should be larger than
		// the gzipped file we'll do next.
		err = app.ExportDB("tmp/users1.sql", "", "db")
		assert.NoError(err)

		// The users1.sql should be something over 2K
		// and should finish with "Dump completed on"
		f, err := os.Stat("tmp/users1.sql")
		assert.NoError(err)
		assert.Greater(f.Size(), int64(2000))
		l, err := readLastLine("tmp/users1.sql")
		assert.NoError(err)
		assert.Contains(l, "ump complete")

		// Now copy our (larger) users1.sql to users1.sql.gz
		// so we can overwrite it and come out with a totally valid new file.
		// Copy is used here instead of mv/rename because of Windows issues.
		err = fileutil.CopyFile("tmp/users1.sql", "tmp/users1.sql.gz")
		assert.NoError(err)

		// Test that we can export-db to an existing gzipped file
		err = app.ExportDB("tmp/users1.sql.gz", "gzip", "db")
		assert.NoError(err)

		// The new gzipped file should be less than 1K
		f, err = os.Stat("tmp/users1.sql.gz")
		assert.NoError(err)
		assert.Less(f.Size(), int64(1500))

		// Validate contents
		err = archive.Ungzip("tmp/users1.sql.gz", "tmp")
		assert.NoError(err)
		stringFound, err := fileutil.FgrepStringInFile("tmp/users1.sql", "Table structure for table `users`")
		assert.NoError(err)
		if !stringFound {
			stringFound, err = fileutil.FgrepStringInFile("tmp/users1.sql", "Name: users; Type: TABLE")
			assert.NoError(err)
		}
		assert.True(stringFound)

		// Simple export-and-validate to various types of compression
		cTypes := map[string]string{
			"gzip":  "gz",
			"bzip2": "bz2",
			"xz":    "xz",
		}
		for cType, ext := range cTypes {
			err = app.ExportDB("tmp/users1.sql."+ext, cType, "db")
			assert.NoError(err)
			switch cType {
			case "gzip":
				err = archive.Ungzip("tmp/users1.sql."+ext, "tmp")
				assert.NoError(err)
			case "bzip2":
				err = archive.UnBzip2("tmp/users1.sql."+ext, "tmp")
				assert.NoError(err)
			case "xz":
				err = archive.UnXz("tmp/users1.sql."+ext, "tmp")
				assert.NoError(err)
			}

			stringFound, err = fileutil.FgrepStringInFile("tmp/users1.sql", "Table structure for table `users`")
			assert.NoError(err)
			if !stringFound {
				stringFound, err = fileutil.FgrepStringInFile("tmp/users1.sql", "Name: users; Type: TABLE")
				assert.NoError(err)
			}
			assert.True(stringFound, "expected info not found %s (%s)", cType, "tmp/users1.sql")
		}

		// Flush needs to be complete before purge or may conflict with mutagen on windows
		err = app.MutagenSyncFlush()
		assert.NoError(err)
		err = fileutil.PurgeDirectory("tmp")
		assert.NoError(err)

		// Export to an ungzipped file and validate
		err = app.ExportDB("tmp/users2.sql", "", "db")
		assert.NoError(err)

		// Validate contents
		stringFound, err = fileutil.FgrepStringInFile("tmp/users2.sql", "Table structure for table `users`")
		assert.NoError(err)
		if !stringFound {
			stringFound, err = fileutil.FgrepStringInFile("tmp/users2.sql", "Name: users; Type: TABLE")
			assert.NoError(err)
		}
		assert.True(stringFound)

		err = app.MutagenSyncFlush()
		assert.NoError(err)

		err = fileutil.PurgeDirectory("tmp")
		assert.NoError(err)

		// Capture to stdout without gzip compression
		stdout := util.CaptureStdOut()
		err = app.ExportDB("", "", "db")
		assert.NoError(err)
		o := stdout()
		assert.Regexp(regexp.MustCompile("CREATE TABLE.*users"), o)

		// Export an alternate database
		importPath = filepath.Join(testDir, "testdata", t.Name(), dbType, "users.sql")
		err = app.ImportDB(importPath, "", false, false, "anotherdb")
		require.NoError(t, err)
		err = app.ExportDB("tmp/anotherdb.sql.gz", "gzip", "anotherdb")
		assert.NoError(err)
		importPath = "tmp/anotherdb.sql.gz"
		err = app.ImportDB(importPath, "", false, false, "thirddb")
		assert.NoError(err)

		c := map[string]string{
			nodeps.MariaDB:  fmt.Sprintf(`echo "SELECT COUNT(*) FROM users;" | mysql -N thirddb`),
			nodeps.Postgres: fmt.Sprintf(`echo "SELECT COUNT(*) FROM users;" | psql -t -q thirddb`),
		}
		out, stderr, err := app.Exec(&ddevapp.ExecOpts{
			Service: "db",
			Cmd:     c[dbType],
		})
		assert.NoError(err, "stdout=%s stderr=%s", out, stderr)
		out = strings.Trim(out, "\n")
		out = strings.Trim(out, " ")
		assert.Equal("2", out)
	}

	runTime()
}

// readLastLine opens the fileName listed and returns the last
// 80 bytes of the file
func readLastLine(fileName string) (string, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return "", err
	}
	defer file.Close()

	buf := make([]byte, 80)
	stat, err := os.Stat(fileName)
	if err != nil {
		return "", err
	}
	start := stat.Size() - 80
	_, err = file.ReadAt(buf, start)
	if err != nil {
		return "", err
	}
	return string(buf), nil
}

// TestDdevFullSiteSetup tests a full import-db and import-files and then looks to see if
// we have a spot-test success hit on a URL
func TestDdevFullSiteSetup(t *testing.T) {
	assert := asrt.New(t)
	app := &ddevapp.DdevApp{}

	for _, site := range TestSites {
		switchDir := site.Chdir()
		defer switchDir()
		runTime := util.TimeTrack(time.Now(), fmt.Sprintf("%s DdevFullSiteSetup", site.Name))
		t.Logf("== BEGIN TestDdevFullSiteSetup for %s\n", site.Name)
		testcommon.ClearDockerEnv()
		err := app.Init(site.Dir)
		assert.NoError(err)

		err = app.Stop(true, false)
		require.NoError(t, err)

		// TestPkgPHP uses mostly stuff from Drupal6, but we'll set the type to php
		if site.Name == "TestPkgPHP" {
			app.Type = nodeps.AppTypePHP
			app.UploadDir = "files"
			err = os.MkdirAll(app.GetHostUploadDirFullPath(), 0755)
			assert.NoError(err)
		}

		// Get files before start, as syncing can start immediately.
		if site.FilesTarballURL != "" {
			_, tarballPath, err := testcommon.GetCachedArchive(site.Name, "local-tarballs-files", "", site.FilesTarballURL)
			require.NoError(t, err)
			err = app.ImportFiles(tarballPath, "")
			assert.NoError(err)
		}

		// Running WriteConfig assures that settings.ddev.php gets written
		// so Drupal 8 won't try to set things unwriteable
		err = app.WriteConfig()
		assert.NoError(err)

		restoreOutput := util.CaptureUserOut()
		err = app.Start()
		assert.NoError(err)
		out := restoreOutput()
		assert.NotContains(out, "Unable to create settings file")

		// Validate PHPMyAdmin is working and database named db is present
		_, _ = testcommon.EnsureLocalHTTPContent(t, app.GetHTTPURL()+":8036/index.php?route=/database/structure&server=1&db=db", "Database:          db")
		// Validate MailHog is working and "connected"
		_, _ = testcommon.EnsureLocalHTTPContent(t, app.GetHTTPURL()+":8025/#", "Connected")

		settingsLocation, err := app.DetermineSettingsPathLocation()
		assert.NoError(err)

		if app.Type != nodeps.AppTypeShopware6 {
			assert.Equal(filepath.Dir(settingsLocation), filepath.Dir(app.SiteSettingsPath))
		}
		if nodeps.ArrayContainsString([]string{"drupal6", "drupal7"}, app.Type) {
			assert.FileExists(filepath.Join(filepath.Dir(app.SiteSettingsPath), "drushrc.php"))
		}

		if site.DBTarURL != "" {
			_, cachedArchive, err := testcommon.GetCachedArchive(site.Name, site.Name+"_siteTarArchive", "", site.DBTarURL)
			if err != nil {
				assert.NoError(err)
				continue
			}
			err = app.ImportDB(cachedArchive, "", false, false, "db")
			if err != nil {
				assert.NoError(err, "failed to import-db with dbtarball %s, app.Type=%s, database=%s", site.DBTarURL, app.Type, app.Database.Type+":"+app.Database.Version)
				continue
			}
		}

		startErr := app.StartAndWait(2)
		if startErr != nil {
			appLogs, getLogsErr := ddevapp.GetErrLogsFromApp(app, startErr)
			assert.NoError(getLogsErr)
			t.Fatalf("app.StartAndWait() failure err=%v; logs:\n=====\n%s\n=====\n", startErr, appLogs)
		}

		// Test static content.
		_, _ = testcommon.EnsureLocalHTTPContent(t, app.GetPrimaryURL()+site.Safe200URIWithExpectation.URI, site.Safe200URIWithExpectation.Expect)
		// Test dynamic php + database content.
		rawurl := app.GetPrimaryURL() + site.DynamicURI.URI
		body, resp, err := testcommon.GetLocalHTTPResponse(t, rawurl, 120)
		assert.NoError(err, "GetLocalHTTPResponse returned err on project=%s rawurl %s, resp=%v: %v", site.Name, rawurl, resp, err)
		if err != nil && strings.Contains(err.Error(), "container ") {
			logs, err := ddevapp.GetErrLogsFromApp(app, err)
			assert.NoError(err)
			t.Fatalf("Logs after GetLocalHTTPResponse: %s", logs)
		}
		assert.Contains(body, site.DynamicURI.Expect, "expected %s on project %s", site.DynamicURI.Expect, site.Name)

		// Load an image from the files section
		if site.FilesImageURI != "" {
			_, resp, err := testcommon.GetLocalHTTPResponse(t, app.GetPrimaryURL()+site.FilesImageURI)
			assert.NoError(err, "failed ImageURI response on project %s", site.Name)
			if err != nil && resp != nil {
				assert.Equal("image/jpeg", resp.Header["Content-Type"][0])
			}
		}

		// Make sure we can do a simple hit against the host-mount of web container.
		_, _ = testcommon.EnsureLocalHTTPContent(t, app.GetWebContainerDirectHTTPURL()+site.Safe200URIWithExpectation.URI, site.Safe200URIWithExpectation.Expect)

		// Project type 'php' should fail ImportFiles if no upload_dir is provided
		if app.Type == nodeps.AppTypePHP {
			app.UploadDir = ""
			err = app.WriteConfig()
			assert.NoError(err)
			_, tarballPath, err := testcommon.GetCachedArchive(site.Name, "local-tarballs-files", "", site.FilesTarballURL)
			if err != nil {
				assert.NoError(err, "GetCachedArchive failed on project %s", site.Name)
				continue
			}
			err = app.ImportFiles(tarballPath, "")
			assert.Error(err)
			assert.Contains(err.Error(), "No upload_dir is set")
		}
		// We don't want all the projects running at once.
		err = app.Stop(true, false)
		assert.NoError(err)

		runTime()
		switchDir()
	}
	fmt.Print()
}

// TestWriteableFilesDirectory tests to make sure that files created on host are writable on container
// and files created in container are correct user on host.
func TestWriteableFilesDirectory(t *testing.T) {
	assert := asrt.New(t)
	origDir, _ := os.Getwd()
	app := &ddevapp.DdevApp{}
	site := TestSites[0]
	runTime := util.TimeTrack(time.Now(), fmt.Sprintf("%s TestWritableFilesDirectory", site.Name))

	testcommon.ClearDockerEnv()
	err := app.Init(site.Dir)
	assert.NoError(err)
	t.Cleanup(func() {
		err = os.Chdir(origDir)
		assert.NoError(err)
		err = app.Stop(true, false)
		assert.NoError(err)
	})
	err = os.Chdir(site.Dir)
	require.NoError(t, err)

	// Not all the example projects have an upload dir, so create it just in case
	err = os.MkdirAll(app.GetHostUploadDirFullPath(), 0777)
	assert.NoError(err)
	err = app.Start()
	require.NoError(t, err)

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
	case nodeps.AppTypeWordPress, nodeps.AppTypeTYPO3, nodeps.AppTypePHP:
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

	err = app.MutagenSyncFlush()
	assert.NoError(err)

	_, _, createFileErr := app.Exec(&ddevapp.ExecOpts{
		Service: "web",
		Cmd:     "echo 'content created inside container\n' >" + inContainerRelativePath,
	})
	assert.NoError(createFileErr)

	err = app.MutagenSyncFlush()
	assert.NoError(err)

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
	case nodeps.AppTypeWordPress, nodeps.AppTypeTYPO3, nodeps.AppTypePHP:
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

	err = app.MutagenSyncFlush()
	assert.NoError(err)

	// if the file exists, add to it. We don't want to add if it's not already there.
	_, _, err = app.Exec(&ddevapp.ExecOpts{
		Service: "web",
		Cmd:     "if [ -f " + inContainerRelativePath + " ]; then echo 'content added inside container\n' >>" + inContainerRelativePath + "; fi",
	})
	assert.NoError(err)
	// grep the file for both the content added on host and that added in container.
	_, _, err = app.Exec(&ddevapp.ExecOpts{
		Service: "web",
		Cmd:     "grep 'base content was inserted on the host' " + inContainerRelativePath + "&& grep 'content added inside container' " + inContainerRelativePath,
	})
	assert.NoError(err)

	err = app.Stop(true, false)
	assert.NoError(err)

	runTime()
}

// TestDdevImportFilesDir tests that "ddev import-files" can successfully import non-archive directories
func TestDdevImportFilesDir(t *testing.T) {
	assert := asrt.New(t)
	origDir, _ := os.Getwd()
	app := &ddevapp.DdevApp{}

	// Create a dummy directory to test non-archive imports
	importDir, err := os.MkdirTemp("", t.Name())
	assert.NoError(err)
	fileNames := make([]string, 0)
	for i := 0; i < 5; i++ {
		fileName := uuid.New().String()
		fileNames = append(fileNames, fileName)

		fullPath := filepath.Join(importDir, fileName)
		err = os.WriteFile(fullPath, []byte(fileName), 0644)
		assert.NoError(err)
	}

	for _, site := range TestSites {
		if site.FilesTarballURL == "" && site.FilesZipballURL == "" {
			t.Logf("== SKIP TestDdevImportFilesDir for %s (FilesTarballURL and FilesZipballURL are not provided)\n", site.Name)
			continue
		}
		runTime := util.TimeTrack(time.Now(), fmt.Sprintf("%s %s", site.Name, t.Name()))
		t.Logf("== BEGIN TestDdevImportFilesDir for %s\n", site.Name)

		testcommon.ClearDockerEnv()
		err = app.Init(site.Dir)
		if err != nil {
			assert.NoError(err, "failed app.Init for %s, continuing", site.Name)
			continue
		}

		t.Cleanup(func() {
			err = os.Chdir(origDir)
			assert.NoError(err)
			err = app.Stop(true, false)
			assert.NoError(err)
		})
		err = os.Chdir(site.Dir)
		require.NoError(t, err)

		// Function under test
		if app.GetUploadDir() == "" {
			continue
		}
		err = app.ImportFiles(importDir, "")
		if err != nil {
			assert.NoError(err, "failed importing files directory %s for site %s: %v", importDir, site.Name, err)
			continue
		}

		// Confirm contents of destination dir after import
		absUploadDir := app.GetHostUploadDirFullPath()
		uploadedFilesDirEntrySlice, err := os.ReadDir(absUploadDir)
		assert.NoError(err)

		uploadedFilesMap := map[string]bool{}
		for _, de := range uploadedFilesDirEntrySlice {
			uploadedFilesMap[filepath.Base(de.Name())] = true
		}

		for _, expectedFile := range fileNames {
			assert.True(uploadedFilesMap[expectedFile], "Expected file %s not found for site: %s", expectedFile, site.Name)
		}

		runTime()
	}
}

// TestDdevImportFiles tests the functionality that is called when "ddev import-files" is executed
func TestDdevImportFiles(t *testing.T) {
	origDir, _ := os.Getwd()
	assert := asrt.New(t)
	app := &ddevapp.DdevApp{}

	for _, site := range TestSites {
		if site.Type == "php" || (site.FilesTarballURL == "" && site.FilesZipballURL == "" && site.FullSiteTarballURL == "") {
			t.Logf("== SKIP TestDdevImportFiles for %s (FilesTarballURL and FilesZipballURL are not provided) or type php\n", site.Name)
			continue
		}

		t.Logf("== BEGIN TestDdevImportFiles for %s\n", site.Name)
		runTime := util.TimeTrack(time.Now(), fmt.Sprintf("%s %s", site.Name, t.Name()))

		testcommon.ClearDockerEnv()
		err := app.Init(site.Dir)
		assert.NoError(err)
		t.Cleanup(func() {
			err = os.Chdir(origDir)
			assert.NoError(err)
			err = app.Stop(true, false)
			assert.NoError(err)
		})
		err = os.Chdir(site.Dir)
		require.NoError(t, err)

		app.Hooks = map[string][]ddevapp.YAMLTask{"post-import-files": {{"exec-host": "touch hello-post-import-files-" + app.Name}}, "pre-import-files": {{"exec-host": "touch hello-pre-import-files-" + app.Name}}}

		if site.FilesTarballURL != "" && app.GetUploadDir() != "" {
			_, tarballPath, err := testcommon.GetCachedArchive(site.Name, "local-tarballs-files", "", site.FilesTarballURL)
			assert.NoError(err)
			err = app.ImportFiles(tarballPath, "")
			if err != nil {
				assert.NoError(err, "failed importing files tarball %s for site %s: %v", tarballPath, site.Name, err)
				continue
			}
		}

		if site.FilesZipballURL != "" {
			_, zipballPath, err := testcommon.GetCachedArchive(site.Name, "local-zipballs-files", "", site.FilesZipballURL)
			assert.NoError(err)
			err = app.ImportFiles(zipballPath, "")
			if err != nil {
				assert.NoError(err)
				continue
			}
		}

		if site.FullSiteTarballURL != "" && site.FullSiteArchiveExtPath != "" {
			_, siteTarPath, err := testcommon.GetCachedArchive(site.Name, "local-site-tar", "", site.FullSiteTarballURL)
			if err != nil {
				assert.NoError(err)
				continue
			}
			err = app.ImportFiles(siteTarPath, site.FullSiteArchiveExtPath)
			if err != nil {
				assert.NoError(err)
				continue
			}
		}

		// Import trivial archive with various types of archive/compression and verify that files arrive
		extensions := []string{"tgz", "zip", "tar.xz", "tar.bz2", "tar.gz"}
		for _, ext := range extensions {
			tarballPath := filepath.Join(origDir, "testdata", t.Name(), "files."+ext)
			_ = os.RemoveAll(filepath.Join(app.GetHostUploadDirFullPath(), "files."+ext+".txt"))
			err = app.ImportFiles(tarballPath, "")
			if err != nil {
				assert.NoError(err)
				continue
			}
			assert.FileExists(filepath.Join(app.GetHostUploadDirFullPath(), ext+".txt"))
		}

		assert.FileExists("hello-pre-import-files-" + app.Name)
		assert.FileExists("hello-post-import-files-" + app.Name)
		err = os.Remove("hello-pre-import-files-" + app.Name)
		assert.NoError(err)
		err = os.Remove("hello-post-import-files-" + app.Name)
		assert.NoError(err)

		runTime()
	}
}

// TestDdevImportFilesCustomUploadDir ensures that files are imported to a custom upload directory when requested
func TestDdevImportFilesCustomUploadDir(t *testing.T) {
	assert := asrt.New(t)
	app := &ddevapp.DdevApp{}

	for _, site := range TestSites {
		switchDir := site.Chdir()
		runTime := util.TimeTrack(time.Now(), fmt.Sprintf("%s %s", site.Name, t.Name()))
		t.Logf("== BEGIN TestDdevImportFilesCustomUploadDir for %s\n", site.Name)

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
			require.NoError(t, err)
			err = app.ImportFiles(tarballPath, "")
			assert.NoError(err)

			// Ensure upload dir isn't empty
			dirEntrySlice, err := os.ReadDir(absUploadDir)
			assert.NoError(err)
			assert.NotEmpty(dirEntrySlice)
		}

		if site.FilesZipballURL != "" {
			_, zipballPath, err := testcommon.GetCachedArchive(site.Name, "local-zipballs-files", "", site.FilesZipballURL)
			require.NoError(t, err)
			err = app.ImportFiles(zipballPath, "")
			assert.NoError(err)

			// Ensure upload dir isn't empty
			dirEntrySlice, err := os.ReadDir(absUploadDir)
			assert.NoError(err)
			assert.NotEmpty(dirEntrySlice)
		}

		if site.FullSiteTarballURL != "" && site.FullSiteArchiveExtPath != "" {
			_, siteTarPath, err := testcommon.GetCachedArchive(site.Name, "local-site-tar", "", site.FullSiteTarballURL)
			require.NoError(t, err)
			err = app.ImportFiles(siteTarPath, site.FullSiteArchiveExtPath)
			assert.NoError(err)

			// Ensure upload dir isn't empty
			dirEntrySlice, err := os.ReadDir(absUploadDir)
			assert.NoError(err)
			assert.NotEmpty(dirEntrySlice)
		}

		runTime()
		switchDir()
	}
}

// TestDdevExec tests the execution of commands inside a docker container of a site.
func TestDdevExec(t *testing.T) {
	assert := asrt.New(t)
	app := &ddevapp.DdevApp{}
	testDir, _ := os.Getwd()

	site := TestSites[0]
	switchDir := site.Chdir()
	runTime := util.TimeTrack(time.Now(), fmt.Sprintf("%s DdevExec", site.Name))

	// Make a dummy service out of busybox
	err := fileutil.CopyFile(filepath.Join(testDir, "testdata", t.Name(), "docker-compose.busybox.yaml"), filepath.Join(site.Dir, ".ddev", "docker-compose.busybox.yaml"))
	assert.NoError(err)

	err = app.Init(site.Dir)
	assert.NoError(err)

	t.Cleanup(func() {
		app.Hooks = nil
		err = app.Stop(true, false)
		assert.NoError(err)
		err = app.WriteConfig()
		assert.NoError(err)
		err = os.RemoveAll(filepath.Join(site.Dir, ".ddev", "docker-compose.busybox.yaml"))
		assert.NoError(err)
	})

	app.Hooks = map[string][]ddevapp.YAMLTask{"post-exec": {{"exec-host": "touch hello-post-exec-" + app.Name}}, "pre-exec": {{"exec-host": "touch hello-pre-exec-" + app.Name}}}

	startErr := app.Start()
	if startErr != nil {
		logs, err := ddevapp.GetErrLogsFromApp(app, startErr)
		assert.NoError(err)
		t.Fatalf("app.Start() failed err=%v, logs from broken container:\n=======\n%s\n========\n", startErr, logs)
	}

	out, _, err := app.Exec(&ddevapp.ExecOpts{
		Service: "web",
		Cmd:     "pwd",
	})
	assert.NoError(err)
	assert.Contains(out, "/var/www/html")

	err = app.MutagenSyncFlush()
	assert.NoError(err)

	assert.FileExists("hello-pre-exec-" + app.Name)
	assert.FileExists("hello-post-exec-" + app.Name)
	err = os.Remove("hello-pre-exec-" + app.Name)
	assert.NoError(err)
	err = os.Remove("hello-post-exec-" + app.Name)
	assert.NoError(err)

	out, _, err = app.Exec(&ddevapp.ExecOpts{
		Service: "web",
		Dir:     "/usr/local",
		Cmd:     "pwd",
	})
	assert.NoError(err)
	assert.Contains(out, "/usr/local")

	// Try out a execRaw example
	out, _, err = app.Exec(&ddevapp.ExecOpts{
		RawCmd: []string{"ls", "/usr/local"},
	})
	assert.NoError(err)
	assert.True(strings.HasPrefix(out, "bin\netc\n"))

	_, _, err = app.Exec(&ddevapp.ExecOpts{
		Service: "db",
		Cmd:     "mysql -e 'DROP DATABASE db;'",
	})
	assert.NoError(err)
	_, _, err = app.Exec(&ddevapp.ExecOpts{
		Service: "db",
		Cmd:     "mysql information_schema -e 'CREATE DATABASE db;'",
	})
	assert.NoError(err)

	// Make sure that exec works on non-ddev container like busybox as well
	grepOpts := &ddevapp.ExecOpts{
		Service: "busybox",
		Cmd:     "ls | grep bin",
	}
	_, _, err = app.Exec(grepOpts)
	assert.NoError(err)

	_, stderr, err := app.Exec(&ddevapp.ExecOpts{
		Service: "busybox",
		Cmd:     "echo $ENVDOESNOTEXIST",
	})
	assert.Error(err)
	assert.Contains(stderr, "parameter not set")

	errorOpts := &ddevapp.ExecOpts{
		Service: "busybox",
		Cmd:     "this is an error;",
	}
	_, stderr, err = app.Exec(errorOpts)
	assert.Error(err)
	assert.Contains(stderr, "this: not found")
	err = app.ExecWithTty(errorOpts)
	assert.Error(err)
	assert.Contains(stderr, "this: not found")

	// Now kill the busybox service and make sure that responses to app.Exec are correct
	client := dockerutil.GetDockerClient()
	bbc, err := dockerutil.FindContainerByName(fmt.Sprintf("ddev-%s-%s", app.Name, "busybox"))
	require.NoError(t, err)
	require.NotEmpty(t, bbc)
	err = client.StopContainer(bbc.ID, 2)
	assert.NoError(err)

	simpleOpts := ddevapp.ExecOpts{
		Service: "busybox",
		Cmd:     "ls",
	}
	_, _, err = app.Exec(&simpleOpts)
	assert.Error(err)
	assert.Contains(err.Error(), "not currently running")
	assert.Contains(err.Error(), "state=exited")

	err = client.RemoveContainer(docker.RemoveContainerOptions{ID: bbc.ID, Force: true})
	assert.NoError(err)
	_, _, err = app.Exec(&simpleOpts)
	assert.Error(err)
	assert.Contains(err.Error(), "does not exist")
	assert.Contains(err.Error(), "state=doesnotexist")

	// Test use of RawCmd, should see the single quotes in there
	stdout, stderr, err := app.Exec(&ddevapp.ExecOpts{
		RawCmd: []string{"bash", "-c", `echo '"hi there"'`},
	})
	assert.NoError(err)
	assert.Contains(stdout, `"hi there"`, "stdout=%s, stderr=%s", stdout, stderr)

	runTime()
	switchDir()
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
	runTime := util.TimeTrack(time.Now(), fmt.Sprintf("%s DdevLogs", site.Name))

	err := app.Init(site.Dir)
	assert.NoError(err)

	t.Cleanup(func() {
		err = app.Stop(true, false)
		assert.NoError(err)
	})

	startErr := app.StartAndWait(0)
	if startErr != nil {
		logs, err := ddevapp.GetErrLogsFromApp(app, startErr)
		assert.NoError(err)
		t.Fatalf("app.Start failed, err=%v, logs=\n========\n%s\n===========\n", startErr, logs)
	}

	out, err := app.CaptureLogs("web", false, "")
	assert.NoError(err)
	assert.Contains(out, "Server started")

	out, err = app.CaptureLogs("db", false, "")
	assert.NoError(err)
	assert.Contains(out, "MySQL init process done. Ready for start up.")

	// Test that we can get logs when project is stopped also
	err = app.Pause()
	assert.NoError(err)

	out, err = app.CaptureLogs("web", false, "")
	assert.NoError(err)
	assert.Contains(out, "Server started")

	out, err = app.CaptureLogs("db", false, "")
	assert.NoError(err)
	assert.Contains(out, "MySQL init process done. Ready for start up.")

	runTime()
	switchDir()
}

// TestDdevPause tests the functionality that is called when "ddev pause" is executed
func TestDdevPause(t *testing.T) {
	assert := asrt.New(t)

	app := &ddevapp.DdevApp{}

	site := TestSites[0]
	switchDir := site.Chdir()
	runTime := util.TimeTrack(time.Now(), fmt.Sprintf("%s DdevStop", site.Name))

	testcommon.ClearDockerEnv()
	err := app.Init(site.Dir)
	assert.NoError(err)
	err = app.StartAndWait(0)
	app.Hooks = map[string][]ddevapp.YAMLTask{"post-pause": {{"exec-host": "touch hello-post-pause-" + app.Name}}, "pre-pause": {{"exec-host": "touch hello-pre-pause-" + app.Name}}}

	defer func() {
		app.Hooks = nil
		_ = app.WriteConfig()
		_ = app.Stop(true, false)
	}()
	require.NoError(t, err)
	err = app.Pause()
	assert.NoError(err)

	for _, containerType := range [3]string{"web", "db", "dba"} {
		containerName, err := constructContainerName(containerType, app)
		assert.NoError(err)
		check, err := testcommon.ContainerCheck(containerName, "exited")
		assert.NoError(err)
		assert.True(check, containerType, "container has exited")
	}
	assert.FileExists("hello-pre-pause-" + app.Name)
	assert.FileExists("hello-post-pause-" + app.Name)
	err = os.Remove("hello-pre-pause-" + app.Name)
	assert.NoError(err)
	err = os.Remove("hello-post-pause-" + app.Name)
	assert.NoError(err)

	runTime()
	switchDir()
}

// TestDdevStopMissingDirectory tests that the 'ddev stop' command works properly on sites with missing directories or ddev configs.
func TestDdevStopMissingDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping because unreliable on Windows")
	}

	assert := asrt.New(t)

	site := TestSites[0]
	testcommon.ClearDockerEnv()
	app := &ddevapp.DdevApp{}
	err := app.Init(site.Dir)
	assert.NoError(err)

	startErr := app.StartAndWait(0)
	//nolint: errcheck
	defer app.Stop(true, false)
	if startErr != nil {
		logs, err := ddevapp.GetErrLogsFromApp(app, startErr)
		assert.NoError(err)
		t.Fatalf("app.StartAndWait failed err=%v logs from broken container: \n=======\n%s\n========\n", startErr, logs)
	}

	tempPath := testcommon.CreateTmpDir("site-copy")
	siteCopyDest := filepath.Join(tempPath, "site")
	defer removeAllErrCheck(tempPath, assert)

	_ = app.Stop(false, false)

	// Move the site directory to a temp location to mimic a missing directory.
	err = os.Rename(site.Dir, siteCopyDest)
	assert.NoError(err)

	//nolint: errcheck
	defer os.Rename(siteCopyDest, site.Dir)

	// ddev stop (in cmd) actually does the check for missing project files,
	// so we imitate that here.
	err = ddevapp.CheckForMissingProjectFiles(app)
	assert.Error(err)
	if err != nil {
		assert.Contains(err.Error(), "If you would like to continue using ddev to manage this project please restore your files to that directory.")
	}
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

	app.Hooks = map[string][]ddevapp.YAMLTask{"post-describe": {{"exec-host": "touch hello-post-describe-" + app.Name}}, "pre-describe": {{"exec-host": "touch hello-pre-describe-" + app.Name}}}

	startErr := app.StartAndWait(0)
	defer func() {
		_ = app.Stop(true, false)
		app.Hooks = nil
		_ = app.WriteConfig()
	}()
	// If we have a problem starting, get the container logs and output.
	if startErr != nil {
		out, logsErr := app.CaptureLogs("web", false, "")
		assert.NoError(logsErr)

		healthcheck, inspectErr := exec.RunCommandPipe("sh", []string{"-c", fmt.Sprintf("docker inspect ddev-%s-web|jq -r '.[0].State.Health.Log[-1]'", app.Name)})
		assert.NoError(inspectErr)

		t.Fatalf("app.StartAndWait(%s) failed: %v, \nweb container healthcheck='%s', \n== web container logs=\n%s\n== END web container logs ==", site.Name, err, healthcheck, out)
	}

	desc, err := app.Describe(false)
	assert.NoError(err)
	assert.EqualValues(ddevapp.SiteRunning, desc["status"], "")
	assert.EqualValues(app.GetName(), desc["name"])
	assert.EqualValues(ddevapp.RenderHomeRootedDir(app.GetAppRoot()), desc["shortroot"])
	assert.EqualValues(app.GetAppRoot(), desc["approot"])
	assert.EqualValues(app.GetPhpVersion(), desc["php_version"])

	assert.FileExists("hello-pre-describe-" + app.Name)
	assert.FileExists("hello-post-describe-" + app.Name)
	err = os.Remove("hello-pre-describe-" + app.Name)
	assert.NoError(err)
	err = os.Remove("hello-post-describe-" + app.Name)
	assert.NoError(err)

	// Now stop it and test behavior.
	err = app.Pause()
	assert.NoError(err)

	desc, err = app.Describe(false)
	assert.NoError(err)
	assert.EqualValues(ddevapp.SitePaused, desc["status"])

	switchDir()
}

// TestDdevDescribeMissingDirectory tests that the describe command works properly on sites with missing directories or ddev configs.
func TestDdevDescribeMissingDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping because unreliable on Windows")
	}

	assert := asrt.New(t)
	site := TestSites[0]
	tempPath := testcommon.CreateTmpDir("site-copy")
	siteCopyDest := filepath.Join(tempPath, "site")
	defer removeAllErrCheck(tempPath, assert)

	app := &ddevapp.DdevApp{}
	err := app.Init(site.Dir)
	assert.NoError(err)
	startErr := app.StartAndWait(0)
	//nolint: errcheck
	defer app.Stop(true, false)
	if startErr != nil {
		logs, err := ddevapp.GetErrLogsFromApp(app, startErr)
		assert.NoError(err)
		t.Fatalf("app.StartAndWait failed err=%v logs from broken container: \n=======\n%s\n========\n", startErr, logs)
	}
	// Move the site directory to a temp location to mimic a missing directory.
	err = app.Stop(false, false)
	assert.NoError(err)
	err = os.Rename(site.Dir, siteCopyDest)
	assert.NoError(err)

	desc, err := app.Describe(false)
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

	// Stop/Remove all sites, which should get the router out of there.
	for _, site := range TestSites {
		switchDir := site.Chdir()

		testcommon.ClearDockerEnv()
		err := app.Init(site.Dir)
		assert.NoError(err)

		if app.SiteStatus() == ddevapp.SiteRunning || app.SiteStatus() == ddevapp.SitePaused {
			err = app.Stop(true, false)
			assert.NoError(err)
		}

		switchDir()
	}

	// Now start one site, it's hard to get router to behave without one site.
	site := TestSites[0]
	testcommon.ClearDockerEnv()

	err := app.Init(site.Dir)
	assert.NoError(err)
	startErr := app.StartAndWait(5)
	//nolint: errcheck
	defer app.Stop(true, false)
	if startErr != nil {
		appLogs, getLogsErr := ddevapp.GetErrLogsFromApp(app, startErr)
		assert.NoError(getLogsErr)
		t.Fatalf("app.StartAndWait() failure; err=%v logs:\n=====\n%s\n=====\n", startErr, appLogs)
	}

	app, err = ddevapp.GetActiveApp(site.Name)
	if err != nil {
		t.Fatalf("Failed to GetActiveApp(%s), err:%v", site.Name, err)
	}
	startErr = app.StartAndWait(5)
	//nolint: errcheck
	defer app.Stop(true, false)
	if startErr != nil {
		appLogs, getLogsErr := ddevapp.GetErrLogsFromApp(app, startErr)
		assert.NoError(getLogsErr)
		t.Fatalf("app.StartAndWait() failure err=%v logs:\n=====\n%s\n=====\n", startErr, appLogs)
	}

	// Stop the router using code from StopRouterIfNoContainers().
	// StopRouterIfNoContainers can't be used here because it checks to see if containers are running
	// and doesn't do its job as a result.
	dest := ddevapp.RouterComposeYAMLPath()
	_, _, err = dockerutil.ComposeCmd([]string{dest}, "-p", ddevapp.RouterProjectName, "down")
	assert.NoError(err, "Failed to stop router using docker-compose, err=%v", err)

	// Occupy port 80 using docker busybox trick, then see if we can start router.
	// This is done with docker so that we don't have to use explicit sudo
	containerID, err := exec.RunHostCommand("sh", "-c", "docker run -d -p80:80 --rm busybox:stable sleep 100 2>/dev/null")
	if err != nil {
		t.Fatalf("Failed to run docker command to occupy port 80, err=%v output=%v", err, containerID)
	}
	containerID = strings.TrimSpace(containerID)

	// Now try to start the router. It should fail because the port is occupied.
	err = ddevapp.StartDdevRouter()
	assert.Error(err, "Failure: router started even though port 80 was occupied")

	// Remove our dummy busybox docker container.
	out, err := exec.RunHostCommand("docker", "rm", "-f", containerID)
	assert.NoError(err, "Failed to docker rm the port-occupier container, err=%v output=%v", err, out)
}

// TestCleanupWithoutCompose ensures app containers can be properly cleaned up without a docker-compose config file present.
func TestCleanupWithoutCompose(t *testing.T) {
	assert := asrt.New(t)

	// Skip test because we can't rename folders while they're in use if running on Windows or with mutagen.
	if runtime.GOOS == "windows" || nodeps.MutagenEnabledDefault == true {
		t.Skip("Skipping test TestCleanupWithoutCompose; doesn't work on Windows or mutagen because of renaming of whole project directory")
	}

	origDir, _ := os.Getwd()
	site := TestSites[0]

	app := &ddevapp.DdevApp{}

	testcommon.ClearDockerEnv()
	err := app.Init(site.Dir)
	assert.NoError(err)

	// Ensure we have a site started so we have something to cleanup

	err = app.Start()
	require.NoError(t, err)

	// Setup by creating temp directory and nesting a folder for our site.
	tempPath := testcommon.CreateTmpDir("site-copy")
	siteCopyDest := filepath.Join(tempPath, "site")

	// Move site directory to a temp directory to mimic a missing directory.
	err = os.Rename(site.Dir, siteCopyDest)
	assert.NoError(err)

	t.Cleanup(func() {
		err = os.Chdir(origDir)
		assert.NoError(err)
		err = app.Stop(true, false)
		assert.NoError(err)
		// Move the site directory back to its original location.
		err = os.Rename(siteCopyDest, site.Dir)
		assert.NoError(err)
		err = os.RemoveAll(tempPath)
		assert.NoError(err)
	})

	// Call the Stop command()
	// Notice that we set the removeData parameter to true.
	// This gives us added test coverage over sites with missing directories
	// by ensuring any associated database files get cleaned up as well.
	err = app.Stop(true, false)
	assert.NoError(err)
	assert.Empty(globalconfig.DdevGlobalConfig.ProjectList[app.Name])
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

// TestGetAppsEmpty ensures that GetActiveProjects returns an empty list when no applications are running.
func TestGetAppsEmpty(t *testing.T) {
	assert := asrt.New(t)

	// Ensure test sites are removed
	for _, site := range TestSites {
		app := &ddevapp.DdevApp{}

		switchDir := site.Chdir()

		testcommon.ClearDockerEnv()
		err := app.Init(site.Dir)
		assert.NoError(err)

		if app.SiteStatus() != ddevapp.SiteStopped {
			err = app.Stop(true, false)
			assert.NoError(err)
		}
		switchDir()
	}

	apps := ddevapp.GetActiveProjects()
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

type URLRedirectExpectations struct {
	scheme              string
	uri                 string
	expectedRedirectURI string
}

// TestAppdirAlreadyInUse tests trying to start a project in an already-used
// directory will fail
func TestAppdirAlreadyInUse(t *testing.T) {
	assert := asrt.New(t)
	originalProjectName := t.Name() + "-originalproject"
	secondProjectName := t.Name() + "_secondproject"
	origDir, _ := os.Getwd()
	// Create a temporary directory and switch to it.
	testDir := testcommon.CreateTmpDir(t.Name())

	app, err := ddevapp.NewApp(testDir, false)

	t.Cleanup(func() {
		app.Name = originalProjectName
		_ = app.Stop(true, false)
		app.Name = secondProjectName
		_ = app.Stop(true, false)
		err = os.Chdir(origDir)
		assert.NoError(err)
		err = os.RemoveAll(testDir)
		assert.NoError(err)
	})

	// Write/create global with the name "originalproject"
	app.Name = originalProjectName
	require.NoError(t, err)
	err = app.WriteConfig()
	require.NoError(t, err)
	err = app.Start()
	require.NoError(t, err)

	// Now change the project name and look for the complaint
	app.Name = secondProjectName

	err = app.Start()
	assert.Error(err)
	assert.Contains(err.Error(), "already contains a project named "+originalProjectName)
	err = app.Stop(true, false)
	assert.NoError(err)

	// Change back to original name
	app.Name = originalProjectName
	assert.NoError(err)

	err = app.Start()
	assert.NoError(err)

	// Now stop and make sure the behavior is same with everything stopped
	err = app.Stop(false, false)
	assert.NoError(err)

	// Now change the project name again and look for the error
	app.Name = secondProjectName
	err = app.Start()
	assert.Error(err)
	assert.Contains(err.Error(), "already contains a project named "+originalProjectName)
}

// TestHttpsRedirection tests to make sure that webserver and php redirect to correct
// scheme (http or https).
func TestHttpsRedirection(t *testing.T) {
	if nodeps.IsMacM1() {
		t.Skip("Skipping on mac M1 to ignore problems with 'connection reset by peer'")
	}
	if globalconfig.GetCAROOT() == "" {
		t.Skip("Skipping because MkcertCARoot is not set, no https")
	}

	assert := asrt.New(t)
	testcommon.ClearDockerEnv()
	packageDir, _ := os.Getwd()

	testDir := testcommon.CreateTmpDir(t.Name())
	appDir := filepath.Join(testDir, t.Name())
	err := fileutil.CopyDir(filepath.Join(packageDir, "testdata", t.Name()), appDir)
	assert.NoError(err)
	err = os.Chdir(appDir)
	assert.NoError(err)

	app, err := ddevapp.NewApp(appDir, true)
	assert.NoError(err)

	_ = app.Stop(true, false)

	t.Cleanup(func() {
		err = app.Stop(true, false)
		assert.NoError(err)
		err = os.Chdir(packageDir)
		assert.NoError(err)
		_ = os.RemoveAll(testDir)
	})

	expectations := []URLRedirectExpectations{
		{"https", "/subdir", "/subdir/"},
		{"https", "/redir_abs.php", "/landed.php"},
		{"https", "/redir_relative.php", "/landed.php"},
		{"http", "/subdir", "/subdir/"},
		{"http", "/redir_abs.php", "/landed.php"},
		{"http", "/redir_relative.php", "/landed.php"},
	}

	types := ddevapp.GetValidAppTypes()
	webserverTypes := []string{nodeps.WebserverNginxFPM, nodeps.WebserverApacheFPM}
	if os.Getenv("GOTEST_SHORT") != "" {
		types = []string{nodeps.AppTypePHP, nodeps.AppTypeDrupal8}
		webserverTypes = []string{nodeps.WebserverNginxFPM, nodeps.WebserverApacheFPM}
	}
	for _, projectType := range types {
		// TODO: Fix the laravel config so it can do the redir_abs.php successfully on nginx-fpm
		if projectType == nodeps.AppTypeLaravel {
			t.Log("Skipping laravel because it can't pass absolute redirect test, fix config")
			continue
		}
		for _, webserverType := range webserverTypes {
			app.WebserverType = webserverType
			app.Type = projectType
			err = app.WriteConfig()
			assert.NoError(err)

			// Do a start on the configured site.
			app, err = ddevapp.GetActiveApp("")
			assert.NoError(err)
			startErr := app.StartAndWait(5)
			assert.NoError(startErr, "app.Start() failed with projectType=%s, webserverType=%s", projectType, webserverType)
			if startErr != nil {
				appLogs, getLogsErr := ddevapp.GetErrLogsFromApp(app, startErr)
				assert.NoError(getLogsErr)
				t.Fatalf("app.StartAndWait failure; err=%v \n===== container logs ==\n%s\n", startErr, appLogs)
			}
			// Test for directory redirects under https and http
			for _, parts := range expectations {
				reqURL := parts.scheme + "://" + strings.ToLower(app.GetHostname()) + parts.uri
				//t.Logf("TestHttpsRedirection trying URL %s with webserver_type=%s", reqURL, webserverType)
				// Add extra hit to avoid occasional nil result
				_, _, _ = testcommon.GetLocalHTTPResponse(t, reqURL, 60)
				out, resp, err := testcommon.GetLocalHTTPResponse(t, reqURL, 60)

				require.NotNil(t, resp, "resp was nil for projectType=%s webserver_type=%s url=%s, err=%v, out='%s'", projectType, webserverType, reqURL, err, out)
				if resp != nil {
					locHeader := resp.Header.Get("Location")

					expectedRedirect := parts.expectedRedirectURI
					// However, if we're hitting redir_abs.php (or apache hitting directory), the redirect will be the whole url.
					if strings.Contains(parts.uri, "redir_abs.php") || webserverType != nodeps.WebserverNginxFPM {
						expectedRedirect = parts.scheme + "://" + strings.ToLower(app.GetHostname()) + parts.expectedRedirectURI
					}
					// Except the php relative redirect is always relative.
					if strings.Contains(parts.uri, "redir_relative.php") {
						expectedRedirect = parts.expectedRedirectURI
					}
					assert.EqualValues(locHeader, expectedRedirect, "For project type=%s webserver_type=%s url=%s expected redirect %s != actual %s", projectType, webserverType, reqURL, expectedRedirect, locHeader)
				}
			}
		}
	}
	// Change back to package dir. Lots of things will have to be cleaned up
	// in defers, and for windows we have to not be sitting in them.
	err = os.Chdir(packageDir)
	assert.NoError(err)
}

// TestMultipleComposeFiles checks to see if a set of docker-compose files gets
// properly loaded in the right order, with .ddev/.ddev-docker-compose*yaml first and
// with docker-compose.override.yaml last.
func TestMultipleComposeFiles(t *testing.T) {
	// Set up tests and give ourselves a working directory.
	assert := asrt.New(t)
	pwd, _ := os.Getwd()

	testDir := testcommon.CreateTmpDir(t.Name())
	//_ = os.Chdir(testDir)
	defer testcommon.CleanupDir(testDir)
	defer testcommon.Chdir(testDir)()

	err := fileutil.CopyDir(filepath.Join(pwd, "testdata", t.Name(), ".ddev"), filepath.Join(testDir, ".ddev"))
	assert.NoError(err)

	// Make sure that valid yaml files get properly loaded in the proper order
	app, err := ddevapp.NewApp(testDir, true)
	assert.NoError(err)
	//nolint: errcheck
	defer app.Stop(true, false)

	err = app.WriteConfig()
	assert.NoError(err)
	_, err = app.ReadConfig(true)
	require.NoError(t, err)
	err = app.WriteDockerComposeYAML()
	require.NoError(t, err)

	app, err = ddevapp.NewApp(testDir, true)
	assert.NoError(err)
	//nolint: errcheck
	defer app.Stop(true, false)

	desc, err := app.Describe(false)
	assert.NoError(err)
	_ = desc

	files, err := app.ComposeFiles()
	assert.NoError(err)
	require.NotEmpty(t, files)
	assert.Equal(4, len(files))
	require.Equal(t, app.GetConfigPath(".ddev-docker-compose-base.yaml"), files[0])
	require.Equal(t, app.GetConfigPath("docker-compose.override.yaml"), files[len(files)-1])

	require.NotEmpty(t, app.ComposeYaml)
	require.True(t, len(app.ComposeYaml) > 0)

	// Verify that the env var DUMMY_BASE got set by docker-compose.override.yaml
	if services, ok := app.ComposeYaml["services"].(map[interface{}]interface{}); ok {
		if w, ok := services["web"].(map[interface{}]interface{}); ok {
			if env, ok := w["environment"].(map[interface{}]interface{}); ok {
				// The docker-compose.override should have won with the value of DUMMY_BASE
				assert.Equal("override", env["DUMMY_BASE"])
				// But each of the DUMMY_COMPOSE_ONE/TWO/OVERRIDE which are unique
				// should come through fine.
				assert.Equal("1", env["DUMMY_COMPOSE_ONE"])
				assert.Equal("2", env["DUMMY_COMPOSE_TWO"])
				assert.Equal("override", env["DUMMY_COMPOSE_OVERRIDE"])
			} else {
				t.Error("Failed to parse environment")
			}
		} else {
			t.Error("failed to parse web service")
		}

	} else {
		t.Error("Unable to access ComposeYaml[services]")
	}

	_, err = app.ComposeFiles()
	assert.NoError(err)
}

// TestGetAllURLs ensures the GetAllURLs function returns the expected number of URLs,
// and include the direct web container URLs.
func TestGetAllURLs(t *testing.T) {
	assert := asrt.New(t)

	site := TestSites[0]
	runTime := util.TimeTrack(time.Now(), fmt.Sprintf("%s GetAllURLs", site.Name))

	testcommon.ClearDockerEnv()
	app := new(ddevapp.DdevApp)

	err := app.Init(site.Dir)
	assert.NoError(err)

	// Add some additional hostnames
	app.AdditionalHostnames = []string{"sub1", "sub2", "sub3"}

	err = app.WriteConfig()
	assert.NoError(err)

	err = app.StartAndWait(0)
	require.NoError(t, err)

	_, _, urls := app.GetAllURLs()

	// Convert URLs to map[string]bool
	urlMap := make(map[string]bool)
	for _, u := range urls {
		urlMap[u] = true
	}

	// We expect two URLs for each hostname (http/https) and two direct web container address.
	expectedNumUrls := len(app.GetHostnames())*2 + 2
	assert.Equal(len(urlMap), expectedNumUrls, "Unexpected number of URLs returned: %d", len(urlMap))

	// Ensure urlMap contains direct address of the web container
	webContainer, err := app.FindContainerByType("web")
	assert.NoError(err)
	require.NotEmpty(t, webContainer)

	expectedDirectAddress := app.GetWebContainerDirectHTTPSURL()
	if globalconfig.GetCAROOT() == "" {
		expectedDirectAddress = app.GetWebContainerDirectHTTPURL()
	}

	exists := urlMap[expectedDirectAddress]

	assert.True(exists, "URL list for app: %s does not contain direct web container address: %s", app.Name, expectedDirectAddress)

	// Multiple projects can't run at the same time with the fqdns, so we need to clean
	// up these for tests that run later.
	app.AdditionalFQDNs = []string{}
	app.AdditionalHostnames = []string{}
	err = app.WriteConfig()
	assert.NoError(err)

	err = app.Stop(true, false)
	assert.NoError(err)

	runTime()
}

// TestWebserverType checks that webserver_type:apache-fpm does the right thing
func TestWebserverType(t *testing.T) {
	assert := asrt.New(t)

	for _, site := range TestSites {
		runTime := util.TimeTrack(time.Now(), fmt.Sprintf("%s TestWebserverType", site.Name))

		app := new(ddevapp.DdevApp)

		err := app.Init(site.Dir)
		assert.NoError(err)

		// Copy our phpinfo into the docroot of testsite.
		pwd, err := os.Getwd()
		assert.NoError(err)
		err = fileutil.CopyFile(filepath.Join(pwd, "testdata", "servertype.php"), filepath.Join(app.AppRoot, app.Docroot, "servertype.php"))

		assert.NoError(err)
		for _, app.WebserverType = range []string{nodeps.WebserverApacheFPM, nodeps.WebserverNginxFPM} {

			err = app.WriteConfig()
			assert.NoError(err)

			testcommon.ClearDockerEnv()

			startErr := app.StartAndWait(30)
			//nolint: errcheck
			defer app.Stop(true, false)
			if startErr != nil {
				appLogs, getLogsErr := ddevapp.GetErrLogsFromApp(app, startErr)
				assert.NoError(getLogsErr)
				t.Fatalf("app.StartAndWait failure for WebserverType=%s; site.Name=%s; err=%v, logs:\n=====\n%s\n=====\n", app.WebserverType, site.Name, startErr, appLogs)
			}
			out, resp, err := testcommon.GetLocalHTTPResponse(t, app.GetWebContainerDirectHTTPURL()+"/servertype.php")
			require.NoError(t, err)

			expectedServerType := "Apache/2"
			if app.WebserverType == nodeps.WebserverNginxFPM {
				expectedServerType = "nginx"
			}
			require.NotEmpty(t, resp.Header["Server"])
			require.NotEmpty(t, resp.Header["Server"][0])
			assert.Contains(resp.Header["Server"][0], expectedServerType, "Server header for project=%s, app.WebserverType=%s should be %s", app.Name, app.WebserverType, expectedServerType)
			assert.Contains(out, expectedServerType, "For app.WebserverType=%s phpinfo expected servertype.php to show %s", app.WebserverType, expectedServerType)
			err = app.Stop(true, false)
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

// TestInternalAndExternalAccessToURL checks we can access content
// from host and from inside container by URL (with port)
func TestInternalAndExternalAccessToURL(t *testing.T) {
	if nodeps.IsMacM1() || dockerutil.IsColima() {
		t.Skip("Skipping on mac M1/Colima to ignore problems with 'connection reset by peer'")
	}

	assert := asrt.New(t)

	runTime := util.TimeTrack(time.Now(), t.Name())

	site := TestSites[0]
	app := new(ddevapp.DdevApp)

	err := app.Init(site.Dir)
	assert.NoError(err)

	t.Cleanup(func() {
		// Set the ports back to the default was so we don't break any following tests.
		app.RouterHTTPSPort = "443"
		app.RouterHTTPPort = "80"
		app.AdditionalFQDNs = []string{}
		app.AdditionalHostnames = []string{}

		err = app.WriteConfig()
		assert.NoError(err)
		err = app.Stop(true, false)
		assert.NoError(err)
	})

	// Add some additional hostnames
	app.AdditionalHostnames = []string{"sub1", "sub2", "sub3"}
	app.AdditionalFQDNs = []string{"junker99.example.com"}

	for _, pair := range []testcommon.PortPair{{HTTPPort: "8000", HTTPSPort: "8143"}, {HTTPPort: "8080", HTTPSPort: "8443"}} {
		testcommon.ClearDockerEnv()
		app.RouterHTTPPort = pair.HTTPPort
		app.RouterHTTPSPort = pair.HTTPSPort
		err = app.WriteConfig()
		assert.NoError(err)

		// Make sure that project is absolutely not running
		err = app.Stop(true, false)
		assert.NoError(err)

		err = app.StartAndWait(5)
		assert.NoError(err)

		expectedNumUrls := len(app.GetHostnames())*2 + 2
		httpURLs, _, urls := app.GetAllURLs()

		// If no https/mkcert, number of hostnames is different
		if globalconfig.GetCAROOT() == "" {
			urls = httpURLs
			expectedNumUrls = len(app.GetHostnames()) + 1
		}

		// Convert URLs to map[string]bool
		urlMap := make(map[string]bool)
		for _, u := range urls {
			urlMap[u] = true
		}

		assert.Equal(len(urlMap), expectedNumUrls, "Unexpected number of URLs returned: %d", len(urlMap))

		httpURLs, _, urls = app.GetAllURLs()
		if globalconfig.GetCAROOT() == "" {
			urls = httpURLs
		}
		urls = append(urls, "http://localhost", "http://localhost")
		for _, item := range urls {
			// Make sure internal (web container) access is successful
			parts, err := url.Parse(item)
			require.NoError(t, err, "url.Parse of item=%v failed", item)
			require.NotNil(t, parts, "url.Parse of item=%v failed", item)
			// Only try it if not an IP address URL; those won't be right
			hostParts := strings.Split(parts.Host, ".")

			// Make sure access from host is successful
			// But "localhost" is only for inside container.
			if parts.Host != "localhost" {
				// Dummy attempt to get webserver "warmed up" before real try.
				// Forced by M1 constant EOFs
				_, _, _ = testcommon.GetLocalHTTPResponse(t, site.Safe200URIWithExpectation.URI, 60)
				_, _ = testcommon.EnsureLocalHTTPContent(t, item+site.Safe200URIWithExpectation.URI, site.Safe200URIWithExpectation.Expect, 60)
			}

			if _, err := strconv.ParseInt(hostParts[0], 10, 64); err != nil {
				out, _, err := app.Exec(&ddevapp.ExecOpts{
					Service: "web",
					Cmd:     "curl -sS --fail " + item + site.Safe200URIWithExpectation.URI,
				})
				assert.NoError(err, "failed curl to %s: %v", item+site.Safe200URIWithExpectation.URI, err)
				assert.Contains(out, site.Safe200URIWithExpectation.Expect)
			}
		}
	}

	out, err := exec.RunHostCommand(DdevBin, "list")
	assert.NoError(err)
	t.Logf("\n=========== output of ddev list ==========\n%s\n============\n", out)
	out, err = exec.RunHostCommand("docker", "logs", "ddev-router")
	assert.NoError(err)
	t.Logf("\n=========== output of docker logs ddev-router ==========\n%s\n============\n", out)

	runTime()
}

// TestCaptureLogs checks that app.CaptureLogs() works
func TestCaptureLogs(t *testing.T) {
	assert := asrt.New(t)

	site := TestSites[0]
	runTime := util.TimeTrack(time.Now(), fmt.Sprintf("%s CaptureLogs", site.Name))

	app := ddevapp.DdevApp{}

	err := app.Init(site.Dir)
	assert.NoError(err)
	err = app.Start()
	assert.NoError(err)

	logs, err := app.CaptureLogs("web", false, "100")
	assert.NoError(err)

	assert.Contains(logs, "INFO spawned")

	err = app.Stop(true, false)
	assert.NoError(err)

	runTime()
}

// TestNFSMount tests ddev start functionality with nfs_mount_enabled: true
// This requires that the test machine must have NFS shares working
// Tests using both app-specific nfs_mount_enabled and global nfs_mount_enabled
func TestNFSMount(t *testing.T) {
	if nodeps.IsWSL2() || dockerutil.IsColima() {
		t.Skip("Skipping on WSL2/Colima")
	}
	if nodeps.MutagenEnabledDefault == true || nodeps.NoBindMountsDefault {
		t.Skip("Skipping because mutagen/nobindmounts enabled")
	}

	assert := asrt.New(t)
	app := &ddevapp.DdevApp{}

	// Make sure this leaves us in the original test directory
	testDir, _ := os.Getwd()
	//nolint: errcheck
	defer os.Chdir(testDir)

	site := TestSites[0]
	switchDir := site.Chdir()
	runTime := util.TimeTrack(time.Now(), fmt.Sprintf("%s %s", site.Name, t.Name()))

	err := app.Init(site.Dir)
	assert.NoError(err)

	defer func() {
		globalconfig.DdevGlobalConfig.NFSMountEnabledGlobal = false
		_ = globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig)
		app.NFSMountEnabled = false
		_ = app.WriteConfig()
		_ = app.Stop(true, false)
	}()

	t.Log("testing with global NFSMountEnabled")
	globalconfig.DdevGlobalConfig.NFSMountEnabledGlobal = true
	err = globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig)
	assert.NoError(err)

	// Run NewApp so that it picks up the global config, as it would in real life
	app, err = ddevapp.NewApp(site.Dir, false)
	assert.NoError(err)
	verifyNFSMount(t, app)

	t.Log("testing with app NFSMountEnabled")
	globalconfig.DdevGlobalConfig.NFSMountEnabledGlobal = false
	err = globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig)
	assert.NoError(err)

	// Run NewApp so that it picks up the global config, as it would in real life
	app, err = ddevapp.NewApp(site.Dir, false)
	assert.NoError(err)

	app.NFSMountEnabled = true
	verifyNFSMount(t, app)

	runTime()
	switchDir()
}

func verifyNFSMount(t *testing.T, app *ddevapp.DdevApp) {
	assert := asrt.New(t)

	err := app.Stop(true, false)
	assert.NoError(err)
	err = app.Start()
	//nolint: errcheck
	defer app.Stop(true, false)
	require.NoError(t, err)

	stdout, _, err := app.Exec(&ddevapp.ExecOpts{
		Service: "web",
		Dir:     "/var/www/html",
		Cmd:     "findmnt -T .",
	})
	assert.NoError(err)

	source := app.AppRoot
	if runtime.GOOS == "darwin" && fileutil.IsDirectory(filepath.Join("/System/Volumes/Data", app.AppRoot)) {
		source = filepath.Join("/System/Volumes/Data", app.AppRoot)
	}
	assert.Contains(stdout, ":"+dockerutil.MassageWindowsNFSMount(source))

	// Create a host-side dir symlink; give a second for it to sync, make sure it can be used in container.
	err = os.Symlink(".ddev", "nfslinked_.ddev")
	assert.NoError(err)
	// nolint: errcheck
	defer os.Remove("nfslinked_.ddev")

	time.Sleep(2 * time.Second)
	_, _, err = app.Exec(&ddevapp.ExecOpts{
		Service: "web",
		Dir:     "/var/www/html",
		Cmd:     "ls nfslinked_.ddev/config.yaml",
	})
	assert.NoError(err)

	// Create a host-side file symlink; give a second for it to sync, make sure it can be used in container.
	err = os.Symlink(".ddev/config.yaml", "nfslinked_config.yaml")
	assert.NoError(err)
	// nolint: errcheck
	defer os.Remove("nfslinked_config.yaml")

	time.Sleep(2 * time.Second)
	_, _, err = app.Exec(&ddevapp.ExecOpts{
		Service: "web",
		Dir:     "/var/www/html",
		Cmd:     "ls nfslinked_config.yaml",
	})
	assert.NoError(err)

	// Create a container-side dir symlink; give a second for it to sync, make sure it can be used on host.
	_, _, err = app.Exec(&ddevapp.ExecOpts{
		Service: "web",
		Dir:     "/var/www/html",
		// symlink creation on windows can seem to fail with mysterious i/o error, when it actually worked
		// so give it a second chance with the ls.
		Cmd: "ln -s  .ddev nfscontainerlinked_ddev || ls nfscontainerlinked_ddev",
	})
	assert.NoError(err)

	// nolint: errcheck
	defer os.Remove("nfscontainerlinked_ddev")

	time.Sleep(2 * time.Second)
	assert.FileExists("nfscontainerlinked_ddev/config.yaml")

	// Create a container-side file symlink; give a second for it to sync, make sure it can be used on host.
	_, _, err = app.Exec(&ddevapp.ExecOpts{
		Service: "web",
		Dir:     "/var/www/html",
		// symlink creation on windows can fail... but actually succeed. We give it a second
		// chance with the ls
		Cmd: "ln -s  .ddev/config.yaml nfscontainerlinked_config.yaml || ls nfscontainerlinked_config.yaml",
	})
	assert.NoError(err)

	// nolint: errcheck
	defer os.Remove("nfscontainerlinked_config.yaml")

	time.Sleep(2 * time.Second)
	assert.FileExists("nfscontainerlinked_config.yaml")
}

// TestHostDBPort tests to make sure that the host_db_port specification has the intended effect
func TestHostDBPort(t *testing.T) {
	if dockerutil.IsColima() {
		t.Skip("Skipping test on colima because of constant port problems")
	}
	assert := asrt.New(t)
	runTime := util.TimeTrack(time.Now(), t.Name())
	defer runTime()
	origDir, _ := os.Getwd()

	site := TestSites[0]
	err := os.Chdir(site.Dir)
	require.NoError(t, err)

	app, err := ddevapp.NewApp(site.Dir, false)
	assert.NoError(err)

	showportPath := app.GetConfigPath("commands/host/showport")
	err = os.MkdirAll(filepath.Dir(showportPath), 0755)
	assert.NoError(err)
	err = fileutil.CopyFile(filepath.Join(origDir, "testdata", t.Name(), "showport"), showportPath)
	assert.NoError(err)

	t.Cleanup(func() {
		err = os.Chdir(origDir)
		assert.NoError(err)
		err = app.Stop(true, false)
		assert.NoError(err)
		app.HostDBPort = ""
		app.Database.Type = nodeps.MariaDB
		app.Database.Version = nodeps.MariaDBDefaultVersion
		err = app.WriteConfig()
		assert.NoError(err)
		err = os.RemoveAll(showportPath)
		assert.NoError(err)
	})
	clientTool := ""

	// Make sure that everything works with and without
	// an explicitly specified hostDBPort
	for _, dbType := range []string{nodeps.MariaDB, nodeps.Postgres} {
		switch dbType {
		case nodeps.MariaDB:
			clientTool = "mysql"
			app.Database = ddevapp.DatabaseDesc{Type: dbType, Version: nodeps.MariaDBDefaultVersion}
		case nodeps.Postgres:
			clientTool = "psql"
			app.Database = ddevapp.DatabaseDesc{Type: dbType, Version: nodeps.PostgresDefaultVersion}
		}

		for _, hostDBPort := range []string{"", "9998"} {
			app.HostDBPort = hostDBPort
			err = app.Stop(true, false)
			require.NoError(t, err)
			err = app.WriteConfig()
			require.NoError(t, err)
			err = app.Start()
			require.NoError(t, err)

			desc, err := app.Describe(false)
			assert.NoError(err)

			dockerIP, err := dockerutil.GetDockerIP()
			assert.NoError(err)
			dbinfo := desc["dbinfo"].(map[string]interface{})
			dbPort := dbinfo["published_port"].(int)
			dbPortStr := strconv.Itoa(dbPort)

			if app.HostDBPort != "" {
				assert.EqualValues(app.HostDBPort, dbPortStr)
			}

			if !util.IsCommandAvailable(clientTool) {
				t.Logf("Skipping %s check because %s tool not available", clientTool, clientTool)
			} else {
				// Running mysql or psql against the container ensures that we can get there via the values
				// in ddev describe
				c := fmt.Sprintf(
					"export MYSQL_PWD=db; mysql -N --user=db --host=%s --port=%d --database=db -e 'SELECT 1;'", dockerIP, dbPort)
				if dbType == nodeps.Postgres {
					c = fmt.Sprintf("export PGPASSWORD=db; psql -U db -t --host=%s --port=%d --dbname=db -c 'SELECT 1;'", dockerIP, dbPort)
				}

				out, err := exec.RunHostCommand("bash", "-c", c)
				assert.NoError(err, "Failed to run%v: err=%v, out=%v", c, err, out)
				out = strings.ReplaceAll(out, "\r", "")
				out = strings.ReplaceAll(out, " ", "")
				lines := strings.Split(out, "\n")
				assert.Equal("1", lines[0])
			}

			// Running the test host custom command "showport" ensures that the DDEV_HOST_DB_PORT
			// is getting in there available to host custom commands.
			_, _ = exec.RunHostCommand(DdevBin)
			out, err := exec.RunHostCommand(DdevBin, "showport")
			assert.NoError(err)
			lines := strings.Split(out, "\n")
			assert.EqualValues("DDEV_HOST_DB_PORT="+dbPortStr, strings.Trim(lines[0], "\r\n"), "dbType=%s, out=%s, err=%v", dbType, out, err)
		}
	}
}

// TestPortSpecifications tests to make sure that one project can't step on the
// ports used by another
func TestPortSpecifications(t *testing.T) {
	if nodeps.IsWSL2() {
		t.Skip("Skipping on WSL2 because of inconsistent docker behavior acquiring ports")
	}
	assert := asrt.New(t)
	runTime := util.TimeTrack(time.Now(), fmt.Sprint("TestPortSpecifications"))
	defer runTime()
	testDir, _ := os.Getwd()

	site0 := TestSites[0]
	switchDir := site0.Chdir()
	defer switchDir()

	nospecApp := ddevapp.DdevApp{}
	err := nospecApp.Init(site0.Dir)
	assert.NoError(err)
	err = nospecApp.WriteConfig()
	require.NoError(t, err)
	err = globalconfig.ReadGlobalConfig()
	require.NoError(t, err)
	// Since host ports were not explicitly set in nospecApp, they shouldn't be in globalconfig.
	require.Empty(t, globalconfig.DdevGlobalConfig.ProjectList[nospecApp.Name].UsedHostPorts)

	err = nospecApp.Start()
	assert.NoError(err)

	// Now that we have a working nospecApp with unspecified ephemeral ports, test that we
	// can't use those ports while nospecApp is running

	_ = os.Chdir(testDir)
	ddevDir, _ := filepath.Abs("./testdata/TestPortSpecifications/.ddev")

	specAppPath := testcommon.CreateTmpDir(t.Name() + "_specapp")

	err = fileutil.CopyDir(ddevDir, filepath.Join(specAppPath, ".ddev"))
	require.NoError(t, err, "could not copy to specAppPath %v", specAppPath)

	specAPP, err := ddevapp.NewApp(specAppPath, false)
	assert.NoError(err)

	t.Cleanup(func() {
		_ = specAPP.Stop(true, false)
		err = os.RemoveAll(specAppPath)
		assert.NoError(err)
	})

	// It should be able to WriteConfig and Start with the configured host ports it came up with
	err = specAPP.WriteConfig()
	assert.NoError(err)
	err = specAPP.Start()
	assert.NoError(err)
	//nolint: errcheck
	err = specAPP.Stop(false, false)
	require.NoError(t, err)
	// Verify that DdevGlobalConfig got updated properly
	require.NotEmpty(t, globalconfig.DdevGlobalConfig.ProjectList[specAPP.Name])
	require.NotEmpty(t, globalconfig.DdevGlobalConfig.ProjectList[specAPP.Name].UsedHostPorts)

	// However, if we change change the name to make it appear to be a
	// different project, we should not be able to config or start
	conflictApp, err := ddevapp.NewApp(specAppPath, false)
	assert.NoError(err)
	conflictApp.Name = "conflictapp"

	t.Cleanup(func() {
		err = conflictApp.Stop(true, false)
		assert.NoError(err)
	})
	err = conflictApp.WriteConfig()
	assert.Error(err)
	err = conflictApp.Start()
	assert.Error(err, "Expected error starting conflictApp=%v", conflictApp)

	// Now delete the specAPP and we should be able to use the conflictApp
	err = specAPP.Stop(true, false)
	assert.NoError(err)
	assert.Empty(globalconfig.DdevGlobalConfig.ProjectList[specAPP.Name])

	err = conflictApp.WriteConfig()
	assert.NoError(err)
	err = conflictApp.Start()
	assert.NoError(err)

	require.NotEmpty(t, globalconfig.DdevGlobalConfig.ProjectList[conflictApp.Name])
	require.NotEmpty(t, globalconfig.DdevGlobalConfig.ProjectList[conflictApp.Name].UsedHostPorts)
}

// TestDdevGetProjects exercises GetProjects()
// It's only here for profiling at this point
func TestDdevGetProjects(t *testing.T) {
	assert := asrt.New(t)
	runTime := util.TimeTrack(time.Now(), fmt.Sprint(t.Name()))
	defer runTime()

	apps, err := ddevapp.GetProjects(false)
	assert.NoError(err)
	_ = apps

}

// TestCustomCerts makes sure that added custom certificates are respected and used
func TestCustomCerts(t *testing.T) {
	if globalconfig.GetCAROOT() == "" {
		t.Skip("can't test custom certs without https enabled, skipping")
	}
	assert := asrt.New(t)

	site := TestSites[0]
	switchDir := site.Chdir()
	defer switchDir()

	app, err := ddevapp.NewApp(site.Dir, false)
	assert.NoError(err)

	certDir := app.GetConfigPath("custom_certs")
	err = os.MkdirAll(certDir, 0755)
	assert.NoError(err)

	t.Cleanup(func() {
		_ = os.RemoveAll(certDir)
		_, _, err = app.Exec(&ddevapp.ExecOpts{
			Cmd: "rm /mnt/ddev-global-cache/custom_certs/" + app.GetHostname() + "*",
		})
		assert.NoError(err)
		err = app.Stop(true, false)
		assert.NoError(err)
	})

	// Start without cert and make sure normal DNS names are there
	err = app.Start()
	assert.NoError(err)
	stdout, _, err := app.Exec(&ddevapp.ExecOpts{
		Cmd: fmt.Sprintf("openssl s_client -connect %s:443 -servername %s </dev/null 2>/dev/null | openssl x509 -noout -text | perl -l -0777 -ne '@names=/\\bDNS:([^\\s,]+)/g; print join(\"\\n\", sort @names);'", app.GetHostname(), app.GetHostname()),
	})
	stdout = strings.Trim(stdout, "\r\n")
	// This should be our regular wildcard cert
	assert.Contains(stdout, "*.ddev.site")

	// Now stop it so we can install new custom cert.
	err = app.Stop(true, false)
	assert.NoError(err)

	// Create a certfile/key in .ddev/custom_certs with just one DNS name in it
	// mkcert --cert-file d9composer.ddev.site.crt --key-file d9composer.ddev.site.key d9composer.ddev.site
	out, err := exec.RunHostCommand("mkcert", "--cert-file", filepath.Join(certDir, app.GetHostname()+".crt"), "--key-file", filepath.Join(certDir, app.GetHostname()+".key"), app.GetHostname())
	assert.NoError(err, "mkcert command failed, out=%s", out)

	err = app.Start()
	assert.NoError(err)

	stdout, _, err = app.Exec(&ddevapp.ExecOpts{
		Cmd: fmt.Sprintf("openssl s_client -connect %s:443 -servername %s </dev/null 2>/dev/null | openssl x509 -noout -text | perl -l -0777 -ne '@names=/\\bDNS:([^\\s,]+)/g; print join(\"\\n\", sort @names);'", app.GetHostname(), app.GetHostname()),
	})
	stdout = strings.Trim(stdout, "\r\n")
	// If we had the regular cert, there would be several things here including *.ddev.site
	// But we should only see the hostname listed.
	assert.Equal(app.GetHostname(), stdout)
}

// TestEnvironmentVariables tests to make sure that documented environment variables appear
// in the web container and on the host.
func TestEnvironmentVariables(t *testing.T) {
	assert := asrt.New(t)

	origDir, _ := os.Getwd()
	pwd, _ := os.Getwd()
	customCmd := filepath.Join(pwd, "testdata", t.Name(), "showhostenvvar")
	site := TestSites[0]

	app, err := ddevapp.NewApp(site.Dir, false)
	assert.NoError(err)
	err = os.Chdir(site.Dir)
	assert.NoError(err)
	customCmdDest := app.GetConfigPath("commands/host/" + "showhostenvvar")

	err = os.MkdirAll(filepath.Dir(customCmdDest), 0755)
	require.NoError(t, err)
	err = fileutil.CopyFile(customCmd, customCmdDest)
	require.NoError(t, err)

	// This set of webContainerExpectations should be maintained to match the list in the docs
	webContainerExpectations := map[string]string{
		"DDEV_DOCROOT":           app.GetDocroot(),
		"DDEV_HOSTNAME":          app.GetHostname(),
		"DDEV_PHP_VERSION":       app.PHPVersion,
		"DDEV_PRIMARY_URL":       app.GetPrimaryURL(),
		"DDEV_PROJECT":           app.Name,
		"DDEV_PROJECT_TYPE":      app.Type,
		"DDEV_ROUTER_HTTP_PORT":  app.RouterHTTPPort,
		"DDEV_ROUTER_HTTPS_PORT": app.RouterHTTPSPort,
		"DDEV_SITENAME":          app.Name,
		"DDEV_TLD":               app.ProjectTLD,
		"DDEV_WEBSERVER_TYPE":    app.WebserverType,
	}

	err = app.Start()
	require.NoError(t, err)
	t.Cleanup(func() {
		err = os.RemoveAll(customCmdDest)
		assert.NoError(err)
		err = app.Stop(true, false)
		assert.NoError(err)
		err = os.Chdir(origDir)
		assert.NoError(err)
	})

	for k, v := range webContainerExpectations {
		envVal, _, err := app.Exec(&ddevapp.ExecOpts{
			Cmd: fmt.Sprintf("echo ${%s}", k),
		})
		assert.NoError(err)
		envVal = strings.Trim(envVal, "\r\n")
		assert.Equal(v, envVal)
	}

	dbPort, err := app.GetPublishedPort("db")
	dbPortStr := strconv.Itoa(dbPort)
	if dbPortStr == "-1" || err != nil {
		dbPortStr = ""
	}
	if app.HostDBPort != "" {
		dbPortStr = app.HostDBPort
	}

	// This set of hostExpectations should be maintained in parallel with documentation
	hostExpectations := map[string]string{
		"DDEV_APPROOT":             app.AppRoot,
		"DDEV_DOCROOT":             app.GetDocroot(),
		"DDEV_HOST_DB_PORT":        dbPortStr,
		"DDEV_HOST_HTTPS_PORT":     app.HostHTTPSPort,
		"DDEV_HOST_WEBSERVER_PORT": app.HostWebserverPort,
		"DDEV_HOSTNAME":            app.GetHostname(),
		"DDEV_PHP_VERSION":         app.PHPVersion,
		"DDEV_PRIMARY_URL":         app.GetPrimaryURL(),
		"DDEV_PROJECT":             app.Name,
		"DDEV_PROJECT_TYPE":        app.Type,
		"DDEV_ROUTER_HTTP_PORT":    app.RouterHTTPPort,
		"DDEV_ROUTER_HTTPS_PORT":   app.RouterHTTPSPort,
		"DDEV_SITENAME":            app.Name,
		"DDEV_TLD":                 app.ProjectTLD,
		"DDEV_WEBSERVER_TYPE":      app.WebserverType,
	}
	for k, v := range hostExpectations {
		envVal, err := exec.RunHostCommand(DdevBin, "showhostenvvar", k)
		assert.NoError(err, "could not run %s %s %s, result=%s", DdevBin, "showhostenvvar", k, envVal)
		envVal = strings.Trim(envVal, "\r\n")
		lines := strings.Split(envVal, "\n")
		assert.Equal(v, lines[0], "expected envvar $%s to equal '%s', but it was '%s'", k, v, envVal)
	}

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
