package dockerutil

import (
	"context"
	"sync"

	cerrdefs "github.com/containerd/errdefs"
	"github.com/ddev/ddev/pkg/interrupt"
	"github.com/ddev/ddev/pkg/util"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

// inFlight holds the temporary containers RunSimpleContainerExtended() is
// creating or has created and not yet removed, and the restore function of a
// terminal setupRawTerminal() has put in raw mode, so an interrupt can undo
// both.
var inFlight = struct {
	sync.Mutex
	containers      map[*trackedContainer]struct{}
	restoreTerminal func()
}{containers: map[*trackedContainer]struct{}{}}

// trackedContainer's done is closed once ContainerCreate has returned, and
// target is then what an interrupt should remove, or empty for nothing.
type trackedContainer struct {
	done   chan struct{}
	target string
}

func init() {
	interrupt.OnInterrupt(cleanUpInFlight)
}

// trackContainer registers a container before it is created, because the
// daemon can list it before ContainerCreate returns. Call created with
// ContainerCreate's result once it has returned.
func trackContainer() (created func(id string, err error), untrack func()) {
	tc := &trackedContainer{done: make(chan struct{})}
	inFlight.Lock()
	defer inFlight.Unlock()
	inFlight.containers[tc] = struct{}{}
	var once sync.Once
	created = func(id string, err error) {
		once.Do(func() {
			// After an error the name may belong to some other container, and
			// socktainer reports a name conflict as a 500, not a 409.
			if err == nil {
				tc.target = id
			}
			close(tc.done)
		})
	}
	untrack = func() {
		inFlight.Lock()
		defer inFlight.Unlock()
		delete(inFlight.containers, tc)
	}
	return created, untrack
}

func trackRawTerminal(restore func()) (untrack func()) {
	inFlight.Lock()
	defer inFlight.Unlock()
	inFlight.restoreTerminal = restore
	return func() {
		inFlight.Lock()
		defer inFlight.Unlock()
		inFlight.restoreTerminal = nil
	}
}

// cleanUpInFlight restores the terminal first, so its message is readable,
// then force-removes every tracked container in parallel.
func cleanUpInFlight(ctx context.Context) {
	inFlight.Lock()
	restore := inFlight.restoreTerminal
	inFlight.restoreTerminal = nil
	containers := inFlight.containers
	inFlight.containers = map[*trackedContainer]struct{}{}
	inFlight.Unlock()

	if restore != nil {
		restore()
	}
	if len(containers) == 0 {
		return
	}
	util.Warning("Interrupted, removing %d temporary %s...", len(containers), util.FormatPlural(len(containers), "container", "containers"))
	_, apiClient, err := GetDockerClient()
	if err != nil {
		return
	}
	var wg sync.WaitGroup
	for tc := range containers {
		wg.Go(func() {
			// Removing before the create returns would find nothing and leak it.
			select {
			case <-tc.done:
			case <-ctx.Done():
				return
			}
			id := tc.target
			if id == "" {
				return
			}
			_, err := apiClient.ContainerRemove(ctx, id, client.ContainerRemoveOptions{Force: true})
			if err == nil || cerrdefs.IsNotFound(err) {
				return
			}
			util.Debug("Unable to remove container %s after interrupt: %v", TruncateID(id), err)
			// A removal RunSimpleContainerExtended() already started is
			// cancelled if the process exits first.
			wait := apiClient.ContainerWait(ctx, id, client.ContainerWaitOptions{Condition: container.WaitConditionRemoved})
			select {
			case <-wait.Result:
			case err := <-wait.Error:
				util.Debug("Unable to wait for removal of container %s: %v", TruncateID(id), err)
			}
		})
	}
	wg.Wait()
}
