package cmd

import (
	"fmt"
	"os"
	"testing"

	log "github.com/Sirupsen/logrus"
	"github.com/drud/ddev/pkg/testcommon"
	"github.com/drud/ddev/pkg/util"

	"path/filepath"

	"github.com/drud/ddev/pkg/appports"
	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/plugins/platform"
	"github.com/stretchr/testify/assert"
)

var (
	// DdevBin is the full path to the drud binary
	DdevBin      = "ddev"
	DevTestSites = []testcommon.TestSite{
		{
			Name:                          "TestMainCmdDrupal8",
			SourceURL:                     "https://github.com/drud/drupal8/archive/v0.6.0.tar.gz",
			ArchiveInternalExtractionPath: "drupal8-0.6.0/",
			FilesTarballURL:               "https://github.com/drud/drupal8/releases/download/v0.6.0/files.tar.gz",
			DBTarURL:                      "https://github.com/drud/drupal8/releases/download/v0.6.0/db.tar.gz",
		},
	}
)

func TestMain(m *testing.M) {
	if os.Getenv("DDEV_BINARY_FULLPATH") != "" {
		DdevBin = os.Getenv("DDEV_BINARY_FULLPATH")
	}
	fmt.Println("Running ddev with ddev=", DdevBin)

	err := os.Setenv("DRUD_NONINTERACTIVE", "true")
	if err != nil {
		fmt.Println("could not set noninteractive mode")
	}

	for i := range DevTestSites {
		err = DevTestSites[i].Prepare()
		if err != nil {
			log.Fatalln("Prepare() failed in TestMain site=%s, err=", DevTestSites[i].Name, err)
		}
	}
	addSites()

	fmt.Println("Running tests.")
	testRun := m.Run()

	removeSites()
	for i := range DevTestSites {
		DevTestSites[i].Cleanup()
	}

	os.Exit(testRun)

}

func TestGetActiveAppRoot(t *testing.T) {
	assert := assert.New(t)

	_, err := platform.GetActiveAppRoot("")
	assert.Contains(err.Error(), "unable to determine the application for this command")

	_, err = platform.GetActiveAppRoot("potato")
	assert.Error(err)

	appRoot, err := platform.GetActiveAppRoot(DevTestSites[0].Name)
	assert.NoError(err)
	assert.Equal(DevTestSites[0].Dir, appRoot)

	switchDir := DevTestSites[0].Chdir()

	appRoot, err = platform.GetActiveAppRoot("")
	assert.NoError(err)
	assert.Equal(DevTestSites[0].Dir, appRoot)

	switchDir()
}

// TestCreateGlobalDdevDir checks to make sure that ddev will create a ~/.ddev (and updatecheck)
func TestCreateGlobalDdevDir(t *testing.T) {
	tmpDir := testcommon.CreateTmpDir("globalDdevCheck")
	origHome := os.Getenv("HOME")

	// Make sure that the tmpDir/.ddev and tmpDir/.ddev/.update don't exist before we run ddev.
	_, err := os.Stat(filepath.Join(tmpDir, ".ddev"))
	assert.Error(t, err)
	assert.True(t, os.IsNotExist(err))

	_, err = os.Stat(filepath.Join(tmpDir, ".ddev", ".update"))
	assert.Error(t, err)
	assert.True(t, os.IsNotExist(err))

	// Change the homedir temporarily
	err = os.Setenv("HOME", tmpDir)
	assert.NoError(t, err)

	args := []string{"list"}
	_, err = exec.RunCommand(DdevBin, args)
	assert.NoError(t, err)

	_, err = os.Stat(filepath.Join(tmpDir, ".ddev", ".update"))
	assert.NoError(t, err)

	// Cleanup our tmp homedir
	err = os.RemoveAll(tmpDir)
	assert.NoError(t, err)

	err = os.Setenv("HOME", origHome)
	assert.NoError(t, err)
}

// addSites runs `ddev start` on the test apps
func addSites() {
	for _, site := range DevTestSites {
		cleanup := site.Chdir()

		// test that you get an error when you run with no args
		args := []string{"start"}
		out, err := exec.RunCommand(DdevBin, args)
		if err != nil {
			log.Fatalln("Error Output from ddev start:", out, "err:", err)
		}

		app, err := platform.GetActiveApp("")
		if err != nil {
			log.Fatalln("Could not find an active ddev configuration:", err)
		}

		urls := []string{
			"http://127.0.0.1/core/install.php",
			"http://127.0.0.1:" + appports.GetPort("mailhog"),
			"http://127.0.0.1:" + appports.GetPort("dba"),
		}

		for _, url := range urls {
			o := util.NewHTTPOptions(url)
			o.ExpectedStatus = 200
			o.Timeout = 180
			o.Headers["Host"] = app.HostName()
			err = util.EnsureHTTPStatus(o)
			if err != nil {
				log.Fatalln("Failed to ensureHTTPStatus on", app.HostName(), url)
			}
		}

		cleanup()
	}
}

// removeSites runs `ddev remove` on the test apps
func removeSites() {
	for _, site := range DevTestSites {
		_ = site.Chdir()

		args := []string{"remove", "-R"}
		out, err := exec.RunCommand(DdevBin, args)
		if err != nil {
			log.Fatalln("Failed to run ddev remove -R command, err: %v, output: %s", err, out)
		}
	}
}
