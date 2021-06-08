package cmd

import (
	"fmt"
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/drud/ddev/pkg/nodeps"
	"github.com/drud/ddev/pkg/util"
	"github.com/stretchr/testify/require"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/drud/ddev/pkg/testcommon"
	log "github.com/sirupsen/logrus"

	"path/filepath"

	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/output"
	asrt "github.com/stretchr/testify/assert"
	osexec "os/exec"
)

var (
	// DdevBin is the full path to the drud binary
	DdevBin   = "ddev"
	TestSites = []testcommon.TestSite{
		{
			Name:                          "TestCmdWordpress",
			SourceURL:                     "https://github.com/drud/wordpress/archive/v0.4.0.tar.gz",
			ArchiveInternalExtractionPath: "wordpress-0.4.0/",
			FilesTarballURL:               "https://github.com/drud/wordpress/releases/download/v0.4.0/files.tar.gz",
			DBTarURL:                      "https://github.com/drud/wordpress/releases/download/v0.4.0/db.tar.gz",
			HTTPProbeURI:                  "wp-admin/setup-config.php",
			Docroot:                       "htdocs",
			Type:                          nodeps.AppTypeWordPress,
			Safe200URIWithExpectation:     testcommon.URIWithExpect{URI: "/readme.html", Expect: "Welcome. WordPress is a very special project to me."},
		},
		// Drupal6 is used here just because it's smaller and we don't actually
		// care much about CMS functionality.
		{
			Name:                          "TestCmdDrupal6",
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
	}
)

func TestMain(m *testing.M) {
	output.LogSetUp()

	if os.Getenv("DDEV_BINARY_FULLPATH") != "" {
		DdevBin = os.Getenv("DDEV_BINARY_FULLPATH")
	}
	log.Println("Running ddev with ddev=", DdevBin)

	err := os.Setenv("DDEV_NONINTERACTIVE", "true")
	if err != nil {
		log.Errorln("could not set noninteractive mode, failed to Setenv, err: ", err)
	}

	// We don't want the tests reporting to Segment.
	_ = os.Setenv("DDEV_NO_INSTRUMENTATION", "true")

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

	log.Debugln("Preparing TestSites")
	for i := range TestSites {
		oldProject := globalconfig.GetProject(TestSites[i].Name)
		if oldProject != nil {
			out, err := osexec.Command(DdevBin, "stop", "-RO", TestSites[i].Name).CombinedOutput()
			if err != nil {
				log.Fatalf("ddev stop -RO on %s failed: %v, output=%s", TestSites[i].Name, err, out)
			}
		}
		if err = globalconfig.ReadGlobalConfig(); err != nil {
			log.Fatalf("Failed to read global config: %v", err)
		}

		log.Debugf("Preparing %s", TestSites[i].Name)
		err = TestSites[i].Prepare()
		if err != nil {
			log.Fatalf("Prepare() failed in TestMain site=%s, err=%v\n", TestSites[i].Name, err)
		}
	}
	log.Debugln("Adding TestSites")
	err = addSites()
	if err != nil {
		removeSites()
		output.UserOut.Fatalf("addSites() failed: %v", err)
	}

	log.Debugln("Running tests.")
	testRun := m.Run()

	removeSites()

	// Avoid being in any of the directories we're cleaning up.
	_ = os.Chdir(os.TempDir())
	for i := range TestSites {
		TestSites[i].Cleanup()
	}

	os.Exit(testRun)

}

func TestGetActiveAppRoot(t *testing.T) {
	assert := asrt.New(t)

	_, err := ddevapp.GetActiveAppRoot("")
	assert.Contains(err.Error(), "Please specify a project name or change directories")

	_, err = ddevapp.GetActiveAppRoot("potato")
	assert.Error(err)

	appRoot, err := ddevapp.GetActiveAppRoot(TestSites[0].Name)
	assert.NoError(err)
	assert.Equal(TestSites[0].Dir, appRoot)

	switchDir := TestSites[0].Chdir()

	appRoot, err = ddevapp.GetActiveAppRoot("")
	assert.NoError(err)
	assert.Equal(TestSites[0].Dir, appRoot)

	switchDir()
}

// TestCreateGlobalDdevDir checks to make sure that ddev will create a ~/.ddev (and updatecheck)
func TestCreateGlobalDdevDir(t *testing.T) {
	assert := asrt.New(t)

	origDir, _ := os.Getwd()
	tmpDir := testcommon.CreateTmpDir("globalDdevCheck")
	_ = TestSites[0].Chdir()

	origHome := os.Getenv("HOME")

	t.Cleanup(
		func() {
			err := os.Chdir(origDir)
			assert.NoError(err)
			err = os.RemoveAll(tmpDir)
			assert.NoError(err)

			err = os.Setenv("HOME", origHome)
			assert.NoError(err)

			// Because the start will have done a poweroff (new version),
			// make sure sites are running again.
			for _, site := range TestSites {
				_, _ = exec.RunCommand(DdevBin, []string{"start", "-y", site.Name})
			}
		})

	// Make sure that the tmpDir/.ddev and tmpDir/.ddev/.update don't exist before we run ddev.
	_, err := os.Stat(filepath.Join(tmpDir, ".ddev"))
	assert.Error(err)
	assert.True(os.IsNotExist(err))

	tmpUpdateFilePath := filepath.Join(tmpDir, ".ddev", ".update")
	_, err = os.Stat(tmpUpdateFilePath)
	assert.Error(err)
	assert.True(os.IsNotExist(err))

	// Change the homedir temporarily
	err = os.Setenv("HOME", tmpDir)
	assert.NoError(err)

	// The .update file is only created by ddev start
	// We have to do the echo technique to get past the prompt about doing a ddev poweroff
	_, err = exec.RunCommand("bash", []string{"-c", fmt.Sprintf("echo y | %s start", DdevBin)})
	assert.NoError(err)

	_, err = os.Stat(tmpUpdateFilePath)
	assert.NoError(err)

}

// TestPoweroffOnNewVersion checks that a poweroff happens when a new ddev version is deployed
func TestPoweroffOnNewVersion(t *testing.T) {
	assert := asrt.New(t)
	var err error

	origDir, _ := os.Getwd()

	tmpGlobal := testcommon.CreateTmpDir(t.Name())
	err = os.Chdir(TestSites[0].Dir)
	assert.NoError(err)

	origHome := os.Getenv("HOME")

	// Create an extra junk project to make sure it gets shut down on our start
	junkName := t.Name() + "-tmpjunkproject"
	tmpJunkProject := testcommon.CreateTmpDir(junkName)
	err = os.Chdir(tmpJunkProject)
	assert.NoError(err)
	_, err = exec.RunCommand(DdevBin, []string{"config", "--auto"})
	assert.NoError(err)
	_, _ = exec.RunCommand(DdevBin, []string{"start", "-y"})
	assert.NoError(err)

	apps := ddevapp.GetActiveProjects()
	activeCount := len(apps)
	assert.GreaterOrEqual(activeCount, 2)

	// Change the homedir temporarily
	err = os.Setenv("HOME", tmpGlobal)
	assert.NoError(err)

	t.Cleanup(
		func() {

			err = os.RemoveAll(tmpGlobal)
			assert.NoError(err)

			err = os.Setenv("HOME", origHome)
			assert.NoError(err)

			err = os.Chdir(tmpJunkProject)
			assert.NoError(err)
			_, _ = exec.RunCommand(DdevBin, []string{"delete", "-Oy"})

			err = os.Chdir(origDir)
			assert.NoError(err)
			err = os.RemoveAll(tmpJunkProject)
			assert.NoError(err)

			// Because the start has done a poweroff (new ddev version),
			// make sure sites are running again.
			for _, site := range TestSites {
				_, _ = exec.RunCommand(DdevBin, []string{"start", "-y", site.Name})
			}
		})

	app, err := ddevapp.GetActiveApp("")
	require.NoError(t, err)
	oldTime, _, err := app.Exec(&ddevapp.ExecOpts{
		Service: "web",
		Cmd:     "date +%s",
	})
	oldTime = strings.Trim(oldTime, "\n")
	oldTimeInt, err := strconv.ParseInt(oldTime, 10, 64)
	require.NoError(t, err)

	bashPath := util.FindWindowsBashPath()
	out, err := exec.RunCommand(bashPath, []string{"-c", fmt.Sprintf("echo y | '%s' start", DdevBin)})
	assert.NoError(err)
	assert.Contains(out, "ddev-ssh-agent container has been removed")
	assert.Contains(out, "ssh-agent container is running")

	apps = ddevapp.GetActiveProjects()
	activeCount = len(apps)
	assert.Equal(activeCount, 1)

	// Verify that the ddev-router and ddev-ssh-agent have just been newly created
	routerC, err := dockerutil.FindContainerByName("ddev-router")
	require.NoError(t, err)
	require.NotEmpty(t, routerC)
	routerCreateTime := (*routerC).Created
	sshC, err := dockerutil.FindContainerByName("ddev-ssh-agent")
	require.NoError(t, err)
	require.NotEmpty(t, sshC)
	sshCreateTime := (*sshC).Created

	// router and ssh-agent should have been created within after we started the new project
	assert.GreaterOrEqual(routerCreateTime, oldTimeInt)
	assert.GreaterOrEqual(sshCreateTime, oldTimeInt)
}

// addSites runs `ddev start` on the test apps
func addSites() error {
	log.Debugln("Removing any existing TestSites")
	for _, site := range TestSites {
		// Make sure the site is gone in case it was hanging around
		_, _ = exec.RunCommand(DdevBin, []string{"stop", "-RO", site.Name})
	}
	log.Debugln("Starting TestSites")
	for _, site := range TestSites {
		cleanup := site.Chdir()
		defer cleanup()

		out, err := exec.RunCommand(DdevBin, []string{"start", "-y"})
		if err != nil {
			log.Fatalln("Error Output from ddev start:", out, "err:", err)
		}
	}
	return nil
}

// removeSites runs `ddev remove` on the test apps
func removeSites() {
	for _, site := range TestSites {
		_ = site.Chdir()

		args := []string{"stop", "-RO"}
		out, err := exec.RunCommand(DdevBin, args)
		if err != nil {
			log.Errorf("Failed to run ddev remove -RO command, err: %v, output: %s\n", err, out)
		}
	}
}
