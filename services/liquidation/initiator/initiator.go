package initiator

import (
	"context"
	"fmt"
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/appstate"
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/metric"
	"github/eugenix-io/logx-inf-backend/libs/subaccount"
	"github/eugenix-io/logx-inf-backend/libs/subaccountTypes"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"github/eugenix-io/logx-inf-backend/services/liquidation/liquidator"
	"github/eugenix-io/logx-inf-backend/services/liquidation/pool"
	"github/eugenix-io/logx-inf-backend/services/liquidation/stats"
	"github/eugenix-io/logx-inf-backend/services/liquidation/types"
	"github/eugenix-io/logx-inf-backend/xclient"

	"log"
	"math/big"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

/** CORE LOGIC:

Useful Terms restricted to liquidation:
	- Total equity: Total spot balance + total unrealised pnl on perps + total unrealised funding fee
	- Total spot balance: Sum of all the spot balances (Total collateral can also be used as a term for this)


Important configs:

	Data structures:
	- Spot Balance
		- LongAmountx18 -> + value
		- ShortAmountx18 -> - value
		- LongQuoteBalancex18 -> - value : The way to understand the sign is that whenever we are long we are buying the base asset so, quote balance will be negative
		- ShortQuoteBalancex18 -> + value
		- VQuoteBalancex18 -> LongQuoteBalancex18 + ShortQuoteBalancex18 + Funding payment



Data fetched from external source:

	- We fetch all the subaccounts with there spot and perp balances. These will be used for calculating total spot balance  + total perp balance
	- Fetch all the oracle prices. Oracle price will be used for calculating unrealised pnl.
	- Fetch all the perpetual markets. This will be used for calculating required maintenance margin & initial margin checks.
	- Fetch current funding rates. This will be used for calculating unrealised funding.
	- This is all we need to do the calculation for liquidation

Liquidation Calculations:


Data flow in the system:

	- We will have 5 subpools: NewSubaccounts, HealthySubaccounts, BelowInitialMarginSubaccounts, BelowMaintanenceMarginSubaccounts, NegativeBalanceSubaccounts
	- NewSubaccounts:
		- Store last subaccounts fetch time in the system. We will always fetch the subaccounts after (this time - 5 minutes) to avoid unexpected misses. Future: We can also use last created at time of subaccount.
		- Add all the newly fetched subaccounts to this pool.
		- TaskRunner will run every 30 seconds on new accounts to check if they are liquidatable.
		- These will be moved to HealthySubaccounts after the first liquidation check.

	- HealthySubaccounts:
		- These are the accounts which are not liquidatable.
		- TaskRunner will run every 30 seconds on healthySubaccounts to check if they are liquidatable (with negative net equity or positive net equity) or below initial margin.
		- If they are liquidatable (below maintanence margin):
			- If total equity is negative, we will move them to NegativeBalanceSubaccounts
			- If total equity is positive, we will move them to BelowInitialMarginSubaccounts
		- If they are below initial margin, we will move them to BelowInitialMarginSubaccounts.
		- If they are healthy, we will keep them in this pool.

	- BelowInitialMarginSubaccounts:
		- These are the accounts which are most likely to be liquidated.
		- Access to open new positions on these accounts will be restricted.
		- Allowed action - Deposit more collateral. Close existing positions.
		- TaskRunner will run every 10 seconds on these accounts to check if they are liquidatable.
		- If they are liquidatable, we will move them with same logic we use for HealthySubaccounts.
		- If they are healthy, we will move them to HealthySubaccounts and remove the restriction on opening new positions.

	- BelowMaintanenceMarginSubaccounts:
		- These are the accounts which are below maintanence margin but have positive equity.
		- These accounts won't have access to open new positions or close existing positions.
		- Only allowed action - Deposit more collateral.
		- Liquidator will run every 5 seconds on these accounts to execute liquidations orders on them.
		- Before placing the liquidation order, we will check if the account is still below maintanence margin.
		- If they are healthy, we will move them to HealthySubaccounts.
		- If they are below initial margin, we will move them to BelowInitialMarginSubaccounts.

	- NegativeBalanceSubaccounts:
		- These are the accounts which have negative equity.
		- These accounts should be liquidated on priority so, the losses are minimum.
		- Liquidator will run every 2 second on these subaccounts checks for liquidation and execute liquidation orders for them.
		- Same restrictions as BelowMaintanenceMarginSubaccounts.
		- If they are healthy (although this is highly unlikely), we will move them to HealthySubaccounts.
		- If they are below initial margin, we will move them to BelowInitialMarginSubaccounts.
		- Else we will liquidate them.
		- TODO: Check if we really need to do any check for liquidation for negative net equity accounts. I feel we can directly liquidate them.

	- Liquidator
		- Cancel all the open orders on the subaccount.
		- Place liquidation orders on behalf of the subaccounts.
		- Matching for liquidation orders can be done at similar to market orders.

**/

type TaskRunner interface {
	RunLiquidationJob1() error
	RunLiquidationJob2() error
	RunLiquidationJob3() error
	RunLiquidationJob4() error
	RunLiquidationJob5() error
	GetAppState(cfg ...appstate.AppStateCacheConfig) (map[string]ctypes.OraclePrice, map[uint]subaccountTypes.PerpetualMarket, map[string]*big.Int, error)
	GetAllSubaccountsAfterLastCheckpoint() ([]subaccountTypes.SubaccountBalances, uint, error)
	CheckSubaccountCollateralization(
		unsettledSubaccount subaccountTypes.SubaccountBalances,
		oraclePricesMap map[string]ctypes.OraclePrice,
		perpetualMarketsMap map[uint]subaccountTypes.PerpetualMarket,
		cumulativeFundingRateMap map[string]*big.Int,
	) (isBelowInitialMargin bool, isBelowMaintenanceMargin bool, requiresInsurance bool, isBelowLiquidationMargin bool)
	RunAllJobs(ctx context.Context, wg *sync.WaitGroup)
	FetchCurrentStateFromRedis() error
	WriteCurrentStateToRedis()
	WriteStats() error
}

type TaskRunnerImpl struct {
	subaccountBalance      subaccount.SubaccountBalance
	pool                   pool.SubaccountPool
	lastSubaccountFetchGID uint
	appState               appstate.AppState
}

// Validate that TaskRunnerImpl implements TaskRunner
var _ TaskRunner = &TaskRunnerImpl{}

func NewTaskRunner() TaskRunner {
	return &TaskRunnerImpl{
		subaccountBalance:      subaccount.NewSubaccountBalanceImpl(),
		pool:                   pool.NewSubaccountPool(),
		lastSubaccountFetchGID: 0,
		appState:               appstate.NewAppState(),
	}
}

// Get time intervals from env and then run all the jobs
func (t *TaskRunnerImpl) RunAllJobs(ctx context.Context, wg *sync.WaitGroup) {
	job1IntervalSecs, err1 := strconv.ParseUint(os.Getenv("JOB1_INTERVAL_SEC"), 10, 64)
	job2IntervalSecs, err2 := strconv.ParseUint(os.Getenv("JOB2_INTERVAL_SEC"), 10, 64)
	job3IntervalSecs, err3 := strconv.ParseUint(os.Getenv("JOB3_INTERVAL_SEC"), 10, 64)
	job4IntervalSecs, err4 := strconv.ParseUint(os.Getenv("JOB4_INTERVAL_SEC"), 10, 64)
	job5IntervalSecs, err5 := strconv.ParseUint(os.Getenv("JOB5_INTERVAL_SEC"), 10, 64)
	job6IntervalSecs, err6 := strconv.ParseUint(os.Getenv("JOB6_INTERVAL_SEC"), 10, 64)
	jobWriteStateIntervalSecs, err7 := strconv.ParseUint(os.Getenv("JOB_WRITE_STATS_INTERVAL_SEC"), 10, 64)
	jobUpdateFundsStatsSecs, err8 := strconv.ParseUint(os.Getenv("JOB_UPDATE_FUNDS_STATS_SEC"), 10, 64)
	jobBurnKromaFundsSecs, err9 := strconv.ParseUint(os.Getenv("JOB_BURN_KROMA_FUNDS_SEC"), 10, 64)
	if err1 != nil || err2 != nil || err3 != nil || err4 != nil || err5 != nil || err6 != nil || err7 != nil || err8 != nil || err9 != nil {
		log.Fatalf("Error parsing job interval seconds: %v, %v, %v, %v, %v, %v, %v, %v, %v", err1, err2, err3, err4, err5, err6, err7, err8, err9)
	}
	if jobUpdateFundsStatsSecs == 0 || job1IntervalSecs == 0 || job2IntervalSecs == 0 || job3IntervalSecs == 0 || job4IntervalSecs == 0 || job5IntervalSecs == 0 || job6IntervalSecs == 0 || jobWriteStateIntervalSecs == 0 || jobBurnKromaFundsSecs == 0 {
		log.Fatalf("Job interval seconds can't be set as 0 seconds: %v, %v, %v, %v, %v, %v, %v, %v, %v", job1IntervalSecs, job2IntervalSecs, job3IntervalSecs, job4IntervalSecs, job5IntervalSecs, job6IntervalSecs, jobWriteStateIntervalSecs, jobUpdateFundsStatsSecs, jobBurnKromaFundsSecs)
	}
	cutils.RunInfiniteJobWithTicker(ctx, wg, t.RunLiquidationJob1, time.Duration(job1IntervalSecs)*time.Second, "LIQUIDATION JOB 1")
	cutils.RunInfiniteJobWithTicker(ctx, wg, t.RunLiquidationJob2, time.Duration(job2IntervalSecs)*time.Second, "LIQUIDATION JOB 2")
	cutils.RunInfiniteJobWithTicker(ctx, wg, t.RunLiquidationJob3, time.Duration(job3IntervalSecs)*time.Second, "LIQUIDATION JOB 3")
	cutils.RunInfiniteJobWithTicker(ctx, wg, t.RunLiquidationJob4, time.Duration(job4IntervalSecs)*time.Second, "LIQUIDATION JOB 4")
	cutils.RunInfiniteJobWithTicker(ctx, wg, t.RunLiquidationJob5, time.Duration(job5IntervalSecs)*time.Second, "LIQUIDATION JOB 5")
	cutils.RunInfiniteJobWithTicker(ctx, wg, t.RunLiquidationJob6, time.Duration(job6IntervalSecs)*time.Second, "LIQUIDATION JOB 6")
	cutils.RunInfiniteJobWithTicker(ctx, wg, t.WriteStats, time.Duration(jobWriteStateIntervalSecs)*time.Second, "WRITE STATS JOB")
	cutils.RunInfiniteJobWithTicker(ctx, wg, t.RunFundsMonitorJob, time.Duration(jobUpdateFundsStatsSecs)*time.Second, "UPDATE FUNDS STATS JOB")
	cutils.RunInfiniteJobWithTicker(ctx, wg, t.RunKromaBurnJob, time.Duration(jobBurnKromaFundsSecs)*time.Second, "KROMA BURN JOB")
}

// This process will fetch all the new subaccounts and add them to the new subaccounts pool
func (t *TaskRunnerImpl) RunLiquidationJob1() error {
	// Implement the logic to run the liquidation job
	defer cutils.LogTime(time.Now(), metric.LIQUIDATION_RUN_JOB)

	// startTime := time.Now()
	newSubaccounts, lastFetchedGID, err := t.GetAllSubaccountsAfterLastCheckpoint()
	if err != nil {
		xlog.Errorf("Error getting new subaccounts: %v", err)
		return err
	}

	if len(newSubaccounts) == 0 {
		xlog.Infof("JOB-1: Last fetched GID for subaccounts: %d", lastFetchedGID)
		xlog.Infof("JOB-1: No new subaccounts fetched. Exiting this iteration of job early")
		return nil
	}

	t.pool.AddToNewSubaccountsPool(newSubaccounts)
	t.lastSubaccountFetchGID = lastFetchedGID

	xlog.Infof("JOB-1: Added %d new subaccounts to the new subaccounts pool", len(newSubaccounts))
	xlog.Infof("JOB-1: Last fetched GID for subaccounts: %d", lastFetchedGID)
	// t.lastCheckpointForSubaccountFetch = startTime

	return nil
}

func runGracefuly(array []string, f func([]string)) {
	limit := 20_000
	for i := 0; i < len(array); i += limit {
		end := min(i+limit, len(array))
		f(array[i:end])

		if end < len(array) {
			xlog.Infof("Fetched %v balances....Sleeping for 1 seconds\n", i)
			time.Sleep(1 * time.Second)
		}
	}
}

// Iterate through all the subaccounts of new pool and get liquidation result
func (t *TaskRunnerImpl) RunLiquidationJob2() error {
	defer cutils.LogTime(time.Now(), metric.LIQUIDATION_RUN_JOB_2)

	oraclePricesMap, perpetualMarketsMap, cumulativeFundingRateMap, err := t.GetAppState(appstate.GetAppConfig(10*time.Second, 5*time.Minute, 24*time.Hour))
	if err != nil {
		xlog.Errorf("This is a critical error. We can't proceed without the app state")
		return nil
	}

	var transferStats types.SubpoolTransferStats

	newSubaccounts := t.pool.GetAllNewSubaccounts()
	subaccountIds := cutils.MapKeys(newSubaccounts)
	runGracefuly(subaccountIds, func(subaccountBatchIds []string) {
		updatedNewSubaccounts := t.subaccountBalance.MustGetSubaccountBalancesFromIds(subaccountBatchIds)
		for _, subaccount := range updatedNewSubaccounts {
			isBelowInitialMargin, isBelowMaintenanceMargin, requiresInsurance, _ := t.CheckSubaccountCollateralization(subaccount, oraclePricesMap, perpetualMarketsMap, cumulativeFundingRateMap)
			// Hack: We don't want to save the balance data for the new subaccounts
			subaccount := newSubaccounts[subaccount.SubaccountId]
			if requiresInsurance {
				transferStats.NegativeBalanceSubaccounts++
				t.pool.MoveSubaccountFromNewTo(subaccount, pool.NegativeBalanceSubaccountsSubpool)
			} else if isBelowMaintenanceMargin {
				transferStats.BelowMaintenanceMarginSubaccounts++
				t.pool.MoveSubaccountFromNewTo(subaccount, pool.BelowMaintanenceMarginSubaccountsSubpool)
			} else if isBelowInitialMargin {
				transferStats.BelowInitialMarginSubaccounts++
				t.pool.MoveSubaccountFromNewTo(subaccount, pool.BelowInitialMarginSubaccountsSubpool)
			} else {
				transferStats.HealthySubaccounts++
				t.pool.MoveSubaccountFromNewTo(subaccount, pool.HealthySubaccountsSubpool)
			}
		}
	})

	xlog.Infof("JOB-2: Transfers from NewSubaccounts completed: Starting count for new accounts: %v | Transfer stats: %+v", len(newSubaccounts), transferStats)
	return nil
}

// Iterate through all the healthy subaccounts.
// Get liquidation results and move them to respective subpools
func (t *TaskRunnerImpl) RunLiquidationJob3() error {
	defer cutils.LogTime(time.Now(), metric.LIQUIDATION_RUN_JOB_3)

	oraclePricesMap, perpetualMarketsMap, cumulativeFundingRateMap, err := t.GetAppState(appstate.GetAppConfig(10*time.Second, 5*time.Minute, 24*time.Hour))
	if err != nil {
		xlog.Errorf("This is a critical error. We can't proceed without the app state")
		return nil
	}

	var transferStats types.SubpoolTransferStats

	healthySubaccounts := t.pool.GetAllHealthySubaccounts()

	subaccountIds := cutils.MapKeys(healthySubaccounts)
	runGracefuly(subaccountIds, func(subaccountBatchIds []string) {
		updatedHealthySubaccounts := t.subaccountBalance.MustGetSubaccountBalancesFromIds(subaccountBatchIds)

		for _, subaccount := range updatedHealthySubaccounts {
			isBelowInitialMargin, isBelowMaintenanceMargin, requiresInsurance, _ := t.CheckSubaccountCollateralization(subaccount, oraclePricesMap, perpetualMarketsMap, cumulativeFundingRateMap)
			// Hack: We don't want to save the balance data for the subaccounts
			subaccount := healthySubaccounts[subaccount.SubaccountId]
			if requiresInsurance {
				transferStats.NegativeBalanceSubaccounts++
				t.pool.MoveSubaccountFromHealthyTo(subaccount, pool.NegativeBalanceSubaccountsSubpool)
			} else if isBelowMaintenanceMargin {
				transferStats.BelowMaintenanceMarginSubaccounts++
				t.pool.MoveSubaccountFromHealthyTo(subaccount, pool.BelowMaintanenceMarginSubaccountsSubpool)
			} else if isBelowInitialMargin {
				transferStats.BelowInitialMarginSubaccounts++
				t.pool.MoveSubaccountFromHealthyTo(subaccount, pool.BelowInitialMarginSubaccountsSubpool)
			}
		}
	})

	xlog.Infof("JOB-3: Transfers from HealthySubaccounts completed: Starting count for healthy accounts: %v | Transfer stats: %+v", len(healthySubaccounts), transferStats)

	return nil
}

// Iterate through all the below initial margin subaccounts, and get liquidation results
// Move them to respective subpools
func (t *TaskRunnerImpl) RunLiquidationJob4() error {
	defer cutils.LogTime(time.Now(), metric.LIQUIDATION_RUN_JOB_4)

	oraclePricesMap, perpetualMarketsMap, cumulativeFundingRateMap, err := t.GetAppState(appstate.GetAppConfig(10*time.Second, 5*time.Minute, 24*time.Hour))
	if err != nil {
		xlog.Errorf("This is a critical error. We can't proceed without the app state")
		return err
	}

	var transferStats types.SubpoolTransferStats
	belowInitialMarginSubaccounts := t.pool.GetAllBelowInitialMarginSubaccounts()

	subaccountIds := cutils.MapKeys(belowInitialMarginSubaccounts)
	runGracefuly(subaccountIds, func(subaccountBatchIds []string) {
		updatedBelowInitialMarginSubaccounts := t.subaccountBalance.MustGetSubaccountBalancesFromIds(subaccountBatchIds)

		for _, subaccount := range updatedBelowInitialMarginSubaccounts {
			isBelowInitialMargin, isBelowMaintenanceMargin, requiresInsurance, _ := t.CheckSubaccountCollateralization(subaccount, oraclePricesMap, perpetualMarketsMap, cumulativeFundingRateMap)
			// Hack: We don't want to save the balance data for the subaccounts
			subaccount := belowInitialMarginSubaccounts[subaccount.SubaccountId]
			if requiresInsurance {
				transferStats.NegativeBalanceSubaccounts++
				t.pool.MoveSubaccountFromBelowInitialMarginTo(subaccount, pool.NegativeBalanceSubaccountsSubpool)
			} else if isBelowMaintenanceMargin {
				transferStats.BelowMaintenanceMarginSubaccounts++
				t.pool.MoveSubaccountFromBelowInitialMarginTo(subaccount, pool.BelowMaintanenceMarginSubaccountsSubpool)
			} else if isBelowInitialMargin {
				continue
			} else {
				transferStats.HealthySubaccounts++
				t.pool.MoveSubaccountFromBelowInitialMarginTo(subaccount, pool.HealthySubaccountsSubpool)
			}
		}
	})

	xlog.Infof("JOB-4: Transfers from BelowInitialMarginSubaccounts completed: Starting count for below initial margin accounts: %v | Transfer stats: %+v", len(belowInitialMarginSubaccounts), transferStats)

	return nil
}

// Default
var MAX_LIQUIDATIONS uint64 = 50
var MAX_INSURANCE uint64 = 50

// Iterate through all below maintenance margin subaccounts - get liquidation results and start liquidation
func (t *TaskRunnerImpl) RunLiquidationJob5() error {
	defer cutils.LogTime(time.Now(), metric.LIQUIDATION_RUN_JOB_5)

	max_liquidations := os.Getenv("MAX_LIQUIDATIONS")
	if max_liquidations != "" {
		MAX_LIQUIDATIONS, _ = strconv.ParseUint(max_liquidations, 10, 64)
	}
	job5IntervalSecs, _ := strconv.ParseUint(os.Getenv("JOB5_INTERVAL_SEC"), 10, 64)

	// Acquire lock for job interval minutes | This is to ensure that only one instance of the liquidation service is running
	lockTime := time.Duration(2*job5IntervalSecs) * time.Second
	retryDelay := time.Second * 10
	maxRetries := 10
	lockOptions := xredis.LockOptions{
		LockExpiry: &lockTime,
		MaxRetries: &maxRetries,
		RetryDelay: &retryDelay,
	}

	liquidationStats := types.LiquidationStats{}
	transferStats := types.SubpoolTransferStats{}
	belowMaintenanceMarginSubaccounts := t.pool.GetAllBelowMaintanenceMarginSubaccounts()

	if len(belowMaintenanceMarginSubaccounts) > 0 {
		xlog.Infof("JOB-5 Acquiring funding rate lock")
		_, err := xredis.WithRedisLock(xredis.GetFundingRateLockKey(), func() (*xredis.NOOP, error) {
			oraclePricesMap, perpetualMarketsMap, cumulativeFundingRateMap, err := t.GetAppState(appstate.GetAppConfig(2*time.Second, 0, 24*time.Hour))
			if err != nil {
				xlog.Errorf("This is a critical error. We can't proceed without the app state")
				return nil, err
			}

			xlog.Debugf("State params: oracle price map: %+v | perpetualMarketMap: %+v | cumulativeFundingRateMap: %+v", oraclePricesMap, perpetualMarketsMap, cumulativeFundingRateMap)

			for _, subaccount := range belowMaintenanceMarginSubaccounts {
				updatedSubaccount := t.subaccountBalance.MustGetSubaccountBalancesFromIds([]string{subaccount.SubaccountId})[0]
				isBelowInitialMargin, isBelowMaintenanceMargin, requiresInsurance, isBelowLiquidationMargin := t.CheckSubaccountCollateralization(updatedSubaccount, oraclePricesMap, perpetualMarketsMap, cumulativeFundingRateMap)
				// Liquidate the subaccount
				if requiresInsurance {
					transferStats.NegativeBalanceSubaccounts++
					t.pool.MoveSubaccountFromBelowMaintenanceMarginTo(subaccount, pool.NegativeBalanceSubaccountsSubpool)
				} else if isBelowLiquidationMargin {
					if liquidationStats.TotalRecorded() >= uint(MAX_LIQUIDATIONS) {
						xlog.Infof("JOB-5: Max liquidations reached. Exiting the job")
						break
					}
					xlog.Infof("Starting liquidation for subaccount: %s", updatedSubaccount.SubaccountId)
					_, fillStatus, err := liquidator.PlaceLiquidationOrder(updatedSubaccount, oraclePricesMap)
					liquidationStats.Record(fillStatus)
					if err != nil {
						xlog.Warnf("Error placing liquidation order for subaccount: %s. Error: %v", updatedSubaccount.SubaccountId, err)
						continue
					}
				} else if isBelowMaintenanceMargin {
					// Keep the subaccount in the same pool
					continue
				} else if isBelowInitialMargin {
					transferStats.BelowInitialMarginSubaccounts++
					t.pool.MoveSubaccountFromBelowMaintenanceMarginTo(subaccount, pool.BelowInitialMarginSubaccountsSubpool)
				} else {
					transferStats.HealthySubaccounts++
					t.pool.MoveSubaccountFromBelowMaintenanceMarginTo(subaccount, pool.HealthySubaccountsSubpool)
				}
			}
			return nil, nil
		}, lockOptions)

		xlog.Infof("Released funding rate lock")
		if err != nil {
			xlog.Errorf("JOB-5 Exiting due to error: %v", err)
			return err
		}
	}

	// We are also include transfered accounts in total skipped
	liquidationStats.TotalSkipped = uint(len(belowMaintenanceMarginSubaccounts)) - liquidationStats.TotalRecorded()
	xlog.Infof("JOB-5 b): Transfers from BelowMaintanenceMarginSubaccounts completed: Starting count for below maintanence margin accounts: %v | Transfer stats: %+v", len(belowMaintenanceMarginSubaccounts), transferStats)
	xlog.Infof("JOB-5: Liquidation stats: %+v", liquidationStats)

	if liquidationStats.AtleastOneRecorded() {
		increaseCountOnRedis(map[string]int64{
			string(ctypes.FULL_LIQUIDATION):    int64(liquidationStats.TotalFullyLiquidated),
			string(ctypes.PARTIAL_LIQUIDATION): int64(liquidationStats.TotalPartiallyLiquidated),
			string(ctypes.FAILED_LIQUIDATION):  int64(liquidationStats.TotalUnliquidated),
		})
	}

	// send discord alert
	if liquidationStats.AtleastOneRecorded() && xclient.GlobalDiscordClient != nil {
		xclient.GlobalDiscordClient.SendWebhookMessage(fmt.Sprintf("Liquidation stats: %+v", liquidationStats))
	}

	return nil
}

func increaseCountOnRedis(keyCountMap map[string]int64) {
	_, err := xredis.GetLiquiRedisClient().TxPipelined(context.Background(), func(pipe redis.Pipeliner) error {
		for key, count := range keyCountMap {
			pipe.HIncrBy(context.Background(), xredis.GetLiquidationStatsKey(), key, count)
		}
		return nil
	})
	if err != nil {
		// Ignore blocking the process
		xlog.Errorf("Error increasing count on redis: %v", err)
	}
}

func (t *TaskRunnerImpl) RunLiquidationJob6() error {
	defer cutils.LogTime(time.Now(), metric.LIQUIDATION_RUN_JOB_5)

	max_insurance := os.Getenv("MAX_INSURANCE")
	if max_insurance != "" {
		MAX_INSURANCE, _ = strconv.ParseUint(max_insurance, 10, 64)
	}

	oraclePricesMap, perpetualMarketsMap, cumulativeFundingRateMap, err := t.GetAppState(appstate.GetAppConfig(10*time.Second, 5*time.Minute, 24*time.Hour))
	if err != nil {
		xlog.Errorf("This is a critical error. We can't proceed without the app state")
		return err
	}

	transferStats := types.SubpoolTransferStats{}
	settleWithInsuranceStats := types.SettleWithInsuranceStats{}

	negativeBalanceSubaccounts := t.pool.GetAllNegativeBalanceSubaccounts()
	for _, subaccount := range negativeBalanceSubaccounts {
		updatedSubaccount := t.subaccountBalance.MustGetSubaccountBalancesFromIds([]string{subaccount.SubaccountId})[0]
		isBelowInitialMargin, isBelowMaintenanceMargin, requiresInsurance, _ := t.CheckSubaccountCollateralization(updatedSubaccount, oraclePricesMap, perpetualMarketsMap, cumulativeFundingRateMap)
		// Liquidate the subaccount
		if requiresInsurance {
			if settleWithInsuranceStats.TotalRecorded() >= uint(MAX_INSURANCE) {
				xlog.Infof("JOB-6: Max settle with insurance reached. Exiting the job")
				break
			}
			xlog.Infof("Starting settlement via insurance funds for subaccount: %s", updatedSubaccount.SubaccountId)
			success, err := liquidator.SettleWithInsurance(updatedSubaccount.SubaccountId)
			settleWithInsuranceStats.Record(success)
			if err != nil {
				xlog.Warnf("Error settling via insurance funds for subaccount: %s. Error: %v", updatedSubaccount.SubaccountId, err)
			}
		} else if isBelowMaintenanceMargin {
			transferStats.BelowMaintenanceMarginSubaccounts++
			t.pool.MoveSubaccountFromNegativeEquityTo(subaccount, pool.BelowMaintanenceMarginSubaccountsSubpool)
		} else if isBelowInitialMargin {
			// This case can happen if user added funds to the account after undergoing liquidation
			transferStats.BelowInitialMarginSubaccounts++
			t.pool.MoveSubaccountFromNegativeEquityTo(subaccount, pool.BelowInitialMarginSubaccountsSubpool)
		} else {
			transferStats.HealthySubaccounts++
			t.pool.MoveSubaccountFromNegativeEquityTo(subaccount, pool.HealthySubaccountsSubpool)
		}
	}

	xlog.Infof("JOB-6 a): Transfers from NegativeBalanceSubaccounts completed: Starting count for negative balance accounts: %v | Transfer stats: %+v", len(negativeBalanceSubaccounts), transferStats)
	xlog.Infof("JOB-6: SettleWithInsuranceFunds stats: %+v", settleWithInsuranceStats)
	// send discord alert
	if settleWithInsuranceStats.AtleastOneRecorded() && xclient.GlobalDiscordClient != nil {
		xclient.GlobalDiscordClient.SendWebhookMessage(fmt.Sprintf("SettleWithInsuranceFunds stats: %+v", settleWithInsuranceStats))
	}

	return nil
}

// 1. Get all the subaccounts from healthy subaccounts, below initial margin subaccounts, below maintenance margin subaccounts
// 2. Get app state and calculate withdrawable funds from
//   - Total equity
//   - Total margin (Equity - Initial Margin)
//
// 3. Sum up withdrawable funds for all the subaccounts for each product id
// 4. Save this in Application memory
func (t *TaskRunnerImpl) RunFundsMonitorJob() error {
	defer cutils.LogTime(time.Now(), metric.FUNDS_MONITORING_RUN_JOB)

	stopFundsMonitoring := os.Getenv("STOP_FUNDS_MONITORING")
	if stopFundsMonitoring == "1" {
		xlog.Infof("Funds monitoring is stopped")
		return nil
	}

	accountsToExcludeStr := os.Getenv("FUNDS_MONITORING_EXCLUDED_ACCOUNTS")
	accountsToExclude := strings.Split(accountsToExcludeStr, ",")

	oraclePricesMap, perpetualMarketsMap, cumulativeFundingRateMap, err := t.GetAppState(appstate.GetAppConfig(1*time.Second, 1*time.Minute, 24*time.Hour))
	if err != nil {
		xlog.Errorf("This is a critical error. We can't proceed without the app state")
		return nil
	}

	healthySubaccounts := t.pool.GetAllHealthySubaccounts()
	belowInitialMarginSubaccounts := t.pool.GetAllBelowInitialMarginSubaccounts()
	belowMaintenanceMarginSubaccounts := t.pool.GetAllBelowMaintanenceMarginSubaccounts()

	mergedSubaccounts := cutils.MergeMaps(healthySubaccounts, belowInitialMarginSubaccounts, belowMaintenanceMarginSubaccounts)
	subaccountIds := cutils.MapKeys(mergedSubaccounts)

	// This tells the token amount that can be withdrawn from all the user accounts at the moment
	var fundsWithBalanceLock map[uint32]*big.Int
	// This tells the token amount that can be withdrawn from all the user accounts if all open orders and positions are closed
	var fundsWithoutBalanceLock map[uint32]*big.Int
	// This tells the token amount that is in users account. Whether withdrawable or not doesn't matter, it is something which is not present with exchange.
	var unsettledFundsWithUser map[uint32]*big.Int

	// Get withdrawable balances 1) with initial marign locking 2) without initial margin locking
	runGracefuly(subaccountIds, func(subaccountBatchIds []string) {
		updatedSubaccounts := t.subaccountBalance.MustGetSubaccountBalancesFromIds(subaccountBatchIds)
		for _, subaccountBalance := range updatedSubaccounts {
			if cutils.SliceExists(accountsToExclude, subaccountBalance.SubaccountId) {
				xlog.Infof("Skipping subaccount: %s", subaccountBalance.SubaccountId)
				continue
			}

			// Calculate withdrawable funds with balance locking
			withdrawableFunds, err := subaccount.GetWithdrawableBalance(subaccountBalance, oraclePricesMap, perpetualMarketsMap, cumulativeFundingRateMap)
			if err != nil {
				xlog.Errorf("Error calculating withdrawable funds for subaccount: %s. Error: %v", subaccountBalance.SubaccountId, err)
				continue
			}
			// Calculate withdrawable funds without balance locking
			withdrawableFundsWithoutLock, err := subaccount.GetWithdrawableBalance(subaccountBalance, oraclePricesMap, perpetualMarketsMap, cumulativeFundingRateMap, true)
			if err != nil {
				xlog.Errorf("Error calculating withdrawable funds without lock for subaccount: %s. Error: %v", subaccountBalance.SubaccountId, err)
				continue
			}

			fundsWithBalanceLock = cutils.GenericMergeMaps(fundsWithBalanceLock, cutils.ModifyMapValues(withdrawableFunds, func(v string) *big.Int {
				return cutils.StrToBigInt(v)
			}), func(v1, v2 *big.Int) *big.Int {
				return new(big.Int).Add(v1, v2)
			})

			fundsWithoutBalanceLock = cutils.GenericMergeMaps(fundsWithoutBalanceLock, cutils.ModifyMapValues(withdrawableFundsWithoutLock, func(v string) *big.Int {
				return cutils.StrToBigInt(v)
			}), func(v1, v2 *big.Int) *big.Int {
				return new(big.Int).Add(v1, v2)
			})

			unsettledFundsWithUser = cutils.GenericMergeMaps(unsettledFundsWithUser, cutils.ModifyMapValues(subaccountBalance.SpotBalances, func(v subaccountTypes.SpotBalance) *big.Int {
				return new(big.Int).Set(v.Balancex18)
			}), func(v1, v2 *big.Int) *big.Int {
				return new(big.Int).Add(cutils.MaxBigInt(cutils.GetBig0(), v1), cutils.MaxBigInt(cutils.GetBig0(), v2))
			})
		}
	})

	// Save the withdrawable funds in the application memory
	stats.GlobalFundsMonitorStats.UpdateFundsMapx18(fundsWithBalanceLock, fundsWithoutBalanceLock, unsettledFundsWithUser)
	xlog.Infof("JOB-8: Updated funds monitoring map")

	return nil
}

func (t *TaskRunnerImpl) RunKromaBurnJob() error {
	defer cutils.LogTime(time.Now(), metric.KROMA_BURN_RUN_JOB)

	stopKromBurnJob := os.Getenv("STOP_KROMA_BURN_JOB")
	if stopKromBurnJob == "1" {
		return nil
	}

	accountsToExcludeStr := os.Getenv("KROMA_BURN_EXCLUDED_ACCOUNTS")
	accountsToExclude := strings.Split(accountsToExcludeStr, ",")

	healthySubaccounts := t.pool.GetAllHealthySubaccounts()
	belowInitialMarginSubaccounts := t.pool.GetAllBelowInitialMarginSubaccounts()
	belowMaintenanceMarginSubaccounts := t.pool.GetAllBelowMaintanenceMarginSubaccounts()

	mergedSubaccounts := cutils.MergeMaps(healthySubaccounts, belowInitialMarginSubaccounts, belowMaintenanceMarginSubaccounts)
	subaccountIds := cutils.MapKeys(mergedSubaccounts)

	kromaBurnResult := struct {
		SuccessCount int
		FailureCount int
		SkipCount    int
	}{
		SuccessCount: 0,
		FailureCount: 0,
		SkipCount:    0,
	}

	runGracefuly(subaccountIds, func(subaccountBatchIds []string) {
		updatedSubaccounts := t.subaccountBalance.MustGetSubaccountBalancesFromIds(subaccountBatchIds)
		for _, subaccountBalance := range updatedSubaccounts {
			if cutils.SliceExists(accountsToExclude, subaccountBalance.SubaccountId) {
				xlog.Infof("Skipping subaccount: %s", subaccountBalance.SubaccountId)
				continue
			}

			// Check if KROMA SPOT Balance is present
			if subaccountBalance.MustGetSpotBalance(contractUtils.KROMA_USDC).Balancex18.Sign() > 0 || subaccountBalance.MustGetSpotBalance(contractUtils.KROMA_USDT).Balancex18.Sign() > 0 {
				// Make api call to burn KROMA balance
				err := xclient.GlobalApiServerClient.BurnKromaBalance(subaccountBalance.SubaccountId)
				if err != nil {
					xlog.Errorf("Error burning KROMA balance for subaccount: %s. Error: %v", subaccountBalance.SubaccountId, err)
					kromaBurnResult.FailureCount++
					continue
				}
				kromaBurnResult.SuccessCount++
				time.Sleep(500 * time.Millisecond) // Sleep for 500ms to avoid rate limiting
			} else {
				kromaBurnResult.SkipCount++
			}
		}
	})

	// Save the withdrawable funds in the application memory
	xlog.Infof("JOB-9: KROMA Burn Job completed with result: %+v", kromaBurnResult)
	xclient.GetGlobalDiscordClient().SendWebhookMessage(fmt.Sprintf("KROMA Burn Job completed with result: %+v", kromaBurnResult))

	return nil
}

// 1. Get all oracle prices from oracle gateway
// 2. Get all perpetual markets from db. This number will be very small so, we can avoid batching
// 3. Get all funding rates from redis
func (t *TaskRunnerImpl) GetAppState(cfg ...appstate.AppStateCacheConfig) (map[string]ctypes.OraclePrice, map[uint]subaccountTypes.PerpetualMarket, map[string]*big.Int, error) {
	return t.appState.GetAppState(cfg...)
}

func (t *TaskRunnerImpl) GetAllSubaccountsAfterLastCheckpoint() ([]subaccountTypes.SubaccountBalances, uint, error) {
	return t.subaccountBalance.GetAllSubaccounts(t.lastSubaccountFetchGID + 1)
}

// NOTE: For AMM this will always return false for all the values, i.e. AMM is always assumed to be healthy
// 1. Get settledSubaccount
// 2. Calculate New Required Margin
// 3. Calculate Total Net Equity
//   - Total equity = Total spot balance + total unrealised pnl on perps + total unrealised funding fee.
//
// 3. Check if Total Net Equity is < Initial Margin
//   - Initial Required Margin = Sum over (position size * initial margin fraction)
//
// 4. Check if Total Net Equity is < Required Margin
//   - Required Maintenance Margin = Sum over (position size * required margin fraction)
//
// 5. Check if Balance is negative and required insurance funds to settle (This means quote spot is negative and there are no positions)

func (t *TaskRunnerImpl) CheckSubaccountCollateralization(
	unsettledSubaccount subaccountTypes.SubaccountBalances,
	oraclePricesMap map[string]ctypes.OraclePrice,
	perpetualMarketsMap map[uint]subaccountTypes.PerpetualMarket,
	cumulativeFundingRateMap map[string]*big.Int,
) (isBelowInitialMargin bool, isBelowMaintenanceMargin bool, requiresInsurance bool, isBelowLiquidationMargin bool) {
	ammSubaccountId := os.Getenv("AMM_SUBACCOUNT_ID")
	if ammSubaccountId == "" {
		log.Fatalf("AMM_SUBACCOUNT_ID is not set")
	} else if strings.EqualFold(ammSubaccountId, unsettledSubaccount.SubaccountId) {
		xlog.Debugf("Skipping AMM subaccount: %s", ammSubaccountId)
		return false, false, false, false
	}

	// Total net equity
	totalNetEquityx36 := subaccount.GetTotalEquityx36(unsettledSubaccount, oraclePricesMap, cumulativeFundingRateMap)

	// Calculate required margins - initial and maintenance
	maintenanceMarginx36, initialMarginx36 := subaccount.GetRequiredMarginValues(unsettledSubaccount, perpetualMarketsMap, oraclePricesMap)
	// This is a special param which is 0.999 of maintenance margin. This is used to place liquidation orders
	liquidationMarginx36 := cutils.Divx18(new(big.Int).Mul(maintenanceMarginx36, cutils.MulxCust(big.NewInt(999), 15)))

	// Spot is negative and there are no positions open
	requiresInsurance = totalNetEquityx36.Sign() == -1 && maintenanceMarginx36.Sign() == 0
	isBelowInitialMargin = initialMarginx36.Sign() == 1 && totalNetEquityx36.Cmp(initialMarginx36) == -1
	isBelowMaintenanceMargin = maintenanceMarginx36.Sign() == 1 && totalNetEquityx36.Cmp(maintenanceMarginx36) == -1
	isBelowLiquidationMargin = isBelowMaintenanceMargin && liquidationMarginx36.Sign() == 1 && totalNetEquityx36.Cmp(liquidationMarginx36) == -1

	if os.Getenv("DEBUG_MODE") == "1" {
		xlog.Debugf("Total net equity for subaccount: %s: %v", unsettledSubaccount.SubaccountId, cutils.X18ToFloatStr(cutils.Divx18(totalNetEquityx36)))
		xlog.Debugf("Maintenance margin for subaccount: %s: %v", unsettledSubaccount.SubaccountId, cutils.X18ToFloatStr(cutils.Divx18(maintenanceMarginx36)))
		xlog.Debugf("Initial margin for subaccount: %s: %v", unsettledSubaccount.SubaccountId, cutils.X18ToFloatStr(cutils.Divx18(initialMarginx36)))
		xlog.Debugf("Result for subaccount: %s: isBelowInitialMargin: %v, isBelowMaintenanceMargin: %v, requiresInsurance: %v", unsettledSubaccount.SubaccountId, isBelowInitialMargin, isBelowMaintenanceMargin, requiresInsurance)
	}
	return isBelowInitialMargin, isBelowMaintenanceMargin, requiresInsurance, isBelowLiquidationMargin
}

// Write pool and last subaccount fetch gid to redis
func (tr *TaskRunnerImpl) WriteCurrentStateToRedis() {
	defer cutils.LogTime(time.Now(), metric.LIQUIDATION_WRITE_REDIS_STATE)
	// Write all the subaccounts to redis
	lastGIDKey := xredis.GetLiquidationLastSubaccountGIDKey()
	_, err := xredis.GetLiquiRedisClient().TxPipelined(context.Background(), func(pipe redis.Pipeliner) error {
		err := pipe.Set(context.Background(), lastGIDKey, tr.lastSubaccountFetchGID, 0).Err()
		if err != nil {
			return err
		}
		return tr.pool.WriteToRedis(pipe)
	})

	if err != nil {
		// TODO: Add discord noitification here
		xlog.Errorf("Error setting last subaccount fetch gid or pool data to redis. This basically means there is some issue with redis connection. This means liquidation will start from state when last upgrade was performed. err: %v | lastGIDKey: %v | lastGID: %v", err, lastGIDKey, tr.lastSubaccountFetchGID)
	} else {
		xlog.Infof("Successfully set last subaccount fetch gid and pool to redis: %v", tr.lastSubaccountFetchGID)
	}
}

// Fetch current state from redis
// This will be used to bootup the system
func (tr *TaskRunnerImpl) FetchCurrentStateFromRedis() error {
	defer cutils.LogTime(time.Now(), metric.LIQUIDATION_FETCH_REDIS_STATE)
	xlog.Infof("Fetching last subaccount fetch gid from redis")
	lastGIDKey := xredis.GetLiquidationLastSubaccountGIDKey()
	lastGID, err := xredis.GetLiquiRedisClient().Get(context.Background(), lastGIDKey).Uint64()
	if err == redis.Nil {
		lastGID = 0
	} else if err != nil {
		xlog.Errorf("Error fetching last subaccount fetch gid from redis: %v. This basically means there is some issue with redis connection. Printing lastGID here so, you can manually update key: %v - %v", err, lastGIDKey, lastGID)
		return err
	}
	xlog.Infof("Received last subaccount fetch gid from redis: %v", lastGID)
	tr.lastSubaccountFetchGID = uint(lastGID)

	xlog.Infof("Fetching pools from redis")
	err = tr.pool.FetchFromRedis()
	if err != nil {
		xlog.Errorf("Error fetching pools from redis: %v", err)
		return err
	}
	xlog.Infof("Successfully fetched pools from redis")

	return nil
}

// Get pool lengths and write to redis
func (tr *TaskRunnerImpl) WriteStats() error {
	// Pipe all the data to redis
	subpoolSizes := make(map[string]int)
	for _, cat := range pool.ALL_SUBPOOL_CATEGORIES {
		subpoolSizes[string(cat)] = tr.pool.GetSubpoolSize(cat)
	}

	stats.GlobalLiquidationStats.UpdateSubpoolSizes(subpoolSizes)
	return nil
}
