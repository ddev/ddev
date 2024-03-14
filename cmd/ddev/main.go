package main

import (
	"bytes"
	"github.com/ddev/ddev/cmd/ddev/cmd"
	"github.com/ddev/ddev/pkg/amplitude"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/util"
	"os"
	"runtime"
	"runtime/pprof"
)

func main() {

	defer func() {
		globalconfig.GoroutineCount = runtime.NumGoroutine()
		if globalconfig.DdevDebug {

			if globalconfig.DdevVerbose {
				buf := new(bytes.Buffer)

				// Lookup "goroutine" profile
				p := pprof.Lookup("goroutine")
				// Write it to stderr
				_ = p.WriteTo(buf, 2)
				util.Verbose(buf.String())
			}
		}
		util.Debug("goroutines=%d at exit of main()", globalconfig.GoroutineCount)
	}()
	defer util.TimeTrack()()

	// Initialization is currently done before via init() func somewhere while
	// creating the ddevapp. This should be cleaned up.
	amplitude.InitAmplitude()
	defer func() {
		amplitude.Flush()
	}()

	// Prevent running as root for most cases
	// We really don't want ~/.ddev to have root ownership, breaks things.
	if os.Geteuid() == 0 && len(os.Args) > 1 && os.Args[1] != "hostname" {
		util.Failed("DDEV is not designed to be run with root privileges, please run as normal user and without sudo")
	}

	cmd.Execute()

}
