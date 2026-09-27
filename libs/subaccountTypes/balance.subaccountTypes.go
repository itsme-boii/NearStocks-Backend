package subaccountTypes

import (
	"encoding/json"
	"fmt"
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"math/big"
	"strconv"
	"strings"
)

type PerpetualMarket struct {
	ProductId                    uint
	BaseAsset                    string
	MaintenanceMarginFractionx18 *big.Int
	InitialMarginFractionx18     *big.Int
	AmmMaxPositionx18            *big.Int // nil or 0 = no cap
}

// AmmPositionAllowed reports whether the AMM may move from oldAmount to newAmount: always when the
// cap is unset or the position does not grow, otherwise only while |newAmount| <= cap.
// Same rule as the NEAR contract (apply_trade / tx_liquidate).
func (pm PerpetualMarket) AmmPositionAllowed(oldAmount, newAmount *big.Int) bool {
	if pm.AmmMaxPositionx18 == nil || pm.AmmMaxPositionx18.Sign() == 0 {
		return true
	}
	absNew := new(big.Int).Abs(newAmount)
	return absNew.Cmp(pm.AmmMaxPositionx18) <= 0 || absNew.Cmp(new(big.Int).Abs(oldAmount)) <= 0
}

type SpotBalance struct {
	ProductId  uint32
	Balancex18 *big.Int
	Lockedx18  *big.Int
}

type PerpBalance struct {
	Amountx18             *big.Int
	VQuoteBalancex18      *big.Int
	LastCumFundingRatex18 *big.Int
	ProductId             uint32
}

type SubaccountBalances struct {
	SubaccountId string
	SpotBalances map[uint32]SpotBalance
	PerpBalances map[uint32]PerpBalance
}

func (sb *SubaccountBalances) GetFirstNonZeroPerpBalance() *PerpBalance {
	for _, perpBalance := range sb.PerpBalances {
		if perpBalance.Amountx18.Sign() != 0 {
			return &perpBalance
		}
	}
	return nil
}

func (sb *SubaccountBalances) GetMaxAbsValuePerpBalance() *PerpBalance {
	var maxAbsQuotePerpBalance *PerpBalance
	for _, perpBalance := range sb.PerpBalances {
		if maxAbsQuotePerpBalance == nil {
			maxAbsQuotePerpBalance = &perpBalance
			continue
		}

		if perpBalance.VQuoteBalancex18.CmpAbs(maxAbsQuotePerpBalance.VQuoteBalancex18) > 0 {
			maxAbsQuotePerpBalance = &perpBalance
		}
	}
	return maxAbsQuotePerpBalance
}

// This function does a equality check on the balances of two subaccounts
// When the balance for a perp or spot product is not present, it is considered as zero
func (sb *SubaccountBalances) Equals(balances *SubaccountBalances) bool {
	if sb.SubaccountId != balances.SubaccountId {
		xlog.Debugf("Subaccount id mismatch: %v != %v", sb.SubaccountId, balances.SubaccountId)
		return false
	}

	// OPTIMISATION: This can be optimized if we take the union of all product ids from both maps.
	// But currently, I don't see major performance issues
	for _, productId := range contractUtils.ALL_SPOTS_IN_ORDER {
		spotBalance := sb.MustGetSpotBalance(productId)
		otherSpotBalance := balances.MustGetSpotBalance(productId)

		if !spotBalance.Equals(otherSpotBalance) {
			return false
		}
	}

	for _, productId := range contractUtils.ALL_PERPS_ON_CONTRACT {
		perpBalance := sb.MustGetPerpBalance(productId)
		otherPerpBalance := balances.MustGetPerpBalance(productId)
		if !perpBalance.Equals(otherPerpBalance) {
			return false
		}
	}

	return true
}

func (sb *SubaccountBalances) IsAMMAccount() bool {
	return strings.EqualFold(sb.SubaccountId, contractUtils.AMM_SUBACCOUNT_ID)
}

// AMM Account won't pay trading fees
// Hacky solution for now with an assumption that all markets have same taker and maker fee fraction
// Returns the fee removed from the quote balance (0 for the AMM).
func (sb *SubaccountBalances) DeductTradingFee(vQuoteBalancex18 *big.Int, isTaker bool) *big.Int {
	if sb.IsAMMAccount() {
		return big.NewInt(0)
	}
	absVquoteBalancex18 := new(big.Int).Abs(vQuoteBalancex18)

	brokerId := int(cutils.ExtractBrokerIdFromSubaccountHex(sb.SubaccountId))

	factor := cutils.GetBrokerFeeFactor(brokerId)

	feeFractionx18 := cutils.MulxCust(big.NewInt(factor), 13)
	if isTaker {
		feeFractionx18 = cutils.MulxCust(big.NewInt(factor), 13)
	}
	feex18 := cutils.Divx18(new(big.Int).Mul(absVquoteBalancex18, feeFractionx18))
	sb.UpdateSpotBalance(contractUtils.QUOTE_TOKEN_PRODUCT_ID, new(big.Int).Neg(feex18))
	return feex18
}

func (sb *SubaccountBalances) UpdateSpotBalance(productId uint32, balanceToIncreasex18 *big.Int) {
	spotBalance := sb.MustGetSpotBalance(productId)
	spotBalance.UpdateBalance(balanceToIncreasex18)
	sb.SpotBalances[productId] = spotBalance
}

// Update perp balance and realised pnl in quote balance
func (sb *SubaccountBalances) UpdatePerpBalance(productId uint32, deltaAmountx18 *big.Int, deltaVQuoteX18 *big.Int, currentFundingRatex18 *big.Int) (*big.Int, *big.Int) {
	perpBalance := sb.MustGetPerpBalance(productId)
	realisedPnlx18, fundingFeesx18 := perpBalance.UpdateBalance(deltaAmountx18, deltaVQuoteX18, currentFundingRatex18)
	sb.PerpBalances[productId] = perpBalance
	sb.UpdateSpotBalance(contractUtils.QUOTE_TOKEN_PRODUCT_ID, realisedPnlx18)
	return realisedPnlx18, fundingFeesx18
}

func (sb *SubaccountBalances) MustGetPerpBalance(productId uint32) PerpBalance {
	if productId%2 != 1 {
		panic("Invalid productId")
	}

	perpBalance, exists := sb.PerpBalances[productId]
	if !exists {
		return PerpBalance{
			Amountx18:             cutils.GetBig0(),
			VQuoteBalancex18:      cutils.GetBig0(),
			LastCumFundingRatex18: cutils.GetBig0(),
			ProductId:             productId,
		}
	}
	return perpBalance
}

func (sb *SubaccountBalances) MustGetSpotBalance(productId uint32) SpotBalance {
	if productId%2 != 0 {
		panic("Invalid productId")
	}

	spotBalance, exists := sb.SpotBalances[productId]
	if !exists {
		return SpotBalance{
			ProductId:  productId,
			Balancex18: cutils.GetBig0(),
			Lockedx18:  cutils.GetBig0(),
		}
	}
	return spotBalance
}

func (sb *SubaccountBalances) MarshalForRedis() (map[string]string, error) {
	balanceMap := make(map[string]string)

	for productId, spotBalance := range sb.SpotBalances {
		data, err := spotBalance.MarshalForRedis()
		if err != nil {
			return nil, err
		}
		balanceMap[fmt.Sprintf("%v", productId)] = string(data)
	}

	for productId, perpBalance := range sb.PerpBalances {
		data, err := perpBalance.MarshalForRedis()
		if err != nil {
			return nil, err
		}
		balanceMap[fmt.Sprintf("%v", productId)] = string(data)
	}

	return balanceMap, nil
}

func (sb *SubaccountBalances) UnmarshalForRedis(balanceMap map[string]string, subaccountId string) error {
	sb.SpotBalances = make(map[uint32]SpotBalance)
	sb.PerpBalances = make(map[uint32]PerpBalance)

	for productId, balanceData := range balanceMap {
		productIdInt, err := strconv.ParseUint(productId, 10, 32)
		if err != nil {
			return err
		}

		if productIdInt%2 == 0 {
			spotBalance := SpotBalance{
				ProductId: uint32(productIdInt),
			}
			err = spotBalance.UnmarshalForRedis([]byte(balanceData), uint32(productIdInt))
			if err != nil {
				return err
			}
			sb.SpotBalances[uint32(productIdInt)] = spotBalance
		} else {
			perpBalance := PerpBalance{}
			err = perpBalance.UnmarshalForRedis([]byte(balanceData), uint32(productIdInt))
			if err != nil {
				return err
			}
			sb.PerpBalances[uint32(productIdInt)] = perpBalance
		}
	}

	sb.SubaccountId = subaccountId
	return nil
}

func (sb *SpotBalance) Equals(other SpotBalance) bool {
	if sb.ProductId != other.ProductId {
		xlog.Debugf("Product id mismatch: %v != %v", sb.ProductId, other.ProductId)
		return false
	}

	if sb.Balancex18.Cmp(other.Balancex18) != 0 {
		xlog.Debugf("Balance mismatch: %v != %v", sb.Balancex18, other.Balancex18)
		return false
	}

	if sb.Lockedx18.Cmp(other.Lockedx18) != 0 {
		xlog.Debugf("Locked balance mismatch: %v != %v", sb.Lockedx18, other.Lockedx18)
		return false
	}

	return true
}

func (sb *SpotBalance) MarshalForRedis() ([]byte, error) {
	val := map[string]string{
		"available": sb.Balancex18.String(),
		"locked":    sb.Lockedx18.String(),
	}
	return json.Marshal(val)
}

// TODO: Handle empty data cases
func (sb *SpotBalance) UnmarshalForRedis(data []byte, productId uint32) error {
	val := make(map[string]string)
	err := json.Unmarshal(data, &val)
	if err != nil {
		return err
	}

	var ok bool
	sb.Balancex18, ok = new(big.Int).SetString(val["available"], 10)
	if !ok {
		return fmt.Errorf("Invalid available balance")
	}

	sb.Lockedx18, ok = new(big.Int).SetString(val["locked"], 10)
	if !ok {
		return fmt.Errorf("Invalid locked balance")
	}

	sb.ProductId = productId

	return nil
}

func (sb *SpotBalance) GetAssetDollarValuex18(spotPricex18 *big.Int) *big.Int {
	return new(big.Int).Mul(sb.Balancex18, spotPricex18)
}

// Increases balance
func (sb *SpotBalance) UpdateBalance(balanceToIncreasex18 *big.Int) {
	sb.Balancex18 = new(big.Int).Add(sb.Balancex18, balanceToIncreasex18)
}

func (pb *PerpBalance) Equals(other PerpBalance) bool {
	if pb.ProductId != other.ProductId {
		xlog.Debugf("Product id mismatch: %v != %v", pb.ProductId, other.ProductId)
		return false
	}

	if pb.Amountx18.Cmp(other.Amountx18) != 0 {
		xlog.Debugf("Amount mismatch: %v != %v", pb.Amountx18, other.Amountx18)
		return false
	}

	if pb.VQuoteBalancex18.Cmp(other.VQuoteBalancex18) != 0 {
		xlog.Debugf("VQuote balance mismatch: %v != %v", pb.VQuoteBalancex18, other.VQuoteBalancex18)
		return false
	}

	if pb.LastCumFundingRatex18.Cmp(other.LastCumFundingRatex18) != 0 {
		xlog.Debugf("Last cummulative funding rate mismatch: %v != %v", pb.LastCumFundingRatex18, other.LastCumFundingRatex18)
		return false
	}

	return true
}

func (pb *PerpBalance) MarshalForRedis() ([]byte, error) {
	val := map[string]string{
		"amount":          pb.Amountx18.String(),
		"vQuoteBalance":   pb.VQuoteBalancex18.String(),
		"lastFundingRate": pb.LastCumFundingRatex18.String(),
	}
	return json.Marshal(val)
}

func (pb *PerpBalance) UnmarshalForRedis(data []byte, productId uint32) error {
	val := make(map[string]string)
	err := json.Unmarshal(data, &val)
	if err != nil {
		return err
	}

	var ok bool
	pb.Amountx18, ok = new(big.Int).SetString(val["amount"], 10)
	if !ok {
		return fmt.Errorf("invalid amount")
	}

	pb.VQuoteBalancex18, ok = new(big.Int).SetString(val["vQuoteBalance"], 10)
	if !ok {
		return fmt.Errorf("invalid vQuoteBalance")
	}

	pb.LastCumFundingRatex18, ok = new(big.Int).SetString(val["lastFundingRate"], 10)
	if !ok {
		return fmt.Errorf("invalid lastFundingRate")
	}

	pb.ProductId = productId
	return nil
}

func (pb *PerpBalance) GetAvgOpenPricex18() *big.Int {
	return new(big.Int).Neg(new(big.Int).Div(cutils.Mulx18(pb.VQuoteBalancex18), pb.Amountx18))
}

func (pb *PerpBalance) GetUpnlx36(perpPricex18 *big.Int) *big.Int {
	currentNotionalValuex36 := new(big.Int).Mul(pb.Amountx18, perpPricex18)
	vQuoteBalancex36 := cutils.Mulx18(pb.VQuoteBalancex18)
	upnlx36 := new(big.Int).Add(currentNotionalValuex36, vQuoteBalancex36)
	return upnlx36
}

func (pb *PerpBalance) ReduceVQuoteBalance(vQuoteBalanceToReduceX18 *big.Int) {
	pb.VQuoteBalancex18 = new(big.Int).Sub(pb.VQuoteBalancex18, vQuoteBalanceToReduceX18)
}

// x = diff in funding rate
// y = vQuoteBalance
// Funding Fee = -x*y
// Returns realised pnl in quote balance and set LastCummFundingRate to current funding rate
func (pb *PerpBalance) RealiseFundingFee(curCummFundingRatex18 *big.Int) *big.Int {
	diffInFundingRatex18 := new(big.Int).Sub(curCummFundingRatex18, pb.LastCumFundingRatex18)
	fundingFeex36 := new(big.Int).Mul(diffInFundingRatex18, pb.VQuoteBalancex18)
	fundingFeex18 := cutils.Divx18(new(big.Int).Neg(fundingFeex36))
	pb.LastCumFundingRatex18 = curCummFundingRatex18
	return fundingFeex18
}

// Returns realised pnl and funding fees separately in quote balance x18
// When amount is increased, the abs(vQuote balance) is increased
// When amount is decreased the abs(vQuote balance) is decreased proportionally
// If amount is positive then delta vQuote balance is negative
// If amount is negative then delta vQuote balance is positive
//
// Funding fee sign convention:
// - Positive funding fee: user has to pay (money goes out from user's account)
// - Negative funding fee: user receives (money comes into user's account)
func (pb *PerpBalance) UpdateBalance(deltaAmountx18 *big.Int, deltaVQuoteX18 *big.Int, currentFundingRatex18 *big.Int) (*big.Int, *big.Int) {
	// funding fee
	fundingFeesx18 := pb.RealiseFundingFee(currentFundingRatex18)

	// realized PnL (including funding fees)
	realisedPnlx18 := new(big.Int).Neg(fundingFeesx18)

	// If amount in same direction as current position, then there won't be any pnl
	// It's fine to include zero amount case
	if pb.Amountx18.Sign()*deltaAmountx18.Sign() >= 0 {
		pb.Amountx18 = new(big.Int).Add(pb.Amountx18, deltaAmountx18)
		pb.VQuoteBalancex18 = new(big.Int).Add(pb.VQuoteBalancex18, deltaVQuoteX18)
		return realisedPnlx18, fundingFeesx18
	}

	// Handle case where position changes direction
	// We divide delta amount into two parts:
	// 1. Part that will be used to reduce vQuote balance and realise pnl
	// 2. Part that will be used to change the position
	deltaAmountPart1x18 := (cutils.MinBigInt(new(big.Int).Abs(pb.Amountx18), new(big.Int).Abs(deltaAmountx18)))
	if deltaAmountx18.Sign() < 0 {
		deltaAmountPart1x18 = new(big.Int).Neg(deltaAmountPart1x18)
	}
	// Delta vQuote Part 1  = deltaVQuote * deltaAmountPart1 / deltaAmount
	deltaVQuotePart1X18 := new(big.Int).Div(new(big.Int).Mul(deltaVQuoteX18, deltaAmountPart1x18), deltaAmountx18)
	// Delta vQuote Part 2 = deltaVQuote - deltaVQuotePart1
	// This can be zero as well
	deltaVQuotePart2X18 := new(big.Int).Sub(deltaVQuoteX18, deltaVQuotePart1X18)

	// vQuoteToRemoveFromExistingx18 =  -Current vQuoteBalance * deltaAmountPart1 / currentAmount
	vQuoteToRemoveFromExistingx18 := new(big.Int).Neg(new(big.Int).Div(new(big.Int).Mul(pb.VQuoteBalancex18, deltaAmountPart1x18), pb.Amountx18))

	// When position is long and short is used to close - deltaVQuoteX18 is positve and vQuoteToAddx18 is positive
	// When position is short and long is used to close - deltaVQuoteX18 is negative and vQuoteToAddx18 is negative
	realisedPnlx18 = new(big.Int).Add(realisedPnlx18, new(big.Int).Add(deltaVQuotePart1X18, vQuoteToRemoveFromExistingx18))
	pb.Amountx18 = new(big.Int).Add(pb.Amountx18, deltaAmountx18)

	// New vQuote balance = vQuoteBalance - vQuoteToRemoveFromExistingx18 + deltaVQuotePart2X18
	pb.VQuoteBalancex18 = new(big.Int).Add(pb.VQuoteBalancex18, new(big.Int).Sub(deltaVQuotePart2X18, vQuoteToRemoveFromExistingx18))

	// xlog.Debugf("Realised Pnl: %v | Delta Amount: %v | Delta VQuote: %v | Delta VQuote Part 1: %v | Delta VQuote Part 2: %v | VQuoteToRemoveFromExisting: %v | VQuoteToAdd: %v | Current VQuote: %v | Current Amount: %v", realisedPnlx18, deltaAmountx18, deltaVQuoteX18, deltaVQuotePart1X18, deltaVQuotePart2X18, vQuoteToRemoveFromExistingx18, deltaVQuotePart2X18, pb.VQuoteBalancex18, pb.Amountx18)
	return realisedPnlx18, fundingFeesx18
}
