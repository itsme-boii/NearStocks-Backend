package tests

// NEAR builders (contract/endpoint.near.go): each existing EndpointContract entry point, with
// NEAR_SETTLEMENT=1, must queue exactly the Borsh payload the Rust contract expects. The expected
// bytes are built with nearchain's encoders, which vectors/payloads.json pins against Rust.

import (
	"bytes"
	"errors"
	"math/big"
	"strings"
	"testing"

	"github/eugenix-io/logx-inf-backend/contract"
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/nearchain"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var recorded []string

func nearCapture(t *testing.T) (*[]contract.QueuedTx, *contract.EndpointContract) {
	t.Setenv("NEAR_SETTLEMENT", "1")
	var got []contract.QueuedTx
	restore := contract.SetNearQueueForTest(func(q contract.QueuedTx) error { got = append(got, q); return nil })
	t.Cleanup(restore)
	t.Cleanup(contract.SetNearDigestRecorderForTest(func(d []string, e []uint64) error { recorded = append(recorded, d...); return nil }))
	return &got, contract.NewEndpointContract()
}

func b(s string) *big.Int {
	v, _ := new(big.Int).SetString(s, 10)
	return v
}

func subs(t *testing.T, account string) (string, [32]byte) {
	addr, err := cutils.NearAccountToAddr20(account)
	require.NoError(t, err)
	id := cutils.CreateSubaccountId(2, addr, 0)
	bytes32, err := cutils.SubaccountIdToBytes32(id)
	require.NoError(t, err)
	return "0x" + common.Bytes2Hex(bytes32[:]), bytes32
}

var sig65 = "0x" + common.Bytes2Hex(bytes.Repeat([]byte{7}, 65))

const skey = "0x4599B3307aFd037958b1f583CC5F44F62dc3E970"

func TestNearMatchOrdersSigns(t *testing.T) {
	recorded = nil
	got, epc := nearCapture(t)
	tHex, tSub := subs(t, "alice.near")
	mHex, mSub := subs(t, "bob.near")
	price := b("65000000000000000000000")
	// taker SELLS 1.5 into a maker bid: taker amount negative, maker positive, matched = maker sign (+)
	err := epc.MatchOrders(contract.MatchOrderRequest{
		ProductId: 3, TakerSubaccountId: tHex, MakerSubaccountId: mHex, TakerPriceX18: big.NewInt(0), MakerPriceX18: price,
		TakerExpiryTs: 11, MakerExpiryTs: 12, TakerMaxAmountX18: b("1500000000000000000"), MakerMaxAmountX18: b("2000000000000000000"),
		TakerSide: ctypes.ORDER_SIDE_SELL, TakerSignature: sig65, MakerSignature: sig65, TakerSessionKey: skey, MakerSessionKey: skey,
		MatchedAmountx18: b("1500000000000000000"),
	}, 1)
	require.NoError(t, err)
	want := nearchain.MatchOrders{
		ProductId:     3,
		Taker:         nearchain.Order{Subaccount: tSub, PriceX18: big.NewInt(0), Amount: b("-1500000000000000000"), Expiration: 11, SessionKey: common.HexToAddress(skey), ProductId: 3},
		Maker:         nearchain.Order{Subaccount: mSub, PriceX18: price, Amount: b("2000000000000000000"), Expiration: 12, SessionKey: common.HexToAddress(skey), ProductId: 3},
		MatchedAmount: b("1500000000000000000"),
	}.Encode()
	require.Len(t, *got, 1)
	q := (*got)[0]
	assert.Equal(t, want, q.Payload)
	assert.Equal(t, 65, len(q.Sig1))
	assert.Equal(t, 65, len(q.Sig2))
	assert.Equal(t, "MatchOrders", q.FunctionName)
	// both order digests are kept for the prune job, and they are the digests the contract uses
	require.Len(t, recorded, 2)
	td, _ := contractUtils.NearOrderDigest("near-stocks.near", 397, tSub, big.NewInt(0), b("-1500000000000000000"), 11, false, skey, 3)
	assert.Equal(t, "0x"+common.Bytes2Hex(td), recorded[0])

	// taker BUYS: maker amount and matched amount become negative
	*got = nil
	err = epc.MatchOrders(contract.MatchOrderRequest{
		ProductId: 3, TakerSubaccountId: tHex, MakerSubaccountId: mHex, TakerPriceX18: price, MakerPriceX18: price,
		TakerMaxAmountX18: b("1"), MakerMaxAmountX18: b("5"), TakerSide: ctypes.ORDER_SIDE_BUY, TakerSignature: sig65, MakerSignature: sig65,
		TakerSessionKey: skey, MakerSessionKey: skey, MatchedAmountx18: b("1"),
	}, 2)
	require.NoError(t, err)
	want = nearchain.MatchOrders{
		ProductId:     3,
		Taker:         nearchain.Order{Subaccount: tSub, PriceX18: price, Amount: b("1"), SessionKey: common.HexToAddress(skey), ProductId: 3},
		Maker:         nearchain.Order{Subaccount: mSub, PriceX18: price, Amount: b("-5"), SessionKey: common.HexToAddress(skey), ProductId: 3},
		MatchedAmount: b("-1"),
	}.Encode()
	assert.Equal(t, want, (*got)[0].Payload)
}

func TestNearLiquidationUsesLiquidateeSignAndSortedPrices(t *testing.T) {
	got, epc := nearCapture(t)
	vHex, vSub := subs(t, "victim.near")
	lHex, lSub := subs(t, "liq.near")
	err := epc.LiquidateSubaccount(contract.LiquidationRequest{
		ProductId: 3, LiquidatorSubaccountId: lHex, LiquidatorPriceX18: b("60000"), LiquidatorAmount: b("9"), LiquidatorExpiryTs: 99,
		LiquidatorSessionKey: skey, LiquidatorSide: ctypes.ORDER_SIDE_BUY, LiquidateeSubaccountId: vHex, LiquidateeAmount: b("4"),
		LiquidateeSide: ctypes.ORDER_SIDE_SELL, LiquidatorSignature: sig65, MatchedAmountx18: b("3"),
		PerpOraclePricesX18: map[uint32]*big.Int{3: b("60000"), 1: b("2500"), 5: nil},
		SpotOraclePricesX18: map[uint32]*big.Int{4: b("1")},
	}, 7)
	require.NoError(t, err)
	want := nearchain.Liquidate{
		ProductId:  3,
		Prices:     []nearchain.PidValue{{ProductId: 1, Value: b("2500")}, {ProductId: 3, Value: b("60000")}, {ProductId: 4, Value: b("1")}},
		Liquidator: nearchain.Order{Subaccount: lSub, PriceX18: b("60000"), Amount: b("9"), Expiration: 99, SessionKey: common.HexToAddress(skey), ProductId: 3},
		Liquidatee: vSub,
		Amount:     b("-3"), // matched size, liquidatee sells
	}.Encode()
	assert.Equal(t, want, (*got)[0].Payload)
	assert.Equal(t, vHex, (*got)[0].Sub1)
}

func TestNearWithdrawRoutesLogXToType13(t *testing.T) {
	got, epc := nearCapture(t)
	aHex, aSub := subs(t, "alice.near")
	w := contractUtils.NearWithdraw{SubaccountId: aHex, SessionKey: skey, ProductId: 4, Amount: "25500000000000000000", Nonce: 3, Receiver: "alice.near"}
	require.NoError(t, epc.NearWithdraw(w, sig65, 1))
	w.ProductId = 0
	require.NoError(t, epc.NearWithdraw(w, sig65, 2))
	assert.Equal(t, nearchain.TxWithdrawCollateral, (*got)[0].Payload[0])
	assert.Equal(t, nearchain.TxWithdrawLogX, (*got)[1].Payload[0])
	want := nearchain.NearWithdraw{Subaccount: aSub, SessionKey: common.HexToAddress(skey), ProductId: 4, Amount: b("25500000000000000000"), Nonce: big.NewInt(3), Receiver: "alice.near"}
	assert.Equal(t, want.Encode(), (*got)[0].Payload)
	assert.Error(t, epc.NearWithdraw(w, "0x1234", 3), "short signature refused")
	w.Amount = "0"
	assert.Error(t, epc.NearWithdraw(w, sig65, 3), "zero amount refused")
}

func TestNearFundingAndPriceTicks(t *testing.T) {
	got, epc := nearCapture(t)
	require.NoError(t, epc.FundingTick([]uint{3, 1}, []*big.Int{b("-5"), b("7")}, 1760000000, 1))
	want := nearchain.PerpTick{Time: 1760000000, Rates: []nearchain.PidValue{{ProductId: 3, Value: b("-5")}, {ProductId: 1, Value: b("7")}}}.Encode()
	assert.Equal(t, want, (*got)[0].Payload)
	assert.Error(t, epc.FundingTick([]uint{3}, []*big.Int{b("1"), b("2")}, 1, 2), "ids and rates must line up")
	require.NoError(t, epc.PriceTick(map[uint32]*big.Int{4: b("1"), 3: b("65000")}, 1760000001, 3))
	want = nearchain.PerpTick{Time: 1760000001, Prices: []nearchain.PidValue{{ProductId: 3, Value: b("65000")}, {ProductId: 4, Value: b("1")}}}.Encode()
	assert.Equal(t, want, (*got)[1].Payload)
	assert.Equal(t, []byte{}, (*got)[1].Sig1, "sequencer-only rows carry an empty signature")
}

func TestNearProductsAndClaims(t *testing.T) {
	got, epc := nearCapture(t)
	aHex, aSub := subs(t, "alice.near")
	key := common.HexToAddress(skey)

	ok, err := epc.PlaceOptionBet(contractUtils.UserOptionBet{SubAccountId: aHex, ProductId: 3, Amount: "-2", Interval: 5, Nonce: 9, SessionKey: skey}, b("-2"), b("65000"), 424242, 180, 5, sig65, 1)
	require.True(t, ok)
	require.NoError(t, err)
	want := nearchain.PlaceOption{Bet: nearchain.OptionBet{Subaccount: aSub, ProductId: 3, Amount: b("-2"), Interval: 5, Nonce: big.NewInt(9), SessionKey: key}, OrderId: 424242, EntryPriceX18: b("65000"), PayoutPct: 180, FeePct: 5}.Encode()
	assert.Equal(t, want, (*got)[0].Payload)

	_, err = epc.CloseOptionBet(424242, b("64999"), 2, aHex)
	require.NoError(t, err)
	assert.Equal(t, nearchain.CloseOption{OrderId: 424242, ExitPriceX18: b("64999")}.Encode(), (*got)[1].Payload)

	order := contractUtils.PlacePreMarketOrderRequest{SubAccountId: aHex, ProductId: 1001, Amount: "100", IsBuy: true, Nonce: 10, SessionKey: skey}
	_, err = epc.PreMarketOrderRequest(order, b("50"), b("1"), sig65, 3)
	require.NoError(t, err)
	_, err = epc.SyntheticSpotOrderRequest(order, b("50"), b("1"), sig65, 4)
	require.NoError(t, err)
	pool := nearchain.PoolTrade{Order: nearchain.PoolOrder{Subaccount: aSub, ProductId: 1001, Amount: b("100"), IsBuy: true, Nonce: big.NewInt(10), SessionKey: key}, QuoteDelta: b("50"), Fees: b("1")}
	assert.Equal(t, pool.EncodePreMarket(), (*got)[2].Payload)
	assert.Equal(t, pool.EncodeSyntheticSpot(), (*got)[3].Payload)

	staker := "0x00000000000000000000000000000000000000aa"
	_, err = epc.StakeLogx(contractUtils.StakeLogXRequest{SubAccountId: aHex, ProductId: 0, TokenAmount: "400", StakerContract: staker, SessionKey: skey, Nonce: 11}, b("400"), b("0"), sig65, 5)
	require.NoError(t, err)
	st := nearchain.StakeRequest{Subaccount: aSub, ProductId: 0, Amount: b("400"), StakerContract: common.HexToAddress(staker), SessionKey: key, Nonce: big.NewInt(11)}
	assert.Equal(t, st.EncodeStake(), (*got)[4].Payload)

	// ClaimRewards carries the claimable total (D-6), not the transient earnings
	_, err = epc.ClaimRewards(contractUtils.ClaimRewards{SubAccountId: aHex, SessionKey: skey, StakerContract: staker, ProductId: 0, Nonce: 13}, b("1"), b("42"), sig65, 6)
	require.NoError(t, err)
	cr := nearchain.ClaimRewards{Subaccount: aSub, SessionKey: key, StakerContract: common.HexToAddress(staker), ProductId: 0, Nonce: big.NewInt(13), AmountX18: b("42")}
	assert.Equal(t, cr.Encode(), (*got)[5].Payload)

	_, err = epc.ClaimLogX(contractUtils.DepositRequest{SubAccountId: aHex, SessionKey: skey, Nonce: 14}, b("1000"), sig65, 7)
	require.NoError(t, err)
	assert.Equal(t, nearchain.ClaimLogX{Subaccount: aSub, TokenAmount: b("1000"), SessionKey: key, Nonce: big.NewInt(14)}.Encode(), (*got)[6].Payload)
	// both claims are paid from the rewards pool, so their rows name it (reconciled, paused on refusal)
	rewardsPool := strings.ToLower(contractUtils.LOGX_REWARDS_SUBACCOUNT_ID)
	assert.Equal(t, rewardsPool, (*got)[5].Sub2)
	assert.Equal(t, rewardsPool, (*got)[6].Sub2)
}

func TestNearRefusesAppchainOnlyFlows(t *testing.T) {
	got, epc := nearCapture(t)
	_, err := epc.ClaimCampaignRewards(contractUtils.CampaignRewardClaimRequest{}, b("1"), sig65, 1)
	assert.True(t, errors.Is(err, contract.ErrNotOnNear))
	_, err, _ = epc.WithdrawCollateral(contractUtils.WithdrawCollateral{}, b("1"), sig65, 1)
	assert.True(t, errors.Is(err, contract.ErrNotOnNear))
	_, err, _ = epc.WithdrawLogX(contractUtils.WithdrawLogX{}, b("1"), sig65, 1)
	assert.True(t, errors.Is(err, contract.ErrNotOnNear))
	assert.True(t, errors.Is(epc.RegisterSessionKey(contractUtils.SessionRequest{}, 1), contract.ErrNotOnNear))
	assert.True(t, errors.Is(epc.FinalisePendingDeposits(contract.FinaliseDepositRequest{}, 1), contract.ErrNotOnNear))
	epc.PerpTick([]*big.Int{b("1")}, 1, 1) // positional ticks are refused (H-2)
	assert.Empty(t, *got, "nothing was queued")
}

func TestNearDomainSwitch(t *testing.T) {
	t.Setenv("NEAR_SETTLEMENT", "1")
	t.Setenv("NEAR_NETWORK", "testnet")
	aHex, _ := subs(t, "alice.near")
	// a request signed under the near-stocks testnet domain verifies through the existing verifier
	msg := contractUtils.SettleUserPnl{SubAccountId: aHex, SessionKey: skey, Nonce: 1, ChainId: contractUtils.NEAR_STOCKS_TESTNET_CHAIN_ID}
	err := contractUtils.VerifySettleUserPnl(msg)
	assert.Error(t, err, "a missing signature still fails")
	assert.Equal(t, "near-stocks.testnet", contractUtils.NearStocksAccount())
	assert.Equal(t, int64(398), contractUtils.NearStocksChainId())
}

// With NEAR settlement the API verifies orders under the same domain and chain id the contract
// hashes them with, so an order that passes the API cannot be refused on-chain for its signature.
func TestNearOrderVerificationMatchesContractDigest(t *testing.T) {
	t.Setenv("NEAR_SETTLEMENT", "1")
	sk, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(sk.PublicKey).Hex()
	_, aSub := subs(t, "alice.near")
	addr20, _ := cutils.NearAccountToAddr20("alice.near")
	aId := cutils.CreateSubaccountId(2, addr20, 0) // the "broker_addr_n" form the order controller passes
	price, amount := b("65000000000000000000000"), b("-1500000000000000000")
	d, err := contractUtils.NearOrderDigest(contractUtils.NearStocksAccount(), contractUtils.NearStocksChainId(), aSub, price, amount, 1790000000000, false, addr, 3)
	require.NoError(t, err)
	sig, err := crypto.Sign(d, sk)
	require.NoError(t, err)
	sig[64] += 27
	err = contractUtils.VerifyOrderSignature(contractUtils.OrderSigRequest{
		SubAccountId: aId, PriceX18: price, Amount: amount, Expiration: 1790000000000, SessionKey: addr,
		ChainId: contractUtils.SessionKeyChainId(), Signature: "0x" + common.Bytes2Hex(sig), ProductId: big.NewInt(3),
	})
	require.NoError(t, err, "the API accepts exactly the digest the contract checks")
	assert.Equal(t, int64(397), contractUtils.SessionKeyChainId())
	// options, pre-market, synthetic, staking and claim requests are signed for near-stocks too
	assert.Equal(t, int64(397), contractUtils.EndpointChainId())
}
