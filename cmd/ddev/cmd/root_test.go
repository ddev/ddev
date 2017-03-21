package cmd

import (
	"fmt"
	"os"
	"testing"

	"github.com/drud/ddev/pkg/testcommon"
)

var (
	// DdevBin is the full path to the drud binary
	DdevBin      = "ddev"
	DevTestSites = []testcommon.TestSite{
		{
			Name: "drupal8",
			URL:  "https://github.com/drud/drupal8/archive/v0.2.1.tar.gz",
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
		DevTestSites[i].Prepare()
	}

	fmt.Println("Running tests.")
	testRun := m.Run()

	for i := range DevTestSites {
		DevTestSites[i].Cleanup()
	}

	os.Exit(testRun)

}
