package controller

import (
	"fmt"
	"github/eugenix-io/logx-inf-backend/contract"
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/logxTokenUtils"
	"github/eugenix-io/logx-inf-backend/libs/metric"
	"github/eugenix-io/logx-inf-backend/libs/transaction"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"github/eugenix-io/logx-inf-backend/services/api-server/middleware"
	"github/eugenix-io/logx-inf-backend/services/api-server/services"
	balanceTypes "github/eugenix-io/logx-inf-backend/services/balance-server/types"
	"github/eugenix-io/logx-inf-backend/xclient"
	"math/big"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/sha3"
)

type TokenController struct {
	LogxRpcUrl        string
	balanceClient     *xclient.BalanceClient
	stakingService    *services.StakingService
	accountService    *services.AccountService
	batchDB           *db.BatchDB
	stakingDB         *db.StakingDB
	depositWithdrawDB *db.DepositWithdrawDB
	LogxTokenUtils    *logxTokenUtils.LogxTokenUtils
	// referralRewardDB  *db.ReferralRewardDB
}

func RegisterTokenController(
	r *gin.RouterGroup,
) {
	tokenController := TokenController{
		LogxRpcUrl:        os.Getenv("RPC_URL"),
		balanceClient:     xclient.GlobalBalanceClient,
		stakingService:    services.NewStakingService(),
		accountService:    services.NewAccountService(),
		LogxTokenUtils:    logxTokenUtils.NewLogxTokenUtils(),
		batchDB:           &db.BatchDB{},
		stakingDB:         &db.StakingDB{},
		depositWithdrawDB: &db.DepositWithdrawDB{},
	}
	rg := r.Group("/token")

	//Endpoints
	rg.POST("/claim", middleware.RequireAuth, middleware.RequireSequencer, middleware.RequireSessionKey, tokenController.ClaimLogxTokens)
	rg.POST("/stake", middleware.RequireAuth, middleware.RequireSequencer, middleware.RequireSessionKey, tokenController.StakeTokens)
	rg.POST("/unstake", middleware.RequireAuth, middleware.RequireSequencer, middleware.RequireSessionKey, tokenController.UnstakeTokens)
	rg.POST("/withdrawLogX", middleware.RequireAuth, middleware.RequireSequencer, middleware.RequireSessionKey, tokenController.WithdrawLogX)
	rg.POST("/claimrewards", middleware.RequireAuth, middleware.RequireSequencer, middleware.RequireSessionKey, tokenController.ClaimRewards)
	rg.POST("/withdrawCollateral", middleware.RequireAuth, middleware.RequireSequencer, middleware.RequireSessionKey, tokenController.WithdrawCollateral)
	rg.POST("/settle-ostrich-pnl", middleware.RequireAuth, middleware.RequireSequencer, middleware.RequireSessionKey, tokenController.SettleBroker2PnL)
	rg.POST("/shiftBalanceKroma", middleware.RequireSequencer, tokenController.ShiftBalanceKroma)
	rg.POST("/burnBalanceKroma", middleware.RequireSequencer, tokenController.BurnBalanceKroma)
	rg.POST("/finalisePendingDeposits", tokenController.FinalisePendingDeposits)
	// rg.POST("/claimReferralRewards", middleware.RequireAuth, middleware.RequireSequencer, middleware.RequireSessionKey, tokenController.ClaimReferralRewards)
	// rg.POST("/claimLogXFeeRewards", middleware.RequireAuth, middleware.RequireSequencer, middleware.RequireSessionKey, tokenController.ClaimLogXFeeRewards)
	// TODO: AUDIT THIS AND THEN ENABLE
	// rg.POST("/claimArbReward", middleware.RequireAuth, tokenController.ClaimArbRewards)

}

// Fetch all pending deposits for finalisation.
// Mark as finalized first, then update balance. If balance update fails, mark as failed.
func (t *TokenController) FinalisePendingDeposits(ctx *gin.Context) {
	fetchTimeLimit := time.Now()
	limit := 1_000
	lastFetchId := uint(0)
	totalFound := uint(0)
	failCount := uint(0)

	// Use Redis lock to prevent concurrent finalization
	xredis.WithRedisLock(xredis.GetFinaliseDepositKey(), func() (*xredis.NOOP, error) {
		for i := 0; i < 10; i++ {
			pendingDeposits := (&db.DepositWithdrawDB{}).GetAllUnfinalisedDeposits(lastFetchId, limit)
			if len(*pendingDeposits) == 0 {
				break // No more deposits
			}

			// Process each deposit
			for _, deposit := range *pendingDeposits {
				// Step 0: Increment transaction counter
				transactionCounter, err := transaction.IncrementCounter(1)
				if err != nil {
					xlog.Errorf("Failed to increment transaction counter for subaccount: %v | message id: %v| err %v", deposit.SubaccountId, deposit.MessageId, err)
					continue
				}

				totalFound++
				lastFetchId = deposit.ID
				// Step 1: Mark as finalized in database
				// First finalize the deposit so that if the balance update doesn't happen due to db delay etc.
				if err := (&db.DepositWithdrawDB{}).FinaliseDeposit(deposit.ID); err != nil {
					msg := fmt.Sprintf("Error marking deposit as finalised for subaccount %v | index: %v | error: %v", deposit.SubaccountId, deposit.ID, err)
					xlog.Errorf(msg)
					xclient.GetGlobalDiscordClient().SendWebhookMessage(msg)
					failCount++
					continue
				}

				tokenAmountBigInt := new(big.Int)
				tokenAmountBigInt, success := tokenAmountBigInt.SetString(deposit.Amount, 10)
				if !success {
					xlog.Errorf("Error converting token amount to big.Int for subaccount: %v | messageId: %v", deposit.SubaccountId, deposit.MessageId)
					continue
				}

				// Step 2: Update batch db

				err = contract.GlobalContracts.EndpointContract.FinalisePendingDeposits(contract.FinaliseDepositRequest{
					SubaccountId:  deposit.SubaccountId,
					Amount:        tokenAmountBigInt,
					ProductId:     deposit.ProductId,
					SourceChainId: big.NewInt(int64(deposit.SourceChainID)),
				}, transactionCounter)

				if err != nil {
					xlog.Errorf("Error finalising deposit for subaccount %v | message id: %v | err: %v", deposit.SubaccountId, deposit.MessageId, err)
					continue
				}

				// Step 3: Update user balance
				err = t.balanceClient.UpdateMultiTokenBalance(&balanceTypes.MultiTokenBalanceRequest{
					SubaccountID:  deposit.SubaccountId,
					ProductIds:    []uint32{deposit.ProductId},
					TokenBalances: []string{deposit.Amount},
				})

				if err != nil {
					// Balance update failed - mark deposit as failed
					xlog.Errorf("Error updating token balance for subaccount %v | deposit ID: %v | err: %v", deposit.SubaccountId, deposit.ID, err)
					failCount++

					// Attempt to mark finalised false again
					if err := (&db.DepositWithdrawDB{}).MarkDepositFailed(deposit.ID); err != nil {
						msg := fmt.Sprintf("Error marking deposit as unfinalised for subaccount %v | index: %v | error: %v", deposit.SubaccountId, deposit.ID, err)
						xlog.Errorf(msg)
						xclient.GlobalDiscordClient.SendWebhookMessage(msg)
					}
					continue
				}

				if deposit.CreatedAt.After(fetchTimeLimit) {
					break
				}
			}
		}
		return nil, nil
	})

	// Send alert if there were failures
	if failCount != 0 {
		xclient.GlobalDiscordClient.SendWebhookMessage(fmt.Sprintf("Error finalising deposits: %v", failCount))
	}

	cutils.ApiSuccess(ctx, gin.H{
		"totalFound": totalFound,
		"failCount":  failCount,
	}, "Finalised all pending deposits")
}

const STAKE_EXPIRY_DURATION = 2 * cutils.MINUTE_MILLI

type Response struct {
	status int
	data   any
}

// AUDIT comments - @mananbordia
// 1. Can be stopped via ENV variable: TODO: Move this to redis key
// 2. Lowercase SubaccountHex is being used
// 3. Special check on if subaccount is paused for claiming - TODO: Check that investors are not paused
// 4. Request chain Id is checked
// 5. Signature and signer address are fetched from headers
// 6. Token amount has positive check as well as exact expected value check
func (t *TokenController) ClaimLogxTokens(ctx *gin.Context) {
	var req contractUtils.DepositRequest
	defer cutils.LogTime(time.Now(), metric.CLAIM_LOGX_TXN)

	if os.Getenv("IS_CLAIM_ALLOWED") == "0" {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Claim is not allowed currently"})
		return
	}

	// Bind request JSON and validate
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid request: unable to bind JSON"})
		return
	}

	// Verify chain ID
	if req.ChainId != contractUtils.EndpointChainId() {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid chain ID"})
		return
	}

	// Convert subaccount ID to hex
	subaccountHex, err := cutils.HackySubaccountIdToHex(req.SubAccountId)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid subaccount ID"})
		return
	}

	// Convert subaccountHex to lowercase
	subaccountHexLower := strings.ToLower(subaccountHex)

	// The claim is for the authenticated subaccount only, and the allocation is looked up by that
	// subaccount's own address: never by a request header (the session-key middleware validates
	// Broker-User-Address, so trusting Logx-User-Address let a caller claim someone else's airdrop).
	currentSubaccount, errSub := getCurrentSubaccount(ctx)
	if errSub != nil {
		ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "subaccount not found"})
		return
	}
	currentHex, errHex := cutils.SubaccountIdToHex(currentSubaccount.ID)
	if errHex != nil || !strings.EqualFold(currentHex, subaccountHex) {
		ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "subaccount does not match credentials"})
		return
	}
	claimantAddress := currentSubaccount.EthAddress

	if cutils.IsSubAccountClaimLogXPaused(subaccountHexLower) {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error": "This account is paused for claim. Please get connected with the team.",
		})
		return
	}

	// Get signature and signer address from headers
	signature := ctx.GetHeader("Logx-Signature")
	signerAddress := ctx.GetHeader("Logx-Signer-Address")

	// Convert token amount from string to big.Int
	tokenAmountBigInt := new(big.Int)
	tokenAmountBigInt, success := tokenAmountBigInt.SetString(req.TokenAmount, 10)
	if !success {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Unable to convert token amount to big.Int"})
		return
	}

	if tokenAmountBigInt.Sign() <= 0 {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Token amount must be greater than zero"})
		return
	}

	// Verify deposit request using signature
	if errVerf := contractUtils.VerifyDepositRequest(req, signature, signerAddress); errVerf != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Signature verification failed: %v", errVerf)})
		return
	}

	// Increment transaction counter
	transactionCounter, err := transaction.IncrementCounter(1)
	if err != nil {
		xlog.Errorf("Failed to increment transaction counter: %v", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to increment transaction counter"})
		return
	}

	// Perform the action within a nonce lock
	response, err := services.WithNonceRedisLock(subaccountHex, func(currentNonce string) (*Response, error) {
		// Validate nonce
		nonceCheck := cutils.CheckNonce(req.Nonce, currentNonce)
		if !nonceCheck {
			return &Response{status: http.StatusBadRequest, data: "Nonce check failed"}, fmt.Errorf("nonce check failed for subAccount: %v", req.SubAccountId)
		}
		// NEAR settlement: the claim is paid from the DAO-funded rewards pool (never minted)
		refund, errPool := services.ReserveLogxRewards(tokenAmountBigInt)
		if errPool != nil {
			return &Response{status: http.StatusServiceUnavailable, data: "LogX claims are temporarily unavailable"}, errPool
		}
		claimed := false
		defer func() {
			if !claimed {
				refund()
			}
		}()

		// Logging the claim request
		xlog.Infof("Processing claimTokens request for subAccount: %v for amount: %v", req.SubAccountId, req.TokenAmount)

		userAddress := claimantAddress

		// Update claimed amount in token user data
		success, hasVesting, updateErr := t.LogxTokenUtils.UpdateTokenUserDataByUserAddress(userAddress, ctypes.NewBigInt(tokenAmountBigInt))
		if updateErr != nil {
			xlog.Errorf("Error updating token user data: %v, userAddress: %s, amount: %s", updateErr, userAddress, req.TokenAmount)
			cutils.PauseSubaccountClaimLogXFlow(subaccountHexLower)
			xclient.GlobalDiscordClient.SendWebhookMessage(fmt.Sprintf("******************* ALERT >>>>>>>>>>>>>>> Error in updating token user data | User Address: %v | Amount: %v <<<<<<<<<<<<< ALERT *******************", userAddress, req.TokenAmount))
			return &Response{status: http.StatusInternalServerError, data: "Failed to update claimed tokens in user data"}, updateErr
		}

		if !success {
			xlog.Errorf("Claim amount mismatch: userAddress: %s, amount: %s", userAddress, req.TokenAmount)
			cutils.PauseSubaccountClaimLogXFlow(subaccountHexLower)
			return &Response{status: http.StatusBadRequest, data: "Claim amount mismatch"}, nil
		}

		xlog.Infof("Claim Logx - Updated token db for subAccount: %v", req.SubAccountId)
		// Update the token balance on the balance server
		status, err := t.balanceClient.UpdateTokenBalance(subaccountHex, 0, req.TokenAmount)
		if err != nil {
			xlog.Errorf("Error updating token balance: %v, subAccountId: %s, userAddress: %s, tokenAmount: %s", err, subaccountHex, userAddress, req.TokenAmount)
			xclient.GlobalDiscordClient.SendWebhookMessage(fmt.Sprintf("******************* ALERT >>>>>>>>>>>>>>> Error in updating token balance | Sub Account Id: %v | Amount: %v <<<<<<<<<<<<< ALERT *******************", subaccountHex, req.TokenAmount))
			return &Response{status: http.StatusInternalServerError, data: "Failed to update token balance"}, err
		}

		// AUDIT: @mananbordia - this won't executed because error with be returned in the above line
		if !status {
			xlog.Errorf("Failed to update token balance on balance server: subAccountId: %s, userAddress: %s, tokenAmount: %s", subaccountHex, userAddress, req.TokenAmount)
			return &Response{status: http.StatusAccepted, data: "something went wrong while updating token balance for subAccount"}, fmt.Errorf("something went wrong while updating token balance in claim logx flow for subAccount: %v", req.SubAccountId)
		}

		xlog.Infof(" Claim Logx - Updated balance service for subAccount: %v", req.SubAccountId)

		// Increment nonce
		err1 := services.IncrementNonce(subaccountHex, currentNonce)
		if err1 != nil {
			xlog.Errorf("Failed to increment nonce: %v, subAccountId: %s, userAddress: %s", err1, subaccountHex, userAddress)
			return &Response{status: http.StatusInternalServerError, data: "Failed to increment nonce"}, err1
		}

		// Call the contract's ClaimLogX function
		status, err2 := contract.GlobalContracts.EndpointContract.ClaimLogX(req, tokenAmountBigInt, signature, transactionCounter)
		if !status || err2 != nil {
			xlog.Errorf("Contract call failed: %v, subAccountId: %s, userAddress: %s", err2, subaccountHex, userAddress)
			return &Response{status: http.StatusBadRequest, data: "Claim tokens error: contract call failed"}, err2
		}
		// Success
		xlog.Infof("Successfully processed token claim for userAddress: %s, subAccountId: %s", userAddress, subaccountHex)

		if !hasVesting {
			cutils.PauseSubaccountClaimLogXFlow(subaccountHexLower)
			xlog.Infof("Successfully paused token claim for userAddress: %s, subAccountId: %s", userAddress, subaccountHex)
		}
		claimed = true
		return &Response{status: http.StatusOK, data: "Claimed tokens successfully"}, nil
	})

	// Handle any errors that occurred within the nonce lock
	if err != nil {
		xlog.Errorf("Error in claim tokens: %v", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	// Return the response
	ctx.JSON(response.status, response.data)
}

// func (t *TokenController) ClaimReferralRewards(ctx *gin.Context) {
// 	var req contractUtils.CampaignRewardClaimRequest
// 	defer cutils.LogTime(time.Now(), metric.CLAIM_CAMPAIGN_REWARDS)

// 	if os.Getenv("IS_CLAIM_REFERRAL_ALLOWED") == "0" {
// 		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Claim Referral is not allowed currently"})
// 		return
// 	}

// 	// Bind request JSON and validate
// 	if err := ctx.ShouldBindJSON(&req); err != nil {
// 		xlog.Errorf("Failed to bind request JSON for campaign rewards claim: %v", err)
// 		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid request: unable to bind JSON"})
// 		return
// 	}

// 	if req.ProductId != contractUtils.ARB_USDC {
// 		xlog.Errorf("Invalid product ID: %v for campaign referral rewards claim", req.ProductId)
// 		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
// 		return
// 	}

// 	// Convert subaccount ID to hex
// 	subaccountHex, err := cutils.HackySubaccountIdToHex(req.SubAccountId)
// 	if err != nil {
// 		xlog.Errorf("Claim Referral Invalid subaccount ID: %s, error: %v", req.SubAccountId, err)
// 		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid subaccount ID"})
// 		return
// 	}

// 	// Verify chain ID
// 	if req.ChainId != contractUtils.EndpointChainId() {
// 		xlog.Warnf("Claim Referral Invalid chain ID: %d for subAccount: %s", req.ChainId, req.SubAccountId)
// 		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid chain ID"})
// 		return
// 	}

// 	// Check if the campaign ID matches REFERRAL_CAMPAIGN_ID
// 	if req.CampaignId != contractUtils.REFERRAL_CAMPAIGN_ID {
// 		xlog.Errorf("Claim Referral Invalid CampaignId: %d, expected: %d for subAccount: %s", req.CampaignId, contractUtils.REFERRAL_CAMPAIGN_ID, req.SubAccountId)
// 		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid Campaign ID"})
// 		return
// 	}

// 	// Get signature and signer address from headers
// 	signature := ctx.GetHeader("Logx-Signature")
// 	signerAddress := ctx.GetHeader("Logx-Signer-Address")
// 	if signature == "" || signerAddress == "" {
// 		xlog.Warnf("Claim Referral Missing signature or signer address in headers for subAccount: %s", req.SubAccountId)
// 		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Missing signature or signer address"})
// 		return
// 	}

// 	// Convert token amount from string to big.Int
// 	rewardAmountBigIntVal := new(big.Int)
// 	if _, success := rewardAmountBigIntVal.SetString(req.Amount, 10); !success {
// 		xlog.Errorf("Claim Referral Failed to convert token amount: %s to big.Int for subAccount: %s", req.Amount, req.SubAccountId)
// 		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Unable to convert token amount to big.Int"})
// 		return
// 	}

// 	if rewardAmountBigIntVal.Cmp(big.NewInt(0)) <= 0 {
// 		xlog.Errorf("Claim Logx Referral Rewards Failed: token amount is not greater than zero for subAccount: %s", req.SubAccountId)
// 		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Token amount must be greater than zero"})
// 		return
// 	}

// 	// Fetch the available USDC balance for the subaccount
// 	funds, err := t.balanceClient.GetSubAccountContractSpotBalance(contractUtils.CAMPAIGN_REWARD_SUBACCOUNT_ID, int(req.ProductId))
// 	if err != nil {
// 		xlog.Errorf("Claim Referral Error fetching funds for contract %s: %v", contractUtils.CAMPAIGN_REWARD_SUBACCOUNT_ID, err)
// 		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Error fetching contract balance."})
// 		return
// 	}

// 	// Compare the requested amount with available funds in contract
// 	if rewardAmountBigIntVal.Cmp(funds) > 0 {
// 		xlog.Errorf("Claim Referral Requested amount %s exceeds available funds %s for subaccount: %s. Contact development team.", rewardAmountBigIntVal.String(), funds.String(), contractUtils.CAMPAIGN_REWARD_SUBACCOUNT_ID)
// 		xclient.GlobalDiscordClient.SendWebhookMessage(fmt.Sprintf("******************* ALERT >>>>>>>>>>>>>>> Add funds in Claim Camapign Reward Contract | Product Id: %v <<<<<<<<<<<<< ALERT *******************", req.ProductId))
// 		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Insufficient funds, please contact the development team."})
// 		return
// 	}

// 	// Verify referral request using signature
// 	if errVerf := contractUtils.VerifyCampaignRewards(req, signature, signerAddress); errVerf != nil {
// 		xlog.Warnf("Claim Referral Signature verification failed for subAccount: %s, error: %v", req.SubAccountId, errVerf)
// 		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Signature verification failed: %v", errVerf)})
// 		return
// 	}

// 	// Increment transaction counter
// 	transactionCounter, err := transaction.IncrementCounter(1)
// 	if err != nil {
// 		xlog.Errorf("Claim Referral Failed to increment transaction counter for subAccount: %s, error: %v", req.SubAccountId, err)
// 		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to increment transaction counter"})
// 		return
// 	}

// 	// Perform the action within a nonce lock
// 	response, err := services.WithNonceRedisLock(subaccountHex, func(currentNonce string) (*Response, error) {
// 		// Validate nonce
// 		nonceCheck := cutils.CheckNonce(req.Nonce, currentNonce)
// 		if !nonceCheck {
// 			xlog.Warnf("Claim Referral Nonce check failed for subAccount: %s, provided nonce: %d, currentNonce: %s", req.SubAccountId, req.Nonce, currentNonce)
// 			return &Response{status: http.StatusBadRequest, data: "Nonce check failed"}, nil
// 		}

// 		// Logging the claim request
// 		xlog.Infof("Processing ClaimReferralRewards for subAccount: %s, amount: %s", req.SubAccountId, req.Amount)

// 		// Get user address from the headers
// 		userAddress := ctx.GetHeader("Logx-User-Address")
// 		if userAddress == "" {
// 			xlog.Warnf("Claim Referral Missing user address in headers for subAccount: %s", req.SubAccountId)
// 			return &Response{status: http.StatusBadRequest, data: "Missing user address"}, nil
// 		}

// 		rewardEntry, err := t.referralRewardDB.GetReferralReward(userAddress, uint(req.ProductId))
// 		if err != nil {
// 			xlog.Errorf("Claim Referral Reward entry not found for userAddress: %s, subAccount: %s, error: %v", userAddress, req.SubAccountId, err)
// 			return &Response{status: http.StatusNotFound, data: "Reward entry not found"}, err
// 		}

// 		// Check if the claimed amount does not exceed the remaining reward
// 		remainingAmount := rewardEntry.RewardAmount.Sub(rewardEntry.ClaimedAmount)
// 		if rewardAmountBigIntVal.Cmp(remainingAmount.Val) > 0 {
// 			xlog.Warnf("Claim amount exceeds remaining reward for userAddress: %s, subAccount: %s, claimedAmount: %s, remainingAmount: %s", userAddress, req.SubAccountId, rewardAmountBigIntVal.String(), remainingAmount.String())
// 			return &Response{status: http.StatusBadRequest, data: "Claim amount exceeds available reward"}, nil
// 		}

// 		// Update the claimed amount
// 		rewardAmountBigInt := ctypes.NewBigInt(rewardAmountBigIntVal)
// 		_, updateErr := t.referralRewardDB.UpdateReferralClaimAmount(userAddress, uint(req.ProductId), rewardAmountBigInt)
// 		if updateErr != nil {
// 			xlog.Errorf("Failed to update referral claim amount for userAddress: %s, subAccount: %s, error: %v", userAddress, req.SubAccountId, updateErr)
// 			return &Response{status: http.StatusInternalServerError, data: "Failed to update claimed tokens in user data"}, nil
// 		}

// 		// Update the token balance on the balance server
// 		status, err := t.balanceClient.UpdateTokenBalance(subaccountHex, req.ProductId, req.Amount)
// 		if err != nil || !status {
// 			xlog.Errorf("Claim Referral Failed to update token balance on balance server for subAccount: %s, userAddress: %s, amount: %s, error: %v", subaccountHex, userAddress, req.Amount, err)
// 			negativeRewardAmountBigInt := rewardAmountBigInt.Copy()
// 			negativeRewardAmountBigInt.Val.Neg(negativeRewardAmountBigInt.Val)

// 			// Revert the previous update by adding the negative amount
// 			_, updateErr := t.referralRewardDB.UpdateReferralClaimAmount(userAddress, uint(req.ProductId), negativeRewardAmountBigInt)
// 			if updateErr != nil {
// 				xlog.Errorf("Failed to revert referral claim amount for userAddress: %s, subAccount: %s, error: %v", userAddress, req.SubAccountId, updateErr)
// 				return &Response{status: http.StatusInternalServerError, data: "Failed to revert claimed tokens in user data"}, nil
// 			}
// 			return &Response{status: http.StatusInternalServerError, data: "Failed to update token balance"}, err
// 		}

// 		xlog.Infof("Claim Referral Updated balance service for subAccount: %s, userAddress: %s", req.SubAccountId, userAddress)

// 		// Increment nonce
// 		if err := services.IncrementNonce(subaccountHex, currentNonce); err != nil {
// 			xlog.Errorf("Claim Referral Failed to increment nonce for subAccount: %s, userAddress: %s, error: %v", subaccountHex, userAddress, err)
// 			return &Response{status: http.StatusInternalServerError, data: "Failed to increment nonce"}, err
// 		}

// 		// Call the contract's ClaimCampaignRewards function
// 		status, err2 := contract.GlobalContracts.EndpointContract.ClaimCampaignRewards(req, rewardAmountBigIntVal, signature, transactionCounter)
// 		if err2 != nil || !status {
// 			xlog.Errorf("Claim Referral Failed to call contract for ClaimCampaignRewards for subAccount: %s, userAddress: %s, error: %v", subaccountHex, userAddress, err2)
// 			return &Response{status: http.StatusBadRequest, data: "Contract call failed"}, err2
// 		}

// 		// Success
// 		xlog.Infof("Successfully processed ClaimCampaignRewards for userAddress: %s, subAccountId: %s", userAddress, subaccountHex)

// 		return &Response{status: http.StatusOK, data: "Claimed Referral Amount successfully"}, nil
// 	})

// 	// Handle any errors that occurred within the nonce lock
// 	if err != nil {
// 		xlog.Errorf("Error processing ClaimReferralRewards: %v", err)
// 		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
// 		return
// 	}

// 	// Return the response
// 	ctx.JSON(response.status, response.data)
// }

func (t *TokenController) isValidStakeTokens(req contractUtils.StakeLogXRequest) bool {
	subaccountID, err := cutils.SubaccountIdToHex(req.SubAccountId)

	if err != nil {
		xlog.Errorf("Failed to convert SubaccountId: %v", req.SubAccountId)
		return false
	}

	// Fetch spot balances
	var spotBalances []ctypes.SpotBalance
	spotBalances, _, _, err = t.balanceClient.GetSpotBalance(subaccountID)
	if err != nil {
		xlog.Errorf("Error fetching spot balances for subaccount %v: %v", subaccountID, err)
		return false
	}

	// Initialize total unstake amount within the last hour
	stakingDB := &db.StakingDB{}
	unstakeSum, err := stakingDB.GetUnstakeSumOfPendingUnstake(req.SubAccountId)
	if err != nil {
		xlog.Errorf("Error fetching unstake sum for subaccount %v: %v", req.SubAccountId, err)
		return false
	}

	// Iterate through spot balances and adjust based on unstake amounts
	var tokenAmountStr string
	for _, spotBalance := range spotBalances {
		if spotBalance.ProductId == 0 {
			// Convert token balance to big.Int
			tokenBalanceInt := new(big.Int)
			if _, ok := tokenBalanceInt.SetString(spotBalance.TokenBalance, 10); !ok {
				xlog.Errorf("Invalid token balance for subaccount %v", subaccountID)
				return false
			}

			// Subtract unstake sum from available balance
			tokenBalanceInt.Sub(tokenBalanceInt, unstakeSum)

			// Store the adjusted token balance as a string
			tokenAmountStr = tokenBalanceInt.String()
		}
	}

	// Convert the adjusted token balance and request amount to big.Int
	tokenAmountInt, _ := new(big.Int).SetString(tokenAmountStr, 10)
	reqAmountInt, _ := new(big.Int).SetString(req.TokenAmount, 10)

	// Check if the requested amount is less than or equal to the available adjusted balance
	return reqAmountInt.Cmp(tokenAmountInt) <= 0
}

// Validate request payload
// Signature verification
// Check for nonce
// Logic to stake tokens
// 1. Check LOGX balance
// 2. Use stLogX balance to know the amount of tokens staked
// 3. Add entry in stakes table
// 4. Atomically update the user balance - If fails delete the entry from stakes table
func (t *TokenController) StakeTokens(ctx *gin.Context) {
	var req contractUtils.StakeLogXRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "stake tokens error: invalid request"})
		return
	}

	tokenAmountBigInt := new(big.Int)
	tokenAmountBigInt, success1 := tokenAmountBigInt.SetString(req.TokenAmount, 10)
	if !success1 {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "stake tokens error: unable to convert token amount string to big.Int"})
		return
	}
	if tokenAmountBigInt.Cmp(big.NewInt(0)) <= 0 {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "stake tokens error: token amount must be greater than zero"})
		return
	}

	if req.ChainId != contractUtils.EndpointChainId() {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "stake tokens error: invalid chainId"})
		return
	}
	if req.StakerContract != contractUtils.STAKER_CONTRACT_ADDRESS {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "stake tokens error: invalid staker contract address"})
		return
	}
	if req.ProductId != contractUtils.ST_LOGX {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "stake tokens error: invalid productId"})
		return
	}

	subaccountHex, err := cutils.HackySubaccountIdToHex(req.SubAccountId)
	if err != nil {
		xlog.Errorf("Failed to convert SubaccountId: %v", req.SubAccountId)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "stake tokens error: unable to convert SubAccountId into Hex form"})
		return
	}

	xlog.Infof("Processing stakeTokens request for subAccount: %v for amount: %v", req.SubAccountId, req.TokenAmount)

	signature := ctx.GetHeader("Logx-Signature")
	signerAddress := ctx.GetHeader("Logx-Signer-Address")

	if errVerf := contractUtils.VerifyStakeRequest(req, signature, signerAddress); errVerf != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("stake tokens error: verification failed: %v", errVerf)})
		return
	}

	transactionCounter, err := transaction.IncrementCounter(1)
	if err != nil {
		xlog.Errorf("Failed to increment transaction counter: %v", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to increment transaction counter",
		})
		return
	}

	response, err := services.WithNonceRedisLock(subaccountHex, func(currentNonce string) (*Response, error) {
		nonceCheck := cutils.CheckNonce(req.Nonce, currentNonce)
		if !nonceCheck {
			return &Response{status: http.StatusBadRequest, data: "nonce check failed"}, fmt.Errorf("nonce check failed")
		}

		if !t.isValidStakeTokens(req) {
			xlog.Errorf("Requested stake amount > available amount. Requested stake: %v", req.TokenAmount)
			// ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "stake tokens error: invalid request"})
			return &Response{status: http.StatusBadRequest, data: gin.H{"error": "stake tokens error: invalid request"}}, fmt.Errorf("requested stake amount > available amount")
		}

		currentEarningsx18, transientEarningx18, stakedAmountx18, cumulativeEarningRatex18, spyMultTimeDeltax18, _, err := t.stakingService.GetCurrentEarningsx18(req.SubAccountId)
		if err != nil {
			xlog.Errorf("Token Controller - error while fetching user earnings: %v", err)
			// ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
			return &Response{status: http.StatusInternalServerError, data: gin.H{"error": fmt.Sprintf("error while calculating rewards: %v", err)}}, err
		}

		// Get new offset = (stakedAmount + tokenAmount) * timeDelta * spyRate
		newStakedAmountx18 := new(big.Int).Add(stakedAmountx18, tokenAmountBigInt)
		newOffsetx18 := cutils.Divx18(new(big.Int).Mul(newStakedAmountx18, spyMultTimeDeltax18))
		newOffsetx18 = new(big.Int).Neg(newOffsetx18)

		// Add amount to stake table
		stakeEntry, errDb := t.stakingDB.Insert(req.SubAccountId, req.TokenAmount, cumulativeEarningRatex18.String(), "stake", currentEarningsx18.String(), newOffsetx18.String(), transientEarningx18.String(), "0")
		if errDb != nil {
			xlog.Errorf("Error writing stake to the db %v\n , address: %s", errDb, req.SubAccountId)
			return &Response{status: http.StatusInternalServerError, data: gin.H{"error": fmt.Sprintf("error while adding to stakedb: %v", errDb)}}, errDb
		}

		// Decrease logX balance and increase stLogX balance by tokenAmount
		err = t.balanceClient.UpdateMultiTokenBalance(&balanceTypes.MultiTokenBalanceRequest{
			SubaccountID:  subaccountHex,
			ProductIds:    []uint32{0, 2},
			TokenBalances: []string{"-" + req.TokenAmount, req.TokenAmount},
		})

		if err != nil {
			xlog.Errorf("Error updating token balances.. Deleting stake entry Id: %v: SubaccountHex: %v | err: %v", stakeEntry.ID, subaccountHex, err)
			// Delete the stake entry from the db
			if errDelete := t.stakingDB.SetDeletedAtById(stakeEntry.ID, time.Date(2012, 11, 9, 0, 0, 0, 0, time.UTC)); errDelete != nil {
				xlog.Errorf("Error deleting stake entry from db: %v", errDelete)
			}
			return &Response{status: http.StatusInternalServerError, data: gin.H{"error": "Error updating token balances"}}, err
		}

		services.IncrementNonce(subaccountHex, currentNonce)
		status, err := contract.GlobalContracts.EndpointContract.StakeLogx(req, tokenAmountBigInt, transientEarningx18, signature, transactionCounter)
		if !status || err != nil {
			// ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "stake tokens error: contract call failed"})
			return &Response{status: http.StatusInternalServerError, data: gin.H{"error": "stake tokens error: contract call failed"}}, err
		}
		return &Response{status: http.StatusOK, data: gin.H{"success": true}}, nil
	})

	if err != nil {
		xlog.Errorf("Error in stake tokens: %v", err)
		ctx.AbortWithStatusJSON(response.status, response.data)
		return
	}

	ctx.JSON(response.status, response.data)
}

func (t *TokenController) UnstakeTokens(ctx *gin.Context) {
	var req contractUtils.UnstakeLogXRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "unstake tokens error: invalid request"})
		return
	}

	subaccountHex, err := cutils.HackySubaccountIdToHex(req.SubAccountId)

	if err != nil {
		xlog.Errorf("Failed to convert SubaccountId: %v", req.SubAccountId)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "stake tokens error: unable to convert SubAccountId into Hex form"})
		return
	}

	signature := ctx.GetHeader("Logx-Signature")
	signerAddress := ctx.GetHeader("Logx-Signer-Address")

	if errVerf := contractUtils.VerifyUnstakeRequest(req, signature, signerAddress); errVerf != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("unstake tokens error: verification failed: %v", errVerf)})
		return
	}

	if req.ChainId != contractUtils.EndpointChainId() {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "unstake tokens error: invalid chainId"})
		return
	}
	if req.StakerContract != contractUtils.STAKER_CONTRACT_ADDRESS {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "unstake tokens error: invalid staker contract address"})
		return
	}
	if req.ProductId != contractUtils.ST_LOGX {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "unstake tokens error: invalid productId"})
		return
	}

	tokenAmountBigInt := new(big.Int)
	tokenAmountBigInt, success1 := tokenAmountBigInt.SetString(req.Amount, 10)
	if !success1 {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "stake tokens error: unable to convert token amount string to big.Int"})
		return
	}

	if tokenAmountBigInt.Cmp(big.NewInt(0)) <= 0 {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "unstake tokens error: token amount must be greater than zero"})
		return
	}

	transactionCounter, err := transaction.IncrementCounter(1)
	if err != nil {
		xlog.Errorf("Failed to increment transaction counter: %v", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to increment transaction counter",
		})
		return
	}

	response, err := services.WithNonceRedisLock(subaccountHex, func(currentNonce string) (*Response, error) {
		nonceCheck := cutils.CheckNonce(req.Nonce, currentNonce)
		if !nonceCheck {
			// ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "nonce check failed"})
			return &Response{status: http.StatusBadRequest, data: "nonce check failed"}, fmt.Errorf("nonce check failed")
		}

		xlog.Infof("Processing UnstakeToken request for subAccount: %v, for amount: %v", req.SubAccountId, req.Amount)

		currentEarningsx18, transientEarningx18, stakedAmountx18, cumulativeEarningRatex18, spyMultTimeDeltax18, _, err := t.stakingService.GetCurrentEarningsx18(req.SubAccountId)
		if err != nil {
			xlog.Errorf("Token Controller - error while fetching user current earnings: %v", err)
			// ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
			return &Response{status: http.StatusInternalServerError, data: gin.H{"error": fmt.Sprintf("error while calculating rewards: %v", err)}}, err
		}

		// Verify that staked amount is greater than or equal to the requested unstake amount
		if stakedAmountx18.Cmp(tokenAmountBigInt) < 0 {
			xlog.Errorf("Requested unstake amount > staked amount. Requested unstake: %v", req.Amount)
			// ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "unstake tokens error: invalid request"})
			return &Response{status: http.StatusBadRequest, data: "unstake tokens error: invalid request"}, fmt.Errorf("requested unstake amount > staked amount")
		}

		// Get new offset. tokenAmountBigInt is positive from frontend
		newStakedAmount := new(big.Int).Sub(stakedAmountx18, tokenAmountBigInt)
		newOffsetx18 := cutils.Divx18(new(big.Int).Mul(newStakedAmount, spyMultTimeDeltax18))
		newOffsetx18 = new(big.Int).Neg(newOffsetx18)

		// Store unstake details in db
		unstakeEntry, errDb := t.stakingDB.Insert(req.SubAccountId, req.Amount, cumulativeEarningRatex18.String(), "unstake", currentEarningsx18.String(), newOffsetx18.String(), transientEarningx18.String(), "0")
		if errDb != nil {
			xlog.Errorf("Could not unstake for subaccount%v from db because of error: %v", req.SubAccountId, errDb)
			return &Response{status: http.StatusInternalServerError, data: gin.H{"error": fmt.Sprintf("error while adding to staking table: %v", errDb)}}, errDb
		}

		// Increase LogX balance and decrease stLogX balance by tokenAmount
		err = t.balanceClient.UpdateMultiTokenBalance(&balanceTypes.MultiTokenBalanceRequest{
			SubaccountID:  subaccountHex,
			ProductIds:    []uint32{0, 2},
			TokenBalances: []string{req.Amount, "-" + req.Amount},
		})

		if err != nil {
			xlog.Errorf("Error updating token balances.. Deleting unstake entry Id: %v: SubaccountHex: %v | err: %v", unstakeEntry.ID, subaccountHex, err)
			// Delete the stake entry from the db
			if errDelete := t.stakingDB.SetDeletedAtById(unstakeEntry.ID, time.Date(2012, 11, 9, 0, 0, 0, 0, time.UTC)); errDelete != nil {
				xlog.Errorf("Error deleting stake entry from db: %v", errDelete)
			}
			return &Response{status: http.StatusInternalServerError, data: gin.H{"error": "Error updating token balances"}}, err
		}

		services.IncrementNonce(subaccountHex, currentNonce)

		status, err := contract.GlobalContracts.EndpointContract.UnstakeLogx(req, tokenAmountBigInt, transientEarningx18, signature, transactionCounter)
		if !status || err != nil {
			// ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "unstake tokens error: contract call failed"})
			return &Response{status: http.StatusInternalServerError, data: gin.H{"error": "unstake tokens error: contract call failed"}}, err
		}

		return &Response{status: http.StatusOK, data: gin.H{"success": true}}, nil
	})

	if err != nil {
		xlog.Errorf("Error in unstake tokens: %v", err)
		ctx.AbortWithStatusJSON(response.status, response.data)
		return
	}

	ctx.JSON(response.status, response.data)

}

func (t *TokenController) isValidWithdrawLogX(req contractUtils.WithdrawLogX) bool {
	subaccountID, err := cutils.SubaccountIdToHex(req.SubAccountId)

	if err != nil {
		xlog.Errorf("Failed to convert SubaccountId: %v", req.SubAccountId)
		return false
	}

	// Fetch spot balances
	var spotBalances []ctypes.SpotBalance
	spotBalances, _, _, err = t.balanceClient.GetSpotBalance(subaccountID)
	if err != nil {
		xlog.Errorf("Error fetching spot balances for subaccount %v: %v", subaccountID, err)
		return false
	}

	// Initialize total unstake amount within the last hour
	stakingDB := &db.StakingDB{}
	unstakeSum, err := stakingDB.GetUnstakeSumOfPendingUnstake(req.SubAccountId)
	if err != nil {
		xlog.Errorf("Error fetching unstake sum for subaccount %v: %v", req.SubAccountId, err)
		return false
	}

	// Iterate through spot balances and adjust based on unstake amounts
	var tokenAmountStr string
	for _, spotBalance := range spotBalances {
		if spotBalance.ProductId == 0 {
			// Convert token balance to big.Int
			tokenBalanceInt := new(big.Int)
			if _, ok := tokenBalanceInt.SetString(spotBalance.TokenBalance, 10); !ok {
				xlog.Errorf("Invalid token balance for subaccount %v", subaccountID)
				return false
			}

			// Subtract unstake sum from available balance
			tokenBalanceInt.Sub(tokenBalanceInt, unstakeSum)

			// Store the adjusted token balance as a string
			tokenAmountStr = tokenBalanceInt.String()
		}
	}

	// Convert the adjusted token balance and request amount to big.Int
	tokenAmountInt, _ := new(big.Int).SetString(tokenAmountStr, 10)
	reqAmountInt, _ := new(big.Int).SetString(req.TokenAmount, 10)

	// Check if the requested amount is less than or equal to the available adjusted balance
	return reqAmountInt.Cmp(tokenAmountInt) <= 0
}

func checkAllowedWithdrawal(address string, productId uint32) bool {
	if productId == 72 || productId == 74 {
		return true
	}
	return address == "0x23684dcc65E7C20E32AD1623323670Ea9ED1bF6E"
}

func (t *TokenController) WithdrawLogX(ctx *gin.Context) {
	var req contractUtils.WithdrawLogX

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "withdraw tokens error: invalid request"})
		return
	}

	if req.ChainId != contractUtils.EndpointChainId() {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "stake tokens error: invalid chainId"})
		return
	}
	if req.BridgeOutContract != contractUtils.BRIDGE_OUT_CONTRACT {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "stake tokens error: invalid staker contract address"})
		return
	}

	// VERIFY IF RECEIVER ADDRESS MATCHES ADDRESS EXATRACTED FROM SUBACCOUNT ID
	addressFromSubaccountID, err := cutils.ExtractEthereumAddress(req.SubAccountId)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusBadRequest, fmt.Sprintf("withdraw error: invalid subaccount ID: %s", req.SubAccountId))
		return
	}

	if !checkAllowedWithdrawal(addressFromSubaccountID, contractUtils.LOGX) {
		cutils.ApiAbort(ctx, http.StatusForbidden, "withdraw error: withdrawals are not allowed")
		return
	}

	isInternal, err := cutils.IsInternalSubaccount(req.SubAccountId)
	if err != nil {
		xlog.Errorf("Error checking if subaccount is internal: %v", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "withdraw tokens error"})
		return
	}
	if isInternal {
		ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "withdraw tokens error: some error occurred"})
		return
	}
	if addressFromSubaccountID != req.Receiver {
		cutils.ApiAbort(ctx, http.StatusBadRequest, fmt.Sprintf("withdraw error: invalid receiver address: %s does not match extracted address from subaccount: %s: %s", req.Receiver, req.SubAccountId, addressFromSubaccountID))
		return
	}
	if !contractUtils.SUPPORTED_LOGX_CHAINS[req.DestinationChainId] {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "withdraw tokens error: unsupported destination chain ID"})
		return
	}

	xlog.Infof("Processing withdrawlLogX request for subAccount: %v for amount: %v", req.SubAccountId, req.TokenAmount)
	signature := ctx.GetHeader("Logx-Signature")
	signerAddress := ctx.GetHeader("Logx-Signer-Address")

	tokenAmountBigInt := new(big.Int)
	tokenAmountBigInt, success1 := tokenAmountBigInt.SetString(req.TokenAmount, 10)
	if !success1 {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "withdraw tokens error: unable to convert token amount string to big.Int"})
		return
	}

	// LOGX token withdrawal should be greater than 5
	if tokenAmountBigInt.Cmp(getWithdrawalFee(contractUtils.LOGX)) <= 0 {
		xlog.Errorf("Withdraw LogX Failed: token amount is not greater than fee for subAccount: %s, tokenAmount: %s", req.SubAccountId, req.TokenAmount)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("withdraw collateral error: amount should be greater than fee: %v LOGX", getWithdrawalFee(contractUtils.LOGX))})
		return
	}

	valueAfterSignChange := "-" + req.TokenAmount

	if errVerf := contractUtils.VerifyWithdrawLogX(req, signature, signerAddress); errVerf != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("withdraw tokens error: verification failed: %v", errVerf)})
		return
	}

	subaccountHex, err := cutils.HackySubaccountIdToHex(req.SubAccountId)
	if err != nil {
		xlog.Errorf("Failed to convert SubaccountId: %v", req.SubAccountId)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "withdraw tokens error: unable to convert SubAccountId into Hex form"})
		return
	}

	transactionCounter, err := transaction.IncrementCounter(1)
	if err != nil {
		xlog.Errorf("Failed to increment transaction counter: %v", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to increment transaction counter",
		})
		return
	}

	// Make a global var messageIdBytes here
	var messageIdBytesGlobal []byte

	response, err := services.WithNonceRedisLock(subaccountHex, func(currentNonce string) (*Response, error) {
		nonceCheck := cutils.CheckNonce(req.Nonce, currentNonce)
		if !nonceCheck {
			// ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "nonce check failed"})
			return &Response{status: http.StatusBadRequest, data: gin.H{"error": "nonce check failed"}}, fmt.Errorf("nonce check failed")
		}

		if !t.isValidWithdrawLogX(req) {
			xlog.Errorf("Requested amount > available amount. Requested Amount: %v", req.TokenAmount)
			// ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "withdraw tokens error: invalid request"})
			return &Response{status: http.StatusBadRequest, data: gin.H{"error": "withdraw tokens error: invalid request"}}, fmt.Errorf("requested amount > available amount")
		}

		//Call Post API to update claim status on analytics database
		//ToDo - handle scneario where contract call is successful but updating in the db is not
		//	In this case, we dont want to send an error response since from user's perspective everythig worked fine
		xlog.Infof("Token Controller - Token Amount for withdrawal of LogX on contract %s\n, address: %s", valueAfterSignChange, req.SubAccountId)

		logxUpdateStatus, err := t.balanceClient.UpdateTokenBalance(subaccountHex, 0, valueAfterSignChange)
		if !logxUpdateStatus || err != nil {
			xlog.Errorf("Token Controller - Error updating LogX amount on balance server: %v, subAccountIdHex: %s, tokenAmount: %s", err, subaccountHex, valueAfterSignChange)
			return &Response{status: http.StatusInternalServerError, data: gin.H{"error": "Awaiting status update on blockchain, please wait for a few minutes"}}, err
		}

		err1 := services.IncrementNonce(subaccountHex, currentNonce)

		if err1 != nil {
			// Revert the token balance update if IncrementNonce fails
			revertStatus, revertErr := t.balanceClient.UpdateTokenBalance(subaccountHex, 0, ctypes.ConvertStringAndNegate(valueAfterSignChange))
			if revertErr != nil || !revertStatus {
				xlog.Errorf("Token Controller - Failed to revert LogX token balance after IncrementNonce failure: %v, original error: %v, subAccountIdHex: %s, tokenAmount: %s", revertErr, err1, subaccountHex, valueAfterSignChange)
			}
			xlog.Errorf("Token Controller - IncrementNonce failed: %v, subAccountIdHex: %s", err1, subaccountHex)
			return &Response{status: http.StatusInternalServerError, data: gin.H{"error": "Awaiting status update on blockchain, please wait for a few minutes"}}, err1
		}

		status, err, messageIdBytes := contract.GlobalContracts.EndpointContract.WithdrawLogX(req, tokenAmountBigInt, signature, transactionCounter)
		if !status || err != nil {
			xlog.Errorf("Token Controller - WithdrawLogX contract call failed: %v, subAccountIdHex: %s, tokenAmount: %s, transactionCounter: %d", err, subaccountHex, valueAfterSignChange, transactionCounter)

			// Attempt to mark the transaction as deleted in the DB
			services.IncrementNonce(subaccountHex, currentNonce, true) // Decrement nonce (no need to capture return value)

			revertStatus, revertErr := t.balanceClient.UpdateTokenBalance(subaccountHex, 0, ctypes.ConvertStringAndNegate(valueAfterSignChange))
			if revertErr != nil || !revertStatus {
				xlog.Errorf("Token Controller - Failed to revert LogX token balance after WithdrawLogX contract call failure: %v, subAccountIdHex: %s, tokenAmount: %s", revertErr, subaccountHex, valueAfterSignChange)
			}
			return &Response{status: http.StatusInternalServerError, data: gin.H{"error": "withdraw tokens error: contract call failed"}}, err
		}
		messageIdBytesGlobal = messageIdBytes

		xlog.Infof("Token Controller - Successfully processed LogX token withdrawal, subAccountIdHex: %s, tokenAmount: %s", subaccountHex, valueAfterSignChange)

		return &Response{status: http.StatusOK, data: gin.H{"success": true}}, nil
	})

	if err != nil {
		xlog.Errorf("Error in withdraw tokens: %v", err)
		ctx.AbortWithStatusJSON(response.status, response.data)
		return
	} else {
		// Calculate messageId
		hash := sha3.NewLegacyKeccak256()
		hash.Write(messageIdBytesGlobal)
		messageId := common.BytesToHash(hash.Sum(nil))
		_, errDb := t.depositWithdrawDB.InsertWithdraw(messageId.Hex(), subaccountHex, req.TokenAmount, uint32(contractUtils.LOGX), uint64(contractUtils.LOGX_CHAIN_ID), uint64(req.DestinationChainId))
		if errDb != nil {
			xlog.Errorf("Error writing withdrawal history to the db %v\n , address: %s", errDb, req.SubAccountId)
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"status":  "error",
				"message": "Failed to write withdrawal history to the db",
			})
			return
		}
		ctx.JSON(response.status, response.data)
	}
}

func (t *TokenController) ClaimRewards(ctx *gin.Context) {
	var req contractUtils.ClaimRewards

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "claim rewards error: invalid request"})
		return
	}

	if req.ChainId != contractUtils.EndpointChainId() {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "claim rewards error: invalid chainId"})
		return
	}
	if req.ProductId != contractUtils.LOGX {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "claim rewards error: invalid productId"})
		return
	}
	if req.StakerContract != contractUtils.STAKER_CONTRACT_ADDRESS {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "claim rewards error: invalid staker contract address"})
		return
	}
	xlog.Infof("Processing claimRewards request for subAccount: %v", req.SubAccountId)
	signature := ctx.GetHeader("Logx-Signature")
	signerAddress := ctx.GetHeader("Logx-Signer-Address")

	if errVerf := contractUtils.VerifyClaimRewards(req, signature, signerAddress); errVerf != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("claim rewards error: verification failed: %v", errVerf)})
		return
	}

	transactionCounter, err := transaction.IncrementCounter(1)
	if err != nil {
		xlog.Errorf("Failed to increment transaction counter: %v", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to increment transaction counter",
		})
		return
	}

	subAccountIdHex, err := cutils.HackySubaccountIdToHex(req.SubAccountId)
	if err != nil {
		xlog.Errorf("Failed to convert SubaccountId: %v", req.SubAccountId)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "claim rewards error: unable to convert SubAccountId into Hex form"})
		return
	}

	response, err := services.WithNonceRedisLock(subAccountIdHex, func(currentNonce string) (*Response, error) {
		nonceCheck := cutils.CheckNonce(req.Nonce, currentNonce)
		if !nonceCheck {
			// ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "nonce check failed"})
			return &Response{status: http.StatusBadRequest, data: gin.H{"error": "nonce check failed"}}, fmt.Errorf("nonce check failed")
		}

		earningData := t.stakingService.GetClaimableEarnings(req.SubAccountId)
		if earningData == nil {
			xlog.Errorf("Token Controller - error while fetching user earnings: %v", err)
			// ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
			return &Response{status: http.StatusInternalServerError, data: gin.H{"error": "earning calculation failed"}}, fmt.Errorf("earning calculation failed")
		}

		if earningData.TotalClaimableEarningsx18.Sign() <= 0 {
			xlog.Errorf("Token Controller - No earnings to claim for subaccount %v", req.SubAccountId)
			// ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "No earnings to claim"})
			return &Response{status: http.StatusBadRequest, data: gin.H{"error": "No earnings to claim"}}, fmt.Errorf("no earnings to claim")
		}
		refund, errPool := services.ReserveLogxRewards(earningData.TotalClaimableEarningsx18)
		if errPool != nil {
			return &Response{status: http.StatusServiceUnavailable, data: gin.H{"error": "reward claims are temporarily unavailable"}}, errPool
		}
		claimed := false
		defer func() {
			if !claimed {
				refund()
			}
		}()

		newOffsetx18 := cutils.Divx18(new(big.Int).Mul(earningData.CurrentStakedAmountx18, earningData.SpyMultTimeDeltax18))
		newOffsetx18 = new(big.Int).Neg(newOffsetx18)

		claimEntry, errDb := t.stakingDB.Insert(req.SubAccountId, "0", earningData.CumulativeEarningsRatex18.String(), "claim", earningData.CurrentEarningx18.String(), newOffsetx18.String(), earningData.CurrentTransientEarningsx18.String(), earningData.TotalClaimableEarningsx18.String())
		if errDb != nil {
			xlog.Errorf("Error writing claim earnings to the db %v\n , address: %s", errDb, req.SubAccountId)
			// ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
			return &Response{status: http.StatusBadRequest, data: gin.H{"error": "failed to insert into staking DB"}}, errDb
		}

		// Increase LogX balance by total claimed earnings made from last claim
		err = t.balanceClient.UpdateMultiTokenBalance(&balanceTypes.MultiTokenBalanceRequest{
			SubaccountID:  subAccountIdHex,
			ProductIds:    []uint32{0},
			TokenBalances: []string{earningData.TotalClaimableEarningsx18.String()},
		})

		if err != nil {
			xlog.Errorf("Error updating token balances.. Deleting claim entry Id: %v: SubaccountHex: %v | err: %v", claimEntry.ID, subAccountIdHex, err)
			// Delete the stake entry from the db
			if errDelete := t.stakingDB.SetDeletedAtById(claimEntry.ID, time.Date(2012, 11, 9, 0, 0, 0, 0, time.UTC)); errDelete != nil {
				xlog.Errorf("Error deleting claim entry from db: %v", errDelete)
			}
			return &Response{status: http.StatusInternalServerError, data: gin.H{"error": "Error updating token balances"}}, err
		}

		services.IncrementNonce(subAccountIdHex, currentNonce)
		status, err := contract.GlobalContracts.EndpointContract.ClaimRewards(req, earningData.CurrentTransientEarningsx18, earningData.TotalClaimableEarningsx18, signature, transactionCounter)
		if !status || err != nil {
			// ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "claim rewards error: contract call failed"})
			return &Response{status: http.StatusBadRequest, data: gin.H{"error": "claim rewards error: contract call failed"}}, nil
		}
		claimed = true
		return &Response{status: http.StatusOK, data: gin.H{"success": true}}, nil
	})

	if err != nil {
		xlog.Errorf("Error in claim rewards: %v", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	ctx.JSON(response.status, response.data)
}

func getWithdrawalFee(productId uint32) *big.Int {
	if val, exists := contractUtils.WITHDRAWAL_FEE_MAP[productId]; exists {
		return cutils.FloatStrToX18(val)
	}

	return cutils.FloatStrToX18("0.2") // Default fee if not found
}

// 1. SubaccountFlagged check
// 2. Chain ID check - And shouldn't be LogX chain id / Product shouldn't be LogX product id
// 3. Receiver shouldn't be empty and should match subaccount address
// 4. Internal subaccounts cannot withdraw
// 5. Product ID isn't LogX
// 6. KROMA withdrawals are blocked
// 7. Chain ID mapping should exist and should match with request
// 8. Amount check - should be positive and greater than withdrawal fee
// 9. Signature verification
// 10. Increase transaction counter
// Under Lock:
// 11. Check nonce and increment
// 12. Check sufficient balance
// 13. Update balance
// 14. Add Call contract
// 15. Add messsage id hex and details in table.
func (t *TokenController) WithdrawCollateral(ctx *gin.Context) {
	var req contractUtils.WithdrawCollateral

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "withdraw collateral error: invalid request"})
		return
	}
	subaccountID, err := cutils.SubaccountIdToHex(req.SubAccountId)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "failed to convert subaccount to hex")
		return
	}
	// Block withdrawals for flagged subaccounts
	isFlagged, err := cutils.IsSubaccountFlagged(ctx, req.SubAccountId)
	if err != nil {
		xlog.Errorf("Error checking flagged status for subaccount %s: %v", req.SubAccountId, err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal error: contact support"})
		return
	}
	if isFlagged {
		xlog.Warnf("Withdrawal blocked: subaccount %s is flagged for high PnL activity", req.SubAccountId)
		ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "withdrawal blocked: contact support"})
		return
	}
	if req.ChainId != contractUtils.EndpointChainId() {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid chain id"})
		return
	}
	if req.Receiver == "" {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid receiver address: cannot be empty"})
		return
	}

	// VERIFY IF RECEIVER ADDRESS MATCHES ADDRESS EXATRACTED FROM SUBACCOUNT ID
	addressFromSubaccountID, err := cutils.ExtractEthereumAddress(req.SubAccountId)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusBadRequest, fmt.Sprintf("withdraw error: invalid subaccount ID: %s", req.SubAccountId))
		return
	}

	if !checkAllowedWithdrawal(addressFromSubaccountID, req.ProductId) {
		cutils.ApiAbort(ctx, http.StatusForbidden, "withdraw error: withdrawals are not allowed")
		return
	}

	if addressFromSubaccountID != req.Receiver {
		cutils.ApiAbort(ctx, http.StatusBadRequest, fmt.Sprintf("withdraw error: invalid receiver address: %s does not match extracted address from subaccount: %s: %s", req.Receiver, req.SubAccountId, addressFromSubaccountID))
		return
	}

	isInternal, err := cutils.IsInternalSubaccount(subaccountID)
	if err != nil {
		xlog.Errorf("Error checking if subaccount is internal: %v", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "withdraw tokens error"})
		return
	}
	if isInternal {
		ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "withdraw tokens error: some error occurred"})
		return
	}

	if req.ProductId == contractUtils.LOGX {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "withdraw collateral error: invalid product ID"})
		return
	}

	if cutils.SliceExists([]uint32{contractUtils.KROMA_USDC, contractUtils.KROMA_USDT}, req.ProductId) {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "withdraw collateral error: KROMA withdrawals are not allowed."})
		return
	}

	expectedChainId, exists := contractUtils.ProductChainMapping[req.ProductId]
	if !exists {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid product ID"})
		return
	}
	if expectedChainId != req.DestinationChainId {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "destination chain ID does not match the expected chain ID"})
		return
	}

	amountCheck, success := new(big.Int).SetString(req.Amount, 10)
	if !success || req.Amount == "" || amountCheck.Sign() <= 0 {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid withdrawal amount"})
		return
	}

	if amountCheck.Cmp(getWithdrawalFee(req.ProductId)) <= 0 {
		xlog.Errorf("Withdrawal amount %v is less than or equal to the withdrawal fee for product ID %v", req.Amount, req.ProductId)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("withdraw collateral error: amount should be greater than fee: %v Amount", getWithdrawalFee(req.ProductId))})
		return
	}

	xlog.Infof("Processing withdrawl request for subAccount: %v for productId: %v for amount: %v", req.SubAccountId, req.ProductId, req.Amount)

	signature := ctx.GetHeader("Logx-Signature")
	if signature == "" {
		signature = ctx.GetHeader("Broker-Signature")
	}

	signerAddress := ctx.GetHeader("Logx-Signer-Address")
	if signerAddress == "" {
		signerAddress = ctx.GetHeader("Broker-Signer-Address")
	}

	amountBigInt := new(big.Int)
	amountBigInt, success1 := amountBigInt.SetString(req.Amount, 10)
	if !success1 {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "withdraw collateral error: unable to convert amount string to big.Int"})
		return
	}

	valueAfterSignChange := "-" + req.Amount

	if errVerf := contractUtils.VerifyWithdrawCollateral(req, signature, signerAddress); errVerf != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("withdraw collateral error: verification failed: %v", errVerf)})
		return
	}

	subAccountIdHex, err := cutils.HackySubaccountIdToHex(req.SubAccountId)
	if err != nil {
		xlog.Errorf("Failed to convert SubaccountId: %v", req.SubAccountId)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "claim rewards error: unable to convert SubAccountId into Hex form"})
		return
	}

	transactionCounter, err := transaction.IncrementCounter(1)
	if err != nil {
		xlog.Errorf("Failed to increment transaction counter: %v", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to increment transaction counter",
		})
		return
	}

	// Make a global var messageIdBytes here
	var messageIdBytesGlobal []byte

	response, err := services.WithNonceRedisLock(subAccountIdHex, func(currentNonce string) (*Response, error) {
		nonceCheck := cutils.CheckNonce(req.Nonce, currentNonce)
		if !nonceCheck {
			// ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "nonce check failed"})
			return &Response{status: http.StatusBadRequest, data: gin.H{"error": "nonce check failed"}}, fmt.Errorf("nonce check failed")
		}

		withdrawableTokenBalance, err := t.balanceClient.GetWithdrawableTokenBalance(subaccountID)

		if err != nil {
			// cutils.ApiAbort(ctx, http.StatusInternalServerError, "failed to fetch available balance")
			return &Response{status: http.StatusInternalServerError, data: "failed to fetch available balance"}, err
		}

		availableAmountStr, exists := withdrawableTokenBalance[req.ProductId]
		if !exists {
			xlog.Errorf("Product ID %d not found in withdrawable balance", req.ProductId)
			return &Response{status: http.StatusBadRequest, data: gin.H{"error": "failed to parse available balance for product ID"}}, fmt.Errorf("product ID not found in withdrawable balance")
		}

		availableAmount, ok := new(big.Int).SetString(availableAmountStr, 10)
		if !ok {
			xlog.Errorf("Failed to parse available balance for product ID %d", req.ProductId)
			return &Response{status: http.StatusInternalServerError, data: gin.H{"error": "failed to parse available balance for product ID"}}, fmt.Errorf("failed to parse available balance for product ID")
		}
		requestedAmount, _ := new(big.Int).SetString(req.Amount, 10)

		if availableAmount.Cmp(requestedAmount) < 0 {
			// ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "User doesn't have sufficient balance to withdraw"})
			return &Response{status: http.StatusBadRequest, data: gin.H{"error": "User doesn't have sufficient balance to withdraw"}}, fmt.Errorf("user doesn't have sufficient balance to withdraw")
		}

		//Call Post API to update claim status on analytics database
		//ToDo - handle scneario where contract call is successful but updating in the db is not
		//	In this case, we dont want to send an error response since from user's perspective everythig worked fine
		err1 := services.IncrementNonce(subAccountIdHex, currentNonce)
		if err1 != nil {
			xlog.Errorf("Token Controller - IncrementNonce failed: %v, subAccountIdHex: %s", err1, subAccountIdHex)
			return &Response{status: http.StatusInternalServerError, data: gin.H{"error": "Awaiting status update on blockchain, please wait for a few minutes"}}, err1
		}

		tokenUpdateStatus, err := t.balanceClient.UpdateTokenBalance(subAccountIdHex, req.ProductId, valueAfterSignChange)
		if !tokenUpdateStatus || err != nil {
			xlog.Errorf("Token Controller - Error updating token amount on balance server: %v, subAccountIdHex: %s, productId: %d, tokenAmount: %s", err, subAccountIdHex, req.ProductId, valueAfterSignChange)

			// Revert nonce increment
			services.IncrementNonce(subAccountIdHex, currentNonce, true) // Decrement nonce
			return &Response{status: http.StatusInternalServerError, data: gin.H{"error": "Awaiting status update on blockchain, please wait for a few minutes"}}, err
		}

		status, err, messageIdBytes := contract.GlobalContracts.EndpointContract.WithdrawCollateral(req, amountBigInt, signature, transactionCounter)
		if !status || err != nil {
			xlog.Errorf("Token Controller - WithdrawCollateral contract call failed: %v, subAccountIdHex: %s, productId: %d, tokenAmount: %s, transactionCounter: %d", err, subAccountIdHex, req.ProductId, valueAfterSignChange, transactionCounter)

			// Attempt to mark the transaction as deleted in the DB
			services.IncrementNonce(subAccountIdHex, currentNonce, true) // Decrement nonce

			revertStatus, revertErr := t.balanceClient.UpdateTokenBalance(subAccountIdHex, req.ProductId, ctypes.ConvertStringAndNegate(valueAfterSignChange))
			if revertErr != nil || !revertStatus {
				xlog.Errorf("Token Controller - Failed to revert token balance after WithdrawCollateral contract call failure: %v, subAccountIdHex: %s, productId: %d, tokenAmount: %s", revertErr, subAccountIdHex, req.ProductId, valueAfterSignChange)
			}
			return &Response{status: http.StatusInternalServerError, data: gin.H{"error": "withdraw collateral error: contract call failed"}}, err
		}

		messageIdBytesGlobal = messageIdBytes

		xlog.Infof("Token Controller - Successfully processed collateral withdrawal, subAccountIdHex: %s, productId: %d, tokenAmount: %s", subAccountIdHex, req.ProductId, valueAfterSignChange)
		// cutils.TokenApiResponse(ctx, http.StatusOK, status)
		return &Response{status: http.StatusOK, data: gin.H{"success": true}}, nil
	})

	if err != nil {
		xlog.Errorf("Error in withdraw collateral: %v", err)
		ctx.AbortWithStatusJSON(response.status, response.data)
		return
	} else {
		// Calculate messageId
		hash := sha3.NewLegacyKeccak256()
		hash.Write(messageIdBytesGlobal)
		messageId := common.BytesToHash(hash.Sum(nil))

		_, errDb := t.depositWithdrawDB.InsertWithdraw(messageId.Hex(), subAccountIdHex, req.Amount, uint32(req.ProductId), uint64(contractUtils.LOGX_CHAIN_ID), uint64(req.DestinationChainId))
		if errDb != nil {
			xlog.Errorf("Error writing withdrawal history to the db %v\n , address: %s", errDb, req.SubAccountId)
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"status":  "error",
				"message": "Failed to write withdrawal history to the db",
			})
			return
		}
		ctx.JSON(response.status, response.data)
	}
}

func (t *TokenController) SettleBroker2PnL(ctx *gin.Context) {
	var req contractUtils.SettleBroker2PnL

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "settle Ostrich PnL error: invalid request"})
		return
	}

	subaccountIdHex, err := cutils.SubaccountIdToHex(req.SubAccountId)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "failed to convert subaccount to hex")
		return
	}

	// Verify broker ID is 2 (Ostrich)
	brokerId := cutils.ExtractBrokerIdFromSubaccountHex(subaccountIdHex)
	if brokerId != 2 {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "settle Ostrich PnL error: subaccount is not an Ostrich account"})
		return
	}

	xlog.Infof("Processing Ostrich PnL settlement request for subAccount: %v", subaccountIdHex)

	response, err := services.WithNonceRedisLock(subaccountIdHex, func(currentNonce string) (*Response, error) {

		err = t.balanceClient.SettleBroker2PnL(subaccountIdHex)
		if err != nil {
			xlog.Errorf("Token Controller - SettleBroker2PnL failed: %v, subAccountIdHex: %s", err, subaccountIdHex)
			return &Response{status: http.StatusInternalServerError, data: gin.H{"error": fmt.Sprintf("failed to settle Ostrich PnL: %v", err)}}, err
		}

		transactionCounter, errTxnCounter := transaction.IncrementCounter(1)
		if errTxnCounter != nil {
			xlog.Errorf("Token Controller - GetTransactionCounter failed: %v", errTxnCounter)
			return &Response{status: http.StatusInternalServerError, data: gin.H{"error": "failed to get transaction counter"}}, errTxnCounter
		}

		success, contractErr := contract.GlobalContracts.EndpointContract.ShiftBalance(req, transactionCounter)
		if contractErr != nil || !success {
			xlog.Errorf("Token Controller - Contract ShiftBalance failed: %v", contractErr)
			return &Response{status: http.StatusInternalServerError, data: gin.H{"error": fmt.Sprintf("failed to add contract transaction: %v", contractErr)}}, contractErr
		}

		xlog.Infof("Token Controller - Successfully processed Ostrich PnL settlement, subAccountIdHex: %s", subaccountIdHex)
		return &Response{status: http.StatusOK, data: gin.H{
			"message": "Ostrich PnL settled successfully",
		}}, nil
	})

	if err != nil {
		xlog.Errorf("Error in settle Ostrich PnL: %v", err)
		ctx.AbortWithStatusJSON(response.status, response.data)
		return
	} else {
		ctx.JSON(response.status, response.data)
	}
}

func (t *TokenController) ShiftBalanceKroma(ctx *gin.Context) {
	var req contractUtils.ShiftBalanceKroma

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "shift Kroma balance error: invalid request"})
		return
	}

	subaccountIdHex, err := cutils.SubaccountIdToHex(req.SubAccountId)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "failed to convert subaccount to hex")
		return
	}

	xlog.Infof("Processing Kroma balance shift request for subAccount: %v", subaccountIdHex)

	response, err := services.WithNonceRedisLock(subaccountIdHex, func(currentNonce string) (*Response, error) {

		err = t.balanceClient.ShiftKromaFunds(subaccountIdHex)
		if err != nil {
			xlog.Errorf("Token Controller - ShiftKromaFunds failed: %v, subAccountIdHex: %s", err, subaccountIdHex)
			return &Response{status: http.StatusInternalServerError, data: gin.H{"error": fmt.Sprintf("failed to shift Kroma funds: %v", err)}}, err
		}

		transactionCounter, errTxnCounter := transaction.IncrementCounter(1)
		if errTxnCounter != nil {
			xlog.Errorf("Token Controller - GetTransactionCounter failed: %v", errTxnCounter)
			return &Response{status: http.StatusInternalServerError, data: gin.H{"error": "failed to get transaction counter"}}, errTxnCounter
		}

		success, contractErr := contract.GlobalContracts.EndpointContract.ShiftBalanceKroma(req, transactionCounter)
		if contractErr != nil || !success {
			xlog.Errorf("Token Controller - Contract ShiftBalanceKroma failed: %v", contractErr)
			return &Response{status: http.StatusInternalServerError, data: gin.H{"error": fmt.Sprintf("failed to add contract transaction: %v", contractErr)}}, contractErr
		}

		xlog.Infof("Token Controller - Successfully processed Kroma balance shift, subAccountIdHex: %s", subaccountIdHex)
		return &Response{status: http.StatusOK, data: gin.H{
			"message": "Kroma funds shifted to Arbitrum successfully",
		}}, nil
	})

	if err != nil {
		xlog.Errorf("Error in shift Kroma balance: %v", err)
		ctx.AbortWithStatusJSON(response.status, response.data)
		return
	} else {
		ctx.JSON(response.status, response.data)
	}
}

func (t *TokenController) BurnBalanceKroma(ctx *gin.Context) {
	var req contractUtils.BurnBalanceKroma

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "burn Kroma balance error: invalid request"})
		return
	}

	subaccountIdHex, err := cutils.SubaccountIdToHex(req.SubAccountId)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "failed to convert subaccount to hex")
		return
	}

	xlog.Infof("Processing Kroma balance burn request for subAccount: %v", subaccountIdHex)

	response, err := services.WithNonceRedisLock(subaccountIdHex, func(currentNonce string) (*Response, error) {

		err = t.balanceClient.BurnKromaFunds(subaccountIdHex)
		if err != nil {
			xlog.Errorf("Token Controller - BurnKromaFunds failed: %v, subAccountIdHex: %s", err, subaccountIdHex)
			return &Response{status: http.StatusInternalServerError, data: gin.H{"error": fmt.Sprintf("failed to burn Kroma funds: %v", err)}}, err
		}

		transactionCounter, errTxnCounter := transaction.IncrementCounter(1)
		if errTxnCounter != nil {
			xlog.Errorf("Token Controller - GetTransactionCounter failed: %v", errTxnCounter)
			return &Response{status: http.StatusInternalServerError, data: gin.H{"error": "failed to get transaction counter"}}, errTxnCounter
		}

		success, contractErr := contract.GlobalContracts.EndpointContract.BurnBalanceKroma(req, transactionCounter)
		if contractErr != nil || !success {
			xlog.Errorf("Token Controller - Contract BurnBalanceKroma failed: %v", contractErr)
			return &Response{status: http.StatusInternalServerError, data: gin.H{"error": fmt.Sprintf("failed to add contract transaction: %v", contractErr)}}, contractErr
		}

		xlog.Infof("Token Controller - Successfully processed Kroma balance burn, subAccountIdHex: %s", subaccountIdHex)
		return &Response{status: http.StatusOK, data: gin.H{
			"message": "Kroma funds burned successfully",
		}}, nil
	})

	if err != nil {
		xlog.Errorf("Error in burn Kroma balance: %v", err)
		ctx.AbortWithStatusJSON(response.status, response.data)
		return
	} else {
		ctx.JSON(response.status, response.data)
	}
}

// func (t *TokenController) ClaimLogXFeeRewards(ctx *gin.Context) {
// 	var req contractUtils.CampaignRewardClaimRequest
// 	defer cutils.LogTime(time.Now(), metric.CLAIM_CAMPAIGN_REWARDS)

// 	if os.Getenv("IS_CLAIM_LOGX_FEE_REWARDS_ALLOWED") == "0" {
// 		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Claim logx fee Rewards is not allowed currently"})
// 		return
// 	}

// 	// Bind request JSON and validate
// 	if err := ctx.ShouldBindJSON(&req); err != nil {
// 		xlog.Errorf("Failed to bind request JSON for campaign rewards claim: %v", err)
// 		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid request: unable to bind JSON"})
// 		return
// 	}

// 	if req.ProductId != contractUtils.LOGX {
// 		xlog.Errorf("Claim Logx Fee Rewards Invalid productId: %d for subAccount: %s", req.ProductId, req.SubAccountId)
// 		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
// 		return
// 	}

// 	// Convert subaccount ID to hex
// 	subaccountHex, err := cutils.HackySubaccountIdToHex(req.SubAccountId)
// 	if err != nil {
// 		xlog.Errorf("Claim Logx Fee Rewards Invalid subaccount ID: %s, error: %v", req.SubAccountId, err)
// 		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid subaccount ID"})
// 		return
// 	}

// 	// Verify chain ID
// 	if req.ChainId != contractUtils.EndpointChainId() {
// 		xlog.Errorf("Claim Logx Fee Rewards Invalid chain ID: %d for subAccount: %s", req.ChainId, req.SubAccountId)
// 		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid chain ID"})
// 		return
// 	}

// 	// Check if the campaign ID matches LOGX_FEE_REWARDS_CAMPAIGN_ID
// 	if req.CampaignId != contractUtils.LOGX_FEE_REWARDS_CAMPAIGN_ID {
// 		xlog.Errorf("Claim Logx Fee Rewards Invalid CampaignId: %d, expected: %d for subAccount: %s", req.CampaignId, contractUtils.LOGX_FEE_REWARDS_CAMPAIGN_ID, req.SubAccountId)
// 		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid Campaign ID"})
// 		return
// 	}

// 	// Get signature and signer address from headers
// 	signature := ctx.GetHeader("Logx-Signature")
// 	signerAddress := ctx.GetHeader("Logx-Signer-Address")
// 	if signature == "" || signerAddress == "" {
// 		xlog.Warnf("Claim Logx Fee Rewards Missing signature or signer address in headers for subAccount: %s", req.SubAccountId)
// 		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Missing signature or signer address"})
// 		return
// 	}

// 	// Convert token amount from string to big.Int
// 	rewardAmountBigIntVal := new(big.Int)
// 	if _, success := rewardAmountBigIntVal.SetString(req.Amount, 10); !success {
// 		xlog.Errorf("Claim Logx Fee Rewards Failed to convert token amount: %s to big.Int for subAccount: %s", req.Amount, req.SubAccountId)
// 		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Unable to convert token amount to big.Int"})
// 		return
// 	}

// 	if rewardAmountBigIntVal.Cmp(big.NewInt(0)) <= 0 {
// 		xlog.Errorf("Claim Logx Fee Rewards Failed: token amount is not greater than zero for subAccount: %s", req.SubAccountId)
// 		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Token amount must be greater than zero"})
// 		return
// 	}

// 	// Fetch the available USDC balance for the subaccount
// 	funds, err := t.balanceClient.GetSubAccountContractSpotBalance(contractUtils.CAMPAIGN_REWARD_SUBACCOUNT_ID, int(req.ProductId))
// 	if err != nil {
// 		xlog.Errorf("Claim Logx Fee Rewards Error fetching funds for contract %s: %v", contractUtils.CAMPAIGN_REWARD_SUBACCOUNT_ID, err)
// 		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Error fetching contract balance."})
// 		return
// 	}

// 	// Compare the requested amount with available funds in contract
// 	if rewardAmountBigIntVal.Cmp(funds) > 0 {
// 		xlog.Errorf("Claim Logx Fee Rewards Requested amount %s exceeds available funds %s for subaccount: %s. Contact development team.", rewardAmountBigIntVal.String(), funds.String(), contractUtils.CAMPAIGN_REWARD_SUBACCOUNT_ID)
// 		xclient.GlobalDiscordClient.SendWebhookMessage("******************* ALERT >>>>>>>>>>>>>>> Add funds in Claim Logx Fee Rewards Contract <<<<<<<<<<<<< ALERT *******************")
// 		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Insufficient funds, please contact the development team."})
// 		return
// 	}

// 	// Verify referral request using signature
// 	if errVerf := contractUtils.VerifyCampaignRewards(req, signature, signerAddress); errVerf != nil {
// 		xlog.Warnf("Claim Logx Fee Rewards Signature verification failed for subAccount: %s, error: %v", req.SubAccountId, errVerf)
// 		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Signature verification failed: %v", errVerf)})
// 		return
// 	}

// 	// Increment transaction counter
// 	transactionCounter, err := transaction.IncrementCounter(1)
// 	if err != nil {
// 		xlog.Errorf("Claim  Logx Fee Rewards Failed to increment transaction counter for subAccount: %s, error: %v", req.SubAccountId, err)
// 		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to increment transaction counter"})
// 		return
// 	}

// 	// Perform the action within a nonce lock
// 	response, err := services.WithNonceRedisLock(subaccountHex, func(currentNonce string) (*Response, error) {
// 		// Validate nonce
// 		nonceCheck := cutils.CheckNonce(req.Nonce, currentNonce)
// 		if !nonceCheck {
// 			xlog.Warnf("Claim  Logx Fee Rewards Nonce check failed for subAccount: %s, provided nonce: %d, currentNonce: %s", req.SubAccountId, req.Nonce, currentNonce)
// 			return &Response{status: http.StatusBadRequest, data: "Nonce check failed"}, nil
// 		}

// 		// Logging the claim request
// 		xlog.Infof("Processing  Logx Fee Rewards for subAccount: %s, amount: %s", req.SubAccountId, req.Amount)

// 		// Get user address from the headers
// 		userAddress := ctx.GetHeader("Logx-User-Address")
// 		if userAddress == "" {
// 			xlog.Warnf("Claim Logx Fee Rewards Missing user address in headers for subAccount: %s", req.SubAccountId)
// 			return &Response{status: http.StatusBadRequest, data: "Missing user address"}, nil
// 		}

// 		// Calculate the total claimable rewards
// 		claimableAmount, err := (&db.UserRewardsDB{}).GetTotalBonusForSubaccount(req.SubAccountId, contractUtils.PRODUCT_ID_SYMBOL_TO_MAP[contractUtils.LOGX])
// 		if err != nil || claimableAmount.Val.Cmp(big.NewInt(0)) == 0 {
// 			xlog.Errorf("No claimable rewards available for userAddress: %s, subAccount: %s, error: %v", userAddress, req.SubAccountId, err)
// 			return &Response{status: http.StatusBadRequest, data: "No claimable rewards available"}, err
// 		}

// 		// Check if the claimed amount does not exceed the claimable amount
// 		if rewardAmountBigIntVal.Cmp(claimableAmount.Val) > 0 {
// 			xlog.Warnf("Claim amount exceeds claimable rewards for userAddress: %s, subAccount: %s, claimedAmount: %s, claimableAmount: %s", userAddress, req.SubAccountId, rewardAmountBigIntVal.String(), claimableAmount.String())
// 			return &Response{status: http.StatusBadRequest, data: "Claim amount exceeds available rewards"}, nil
// 		}

// 		// Update the claim status in the db
// 		err = (&db.UserRewardsDB{}).MarkRewardsAsClaimed(req.SubAccountId, contractUtils.PRODUCT_ID_SYMBOL_TO_MAP[contractUtils.LOGX])
// 		if err != nil {
// 			xlog.Errorf("Failed to update reward claim status for userAddress: %s, subAccount: %s, error: %v", userAddress, req.SubAccountId, err)
// 			return &Response{status: http.StatusInternalServerError, data: "Failed to update reward claim status"}, err
// 		}

// 		// Update the token balance on the balance server
// 		status, err := t.balanceClient.UpdateTokenBalance(subaccountHex, req.ProductId, req.Amount)
// 		if err != nil || !status {
// 			xlog.Errorf("Claim Logx Fee Rewards Failed to update token balance on balance server for subAccount: %s, userAddress: %s, amount: %s, error: %v", subaccountHex, userAddress, req.Amount, err)
// 			return &Response{status: http.StatusInternalServerError, data: "Failed to update token balance"}, err
// 		}

// 		xlog.Infof("Claim Logx Fee Rewards Updated balance service for subAccount: %s, userAddress: %s", req.SubAccountId, userAddress)

// 		// Increment nonce
// 		if err := services.IncrementNonce(subaccountHex, currentNonce); err != nil {
// 			xlog.Errorf("Claim Logx Fee Rewards Failed to increment nonce for subAccount: %s, userAddress: %s, error: %v", subaccountHex, userAddress, err)
// 			return &Response{status: http.StatusInternalServerError, data: "Failed to increment nonce"}, err
// 		}

// 		// Call the contract's ClaimCampaignRewards function
// 		status, err2 := contract.GlobalContracts.EndpointContract.ClaimCampaignRewards(req, rewardAmountBigIntVal, signature, transactionCounter)
// 		if err2 != nil || !status {
// 			xlog.Errorf("Claim Logx Fee Rewards Failed to call contract for ClaimCampaignRewards for subAccount: %s, userAddress: %s, error: %v", subaccountHex, userAddress, err2)
// 			return &Response{status: http.StatusBadRequest, data: "Contract call failed"}, err2
// 		}

// 		// Success
// 		xlog.Infof("Successfully processed Logx Fee Rewards for userAddress: %s, subAccountId: %s, claimedAmount: %s", userAddress, subaccountHex, rewardAmountBigIntVal.String())

// 		return &Response{status: http.StatusOK, data: "Claimed Logx Fee Rewards Amount successfully"}, nil
// 	})

// 	// Handle any errors that occurred within the nonce lock
// 	if err != nil {
// 		xlog.Errorf("Error processing Logx Fee Rewards: %v", err)
// 		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
// 		return
// 	}

// 	// Return the response
// 	ctx.JSON(response.status, response.data)
// }
