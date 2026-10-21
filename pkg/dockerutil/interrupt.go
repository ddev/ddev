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

// inFlight holds the temporary containers RunSimpleContainerExtended() has
// created and not yet removed, and the restore function of a terminal
// setupRawTerminal() has put in raw mode, so an interrupt can undo both.
var inFlight = struct {
	sync.Mutex
	containers      map[string]struct{}
	restoreTerminal func()
}{containers: map[string]struct{}{}}

func init() {
	interrupt.OnInterrupt(cleanUpInFlight)
}

func trackContainer(id string) (untrack func()) {
	inFlight.Lock()
	defer inFlight.Unlock()
	inFlight.containers[id] = struct{}{}
	return func() {
		inFlight.Lock()
		defer inFlight.Unlock()
		delete(inFlight.containers, id)
	}
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
	ids := make([]string, 0, len(inFlight.containers))
	for id := range inFlight.containers {
		ids = append(ids, id)
	}
	clear(inFlight.containers)
	inFlight.Unlock()

	if restore != nil {
		restore()
	}
	if len(ids) == 0 {
		return
	}
	util.Warning("Interrupted, removing %d temporary %s...", len(ids), util.FormatPlural(len(ids), "container", "containers"))
	_, apiClient, err := GetDockerClient()
	if err != nil {
		return
	}
	var wg sync.WaitGroup
	for _, id := range ids {
		wg.Go(func() {
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
