package interrupt

import (
	"os"
	"syscall"
	"time"
)

// Go never delivers SIGHUP on Windows. It reports a closed console window,
// logoff, or shutdown as SIGTERM, and Windows ends the process about five
// seconds later, so that cleanup has to finish sooner.
var handledSignals = []os.Signal{os.Interrupt, syscall.SIGTERM}

func cleanupTimeout(sig os.Signal) time.Duration {
	if sig == syscall.SIGTERM {
		return 3 * time.Second
	}
	return 5 * time.Second
}

func exit(sig os.Signal) {
	os.Exit(ExitCode(sig))
}
