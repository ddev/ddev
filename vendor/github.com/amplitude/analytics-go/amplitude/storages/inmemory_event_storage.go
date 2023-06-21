package storages

import (
	"sort"
	"sync"
	"time"

	"github.com/amplitude/analytics-go/amplitude/types"
)

func NewInMemoryEventStorage() types.EventStorage {
	return &inMemoryEventStorage{}
}

type inMemoryEventStorage struct {
	events        []*types.StorageEvent
	retriedEvents []*types.StorageEvent

	mu sync.RWMutex
}

func (s *inMemoryEventStorage) PushNew(event *types.StorageEvent) {
	s.push(false, event)
}

func (s *inMemoryEventStorage) ReturnBack(events ...*types.StorageEvent) {
	s.push(true, events...)
}

func (s *inMemoryEventStorage) push(prepend bool, events ...*types.StorageEvent) {
	if len(events) == 0 {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	prependIndex := 0

	for _, event := range events {
		if event.RetryAt.IsZero() {
			s.addNonRetriedEvent(event, prepend, &prependIndex)
		} else {
			s.addRetriedEvent(event)
		}
	}
}

// Pull returns a chunk of events.
func (s *inMemoryEventStorage) Pull(count int, before time.Time) []*types.StorageEvent {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.events) >= count {
		events := make([]*types.StorageEvent, count)
		copy(events, s.events)
		copy(s.events, s.events[count:])
		s.events = s.events[:len(s.events)-count]

		return events
	}

	events := make([]*types.StorageEvent, len(s.events), count)
	copy(events, s.events)
	s.events = s.events[:0]

	retriedCount := 0
	for retriedCount < len(s.retriedEvents) && len(s.events)+retriedCount < count && s.retriedEvents[retriedCount].RetryAt.Before(before) {
		retriedCount++
	}

	if retriedCount == 0 {
		return events
	}

	events = append(events, s.retriedEvents[:retriedCount]...)
	copy(s.retriedEvents, s.retriedEvents[retriedCount:])
	s.retriedEvents = s.retriedEvents[:len(s.retriedEvents)-retriedCount]

	return events
}

func (s *inMemoryEventStorage) Count(before time.Time) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := len(s.events)

	for _, event := range s.retriedEvents {
		if event.RetryAt.Before(before) {
			count++
		} else {
			break
		}
	}

	return count
}

func (s *inMemoryEventStorage) addNonRetriedEvent(event *types.StorageEvent, prepend bool, prependIndex *int) {
	if prepend {
		s.events = append(s.events, nil)
		copy(s.events[*prependIndex+1:], s.events[*prependIndex:])
		s.events[*prependIndex] = event
		*prependIndex++
	} else {
		s.events = append(s.events, event)
	}
}

func (s *inMemoryEventStorage) addRetriedEvent(event *types.StorageEvent) {
	index := sort.Search(len(s.retriedEvents), func(i int) bool {
		return s.retriedEvents[i].RetryAt.After(event.RetryAt)
	})
	if index == len(s.retriedEvents) {
		s.retriedEvents = append(s.retriedEvents, event)
	} else {
		s.retriedEvents = append(s.retriedEvents, nil)
		copy(s.retriedEvents[index+1:], s.retriedEvents[index:])
		s.retriedEvents[index] = event
	}
}
