package cmd

import (
	"fmt"
	"os"
	"testing"

	"github.com/drud/drud-go/utils"
)

var (
	// DrudBin is the name of the DRUD binary
	DrudBin = "drud"
	// LegacyTestApp is the name of the legacy DRUD app to test
	LegacyTestApp = "drudio"
	// LegacyTestEnv is the name of the legacy DRUD environment to test
	LegacyTestEnv = "production"
)

func TestMain(m *testing.M) {

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
