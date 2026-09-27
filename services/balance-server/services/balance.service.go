package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/appstate"
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/marketutils"
	"github/eugenix-io/logx-inf-backend/libs/perputils"
	"github/eugenix-io/logx-inf-backend/libs/spotUtils"
	"github/eugenix-io/logx-inf-backend/libs/subaccount"
	"github/eugenix-io/logx-inf-backend/libs/subaccountTypes"
	"github/eugenix-io/logx-inf-backend/libs/transaction"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"github/eugenix-io/logx-inf-backend/services/balance-server/types"
	"github/eugenix-io/logx-inf-backend/xclient"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

type BalanceService struct {
	redisClient       *redis.Client
	subaccountBalance subaccount.SubaccountBalance
	appState          appstate.AppState
	perpUtils         *perputils.PerpUtils
	SubaccountBal     subaccount.SubaccountBalance
}

type OraclePriceResponse struct {
	Data map[string]struct {
		Price struct {
			Type string `json:"type"`
			Hex  string `json:"hex"`
		} `json:"price"`
		Expo        int `json:"expo"`
		PublishTime int `json:"publishTime"`
	} `json:"data"`
	Tokens  []string `json:"tokens"`
	Message string   `json:"message"`
}

// NewBalanceServiceWithAppState is NewBalanceService with an injected price/market/funding source
// (tests and the parity generator).
func NewBalanceServiceWithAppState(as appstate.AppState) *BalanceService {
	bs := NewBalanceService()
	bs.appState = as
	return bs
}

func NewBalanceService() *BalanceService {
	return &BalanceService{
		redisClient:       xredis.GetRedisClient(),
		subaccountBalance: subaccount.NewSubaccountBalanceImpl(),
		appState:          appstate.NewAppState(),
		perpUtils:         perputils.NewPerpUtils(),
		SubaccountBal:     subaccount.NewSubaccountBalanceImpl(),
	}
}

func (s *BalanceService) UpdateTokenBalance(subaccountID string, productId uint32, tokenBalance string) error {
	// Check if productId is even
	defer cutils.LogTime(time.Now(), "UpdateTokenBalance")

	fmt.Printf("updating token balances: subaccountID=%s, productId=%d, tokenBalance=%v\n", subaccountID, productId, tokenBalance)

	if productId%2 != 0 {
		return fmt.Errorf("productId must be even for spot balances")
	}

	_, err := xredis.WithRedisLock(xredis.GetBalanceRedisLockKey(subaccountID), func() (*NOP, error) {
		// Retrieve the existing token balance for the specified productId
		hashKey := xredis.GetBalanceKey(subaccountID)
		field := xredis.GetBalanceField(productId)

		existingBalance, err := s.redisClient.HGet(ctx, hashKey, field).Result()
		if err != nil && err != redis.Nil {
			xlog.Errorf("Unable to get token balance: %v\n", err)
			return nil, err
		}

		// Define a structure to hold the balance information
		var balance types.Balance
		if existingBalance != "" {
			if err := json.Unmarshal([]byte(existingBalance), &balance); err != nil {
				return nil, fmt.Errorf("failed to unmarshal existing balance: %v", err)
			}
		} else {
			balance = types.Balance{
				Available: "0",
				Locked:    "0",
			}
		}

		// Convert the available balance to big.Int
		existingAvailableBigInt := new(big.Int)
		if balance.Available != "" {
			_, ok := existingAvailableBigInt.SetString(balance.Available, 10)
			if !ok {
				return nil, fmt.Errorf("failed to parse existing available balance: %s", balance.Available)
			}
		}

		// Convert the new token balance to big.Int
		newBalanceBigInt := new(big.Int)
		_, ok := newBalanceBigInt.SetString(tokenBalance, 10)
		if !ok {
			return nil, fmt.Errorf("failed to parse new balance: %s", tokenBalance)
		}

		// Update the available balance
		updatedAvailableBalance := new(big.Int).Add(existingAvailableBigInt, newBalanceBigInt)
		balance.Available = updatedAvailableBalance.String()

		// Marshal the updated balance back to JSON
		updatedBalanceJSON, err := json.Marshal(balance)
		if err != nil {
			xlog.Errorf("Unable to marshal updated balance: %v\n", err)
			return nil, err
		}

		// Store the updated balance in Redis
		err = s.redisClient.HSet(ctx, hashKey, field, updatedBalanceJSON).Err()
		if err != nil {
			xlog.Errorf("Unable to update token balance: %v\n", err)
			return nil, err
		}
		return nil, nil
	})

	if err != nil {
		return err
	}

	return nil
}

func (s *BalanceService) UpdatePreMarketBalance(subaccountID string, productId uint32, preMarketBalance string) error {
	defer cutils.LogTime(time.Now(), "UpdatePreMarketBalance")

	fmt.Printf("updating pre-market balances: subaccountID=%s, productId=%d, preMarketBalance=%v\n", subaccountID, productId, preMarketBalance)

	_, err := xredis.WithRedisLock(xredis.GetPreMarketBalanceLockKey(subaccountID), func() (*NOP, error) {
		hashKey := xredis.GetPreMarketBalanceKey(subaccountID)
		field := xredis.GetBalanceField(productId)

		existingBalance, err := s.redisClient.HGet(ctx, hashKey, field).Result()
		if err != nil && err != redis.Nil {
			xlog.Errorf("Unable to get pre-market balance: %v\n", err)
			return nil, err
		}

		var balance types.PreMarketBalance
		if existingBalance != "" {
			if err := json.Unmarshal([]byte(existingBalance), &balance); err != nil {
				return nil, fmt.Errorf("failed to unmarshal existing balance: %v", err)
			}
		} else {
			balance = types.PreMarketBalance{
				Available: "0",
			}
		}

		existingAvailableBigInt := new(big.Int)
		if balance.Available != "" {
			_, ok := existingAvailableBigInt.SetString(balance.Available, 10)
			if !ok {
				return nil, fmt.Errorf("failed to parse existing available balance: %s", balance.Available)
			}
		}

		newBalanceBigInt := new(big.Int)
		_, ok := newBalanceBigInt.SetString(preMarketBalance, 10)
		if !ok {
			return nil, fmt.Errorf("failed to parse new balance: %s", preMarketBalance)
		}

		updatedAvailableBalance := new(big.Int).Add(existingAvailableBigInt, newBalanceBigInt)
		balance.Available = updatedAvailableBalance.String()

		updatedBalanceJSON, err := json.Marshal(balance)
		if err != nil {
			xlog.Errorf("Unable to marshal updated balance: %v\n", err)
			return nil, err
		}

		err = s.redisClient.HSet(ctx, hashKey, field, updatedBalanceJSON).Err()
		if err != nil {
			xlog.Errorf("Unable to update pre-market balance: %v\n", err)
			return nil, err
		}
		return nil, nil
	})

	if err != nil {
		return err
	}

	return nil
}

func (s *BalanceService) GetPreMarketBalance(subaccountID string, productId uint32) (string, error) {
	defer cutils.LogTime(time.Now(), "GetPreMarketBalance")

	hashKey := xredis.GetPreMarketBalanceKey(subaccountID)
	field := xredis.GetBalanceField(productId)

	existingBalance, err := s.redisClient.HGet(ctx, hashKey, field).Result()
	if err != nil {
		if err == redis.Nil {
			// No balance found, return zero
			return "0", nil
		}
		xlog.Errorf("Unable to get pre-market balance: %v\n", err)
		return "", err
	}

	var balance types.PreMarketBalance
	if err := json.Unmarshal([]byte(existingBalance), &balance); err != nil {
		return "", fmt.Errorf("failed to unmarshal existing balance: %v", err)
	}

	return balance.Available, nil
}

func (s *BalanceService) GetPreMarketBalances(subaccountID string) (map[uint32]string, error) {
	defer cutils.LogTime(time.Now(), "GetPreMarketBalances")

	hashKey := xredis.GetPreMarketBalanceKey(subaccountID)

	// Get all fields (product IDs) and their corresponding balances
	balanceData, err := s.redisClient.HGetAll(ctx, hashKey).Result()
	if err != nil {
		if err == redis.Nil {
			// No balances found, return empty map
			return map[uint32]string{}, nil
		}
		xlog.Errorf("Unable to get pre-market balances: %v\n", err)
		return nil, err
	}

	result := make(map[uint32]string)
	for field, value := range balanceData {
		productID, err := strconv.ParseUint(field, 10, 32)
		if err != nil {
			xlog.Warnf("Skipping invalid product ID field: %s", field)
			continue
		}

		var balance types.PreMarketBalance
		if err := json.Unmarshal([]byte(value), &balance); err != nil {
			xlog.Warnf("Skipping invalid balance data for product ID %d: %v", productID, err)
			continue
		}

		result[uint32(productID)] = balance.Available
	}

	return result, nil
}

// Same as UpdatePreMarketBalance
func (s *BalanceService) UpdateSyntheticSpotBalance(subaccountID string, productId uint32, syntheticSpotBalance string) error {
	defer cutils.LogTime(time.Now(), "UpdateSyntheticSpotBalance")

	fmt.Printf("updating synthetic spot balances: subaccountID=%s, productId=%d, syntheticSpotBalance=%v\n", subaccountID, productId, syntheticSpotBalance)

	_, err := xredis.WithRedisLock(xredis.GetSyntheticSpotBalanceLockKey(subaccountID), func() (*NOP, error) {
		hashKey := xredis.GetSyntheticSpotBalanceKey(subaccountID)
		field := xredis.GetBalanceField(productId)

		existingBalance, err := s.redisClient.HGet(ctx, hashKey, field).Result()
		if err != nil && err != redis.Nil {
			xlog.Errorf("Unable to get synthetic spot balance: %v\n", err)
			return nil, err
		}

		var balance types.SyntheticSpotBalance
		if existingBalance != "" {
			if err := json.Unmarshal([]byte(existingBalance), &balance); err != nil {
				return nil, fmt.Errorf("failed to unmarshal existing balance: %v", err)
			}
		} else {
			balance = types.SyntheticSpotBalance{
				Available: "0",
			}
		}

		existingAvailableBigInt := new(big.Int)
		if balance.Available != "" {
			_, ok := existingAvailableBigInt.SetString(balance.Available, 10)
			if !ok {
				return nil, fmt.Errorf("failed to parse existing available balance: %s", balance.Available)
			}
		}

		newBalanceBigInt := new(big.Int)
		_, ok := newBalanceBigInt.SetString(syntheticSpotBalance, 10)
		if !ok {
			return nil, fmt.Errorf("failed to parse new balance: %s", syntheticSpotBalance)
		}

		updatedAvailableBalance := new(big.Int).Add(existingAvailableBigInt, newBalanceBigInt)
		balance.Available = updatedAvailableBalance.String()

		updatedBalanceJSON, err := json.Marshal(balance)
		if err != nil {
			xlog.Errorf("Unable to marshal updated balance: %v\n", err)
			return nil, err
		}

		err = s.redisClient.HSet(ctx, hashKey, field, updatedBalanceJSON).Err()
		if err != nil {
			xlog.Errorf("Unable to update synthetic spot balance: %v\n", err)
			return nil, err
		}
		return nil, nil
	})

	if err != nil {
		return err
	}

	return nil
}

// Same as GetPreMarketBalance
func (s *BalanceService) GetSyntheticSpotBalance(subaccountID string, productId uint32) (string, error) {
	defer cutils.LogTime(time.Now(), "GetSyntheticSpotBalance")

	hashKey := xredis.GetSyntheticSpotBalanceKey(subaccountID)
	field := xredis.GetBalanceField(productId)

	existingBalance, err := s.redisClient.HGet(ctx, hashKey, field).Result()
	if err != nil {
		if err == redis.Nil {
			// No balance found, return zero
			return "0", nil
		}

		xlog.Errorf("Unable to get synthetic spot balance: %v\n", err)
		return "", err
	}

	var balance types.SyntheticSpotBalance
	if err := json.Unmarshal([]byte(existingBalance), &balance); err != nil {
		return "", fmt.Errorf("failed to unmarshal existing balance: %v", err)
	}

	return balance.Available, nil
}

// Same as GetPreMarketBalances
func (s *BalanceService) GetSyntheticSpotBalances(subaccountID string) (map[uint32]string, error) {
	defer cutils.LogTime(time.Now(), "GetSyntheticSpotBalances")

	hashKey := xredis.GetSyntheticSpotBalanceKey(subaccountID)

	// Get all fields (product IDs) and their corresponding balances
	balanceData, err := s.redisClient.HGetAll(ctx, hashKey).Result()
	if err != nil {
		if err == redis.Nil {
			// No balances found, return empty map
			return map[uint32]string{}, nil
		}
		xlog.Warnf("Unable to get synthetic spot balances: %v\n", err)
		return nil, err
	}

	result := make(map[uint32]string)
	for field, value := range balanceData {
		productID, err := strconv.ParseUint(field, 10, 32)
		if err != nil {
			xlog.Warnf("Skipping invalid product ID field: %s", field)
			continue
		}

		var balance types.SyntheticSpotBalance
		if err := json.Unmarshal([]byte(value), &balance); err != nil {
			xlog.Warnf("Skipping invalid balance data for product ID %d: %v", productID, err)
			continue
		}

		result[uint32(productID)] = balance.Available
	}

	return result, nil
}

func (bs *BalanceService) UpdateMultiTokenBalance(subaccountHex string, productIds []uint32, tokenBalances []string) error {
	defer cutils.LogTime(time.Now(), "UpdateMultiTokenBalance")

	if len(productIds) != len(tokenBalances) {
		return fmt.Errorf("productIds and tokenBalances must be of the same length")
	}

	_, err := xredis.WithRedisLock(xredis.GetBalanceRedisLockKey(subaccountHex), func() (*NOP, error) {
		subaccountBalance := bs.subaccountBalance.MustGetSubaccountBalancesFromIds([]string{subaccountHex})[0]

		for i, productId := range productIds {
			// Check if productId is even
			if productId%2 != 0 {
				xlog.Warnf("BS - productId must be even for spot balances. productId: %d... Skipping: SubaccountHex - %v", productId, subaccountHex)
				continue
			}

			tokenBalancex18 := cutils.StrToBigInt(tokenBalances[i])
			subaccountBalance.UpdateSpotBalance(productId, tokenBalancex18)
		}

		_, errInner := xredis.GetRedisClient().TxPipelined(context.Background(), func(pipe redis.Pipeliner) error {
			// Update the subaccount balance in redis
			return subaccount.ReplaceBalanceInRedis(pipe, &subaccountBalance)
		})

		return nil, errInner
	})

	return err
}

// for testing purpose
func (s *BalanceService) GetBalance(subaccountID string) (map[string]interface{}, error) {
	// Use balance_subaccountID directly as the hash key
	hashKey := fmt.Sprintf("v0_balance_%s", subaccountID)

	// Retrieve all balances from Redis
	result := make(map[string]interface{})
	balances, err := s.redisClient.HGetAll(ctx, hashKey).Result()
	if err == redis.Nil {
		xlog.Warnf("No key in redis to fetch balance for subaccountID: %s", subaccountID)
		return result, nil
	} else if err != nil {
		return nil, fmt.Errorf("unable to get balances: %v", err)
	}

	for field, value := range balances {
		var balance interface{}

		// Convert field name to productId
		productId, err := strconv.Atoi(field)
		if err != nil {
			fmt.Printf("invalid product ID in field %s: %v\n", field, err)
			continue
		}

		// Determine the type of balance based on productId
		if productId%2 == 0 {
			// Even productId indicates TokenBalance
			var tokenBalance struct {
				Available string `json:"available"`
				Locked    string `json:"locked"`
			}
			if err := json.Unmarshal([]byte(value), &tokenBalance); err != nil {
				fmt.Printf("unable to unmarshal token balance data for field %s: %v\n", field, err)
				continue
			}
			balance = tokenBalance
		} else {
			// Odd productId indicates PerpBalance
			var perpBalance types.PerpBalance
			if err := json.Unmarshal([]byte(value), &perpBalance); err != nil {
				fmt.Printf("unable to unmarshal perp balance data for field %s: %v\n", field, err)
				continue
			}
			balance = perpBalance
		}

		result[field] = balance
	}

	return result, nil
}

// audit: @mananbordia this has no lock which means it can be called concurrently with update function
// Not a major risk for now but should be handled in future
func (bs *BalanceService) WithdrawableTokenBalancev2(subaccountIdHex string) (map[uint32]string, error) {
	// Get app state for prices
	subaccountBalance := bs.subaccountBalance.MustGetSubaccountBalancesFromIds([]string{subaccountIdHex})[0]
	oraclePricesMap, perpetualMarketsMap, cumulativeFundingRateMap, err := bs.appState.GetAppState()
	if err != nil {
		return nil, err
	}

	return subaccount.GetWithdrawableBalance(subaccountBalance, oraclePricesMap, perpetualMarketsMap, cumulativeFundingRateMap)
}

// Total token balance deposited
func (s *BalanceService) GetTokenBalanceMap(subaccountID string) (map[uint32]string, error) {
	// Use balance_subaccountID directly as the hash key
	hashKey := fmt.Sprintf("v0_balance_%s", subaccountID)

	// Retrieve all balances from Redis
	balances, err := s.redisClient.HGetAll(ctx, hashKey).Result()
	if err != nil {
		if err == redis.Nil {
			// Key does not exist, return an empty map
			return map[uint32]string{}, nil
		}
		return nil, fmt.Errorf("unable to get token balances: %v", err)
	}

	tokenBalanceMap := make(map[uint32]string)

	for field, value := range balances {
		productId, err := strconv.Atoi(field)
		if err != nil {
			return nil, fmt.Errorf("invalid product ID in field %s: %v", field, err)
		}

		// Only consider even productIds (TokenBalance)
		if productId%2 == 0 {
			var tokenBalance struct {
				Available string `json:"available"`
				Locked    string `json:"locked"`
			}
			if err := json.Unmarshal([]byte(value), &tokenBalance); err != nil {
				return nil, fmt.Errorf("unable to unmarshal token balance data for field %s: %v", field, err)
			}

			tokenBalanceMap[uint32(productId)] = tokenBalance.Available
		}
	}

	return tokenBalanceMap, nil
}

// If only one product id is provided fetch the field of hash
// Else fetch entire hash and iterate over the fields
// No product ids means fetch all perp positions
func (s *BalanceService) GetPerpPositions(subaccountID string, productIds []uint32) ([]map[string]interface{}, error) {
	// Use balance_subaccountID directly as the hash key
	numberOfProductIds := len(productIds)
	positions := []map[string]interface{}{}
	hashKey := xredis.GetBalanceKey(subaccountID)

	if numberOfProductIds != 1 {
		// Retrieve all balances from Redis
		balances, err := s.redisClient.HGetAll(ctx, hashKey).Result()
		if err != nil {
			if err == redis.Nil {
				// Key does not exist, return an empty slice
				return []map[string]interface{}{}, nil
			}
			// Log the error and return an empty slice
			fmt.Printf("Unable to get perp balances: %v\n", err)
			return []map[string]interface{}{}, nil
		}

		if numberOfProductIds == 0 {
			for field, value := range balances {
				productId, err := strconv.Atoi(field)
				if err != nil {
					fmt.Printf("Invalid product ID format: %v\n", field)
					continue
				}

				// Only consider odd productIds (PerpBalance)
				if productId%2 != 0 {
					var perpBalance types.PerpBalance
					if err := json.Unmarshal([]byte(value), &perpBalance); err != nil {
						fmt.Printf("Unable to unmarshal perp balance data for field %s: %v\n", field, err)
						continue
					}

					positions = append(positions, map[string]interface{}{
						"productID":       productId,
						"amount":          perpBalance.Amount,
						"vQuoteBalance":   perpBalance.VQuoteBalance,
						"lastFundingRate": perpBalance.LastFundingRate,
					})
				}
			}
		} else {
			// Iterate through the productIds and fetch the corresponding perp positions
			for _, productId := range productIds {
				if productId%2 == 0 {
					return nil, fmt.Errorf("invalid product ID: %d", productId)
				}
				productIdsStr := fmt.Sprintf("%d", productId)
				if value, ok := balances[productIdsStr]; ok {
					var perpBalance types.PerpBalance
					if err := json.Unmarshal([]byte(value), &perpBalance); err != nil {
						fmt.Printf("Unable to unmarshal perp balance data for field %s: %v\n", productIdsStr, err)
						continue
					}

					positions = append(positions, map[string]interface{}{
						"productID":       productId,
						"amount":          perpBalance.Amount,
						"vQuoteBalance":   perpBalance.VQuoteBalance,
						"lastFundingRate": perpBalance.LastFundingRate,
					})
				} else {
					fmt.Printf("No perp balance found for product ID: %d\n. Returning default values", productId)
					positions = append(positions, getDefaultPerpBalance(productId))
				}
			}
		}
	} else {
		if productIds[0]%2 == 0 {
			return nil, fmt.Errorf("invalid product ID: %d", productIds[0])
		}

		fieldKey := xredis.GetBalanceField(productIds[0])
		value, err := s.redisClient.HGet(ctx, hashKey, fieldKey).Result()
		if err == redis.Nil {
			fmt.Printf("No perp balance found for product ID: %d\n. Returning default values", productIds[0])
			positions = []map[string]interface{}{getDefaultPerpBalance(productIds[0])}
		} else if err != nil {
			return nil, fmt.Errorf("unable to get perp balance: %v", err)
		} else {
			var perpBalance types.PerpBalance
			if err := json.Unmarshal([]byte(value), &perpBalance); err != nil {
				fmt.Printf("Unable to unmarshal perp balance data for field %s: %v\n", value, err)
			}
			positions = []map[string]interface{}{
				{
					"productID":       productIds[0],
					"amount":          perpBalance.Amount,
					"vQuoteBalance":   perpBalance.VQuoteBalance,
					"lastFundingRate": perpBalance.LastFundingRate,
				},
			}
		}
	}

	return positions, nil
}

// sum of Pnl per token = perp.amount * tokenPrice + vQuote
func (s *BalanceService) GetTotalPnL(subaccountID string) (map[uint32]string, string, error) {
	oraclePricesMap, err := s.appState.GetAllOraclePrices(500 * time.Millisecond)
	if err != nil {
		return nil, "", fmt.Errorf("error fetching oracle prices: %v", err)
	}

	// Use balance_subaccountID directly as the hash key
	hashKey := fmt.Sprintf("v0_balance_%s", subaccountID)

	// Retrieve all balances from Redis
	balances, err := s.redisClient.HGetAll(ctx, hashKey).Result()
	if err != nil {
		if err == redis.Nil {
			return map[uint32]string{}, "0", nil
		}
		return nil, "0", fmt.Errorf("unable to get balances: %v", err)
	}

	totalPnLx18 := big.NewInt(0)
	pnlPerToken := make(map[uint32]string)

	for field, value := range balances {
		productId, err := strconv.Atoi(field)
		if err != nil {
			return nil, "0", fmt.Errorf("invalid product ID in field %s: %v", field, err)
		}

		// Only consider odd productIds (PerpBalance)
		if productId%2 != 0 {
			var perpBalance types.PerpBalance
			if err := json.Unmarshal([]byte(value), &perpBalance); err != nil {
				return nil, "0", fmt.Errorf("unable to unmarshal perp balance data for field %s: %v", field, err)
			}

			amount := new(big.Int)
			vQuoteBalance := new(big.Int)

			if _, ok := amount.SetString(perpBalance.Amount, 10); !ok {
				return nil, "0", fmt.Errorf("unable to parse amount: %v", perpBalance.Amount)
			}
			if _, ok := vQuoteBalance.SetString(perpBalance.VQuoteBalance, 10); !ok {
				return nil, "0", fmt.Errorf("unable to parse vQuoteBalance: %v", perpBalance.VQuoteBalance)
			}

			currentQuoteValuex36, err := perputils.GetQuoteValuex36(uint32(productId), amount, oraclePricesMap)
			if err != nil {
				return nil, "", fmt.Errorf("unable to get quote value for product %d: %v", productId, err)
			}

			pnlx18 := new(big.Int).Add(cutils.Divx18(currentQuoteValuex36), vQuoteBalance)
			pnlPerToken[uint32(productId)] = pnlx18.String()

			totalPnLx18.Add(totalPnLx18, pnlx18)
		}
	}

	return pnlPerToken, totalPnLx18.String(), nil
}

// TODO: FIXME: Currently we truncating the value to not have decimal places
func (s *BalanceService) GetValueInQuoteTokenx18(productId uint32, tokenAmount *big.Int) (*big.Int, error) {
	oraclePrices, err := s.appState.GetAllOraclePrices(500 * time.Millisecond)
	if err != nil {
		return nil, err
	}

	valuex36, err := spotUtils.GetQuoteValuex36(productId, tokenAmount, oraclePrices)
	if err != nil {
		return nil, err
	}

	return cutils.Divx18(valuex36), nil
}

// to add productId -> symbol mapping in redis
// Note - avoiding distributed redis lock here since we are not expecting race conditions
func (s *BalanceService) AddMapping(productId uint32, tokenAddress, symbol string) error {
	productIdKey := fmt.Sprintf("product:%d", productId)

	// Check if the productId already exists in Redis
	existingSymbol, err := s.redisClient.Get(ctx, productIdKey).Result()
	if err != nil && err != redis.Nil {
		return fmt.Errorf("unable to check productId in Redis: %v", err)
	}
	if existingSymbol != "" {
		return fmt.Errorf("productId already exists in Redis")
	}

	// Add the productId to symbol mapping to Redis
	err = s.redisClient.Set(ctx, productIdKey, symbol, 0).Err()
	if err != nil {
		return fmt.Errorf("unable to add productId to symbol mapping to Redis: %v", err)
	}

	// Add the token address to symbol mapping to Redis if token address is provided
	if tokenAddress != "" {
		existingTokenSymbol, err := s.redisClient.Get(ctx, tokenAddress).Result()
		if err != nil && err != redis.Nil {
			return fmt.Errorf("unable to check token address in Redis: %v", err)
		}
		if existingTokenSymbol != "" {
			return fmt.Errorf("token address already exists in Redis")
		}

		err = s.redisClient.Set(ctx, tokenAddress, symbol, 0).Err()
		if err != nil {
			return fmt.Errorf("unable to add token address mapping to Redis: %v", err)
		}
	}

	return nil
}

// Current app state caching at
// oracle - 500ms | funding - 5s | perptuals - 24h
// Fetch and calculate spot balance in dollars
// Total equity = Spot balance + uPnl + uFunding
// Available margin = Total Equity - initial margin of all positions - locked funds
func (bs *BalanceService) GetAvailableMarginV2(subaccountIdHex string) (*big.Int, error) {
	// Fetch balance of subaccount and app state
	subaccountBalance := bs.subaccountBalance.MustGetSubaccountBalancesFromIds([]string{subaccountIdHex})[0]
	oraclePricesMap, perpetualMarketsMap, cumulativeFundingRateMap, err := bs.appState.GetAppState(appstate.GetAppConfig(500*time.Millisecond, 5*time.Second, 24*time.Hour))
	if err != nil {
		xlog.Errorf("BS - Error while fetching app state: %v", err)
		return new(big.Int).SetInt64(0), err
	}
	return bs.getAvailableMarginV2(subaccountBalance, oraclePricesMap, perpetualMarketsMap, cumulativeFundingRateMap), nil
}

// Return x18 value of available margin
func (bs *BalanceService) getAvailableMarginV2(subaccountBalance subaccountTypes.SubaccountBalances, oraclePricesMap map[string]ctypes.OraclePrice, perpetualMarketsMap map[uint]subaccountTypes.PerpetualMarket, cumulativeFundingRateMap map[string]*big.Int) *big.Int {
	_, availableMarginx36 := subaccount.GetCollateralDetails(subaccountBalance, oraclePricesMap, perpetualMarketsMap, cumulativeFundingRateMap)
	return cutils.Divx18(availableMarginx36)
}

// Return x18 value of safety margin
// Safety margin is basically the (available margin + 0.04 Initial Margin)
// 4% of initial margin is kept as buffer so, orders don't cancel due to small fluctuations
func (bs *BalanceService) getSafetyMarginV2(subaccountBalance subaccountTypes.SubaccountBalances, oraclePricesMap map[string]ctypes.OraclePrice, perpetualMarketsMap map[uint]subaccountTypes.PerpetualMarket, cumulativeFundingRateMap map[string]*big.Int) *big.Int {
	return subaccount.SafetyMarginx18(subaccountBalance, oraclePricesMap, perpetualMarketsMap, cumulativeFundingRateMap)
}

func getDefaultPerpBalance(productId uint32) map[string]interface{} {
	return map[string]interface{}{
		"productID":       productId,
		"amount":          "0",
		"vQuoteBalance":   "0",
		"lastFundingRate": "0",
	}
}

func (s *BalanceService) FetchSpotBalances(subAccountId string) (productIdToBalance map[uint32]types.Balance) {
	// Use balance_subaccountID directly as the hash key
	hashKey := fmt.Sprintf("v0_balance_%s", subAccountId)

	productIdToBalance = make(map[uint32]types.Balance)

	// Retrieve all balances from Redis
	balances, err := s.redisClient.HGetAll(ctx, hashKey).Result()
	if err != nil {
		if err == redis.Nil {
			xlog.Errorf("Balance Service - No spot balances found for subaccount: %v\n", subAccountId)
			return productIdToBalance
		}
		xlog.Errorf("Balance Service - Error while fetching spot balance for %s : %v\n", subAccountId, err)
		return productIdToBalance
	}

	for field, value := range balances {
		productId, err := strconv.Atoi(field)
		if err != nil {
			xlog.Errorf("Balance service - Recieved invalid product ID format while fetching spot balances for %s. ProductId: %v, err: %v\n", subAccountId, field, err)
			continue
		}

		if productId%2 == 0 {
			var tokenBalance types.Balance
			if err := json.Unmarshal([]byte(value), &tokenBalance); err != nil {
				xlog.Errorf("Balance service - Unable to unmarshal token balance data for account: %s productId %s: err: %v\n", subAccountId, field, err)
				continue
			}

			productIdToBalance[uint32(productId)] = tokenBalance
		}
	}
	return productIdToBalance
}

// 1. Get spot balances
// 2. Calculate the dollar value of spot balances
// 3. Get locked dollar value
func (bs *BalanceService) GetSpotBalance(subaccountIDHex string) ([]ctypes.SpotBalance, string, string, error) {
	oraclePricesMap, err := bs.appState.GetAllOraclePrices(200 * time.Millisecond)
	if err != nil {
		return nil, "0", "0", err
	}

	subaccountBalance := bs.subaccountBalance.MustGetSubaccountBalancesFromIds([]string{subaccountIDHex})[0]

	var spotBalances []ctypes.SpotBalance
	for productId, spot := range subaccountBalance.SpotBalances {
		var lockedX18 *big.Int
		// TODO: Remove this after fixing the issue
		if spot.Lockedx18.Sign() == -1 {
			lockedX18 = big.NewInt(0)
			xlog.Errorf("Negative locked balance for subaccount %s, product %d", subaccountIDHex, productId)
			xclient.GlobalDiscordClient.SendWebhookMessage(fmt.Sprintf("Negative locked balance for subaccount %s, product %d", subaccountIDHex, productId))
		} else {
			lockedX18 = spot.Lockedx18
		}

		withdrawableAmount := new(big.Int).Sub(spot.Balancex18, lockedX18)

		spotBalances = append(spotBalances, ctypes.SpotBalance{
			ProductId:       uint32(productId),
			TokenBalance:    spot.Balancex18.String(),
			WithdrawBalance: withdrawableAmount.String(),
		})
	}

	freeBalancex36 := subaccount.GetTotalSpotBalancex36(subaccountBalance, oraclePricesMap)

	usedBalancex36 := subaccount.GetLockedValuex36(subaccountBalance, oraclePricesMap)

	return spotBalances, cutils.Divx18(freeBalancex36).String(), cutils.Divx18(usedBalancex36).String(), nil
}

func (s *BalanceService) deleteSubaccountKeys(subaccountIDs []string) error {
	ctx := context.Background()

	// Convert subaccount IDs to lowercase
	for i, subaccountID := range subaccountIDs {
		subaccountIDs[i] = strings.ToLower(subaccountID)
	}

	pipe := xredis.GetRedisClient().Pipeline()

	for _, subaccountID := range subaccountIDs {
		hashKey := xredis.GetBalanceKey(subaccountID)

		// Get all fields (product IDs) in the hash outside the pipeline
		fields, err := xredis.GetRedisClient().HKeys(ctx, hashKey).Result()
		if err != nil {
			return fmt.Errorf("error getting fields for subaccount %s: %v", subaccountID, err)
		}

		// Iterate through fields and collect those with odd product IDs
		var fieldsToDelete []string
		for _, field := range fields {
			productID, err := strconv.ParseUint(field, 10, 32)
			if err != nil {
				return fmt.Errorf("error parsing product ID %s: %v", field, err)
			}

			if productID%2 != 0 {
				fieldsToDelete = append(fieldsToDelete, field)
			}
		}

		// Queue up the HDel commands in the pipeline
		if len(fieldsToDelete) > 0 {
			pipe.HDel(ctx, hashKey, fieldsToDelete...)
			xlog.Infof("Queued deletion of product IDs %v for subaccount %s", fieldsToDelete, subaccountID)
		}
	}

	// Execute the pipeline
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("error executing pipeline: %v", err)
	}

	return nil
}

func (s *BalanceService) SyncSubaccounts(subaccountIDs []string) error {
	if len(subaccountIDs) == 0 {
		return nil
	}

	// Delete redis keys for all subaccounts
	if err := s.deleteSubaccountKeys(subaccountIDs); err != nil {
		xlog.Errorf("Error deleting subaccount keys: %v", err)
		return err
	}

	if err := s.fetchAndSyncSpotBalances(subaccountIDs); err != nil {
		xlog.Errorf("Error processing spot balances: %v", err)
		return err
	}
	if err := s.fetchAndSyncPerpPositions(subaccountIDs); err != nil {
		xlog.Errorf("Error processing perpetual positions: %v", err)
		return err
	}
	if err := s.syncNonceRedisToContract(subaccountIDs); err != nil {
		xlog.Errorf("Error syncing Redis nonce to contract nonce: %v", err)
		return err
	}
	return nil
}

func (s *BalanceService) SyncCumulativeFundingRates() error {
	return fmt.Errorf("SyncCumulativeFundingRates requires new implementation")
}

func (s *BalanceService) SyncOi() error {
	return fmt.Errorf("SyncOi requires new implementation")
}

func (s *BalanceService) syncNonceRedisToContract(subaccountIDs []string) error {
	return fmt.Errorf("syncNonceRedisToContract requires new implementation")
}

func (s *BalanceService) fetchAndSyncSpotBalances(subaccountIDs []string) error {
	return fmt.Errorf("fetchAndSyncSpotBalances requires new implementation")
}
func getSpotProductID(index int) uint32 {
	switch index {
	case 0:
		return 4
	case 1:
		return 0
	case 2:
		return 2
	default:
		return uint32(2 * index)
	}
}
func getPerpProductID(index int) uint32 {
	return uint32(2*index + 1)
}
func (s *BalanceService) syncTokenBalance(subaccountID string, productId uint32, tokenBalance string) error {
	xlog.Infof("Updating token balances: subaccountID=%s, productId=%d, tokenBalance=%v\n", subaccountID, productId, tokenBalance)

	// Check if productId is valid (even number for spot balances)
	if productId%2 != 0 {
		return fmt.Errorf("productId must be even for spot balances")
	}

	ctx := context.Background()
	hashKey := fmt.Sprintf("v0_balance_%s", subaccountID)
	field := fmt.Sprintf("%d", productId)

	var lockedBalance string = "0" // Default locked balance

	// Fetch existing balance from Redis, if it exists
	existingBalanceJSON, err := s.redisClient.HGet(ctx, hashKey, field).Result()
	if err == nil && existingBalanceJSON != "" {
		var existingBalance types.Balance
		// Unmarshal the existing balance
		err := json.Unmarshal([]byte(existingBalanceJSON), &existingBalance)
		xlog.Infof("Existing balance: %v\n", existingBalance)
		if err == nil {
			// Use the existing locked balance
			lockedBalance = existingBalance.Locked
			xlog.Infof("Existing locked balance: %v\n", lockedBalance)
		} else {
			xlog.Errorf("Unable to unmarshal existing balance: %v\n", err)
			return err
		}
	} else if err != redis.Nil {
		// Log the error if it's not a nil error
		xlog.Errorf("Unable to fetch existing balance: %v\n", err)
		return err
	}

	// If both available and locked balances are zero, delete the entry
	if tokenBalance == "0" && lockedBalance == "0" {
		xlog.Infof("Both tokenBalance and lockedBalance are zero for subaccountID=%s, productId=%d. Deleting entry.", subaccountID, productId)
		err = s.redisClient.HDel(ctx, hashKey, field).Err()
		if err != nil {
			xlog.Errorf("Unable to delete zero balance entry: %v\n", err)
			return err
		}
		return nil
	}

	// Update balance
	balance := types.Balance{
		Available: tokenBalance,
		Locked:    lockedBalance,
	}

	updatedBalanceJSON, err := json.Marshal(balance)
	if err != nil {
		xlog.Errorf("Unable to marshal updated balance: %v\n", err)
		return err
	}

	// Update Redis with the new balance
	err = s.redisClient.HSet(ctx, hashKey, field, updatedBalanceJSON).Err()
	if err != nil {
		xlog.Errorf("Unable to update token balance: %v\n", err)
		return err
	}
	return nil
}

func (s *BalanceService) fetchAndSyncPerpPositions(subaccountIDs []string) error {
	return fmt.Errorf("fetchAndSyncPerpPositions requires new implementation")
}

func (s *BalanceService) syncPerpBalanceInRedis(subaccountID string, productId uint32, updatedAmount, updatedVQuoteBalance, lastFundingRate string) error {
	ctx := context.Background()
	hashKey := fmt.Sprintf("v0_balance_%s", subaccountID)
	field := fmt.Sprintf("%d", productId)

	balance := types.PerpBalance{
		Amount:          updatedAmount,
		VQuoteBalance:   updatedVQuoteBalance,
		LastFundingRate: lastFundingRate,
	}

	perpBalanceJSON, err := json.Marshal(balance)
	if err != nil {
		return fmt.Errorf("unable to marshal perp balance data: %v", err)
	}

	err = s.redisClient.HSet(ctx, hashKey, field, perpBalanceJSON).Err()
	if err != nil {
		return fmt.Errorf("unable to update perp balance: %v", err)
	}

	return nil
}

func (s *BalanceService) GetNonce(subAccountId string) (string, error) {
	ctx := context.Background()

	subAccountId = strings.ToLower(subAccountId)
	key := xredis.GetNonceKey(subAccountId)

	nonce, err := s.redisClient.Get(ctx, key).Result()
	if err != nil && err != redis.Nil {
		return "0", err
	}

	if err == redis.Nil {
		return "0", nil
	}

	return nonce, nil
}

// Update the balance in memory
// Returns new state and tell whether update was healthy or not
// NOTE: Taking safety margin into account to avoid failing trades at edges when full available margin is used
func (t *BalanceService) UpdateLocalBalanceForOrderMatch(subaccountBalance subaccountTypes.SubaccountBalances, balancePerpRequest types.BalancePerpPayload, productId uint32, isTaker bool, oraclePricesMap map[string]ctypes.OraclePrice, perpetualMarketsMap map[uint]subaccountTypes.PerpetualMarket, cumulativeFundingRateMap map[string]*big.Int) (balance *subaccountTypes.SubaccountBalances, isHealthyBalanceUpdate bool, realisedPnl *big.Int, fundingFees *big.Int) {
	balance, isHealthyBalanceUpdate, realisedPnl, fundingFees, _ = t.UpdateLocalBalanceForOrderMatchWithFee(subaccountBalance, balancePerpRequest, productId, isTaker, oraclePricesMap, perpetualMarketsMap, cumulativeFundingRateMap)
	return
}

// UpdateLocalBalanceForOrderMatchWithFee also returns the trading fee deducted (credited to the
// D-7 fee account by the caller). The AMM skips health but is bound by the market's
// AmmMaxPositionx18 cap, like the NEAR contract.
func (t *BalanceService) UpdateLocalBalanceForOrderMatchWithFee(subaccountBalance subaccountTypes.SubaccountBalances, balancePerpRequest types.BalancePerpPayload, productId uint32, isTaker bool, oraclePricesMap map[string]ctypes.OraclePrice, perpetualMarketsMap map[uint]subaccountTypes.PerpetualMarket, cumulativeFundingRateMap map[string]*big.Int) (balance *subaccountTypes.SubaccountBalances, isHealthyBalanceUpdate bool, realisedPnl *big.Int, fundingFees *big.Int, feex18 *big.Int) {
	previousSafetyMarginx18 := t.getSafetyMarginV2(subaccountBalance, oraclePricesMap, perpetualMarketsMap, cumulativeFundingRateMap)
	symbol, exists := marketutils.GetFundingSymbolForProduct(productId)
	if !exists {
		xlog.Errorf("Symbol not found for productID %v", productId)
		return nil, false, nil, nil, nil
	}
	oldAmountx18 := new(big.Int).Set(subaccountBalance.MustGetPerpBalance(productId).Amountx18)
	realisedPnlx18, fundingFeesx18 := subaccountBalance.UpdatePerpBalance(productId, balancePerpRequest.Amountx18, balancePerpRequest.VQuoteBalancex18, cumulativeFundingRateMap[symbol])
	feex18 = subaccountBalance.DeductTradingFee(balancePerpRequest.VQuoteBalancex18, isTaker)
	if subaccountBalance.IsAMMAccount() {
		newAmountx18 := subaccountBalance.MustGetPerpBalance(productId).Amountx18
		return &subaccountBalance, perpetualMarketsMap[uint(productId)].AmmPositionAllowed(oldAmountx18, newAmountx18), realisedPnlx18, fundingFeesx18, feex18
	}
	safetyMarginx18 := t.getSafetyMarginV2(subaccountBalance, oraclePricesMap, perpetualMarketsMap, cumulativeFundingRateMap)
	return &subaccountBalance, safetyMarginx18.Sign() >= 0 || safetyMarginx18.Cmp(previousSafetyMarginx18) >= 0, realisedPnlx18, fundingFeesx18, feex18
}

type ErrorUpdatingBalance struct {
	TakerError bool
	MakerError bool
}

func (e *ErrorUpdatingBalance) Error() string {
	if e.TakerError && e.MakerError {
		return "Error updating balance for both taker and maker"
	}
	if e.TakerError {
		return "Error updating taker balance. Most probably account has low margin"
	}
	if e.MakerError {
		return "Error updating maker balance. Most probably account has low margin"
	}
	return ""
}

type NOP struct{}

// Caching at 500ms | 5s | 24h
// Fetch balances from Redis.
// Fetch app state.
// Update the balance in memory.
// Verify that the available margin is not negative.
func (t *BalanceService) AtomicUpdateBalanceForOrderMatch(request types.UpdateSubaccountForMatchRequest) (*ErrorUpdatingBalance, *big.Int, *big.Int, *big.Int, *big.Int) {
	xlog.Infof("BS - Updating balances for order match: Maker: %v | Taker: %v | Maker Order Id: %v | Taker Order Id: %v", request.Maker.SubaccountId, request.Taker.SubaccountId, request.Maker.OrderId, request.Taker.OrderId)
	// Fetch app state
	oraclePricesMap, perpetualMarketsMap, cumulativeFundingRateMap, err := t.appState.GetAppState(appstate.GetAppConfig(500*time.Millisecond, 200*time.Millisecond, 24*time.Hour))

	if err != nil {
		xlog.Errorf("BS - Error while fetching app state: %v. This means there is a major issue either in redis or oracle.. Cancelling taker order and keeping maker order", err)
		return &ErrorUpdatingBalance{MakerError: false, TakerError: true}, nil, nil, nil, nil
	}

	var makerRealizedPnl *big.Int
	var takerRealizedPnl *big.Int
	var makerFundingFees *big.Int
	var takerFundingFees *big.Int

	// The locks wrap the pipeline, so the writes are EXECuted while both balances are still locked.
	// (Before, TxPipelined ran EXEC after its callback returned, i.e. after the locks were released,
	// so a concurrent match on the same subaccount could read the old balance and lose an update.)
	var feesx18 *big.Int
	_, err = xredis.WithRedisLock(xredis.GetBalanceRedisLockKey(request.Taker.SubaccountId), func() (*NOP, error) {
		return xredis.WithRedisLock(xredis.GetBalanceRedisLockKey(request.Maker.SubaccountId), func() (*NOP, error) {
			var errUpdateBalance ErrorUpdatingBalance

			// Fetch maker and taker's balances
			subaccountBalances := t.subaccountBalance.MustGetSubaccountBalancesFromIds([]string{request.Maker.SubaccountId, request.Taker.SubaccountId})
			oldSubaccountMakerAmount := subaccountBalances[0].MustGetPerpBalance(uint32(request.MarketId)).Amountx18
			oldSubaccountTakerAmount := subaccountBalances[1].MustGetPerpBalance(uint32(request.MarketId)).Amountx18
			// Try updating the balance for maker
			newMakerBalance, isHealthyBalanceUpdate, realizedPnl, fundingFees, makerFee := t.UpdateLocalBalanceForOrderMatchWithFee(subaccountBalances[0], request.Maker, uint32(request.MarketId), false, oraclePricesMap, perpetualMarketsMap, cumulativeFundingRateMap)
			makerRealizedPnl = realizedPnl
			makerFundingFees = fundingFees
			if !isHealthyBalanceUpdate {
				xlog.Errorf("BS - Balance update wasn't healthy for maker subaccount %s. Cancelling update", request.Maker.SubaccountId)
				errUpdateBalance.MakerError = true
			}

			// Try updating the balance for taker
			newTakerBalance, isHealthyBalanceUpdate, realizedPnl, fundingFees, takerFee := t.UpdateLocalBalanceForOrderMatchWithFee(subaccountBalances[1], request.Taker, uint32(request.MarketId), true, oraclePricesMap, perpetualMarketsMap, cumulativeFundingRateMap)
			takerRealizedPnl = realizedPnl
			takerFundingFees = fundingFees
			if !isHealthyBalanceUpdate {
				xlog.Errorf("BS - Balance update wasn't healthy for taker subaccount %s. Cancelling update", request.Taker.SubaccountId)
				errUpdateBalance.TakerError = true
			}

			if errUpdateBalance.MakerError || errUpdateBalance.TakerError {
				return nil, &errUpdateBalance
			}

			if request.Maker.SubaccountId != contractUtils.AMM_SUBACCOUNT_ID {
				errM := t.perpUtils.AddLongShortOIAtReddis(uint32(request.MarketId), oldSubaccountMakerAmount, request.Maker.Amountx18)
				if errM != nil {
					xlog.Infof("Error updating OI position for productID in maker order %d: %v\n", uint32(request.MarketId), errM)
				}
			}

			if request.Taker.SubaccountId != contractUtils.AMM_SUBACCOUNT_ID {
				errT := t.perpUtils.AddLongShortOIAtReddis(uint32(request.MarketId), oldSubaccountTakerAmount, request.Taker.Amountx18)
				if errT != nil {
					xlog.Infof("Error updating OI position for productID in taker order %d: %v\n", uint32(request.MarketId), errT)
				}
			}

			_, errPipe := xredis.GetRedisClient().TxPipelined(context.Background(), func(pipe redis.Pipeliner) error {
				if err := subaccount.ReplaceBalanceInRedis(pipe, newMakerBalance); err != nil {
					errUpdateBalance.MakerError = true
					return &errUpdateBalance
				}
				if err := subaccount.ReplaceBalanceInRedis(pipe, newTakerBalance); err != nil {
					errUpdateBalance.TakerError = true
					return &errUpdateBalance
				}
				// D-7: queued atomically with the trade, folded into the fee account below
				subaccount.AccrueFee(pipe, makerFee)
				subaccount.AccrueFee(pipe, takerFee)
				if err := transaction.WriteBalanceUpdateForTxn(pipe, *request.TxnCounter, transaction.TxnBalanceUpdate{
					Subaccount1: *newMakerBalance,
					Subaccount2: *newTakerBalance,
				}); err != nil {
					xlog.Errorf("BS - Error while writing balance update for order match and txnCounter: %v| err: %v", request.TxnCounter, err)
					// Do not return error here. This is not a critical error
				}
				return nil
			})
			if errPipe != nil {
				return nil, errPipe
			}
			feesx18 = new(big.Int).Add(makerFee, takerFee)
			return nil, nil
		})
	})
	if err == nil && feesx18 != nil && feesx18.Sign() > 0 {
		if _, errFold := subaccount.FoldFeeAccruals(xredis.GetRedisClient(), t.subaccountBalance); errFold != nil {
			// Not lost: the fees stay queued and the next fold picks them up.
			xlog.Errorf("BS - fee accrual fold failed (fees stay queued): %v", errFold)
		}
	}

	var errUpdateBalance *ErrorUpdatingBalance
	if err != nil {
		if errors.As(err, &errUpdateBalance) {
			xlog.Errorf("BS - Error while updating balance for order match: %v", err)
			return errUpdateBalance, makerRealizedPnl, takerRealizedPnl, makerFundingFees, takerFundingFees
		} else {
			xlog.Errorf("Looks like there was some error while updating balance in redis for order match: %v....Cancelling taker order and keeping maker order", err)
			return &ErrorUpdatingBalance{MakerError: false, TakerError: true}, nil, nil, nil, nil
		}
	}

	return errUpdateBalance, makerRealizedPnl, takerRealizedPnl, makerFundingFees, takerFundingFees
}

// Caching at 1sec | 10s | 24h
// VVVVVVIMP NOTE: While calculating total equity we are removing locked funds (initial margin of all open orders) but same is not done in liquidation
func (bs *BalanceService) GetHealth(subaccountID string) (*types.Health, error) {
	oraclePricesMap, perpetualMarketsMap, cumulativeFundingRateMap, err := bs.appState.GetAppState(appstate.GetAppConfig(1*time.Second, 10*time.Second, 24*time.Hour))
	if err != nil {
		xlog.Errorf("BS - Error while fetching app state: %v", err)
		return nil, err
	}

	subaccountBalance := bs.subaccountBalance.MustGetSubaccountBalancesFromIds([]string{subaccountID})[0]
	maintenanceMarginx36, initialMarginx36 := subaccount.GetRequiredMarginValues(subaccountBalance, perpetualMarketsMap, oraclePricesMap)

	totalEquityx36 := subaccount.GetTotalEquityx36(subaccountBalance, oraclePricesMap, cumulativeFundingRateMap)
	lockedBalancex36 := subaccount.GetLockedValuex36(subaccountBalance, oraclePricesMap)
	totalEquityx36 = new(big.Int).Sub(totalEquityx36, lockedBalancex36)

	return &types.Health{
		BelowInitialMargin:     initialMarginx36.Sign() != 0 && totalEquityx36.Cmp(initialMarginx36) < 0,
		BelowMaintenanceMargin: maintenanceMarginx36.Sign() != 0 && totalEquityx36.Cmp(maintenanceMarginx36) < 0,
		RequireInsurance:       initialMarginx36.Sign() == 0 && totalEquityx36.Sign() < 0,
	}, nil
}

func (s *BalanceService) SettlePnLForSubaccounts(subaccountIDs []string) (map[uint32]*big.Int, map[string]string, error) {

	// Fetch all oracle prices
	oraclePrices, err := appstate.NewAppState().GetAllOraclePrices()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch oracle prices: %w", err)
	}

	// Fetch token prices for all product IDs
	tokenPricesx18 := make(map[uint32]*big.Int)
	for _, productID := range contractUtils.ALL_COLLATERAL_SPOTS {
		if symbol, exists := marketutils.GetBaseSymbolForProduct(productID); exists {
			if priceData, exists := oraclePrices[symbol]; exists {
				tokenPricesx18[productID] = new(big.Int).Set(priceData.Pricex18)
			} else {
				xclient.GlobalDiscordClient.SendWebhookMessage("Price doesnt exist for symbol: " + symbol)
				continue
			}
		} else {
			xlog.Errorf("Symbol for productID %d not found", productID)
			continue
		}
	}

	failedSubAccounts := make(map[string]string)
	// Process each subaccount
	for _, subaccountID := range subaccountIDs {
		_, err := xredis.WithRedisLock(xredis.GetBalanceRedisLockKey(subaccountID), func() (*NOP, error) {
			subaccountBalance := s.subaccountBalance.MustGetSubaccountBalancesFromIds([]string{subaccountID})[0]

			quoteToSettleX18 := subaccountBalance.MustGetSpotBalance(contractUtils.QUOTE_TOKEN_PRODUCT_ID).Balancex18
			if quoteToSettleX18.Sign() >= 0 {
				return nil, fmt.Errorf("quote token balance >= 0 nothing to settle")
			}
			transaction.IncrementOverAllSettledPnl(quoteToSettleX18.Int64())
			quoteToSettleX36 := cutils.Mulx18(quoteToSettleX18)
			subaccountBalance.UpdateSpotBalance(contractUtils.QUOTE_TOKEN_PRODUCT_ID, new(big.Int).Neg(quoteToSettleX18))
			SettleLiqPnlUsingSpots(tokenPricesx18, quoteToSettleX36, &subaccountBalance)
			_, errInner := xredis.GetRedisClient().TxPipelined(context.Background(), func(pipe redis.Pipeliner) error {
				// Update the subaccount balance in redis
				return subaccount.ReplaceBalanceInRedis(pipe, &subaccountBalance)
			})

			return nil, errInner
		})
		if err != nil {
			xlog.Errorf("Failed to update balance while settling pnl for subAccount: %v with error: %v", subaccountID, err)
			failedSubAccounts[subaccountID] = err.Error()
		}
	}

	return tokenPricesx18, failedSubAccounts, nil
}

// NOTE: This should be used in read only endpoint as it doesn't acquire lock
// Cache time for oracle prices: 500ms
func (bs *BalanceService) GetTotalSpotValuex18(subaccountHex string) (*big.Int, error) {
	subaccountBalance := bs.subaccountBalance.MustGetSubaccountBalancesFromIds([]string{subaccountHex})[0]
	oraclePricesMap, err := bs.appState.GetAllOraclePrices(500 * time.Millisecond)
	if err != nil {
		return nil, err
	}

	return cutils.Divx18(subaccount.GetTotalSpotBalancex36(subaccountBalance, oraclePricesMap)), nil
}

func (s *BalanceService) SettleBroker2PnL(subaccountID string) error {
	_, err := xredis.WithRedisLock(xredis.GetBalanceRedisLockKey(subaccountID), func() (*NOP, error) {
		subaccountBalance := s.subaccountBalance.MustGetSubaccountBalancesFromIds([]string{subaccountID})[0]

		// Get balance from product ID 4 (original quote token)
		quoteBalance := subaccountBalance.MustGetSpotBalance(contractUtils.QUOTE_TOKEN_PRODUCT_ID).Balancex18

		// Only transfer if balance is positive
		if quoteBalance.Sign() <= 0 {
			return nil, fmt.Errorf("quote token balance is not positive, nothing to transfer (balance: %s)", quoteBalance.String())
		}

		xlog.Infof("Broker 2 PnL Settlement: Transferring %s from product ID %d to product ID %d for subaccount %s",
			quoteBalance.String(), contractUtils.QUOTE_TOKEN_PRODUCT_ID, contractUtils.BROKER_2_UPDATED_QUOTE_TOKEN_PRODUCT_ID, subaccountID)

		// Transfer the balance: subtract from product ID 4, add to product ID 72
		subaccountBalance.UpdateSpotBalance(contractUtils.QUOTE_TOKEN_PRODUCT_ID, new(big.Int).Neg(quoteBalance))
		subaccountBalance.UpdateSpotBalance(contractUtils.BROKER_2_UPDATED_QUOTE_TOKEN_PRODUCT_ID, quoteBalance)

		// Update the subaccount balance in Redis
		_, errInner := xredis.GetRedisClient().TxPipelined(context.Background(), func(pipe redis.Pipeliner) error {
			return subaccount.ReplaceBalanceInRedis(pipe, &subaccountBalance)
		})

		return nil, errInner
	})

	if err != nil {
		xlog.Errorf("Failed to settle broker 2 PnL for subaccount %s: %v", subaccountID, err)
		return err
	}

	return nil
}

func (s *BalanceService) ShiftKromaFunds(subaccountID string) error {
	updatedBalance := false
	_, err := xredis.WithRedisLock(xredis.GetBalanceRedisLockKey(subaccountID), func() (*NOP, error) {
		subaccountBalance := s.subaccountBalance.MustGetSubaccountBalancesFromIds([]string{subaccountID})[0]

		kromaToArbMapping := map[uint32]uint32{
			contractUtils.KROMA_USDC: contractUtils.ARB_USDC,
			contractUtils.KROMA_USDT: contractUtils.ARB_USDT,
		}

		for kromaProductID, arbProductID := range kromaToArbMapping {
			kromaBalance := subaccountBalance.MustGetSpotBalance(kromaProductID).Balancex18

			if kromaBalance.Sign() > 0 {
				xlog.Infof("Kroma Funds Shift: Transferring %s from product ID %d to product ID %d for subaccount %s",
					kromaBalance.String(), kromaProductID, arbProductID, subaccountID)

				subaccountBalance.UpdateSpotBalance(kromaProductID, new(big.Int).Neg(kromaBalance))
				subaccountBalance.UpdateSpotBalance(arbProductID, kromaBalance)
				updatedBalance = true
			}
		}

		if !updatedBalance {
			xlog.Infof("No Kroma funds to shift for subaccount %s", subaccountID)
			return nil, fmt.Errorf("no Kroma funds to shift for subaccount %s", subaccountID)
		}

		// Update the user subaccount balance in Redis
		_, errInner := xredis.GetRedisClient().TxPipelined(context.Background(), func(pipe redis.Pipeliner) error {
			return subaccount.ReplaceBalanceInRedis(pipe, &subaccountBalance)
		})

		return nil, errInner
	})

	if err != nil {
		xlog.Errorf("Failed to shift Kroma funds for subaccount %s: %v", subaccountID, err)
		return err
	}

	return nil
}

func (s *BalanceService) BurnKromaFunds(subaccountID string) error {
	updated := false
	_, err := xredis.WithRedisLock(xredis.GetBalanceRedisLockKey(subaccountID), func() (*NOP, error) {
		subaccountBalance := s.subaccountBalance.MustGetSubaccountBalancesFromIds([]string{subaccountID})[0]

		kromaProductIDs := []uint32{contractUtils.KROMA_USDC, contractUtils.KROMA_USDT}

		for _, kromaProductID := range kromaProductIDs {
			kromaBalance := subaccountBalance.MustGetSpotBalance(kromaProductID).Balancex18

			if kromaBalance.Sign() > 0 {
				xlog.Infof("Kroma Funds Burn: Burning %s from product ID %d for subaccount %s",
					kromaBalance.String(), kromaProductID, subaccountID)

				subaccountBalance.UpdateSpotBalance(kromaProductID, new(big.Int).Neg(kromaBalance))
				updated = true
			}
		}

		if !updated {
			xlog.Infof("No Kroma funds to burn for subaccount %s", subaccountID)
			return nil, fmt.Errorf("no Kroma funds to burn for subaccount %s", subaccountID)
		}
		// Update the user subaccount balance in Redis
		_, errInner := xredis.GetRedisClient().TxPipelined(context.Background(), func(pipe redis.Pipeliner) error {
			return subaccount.ReplaceBalanceInRedis(pipe, &subaccountBalance)
		})

		return nil, errInner
	})

	if err != nil {
		xlog.Errorf("Failed to burn Kroma funds for subaccount %s: %v", subaccountID, err)
		return err
	}

	return nil
}
