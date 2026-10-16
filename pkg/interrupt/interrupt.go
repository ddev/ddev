// Package interrupt runs registered cleanup when ddev receives Ctrl-C,
// SIGTERM, or SIGHUP, so that an interrupted command does not leave
// temporary containers or a raw-mode terminal behind.
//
// Code that handles some of these signals itself calls Delegate() while it
// does, and the handler then runs the cleanup for those without exiting.
package interrupt

import (
	"context"
	"maps"
	"os"
	"os/signal"
	"slices"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

var (
	mu        sync.Mutex
	cleanups  = map[int]func(context.Context){}
	nextID    int
	delegates = map[os.Signal]int{}
	installed bool
	exiting   atomic.Bool
)

// OnInterrupt registers f to run on every interrupt until unregister is
// called. f runs more than once when a caller has delegated, and should give
// up when ctx is done.
func OnInterrupt(f func(context.Context)) (unregister func()) {
	mu.Lock()
	defer mu.Unlock()
	id := nextID
	nextID++
	cleanups[id] = f
	return func() {
		mu.Lock()
		defer mu.Unlock()
		delete(cleanups, id)
	}
}

// Delegate tells the handler that the caller handles sigs itself, so for
// those the handler runs the cleanup but leaves exiting to the caller. Call
// the returned release function when the caller stops handling them.
func Delegate(sigs ...os.Signal) (release func()) {
	mu.Lock()
	defer mu.Unlock()
	for _, sig := range sigs {
		delegates[sig]++
	}
	var once sync.Once
	return func() {
		once.Do(func() {
			mu.Lock()
			defer mu.Unlock()
			for _, sig := range sigs {
				delegates[sig]--
			}
		})
	}
}

// Install starts the handler. It is safe to call more than once.
func Install() {
	mu.Lock()
	defer mu.Unlock()
	if installed {
		return
	}
	installed = true
	sigs := make(chan os.Signal, 2)
	for _, sig := range handledSignals {
		// Notify would undo nohup, or a shell ignoring SIGINT for a background job.
		if !signal.Ignored(sig) {
			signal.Notify(sigs, sig)
		}
	}
	go func() {
		for sig := range sigs {
			handle(sig, sigs)
		}
	}()
}

func handle(sig os.Signal, sigs chan os.Signal) {
	mu.Lock()
	fns := slices.Collect(maps.Values(cleanups))
	delegated := delegates[sig] > 0
	mu.Unlock()
	if !delegated {
		exiting.Store(true)
	}

	done := make(chan struct{})
	go func() {
		Run(fns, cleanupTimeout(sig))
		close(done)
	}()
	if delegated {
		<-done
		return
	}
	// A second signal while cleanup is stuck, for example on a hung Docker
	// daemon, exits at once.
	select {
	case <-done:
	case <-sigs:
	}
	exit(sig)
}

// WaitIfExiting blocks for good once the handler has begun cleaning up before
// exiting. Fatal error paths call it, because cleanup such as removing a
// container makes in-flight work fail, and exiting on that failure would
// beat the handler and report an error instead of the signal.
func WaitIfExiting() {
	if exiting.Load() {
		select {}
	}
}

// Run calls each cleanup function in parallel and waits for them to return
// or for timeout to pass.
func Run(fns []func(context.Context), timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	var wg sync.WaitGroup
	for _, f := range fns {
		wg.Go(func() { f(ctx) })
	}
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-ctx.Done():
	}
}

// ExitCode returns the shell convention 128+signal number for sig.
func ExitCode(sig os.Signal) int {
	if s, ok := sig.(syscall.Signal); ok {
		return 128 + int(s)
	}
	return 1
}
