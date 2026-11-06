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

	done := startCleanUpInFlight(t)
	id, err := createTestContainer(name)
	require.NoError(t, err)
	select {
	case <-done:
		t.Fatal("cleanup finished before ContainerCreate returned")
	case <-time.After(500 * time.Millisecond):
	}

	created(id, err)
	waitForCleanUp(t, done)
	requireContainerExists(t, name, false)
}

// TestCleanUpInFlightSkipsNameConflict checks that a create refused because
// another container holds the name doesn't make cleanup remove that one.
func TestCleanUpInFlightSkipsNameConflict(t *testing.T) {
	name := "TestCleanUpInFlightConflict-" + util.RandString(6)
	_, err := createTestContainer(name)
	require.NoError(t, err)
	t.Cleanup(func() { _ = RemoveContainer(name) })

	created, untrack := trackContainer(name)
	t.Cleanup(untrack)
	done := startCleanUpInFlight(t)
	id, err := createTestContainer(name)
	require.Error(t, err, "the name is taken, so the create should fail")

	created(id, err)
	waitForCleanUp(t, done)
	requireContainerExists(t, name, true)
}

func startCleanUpInFlight(t *testing.T) <-chan struct{} {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	t.Cleanup(cancel)
	done := make(chan struct{})
	go func() {
		cleanUpInFlight(ctx)
		close(done)
	}()
	return done
}

func waitForCleanUp(t *testing.T, done <-chan struct{}) {
	select {
	case <-done:
	case <-time.After(time.Minute):
		t.Fatal("cleanup did not finish")
	}
}

func createTestContainer(name string) (string, error) {
	ctx, apiClient, err := GetDockerClient()
	if err != nil {
		return "", err
	}
	c, err := apiClient.ContainerCreate(ctx, client.ContainerCreateOptions{
		Config: &container.Config{Image: ddevImages.GetWebImage(), Cmd: []string{"true"}},
		Name:   name,
	})
	return c.ID, err
}

func requireContainerExists(t *testing.T, name string, exists bool) {
	ctx, apiClient, err := GetDockerClient()
	require.NoError(t, err)
	_, err = apiClient.ContainerInspect(ctx, name, client.ContainerInspectOptions{})
	if exists {
		require.NoError(t, err, "container %s should still exist", name)
	} else {
		require.True(t, cerrdefs.IsNotFound(err), "container %s should have been removed, inspect err=%v", name, err)
	}
}
