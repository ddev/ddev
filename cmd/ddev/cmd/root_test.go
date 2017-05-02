package cmd

import (
	"fmt"
	"os"
	"testing"

	log "github.com/Sirupsen/logrus"
	"github.com/drud/ddev/pkg/testcommon"

	"github.com/drud/ddev/pkg/appports"
	"github.com/drud/drud-go/utils/network"
	"github.com/drud/drud-go/utils/system"
)

var (
	// DdevBin is the full path to the drud binary
	DdevBin      = "ddev"
	DevTestSites = []testcommon.TestSite{
		{
			Name:      "drupal8",
			SourceURL: "https://github.com/drud/drupal8/archive/v0.5.0.tar.gz",
			FileURL:   "https://github.com/drud/drupal8/releases/download/v0.5.0/files.tar.gz",
			DBURL:     "https://github.com/drud/drupal8/releases/download/v0.5.0/db.tar.gz",
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

// addSites tests a `drud Dev add`
func addSites() {
	for _, site := range DevTestSites {
		cleanup := site.Chdir()

		// test that you get an error when you run with no args
		args := []string{"start"}
		out, err := system.RunCommand(DdevBin, args)
		if err != nil {
			log.Fatalln("Error Output from ddev start:", out, "err:", err)
		}

		app, err := getActiveApp()
		if err != nil {
			log.Fatalln("Could not find an active ddev configuration:", err)
		}

		urls := []string{
			"http://127.0.0.1/core/install.php",
			"http://127.0.0.1:" + appports.GetPort("mailhog"),
			"http://127.0.0.1:" + appports.GetPort("dba"),
		}

		for _, url := range urls {
			o := network.NewHTTPOptions(url)
			o.ExpectedStatus = 200
			o.Timeout = 180
			o.Headers["Host"] = app.HostName()
			err = network.EnsureHTTPStatus(o)
			if err != nil {
				log.Fatalln("Failed to ensureHTTPStatus on", app.HostName(), url)
			}
		}

		cleanup()
	}
}

// removeSites runs `drud legacy rm` on the test apps
func removeSites() {
	for _, site := range DevTestSites {
		_ = site.Chdir()

		args := []string{"rm"}
		out, err := system.RunCommand(DdevBin, args)
		if err != nil {
			log.Fatalln("Failed to runCommand ddev", args, "err:", err, "output:", out)
		}

		_, err = getActiveApp()
		if err != nil {
			log.Println("Could not find an active ddev configuration:", err)
		}
	}
}
