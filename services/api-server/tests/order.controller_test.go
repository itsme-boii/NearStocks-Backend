package tests

// import (
// 	"fmt"
// 	"sync"
// 	"testing"
// 	"time"

// 	"github/eugenix-io/logx-inf-backend/libs/xlock"
// )

// type MockOrderServiceImpl struct {
// 	functionLocks *xlock.LockMap
// }

// func (o *MockOrderServiceImpl) matchOrdersOnChain(shouldFail bool) error {

// 	// Simulate some processing time
// 	time.Sleep(100 * time.Millisecond)

// 	// Simulate an error if shouldFail is true
// 	if shouldFail {
// 		return fmt.Errorf("simulated error")
// 	}

// 	return nil
// }

// func TestMatchOrdersOnChain_Locking(t *testing.T) {
// 	orderService := &MockOrderServiceImpl{
// 		functionLocks: xlock.NewLockMap(),
// 	}

// 	var wg sync.WaitGroup
// 	wg.Add(2)

// 	firstCallDone := make(chan struct{})

// 	// First goroutine: starts and acquires the lock
// 	go func() {
// 		defer wg.Done()
// 		t.Log("First goroutine: Attempting to acquire lock")
// 		err := orderService.matchOrdersOnChain(false)
// 		if err != nil {
// 			t.Errorf("First call failed: %v", err)
// 		}
// 		t.Log("First goroutine: Lock acquired and processing done")
// 		firstCallDone <- struct{}{}
// 	}()

// 	// Second goroutine: waits until the first has the lock, then attempts to acquire it
// 	go func() {
// 		defer wg.Done()

// 		<-firstCallDone

// 		start := time.Now()
// 		t.Log("Second goroutine: Attempting to acquire lock")
// 		err := orderService.matchOrdersOnChain(false)
// 		duration := time.Since(start)

// 		if err != nil {
// 			t.Errorf("Second call failed: %v", err)
// 		}

// 		// The second call should take longer if it had to wait for the first to release the lock
// 		if duration < 100*time.Millisecond {
// 			t.Errorf("Expected the second call to be delayed due to locking, but it was not. Duration: %v", duration)
// 		} else {
// 			t.Logf("Second goroutine: Lock acquired after waiting. Duration: %v", duration)
// 		}
// 	}()

// 	wg.Wait()
// }

// func TestMatchOrdersOnChain_Unlocking(t *testing.T) {
// 	// Setup MockOrderServiceImpl with LockMap
// 	orderService := &MockOrderServiceImpl{
// 		functionLocks: xlock.NewLockMap(),
// 	}

// 	var wg sync.WaitGroup
// 	wg.Add(2)

// 	// Channel to signal when the first goroutine is done
// 	firstCallDone := make(chan struct{})

// 	// First goroutine: locks, does some work, and unlocks
// 	go func() {
// 		defer wg.Done()

// 		functionLock := orderService.functionLocks.GetLock("MatchOrdersOnChain")
// 		functionLock.Lock()

// 		// Simulate some processing time
// 		time.Sleep(500 * time.Millisecond)

// 		functionLock.Unlock() // Explicitly unlock

// 		firstCallDone <- struct{}{}
// 	}()

// 	// Second goroutine: waits for the first to finish and then should proceed immediately
// 	go func() {
// 		defer wg.Done()

// 		// Wait until the first goroutine signals that it's done
// 		<-firstCallDone

// 		start := time.Now()

// 		duration := time.Since(start)

// 		// The second call should proceed immediately after the first goroutine unlocks
// 		if duration >= 100*time.Millisecond {
// 			t.Error("Expected the second call to proceed immediately after unlocking, but it was delayed.")
// 		}
// 	}()

// 	wg.Wait()
// }

// func TestMatchOrdersOnChain_ErrorHandling(t *testing.T) {
// 	// Setup MockOrderServiceImpl with LockMap
// 	orderService := &MockOrderServiceImpl{
// 		functionLocks: xlock.NewLockMap(),
// 	}

// 	var wg sync.WaitGroup
// 	wg.Add(2)

// 	// Channel to signal when the first goroutine is done
// 	firstCallDone := make(chan struct{})

// 	// First goroutine: locks, simulates an error, and unlocks
// 	go func() {
// 		defer wg.Done()

// 		err := orderService.matchOrdersOnChain(true) // Simulate an error
// 		if err == nil {
// 			t.Error("Expected an error but got nil")
// 		}

// 		firstCallDone <- struct{}{}
// 	}()

// 	// Second goroutine: waits for the first to finish and then should proceed
// 	go func() {
// 		defer wg.Done()

// 		// Wait until the first goroutine signals that it's done
// 		<-firstCallDone

// 		start := time.Now()
// 		err := orderService.matchOrdersOnChain(false)
// 		duration := time.Since(start)

// 		if err != nil {
// 			t.Errorf("Second call failed: %v", err)
// 		}

// 		// Check if the second call was able to proceed immediately
// 		if duration >= 150*time.Millisecond {
// 			t.Errorf("Expected the second call to proceed immediately after the first encountered an error and unlocked, but it was delayed. Duration: %v", duration)
// 		}
// 	}()

// 	wg.Wait()
// }
