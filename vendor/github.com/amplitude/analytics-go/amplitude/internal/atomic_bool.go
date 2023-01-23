package internal

import (
	"sync/atomic"
)

type AtomicBool int32

func NewAtomicBool(value bool) *AtomicBool {
	b := new(AtomicBool)
	if value {
		b.Set()
	}

	return b
}

func (b *AtomicBool) Set() {
	atomic.StoreInt32((*int32)(b), 1)
}

func (b *AtomicBool) UnSet() {
	atomic.StoreInt32((*int32)(b), 0)
}

func (b *AtomicBool) IsSet() bool {
	return atomic.LoadInt32((*int32)(b)) == 1
}
