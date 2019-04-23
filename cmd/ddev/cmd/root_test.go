package cmd

import (
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/globalconfig"
	"os"
	"testing"

	"github.com/drud/ddev/pkg/testcommon"
	log "github.com/sirupsen/logrus"

	"path/filepath"

	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/output"
	asrt "github.com/stretchr/testify/assert"
)

var (
	// DdevBin is the full path to the drud binary
	DdevBin      = "ddev"
	DevTestSites = []testcommon.TestSite{{
		Name:                          "TestMainCmdWordpress",
		SourceURL:                     "https://github.com/drud/wordpress/archive/v0.4.0.tar.gz",
		ArchiveInternalExtractionPath: "wordpress-0.4.0/",
		FilesTarballURL:               "https://github.com/drud/wordpress/releases/download/v0.4.0/files.tar.gz",
		DBTarURL:                      "https://github.com/drud/wordpress/releases/download/v0.4.0/db.tar.gz",
		HTTPProbeURI:                  "wp-admin/setup-config.php",
		Docroot:                       "htdocs",
		Type:                          ddevapp.AppTypeWordPress,
		Safe200URIWithExpectation:     testcommon.URIWithExpect{URI: "/readme.html", Expect: "Welcome. WordPress is a very special project to me."},
	},
	}
)

func TestMain(m *testing.M) {
	output.LogSetUp()

	// Start with no global config file so we're sure to have defaults
	configFile := globalconfig.GetGlobalConfigPath()
	if fileutil.FileExists(configFile) {
		_ = os.Remove(configFile)
	}

	if os.Getenv("DDEV_BINARY_FULLPATH") != "" {
		DdevBin = os.Getenv("DDEV_BINARY_FULLPATH")
	}
	log.Println("Running ddev with ddev=", DdevBin)

	err := os.Setenv("DRUD_NONINTERACTIVE", "true")
	if err != nil {
		log.Errorln("could not set noninteractive mode, failed to Setenv, err: ", err)
	}

	// We don't want the tests reporting to Sentry.
	_ = os.Setenv("DDEV_NO_SENTRY", "true")

	// Attempt to stop/remove all running containers before starting a test.
	// If no projects are running, this will exit silently and without error.
	if _, err = exec.RunCommand(DdevBin, []string{"stop", "--all", "--stop-ssh-agent"}); err != nil {
		log.Warnf("Failed to stop/remove all running projects: %v", err)
	}

	for i := range DevTestSites {
		err = DevTestSites[i].Prepare()
		if err != nil {
			log.Fatalf("Prepare() failed in TestMain site=%s, err=%v\n", DevTestSites[i].Name, err)
		}
	}
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
	for i := range DevTestSites {
		DevTestSites[i].Cleanup()
	}

	os.Exit(testRun)

}

func TestGetActiveAppRoot(t *testing.T) {
	assert := asrt.New(t)

	_, err := ddevapp.GetActiveAppRoot("")
	assert.Contains(err.Error(), "Please specify a project name or change directories")

	_, err = ddevapp.GetActiveAppRoot("potato")
	assert.Error(err)

	appRoot, err := ddevapp.GetActiveAppRoot(DevTestSites[0].Name)
	assert.NoError(err)
	assert.Equal(DevTestSites[0].Dir, appRoot)

	switchDir := DevTestSites[0].Chdir()

	appRoot, err = ddevapp.GetActiveAppRoot("")
	assert.NoError(err)
	assert.Equal(DevTestSites[0].Dir, appRoot)

	switchDir()
}

// TestCreateGlobalDdevDir checks to make sure that ddev will create a ~/.ddev (and updatecheck)
func TestCreateGlobalDdevDir(t *testing.T) {
	assert := asrt.New(t)

	tmpDir := testcommon.CreateTmpDir("globalDdevCheck")
	origHome := os.Getenv("HOME")

	// Make sure that the tmpDir/.ddev and tmpDir/.ddev/.update don't exist before we run ddev.
	_, err := os.Stat(filepath.Join(tmpDir, ".ddev"))
	assert.Error(err)
	assert.True(os.IsNotExist(err))

	_, err = os.Stat(filepath.Join(tmpDir, ".ddev", ".update"))
	assert.Error(err)
	assert.True(os.IsNotExist(err))

	// Change the homedir temporarily
	err = os.Setenv("HOME", tmpDir)
	assert.NoError(err)

	args := []string{"list"}
	_, err = exec.RunCommand(DdevBin, args)
	assert.NoError(err)

	_, err = os.Stat(filepath.Join(tmpDir, ".ddev", ".update"))
	assert.NoError(err)

	// Cleanup our tmp homedir
	err = os.RemoveAll(tmpDir)
	assert.NoError(err)

	err = os.Setenv("HOME", origHome)
	assert.NoError(err)
}

// addSites runs `ddev start` on the test apps
func addSites() error {
	for _, site := range DevTestSites {
		cleanup := site.Chdir()
		defer cleanup()

		// test that you get an error when you run with no args
		args := []string{"start"}
		out, err := exec.RunCommand(DdevBin, args)
		if err != nil {
			log.Fatalln("Error Output from ddev start:", out, "err:", err)
		}
	}
	return nil
}

// removeSites runs `ddev remove` on the test apps
func removeSites() {
	for _, site := range DevTestSites {
		_ = site.Chdir()

		args := []string{"stop", "-RO"}
		out, err := exec.RunCommand(DdevBin, args)
		if err != nil {
			log.Errorf("Failed to run ddev stop -RO command, err: %v, output: %s\n", err, out)
		}
	}
}
