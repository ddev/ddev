package storages_test

import (
	"testing"
	"time"

	"github.com/amplitude/analytics-go/amplitude/types"
	"github.com/ddev/ddev/pkg/amplitude/storages"
	"github.com/stretchr/testify/suite"
)

func TestInMemoryEvent(t *testing.T) {
	suite.Run(t, new(InMemoryEventStorageSuite))
}

type InMemoryEventStorageSuite struct {
	suite.Suite
}

func (t *InMemoryEventStorageSuite) TestSimple() {
	event1 := &types.StorageEvent{Event: &types.Event{EventType: "event-A"}}
	event2 := &types.StorageEvent{Event: &types.Event{EventType: "event-B"}}
	event3 := &types.StorageEvent{Event: &types.Event{EventType: "event-C"}}
	event4 := &types.StorageEvent{Event: &types.Event{EventType: "event-D"}}

	require := t.Require()

	s := storages.NewDelayedTransmissionEventStorage()
	s.PushNew(event1)
	require.Equal(1, s.Count(time.Time{}))
	s.PushNew(event2)
	require.Equal(2, s.Count(time.Time{}))
	s.PushNew(event3)
	require.Equal(3, s.Count(time.Time{}))
	s.PushNew(event4)
	require.Equal(4, s.Count(time.Time{}))

	chunk := s.Pull(3, time.Time{})
	require.Equal(1, s.Count(time.Time{}))
	require.Equal([]*types.StorageEvent{event1, event2, event3}, chunk)

	s.PushNew(event2)
	require.Equal(2, s.Count(time.Time{}))
	s.PushNew(event3)
	require.Equal(3, s.Count(time.Time{}))
	s.PushNew(event1)
	require.Equal(4, s.Count(time.Time{}))

	chunk = s.Pull(3, time.Time{})
	require.Equal(1, s.Count(time.Time{}))
	require.Equal([]*types.StorageEvent{event4, event2, event3}, chunk)

	chunk = s.Pull(3, time.Time{})
	require.Equal(0, s.Count(time.Time{}))
	require.Equal([]*types.StorageEvent{event1}, chunk)

	chunk = s.Pull(3, time.Time{})
	require.Equal(0, s.Count(time.Time{}))
	require.Empty(chunk)
}

func (t *InMemoryEventStorageSuite) TestReturnBack() {
	event1 := &types.StorageEvent{Event: &types.Event{EventType: "event-A"}}
	event2 := &types.StorageEvent{Event: &types.Event{EventType: "event-B"}}
	event3 := &types.StorageEvent{Event: &types.Event{EventType: "event-C"}}
	event4 := &types.StorageEvent{Event: &types.Event{EventType: "event-D"}}
	event5 := &types.StorageEvent{Event: &types.Event{EventType: "event-E"}}

	require := t.Require()

	s := storages.NewDelayedTransmissionEventStorage()
	s.PushNew(event1)
	s.PushNew(event2)
	s.PushNew(event3)
	s.PushNew(event4)
	s.PushNew(event5)

	require.Equal(5, s.Count(time.Time{}))
	chunk := s.Pull(4, time.Time{})
	require.Equal([]*types.StorageEvent{event1, event2, event3, event4}, chunk)

	now := time.Now()
	event3.RetryAt = now.Add(time.Millisecond * 300)
	event4.RetryAt = now.Add(time.Millisecond * 100)
	s.ReturnBack(event2, event3, event4, event1)

	require.Equal(3, s.Count(now))
	chunk = s.Pull(4, now)
	require.Equal([]*types.StorageEvent{event2, event1, event5}, chunk)

	require.Equal(0, s.Count(now))
	chunk = s.Pull(4, now)
	require.Empty(chunk)

	now = now.Add(time.Millisecond * 150)
	require.Equal(1, s.Count(now))
	chunk = s.Pull(4, now)
	require.Equal([]*types.StorageEvent{event4}, chunk)

	require.Equal(0, s.Count(now))
	chunk = s.Pull(4, now)
	require.Empty(chunk)

	now = now.Add(time.Millisecond * 200)
	require.Equal(1, s.Count(now))
	chunk = s.Pull(4, now)
	require.Equal([]*types.StorageEvent{event3}, chunk)

	require.Equal(0, s.Count(now))
	chunk = s.Pull(4, now)
	require.Empty(chunk)
}
