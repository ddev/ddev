package cmd

import (
	"fmt"
	"os"
	"testing"

	log "github.com/Sirupsen/logrus"
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/testcommon"
	"github.com/drud/ddev/pkg/util"

	"github.com/drud/ddev/pkg/appports"
	"github.com/drud/ddev/pkg/exec"
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

	_, err := getActiveAppRoot("")
	assert.Contains(err.Error(), "unable to determine the application for this command")

	_, err = getActiveAppRoot("potato")
	assert.Error(err)

	appRoot, err := getActiveAppRoot(DevTestSites[0].Name)
	assert.NoError(err)
	assert.Equal(DevTestSites[0].Dir, appRoot)

	switchDir := DevTestSites[0].Chdir()

	appRoot, err = getActiveAppRoot("")
	assert.NoError(err)
	assert.Equal(DevTestSites[0].Dir, appRoot)

	switchDir()
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

		app, err := getActiveApp("")
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

		args := []string{"remove", "--remove-data"}
		out, err := exec.RunCommand(DdevBin, args)
		if err != nil {
			log.Fatalln("Failed to run ddev remove -y command, err: %v, output: %s", err, out)
		}

		allVolumes, err := dockerutil.GetVolumes()
		if err != nil {
			log.Fatalf("Could not ensure volumes are empty for site %s during teardown", site.Name)
		}

		removedNames := []string{
			fmt.Sprintf("ddev%s_mysql", site.Name),
			fmt.Sprintf("ddev%s_nginx-logs", site.Name),
		}

		for _, remainingVolume := range allVolumes {
			for _, removedName := range removedNames {
				if removedName == remainingVolume.Name {
					log.Fatalf("Volume %s still remaining after site removal", remainingVolume.Name)
				}
			}
		}
	}
}
