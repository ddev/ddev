package dockerutil

import (
	"context"
	"maps"
	"sync"

	cerrdefs "github.com/containerd/errdefs"
	"github.com/ddev/ddev/pkg/interrupt"
	"github.com/ddev/ddev/pkg/util"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

// inFlight holds the temporary containers RunSimpleContainerExtended() is
// creating or has created and not yet removed, keyed by name, and the restore
// function of a terminal setupRawTerminal() has put in raw mode, so an
// interrupt can undo both. Each container's channel is closed once its
// ContainerCreate call has returned.
var inFlight = struct {
	sync.Mutex
	containers      map[string]chan struct{}
	restoreTerminal func()
}{containers: map[string]chan struct{}{}}

func init() {
	interrupt.OnInterrupt(cleanUpInFlight)
}

// trackContainer registers name before the container is created, because
// the daemon can list it before ContainerCreate returns. Call created once
// ContainerCreate has returned, whether or not it succeeded.
func trackContainer(name string) (created func(), untrack func()) {
	ch := make(chan struct{})
	inFlight.Lock()
	defer inFlight.Unlock()
	inFlight.containers[name] = ch
	created = sync.OnceFunc(func() { close(ch) })
	untrack = func() {
		inFlight.Lock()
		defer inFlight.Unlock()
		if inFlight.containers[name] == ch {
			delete(inFlight.containers, name)
		}
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
	containers := maps.Clone(inFlight.containers)
	clear(inFlight.containers)
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
	for name, created := range containers {
		wg.Go(func() {
			// Removing before the create returns would find nothing and leak it.
			select {
			case <-created:
			case <-ctx.Done():
				return
			}
			_, err := apiClient.ContainerRemove(ctx, name, client.ContainerRemoveOptions{Force: true})
			if err == nil || cerrdefs.IsNotFound(err) {
				return
			}
			util.Debug("Unable to remove container %s after interrupt: %v", name, err)
			// A removal RunSimpleContainerExtended() already started is
			// cancelled if the process exits first.
			wait := apiClient.ContainerWait(ctx, name, client.ContainerWaitOptions{Condition: container.WaitConditionRemoved})
			select {
			case <-wait.Result:
			case err := <-wait.Error:
				util.Debug("Unable to wait for removal of container %s: %v", name, err)
			}
		})
	}
	wg.Wait()
}
