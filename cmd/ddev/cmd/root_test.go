package cmd

import (
	"fmt"
	"os"
	osexec "os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/config/types"
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/ddev/ddev/pkg/util"
	"github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	globalconfig.EnsureGlobalConfig()
}

var (
	// DdevBin is the full path to the ddev binary
	DdevBin   = "ddev"
	TestSites = []testcommon.TestSite{
		{
			SourceURL:                     "https://wordpress.org/wordpress-5.8.2.tar.gz",
			ArchiveInternalExtractionPath: "wordpress/",
			FilesTarballURL:               "https://github.com/ddev/ddev_test_tarballs/releases/download/v1.1/wordpress5.8.2_files.tar.gz",
			DBTarURL:                      "https://github.com/ddev/ddev_test_tarballs/releases/download/v1.1/wordpress5.8.2_db.sql.tar.gz",
			Docroot:                       "",
			Type:                          nodeps.AppTypeWordPress,
			Safe200URIWithExpectation:     testcommon.URIWithExpect{URI: "/readme.html", Expect: "Welcome. WordPress is a very special project to me."},
			DynamicURI:                    testcommon.URIWithExpect{URI: "/", Expect: "this post has a photo"},
			FilesImageURI:                 "/wp-content/uploads/2021/12/DSCF0436-randy-and-nancy-with-traditional-wedding-out-fit-2048x1536.jpg",
			Name:                          "TestCmdWordpress",
			HTTPProbeURI:                  "wp-admin/setup-config.php",
		},
		// Drupal6 is used here just because it's smaller and we don't actually
		// care much about CMS functionality.
		{
			Name:                          "TestCmdDrupal6",
			SourceURL:                     "https://ftp.drupal.org/files/projects/drupal-6.38.tar.gz",
			ArchiveInternalExtractionPath: "drupal-6.38/",
			DBTarURL:                      "https://github.com/ddev/ddev_test_tarballs/releases/download/v1.1/drupal6.38_db.tar.gz",
			FullSiteTarballURL:            "",
			FilesTarballURL:               "https://github.com/ddev/ddev_test_tarballs/releases/download/v1.1/drupal6_files.tar.gz",
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
	_ = os.Setenv("MUTAGEN_DATA_DIRECTORY", globalconfig.GetMutagenDataDirectory())

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
		util.Failed("addSites() failed: %v", err)
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

	// Looking for active approot here should fail, because there is none
	_, err := ddevapp.GetActiveAppRoot("")
	assert.Contains(err.Error(), "Please specify a project name or change directories")

	// There is also no project named "potato"
	_, err = ddevapp.GetActiveAppRoot("potato")
	assert.Error(err)

	// However, TestSites[0] is running, so we should find it.
	appRoot, err := ddevapp.GetActiveAppRoot(TestSites[0].Name)
	assert.NoError(err)
	assert.Equal(TestSites[0].Dir, appRoot)

	origDir, _ := os.Getwd()
	err = os.Chdir(TestSites[0].Dir)
	require.NoError(t, err)

	// We should also now be able to get it in the project directory
	// since it's running and we're in the directory
	appRoot, err = ddevapp.GetActiveAppRoot("")
	assert.NoError(err)
	assert.Equal(TestSites[0].Dir, appRoot)

	// Make sure that shell commands show in regular `ddev` output - launch should show
	b := util.FindBashPath()
	_, err = exec.RunHostCommand(b, "-c", fmt.Sprintf("%s | grep launch", DdevBin))
	assert.NoError(err)

	// And we should be able to stop it and find it as well
	app, err := ddevapp.GetActiveApp("")
	require.NoError(t, err)
	err = app.Stop(false, true)
	require.NoError(t, err)

	t.Cleanup(func() {
		err = os.Chdir(origDir)
		assert.NoError(err)
		// Leave it running in case anybody cares
		err = app.Start()
		assert.NoError(err)
	})

	appRoot, err = ddevapp.GetActiveAppRoot(app.Name)
	assert.NoError(err)
	assert.Equal(TestSites[0].Dir, appRoot)
}

// TestCreateGlobalDdevDir checks to make sure that ddev will create a ~/.ddev (and updatecheck)
func TestCreateGlobalDdevDir(t *testing.T) {
	if nodeps.PerformanceModeDefault == types.PerformanceModeMutagen ||
		(globalconfig.DdevGlobalConfig.IsMutagenEnabled() &&
			nodeps.PerformanceModeDefault != types.PerformanceModeNone) ||
		nodeps.NoBindMountsDefault {
		t.Skip("Skipping because this changes homedir and breaks mutagen functionality")
	}

	assert := asrt.New(t)

	origDir, _ := os.Getwd()
	origHomeDir, err := homedir.Dir()
	require.NoError(t, err)

	tmpHomeDir := testcommon.CreateTmpDir("globalDdevCheck")

	t.Cleanup(
		func() {
			_, err := exec.RunHostCommand(DdevBin, "poweroff")
			assert.NoError(err)
			err = os.Chdir(origDir)
			assert.NoError(err)
			err = os.RemoveAll(tmpHomeDir)
			assert.NoError(err)

			// Because the start will have done a poweroff (new version),
			// make sure sites are running again.
			for _, site := range TestSites {
				_, _ = exec.RunCommand(DdevBin, []string{"start", "-y", site.Name})
			}
		})

	err = os.Chdir(TestSites[0].Dir)
	require.NoError(t, err)

	// Change the homedir temporarily
	t.Setenv("HOME", tmpHomeDir)
	t.Setenv("USERPROFILE", tmpHomeDir)

	// Make sure that the tmpDir/.ddev and tmpDir/.ddev/.update don't exist before we run ddev.
	_, err = os.Stat(filepath.Join(tmpHomeDir, ".ddev"))
	require.Error(t, err)
	assert.True(os.IsNotExist(err))

	out, err := exec.RunHostCommand(DdevBin, "config", "--auto")
	require.NoError(t, err, "failed to ddev config --auto, out=%v, err=%v", out, err)

	// Now global .ddev should exist
	_, err = os.Stat(filepath.Join(tmpHomeDir, ".ddev"))
	require.NoError(t, err)

	// Make sure we have the .ddev/bin dir we need for docker-compose and mutagen
	err = fileutil.CopyDir(filepath.Join(origHomeDir, ".ddev/bin"), filepath.Join(tmpHomeDir, ".ddev/bin"))
	require.NoError(t, err)

	// Make sure that tmpHomeDir/.ddev/.update don't exist before we run ddev start
	tmpUpdateFilePath := filepath.Join(tmpHomeDir, ".ddev", ".update")
	_, err = os.Stat(tmpUpdateFilePath)
	require.Error(t, err)
	assert.True(os.IsNotExist(err))

	// The .update file is only created by ddev start
	out, err = exec.RunHostCommand(DdevBin, "start", "-y")
	assert.NoError(err, "failed to start, out=%v, err=%v", out, err)

	_, err = os.Stat(tmpUpdateFilePath)
	assert.NoError(err)
}

// TestPoweroffOnNewVersion checks that a poweroff happens when a new ddev version is deployed
func TestPoweroffOnNewVersion(t *testing.T) {
	if nodeps.IsWSL2() && dockerutil.IsDockerDesktop() {
		t.Skip("Fails on docker desktop because of removal of ~/.docker apparently, skipping")
	}
	assert := asrt.New(t)
	var err error

	origDir, _ := os.Getwd()

	tmpHome := testcommon.CreateTmpDir(t.Name())
	t.Cleanup(func() {
		assert.NoError(os.RemoveAll(tmpHome))
	})
	err = os.Chdir(TestSites[0].Dir)
	assert.NoError(err)

	origHome := os.Getenv("HOME")
	if runtime.GOOS == "windows" {
		origHome = os.Getenv("USERPROFILE")
	}

	// Create an extra junk project to make sure it gets shut down on our start
	junkName := t.Name() + "-tmpjunkproject"
	_, _ = exec.RunHostCommand(DdevBin, "delete", "-Oy", junkName)

	tmpJunkProjectDir := testcommon.CreateTmpDir(junkName)
	err = os.Chdir(tmpJunkProjectDir)
	assert.NoError(err)
	out, err := exec.RunHostCommand(DdevBin, "config", "--project-name", junkName)
	assert.NoError(err, "out=%s", out)
	_, err = exec.RunHostCommand(DdevBin, "start", "-y")
	assert.NoError(err)
	t.Cleanup(func() {
		_, err = exec.RunHostCommand(DdevBin, "delete", "-Oy", junkName)
		assert.NoError(err)

		err = os.Chdir(origDir)
		assert.NoError(err)
		_ = os.RemoveAll(tmpJunkProjectDir)

		// Because the start has done a poweroff (new ddev version),
		// make sure sites are running again.
		for _, site := range TestSites {
			_, _ = exec.RunCommand(DdevBin, []string{"start", "-y", site.Name})
		}
	})

	apps := ddevapp.GetActiveProjects()
	activeCount := len(apps)
	assert.GreaterOrEqual(activeCount, 2)

	// Change the homedir temporarily
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)

	// Make sure we have the .ddev/bin dir we need
	err = fileutil.CopyDir(filepath.Join(origHome, ".ddev/bin"), filepath.Join(tmpHome, ".ddev/bin"))
	require.NoError(t, err)

	// docker-compose v2 is dependent on the ~/.docker directory
	_ = fileutil.CopyDir(filepath.Join(origHome, ".docker"), filepath.Join(tmpHome, ".docker"))

	app, err := ddevapp.GetActiveApp("")
	require.NoError(t, err)
	oldTime, stderr, err := app.Exec(&ddevapp.ExecOpts{
		Service: "web",
		Cmd:     "date +%s",
	})
	require.NoError(t, err, "failed to run exec: %v, output='%s', stderr='%s'", err, oldTime, stderr)
	oldTime = strings.Trim(oldTime, "\r\n")
	oldTimeInt, err := strconv.ParseInt(oldTime, 10, 64)
	require.NoError(t, err)

	// Make sure we have starting version that is not v0.0
	err = fileutil.AppendStringToFile(filepath.Join(globalconfig.GetGlobalDdevDir(), "global_config.yaml"), "last_started_version: v0.1")
	require.NoError(t, err)
	out, err = exec.RunHostCommand(DdevBin, "start")
	require.NoError(t, err, "start failed, out='%s', err=%v", out, err)
	assert.Contains(out, "ddev-ssh-agent container has been removed")
	assert.Contains(out, "ssh-agent container is running")
	t.Cleanup(func() {
		_, err := exec.RunHostCommand(DdevBin, "poweroff")
		assert.NoError(err)

		_, err = os.Stat(globalconfig.GetMutagenPath())
		if err == nil && app.IsMutagenEnabled() {
			ddevapp.StopMutagenDaemon()
		}
	})

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
		_, _ = exec.RunHostCommand(DdevBin, "stop", "-RO", site.Name)
	}
	log.Debugln("Starting TestSites")
	origDir, _ := os.Getwd()
	defer func() {
		_ = os.Chdir(origDir)
	}()
	for _, site := range TestSites {
		err := os.Chdir(site.Dir)
		if err != nil {
			log.Fatalf("Failed to Chdir to %v", site.Dir)
		}
		out, err := exec.RunHostCommand(DdevBin, "start", "-y")
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
