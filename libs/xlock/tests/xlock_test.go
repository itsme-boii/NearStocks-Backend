package xlock_test

import (
	"fmt"
	"github/eugenix-io/logx-inf-backend/libs/xlock"
	"sync"
	"testing"
)

func TestNewLockMap(t *testing.T) {
	lockMap := xlock.NewLockMap()
	if lockMap == nil {
		t.Fatal("Expected non-nil LockMap instance")
	}
}

func TestGetLock_CreatesNewLock(t *testing.T) {
	lockMap := xlock.NewLockMap()
	lock := lockMap.GetLock("test")

	if lock == nil {
		t.Fatal("Expected non-nil lock to be created")
	}

	// Ensure that the lock is stored and retrieved correctly
	retrievedLock := lockMap.GetLock("test")
	if lock != retrievedLock {
		t.Fatal("Expected the same lock to be retrieved for the same key")
	}
}

func TestGetLock_ReturnsSameLockForKey(t *testing.T) {
	lockMap := xlock.NewLockMap()
	lock1 := lockMap.GetLock("test")
	lock2 := lockMap.GetLock("test")

	if lock1 != lock2 {
		t.Fatal("Expected the same lock to be returned for the same key")
	}
}

func TestGetLock_Concurrency(t *testing.T) {
	lockMap := xlock.NewLockMap()
	var wg sync.WaitGroup
	key := "concurrentKey"

	wg.Add(2)
	go func() {
		defer wg.Done()
		lock := lockMap.GetLock(key)
		lock.Lock()
		defer lock.Unlock()
	}()

	go func() {
		defer wg.Done()
		lock := lockMap.GetLock(key)
		lock.Lock()
		defer lock.Unlock()
	}()

	wg.Wait()
}

func TestLockAndUnlock(t *testing.T) {
	lockMap := xlock.NewLockMap()
	lock := lockMap.GetLock("test")

	// Test that the lock can be acquired and released without deadlocking
	lock.Lock()
	isLocked := true

	// Channel to communicate errors and ensure the lock is released
	errChan := make(chan error, 1)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()

		// This will block until the lock is released
		lock.Lock()
		defer lock.Unlock()

		if isLocked {
			errChan <- fmt.Errorf("Lock should be released before the goroutine can acquire it")
		} else {
			errChan <- nil
		}
	}()

	t.Log("Main goroutine holding the lock")
	lock.Unlock() // Unlock so that the goroutine can proceed
	isLocked = false
	t.Log("Main goroutine released the lock")

	// Wait for the goroutine to finish
	wg.Wait()

	// Check for any errors reported by the goroutine
	if err := <-errChan; err != nil {
		t.Fatal(err)
	}

	// Close the error channel
	close(errChan)
}
