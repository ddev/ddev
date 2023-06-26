package storages_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/amplitude/analytics-go/amplitude/types"
	"github.com/ddev/ddev/pkg/amplitude/loggers"
	"github.com/ddev/ddev/pkg/amplitude/storages"
	"github.com/stretchr/testify/suite"
)

func TestDelayedTransmissionEventStorage(t *testing.T) {
	suite.Run(t, new(DelayedTransmissionEventStorageSuite))
}

type DelayedTransmissionEventStorageSuite struct {
	suite.Suite
}

func (t *DelayedTransmissionEventStorageSuite) TestSimple() {
	event1 := &types.StorageEvent{Event: &types.Event{EventType: "event-A"}}
	event2 := &types.StorageEvent{Event: &types.Event{EventType: "event-B"}}
	event3 := &types.StorageEvent{Event: &types.Event{EventType: "event-C"}}
	event4 := &types.StorageEvent{Event: &types.Event{EventType: "event-D"}}

	require := t.Require()

	s := storages.NewDelayedTransmissionEventStorage(loggers.NewDdevLogger(true, true), 0, 0, filepath.Join(t.T().TempDir(), `TestSimple.cache`))
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

func (t *DelayedTransmissionEventStorageSuite) TestReturnBack() {
	event1 := &types.StorageEvent{Event: &types.Event{EventType: "event-A"}}
	event2 := &types.StorageEvent{Event: &types.Event{EventType: "event-B"}}
	event3 := &types.StorageEvent{Event: &types.Event{EventType: "event-C"}}
	event4 := &types.StorageEvent{Event: &types.Event{EventType: "event-D"}}
	event5 := &types.StorageEvent{Event: &types.Event{EventType: "event-E"}}

	require := t.Require()

	s := storages.NewDelayedTransmissionEventStorage(loggers.NewDdevLogger(true, true), 0, 0, filepath.Join(t.T().TempDir(), `TestReturnBack.cache`))
	s.PushNew(event1)
	s.PushNew(event2)
	s.PushNew(event3)
	s.PushNew(event4)
	s.PushNew(event5)

	require.Equal(5, s.Count(time.Time{}))
	chunk := s.Pull(4, time.Time{})
	require.Equal([]*types.StorageEvent{event1, event2, event3, event4}, chunk)

	now := time.Now()
	s.ReturnBack(event2, event3, event4, event1)

	require.Equal(5, s.Count(now))
	chunk = s.Pull(4, now)
	require.Equal([]*types.StorageEvent{event2, event3, event4, event1}, chunk)

	require.Equal(1, s.Count(now))
	chunk = s.Pull(4, now)
	require.Equal([]*types.StorageEvent{event5}, chunk)

	require.Equal(0, s.Count(now))
	chunk = s.Pull(4, now)
	require.Empty(chunk)
}

func (t *DelayedTransmissionEventStorageSuite) TestCache() {
	event1 := &types.StorageEvent{Event: &types.Event{EventType: "event-A"}}
	event2 := &types.StorageEvent{Event: &types.Event{EventType: "event-B"}}
	event3 := &types.StorageEvent{Event: &types.Event{EventType: "event-C"}}
	event4 := &types.StorageEvent{Event: &types.Event{EventType: "event-D"}}
	event5 := &types.StorageEvent{Event: &types.Event{EventType: "event-E"}}

	cacheFile := filepath.Join(t.T().TempDir(), `TestCache.cache`)

	require := t.Require()

	s1 := storages.NewDelayedTransmissionEventStorage(loggers.NewDdevLogger(true, true), 0, 0, cacheFile)
	s1.PushNew(event1)
	s1.PushNew(event2)
	s1.PushNew(event3)
	s1.PushNew(event4)
	s1.PushNew(event5)

	require.Equal(5, s1.Count(time.Time{}))
	chunk := s1.Pull(2, time.Time{})
	require.Equal([]*types.StorageEvent{event1, event2}, chunk)

	s2 := storages.NewDelayedTransmissionEventStorage(loggers.NewDdevLogger(true, true), 0, 0, cacheFile)
	require.Equal(3, s2.Count(time.Time{}))
	chunk = s2.Pull(2, time.Time{})
	require.Equal([]*types.StorageEvent{event3, event4}, chunk)
}
