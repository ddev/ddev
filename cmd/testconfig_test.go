package cmd

import (
	"fmt"
	"os"
	"testing"

	"github.com/drud/drud-go/utils"
)

var (
	// DrudBin is the full path to the drud binary
	DrudBin = "drud"

	// LegacyTestEnv is the name of the legacy DRUD environment to test
	LegacyTestEnv = "production"

	// LegacyTestApp is the name of the legacy DRUD app to testes
	LegacyTestApp = "drudio"

	LegacyTestSites = [][]string{
		[]string{"drudio", LegacyTestEnv},
		[]string{"d8", LegacyTestEnv},
		[]string{"talentreef", LegacyTestEnv},
	}
)

func TestMain(m *testing.M) {
	if os.Getenv("DRUD_BINARY_FULLPATH") != "" {
		DrudBin = os.Getenv("DRUD_BINARY_FULLPATH")
	}

	err := os.Setenv("DRUD_NONINTERACTIVE", "true")
	if err != nil {
		fmt.Println("could not set noninteractive mode")
	}
	fmt.Println("Running tests")
	os.Exit(m.Run())
}

func setActiveApp(appName string, deployName string) error {
	if appName == "" && deployName == "" {
		_, err := utils.RunCommand(DrudBin, []string{"config", "unset", "active_app", "active_deploy"})
		return err
	}

	_, err := utils.RunCommand(DrudBin, []string{"config", "set", "-a", appName, "-d", deployName})
	return err
}
