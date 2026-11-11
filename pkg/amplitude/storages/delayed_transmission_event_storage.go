package storages

import (
	"bytes"
	"encoding/gob"
	"os"
	"sync"
	"time"

	"github.com/amplitude/analytics-go/amplitude/types"
)

// NewDelayedTransmissionEventStorage initializes a new EventStorage of type delayedTransmissionEventStorage.
func NewDelayedTransmissionEventStorage(
	logger types.Logger,
	capacity int,
	interval time.Duration,
	fileName string,
) types.EventStorage {
	return &delayedTransmissionEventStorage{
		logger:   logger,
		fileName: fileName,
		capacity: capacity,
		interval: interval,
	}
}

type delayedTransmissionEventStorage struct {
	logger   types.Logger
	fileName string
	capacity int
	interval time.Duration
	cache    eventCache

	mu sync.RWMutex
}

// eventCache is the structure used for the cache file.
type eventCache struct {
	LastSubmittedAt time.Time
	Events          []*types.StorageEvent
}

// PushNew writes a new event to the cache.
func (s *delayedTransmissionEventStorage) PushNew(event *types.StorageEvent) {
	s.push(false, event)
}

// ReturnBack is used to return back previously pulled events to the cache.
func (s *delayedTransmissionEventStorage) ReturnBack(events ...*types.StorageEvent) {
	s.push(true, events...)
}

// push prepends or appends events to the cache.
func (s *delayedTransmissionEventStorage) push(prepend bool, events ...*types.StorageEvent) {
	if len(events) == 0 {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Without the lock, pushing anyway risks only the old race.
	if unlock, ok := LockFile(s.fileName + ".lock"); ok {
		defer unlock()
	}

	err := s.readCache()
	if err != nil {
		s.logger.Errorf("Error '%s', while reading event cache.", err)
	}

	prependIndex := 0

	for _, event := range events {
		s.addNonRetriedEvent(event, prepend, &prependIndex)
	}

	err = s.writeCache()
	if err != nil {
		s.logger.Errorf("Error '%s', while writing event cache.", err)
	}

	s.logger.Debugf("Pushed %d events to the cache.", len(events))
}

// Pull returns a chunk of events and removes returned events from the cache.
func (s *delayedTransmissionEventStorage) Pull(count int, before time.Time) []*types.StorageEvent {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Without the lock, a later Pull tries again.
	unlock, ok := LockFile(s.fileName + ".lock")
	if !ok {
		return make([]*types.StorageEvent, 0)
	}
	defer unlock()

	err := s.readCache()
	if err != nil {
		s.logger.Errorf("Error '%s', while reading event cache.", err)
	}

	// Early return if capacity and interval hasn't reached.
	if !s.cache.LastSubmittedAt.Add(s.interval).Before(before) && !(len(s.cache.Events) >= s.capacity) {
		return make([]*types.StorageEvent, 0)
	}

	defer func() {
		err = s.writeCache()
		if err != nil {
			s.logger.Errorf("Error '%s', while writing event cache.", err)
		}
	}()

	s.cache.LastSubmittedAt = time.Now()

	if len(s.cache.Events) >= count {
		events := make([]*types.StorageEvent, count)
		copy(events, s.cache.Events)
		copy(s.cache.Events, s.cache.Events[count:])

		// Remove copied items from cache.
		s.cache.Events = s.cache.Events[:len(s.cache.Events)-count]

		s.logger.Debugf("Pulled %d events from the cache.", len(events))

		return events
	}

	events := make([]*types.StorageEvent, len(s.cache.Events), count)
	copy(events, s.cache.Events)

	// Clear the cache
	s.cache.Events = s.cache.Events[:0]

	s.logger.Debugf("Pulled %d events from the cache.", len(events))

	return events
}

// Count returns the number of events ready for transmission.
func (s *delayedTransmissionEventStorage) Count(before time.Time) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	if unlock, ok := LockFile(s.fileName + ".lock"); ok {
		defer unlock()
	}

	err := s.readCache()
	if err != nil {
		s.logger.Errorf("Error '%s', while reading event cache.", err)
	}

	if s.cache.LastSubmittedAt.Add(s.interval).Before(before) || len(s.cache.Events) >= s.capacity {
		return len(s.cache.Events)
	}

	return 0
}

// addNonRetriedEvent adds the events to the internal memory cache.
func (s *delayedTransmissionEventStorage) addNonRetriedEvent(event *types.StorageEvent, prepend bool, prependIndex *int) {
	if prepend {
		s.cache.Events = append(s.cache.Events, nil)
		copy(s.cache.Events[*prependIndex+1:], s.cache.Events[*prependIndex:])
		s.cache.Events[*prependIndex] = event
		*prependIndex++
	} else {
		s.cache.Events = append(s.cache.Events, event)
	}
}

// readCache reads the events from the cache file. It reads the file every
// time, because other ddev processes may have changed it since the last read.
func (s *delayedTransmissionEventStorage) readCache() error {
	s.cache = eventCache{}

	file, err := os.Open(s.fileName)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	defer file.Close()

	stat, err := file.Stat()
	if err != nil || stat.Size() == 0 {
		s.logger.Infof("Event cache is empty")

		return nil
	}

	decoder := gob.NewDecoder(file)
	if err = decoder.Decode(&s.cache); err != nil {
		// Don't keep a partly decoded cache.
		s.cache = eventCache{}
		return err
	}

	s.logger.Debugf("Read %d events from the cache.", len(s.cache.Events))

	return nil
}

// writeCache writes the events to the cache file.
func (s *delayedTransmissionEventStorage) writeCache() error {
	var content bytes.Buffer
	if err := gob.NewEncoder(&content).Encode(&s.cache); err != nil {
		return err
	}
	if err := WriteFileAtomic(s.fileName, content.Bytes()); err != nil {
		return err
	}

	s.logger.Debugf("Wrote %d events to the cache.", len(s.cache.Events))

	return nil
}
