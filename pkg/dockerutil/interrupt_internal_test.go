package dockerutil

import (
	"context"
	"errors"
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
	created, untrack := trackContainer(util.RandString(16))
	t.Cleanup(untrack)
	t.Cleanup(func() { _ = RemoveContainer(name) })

	done := startCleanUpInFlight(t, time.Minute)
	id, err := createTestContainer(name, nil)
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

// TestCleanUpInFlightCreateNeverReturns stands in for a create whose response
// is slower than the interrupt timeout, so created is never called.
func TestCleanUpInFlightCreateNeverReturns(t *testing.T) {
	name := "TestCleanUpInFlightSlow-" + util.RandString(6)
	label := util.RandString(16)
	_, untrack := trackContainer(label)
	t.Cleanup(untrack)
	t.Cleanup(func() { _ = RemoveContainer(name) })

	_, err := createTestContainer(name, map[string]string{temporaryLabel: label})
	require.NoError(t, err)
	done := startCleanUpInFlight(t, 3*time.Second)
	waitForCleanUp(t, done)
	requireContainerExists(t, name, false)
}

// TestCleanUpInFlightCreateFails stands in for a create whose connection
// dropped after the daemon had made the container, so it returned an error.
func TestCleanUpInFlightCreateFails(t *testing.T) {
	name := "TestCleanUpInFlightFailed-" + util.RandString(6)
	label := util.RandString(16)
	created, untrack := trackContainer(label)
	t.Cleanup(untrack)
	t.Cleanup(func() { _ = RemoveContainer(name) })

	id, err := createTestContainer(name, map[string]string{temporaryLabel: label})
	require.NoError(t, err)
	require.NotEmpty(t, id)
	created("", errors.New("connection reset by peer"))
	done := startCleanUpInFlight(t, time.Minute)
	waitForCleanUp(t, done)
	requireContainerExists(t, name, false)
}

// TestCleanUpInFlightSkipsNameConflict checks that a create refused because
// another container holds the name doesn't make cleanup remove that one.
func TestCleanUpInFlightSkipsNameConflict(t *testing.T) {
	name := "TestCleanUpInFlightConflict-" + util.RandString(6)
	_, err := createTestContainer(name, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = RemoveContainer(name) })

	created, untrack := trackContainer(util.RandString(16))
	t.Cleanup(untrack)
	done := startCleanUpInFlight(t, time.Minute)
	id, err := createTestContainer(name, nil)
	require.Error(t, err, "the name is taken, so the create should fail")

	created(id, err)
	waitForCleanUp(t, done)
	requireContainerExists(t, name, true)
}

// TestRemoveByTemporaryLabelChecksLabel stands in for an API that ignores
// the label filter, which must not make cleanup remove other containers.
func TestRemoveByTemporaryLabelChecksLabel(t *testing.T) {
	label := util.RandString(16)
	fake := &ignoreFilterClient{items: []container.Summary{
		{ID: "mine", Labels: map[string]string{temporaryLabel: label}},
		{ID: "other-temporary", Labels: map[string]string{temporaryLabel: "other"}},
		{ID: "unlabeled"},
	}}
	removeByTemporaryLabel(context.Background(), fake, label)
	require.Equal(t, []string{"mine"}, fake.removed)
}

// ignoreFilterClient's ContainerList returns every container, whatever the
// filters, and ContainerRemove records what it was asked to remove.
type ignoreFilterClient struct {
	client.APIClient
	items   []container.Summary
	removed []string
}

func (c *ignoreFilterClient) ContainerList(context.Context, client.ContainerListOptions) (client.ContainerListResult, error) {
	return client.ContainerListResult{Items: c.items}, nil
}

func (c *ignoreFilterClient) ContainerRemove(_ context.Context, id string, _ client.ContainerRemoveOptions) (client.ContainerRemoveResult, error) {
	c.removed = append(c.removed, id)
	return client.ContainerRemoveResult{}, nil
}

func startCleanUpInFlight(t *testing.T, timeout time.Duration) <-chan struct{} {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
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

func createTestContainer(name string, labels map[string]string) (string, error) {
	ctx, apiClient, err := GetDockerClient()
	if err != nil {
		return "", err
	}
	c, err := apiClient.ContainerCreate(ctx, client.ContainerCreateOptions{
		Config: &container.Config{Image: ddevImages.GetWebImage(), Cmd: []string{"true"}, Labels: labels},
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
