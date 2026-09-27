package testutils

import (
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"os"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
)

func WithSetupMockRedis(t *testing.T, fn func()) {
	mredis, err := miniredis.Run()
	assert.NoError(t, err, "Error in running miniredis server")
	defer mredis.Close()
	os.Setenv("REDIS_ADDR", mredis.Addr())
	defer os.Unsetenv("REDIS_ADDR")

	// Initialize Redis client using the Miniredis address
	xredis.Initialize()

	// Ensure that Redis is completely flushed before running the test
	mredis.FlushAll()

	fn()
}
