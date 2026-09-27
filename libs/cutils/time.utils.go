// NOTE: All timestamps will be taken in milliseconds
package cutils

import (
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"time"
)

const SECOND_MILLI = 1000
const MINUTE_MILLI = 60 * SECOND_MILLI
const HOUR_MILLI = 60 * MINUTE_MILLI
const DAY_MILLI = 24 * HOUR_MILLI
const WEEK_MILLI = 7 * DAY_MILLI

const ALLOWED_NETWORK_DELAY_MILLI = 1 * MINUTE_MILLI

// Timestamp should be in milliseconds
func IsTimestampExpired(timestamp uint64) bool {
	return uint64(time.Now().UnixMilli()) > timestamp
}

func IsTimestampExpiredV2(timestamp uint64, allowedDelta uint64) bool {
	return uint64(time.Now().UnixMilli()) > allowedDelta+timestamp
}

func TimestampMilliNow() uint64 {
	return uint64(time.Now().UnixMilli())
}

func IsTimestampExceeding(timestamp uint64, allowedDelta uint64) bool {
	return uint64(time.Now().UnixMilli())+allowedDelta < timestamp
}

// IsTimestampWithinWindow checks if a timestamp is within the allowed time window (past or future)
// Returns true if the timestamp is within the window, false otherwise
// windowMillis is the allowed deviation in milliseconds (e.g., 5*60*1000 for 5 minutes)
func IsTimestampWithinWindow(timestamp int64, windowMillis int64) bool {
	currentTs := time.Now().UnixMilli()
	diff := currentTs - timestamp
	if diff < 0 {
		diff = -diff
	}
	return diff <= windowMillis
}

func LogTime(start time.Time, source string) {
	elapsed := time.Since(start)
	xlog.Infof("Time taken to execute %s: %s", source, elapsed)
}
