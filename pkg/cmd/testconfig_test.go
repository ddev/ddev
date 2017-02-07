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

	// DevTestEnv is the name of the Dev DRUD environment to test
	DevTestEnv = "production"

	// DevTestApp is the name of the Dev DRUD app to test
	DevTestApp = "drudio"

	DevTestSites = [][]string{
		[]string{"drudio", DevTestEnv},
		[]string{"d8", DevTestEnv},
		[]string{"talentreef", DevTestEnv},
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

	args := []string{"auth", "github"}
	out, err := utils.RunCommand(DrudBin, args)
	if err != nil {
		fmt.Println(	"Failed to run command 'drud auth github' output=", out)
		os.Exit(1)
	}

	fmt.Println("Running tests.")
	os.Exit(m.Run())
}

func setActiveApp(appName string, deployName string) error {
	if appName == "" && deployName == "" {
		_, err := utils.RunCommand(DrudBin, []string{"config", "unset", "--activeapp", "--activedeploy"})
		return err
	}

	_, err := utils.RunCommand(DrudBin, []string{"config", "set", "--activeapp", appName, "--activedeploy", deployName})
	return err
}
