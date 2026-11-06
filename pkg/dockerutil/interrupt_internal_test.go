package dockerutil

import (
	"context"
	"testing"
	"time"

	cerrdefs "github.com/containerd/errdefs"
	ddevImages "github.com/ddev/ddev/pkg/docker"
	"github.com/ddev/ddev/pkg/util"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
	"github.com/stretchr/testify/require"
)

// TestCleanUpInFlightWaitsForCreate interrupts after the daemon has created a
// container but before ContainerCreate has returned to DDEV.
func TestCleanUpInFlightWaitsForCreate(t *testing.T) {
	name := "TestCleanUpInFlight-" + util.RandString(6)
	created, untrack := trackContainer(name)
	t.Cleanup(untrack)
	t.Cleanup(func() { _ = RemoveContainer(name) })

	ctx, apiClient, err := GetDockerClient()
	require.NoError(t, err)
	cleanupCtx, cancel := context.WithTimeout(ctx, time.Minute)
	t.Cleanup(cancel)
	done := make(chan struct{})
	go func() {
		cleanUpInFlight(cleanupCtx)
		close(done)
	}()

	_, err = apiClient.ContainerCreate(ctx, client.ContainerCreateOptions{
		Config: &container.Config{Image: ddevImages.GetWebImage(), Cmd: []string{"true"}},
		Name:   name,
	})
	require.NoError(t, err)
	select {
	case <-done:
		t.Fatal("cleanup finished before ContainerCreate returned")
	case <-time.After(500 * time.Millisecond):
	}

	created()
	select {
	case <-done:
	case <-time.After(time.Minute):
		t.Fatal("cleanup did not finish")
	}
	_, err = apiClient.ContainerInspect(ctx, name, client.ContainerInspectOptions{})
	require.True(t, cerrdefs.IsNotFound(err), "container %s should have been removed, inspect err=%v", name, err)
}
