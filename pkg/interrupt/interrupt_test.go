package interrupt

import (
	"context"
	"os"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRunStopsAtTimeout(t *testing.T) {
	var finished atomic.Bool
	start := time.Now()
	Run([]func(context.Context){
		func(context.Context) { time.Sleep(time.Minute) },
		func(context.Context) { finished.Store(true) },
	}, 200*time.Millisecond)
	require.Less(t, time.Since(start), 5*time.Second)
	require.True(t, finished.Load())
}

// A delegated signal runs the cleanup and returns instead of exiting, which
// would end the test binary.
func TestDelegatedSignalRunsCleanup(t *testing.T) {
	var calls atomic.Int32
	unregister := OnInterrupt(func(context.Context) { calls.Add(1) })
	release := Delegate(syscall.SIGTERM)
	t.Cleanup(release)

	handle(syscall.SIGTERM, make(chan os.Signal))
	handle(syscall.SIGTERM, make(chan os.Signal))
	require.Equal(t, int32(2), calls.Load(), "cleanup runs on every delegated signal")

	unregister()
	handle(syscall.SIGTERM, make(chan os.Signal))
	require.Equal(t, int32(2), calls.Load(), "unregistered cleanup does not run")
}

func TestExitCode(t *testing.T) {
	require.Equal(t, 130, ExitCode(syscall.SIGINT))
	require.Equal(t, 143, ExitCode(syscall.SIGTERM))
}
