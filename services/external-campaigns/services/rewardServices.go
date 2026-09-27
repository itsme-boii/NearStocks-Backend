package services

import (
	"context"
	"fmt"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"math/big"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

type RewardServices struct {
	FillDB *db.FillDB
}

type LeaderboardEntry struct {
	Rank        int     `json:"rank"`
	Address     *string `json:"address,omitempty"`
	UserName    *string `json:"userName,omitempty"`
	NetPnl      float64 `json:"netPnl"`
	TotalTrades int     `json:"totalTrades"`
}

type UserRankInfo struct {
	// Rank        int     `json:"rank"`
	TotalTrades int     `json:"totalTrades"`
	NetPnl      float64 `json:"netPnl"`
	// WinRate     float64 `json:"winRate"`
}

type UserStatsInfo struct {
	Address     *string `json:"address"`
	TotalTrades int     `json:"totalTrades"`
	NetPnl      float64 `json:"netPnl"`
	WinRate     float64 `json:"winRate"`
	TotalVolume float64 `json:"totalVolume"`
	Last7DayFee float64 `json:"last7DayFee"`
}

// DEX2: This function uses hardcoded BROKER ID = 1
func (s *RewardServices) GetLeaderboardWithRank(ctx context.Context, address string) ([]LeaderboardEntry, *UserRankInfo, error) {
	brokerId := uint(1)
	startDateStr := os.Getenv("PNL_LEADERBOARD_START_DATE")
	endDateStr := os.Getenv("PNL_LEADERBOARD_END_DATE")

	if startDateStr == "" {
		startDateStr = "2024-01-01 00:00:00"
	}
	if endDateStr == "" {
		endDateStr = "2024-12-31 23:59:59"
	}

	xlog.Infof("PnL Leaderboard : Start Date: %s, End Date: %s", startDateStr, endDateStr)

	// Assuming your database function can accept the startDateStr and endDateStr directly
	results, err := s.FillDB.GetTopUsersByPnlInRange(brokerId, startDateStr, endDateStr)
	if err != nil {
		xlog.Infof("Error getting the top users by Pnl in range: %s and %s", startDateStr, endDateStr)
		return nil, nil, err
	}

	var leaderboard []LeaderboardEntry
	var userRankInfo *UserRankInfo
	subAccountID := cutils.CreateSubaccountId(brokerId, address, 1)
	for i, result := range results {
		netPnl, _, totalTrades := s.calculateNetPnlAndWinRate(result.TotalRealizedPnl.Val, result.LosingTrades, result.WinningTrades, result.ReduceZeroPnlTrades)

		var addressField *string
		if result.UserName == nil {
			truncatedAddress := truncateAddress(result.EthAddress)
			addressField = &truncatedAddress
		}

		entry := LeaderboardEntry{
			Rank:        i + 1,
			Address:     addressField,
			UserName:    result.UserName,
			NetPnl:      netPnl,
			TotalTrades: totalTrades,
		}

		leaderboard = append(leaderboard, entry)
		if strings.EqualFold(result.EthAddress, address) {
			userRankInfo = &UserRankInfo{
				// Rank:        entry.Rank,
				TotalTrades: totalTrades,
				NetPnl:      netPnl,
			}
		}
	}

	if userRankInfo == nil {
		userRankInfo, err = s.calculateUserRank(subAccountID, startDateStr, endDateStr)
		if err != nil {
			xlog.Infof("Error calculating user rank: %s", subAccountID)
			return nil, nil, err
		}
	}

	return leaderboard, userRankInfo, nil
}

// DEX2: Broker ID = 1 is hardcoded
func (s *RewardServices) GetVolumeLeaderboardWithRank(ctx context.Context, address string, marketIDs []uint32) ([]map[string]interface{}, map[string]interface{}, map[string]interface{}, error) {
	brokerId := uint(1)
	// Fetching start and end dates from environment variables
	startDate := os.Getenv("VOLUME_LEADERBOARD_START_DATE")
	if startDate == "" {
		startDate = "2025-06-17 00:00:00"
	}

	startTime, err := time.Parse("2006-01-02 15:04:05", startDate)
	if err != nil {
		xlog.Infof("Error parsing start date: %v", err)
		return nil, nil, nil, err
	}

	endDate := os.Getenv("VOLUME_LEADERBOARD_END_DATE")
	if endDate == "" {
		endDate = "2025-07-16 00:00:00"
	}

	endTime, err := time.Parse("2006-01-02 15:04:05", endDate)
	if err != nil {
		xlog.Infof("Error parsing end date: %v", err)
		return nil, nil, nil, err
	}

	if time.Now().UTC().Before(startTime) {
		xlog.Warnf("Leaderboard has not started yet. Start time: %s, End time: %s", startTime, endTime)
		return nil, nil, nil, nil
	}

	// Fetch raw leaderboard data
	rawLeaderboard, userRankInfo, totalTradeData, err := s.FillDB.GetVolumeAndTraderCount(brokerId, startTime, endTime, 100, fmt.Sprintf("%v_"+address+"_1", brokerId))
	if err != nil {
		xlog.Errorf("Error getting daily leaderboard: %v", err)
		return nil, nil, nil, err
	}

	// Process leaderboard to add rank and replace subaccount_id with username or truncated Ethereum address
	var ethAddresses []string
	for _, entry := range rawLeaderboard {
		subaccountID := entry["subaccount_id"].(string)
		ethAddress, err := cutils.ExtractEthereumAddress(subaccountID)
		if err != nil {
			xlog.Infof("Error extracting Ethereum address from subaccount ID %s: %v", subaccountID, err)
			continue
		}
		ethAddresses = append(ethAddresses, ethAddress)
		entry["eth_address"] = ethAddress // Store eth_address temporarily for username lookup
	}

	// Fetch usernames for Ethereum addresses
	usernames, err := (&db.SubaccountDB{}).GetUserNamesByEthAddresses(ethAddresses)
	if err != nil {
		return nil, nil, nil, err
	}

	// Build final leaderboard with rank and username or truncated address
	var leaderboard []map[string]interface{}
	// Return only the top 10 entries
	for i, entry := range rawLeaderboard {
		if i >= 10 {
			break
		}

		ethAddress := entry["eth_address"].(string)
		username, exists := usernames[ethAddress]
		if !exists || username == "" {
			username = truncateAddress(ethAddress)
		}

		leaderboard = append(leaderboard, map[string]interface{}{
			"rank":         entry["rank"],
			"username":     username,
			"total_volume": entry["total_volume"],
			"total_trades": entry["total_trades"],
		})
	}

	// Refine the userRankInfo based on the specific subaccount information
	specificEthAddress := address
	if specificEthAddress != "" {
		userRankInfo["eth_address"] = specificEthAddress
		if username, exists := usernames[specificEthAddress]; exists && username != "" {
			userRankInfo["username"] = username
		} else {
			userRankInfo["username"] = truncateAddress(specificEthAddress)
		}
	}

	return leaderboard, userRankInfo, totalTradeData, nil
}

func (s *RewardServices) calculateNetPnlAndWinRate(totalRealizedPnl *big.Int, losingTrades, winningTrades, reduceZeroPnlTrades int) (netPnl, winRate float64, totalTrades int) {
	netPnlFloat := new(big.Float).SetInt(totalRealizedPnl)
	netPnlFloat = new(big.Float).Quo(netPnlFloat, big.NewFloat(1e18))
	netPnl, _ = netPnlFloat.Float64()
	totalTrades = winningTrades + losingTrades + reduceZeroPnlTrades
	if totalTrades > 0 {
		winRate = (float64(winningTrades) / float64(totalTrades)) * 100
	} else {
		winRate = 0
	}
	return netPnl, winRate, totalTrades
}

// DEX2: This function uses hardcoded BROKER ID = 1
func (s *RewardServices) calculateUserRank(address string, startDate, endDate string) (*UserRankInfo, error) {
	brokerId := uint(1)
	allResults, err := s.FillDB.GetAllUsersByPnlInRange(brokerId, startDate, endDate)
	if err != nil {
		xlog.Infof("Error getting pnl for all users from db")
		return nil, err
	}

	sort.Slice(allResults, func(i, j int) bool {
		return allResults[i].TotalRealizedPnl.Val.Cmp(allResults[j].TotalRealizedPnl.Val) > 0
	})

	for _, result := range allResults {
		if strings.EqualFold(result.UserAddress, address) {
			netPnl, _, totalTrades := s.calculateNetPnlAndWinRate(result.TotalRealizedPnl.Val, result.LosingTrades, result.WinningTrades, result.ReduceZeroPnlTrades)

			return &UserRankInfo{
				// Rank:        i + 1,
				TotalTrades: totalTrades,
				NetPnl:      netPnl,
				// WinRate:     winRate,
			}, nil
		}
	}

	// lastRank := len(allResults) + 1
	return &UserRankInfo{
		// Rank:        lastRank,
		TotalTrades: 0,
		NetPnl:      0,
		// WinRate:     0,
	}, nil
}

func truncateAddress(address string) string {
	if len(address) <= 10 {
		return address
	}
	return address[:6] + "..." + address[len(address)-4:]
}

func (s *RewardServices) UserStats(address string, startDate, endDate string) (*UserStatsInfo, error) {
	if address == "" {
		return &UserStatsInfo{
			Address:     &address,
			TotalTrades: 0,
			NetPnl:      0,
			WinRate:     0,
			TotalVolume: 0,
			Last7DayFee: 0,
		}, nil
	}

	userStats, err := s.FillDB.GetAllUsersStats(address, startDate, endDate)
	if err != nil {
		xlog.Infof("Error getting pnl for user %s from db: %v", address, err)
		return nil, err
	}

	totalVolumeStr, last7DayFeesStr, err := s.FillDB.GetUserVolumeAndFees(address, 7)
	if err != nil {
		xlog.Infof("Error getting total fee and volume for user %s: %v", address, err)
		return nil, err
	}
	fmt.Printf("Total Volume: %s, Total Fee: %s\n", totalVolumeStr, last7DayFeesStr)
	totalVolume, err := strconv.ParseFloat(totalVolumeStr, 64)
	if err != nil {
		xlog.Infof("Error parsing total volume for user %s: %v", address, err)
		totalVolume = 0
	}

	totalFee, err := strconv.ParseFloat(last7DayFeesStr, 64)
	if err != nil {
		xlog.Infof("Error parsing total fee for user %s: %v", address, err)
		totalFee = 0
	}
	totalVolume = totalVolume / 1e18
	totalFee = totalFee / 1e18

	if userStats.TotalTrades == 0 {
		return &UserStatsInfo{
			Address:     &address,
			TotalTrades: 0,
			NetPnl:      0,
			WinRate:     0,
			TotalVolume: totalVolume,
			Last7DayFee: totalFee,
		}, nil
	}

	var pnlValue *big.Int
	if userStats.TotalRealizedPnl == nil {
		pnlValue = big.NewInt(0)
	} else {
		pnlValue = userStats.TotalRealizedPnl.Val
	}
	netPnl, winRate, totalTrades := s.calculateNetPnlAndWinRate(
		pnlValue,
		userStats.LosingTrades,
		userStats.WinningTrades,
		userStats.ReduceZeroPnlTrades,
	)

	return &UserStatsInfo{
		Address:     &address,
		TotalTrades: totalTrades,
		NetPnl:      netPnl,
		WinRate:     winRate,
		TotalVolume: totalVolume,
		Last7DayFee: totalFee,
	}, nil
}
