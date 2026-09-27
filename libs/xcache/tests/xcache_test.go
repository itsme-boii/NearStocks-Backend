package tests

import (
	"github/eugenix-io/logx-inf-backend/libs/xcache"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Test cache Set() function
// Test cache Get() function
// Test cache IsExpired() function

func TestCacheSet(t *testing.T) {
	cache := xcache.NewCache[int](0, time.Now())
	cache.Set(10, time.Second)
	cacheData := *cache.Get()
	assert.Equalf(t, 10, cacheData, "Expected 10, got %v", cacheData)
}

func TestCacheGetWithExpiry(t *testing.T) {
	cache := xcache.NewCache[int](0, time.Now())
	cache.Set(10, 500*time.Microsecond)
	cacheData := cache.Get()
	assert.Equalf(t, *cacheData, 10, "Expected 10, got %v", cacheData)
	// Expire the cache
	time.Sleep(500 * time.Microsecond)
	cacheData = cache.Get()
	assert.Nil(t, cacheData, "Expected nil, got %v", cacheData)
}

func TestCacheGetWithExpiry2(t *testing.T) {
	cache := xcache.NewCache[int](0, time.Now())
	// Test with new expiry duration
	cache.Set(10, 20*time.Microsecond)
	time.Sleep(21 * time.Microsecond)
	cacheData := cache.Get(1 * time.Microsecond)
	assert.Nil(t, cacheData, "Expected nil, got %v", cacheData)

	cacheData = cache.Get(10 * time.Millisecond)
	assert.Equalf(t, *cacheData, 10, "Expected 10, got %v", cacheData)
}

func TestCacheIsExpired(t *testing.T) {
	cache := xcache.NewCache(0, time.Now())
	cache.Set(10, 5*time.Millisecond)
	assert.False(t, cache.IsExpired(), "Expected false, got true")
	// Expire the cache
	time.Sleep(1 * 5*time.Millisecond)
	assert.True(t, cache.IsExpired(), "Expected true, got false")

	// Test with new expiry duration
	cache.Set(10, 5*time.Millisecond)
	time.Sleep(5 * time.Millisecond)
	assert.True(t, cache.IsExpired(1*time.Millisecond), "Expected true, got false")
	assert.False(t, cache.IsExpired(15*time.Millisecond), "Expected false, got true")
}
