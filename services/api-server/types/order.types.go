package types

import (
	"fmt"
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"math/big"
	"net/http"
	"time"
)

type LiquidationOrderRequest struct {
	SubaccountId  string              `json:"subaccountId" binding:"required"`     // This is the subaccount of trader getting liquididated. It should be in string format <BROKER_ID>_<CHECKSUM_ETH_ADDRESS>_<SUBCCOUNT_NUMBER>
	Party         ctypes.Party        `json:"party" binding:"oneof=TRADER SOLVER"` // Currently we will be setting this to TRADER as default value
	AmountStr     string              `json:"amount" binding:"required"`           // Amount float string
	MarketId      uint                `json:"marketId" binding:"required"`         // This is same as ProductId. Keeping MarketId for consistency
	IsBuy         *bool               `json:"isBuy" binding:"required"`
	SpotPricesx18 map[uint32]*big.Int `json:"spotPricesx18" binding:"required"`
	PerpPricesx18 map[uint32]*big.Int `json:"perpPricesx18" binding:"required"`
}

type ConditionalOrderRequest struct {
	Id uint `json:"id"`
}

type SettleWithInsuranceRequest struct {
	SubaccountId string `json:"subaccountId" binding:"required"` // It should be in string format <BROKER_ID>_<CHECKSUM_ETH_ADDRESS>_<SUBCCOUNT_NUMBER>
}

type SettleSubaccountPnl struct {
	SubaccountId string `json:"subaccountId" binding:"required"`
}

type PlaceOrderResponse struct {
	Id               uint                    `json:"id"`
	SubaccountId     string                  `json:"subaccountId"`
	BrokerId         uint                    `json:"brokerId"`
	MarketType       ctypes.MarketType       `json:"marketType"`
	MarketSymbol     string                  `json:"marketSymbol"`
	Type             ctypes.OrderType        `json:"type"`
	Side             ctypes.OrderSide        `json:"side"`
	Amountx18        string                  `json:"amountx18"`
	TotalFilledx18   string                  `json:"totalFilledx18"`
	Pricex18         string                  `json:"pricex18"`
	ExpiryTs         uint64                  `json:"expiryTs"`
	Status           ctypes.OrderStatus      `json:"status"`
	CreatedAt        time.Time               `json:"createdAt"`
	UpdatedAt        time.Time               `json:"updatedAt"`
	TriggerPricex18  string                  `json:"triggerPricex18"`
	TriggerCondition ctypes.TriggerCondition `json:"triggerCondition"`
}

// No need to check expiry time since we are performing IOC
// No checks on minimum amount since we are performing liquidation
// No signature verification since this is coming from liquidation engine
func (lor *LiquidationOrderRequest) ValidateAndGetOrder(subaccount *db.SubaccountTable, market *db.MarketTable) (liquidationOrder *db.OrderTable, status int, err error) {

	// Check if amount is multiple of quantums
	if err = cutils.QuantumPrecisionCheck(lor.AmountStr, market.AmtToQtmConversionExpo); err != nil {
		return nil, http.StatusBadRequest, err
	}

	bigIntAmountx18 := ctypes.NewBigInt(cutils.FloatStrToX18(lor.AmountStr))
	// Verify amount is positive
	if bigIntAmountx18.Sign() < 0 {
		return nil, http.StatusBadRequest, fmt.Errorf("amount should be positive")
	}

	liquidationOrder = &db.OrderTable{
		SubaccountId:     subaccount.ID,
		BrokerId:         subaccount.BrokerId,
		MarketId:         lor.MarketId,
		Side:             ctypes.NewOrderSide(*lor.IsBuy),
		Party:            lor.Party,
		Type:             ctypes.ORDER_TYPE_LIQUIDATION,
		TotalFilledx18:   ctypes.NewBigInt(big.NewInt(0)),
		ExpiryTs:         cutils.TimestampMilliNow() + cutils.MINUTE_MILLI, // This really doesn't matter since we are performing IOC
		Status:           ctypes.ORDER_STATUS_OPEN,
		Timestamp:        cutils.TimestampMilliNow(),
		IsReduce:         true, // Liquidation orders are always reduce only
		Signature:        "",   // No signature for liquidation orders
		SessionKey:       "",   // No session key for liquidation orders
		Amountx18:        bigIntAmountx18,
		Pricex18:         ctypes.NewBigInt(big.NewInt(0)),
		TriggerPricex18:  ctypes.NewBigInt(big.NewInt(0)),
		TriggerCondition: "",
	}

	return liquidationOrder, http.StatusOK, nil
}

type CreateBulkOrderHeader struct {
	SignerAddress string `header:"Logx-Signer-Address" binding:"required"`
	BrokerId      uint   `header:"Logx-Broker-Id" binding:"required"`
}

type SingleOrderBody struct {
	Signature string                  `json:"signature" binding:"required"`
	Details   *ctypes.CreateOrderBody `json:"details" binding:"required"`
}

type CreateBulkOrderBody struct {
	Orders   []SingleOrderBody `json:"orders" binding:"required"`
	MarketId *uint             `json:"marketId" binding:"required"`
	IsBuy    *bool             `json:"isBuy" binding:"required"` // This is the order side of the market. It can be either BUY or SELL
}

type RejectedOrder struct {
	Order  db.OrderTable
	Reason string `json:"reason"`
}

type CreateBulkOrderResponse struct {
	Orders         *[]db.OrderTable `json:"orders"`
	RejectedOrders []RejectedOrder  `json:"rejectedOrders"`
}
