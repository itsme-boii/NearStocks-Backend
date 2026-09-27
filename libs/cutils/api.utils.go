package cutils

import (
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"net/http"

	"github.com/gin-gonic/gin"
)

func ApiResponse(ctx *gin.Context, body any, statusCode int, message string) {
	ctx.JSON(statusCode, gin.H{
		"status":  statusCode,
		"body":    body,
		"message": message,
	})
}

func TokenApiResponse(ctx *gin.Context, statusCode int, success bool) {
	ctx.JSON(statusCode, gin.H{
		"success": success,
	})
}

func ApiAbort(ctx *gin.Context, statusCode int, message string, customErrMsg ...string) {
	if len(customErrMsg) > 0 {
		xlog.ErrorfPrev(customErrMsg[0])
	} else {
		xlog.ErrorfPrev(message)
	}

	ctx.AbortWithStatusJSON(statusCode, gin.H{
		"status":  statusCode,
		"message": message,
	})
}

func ApiSuccess(ctx *gin.Context, body any, message string) {
	ctx.JSON(http.StatusOK, gin.H{
		"status":  http.StatusOK,
		"body":    body,
		"message": message,
	})
}

func ApiSuccessWithoutMessage(ctx *gin.Context, body any, statusCode int) {
	ctx.JSON(statusCode, body)
}
