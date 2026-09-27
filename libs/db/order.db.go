package db

import (
	"fmt"
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
)

const ORDER_EXPIRY_DURATION = 30 * cutils.DAY_MILLI

type OrderDB struct{}

// TODO: Handle errors
func (*OrderDB) CreateOrder(orderData OrderTable) *OrderTable {
	db.Create(&orderData)
	return &orderData
}

func (*OrderDB) GetById(id uint) *OrderTable {
	order := OrderTable{}
	return GetDbObjOrNil(db.First(&order, id), &order)
}

// Update some columns of the order row
func (*OrderDB) Update(order *OrderTable, columns map[string]interface{}) *OrderTable {
	return GetDbObjOrNil(db.Model(order).Updates(columns), order)
}

func (*OrderDB) GetBySubaccount_Id(subaccountId string, id uint) *OrderTable {
	order := OrderTable{}
	return GetDbObjOrNil(db.Where("subaccount_id = ? AND id = ?", subaccountId, id).First(&order), &order)
}

func (*OrderDB) GetAllBySubaccount(subaccountId string) *[]OrderTable {
	orders := []OrderTable{}
	return GetDBObjOrEmptyList(db.Where("subaccount_id = ?", subaccountId).Find(&orders), &orders)
}
func (o *OrderDB) GetAllOpenLimitOrdersBySubaccount(subaccountID string) *[]OrderTable {

	orders := []OrderTable{}
	query := db.Where("subaccount_id = ? AND type = ? AND status != ? AND status!= ?", subaccountID, ctypes.ORDER_TYPE_LIMIT, ctypes.ORDER_STATUS_CANCELLED, ctypes.ORDER_STATUS_FILLED)
	result := query.Find(&orders)

	if result.Error != nil {
		fmt.Printf("Error executing query: %v\n", result.Error)
	}

	return GetDBObjOrEmptyList(result, &orders)
}

// Returns open limit orders and open conditional orders
func (o *OrderDB) GetAllOpenOrders(subaccountId string) *[]OrderTable {
	orders := []OrderTable{}
	query := db.Where("subaccount_id = ? AND (status = ? OR status = ?)", subaccountId, ctypes.ORDER_STATUS_OPEN, ctypes.ORDER_STATUS_PARTIAL)
	result := query.Find(&orders)

	if result.Error != nil {
		fmt.Printf("Error executing query: %v\n", result.Error)
	}

	return GetDBObjOrEmptyList(result, &orders)
}

func (*OrderDB) GetMultipleByIds(ids []uint) *[]OrderTable {
	orders := []OrderTable{}
	return GetDBObjOrEmptyList(db.Where("id IN ?", ids).Find(&orders), &orders)
}

// TODO: Add pagination
func (*OrderDB) GetAllCancellableOrdersForMarket(subaccountId string, marketId uint) *[]OrderTable {
	orders := []OrderTable{}
	// Get all orders that are open or partially filled and match the given side
	return GetDBObjOrEmptyList(db.Where("subaccount_id = ? AND (status = ? OR status = ?) AND market_id = ?", subaccountId, ctypes.ORDER_STATUS_OPEN, ctypes.ORDER_STATUS_PARTIAL, marketId).Find(&orders), &orders)
}

func (*OrderDB) GetAllCancellableOrdersWithSideForMarket(subaccountId string, marketId uint, side ctypes.OrderSide) *[]OrderTable {
	orders := []OrderTable{}
	// Get all orders that are open or partially filled and match the given side
	return GetDBObjOrEmptyList(db.Where("subaccount_id = ? AND (status = ? OR status = ?) AND market_id = ? AND side = ?", subaccountId, ctypes.ORDER_STATUS_OPEN, ctypes.ORDER_STATUS_PARTIAL, marketId, side).Find(&orders), &orders)
}

func (*OrderDB) UpdateMultiple(orders *[]OrderTable, columns map[string]interface{}) *[]OrderTable {
	return GetDBObjOrEmptyList(db.Model(orders).Updates(columns), orders)
}

func (*OrderDB) CountDecreaseOrderByType(subaccountId string, side string, marketId int64) (int64, error) {
	var count int64
	err := db.Raw(`
		SELECT COUNT(*)
		FROM order_tables
		WHERE subaccount_id = ? AND side = ? AND market_id = ? AND is_reduce = true
    `, subaccountId, side, marketId).Scan(&count).Error

	if err != nil {
		fmt.Println("Error querying database:", err)
		return 0, err
	}
	return count, nil
}

func (*OrderDB) IsSignatureUsed(signature string) bool {
	var count int64
	// Check if the signature is used in any order
	db.Raw(`SELECT COUNT(*) FROM order_tables WHERE signature = ?`, signature).Scan(&count)
	return count > 0
}

// Warning: Do not use this anywhere else except in cron service

func (*OrderDB) DeleteUnmatchedAMMOrders() error {
	result := db.Exec(`DELETE FROM order_tables WHERE party = 'SOLVER' and status='CANCELLED' and total_filledx18='0'`)
	xlog.Infof("Deleted %d unmatched AMM orders", result.RowsAffected)
	return result.Error
}

// ------------------ NOT FULLY TESTED ------------------
// ------------------ NOT FULLY TESTED ------------------
// ------------------ NOT FULLY TESTED ------------------
