package xlock

import (
	"sync"
)

type LockMap struct {
	locks    map[string]*sync.Mutex
	mapMutex sync.Mutex
}

// NewLockMap initializes a new LockMap instance.
func NewLockMap() *LockMap {
	return &LockMap{
		locks: make(map[string]*sync.Mutex),
	}
}

// GetLock returns the lock associated with the given key, creating it if it doesn't exist.
func (lm *LockMap) GetLock(key string) *sync.Mutex {
	lm.mapMutex.Lock()
	defer lm.mapMutex.Unlock()

	// If the lock doesn't exist for this key, create it
	if _, exists := lm.locks[key]; !exists {
		lm.locks[key] = &sync.Mutex{}
	}

	return lm.locks[key]
}
