package dockerutil

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/gofrs/flock"
)

// These are vars so tests can shorten them.
var (
	globalLockTimeout        = 2 * time.Minute
	globalLockReportInterval = 15 * time.Second
	parentPID                = os.Getppid
)

const globalLockDocsURL = "https://docs.ddev.com/en/stable/users/usage/troubleshooting/#waiting-for-another-ddev-process"

// globalLockHolder describes the process holding the global lock. It lives
// in a separate file because on Windows the locked file can't be read.
type globalLockHolder struct {
	PID     int       `json:"pid"`
	Command string    `json:"command"`
	Dir     string    `json:"dir"`
	Reason  string    `json:"reason"`
	Started time.Time `json:"started"`
}

func globalLockPath() string {
	return filepath.Join(globalconfig.GetGlobalDdevDir(), ".global.lock")
}

func globalLockInfoPath() string {
	return globalLockPath() + ".info"
}

// readGlobalLockHolder returns nil when the holder hasn't written its info yet.
func readGlobalLockHolder() *globalLockHolder {
	b, err := os.ReadFile(globalLockInfoPath())
	if err != nil {
		return nil
	}
	var h globalLockHolder
	if json.Unmarshal(b, &h) != nil || h.PID == 0 {
		return nil
	}
	return &h
}

// String reads like "'ddev start' in ~/workspace/d11 (pid 41233, running 14s), which is doing ddev-router setup".
func (h *globalLockHolder) String() string {
	if h == nil {
		return "another ddev process"
	}
	return fmt.Sprintf("'%s' in %s (pid %d, running %s), which is doing %s",
		h.Command, h.Dir, h.PID, time.Since(h.Started).Round(time.Second), h.Reason)
}

// AcquireGlobalLock serializes the check-then-act sections that touch Docker
// resources shared by every project (router, network). The returned unlock
// function is safe to call more than once. The lock is not reentrant, so
// do not call this again while holding it.
//
// While waiting it names the holding process and reports progress, so a
// wait isn't mistaken for a hang. After globalLockTimeout it warns and
// proceeds unlocked, which is the behavior from before the lock existed.
func AcquireGlobalLock(reason string) (unlock func()) {
	noop := func() {}
	fl := flock.New(globalLockPath())
	locked, err := fl.TryLock()
	if err == nil && !locked {
		locked, err = waitForGlobalLock(fl, reason)
	}
	if err != nil || !locked {
		_ = fl.Close()
		if err != nil {
			util.Warning("Unable to use the global ddev lock %s for %s, continuing without it: %v", globalLockPath(), reason, err)
		}
		return noop
	}

	acquired := time.Now()
	writeGlobalLockHolder(reason)
	util.Debug("Acquired global ddev lock for %s", reason)
	var once sync.Once
	return func() {
		once.Do(func() {
			_ = os.Remove(globalLockInfoPath())
			_ = fl.Unlock()
			_ = fl.Close()
			util.Debug("Released global ddev lock for %s after %s", reason, time.Since(acquired).Round(time.Millisecond))
		})
	}
}

// waitForGlobalLock returns locked=false with a nil error when it gives up.
func waitForGlobalLock(fl *flock.Flock, reason string) (bool, error) {
	holder := readGlobalLockHolder()
	// No holder info means the holder just released the lock or hasn't written it yet.
	for i := 0; holder == nil && i < 5; i++ {
		if locked, err := fl.TryLock(); err != nil || locked {
			return locked, err
		}
		time.Sleep(50 * time.Millisecond)
		holder = readGlobalLockHolder()
	}
	// A ddev run by the holder, such as from a hook, would otherwise wait on its own parent.
	if holder != nil && holder.PID == parentPID() {
		util.Debug("Global ddev lock for %s is held by parent process %d, continuing without it", reason, holder.PID)
		return false, nil
	}
	output.UserOut.Printf("Waiting for %s. Press Ctrl-C to cancel.", holder)

	start := time.Now()
	deadline := start.Add(globalLockTimeout)
	for {
		next := time.Now().Add(globalLockReportInterval)
		if next.After(deadline) {
			next = deadline
		}
		ctx, cancel := context.WithDeadline(context.Background(), next)
		locked, err := fl.TryLockContext(ctx, 200*time.Millisecond)
		cancel()
		waited := time.Since(start).Round(time.Second)
		switch {
		case locked:
			output.UserOut.Printf("Done waiting for the global ddev lock after %s.", waited)
			return true, nil
		case err != nil && !errors.Is(err, context.DeadlineExceeded):
			return false, err
		case !time.Now().Before(deadline):
			util.Warning("Gave up after %s waiting for %s. Continuing without the lock, so the two commands may conflict over the shared router or network. See %s", waited, readGlobalLockHolder(), globalLockDocsURL)
			return false, nil
		}
		if h := readGlobalLockHolder(); h != nil && (holder == nil || h.PID != holder.PID) {
			holder = h
			output.UserOut.Printf("Still waiting (%s), now for %s...", waited, holder)
		} else {
			output.UserOut.Printf("Still waiting (%s)...", waited)
		}
	}
}

func writeGlobalLockHolder(reason string) {
	dir, _ := os.Getwd()
	h := globalLockHolder{
		PID:     os.Getpid(),
		Command: strings.Join(append([]string{"ddev"}, os.Args[1:]...), " "),
		Dir:     fileutil.ShortHomeJoin(dir),
		Reason:  reason,
		Started: time.Now(),
	}
	b, _ := json.Marshal(h)
	if err := os.WriteFile(globalLockInfoPath(), b, 0644); err != nil {
		util.Debug("Unable to write %s: %v", globalLockInfoPath(), err)
	}
}
