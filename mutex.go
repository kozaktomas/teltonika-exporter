package main

import (
	"sync"
	"sync/atomic"
)

// Mutex is a custom mutex that tracks whether it is locked or not.
type Mutex struct {
	sync.Mutex
	locked int32
}

func (m *Mutex) Lock() {
	m.Mutex.Lock()
	atomic.StoreInt32(&m.locked, 1)
}

func (m *Mutex) Unlock() {
	m.Mutex.Unlock()
	atomic.StoreInt32(&m.locked, 0)
}

func (m *Mutex) IsLocked() bool {
	return atomic.LoadInt32(&m.locked) == 1
}
