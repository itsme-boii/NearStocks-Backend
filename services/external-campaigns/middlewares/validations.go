package middlewares

import (
	"net/http"
	"services/external-campaigns/dtos"
	"strconv"
	"github.com/gin-gonic/gin"
)

func ValidateDiscordMessageRequestBody() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var requestBody struct {
			Message string `json:"message" binding:"required"`
		}
		if err := ctx.ShouldBindJSON(&requestBody); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"message": "Request body must contain a valid 'message' field",
			})
			ctx.Abort()
			return
		}

		// Validate message length
		if len(requestBody.Message) > 200 {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"message": "'message' must not exceed 200 characters",
			})
			ctx.Abort()
			return
		}

		// Store the validated request body in the context for use in the handler
		ctx.Set("validatedMessage", requestBody.Message)
		ctx.Next()
	}
}

// ValidateAddressQueryParam middleware ensures the "address" query parameter is present.
func ValidateAddressQueryParam() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		address := ctx.Query("address")
		if address == "" {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"message": "address query parameter is required",
			})
			ctx.Abort()
			return
		}
		ctx.Next()
	}
}
// ValidateTimeRangeQueryParams middleware ensures "start" and "end" query parameters are present and valid Unix timestamps (in seconds or milliseconds),
// and "end" is greater than "start", and the time range is not more than 48 hours.
func ValidateTimeRangeQueryParams() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		start := ctx.Query("start")
		end := ctx.Query("end")

		// Check if both start and end are present
		if start == "" || end == "" {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"message": "Both 'start' and 'end' query parameters are required",
			})
			ctx.Abort()
			return
		}

		// Convert start and end to integers (Unix timestamps)
		startTime, err := strconv.ParseInt(start, 10, 64)
		if err != nil || startTime <= 0 {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"message": "'start' must be a valid Unix timestamp",
			})
			ctx.Abort()
			return
		}

		endTime, err := strconv.ParseInt(end, 10, 64)
		if err != nil || endTime <= 0 {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"message": "'end' must be a valid Unix timestamp",
			})
			ctx.Abort()
			return
		}
		// Check if timestamps are in milliseconds (13 digits) and convert to seconds if needed
		if len(start) == 13 {
			startTime /= 1000 // Convert milliseconds to seconds
		}
		if len(end) == 13 {
			endTime /= 1000 // Convert milliseconds to seconds
		}

		// Ensure end is greater than start
		if endTime <= startTime {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"message": "'end' must be greater than 'start'",
			})
			ctx.Abort()
			return
		}

		// Ensure the time range does not exceed 48 hours
		const maxDurationSeconds = 172800 // 48 hours in seconds
		if endTime-startTime > maxDurationSeconds {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"message": "The time range between 'start' and 'end' cannot exceed 48 hours",
			})
			ctx.Abort()
			return
		}
		ctx.Next()
	}
}


// ValidateQuestClaimDto validates the JSON body for ZealyQuestClaimDto
func ValidateZealyClaimDto() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var reqBody dtos.ZealyQuestClaimDto
		if err := ctx.ShouldBindJSON(&reqBody); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"message":   "Invalid request body: " + err.Error(),
				"requestId": reqBody.RequestId,
			})
			ctx.Abort()
			return
		}
		// Ensure that Accounts and Wallet are present
		if reqBody.Accounts == nil || reqBody.Accounts.Wallet == "" {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"message":   "Missing accounts.wallet field",
				"requestId": reqBody.RequestId,
			})
			return
		}
		ctx.Set("reqBody", reqBody)
		ctx.Next()
	}
}

// ValidateUserContactDto validates the JSON body for InteractDto
func ValidateInteractContactDto() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var reqBody dtos.InteractDto
		if err := ctx.ShouldBindJSON(&reqBody); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"message": "Invalid request body: " + err.Error(),
			})
			ctx.Abort()
			return
		}
		ctx.Set("reqBody", reqBody)
		ctx.Next()
	}
}