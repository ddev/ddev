package main

import (
	"github.com/ddev/ddev/cmd/ddev/cmd"
	"github.com/ddev/ddev/pkg/amplitude"
	"github.com/ddev/ddev/pkg/util"
	"os"
)

func main() {
	defer util.CheckGoroutines()
	defer util.TimeTrack()()

	// Initialization is currently done before via init() func somewhere while
	// creating the ddevapp. This should be cleaned up.
	amplitude.InitAmplitude()
	defer func() {
		amplitude.Flush()
	}()

	// Prevent running as root
	// We really don't want ~/.ddev to have root ownership, breaks things.
	if os.Geteuid() == 0 {
		util.Failed("DDEV is not designed to be run with root privileges, please run as normal user and without sudo")
	}

	cmd.Execute()
}
