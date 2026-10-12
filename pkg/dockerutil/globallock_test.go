package dockerutil

import (
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/ddev/ddev/pkg/util"
	"github.com/stretchr/testify/require"
)

// A copy of the test binary started with this variable set holds the global
// lock as a separate process. It runs in init() so it skips TestMain's
// container setup.
func init() {
	if os.Getenv("DDEV_TEST_GLOBAL_LOCK_HOLDER") == "" {
		return
	}
	AcquireGlobalLock("test child holder")
	time.Sleep(time.Minute)
	os.Exit(0)
}

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
	t.Cleanup(func() { setGaveUpOn(0) })
	start := time.Now()
	AcquireGlobalLock("test")()
	require.GreaterOrEqual(t, time.Since(start), 400*time.Millisecond)

	// Having given up on that holder, later acquires don't wait for it again.
	start = time.Now()
	AcquireGlobalLock("test")()
	require.Less(t, time.Since(start), 200*time.Millisecond)
	setGaveUpOn(0)

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

func TestAcquireGlobalLockCrossProcess(t *testing.T) {
	child := exec.Command(os.Args[0], "-test.run=^$")
	child.Env = append(os.Environ(), "DDEV_TEST_GLOBAL_LOCK_HOLDER=true")
	require.NoError(t, child.Start())
	t.Cleanup(func() {
		_ = child.Process.Kill()
		_ = child.Wait()
	})
	require.Eventually(t, func() bool {
		h := readGlobalLockHolder()
		return h != nil && h.PID == child.Process.Pid
	}, 30*time.Second, 50*time.Millisecond, "child never took the global lock")

	origTimeout, origInterval := globalLockTimeout, globalLockReportInterval
	globalLockTimeout, globalLockReportInterval = 500*time.Millisecond, 200*time.Millisecond
	t.Cleanup(func() { globalLockTimeout, globalLockReportInterval = origTimeout, origInterval })
	t.Cleanup(func() { setGaveUpOn(0) })

	getOut, getErr := util.CaptureUserOut(), util.CaptureUserErr()
	start := time.Now()
	AcquireGlobalLock("test")()
	waited := time.Since(start)
	start = time.Now()
	AcquireGlobalLock("test")()
	second := time.Since(start)
	out, errOut := getOut(), getErr()

	require.GreaterOrEqual(t, waited, 400*time.Millisecond)
	require.Contains(t, out, fmt.Sprintf("(pid %d,", child.Process.Pid))
	require.Contains(t, out, "which is doing test child holder")
	require.Contains(t, errOut, "Gave up after")
	require.Less(t, second, 200*time.Millisecond, "waited again on a holder it already gave up on")

	// The OS releases the lock when the holder dies, even without unlock.
	require.NoError(t, child.Process.Kill())
	_ = child.Wait()
	start = time.Now()
	AcquireGlobalLock("test")()
	require.Less(t, time.Since(start), 400*time.Millisecond)
}
