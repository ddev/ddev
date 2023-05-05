package storages

import (
	"encoding/gob"
	"os"
	"sync"
	"time"

	"github.com/amplitude/analytics-go/amplitude/types"
	"github.com/ddev/ddev/pkg/util"
)

// NewDelayedTransmissionEventStorage initializes a new EventStorage of type delayedTransmissionEventStorage.
func NewDelayedTransmissionEventStorage(capacity int, interval time.Duration, fileName string) types.EventStorage {
	return &delayedTransmissionEventStorage{
		fileName: fileName,
		loaded:   false,
		capacity: capacity,
		interval: interval,
	}
}

type delayedTransmissionEventStorage struct {
	fileName string
	loaded   bool
	capacity int
	interval time.Duration
	cache    eventCache

	mu sync.RWMutex
}

type eventCache struct {
	lastSubmittedAt time.Time
	events          []*types.StorageEvent
	//retriedEvents   []*types.StorageEvent
}

func (s *delayedTransmissionEventStorage) PushNew(event *types.StorageEvent) {
	s.push(false, event)
}

func (s *delayedTransmissionEventStorage) ReturnBack(events ...*types.StorageEvent) {
	s.push(true, events...)
}

func (s *delayedTransmissionEventStorage) push(prepend bool, events ...*types.StorageEvent) {
	if len(events) == 0 {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	err := s.readCache()
	if err != nil {
		util.Error("Error '%s', while reading event cache.", err)
	}

	prependIndex := 0

	for _, event := range events {
		s.addNonRetriedEvent(event, prepend, &prependIndex)
	}

	err = s.writeCache()
	if err != nil {
		util.Error("Error '%s', while writing event cache.", err)
	}
}

// Pull returns a chunk of events and removes returned events from the cache.
func (s *delayedTransmissionEventStorage) Pull(count int, before time.Time) []*types.StorageEvent {
	s.mu.Lock()
	defer s.mu.Unlock()

	err := s.readCache()
	if err != nil {
		util.Error("Error '%s', while reading event cache.", err)
	}

	// Early return if capacity and interval hasn't reached.
	if !s.cache.lastSubmittedAt.Add(s.interval).Before(before) && !(len(s.cache.events) >= s.capacity) {
		return make([]*types.StorageEvent, 0)
	}

	defer func() {
		err = s.writeCache()
		if err != nil {
			util.Error("Error '%s', while writing event cache.", err)
		}
	}()

	s.cache.lastSubmittedAt = time.Now()

	if len(s.cache.events) >= count {
		events := make([]*types.StorageEvent, count)
		copy(events, s.cache.events)
		copy(s.cache.events, s.cache.events[count:])

		// Remove copied items from cache.
		s.cache.events = s.cache.events[:len(s.cache.events)-count]

		return events
	}

	events := make([]*types.StorageEvent, len(s.cache.events), count)
	copy(events, s.cache.events)

	// Clear the cache
	s.cache.events = s.cache.events[:0]

	return events
}

func (s *delayedTransmissionEventStorage) Count(before time.Time) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	err := s.readCache()
	if err != nil {
		util.Error("Error '%s', while reading event cache.", err)
	}

	if s.cache.lastSubmittedAt.Add(s.interval).Before(before) || len(s.cache.events) >= s.capacity {
		return len(s.cache.events)
	}

	return 0
}

func (s *delayedTransmissionEventStorage) addNonRetriedEvent(event *types.StorageEvent, prepend bool, prependIndex *int) {
	if prepend {
		s.cache.events = append(s.cache.events, nil)
		copy(s.cache.events[*prependIndex+1:], s.cache.events[*prependIndex:])
		s.cache.events[*prependIndex] = event
		*prependIndex++
	} else {
		s.cache.events = append(s.cache.events, event)
	}
}

// readCache reads the events from the cache file.
func (s *delayedTransmissionEventStorage) readCache() error {
	// The cache is read once per run time, early exist if already loaded.
	if s.loaded {
		return nil
	}

	file, err := os.Open(s.fileName)
	if err == nil {
		decoder := gob.NewDecoder(file)
		err = decoder.Decode(&s.cache)
	}
	file.Close()

	// If the file was properly read, mark the cache as loaded.
	if err == nil {
		s.loaded = true
	}

	return err
}

// writeCache writes the events to the cache file.
func (s *delayedTransmissionEventStorage) writeCache() error {
	file, err := os.Create(s.fileName)
	if err == nil {
		encoder := gob.NewEncoder(file)
		err = encoder.Encode(&s.cache)
	}
	file.Close()

	return err
}
