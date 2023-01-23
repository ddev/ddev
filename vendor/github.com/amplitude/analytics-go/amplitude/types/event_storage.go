package types

import "time"

type EventStorage interface {
	PushNew(event *StorageEvent)
	ReturnBack(events ...*StorageEvent)
	Pull(count int, before time.Time) []*StorageEvent
	Count(before time.Time) int
}

type StorageEvent struct {
	*Event

	RetryAt    time.Time
	RetryCount int
}
