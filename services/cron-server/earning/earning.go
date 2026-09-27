package earning

import (
	"context"
	"encoding/json"
	"fmt"
	"github/eugenix-io/logx-inf-backend/contract"
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/transaction"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"github/eugenix-io/logx-inf-backend/xclient"
	"math/big"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/robfig/cron/v3"
)

var (
	redisClient *redis.Client
)

// How does this code work ?
// 1. After every minute we fetch the latest earning per year factor - 1.2 means 120% APY | 0.87 means 87% APY
// 2. We convert that into seconds fraction - 1.2 APY means 1.2 / 31536000 =~ 3.8e-8 SPY
// 3. We update the cumulative earning rate in redis every minute and then also update the current minute earning per year factor
// 4. This current minute earning factor is used to calculate transient rewards for the stakers
func FetchCurrentEarningPerYearFactor() (*big.Int, string, error) {
	redisClient = xredis.GetRedisClient()

	apyEarningFactorKey := xredis.GetCurrentApyEarningFactorKey()
	apyEarningFactorFloatStr, err := redisClient.Get(context.Background(), apyEarningFactorKey).Result()
	if err != nil {
		xlog.Errorf("Staking Service - Error fetching APY earning factor key value:", err)
		xclient.GlobalDiscordClient.SendWebhookMessage(fmt.Sprintf("Earning Cron - Error fetching APY earning factor key value: %+v", err))
		return nil, "0", err
	}

	earningApyfractionX18 := cutils.FloatStrToX18(apyEarningFactorFloatStr)
	if earningApyfractionX18.Cmp(cutils.FloatStrToX18("2")) > 0 {
		xlog.Errorf("Earning Cron - Invalid APY earning factor from redis. We can't have APY > 200%")
		xclient.GlobalDiscordClient.SendWebhookMessage(fmt.Sprintf("Earning Cron - Invalid APY earning factor from redis. We can't have APY > 200 percent | current factor: %+v", err))
		return nil, "0", fmt.Errorf("earning Cron - Invalid APY earning factor from redis. We can't have APY > 200%%")
	}

	xlog.Infof("Earning Cron - Successfully Initialised")
	// Seconds fraction
	return new(big.Int).Div(earningApyfractionX18, big.NewInt(3_15_36_000)), apyEarningFactorFloatStr, nil
}

func StartEarningCron() {
	c := cron.New(cron.WithSeconds())
	ctx := context.Background()

	cumulativeEarningRateKey := xredis.GetCumulativeEarningRateKey()
	var (
		cumulativeEarningDict xredis.CumulativeEarningRateData
		cumulativeRate        *big.Int
	)

	// Job runs every minute
	// c.AddFunc("0 0 * * * *", func() {
	c.AddFunc("0 */1 * * * *", func() {
		currentTimestamp := time.Now().Unix()
		currentEarningSpyFractionX18, apyEarningFactorFloatStr, err := FetchCurrentEarningPerYearFactor()

		if err != nil {
			xlog.Errorf("Earning Cron - Error fetching latest earning factor. This is a critical issue:", err)
			return
		}

		//Check and fetch previous earning rate
		cumulativeEarningKeyExists, err := redisClient.Exists(ctx, cumulativeEarningRateKey).Result()
		if err != nil {
			xlog.Errorf("Earning Cron - Error checking key existence:", err)
			return
		}

		if cumulativeEarningKeyExists > 0 {
			// Key exists, fetch the value
			value, err := redisClient.Get(ctx, cumulativeEarningRateKey).Result()
			if err != nil {
				xlog.Errorf("Earning Cron - Error fetching key value:", err)
				return
			}

			// Assuming the value is stored as JSON, unmarshal it into the struct
			err = json.Unmarshal([]byte(value), &cumulativeEarningDict)
			if err != nil {
				xlog.Errorf("Earning Cron - Error unmarshalling redis value:", err)
				return
			}

			previousCumulativeRatex18 := new(big.Int)
			_, ok := previousCumulativeRatex18.SetString(cumulativeEarningDict.CumulativeEarningRate, 10)
			if !ok {
				xlog.Errorf("Earning Cron - Error converting redis value to big Int")
				return
			}
			fmt.Printf("APY in seconds :%v", currentEarningSpyFractionX18.String())
			timeDelta := big.NewInt(currentTimestamp - int64(cumulativeEarningDict.CumulativeEarningTimestamp))
			fmt.Printf("Time Delta :%v", timeDelta)
			cumulativeRateDeltax18 := new(big.Int).Mul(currentEarningSpyFractionX18, timeDelta)
			cumulativeRate = new(big.Int).Add(previousCumulativeRatex18, cumulativeRateDeltax18)

			cumulativeEarningDict = xredis.CumulativeEarningRateData{
				CumulativeEarningRate:      cumulativeRate.String(),
				CumulativeEarningTimestamp: currentTimestamp,
			}
		} else {
			cumulativeRate = currentEarningSpyFractionX18
			// Key does not exist, initialize to current values
			cumulativeEarningDict = xredis.CumulativeEarningRateData{
				CumulativeEarningRate:      currentEarningSpyFractionX18.String(),
				CumulativeEarningTimestamp: currentTimestamp,
			}
		}
		xlog.Infof("Earning Cron - Cumulative Earning Rate %s at timestamp %d", cumulativeEarningDict.CumulativeEarningRate, cumulativeEarningDict.CumulativeEarningTimestamp)
		data, err := json.Marshal(cumulativeEarningDict)
		if err != nil {
			xlog.Errorf("Earning Cron - Error marshalling data:", err)
			return
		}

		err = redisClient.Set(ctx, cumulativeEarningRateKey, data, 0).Err()
		if err != nil {
			xlog.Errorf("Earning Cron - Error setting key in Redis:", err)
			return
		}

		// Get new earning factor and update it in redis
		newApyEarningFactorFloatStr, err := redisClient.Get(ctx, xredis.GetApyEarningFactorKey()).Result()
		if err != nil {
			xlog.Errorf("Earning Cron - Error fetching New APY earning factor key value:", err)
		}

		if newApyEarningFactorFloatStr != apyEarningFactorFloatStr {
			currentApyEarningFactorKey := xredis.GetCurrentApyEarningFactorKey()
			xlog.Infof("Earning Cron - Updating current APY earning factor to %s", newApyEarningFactorFloatStr)
			err = redisClient.Set(ctx, currentApyEarningFactorKey, newApyEarningFactorFloatStr, 0).Err()
			if err != nil {
				xlog.Errorf("Earning Cron - Error setting current APY earning factor in Redis:", err)
				return
			}
			xclient.GlobalDiscordClient.SendWebhookMessage(fmt.Sprintf("Earning Cron - Updated current APY earning factor to %s", newApyEarningFactorFloatStr))
		}

		transactionCounter, err := transaction.IncrementCounter(1)
		if err != nil {
			xlog.Errorf("Failed to increment transaction counter in RewardTick: %v", err)
		}

		//Write Earnings Rate to Contract
		contract.GlobalContracts.EndpointContract.RewardRateTick(contractUtils.STAKER_CONTRACT_ADDRESS, cumulativeRate, transactionCounter)
	})
	// Start the cron scheduler
	c.Start()

	// Keep the cron scheduler running in the background
	select {}
}
