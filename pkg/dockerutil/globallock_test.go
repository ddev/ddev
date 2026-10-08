package dockerutil

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAcquireGlobalLock(t *testing.T) {
	unlock := AcquireGlobalLock("test")
	unlock()
	unlock() // Safe to call twice.
	require.NoFileExists(t, globalLockInfoPath())

	origTimeout, origInterval := globalLockTimeout, globalLockReportInterval
	globalLockTimeout, globalLockReportInterval = 500*time.Millisecond, 200*time.Millisecond
	t.Cleanup(func() { globalLockTimeout, globalLockReportInterval = origTimeout, origInterval })

	// While held, a second acquire waits, then proceeds unlocked after the timeout.
	held := AcquireGlobalLock("test holder")
	t.Cleanup(held)
	holder := readGlobalLockHolder()
	require.NotNil(t, holder)
	require.Equal(t, os.Getpid(), holder.PID)
	require.Contains(t, holder.String(), "which is doing test holder")
	start := time.Now()
	AcquireGlobalLock("test")()
	require.GreaterOrEqual(t, time.Since(start), 400*time.Millisecond)

	// A child of the holder doesn't wait for it.
	origParent := parentPID
	parentPID = os.Getpid
	start = time.Now()
	AcquireGlobalLock("test")()
	parentPID = origParent
	require.Less(t, time.Since(start), 200*time.Millisecond)

	held()
	require.NoFileExists(t, globalLockInfoPath())
	start = time.Now()
	AcquireGlobalLock("test")()
	require.Less(t, time.Since(start), 400*time.Millisecond)
}
