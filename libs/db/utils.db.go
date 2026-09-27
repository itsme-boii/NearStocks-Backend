package db

import (
	"errors"
	"fmt"
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"math/big"
	"strconv"

	"gorm.io/gorm"
)

func GetDbObjOrNil[T interface{}](result *gorm.DB, dbObj *T) *T {
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		fmt.Println("Record not found returning nil")
		return nil
	} else if result.Error != nil {
		xlog.Errorf("Some error occurred: %v", result.Error)
		return nil
	} else if result.RowsAffected == 0 {
		fmt.Println("Couldn't change any row")
		return nil
	}
	return dbObj
}

func GetDBObjOrEmptyList[T interface{}](result *gorm.DB, dbObj *[]T) *[]T {
	ret := GetDbObjOrNil(result, dbObj)
	if ret == nil {
		return (&[]T{})
	} else {
		return ret
	}
}

func ParseValue(value any, err error) any {
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		panic(err)
	}
	return value
}

// NOTE: ID, CreatedAt, UpdatedAt, DeletedAt are not included in the struct
func StringMapToOrderTable(data map[string]string, target *OrderTable) error {
	// fmt.Printf("Data: %v\n", data)
	var ok bool
	target.Amountx18.Val, ok = new(big.Int).SetString(data["amount_x18"], 10)
	if !ok {
		return fmt.Errorf("Error parsing amount_x18: %+v", data)
	}

	target.BrokerId = uint(ParseValue(strconv.ParseUint(data["broker_id"], 10, 64)).(uint64))
	target.ExpiryTs = ParseValue(strconv.ParseUint(data["expiry_ts"], 10, 64)).(uint64)
	target.MarketId = uint(ParseValue(strconv.ParseUint(data["market_id"], 10, 64)).(uint64))

	target.TotalFilledx18.Val, ok = new(big.Int).SetString(data["total_filled_x18"], 10)
	if !ok {
		return fmt.Errorf("Error parsing total_filled_x18: %+v", data)
	}

	target.Pricex18.Val, ok = new(big.Int).SetString(data["price_x18"], 10)
	if !ok {
		return fmt.Errorf("Error parsing price_x18: %+v", data)
	}

	trigger_price_x18, exists := data["trigger_price_x18"]
	if !exists || trigger_price_x18 == "" {
		target.TriggerPricex18.Val = big.NewInt(0)
	} else {
		target.TriggerPricex18.Val, ok = new(big.Int).SetString(trigger_price_x18, 10)
		if !ok {
			target.TriggerPricex18.Val = big.NewInt(0)
			xlog.Errorf("Error parsing trigger_price_x18: %+v", data)
		}
	}

	target.Side = ctypes.OrderSide(data["side"])
	target.Status = ctypes.OrderStatus(data["status"])
	target.Side = ctypes.OrderSide(data["side"])
	target.Signature = data["signature"]
	target.SessionKey = data["session_key"]
	target.SubaccountId = data["subaccount_id"]
	target.Type = ctypes.OrderType(data["type"])
	target.Party = ctypes.Party(data["party"])
	target.Timestamp = ParseValue(strconv.ParseUint(data["timestamp"], 10, 64)).(uint64)
	target.IsReduce = data["is_reduce"] == "1"

	if triggerCondition, exists := data["trigger_condition"]; !exists {
		target.TriggerCondition = ctypes.TriggerCondition("")
	} else {
		target.TriggerCondition = ctypes.TriggerCondition(triggerCondition)
	}
	return nil
}
