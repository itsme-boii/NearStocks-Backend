package mocks

import (
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/xclient"
	"math/big"
	"sync"
)

type MockOracleClientImpl struct {
	functionCallCounter map[string]int
	lock                sync.Mutex
}

func GetMockOraclePrices() map[string]ctypes.OraclePrice {
	// 4 OCT 2024
	return map[string]ctypes.OraclePrice{
		"USDC": {
			Pricex18: cutils.FloatStrToX18("1"),
			Symbol: "USDC",
		},
		"USDT": {
			Pricex18:  cutils.FloatStrToX18("0.99975"),
			Symbol: "USDT",
		},
		"ETH": {
			Pricex18:  cutils.FloatStrToX18("2382.57"),
			Symbol: "ETH",
		},
		"BTC": {
			Pricex18:  cutils.FloatStrToX18("61528.02"),
			Symbol: "BTC",
		},
		"PEPE": {
			Pricex18:  cutils.FloatStrToX18("0.0000091377"),
			Symbol: "PEPE",
		},
		"DOGE": {
			Pricex18:  cutils.FloatStrToX18("0.108"),
			Symbol: "DOGE",
		},
		"SOL": {
			Pricex18:  cutils.FloatStrToX18("140.7"),
			Symbol: "SOL",
		},
		"ARB": {
			Pricex18:  cutils.FloatStrToX18("0.55315"),
			Symbol: "ARB",
		},
		"NEAR": {
			Pricex18:  cutils.FloatStrToX18("4.752"),
			Symbol: "NEAR",
		},
		"XRP": {
			Pricex18:  cutils.FloatStrToX18("0.5252"),
			Symbol: "XRP",
		},
		"LINK": {
			Pricex18:  cutils.FloatStrToX18("10.9765"),
			Symbol: "LINK",
		},
		"EIGEN": {
			Pricex18:  cutils.FloatStrToX18("0.3497"),
			Symbol: "EIGEN",
		},
		"TRUMP": {
			Pricex18:  cutils.FloatStrToX18("0.495"),
			Symbol: "TRUMP",
		},
		"BIDEN": {
			Pricex18:  cutils.FloatStrToX18("0"),
			Symbol: "BIDEN",
		},
		"HARRIS": {
			Pricex18:  cutils.FloatStrToX18("0.497"),
			Symbol: "HARRIS",
		},
	}
}

// Validate MockOracleClientImpl implements xclient.OracleClient
var _ xclient.OracleClient = &MockOracleClientImpl{}

func NewMockOracleClient() xclient.OracleClient {
	return &MockOracleClientImpl{
		functionCallCounter: make(map[string]int),
	}
}

func (oc *MockOracleClientImpl) GetBaseUrl() string {
	return "mockOracleUrl"
}

func (oc *MockOracleClientImpl) GetFunctionCallCount(functionName string) int {
	oc.lock.Lock()
	defer oc.lock.Unlock()
	return oc.functionCallCounter[functionName]
}

func (oc *MockOracleClientImpl) GetAllPrices() (map[string]xclient.TokenPrice, error) {
	oc.lock.Lock()
	defer oc.lock.Unlock()
	oc.functionCallCounter["GetAllPrices"]++

	e := int64(2)
	return map[string]xclient.TokenPrice{
		"token1": {
			PriceX: big.NewInt(100),
			Expo:   &e,
		},
		"token2": {
			PriceX: big.NewInt(200),
			Expo:   &e,
		},
	}, nil
}

func (oc *MockOracleClientImpl) GetAllCollateralTokenPrices() (map[string]xclient.TokenPrice, error) {
	oc.lock.Lock()
	defer oc.lock.Unlock()
	oc.functionCallCounter["GetAllCollateralTokenPrices"]++
	return nil, nil
}
