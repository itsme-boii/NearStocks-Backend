package controller

import (
	"fmt"
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/services/api-server/types"

	"github.com/gin-gonic/gin"
)

// ----------------- SUBACCOUNT -----------------
func getCurrentSubaccount(ctx *gin.Context) (*db.SubaccountTable, error) {
	untypeSubaccount, found := ctx.Get("current-subaccount")
	if !found {
		return nil, fmt.Errorf("subaccount not found")
	}

	curSubaccount, ok := untypeSubaccount.(*db.SubaccountTable)

	if !ok {
		return nil, fmt.Errorf("current subaccount type assertion failed")
	}

	return curSubaccount, nil
}

// ----------------- ORDER -----------------
func getOrderResponse(order *db.OrderTable, market *db.MarketTable) gin.H {
	return gin.H{
		"id":             order.ID,
		"subaccountId":   order.SubaccountId,
		"brokerId":       order.BrokerId,
		"marketType":     market.Type,
		"marketSymbol":   market.Symbol,
		"type":           order.Type,
		"side":           order.Side,
		"amountx18":      order.Amountx18,
		"totalFilledx18": order.TotalFilledx18,
		"pricex18":       order.Pricex18,
		"expiryTs":       order.ExpiryTs,
		"status":         order.Status,
		"createdAt":      order.CreatedAt,
		"updatedAt":      order.UpdatedAt,
	}
}

type CreateOrderResponse struct {
	OrderDetails  gin.H `json:"orderDetails"`
	Miscellaneous gin.H `json:"miscellaneous"` // Always included
}

func getOrderResponseWithMisc(order *db.OrderTable, market *db.MarketTable, feeBonus ctypes.BigInt) CreateOrderResponse {
	// Reuse the getOrderResponse function to get order details
	orderDetails := getOrderResponse(order, market)

	// Add miscellaneous fields
	miscellaneous := gin.H{
		"feeBonus": feeBonus.Val.String(),
		// Other extra parameters can be added here in the future
	}

	// Return CreateOrderResponse struct
	return CreateOrderResponse{
		OrderDetails:  orderDetails,
		Miscellaneous: miscellaneous,
	}
}

func getOrderResponsev2(order *db.OrderTable, market *db.MarketTable) types.PlaceOrderResponse {
	return types.PlaceOrderResponse{
		Id:             order.ID,
		SubaccountId:   order.SubaccountId,
		BrokerId:       order.BrokerId,
		MarketType:     market.Type,
		MarketSymbol:   market.Symbol,
		Type:           order.Type,
		Side:           order.Side,
		Amountx18:      order.Amountx18.String(),
		TotalFilledx18: order.TotalFilledx18.String(),
		Pricex18:       order.Pricex18.String(),
		ExpiryTs:       order.ExpiryTs,
		Status:         order.Status,
		CreatedAt:      order.CreatedAt,
		UpdatedAt:      order.UpdatedAt,
		TriggerPricex18: order.TriggerPricex18.String(),
		TriggerCondition: order.TriggerCondition,
	}
}
