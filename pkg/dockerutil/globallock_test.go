package dockerutil

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAcquireGlobalLock(t *testing.T) {
	unlock := AcquireGlobalLock("test")
	unlock()
	unlock() // Safe to call twice.

	origTimeout := globalLockTimeout
	globalLockTimeout = 500 * time.Millisecond
	t.Cleanup(func() { globalLockTimeout = origTimeout })

	// While held, a second acquire waits, then proceeds unlocked after the timeout.
	held := AcquireGlobalLock("test")
	defer held()
	start := time.Now()
	AcquireGlobalLock("test")()
	require.GreaterOrEqual(t, time.Since(start), 400*time.Millisecond)

	held()
	start = time.Now()
	AcquireGlobalLock("test")()
	require.Less(t, time.Since(start), 400*time.Millisecond)
}
