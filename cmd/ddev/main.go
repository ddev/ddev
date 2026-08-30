package main

import (
	"os"

	"github.com/ddev/ddev/cmd/ddev/cmd"
	"github.com/ddev/ddev/pkg/amplitude"
	"github.com/ddev/ddev/pkg/util"
)

func main() {
	defer util.CheckGoroutines()
	defer util.TimeTrack()()

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
