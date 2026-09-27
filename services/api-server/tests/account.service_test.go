package tests

import (
	"fmt"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"github/eugenix-io/logx-inf-backend/services/api-server/services"
	"github/eugenix-io/logx-inf-backend/testutils"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestIncrementNonce(t *testing.T) {
	subaccountIdHex := "0x000000000001a7de990d10a7d8b53476525b1212bacf645cb7b0000000000002"
	testutils.WithSetupMockRedis(t, func() {
		// Increment nonce multiple times
		for i := 121234; i < 121239; i++ {
			services.IncrementNonce(subaccountIdHex, fmt.Sprintf("%d", i))
			newNonce, err := services.GetNonce(subaccountIdHex)
			assert.NoErrorf(t, err, "Failed to get nonce: %v", err)
			assert.Equalf(t, fmt.Sprintf("%d", i+1), newNonce, "Expected nonce to be 1, got %s", newNonce)
		}
	})
}

func TestWithNonceRedisLock(t *testing.T) {
	subaccountIdHex := "0x000000000001a7de990d10a7d8b53476525b1212bacf645cb7b0000000000002"
	rand.NewSource(time.Now().UnixNano())

	testutils.WithSetupMockRedis(t, func() {
		// Set initial nonce value
		services.IncrementNonce(subaccountIdHex, "121233")
		var wg sync.WaitGroup
		for i := 121234; i < 121334; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := services.WithNonceRedisLock(subaccountIdHex, func(currentNonce string) (*xredis.NOOP, error) {
					services.IncrementNonce(subaccountIdHex, currentNonce)
					var timeDelay = time.Duration(rand.Intn(3)) * time.Millisecond
					time.Sleep(timeDelay)
					return nil, nil
				})
				assert.NoErrorf(t, err, "Failed to get nonce: %v", err)
			}()
		}
		wg.Wait()
		nonce, err := services.GetNonce(subaccountIdHex)
		assert.NoError(t, err, "Failed to get nonce")
		assert.Equal(t, "121334", nonce, "Expected nonce to be 121334, got %s", nonce)
	})
}
