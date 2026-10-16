//go:build !windows

package interrupt

import (
	"os"
	"os/signal"
	"syscall"
	"time"
)

var handledSignals = []os.Signal{syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP}

func cleanupTimeout(_ os.Signal) time.Duration {
	return 5 * time.Second
}

// exit dies from sig rather than exiting with a code, because a shell
// running ddev in a loop only stops on Ctrl-C if its child was killed by it.
func exit(sig os.Signal) {
	if s, ok := sig.(syscall.Signal); ok {
		signal.Reset(s)
		_ = syscall.Kill(os.Getpid(), s)
		time.Sleep(time.Second)
	}
	os.Exit(ExitCode(sig))
}
