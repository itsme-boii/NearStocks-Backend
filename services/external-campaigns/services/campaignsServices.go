package services

import (
	"fmt"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// DEX2 - BROKER ID is HARD CODED TO 1
func TradeCountForAddress(address string) (int, error) {
	checkSumAddress := common.HexToAddress(address).Hex()
	subaccountId := "1_" + checkSumAddress + "_1"

	// read trade count from db
	tradeCount, err := (&db.FillDB{}).TradeCount(subaccountId)
	if err != nil {
		xlog.Errorf("Rewards Controller - Error reading from db subaccountID: %v", err)
		return 0, err
	}

	return tradeCount, nil
}
func ReferralCountForAddress(address string) (int64, error) {
	checkSumAddress := common.HexToAddress(address).Hex()

	// read trade count from db
	referralCount, err := (&db.ReferralDB{}).CountActiveReferralsByReferrerID(checkSumAddress)
	if err != nil {
		xlog.Errorf("Rewards Controller - Error reading from db subaccountID: %v", err)
		return 0, err
	}

	return referralCount, nil
}

// DEX2 - BROKER ID is HARD CODED TO 2
func GetUserVolumeAfterCertainDate(address string, startTime time.Time) (*big.Float, error) {
	checkSumAddress := common.HexToAddress(address).Hex()
	subaccountId := "2_" + checkSumAddress + "_1"

	userVolume, err := (&db.FillDB{}).GetUserVolumeAfterCertainDate(subaccountId, startTime)
	if err != nil {
		return new(big.Float).SetFloat64(0), err
	}

	return userVolume, nil

}

// DEX2 - BROKER ID is HARD CODED TO 1
func GetOptionsVolumeAfterCertainDate(address string, startTime time.Time) (*big.Float, error) {
	checkSumAddress := common.HexToAddress(address).Hex()
	subaccountId := "1_" + checkSumAddress + "_1"

	userVolume, err := (&db.OptionsDB{}).GetOptionsVolumeAfterCertainDate(subaccountId, startTime)
	if err != nil {
		return new(big.Float).SetFloat64(0), err
	}

	return userVolume, nil

}

// DEX2 - BROKER ID is HARD CODED TO 1
func GetOptionsCountToday(address string) (int, error) {
	checkSumAddress := common.HexToAddress(address).Hex()
	subaccountId := "1_" + checkSumAddress + "_1"

	optionsCount, err := (&db.OptionsDB{}).HasTradedToday(subaccountId)
	if err != nil {
		return 0, err
	}

	return optionsCount, nil
}

// DEX2 - BROKER ID is HARD CODED TO 1
func CheckIfDepositPresentFromChain(address string, chainId uint64) (int64, error) {
	// convert address to subaccountid
	checkSumAddress := common.HexToAddress(address).Hex()
	subaccountId := "1_" + checkSumAddress + "_1"
	subaccountIdHex, err := cutils.SubaccountIdToHex(subaccountId)
	if err != nil {
		xlog.Errorf("Error converting subaccountId:", err)
		return 0, err
	}
	// check if deposit present in the deposit withdraw table
	count, err := (&db.DepositWithdrawDB{}).CountDepositsBySubaccountAndChain(subaccountIdHex, chainId)
	if err != nil {
		xlog.Errorf("Error counting deposits:", err)
		return 0, err
	}

	return count, nil
}

// DEX2 - BROKER ID is HARD CODED TO 1
func UserVolumeCheck(address string, startTime time.Time, brokerId int) (*big.Float, error) {
	checkSumAddress := common.HexToAddress(address).Hex()
	subaccountId := fmt.Sprintf("%d_%s_1", brokerId, checkSumAddress)
	totalVolumeStr, err := (&db.FillDB{}).GetUserVolumeAfterCertainDate(subaccountId, startTime)
	if err != nil {
		xlog.Errorf("Rewards Controller - Error reading from db subaccountID: %v", err)
		return new(big.Float).SetFloat64(0), err
	}

	return totalVolumeStr, nil
}
