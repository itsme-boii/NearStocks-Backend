package services

import (
	"context"
	"encoding/json"
	"fmt"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"math/big"
	"time"

	"github.com/redis/go-redis/v9"
)

type StakingService struct {
	redisClient       *redis.Client
	stakingDB         *db.StakingDB
	earningsPrecision *big.Int
}

func NewStakingService() *StakingService {
	return &StakingService{
		redisClient:       xredis.GetRedisClient(),
		stakingDB:         &db.StakingDB{},
		earningsPrecision: new(big.Int).Exp(big.NewInt(10), big.NewInt(14), nil),
	}
}

type EarningData struct {
	CumulativeEarningRatex18   *big.Int
	CumulativeEarningTimestamp int64
	EarningApyfx18             *big.Int
}

func GetEarningDataFromRedis() (*EarningData, error) {
	context := context.Background()
	cumulativeEarningRateKey := xredis.GetCumulativeEarningRateKey()
	var cumulativeEarningDict xredis.CumulativeEarningRateData
	value, err := xredis.GetRedisClient().Get(context, cumulativeEarningRateKey).Result()
	if err != nil {
		xlog.Errorf("Staking Service - Error fetching cummulative earning rate key value:", err)
		return nil, err
	}

	err = json.Unmarshal([]byte(value), &cumulativeEarningDict)
	if err != nil {
		xlog.Errorf("Staking Service - Error unmarshalling redis value:", err)
		return nil, err
	}

	var earningDataFromRedis EarningData
	var ok bool
	earningDataFromRedis.CumulativeEarningRatex18, ok = new(big.Int).SetString(cumulativeEarningDict.CumulativeEarningRate, 10)
	if !ok {
		xlog.Errorf("Staking Service - Error parsing CumulativeEarningRate from redis")
		return nil, fmt.Errorf("error parsing CumulativeEarningRate from redis")
	}

	xlog.Debugf("CumulativeEarningRate: %s", earningDataFromRedis.CumulativeEarningRatex18.String())

	earningDataFromRedis.CumulativeEarningTimestamp = cumulativeEarningDict.CumulativeEarningTimestamp
	// Get the APY earning factor from redis
	currentApyEarningFactorKey := xredis.GetCurrentApyEarningFactorKey()
	apyEarningFactorFloatStr, err := xredis.GetRedisClient().Get(context, currentApyEarningFactorKey).Result()
	if err != nil {
		xlog.Errorf("Staking Service - Error fetching APY earning factor key value:", err)
		return nil, err
	}

	earningDataFromRedis.EarningApyfx18 = cutils.FloatStrToX18(apyEarningFactorFloatStr)
	return &earningDataFromRedis, nil
}

// Total claimable : Offset + (Amount * (Latest cummulative - last op cummulative)) + (currentTime - last cummulative timestamp) * Amount
// Outstanding Earnings =
//   - (Delta Earning upto last minute based on cumulative funding)
//   - (Total pending realised earning)
//   - (On the fly Delta Earning within minute)
//
// Total pending realised earning = Total realised earning - Total claimed earning

// From last claim entry to current entry - sum of all earnings
// + calculate Current Earningsx18
type StakeEarnings struct {
	TotalClaimableEarningsx18   *big.Int
	TotalTransientEarningsx18   *big.Int
	CurrentStakedAmountx18      *big.Int
	CumulativeEarningsRatex18   *big.Int
	CurrentEarningx18           *big.Int
	CurrentTransientEarningsx18 *big.Int
	SpyMultTimeDeltax18         *big.Int
	EarningMetadata             *EarningData
}

func (s *StakingService) GetClaimableEarnings(subaccountId string) *StakeEarnings {
	// Assume this to be in desc order of creation
	stakes, err := s.stakingDB.GetAllBySubaccountId(subaccountId)
	if err != nil {
		xlog.Errorf("Staking Service - Error fetching stakes for subaccountHex: %s", subaccountId)
		return nil
	}

	lastEarningsx18 := big.NewInt(0)
	lastTransientEarningsx18 := big.NewInt(0)
	for _, stake := range stakes {
		if stake.Action == "claim" {
			break
		}
		lastEarningsx18 = new(big.Int).Add(lastEarningsx18, cutils.StrToBigInt(stake.Earnings))
		lastTransientEarningsx18 = new(big.Int).Add(lastTransientEarningsx18, cutils.FloatStrToX18(stake.TransientEarningsx18))
	}

	currentEarningsx18, curTransientEarningsx18, curStakedAmountx18, cumulativeEarningRatex18, spyMultTimeDeltax18, earningMetadata, err := s.GetCurrentEarningsx18(subaccountId)
	if err != nil {
		xlog.Errorf("Staking Service - Error fetching current earnings for subaccounId: %s", subaccountId)
		return nil
	}

	data := &StakeEarnings{
		TotalClaimableEarningsx18:   new(big.Int).Add(lastEarningsx18, currentEarningsx18),
		TotalTransientEarningsx18:   new(big.Int).Add(lastTransientEarningsx18, curTransientEarningsx18),
		CurrentStakedAmountx18:      curStakedAmountx18,
		CumulativeEarningsRatex18:   cumulativeEarningRatex18,
		SpyMultTimeDeltax18:         spyMultTimeDeltax18,
		CurrentEarningx18:           currentEarningsx18,
		CurrentTransientEarningsx18: curTransientEarningsx18,
		EarningMetadata:             earningMetadata,
	}

	return data
}

// Current earnings = Last row offset + on the fly earning for current Stake + Unstake + Cumulative delta earnings
// Return Current Earning and Transient earning and current stake amount
func (s *StakingService) GetCurrentEarningsx18(subaccountId string) (_currentEarningx18 *big.Int, _transientEarningx18 *big.Int, _stakedAmountx18 *big.Int, _cumulativeEarningRatex18 *big.Int, _spyMultTimeDeltax18 *big.Int, _earningMetadata *EarningData, _err error) {
	timestampNowInSeconds := time.Now().Unix()
	earningData, err := GetEarningDataFromRedis()
	if err != nil {
		xlog.Errorf("Staking Service - Error fetching earning data from redis :%v", err)
		return nil, nil, nil, nil, nil, nil, fmt.Errorf("error fetching earning data from redis")
	}

	// Get the last recorded entry (assuming stakes are ordered by created_at DESC)
	lastStake := s.stakingDB.GetLastEntryBySubaccountHex(subaccountId)
	timeDeltaInSeconds := timestampNowInSeconds - earningData.CumulativeEarningTimestamp
	xlog.Debugf("Staking Service - Time delta in seconds: %d", timeDeltaInSeconds)

	spyMultTimeDeltax18 := new(big.Int).Mul(ApyfToSpyf(earningData.EarningApyfx18), big.NewInt(timeDeltaInSeconds))
	// This means it's a new entry. So, for the first entry there won't be any transient earning / current earnings

	if lastStake == nil {
		xlog.Warnf("Staking Service - No stake entry found for subaccount ID: %s", subaccountId)
		return big.NewInt(0), big.NewInt(0), big.NewInt(0), earningData.CumulativeEarningRatex18, spyMultTimeDeltax18, earningData, nil
	}

	offsetx18, ok := new(big.Int).SetString(lastStake.Offsetx18, 10)
	if !ok {
		xlog.Errorf("Staking Service - error parsing stake offset")
		return nil, nil, nil, nil, nil, nil, fmt.Errorf("error parsing stake offset")
	}

	// CumulativeEarningRate is stored as a string in lastStake.CumulativeEarningRate
	latestCumulativeRatex18 := new(big.Int)
	latestCumulativeRatex18, ok = new(big.Int).SetString(lastStake.CumulativeEarningsRate, 10)
	if !ok {
		return nil, nil, nil, nil, nil, nil, fmt.Errorf("failed to parse CumulativeEarningRate for subaccount ID: %s", subaccountId)
	}

	stakedAmountx18, err := s.GetTotalStakedAmount(subaccountId)
	if err != nil {
		xlog.Errorf("Staking Service - Error fetching total staked amount :%v", err)
		return nil, nil, nil, nil, nil, nil, fmt.Errorf("error fetching total staked amount")
	}
	cumulativeDeltaEarningsx18 := GetCumulativeDeltaEarningsx18(stakedAmountx18, earningData.CumulativeEarningRatex18, latestCumulativeRatex18)
	onTheFlyDeltaEarningsx18 := GetOnTheFlyDeltaEarningsx18(stakedAmountx18, earningData.CumulativeEarningTimestamp, timestampNowInSeconds, earningData.EarningApyfx18)
	transientEarningsx18 := new(big.Int).Add(onTheFlyDeltaEarningsx18, offsetx18)
	currentEarningsx18 := new(big.Int).Add(transientEarningsx18, cumulativeDeltaEarningsx18)

	return currentEarningsx18, transientEarningsx18, stakedAmountx18, earningData.CumulativeEarningRatex18, spyMultTimeDeltax18, earningData, nil
}

func GetCumulativeDeltaEarningsx18(stakedAmountx18 *big.Int, curCumulativeEarningsRatex18 *big.Int, latestCumulativeRatex18 *big.Int) *big.Int {
	cumDeltaEarningsx18 := new(big.Int).Sub(curCumulativeEarningsRatex18, latestCumulativeRatex18)
	cumDeltaEarningsx36 := new(big.Int).Mul(cumDeltaEarningsx18, stakedAmountx18)
	return cutils.Divx18(cumDeltaEarningsx36)
}

func GetOnTheFlyDeltaEarningsx18(stakedAmountx18 *big.Int, lastUpdatedTimestamp int64, timestampNow int64, earningApyFactorx18 *big.Int) *big.Int {
	// Calculate the time difference in seconds
	timeDifferenceInSeconds := timestampNow - lastUpdatedTimestamp
	// Calculate the earnings within the minute
	earningWithinMinutex18 := new(big.Int).Mul(stakedAmountx18, big.NewInt(timeDifferenceInSeconds))
	// Multiply by the APY factor after converting to SPY factor
	spyfx18 := ApyfToSpyf(earningApyFactorx18)
	earningWithinMinutex18 = cutils.Divx18(new(big.Int).Mul(earningWithinMinutex18, spyfx18))
	return earningWithinMinutex18
}

// Annual yield factor (1e18 if 100%) to second yield factor (1e18 / 365 / 24 / 60 / 60 if 100% AYF)
// Calculate the APYF to SPYF
func ApyfToSpyf(apyx18 *big.Int) *big.Int {
	yearToSec := big.NewInt(365 * 24 * 60 * 60)
	spyx18 := new(big.Int).Div(apyx18, yearToSec)
	return spyx18
}

func (s *StakingService) GetTotalStakedAmount(subaccountId string) (*big.Int, error) {
	// Call the existing function to get all stakes for the subaccount
	stakes, err := s.stakingDB.GetAllBySubaccountId(subaccountId)
	if err != nil {
		return nil, err
	}

	totalStakedAmountx18 := big.NewInt(0)

	for _, stake := range stakes {
		amountx18 := new(big.Int)
		_, ok := amountx18.SetString(stake.Amount, 10) // Assuming stake.Amount is a string
		if !ok {
			xlog.Errorf("Staking Service - error parsing stake amount")
		}

		switch stake.Action {
		case "stake":
			totalStakedAmountx18.Add(totalStakedAmountx18, amountx18)
		case "unstake":
			totalStakedAmountx18.Sub(totalStakedAmountx18, amountx18)
		case "claim":
			// Do nothing
		default:
			// xlog.Errorf("Staking Service - unknown action type")
		}
	}

	return totalStakedAmountx18, nil
}

// ToDo - remove this and change implementation in market controller
func (s *StakingService) PopulateActiveSymbols() ([]string, error) {
	activeMarkets := (&db.MarketDB{}).GetAllActiveMarkets()
	if activeMarkets == nil {
		return nil, fmt.Errorf("no active markets found")
	}

	var symbols []string
	for _, market := range *activeMarkets {
		symbols = append(symbols, market.Symbol)
	}

	return symbols, nil
}
