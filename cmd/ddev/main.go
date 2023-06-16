package main

import (
	"os"

	"github.com/ddev/ddev/cmd/ddev/cmd"
	"github.com/ddev/ddev/pkg/amplitude"
	"github.com/ddev/ddev/pkg/util"
)

func main() {
	defer util.TimeTrack()()

	// Initialization is currently done before via init() func somewhere while
	// creating the ddevapp. This should be cleaned up.
	amplitude.InitAmplitude()
	defer func() {
		amplitude.Flush()
		amplitude.CheckSetUp()
	}()

	// Prevent running as root for most cases
	// We really don't want ~/.ddev to have root ownership, breaks things.
	if os.Geteuid() == 0 && len(os.Args) > 1 && os.Args[1] != "hostname" {
		util.Failed("ddev is not designed to be run with root privileges, please run as normal user and without sudo")
	}

	cmd.Execute()
}
