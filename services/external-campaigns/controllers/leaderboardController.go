package controllers

import (
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"net/http"
	"services/external-campaigns/services"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type LeaderboardController struct {
	RewardService     *services.RewardServices
	referralHistoryDB db.ReferralHistoryDB
}

func NewLeaderboardController(rewardService *services.RewardServices) *LeaderboardController {
	return &LeaderboardController{
		RewardService: rewardService,
	}
}

func (c *LeaderboardController) GetLeaderboard(ctx *gin.Context) {
	address := ctx.Query("address")
	if address == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Address parameter is required"})
		return
	}

	leaderboard, userRankInfo, err := c.RewardService.GetLeaderboardWithRank(ctx, address)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := gin.H{
		"leaderboard": leaderboard,
		"user":        userRankInfo,
	}

	ctx.JSON(http.StatusOK, response)
}

func (c *LeaderboardController) GetVolumeLeaderboard(ctx *gin.Context) {
	address := ctx.Query("address")
	if address == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Address parameter is required"})
		return
	}

	// Use predefined list of meme token market IDs from constantsutils
	marketIDs := contractUtils.ALL_MEME_PERPS

	// Fetch the daily meme token leaderboard and user rank information
	leaderboard, userRankInfo, totalTradeData, err := c.RewardService.GetVolumeLeaderboardWithRank(ctx, address, marketIDs)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := gin.H{
		"leaderboard": leaderboard,
		"user":        userRankInfo,
		"all":         totalTradeData,
	}

	ctx.JSON(http.StatusOK, response)
}

// DEX2: Broker ID = 1 is hardcoded
func (c *LeaderboardController) GetUserStats(ctx *gin.Context) {
	address := ctx.Query("address")
	if address == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Address parameter is required"})
		return
	}
	// Determine brokerId (default to 1 if not provided or invalid)
	brokerIdStr := ctx.Query("brokerId")
	var brokerId uint = 1
	if brokerIdStr != "" {
		if parsed, err := strconv.ParseUint(brokerIdStr, 10, 32); err == nil && parsed != 0 {
			brokerId = uint(parsed)
		}
	}
	startDateStr := "2024-01-01 00:00:00"
	subAccountID := cutils.CreateSubaccountId(brokerId, address, 1)
	endDateStr := time.Now().Format("2006-01-02 15:04:05")
	userRankInfo, err := c.RewardService.UserStats(subAccountID, startDateStr, endDateStr)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := gin.H{
		"user": userRankInfo,
	}

	ctx.JSON(http.StatusOK, response)
}
func (c *LeaderboardController) GetReferralStats(ctx *gin.Context) {
	referrerUserID := ctx.Query("referrer_user_id")

	if referrerUserID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Referrer user ID is required"})
		return
	}

	stats, _, err := c.referralHistoryDB.CalculateReferralStats(referrerUserID)
	if err != nil {
		xlog.Errorf("Failed to calculate referral stats for user %s: %v", referrerUserID, err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate referral stats"})
		return
	}

	netVolume, err := strconv.ParseFloat(stats.NetVolume.String(), 64)
	if err != nil {
		xlog.Errorf("Failed to parse net volume for user %s: %v", referrerUserID, err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process referral stats"})
		return
	}

	totalFees, err := strconv.ParseFloat(stats.TotalRefereeFeesOverall.String(), 64)
	if err != nil {
		xlog.Errorf("Failed to parse total fees for user %s: %v", referrerUserID, err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process referral stats"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"net_volume":              netVolume / 1e18,
		"total_distinct_referees": stats.TotalDistinctReferees,
		"total_fees":              totalFees / 1e18,
	})
}
