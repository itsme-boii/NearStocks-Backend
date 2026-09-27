package appstate

import (
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/funding"
	"github/eugenix-io/logx-inf-backend/libs/metric"
	"github/eugenix-io/logx-inf-backend/libs/perp"
	"github/eugenix-io/logx-inf-backend/libs/subaccountTypes"
	"github/eugenix-io/logx-inf-backend/libs/xcache"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"os"

	"github/eugenix-io/logx-inf-backend/xclient"
	"math/big"
	"time"
)

var debugCache bool

type AppStateCacheConfig struct {
	oraclePriceExpiration time.Duration
	perpMarketExpiration  time.Duration
	fundingRateExpiration time.Duration
}

func GetAppConfig(oraclePriceExp, fundingExp, perpetualExp time.Duration) AppStateCacheConfig {
	return AppStateCacheConfig{
		oraclePriceExpiration: oraclePriceExp,
		perpMarketExpiration:  perpetualExp,
		fundingRateExpiration: fundingExp,
	}
}

func parseAppStateCacheConfig(cacheConfig ...AppStateCacheConfig) AppStateCacheConfig {
	if len(cacheConfig) > 0 {
		return cacheConfig[0]
	}
	return AppStateCacheConfig{}
}

type AppState interface {
	GetAppState(_cacheConfig ...AppStateCacheConfig) (map[string]ctypes.OraclePrice, map[uint]subaccountTypes.PerpetualMarket, map[string]*big.Int, error)
	GetAllPerpMarkets(expiryDuration ...time.Duration) map[uint]subaccountTypes.PerpetualMarket
	GetAllFundingRates(expiryDuration ...time.Duration) map[string]*big.Int
	GetAllOraclePrices(expiryDuration ...time.Duration) (map[string]ctypes.OraclePrice, error)
}

type AppStateImp struct {
	oraclePricesCache             xcache.Cache[map[string]ctypes.OraclePrice]
	perpetualMarketsCache         xcache.Cache[map[uint]subaccountTypes.PerpetualMarket]
	cumulativeFundingRateMapCache xcache.Cache[map[string]*big.Int]
}

var _ AppState = &AppStateImp{}

func NewAppState() AppState {
	debugCache = os.Getenv("DEBUG_CACHE") == "1"
	return &AppStateImp{
		oraclePricesCache:             xcache.NewCache[map[string]ctypes.OraclePrice](nil, time.Time{}),
		perpetualMarketsCache:         xcache.NewCache[map[uint]subaccountTypes.PerpetualMarket](nil, time.Time{}),
		cumulativeFundingRateMapCache: xcache.NewCache[map[string]*big.Int](nil, time.Time{}),
	}
}

func (as *AppStateImp) GetAppState(_cacheConfig ...AppStateCacheConfig) (map[string]ctypes.OraclePrice, map[uint]subaccountTypes.PerpetualMarket, map[string]*big.Int, error) {
	defer cutils.LogTime(time.Now(), metric.GET_APP_STATE)

	cacheConfig := parseAppStateCacheConfig(_cacheConfig...)

	oraclePrices, err := as.GetAllOraclePrices(cacheConfig.oraclePriceExpiration)
	if err != nil {
		return nil, nil, nil, err
	}

	perpetualMarkets := as.GetAllPerpMarkets(cacheConfig.perpMarketExpiration)

	// Get all the funding rates
	cumulativeFundingRateMap := as.GetAllFundingRates(cacheConfig.fundingRateExpiration)

	return oraclePrices, perpetualMarkets, cumulativeFundingRateMap, nil
}

// Writes cache with 1 Minute expiry by default
func (as *AppStateImp) GetAllPerpMarkets(expiryDuration ...time.Duration) map[uint]subaccountTypes.PerpetualMarket {
	if len(expiryDuration) > 0 && expiryDuration[0] != 0 {
		cacheData := as.perpetualMarketsCache.Get(expiryDuration[0])
		if cacheData != nil {
			if debugCache {
				xlog.Infof("Returning all perp market from cache")
			}
			return *cacheData
		}
		if debugCache {
			xlog.Infof("Perp market not found in cache")
		}
	}
	newData := perp.GetAllPerpMarkets()
	as.perpetualMarketsCache.Set(newData, 1*time.Minute)
	return newData
}

func (as *AppStateImp) GetAllFundingRates(expiryDuration ...time.Duration) map[string]*big.Int {
	if len(expiryDuration) > 0 && expiryDuration[0] != 0 {
		cacheData := as.cumulativeFundingRateMapCache.Get(expiryDuration[0])
		if cacheData != nil {
			if debugCache {
				xlog.Infof("Returning Funding rates from cache")
			}
			return *cacheData
		}
		if debugCache {
			xlog.Infof("Funding rates not found in cache")
		}
	}
	newData := funding.GetAllCumulativeFundingRates()
	as.cumulativeFundingRateMapCache.Set(newData, 1*time.Minute)
	return newData
}

func (as *AppStateImp) GetAllOraclePrices(expiryDuration ...time.Duration) (map[string]ctypes.OraclePrice, error) {
	defer cutils.LogTime(time.Now(), metric.GET_ALL_ORACLE_PRICES)

	if len(expiryDuration) > 0 && expiryDuration[0] != 0 {
		cacheData := as.oraclePricesCache.Get(expiryDuration[0])
		if cacheData != nil {
			if debugCache {
				xlog.Infof("Returning oracle prices from cache")
			}
			return *cacheData, nil
		}
		if debugCache {
			xlog.Infof("Oracle prices not found in cache")
		}
	}

	oracleClientPrices, err := xclient.GlobalOracleClient.GetAllPrices()
	if err != nil {
		xlog.Errorf("Looks like oracle server is down: %v", err)
		return nil, err
	}

	oraclePrices := make(map[string]ctypes.OraclePrice, len(oracleClientPrices))
	for symbol, price := range oracleClientPrices {
		oraclePrices[symbol] = ctypes.OraclePrice{
			Pricex18: cutils.ConvertXCustToX18(price.PriceX, *price.Expo),
			Symbol:   symbol,
		}
	}

	as.oraclePricesCache.Set(oraclePrices, 5*time.Second)
	return oraclePrices, nil
}
