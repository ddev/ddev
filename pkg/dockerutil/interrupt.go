package dockerutil

import (
	"context"
	"sync"
	"time"

	cerrdefs "github.com/containerd/errdefs"
	"github.com/ddev/ddev/pkg/interrupt"
	"github.com/ddev/ddev/pkg/util"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

// temporaryLabel marks a temporary container with a value unique to one
// RunSimpleContainerExtended() call, so an interrupt can find the container
// before ContainerCreate has returned its ID.
const temporaryLabel = "com.ddev.temporary-id"

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
// label is the temporaryLabel value the container is created with.
type trackedContainer struct {
	done   chan struct{}
	target string
	label  string
}

func init() {
	interrupt.OnInterrupt(cleanUpInFlight)
}

// trackContainer registers a container before it is created, because the
// daemon can list it before ContainerCreate returns. Create the container
// with temporaryLabel set to label, and call created with ContainerCreate's
// result once it has returned.
func trackContainer(label string) (created func(id string, err error), untrack func()) {
	tc := &trackedContainer{done: make(chan struct{}), label: label}
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
	// Leave time to remove by label a container whose create hasn't returned.
	// Docker Desktop on macOS can take more than a second to force-remove one.
	waitCtx := ctx
	if deadline, ok := ctx.Deadline(); ok {
		var cancel context.CancelFunc
		waitCtx, cancel = context.WithDeadline(ctx, deadline.Add(-2*time.Second))
		defer cancel()
	}
	var wg sync.WaitGroup
	for tc := range containers {
		wg.Go(func() {
			// Removing before the create returns would find nothing and leak it.
			select {
			case <-tc.done:
			case <-waitCtx.Done():
				removeByTemporaryLabel(ctx, apiClient, tc.label)
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

// removeByTemporaryLabel removes the container a create that hasn't returned
// may already have made. Unlike its name, the label can't belong to another
// container.
func removeByTemporaryLabel(ctx context.Context, apiClient client.APIClient, label string) {
	if label == "" || ctx.Err() != nil {
		return
	}
	list, err := apiClient.ContainerList(ctx, client.ContainerListOptions{
		All:     true,
		Filters: client.Filters{}.Add("label", temporaryLabel+"="+label),
	})
	if err != nil {
		util.Debug("Unable to list container labeled %s=%s after interrupt: %v", temporaryLabel, label, err)
		return
	}
	for _, c := range list.Items {
		// Some APIs ignore the label filter, as socktainer's image list did.
		if c.Labels[temporaryLabel] != label {
			continue
		}
		_, err := apiClient.ContainerRemove(ctx, c.ID, client.ContainerRemoveOptions{Force: true})
		if err != nil && !cerrdefs.IsNotFound(err) {
			util.Debug("Unable to remove container %s after interrupt: %v", TruncateID(c.ID), err)
		}
	}
}
