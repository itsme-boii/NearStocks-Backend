package middleware

import (
	"encoding/hex"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

func RequireSequencer(ctx *gin.Context) {
	if !cutils.IsSequencerRunning() {
		xlog.Errorf("no transactions are currently allowed - Sequencer is paused")
		ctx.JSON(http.StatusForbidden, gin.H{"error": "no transactions are currently allowed - please contact dev team", "reason": "SEQUENCER_PAUSED"})
		ctx.Abort()
		return
	}

	currentSubaccount := getCurrentSubaccount(ctx)
	if currentSubaccount == nil {
		// If no subaccount is found, continue with the next handler
		ctx.Next()
		return
	}

	subaccountIdBytes, err := cutils.SubaccountIdToBytes32(currentSubaccount.ID)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "")
		return
	}
	subAccountIdHex := "0x" + hex.EncodeToString(subaccountIdBytes[:])
	subaccountID := strings.ToLower(subAccountIdHex)

	if os.Getenv("IS_UNDER_MAINTENANCE") == "1" && !cutils.IsWhitelistedSubaccount(subaccountID) {
		xlog.Errorf("no transactions are currently allowed - SubAccount is paused %v", subaccountID)
		ctx.JSON(http.StatusForbidden, gin.H{"error": "system under maintenance - no transactions are currently allowed - please contact dev team", "reason": "SYSTEM_UPGRADE"})
		ctx.Abort()
		return
	}

	if cutils.IsSubAccountPaused(subaccountID) {
		xlog.Errorf("no transactions are currently allowed - SubAccount is paused %v", subaccountID)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "no transactions are currently allowed - please contact dev team", "reason": "TXN_FAILED"})
		ctx.Abort()
		return
	}

	ctx.Next()
}

func getCurrentSubaccount(ctx *gin.Context) *db.SubaccountTable {
	untypeSubaccount, found := ctx.Get("current-subaccount")
	if !found {
		return nil
	}

	curSubaccount, ok := untypeSubaccount.(*db.SubaccountTable)
	if !ok {
		return nil
	}

	return curSubaccount
}
