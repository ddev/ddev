package cmd

import (
	"fmt"
	"os"
	"testing"

	"github.com/drud/drud-go/utils/system"
)

var (
	// DdevBin is the full path to the drud binary
	DdevBin = "ddev"

	// DevTestEnv is the name of the Dev DRUD environment to test
	DevTestEnv = "production"

	// DevTestApp is the name of the Dev DRUD app to test
	DevTestApp = "drud-d8"

	DevTestSites = [][]string{
		[]string{"drudio", DevTestEnv},
		[]string{"drud-d8", DevTestEnv},
		[]string{"talentreef", DevTestEnv},
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

	fmt.Println("Running tests.")
	os.Exit(m.Run())
}

func setActiveApp(appName string, deployName string) error {
	if appName == "" && deployName == "" {
		_, err := system.RunCommand(DdevBin, []string{"config", "unset", "--activeapp", "--activedeploy"})
		return err
	}

	_, err := system.RunCommand(DdevBin, []string{"config", "set", "--activeapp", appName, "--activedeploy", deployName})
	return err
}
