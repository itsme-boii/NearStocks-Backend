package controller

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/services/api-server/middleware"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// Task struct represents each task
type Task struct {
	ID          int    `json:"id"`
	Description string `json:"description"`
}

// CategoryTasks represents tasks under each category
type CategoryTasks map[string]Task

// MainData represents all categories and tasks
type MainData map[string]CategoryTasks

var data MainData

type RewardsController struct{}

// todo: subaccount controller to delete
// todo: order controller cleanup
// todo: add auth for market
// todo: remove address controller
func RegisterRewardsController(r *gin.RouterGroup) {
	rewardsController := &RewardsController{}
	if err := LoadConfig(); err != nil {
		xlog.Errorf("Rewards Controller - Failed to load config: %v\n", err)
		return
	}
	rg := r.Group("/tasks")
	// endpoints
	rg.GET("/all", rewardsController.FetchCategoriesTasksHandler)
	rg.GET("/completed", middleware.RequireAuth, rewardsController.GetCompletedTasksHandler)
}

// FetchCategoriesTasksHandler handles the /tasks/all endpoint
func (rc *RewardsController) FetchCategoriesTasksHandler(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, data)
}

func (rc *RewardsController) GetCompletedTasksHandler(ctx *gin.Context) {
	currentSubaccount, err := getCurrentSubaccount(ctx)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "subaccount not found for the api key")
		return
	}
	subaccountIdBytes, err := cutils.SubaccountIdToBytes32(currentSubaccount.ID)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "")
		return
	}
	subAccountID := "0x" + hex.EncodeToString(subaccountIdBytes[:])

	tasks, dailyPoints, err := db.GetCompletedTasksForUser(subAccountID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve tasks"})
		return
	}
	subaccountTrim := strings.TrimPrefix(subAccountID, "0x")
	subaccountVal, err := cutils.HexToSubaccountId(subaccountTrim)
	if err != nil {
		xlog.Errorf("Rewards Controller - Error converting subaccountID: %v", err)
	}
	// check if user traded today, if yes add 1(taskId) to the list
	tradedToday, err2 := (&db.FillDB{}).TradedToday(subaccountVal)
	if err2 != nil {
		xlog.Errorf("Rewards Controller - Error reading from db subaccountID: %v", err)
	}

	totalPoints := dailyPoints

	if tradedToday {
		tasks = append(tasks, 1)
		totalPoints += (len(tasks) - 1)
	} else {
		totalPoints += len(tasks)
	}

	ctx.JSON(http.StatusOK, gin.H{
		"completed_tasks": tasks,
		"totalpoints":     totalPoints,
	})
}

func LoadConfig() error {
	configDataUpdated := `{
		"Daily Trading": {
			"Trade Daily": {
				"id": 1,
				"description": "Execute daily trades with no minimum value requirement."
			}
		},
		"Getting Started": {
			"Get Faucet Tokens": {
				"id": 2,
				"description": "Use LogX Faucet to get collateral tokens to trade"
			},
			"Trade on LogX Network": {
				"id": 3,
				"description": "Place a trade on any market on LogX"
			},
			"Mint Unique Username": {
				"id": 23,
				"description": "Secure your unique identity by minting a username on LogX"
			},
			"Apply referral code": {
				"id": 4,
				"description": "If you were referred by someone, verify this task."
			}
		},
		"$tLOGX": {
			"Claim $tLOGX": {
				"id": 5,
				"description": "Claim your earned $tLOGX tokens."
			},
			"Stake $tLOGX": {
				"id": 6,
				"description": "Stake your $tLOGX tokens to earn rewards."
			},
			"Unstake $tLOGX": {
				"id": 7,
				"description": "Unstake your staked $tLOGX tokens."
			},
			"Withdraw $tLOGX": {
				"id": 8,
				"description": "Withdraw your $tLOGX tokens from LogX network to Arb or Eth Sepolia."
			},
			"Deposit $tLOGX": {
				"id": 9,
				"description": "Deposit $tLOGX tokens back to LogX network to Arb or Eth Sepolia."
			}
		},
		"Trade Perp markets": {
			"Withdraw Buying Power": {
				"id": 10,
				"description": "Withdraw collateral tokens from LogX network to any chain."
			},
			"Deposit Buying Power": {
				"id": 11,
				"description": "Deposit collateral tokens to Logx network from any chain."
			},
			"Open long position": {
				"id": 12,
				"description": "Open a long position for any token market."
			},
			"Open short position": {
				"id": 13,
				"description": "Open a short position for any token market."
			},
			"Fill a limit order": {
				"id": 14,
				"description": "Create and successfully fill a limit order of any token market."
			},
			"Close a long position": {
				"id": 15,
				"description": "Close long position for any token market."
			},
			"Close a short position": {
				"id": 16,
				"description": "Close a short position for any token market."
			}
		},
		"Prediction Market": {
			"Long on prediction market": {
				"id": 17,
				"description": "Open a long position for any prediction market."
			},
			"Short on prediction market": {
				"id": 18,
				"description": "Open a short position for any prediction market."
			},
			"Fill a limit order": {
				"id": 19,
				"description": "Create and successfully fill a limit order of any prediction market."
			},
			"Close long position": {
				"id": 20,
				"description": "Close long position for any prediction market."
			},
			"Close short position": {
				"id": 21,
				"description": "Close a short position for any prediction market."
			}
		},
		"Referrals": {
			"Refer 3 friends to LogX Test Net": {
				"id": 22,
				"description": "Invite friends to join the LogX Test Network and earn rewards."
			}
		}
	}`

	// Unmarshal JSON data into MainData struct
	if err := json.Unmarshal([]byte(configDataUpdated), &data); err != nil {
		return fmt.Errorf("failed to unmarshal config data: %v", err)
	}

	return nil
}
