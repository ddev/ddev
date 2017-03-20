package cmd

import (
	"fmt"
	"os"
	"testing"

	"github.com/drud/ddev/pkg/testcommon"
)

var (
	// DdevBin is the full path to the drud binary
	DdevBin = "ddev"

	// DevTestEnv is the name of the Dev DRUD environment to test
	DevTestEnv = "production"

	// DevTestApp is the name of the Dev DRUD app to test
	DevTestApp = "drud-d8"

	DevTestSites = []testcommon.TestSite{
		// The third parameter (TmpDir) is purposefully left empty to hold the tmpDir, once created.
		{
			Name: "drupal8",
			URL:  "https://github.com/drud/drupal8/archive/v0.2.1.tar.gz",
			Path: "",
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
		testcommon.PrepareTest(&DevTestSites[i])
	}

	fmt.Println("Running tests.")
	testRun := m.Run()

	for _, v := range DevTestSites {
		testcommon.CleanupTest(v)
	}

	os.Exit(testRun)

}
