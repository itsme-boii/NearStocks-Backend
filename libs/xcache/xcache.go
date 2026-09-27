package xcache

import (
	"sync"
	"time"
)

// NOTE NOTE NOTE: If your object stores nil value then you can't differentiate between nil value and expired value
// So, use IsExpired() method to check if the value is expired or not

type CacheImp[T any] struct {
	object      T
	lastSetTime time.Time
	expiry      time.Time
	lock        sync.Mutex
}

type Cache[T any] interface {
	Get(...time.Duration) *T
	Set(T, time.Duration)
	IsExpired(...time.Duration) bool
}

// Validate Cache[T] implements Cache[T]
var _ Cache[int] = &CacheImp[int]{}

// If expiry duration is provided then it will check if the object is not expired according to that duration
// Otherwise it will check if the object is not expired based on expiry provided at the time of setting
func (c *CacheImp[T]) Get(newExpiryDuration ...time.Duration) *T {
	if c.IsExpired(newExpiryDuration...) {
		return nil
	}
	return &c.object
}

func (c *CacheImp[T]) Set(object T, expiryDuration time.Duration) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.lastSetTime = time.Now()
	c.expiry = c.lastSetTime.Add(expiryDuration)
	c.object = object
}

// If expiry duration is provided then it will check if the object is not expired according to that duration
// Otherwise it will check if the object is not expired based on expiry provided at the time of setting
func (c *CacheImp[T]) IsExpired(newExpiryDuration ...time.Duration) bool {
	if len(newExpiryDuration) > 0 {
		return time.Now().After(c.lastSetTime.Add(newExpiryDuration[0]))
	}

	return time.Now().After(c.expiry)
}

func NewCache[T any](object T, expiry time.Time) Cache[T] {
	return &CacheImp[T]{
		object: object,
		expiry: expiry,
	}
}
