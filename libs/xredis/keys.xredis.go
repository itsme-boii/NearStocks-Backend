package xredis

import (
	"fmt"
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"strings"
)

// This is a key for zset that stores all the prices levels for a market-party-side in sorted order
// BUY orders will have DECR direction
// SELL orders will have INCR direction
func GetOrderbookKey(marketId uint, party ctypes.Party, side ctypes.OrderSide) string {
	return fmt.Sprintf("LOGX_ORDERBOOK_LEVEL:%v:%v:%v", marketId, party, side)
}

// This is a key for zset that stores all trigger order prices in sorted order according to the trigger direction
// TP-BUY | SL-SELL will have DECR direction
// TP-SELL | SL-BUY will have INCR direction
func GetTriggerOrderbookKey(marketId uint, party ctypes.Party, direction ctypes.TriggerDirection) string {
	return fmt.Sprintf("LOGX_TRIG_ORDERBOOK_LEVEL:%v:%v:%v", marketId, party, direction)
}

// This is a key for hashset that stores the total quantity at each price level for a market-party-side
func GetOrderbookQtyKey(marketId uint, party ctypes.Party, side ctypes.OrderSide) string {
	return fmt.Sprintf("LOGX_ORDERBOOK_LEVEL_QTY:%v:%v:%v", marketId, party, side)
}

// This is a key for hashset that stores the total quantity at each price level for a market-party-direction
func GetTriggerOrderbookQtyKey(marketId uint, party ctypes.Party, direction ctypes.TriggerDirection) string {
	return fmt.Sprintf("LOGX_TRIG_ORDERBOOK_LEVEL_QTY:%v:%v:%v", marketId, party, direction)
}

// This is the field for the hashset GetTriggerOrderbookQtyKey
func GetTriggerOrderbookQtyField(priceQuantum uint64) string {
	return fmt.Sprintf("%v", priceQuantum)
}

// This is a key for zset that stores all the orders ids at a price level for a market-party-side in sorted order of timestamp
func GetOrdersAtLvlKey(marketId uint, party ctypes.Party, side ctypes.OrderSide, priceQuantum uint64) string {
	return fmt.Sprintf("LOGX_LEVEL_ORDERS:%v:%v:%v:%v", marketId, party, side, priceQuantum)
}

// This is a key for zset that stores all the trigger orders ids at a price level for a market-party-direction in sorted order of timestamp
func GetTriggerOrdersAtLvlKey(marketId uint, party ctypes.Party, direction ctypes.TriggerDirection, priceQuantum uint64) string {
	return fmt.Sprintf("LOGX_TRIG_LEVEL_ORDERS:%v:%v:%v:%v", marketId, party, direction, priceQuantum)
}

// This is a key for hashset that stores the details of an order | This will be used for all types of orders
func GetOrderDetailsKey(orderId uint) string {
	return fmt.Sprintf("LOGX_ORDER_DETAILS:%v", orderId)
}

func GetBalanceKey(subaccountID string) string {
	return fmt.Sprintf("v0_balance_%s", subaccountID)
}

func GetPreMarketBalanceKey(subaccountID string) string {
	return fmt.Sprintf("v0_pre_market_balance_%s", subaccountID)
}

func GetSyntheticSpotBalanceKey(subaccountID string) string {
	return fmt.Sprintf("v0_syn_spot_balance_%s", subaccountID)
}

func GetBalanceField(productId uint32) string {
	return fmt.Sprintf("%d", productId)
}

func GetFundingRateKey(symbol string) string {
	return fmt.Sprintf(symbol + "_funding_rate")
}

func GetCumulativeFundingRateKey(symbol string) string {
	// Add -USD to the symbol if it is not already present
	if !strings.HasSuffix(symbol, "-USD") && !strings.HasSuffix(symbol, "-USDC") {
		symbol += "-USD"
	}

	return fmt.Sprintf(symbol + "_cumulative_funding_rate")
}

func GetCumulativeEarningRateKey() string {
	return "cumulative_earning_rate"
}

func GetBalanceLockerKey(subaccountID string) string {
	return fmt.Sprintf("LOCK_BALANCE_%s", subaccountID)
}

func GetBalanceLockerField(entity ctypes.LockerEntity, entityId string) string {
	return fmt.Sprintf("%s_%s", entity, entityId)
}

func GetBalanceLockerValue(productId uint32, amountStr string) string {
	return fmt.Sprintf("%d_%s", productId, amountStr)
}

func GetOIKey() string {
	return "NETWORK_OI"
}

func GetNonceKey(subaccountID string) string {
	return fmt.Sprintf("%s_nonce", subaccountID)
}

func GetAveragePremiumKey(market string) string {
	return fmt.Sprintf(market + "_average_premium")
}

func GetPauseSequencerKey() string {
	return "PAUSE_SEQUENCER"
}
func GetPauseBatchingCronKey() string {
	return "PAUSE_BATCHING_CRON"
}
func GetMarketSpreadKey() string {
	return "MARKET_SPREADS"
}

func GetBalanceRedisLockKey(subaccountID string) string {
	return fmt.Sprintf("balance-lock-%s", subaccountID)
}

func GetPreMarketBalanceLockKey(subaccountID string) string {
	return fmt.Sprintf("pre-market-balance-lock-%s", subaccountID)
}

func GetSyntheticSpotBalanceLockKey(subaccountID string) string {
	return fmt.Sprintf("syn-spot-balance-lock-%s", subaccountID)
}

func GetNetworkOracleKey(symbol string) string {
	return fmt.Sprintf("LOGX_DARKORACLE_SERVICE_" + symbol)
}

func GetLiquidationEngineLockKey() string {
	return "LIQUIDATION_ENGINE_LOCK"
}
func GetAMMLockKey() string {
	return "AMM_LOCK"
}
func GetNonceLockKey(subaccountHex string) string {
	return fmt.Sprintf("NONCE_LOCK_%s", subaccountHex)
}

func GetWithdrawLockKey(subaccountHex string) string {
	return fmt.Sprintf("v0_withdraw_%s", subaccountHex)
}

func GetLiquidationLastSubaccountGIDKey() string {
	return "LIQUIDATION_LAST_SUBACCOUNT_GID"
}

func GetLiquidationSubpoolKey(subpool string) string {
	return fmt.Sprintf("LIQUIDATION_SUBPOOL_%s", subpool)
}

// This is a key for hashset
func GetSubacountPauseSequencerKey() string {
	return "SUBACCOUNT_PAUSE_SEQUENCER"
}

func GetSubacountPauseClaimLogxKey() string {
	return "SUBACCOUNT_PAUSE_CLAIM_LOGX_FLOW"
}
func GetSubacountBonusfeeKey() string {
	return "SUBACCOUNT_BONUS_FEE_UTILIZED"
}

// This is a field within the hashset GetSubacountPauseSequencerKey
func GetSubaccountLvlSequencerField(subaccountIDHex string) string {
	return subaccountIDHex
}

func GetSubaccountClaimFlowPausedField(subaccountIDHex string) string {
	return subaccountIDHex
}

func GetSubaccountBonusFeeField(subaccountIDHex string) string {
	return subaccountIDHex
}

// This lock is acquired when:
// The funding rate is being updated
// Liquidation is being processed
func GetFundingRateLockKey() string {
	return "FUNDING_RATE_LOCK"
}

func GetWhitelistHashKey() string {
	return "WHITELIST"
}

func GetWhitelistSubaccountField(subaccountIDHex string) string {
	return subaccountIDHex
}

func GetLiquidationStatsKey() string {
	return "LIQUIDATION_STATS"
}

func GetLiquidationStatsField(key ctypes.LiquidationStatsField) string {
	return string(key)
}

func GetInsuranceUtilisationPerDayHashKey() string {
	return "INSURANCE_UTILISATION_PER_DAY"
}

func GetInsuranceUtilisationPerDayField(date string) string {
	return date
}

func GetTxnBalanceUpdateKey(txnCounter uint) string {
	return fmt.Sprintf("TXN_BALANCE_UPDATE_%d", txnCounter)
}
func GetTotalLongPositionKey(productID string) string {
	return fmt.Sprintf("TOTAL_LONG_POSITION_%s", productID)
}

func GetTotalShortPositionKey(productID string) string {
	return fmt.Sprintf("TOTAL_SHORT_POSITION_%s", productID)
}
func GetTotalOILockKey(productID string) string {
	return fmt.Sprintf("TOTAL_PRODUCT_OI_LOCK_%s", productID)
}
func GetMarketLongOICap(productID string) string {
	return fmt.Sprintf("MARKET_LEVEL_LONG_OI_CAP_%s", productID)
}
func GetMarketShortOICap(productID string) string {
	return fmt.Sprintf("MARKET_LEVEL_SHORT_OI_CAP_%s", productID)
}
func GetSubaccountOICap() string {
	return "SUBACCOUNT_LEVEL_OI_CAP"
}
func GetSubaccountMarketCap(productID string) string {
	return fmt.Sprintf("SUBACCOUNT_MARKET_CAP_%s", productID)
}
func GetGlobalNegativeOICap() string {
	return "GLOBAL_OI_CAP"
}

func GetOrderLockKey(orderId uint) string {
	return fmt.Sprintf("ORDER_LOCK_%d", orderId)
}

// NOTE: Result of this will be in float string
func GetApyEarningFactorKey() string {
	return "APY_EARNING_FRACTION"
}

// What's the difference between this and GetApyEarningFactorKey?
// This stores the current minutes' APY earning factor and is used to calculate transient rewards for the stakers
// This helps us to change apy earning fraction anytime without worrying about if it messes up any calculations
func GetCurrentApyEarningFactorKey() string {
	return "CURRENT_APY_EARNING_FRACTION"
}

func GetSourceChainTrackerLastBlock() string {
	return "SOURCE_CHAIN_TRACKER_LAST_BLOCK"
}
func GetDestinationChainTrackerLastBlock() string {
	return "DESTINATION_CHAIN_TRACKER_LAST_BLOCK"
}

// This is a hashset key within DestinationChainBackFill
func GetDestinationChainBackfillKey(productId uint32) string {
	return fmt.Sprintf("DESTINATION_CHAIN_BACK_FILL_%d", productId)
}

func GetDestinationChainBackfillStartField() string {
	return "START_BLOCK"
}

func GetDestinationChainBackfillEndField() string {
	return "END_BLOCK"
}

func GetSourceChainTrackerLastBlockField(productId uint32) string {
	return fmt.Sprintf("%d", productId)
}

func GetFinaliseDepositKey() string {
	return "FINALISE_DEPOSIT"
}

func GetTotalArbRewardsAccumulates() string {
	return "TOTAL_ARB_REWARDS_ACCUMULATED"
}

func GetTotalLogxRewardsAccumulated() string {
	return "TOTAL_LOGX_REWARDS_ACCUMULATED"
}

func GetLastCachedUserDepositKey() string {
	return "CACHED_USERS_DEPOSITS"
}

func GetReferralRebateMapping() string {
	return "REFERRAL_REBATE_MAPPING"
}
func GetAffiliateReferralCodesKey() string {
	return "AFFILIATE_REFERRAL_CODES"
}
func GetReferralCronStartIndex() string {
	return "REFERRAL_CRON_START_INDEX"
}

func GetSettlePnlSubAccounts() string {
	return "SETTLE_PNL_SUBACCOUNT"
}

func GetSubaccountTrackingKey() string {
	return "TRACK_SUBACCOUNTS"
}

func GetSubaccountTrackingField(subaccountIdHex string) string {
	return subaccountIdHex
}

func GetUnrealisedPnlFillDbIdKey() string {
	return "UNREALISED_PNL_FILLS_DB_ID"
}

func GetPositionsMapKey() string {
	return "POSITIONS_MAP"
}

func GetExternalFundingRates() string {
	return "EXTERNAL_FUNDING_RATE"
}

func GetOptionsPayoutKey() string {
	return "OPTIONS_PAYOUT_PERCENTAGE"
}

func GetOptionsFeesKey() string {
	return "OPTIONS_FEES_PERCENTAGE"
}

func GetOptionsProductIDField(productId uint32, interval uint32) string {
	return fmt.Sprintf("%d:%d", productId, interval)
}

func GetOptionsCloseJobsKey() string {
	return "OPTIONS_CLOSE_JOBS"
}

func GetOptionsLockKey() string {
	return "OPTIONS_CLOSE_JOBS_LOCK_KEY"
}

func GetOptionsEnabledKey() string {
	return "OPTIONS_ENABLED"
}

func GetOptionsMinimumAmountKey() string {
	return "OPTIONS_MINIMUM_AMOUNT"
}

func GetOptionsMaximumAmountKey() string {
	return "OPTIONS_MAXIMUM_AMOUNT"
}

func GetPreMarketLockKey(productId uint32) string {
	return fmt.Sprintf("PRE_MARKET_LOCK_%d", productId)
}

// GetOptionBetLockKey returns a Redis lock key for a specific option bet
func GetOptionBetLockKey(betId uint) string {
	return fmt.Sprintf("OPTIONS_BET_LOCK_%d", betId)
}

func GetPreMarketPricingKey() string {
	return "PRE_MARKET_PRICING"
}

func GetPreMarketPricingField(productId uint32) string {
	return fmt.Sprintf("%d", productId)
}

func GetPreMarketFeesKey() string {
	return "PRE_MARKET_FEES"
}

func GetPreMarketAdminKey() string {
	return "PRE_MARKET_ADMIN_APIS"
}

func GetSyntheticSpotsEnabledKey() string {
	return "SYNTHETIC_SPOTS_ENABLED"
}

func GetSyntheticSpotsFeesKey() string {
	return "SYNTHETIC_SPOTS_FEES"
}

func GetSyntheticSpotsSlippageKey() string {
	return "SYNTHETIC_SPOTS_SLIPPAGE"
}

func GetSyntheticSpotsNetPositionKey() string {
	return "SYNTHETIC_SPOTS_NET_POSITION"
}

func GetSyntheticSpotsPositionCapKey() string {
	return "SYNTHETIC_SPOTS_POSITION_CAP"
}

func GetBatchingLockKey() string {
	return "BATCHING_LOCK"
}

func GetProcessWithdrawLockKey() string {
	return "WITHDRAW_CRON_LOCK"
}

func GetWithdrawalMessageKey(messageID string) string {
	return fmt.Sprintf("WITHDRAWAL_MESSAGE:%s", messageID)
}

func GetGovernanceVoteLockKey(subaccountID string, proposalID uint) string {
	return fmt.Sprintf("GOVERNANCE_VOTE_LOCK:%s:%d", subaccountID, proposalID)
}

func GetCancelAllAndPlaceOrderLockKey(marketId uint32, userAddress string) string {
	return fmt.Sprintf("CANCEL_ALL_AND_PLACE_ORDER_LOCK_%d_%s", marketId, userAddress)
}

func GetMatchingEngineLockKey() string {
	return "MATCHING_ENGINE_LOCK"
}

func GetDepositLogXChainBackfillStartBlockKey() string {
	return "DEPOSIT_LOGX_CHAIN_BACK_FILL_START_BLOCK"
}

func GetDepositLogXChainBackfillEndBlockKey() string {
	return "DEPOSIT_LOGX_CHAIN_BACK_FILL_END_BLOCK"
}

func GetAMMFundingCapsKey() string {
	return "AMM_FUNDING_CAPS"
}

func GetAMMFundingCapsField(marketId uint) string {
	return fmt.Sprintf("%d", marketId)
}

func GetAMMSizesKey() string {
	return "AMM_SIZES"
}

func GetAMMSizesField(marketId uint) string {
	return fmt.Sprintf("%d", marketId)
}

func GetAMMSlippageKey() string {
	return "AMM_SLIPPAGE"
}

func GetAMMSlippageField(marketId uint) string {
	return fmt.Sprintf("%d", marketId)
}

func GetAMMFundingRateFactorKey() string {
	return "AMM_FUNDING_RATE_FACTOR"
}

func GetStopInternalSubaccountWithdrawalKey() string {
	return "STOP_INTERNAL_SUBACCOUNT_WITHDRAWAL"
}
func GetMarketOrdersEnabledKey() string {
	return "MARKET_ORDERS_ENABLED"
}

// Defillama stats configuration keys
func GetDefillamaStatsEnabledKey() string {
	return "DEFILLAMA_STATS_ENABLED"
}

func GetDefillamaStatsBaseVolumeKey() string {
	return "DEFILLAMA_STATS_BASE_VOLUME"
}

func GetDefillamaStatsGrowthRateKey() string {
	return "DEFILLAMA_STATS_GROWTH_RATE"
}

func GetDefillamaStatsVolatilityKey() string {
	return "DEFILLAMA_STATS_VOLATILITY"
}
func GetDefillamaStatsBase24hVolumeKey() string {
	return "DEFILLAMA_STATS_BASE_24H_VOLUME"
}

func GetDefillamaStatsLastUpdateKey() string {
	return "DEFILLAMA_STATS_LAST_UPDATE"
}

// Dashboard broker-specific keys
func GetDashboardBrokerKey(brokerId uint, statType string) string {
	return fmt.Sprintf("dashboard-broker%d-%s", brokerId, statType)
}

func GetDashboardBrokerTotalTradesKey(brokerId uint) string {
	return fmt.Sprintf("dashboard-broker%d-totalTrades", brokerId)
}

func GetDashboardBrokerDailyTradesKey(brokerId uint) string {
	return fmt.Sprintf("dashboard-broker%d-dailyTrades", brokerId)
}

func GetDashboardBrokerDailyTotalVolumesKey(brokerId uint) string {
	return fmt.Sprintf("dashboard-broker%d-dailyTotalVolumes", brokerId)
}

func GetDashboardBrokerTotalUsersKey(brokerId uint) string {
	return fmt.Sprintf("dashboard-broker%d-totalUsers", brokerId)
}

func GetDashboardBrokerDailyActiveUsersKey(brokerId uint) string {
	return fmt.Sprintf("dashboard-broker%d-dailyActiveUsers", brokerId)
}

func GetDashboardBrokerWeeklyActiveUsersKey(brokerId uint) string {
	return fmt.Sprintf("dashboard-broker%d-weeklyActiveUsers", brokerId)
}

func GetDashboardBrokerNewUsersDailyKey(brokerId uint) string {
	return fmt.Sprintf("dashboard-broker%d-newUsersDaily", brokerId)
}

func GetDashboardBrokerUser24HrDataKey(brokerId uint) string {
	return fmt.Sprintf("dashboard-broker%d-user24HrData", brokerId)
}

func GetDashboardBrokerLast7dayUserCountKey(brokerId uint) string {
	return fmt.Sprintf("dashboard-broker%d-last7dayUserCount", brokerId)
}

func GetDashboardBrokerCumulative30DayAgoTradesKey(brokerId uint) string {
	return fmt.Sprintf("dashboard-broker%d-cumulative30DayAgoTrades", brokerId)
}

func GetDashboardBrokerCumulative30DayAgoFeesKey(brokerId uint) string {
	return fmt.Sprintf("dashboard-broker%d-cumulative30DayAgoFees", brokerId)
}

func GetDashboardBrokerCumulative30DayAgoVolumeKey(brokerId uint) string {
	return fmt.Sprintf("dashboard-broker%d-cumulative30DayAgoVolume", brokerId)
}

func GetDashboardBrokerCumulative30DayAgoUsersKey(brokerId uint) string {
	return fmt.Sprintf("dashboard-broker%d-cumulative30DayAgoUsers", brokerId)
}

func GetDashboardBrokerCumulativeTotalFeesKey(brokerId uint) string {
	return fmt.Sprintf("dashboard-broker%d-cumulativeTotalFees", brokerId)
}

func GetDashboardBrokerDailyPnLKey(brokerId uint) string {
	return fmt.Sprintf("dashboard-broker%d-dailyPnL", brokerId)
}

func GetDashboardBrokerDailyFeesKey(brokerId uint) string {
	return fmt.Sprintf("dashboard-broker%d-dailyFees", brokerId)
}

func GetDashboardBrokerDailyVolumesKey(brokerId uint) string {
	return fmt.Sprintf("dashboard-broker%d-dailyVolumes", brokerId)
}

// Dummy data cache keys for date-specific caching
func GetDummyDataCacheKey(redisKey string, brokerId uint, date string) string {
	return fmt.Sprintf("DUMMY_DATA_%s_BROKER_%d_DATE_%s", redisKey, brokerId, date)
}

// Key for storing flagged subaccounts (used by flag user cron and withdrawal blocker)
func GetFlaggedSubaccountsKey() string {
	return "FLAGGED_SUBACCOUNTS"
}

// Key for storing flag user threshold value
func GetFlagUserThresholdKey() string {
	return "FLAG_USER_THRESHOLD"
}

func GetSaleDataKey() string {
	return "SALE_DATA"
}

// Internal dashboard keys
func GetStatsCacheKey(brokerId uint, statType string) string {
	return fmt.Sprintf("STATS_CACHE:%d:%s", brokerId, statType)
}

func GetStatsCacheTotalVolumeKey(brokerId uint) string {
	return GetStatsCacheKey(brokerId, "totalVolume")
}

func GetStatsCache24hRealisedPnlKey(brokerId uint) string {
	return GetStatsCacheKey(brokerId, "24hRealisedPnl")
}

func GetStatsCache24hTradingFeeKey(brokerId uint) string {
	return GetStatsCacheKey(brokerId, "24hTradingFee")
}

func GetStatsCache24hFundingFeeKey(brokerId uint) string {
	return GetStatsCacheKey(brokerId, "24hFundingFee")
}

func GetStatsCacheTradingFeeKey(brokerId uint) string {
	return GetStatsCacheKey(brokerId, "tradingFee")
}

func GetStatsCacheFundingFeeKey(brokerId uint) string {
	return GetStatsCacheKey(brokerId, "fundingFee")
}

func GetStatsCacheUserRealisedPnlKey(brokerId uint) string {
	return GetStatsCacheKey(brokerId, "userRealisedPnl")
}

func GetStatsCacheDailyActiveTradersKey(brokerId uint) string {
	return GetStatsCacheKey(brokerId, "dailyActiveTraders")
}

func GetStatsCacheLiquidationStatsKey(brokerId uint) string {
	return GetStatsCacheKey(brokerId, "liquidationStats")
}

func GetKolAllocationHashKey() string {
	return "KOL_ALLOCATIONS"
}
