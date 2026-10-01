package dockerutil

import (
	"context"
	"path/filepath"
	"sync"
	"time"

	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/gofrs/flock"
)

// globalLockTimeout bounds the wait for the global lock. It is a var so tests can shorten it.
var globalLockTimeout = 2 * time.Minute

// AcquireGlobalLock serializes the check-then-act sections that touch Docker
// resources shared by every project (router, network). The returned unlock
// function is safe to call more than once. The lock is not reentrant, so
// do not call this again while holding it.
//
// If the lock can't be had within globalLockTimeout, it warns and proceeds
// unlocked, which is the behavior from before the lock existed.
func AcquireGlobalLock(reason string) (unlock func()) {
	noop := func() {}
	fl := flock.New(filepath.Join(globalconfig.GetGlobalDdevDir(), ".global.lock"))
	locked, err := fl.TryLock()
	if err == nil && !locked {
		output.UserOut.Printf("Waiting for another ddev process to finish %s...", reason)
		ctx, cancel := context.WithTimeout(context.Background(), globalLockTimeout)
		defer cancel()
		locked, err = fl.TryLockContext(ctx, 200*time.Millisecond)
	}
	if err != nil || !locked {
		util.Warning("Unable to get the global ddev lock for %s, continuing without it (err=%v)", reason, err)
		_ = fl.Close()
		return noop
	}
	var once sync.Once
	return func() {
		once.Do(func() {
			_ = fl.Unlock()
			_ = fl.Close()
		})
	}
}
