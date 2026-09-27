package cutils

import (
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"strconv"
)

func CheckNonce(requestNonce int64, currentNonce string) bool {
	// Convert currentNonce (string) to int64
	currentNonceInt64, err := strconv.ParseInt(currentNonce, 10, 64)
	if err != nil {
		xlog.Errorf("Failed to convert nonce to int64 with the error %v", err)
		return false
	}

	return currentNonceInt64 == requestNonce
}
