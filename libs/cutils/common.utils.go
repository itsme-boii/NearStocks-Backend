// TODO:@manan - AUDIT REQUIRED
package cutils

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"log"
	"os"
	"strings"
	"time"

	"github/eugenix-io/logx-inf-backend/libs/xredis"

	"github.com/ethereum/go-ethereum/common"
	"github.com/redis/go-redis/v9"
)

// ExtractETHAddress extracts the Ethereum wallet address from a given hex string.
func ExtractETHAddress(hexString string) (string, error) {
	if len(hexString) >= 2 && hexString[:2] == "0x" {
		hexString = hexString[2:]
	}
	bytes, err := hex.DecodeString(hexString)
	if err != nil {
		return "", fmt.Errorf("failed to decode hex string: %w", err)
	}
	if len(bytes) != 32 {
		return "", fmt.Errorf("unexpected byte slice length: got %d, want 32", len(bytes))
	}

	addressBytes := bytes[6:26]
	address := common.BytesToAddress(addressBytes).Hex()

	return address, nil
}

// Uint64ToBytes converts the given uint64 value to slice of bytes.
func Uint64ToBytes32(val uint64) [32]byte {
	b := make([]byte, 32)
	binary.LittleEndian.PutUint64(b, val)
	return [32]byte(b)
}

// HexToBytes32 converts a hex string to a [32]byte array
func HexToBytes32(hexStr string) ([32]byte, error) {
	var b32 [32]byte
	hexStr = strings.TrimPrefix(hexStr, "0x")

	if len(hexStr) != 64 {
		return b32, fmt.Errorf("hex string is not 64 characters long: %s", hexStr)
	}
	bytes, err := hex.DecodeString(hexStr)
	if err != nil {
		return b32, fmt.Errorf("failed to decode hex string: %v", err)
	}
	copy(b32[:], bytes)
	return b32, nil
}

// UintToPaddedString converts an int to string of length 19 by padding with 0
func UintToPaddedString(num int64) string {
	return fmt.Sprintf("%019d", num)
}

// GetTickChannelID is used to get the channel id for OHLCV data streaming
// it takes pairname, duration and units of data streaming
func GetTickChannelID(bt, qt common.Address, unit string, duration int64) string {
	pair := GetPairKey(bt, qt)
	return fmt.Sprintf("%s::%d::%s", pair, duration, unit)
}

// GetPairKey return the pair key identifier corresponding to two
func GetPairKey(bt, qt common.Address) string {
	return strings.ToLower(fmt.Sprintf("%s::%s", bt.Hex(), qt.Hex()))
}

func GetTradeChannelID(bt, qt common.Address) string {
	return strings.ToLower(fmt.Sprintf("%s::%s", bt.Hex(), qt.Hex()))
}

func GetOHLCVChannelID(bt, qt common.Address, unit string, duration int64) string {
	pair := GetPairKey(bt, qt)
	return fmt.Sprintf("%s::%d::%s", pair, duration, unit)
}

func GetOrderBookChannelID(bt, qt common.Address) string {
	return strings.ToLower(fmt.Sprintf("%s::%s", bt.Hex(), qt.Hex()))
}

func PrintJSON(x interface{}) {
	b, err := json.MarshalIndent(x, "", "  ")
	if err != nil {
		fmt.Println("Error: ", err)
	}

	fmt.Print(string(b), "\n")
}

// Function to check if a value exists in an array
func UintArrayContains(arr []uint, value uint) bool {
	for _, v := range arr {
		if v == value {
			return true
		}
	}
	return false
}

func PrintError(msg string, err error) {
	log.Printf("\n%v: %v\n", msg, err)
}

// Util function to handle unused variables while testing
func Use(...interface{}) {

}

func LogByParty(party ctypes.Party, format string, args ...interface{}) {
	if party == ctypes.PARTY_SOLVER {
		xlog.Debugf(format, args...)
	} else {
		xlog.Infof(format, args...)
	}
}

// Broker ID 2 has a fee 60 (6bps), all 1 has 0 (0bps: Only for leaderboard purposes)
// Multiply by 1e13
//
//go:noinline
func GetBrokerFeeFactor(brokerId int) int64 {
	if brokerId == 2 {
		return 60
	}
	return 0
}

// Returns true if the subaccount is internal and internal withdrawals are stopped
func IsInternalSubaccount(subaccountID string) (bool, error) {
	redisClient := xredis.GetRedisClient()
	stopInternalWithdrawal, err := redisClient.Get(context.Background(), xredis.GetStopInternalSubaccountWithdrawalKey()).Result()
	if err != nil && err.Error() != "redis: nil" {
		return false, fmt.Errorf("failed to get STOP_INTERNAL_SUBACCOUNT_WITHDRAWAL from Redis: %w", err)
	}

	// If Redis flag is not set to 1, allow all withdrawals
	if stopInternalWithdrawal != "1" {
		return false, nil
	}

	internalSubaccountIDs := GetTrimmedSplitEnv("INTERNAL_SUBACCOUNT_IDS", ",")
	return SliceExists(internalSubaccountIDs, subaccountID), nil
}

func GetTrimmedSplitValue(value, delimiter string) []string {
	if value == "" {
		return []string{}
	}
	
	splitValues := strings.Split(value, delimiter)
	trimmedValues := make([]string, 0, len(splitValues))

	for _, v := range splitValues {
		trimmedValue := strings.TrimSpace(v)
		if trimmedValue != "" {
			trimmedValues = append(trimmedValues, trimmedValue)
		}
	}
	
	return trimmedValues
}

func GetTrimmedSplitEnv(envKey, delimiter string) []string {
	envValue := os.Getenv(envKey)

	return GetTrimmedSplitValue(envValue, delimiter)
}

func GetStringValue(data map[string]interface{}, key string) (string, bool) {
	if val, ok := data[key].(string); ok {
		return val, true
	}
	return "", false
}

func GetInt64Value(data map[string]interface{}, key string) int64 {
	if val, ok := data[key].(int64); ok {
		return val
	}
	if val, ok := data[key].(float64); ok {
		return int64(val)
	}
	return 0
}

func GetTimeValue(data map[string]interface{}, key string) time.Time {
	if val, ok := data[key].(time.Time); ok {
		return val
	}
	if val, ok := data[key].(string); ok {
		formats := []string{
			"2006-01-02T15:04:05Z",
			"2006-01-02 15:04:05",
			time.RFC3339,
			"2006-01-02T15:04:05.000Z",
		}

		for _, format := range formats {
			if parsedTime, err := time.Parse(format, val); err == nil {
				return parsedTime
			}
		}
	}
	return time.Time{}
}

func GetFloat64Value(data map[string]interface{}, key string) float64 {
	if val, ok := data[key].(float64); ok {
		return val
	}
	if val, ok := data[key].(int64); ok {
		return float64(val)
	}
	return 0.0
}

// IsSubaccountFlagged checks if a subaccount is flagged in Redis.
// Returns true if the subaccount is flagged (value = "1"), false otherwise.
// Flagging states: 0 or non-existing = Not flagged, 1 = Flagged, 2 = Unflagged
func IsSubaccountFlagged(ctx context.Context, subaccountId string) (bool, error) {
	flaggedKey := xredis.GetFlaggedSubaccountsKey()
	redisClient := xredis.GetRedisClient()

	// Get the flag value from Redis
	flagValue, err := redisClient.HGet(ctx, flaggedKey, subaccountId).Result()
	if err != nil {
		if err == redis.Nil {
			// Key doesn't exist, not flagged
			return false, nil
		}
		return false, err
	}

	// Check if the value is "1" (flagged)
	return flagValue == "1", nil
}

// GetSubaccountFlagStatus returns the current flag status of a subaccount.
// Returns: "0" = not flagged, "1" = flagged, "2" = unflagged, "" = not found
func GetSubaccountFlagStatus(ctx context.Context, subaccountId string) (string, error) {
	flaggedKey := xredis.GetFlaggedSubaccountsKey()
	redisClient := xredis.GetRedisClient()

	flagValue, err := redisClient.HGet(ctx, flaggedKey, subaccountId).Result()
	if err != nil {
		if err == redis.Nil {
			// Key doesn't exist, return "0" (not flagged)
			return "0", nil
		}
		return "", err
	}

	return flagValue, nil
}
