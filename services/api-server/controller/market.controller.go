// TODO: Move some logic into service file
package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"github/eugenix-io/logx-inf-backend/services/api-server/client"
	"github/eugenix-io/logx-inf-backend/services/api-server/services"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/redis/go-redis/v9"

	"github.com/gin-gonic/gin"
)

var ActiveSymbols []string
var SYMBOL_TO_PRODUCT_ID_MAP map[string]uint32

type MarketController struct {
	stakingService *services.StakingService
}

func RegisterMarketController(
	r *gin.RouterGroup,
) {
	marketController := MarketController{
		stakingService: services.NewStakingService(),
	}

	//Populate active symbols
	var err error
	ActiveSymbols, err = marketController.stakingService.PopulateActiveSymbols()
	if err != nil {
		xlog.Errorf("Error initialising active market symbols in market controller %v", err)
	}

	SYMBOL_TO_PRODUCT_ID_MAP = make(map[string]uint32)
	for productId, symbol := range contractUtils.PRODUCT_ID_SYMBOL_TO_MAP {
		SYMBOL_TO_PRODUCT_ID_MAP[symbol] = productId
	}

	rg := r.Group("/market")

	// Public Endpoints
	rg.GET("/:id", marketController.GetMarketById)
	rg.GET("", marketController.GetAllMarkets)
	rg.GET("/fundingrate/:symbol", marketController.GetFundingRate)
	rg.GET("/fundingrates", marketController.GetAllFundingRates)
	rg.GET("/externalFundingRate", marketController.GetAllCombinedFundingRates) // for external funding rate for comaprsion

	// Private Admin only Endpoint
	// FIXME: Add auth middleware
	rg.POST("", marketController.CreateMarket)

	// FIXME: Add auth middleware
	// Private Internal Endpoints
	rg.GET("/:id/impact-prices", marketController.GetImpactPrices)

}

type GetAllMarketsRequest struct {
	Type     ctypes.MarketType `form:"type" binding:"omitempty,oneof=SPOT PERPETUAL"` // SPOT, PERPETUAL
	BrokerId *uint             `form:"brokerId" binding:"omitempty"`
}

func (*MarketController) GetAllMarkets(ctx *gin.Context) {
	var requestObj GetAllMarketsRequest
	if err := ctx.BindQuery(&requestObj); err != nil {
		cutils.ApiAbort(ctx, http.StatusBadRequest, "Invalid request parameters")
		return
	}

	markets := (&db.MarketDB{}).GetAllMarkets(gin.H{"type": requestObj.Type})

	if requestObj.BrokerId == nil || *requestObj.BrokerId != 2 {
		filteredMarkets := []db.MarketTable{}
		for _, market := range *markets {
			if !strings.HasSuffix(market.BaseAsset, "_OSTRICH") {
				filteredMarkets = append(filteredMarkets, market)
			}
		}
		markets = &filteredMarkets
	}

	cutils.ApiResponse(ctx, gin.H{"markets": markets}, 200, "")
}

type GetMarketByIdRequest struct {
	Id uint `uri:"id" binding:"required"`
}

func (*MarketController) GetMarketById(ctx *gin.Context) {
	var requestObj GetMarketByIdRequest
	if err := ctx.BindUri(&requestObj); err != nil {
		cutils.ApiAbort(ctx, http.StatusBadRequest, "Invalid request parameters")
		return
	}

	market := (&db.MarketDB{}).GetById(requestObj.Id)
	if market == nil {
		cutils.ApiAbort(ctx, http.StatusNotFound, "Market not found")
		return
	}

	cutils.ApiSuccess(ctx, gin.H{"market": market}, "")
}

// TODO: Not super imp: Add validation logic for type
type CreateMarketBody struct {
	ID                            uint              `json:"id" binding:"required"`
	Symbol                        string            `binding:"required"`                         // "ETH-USD"
	Type                          ctypes.MarketType `binding:"required,oneof=SPOT PERPETUAL"`    // SPOT, PERPETUAL
	PriceToQuantumConversionExpo  int32             `binding:"required"`                         // Exponent to convert price to quantum
	AmountToQuantumConversionExpo int32             `binding:"required"`                         // Exponent to convert amount to quantum
	MaxPositionValueStr           string            `json:"maxPositionValue" binding:"required"` // Maximum position value in quote token in human readible format
	MinAmountStr                  string            `json:"minAmount" binding:"required"`        // Max minimum amount of base token in hum readible format
	BaseAsset                     string            `binding:"required"`                         // Eg: ETH
	QuoteAsset                    string            `binding:"required"`                         // Eg: USDC
	IsActive                      bool
	MakerFeeFractionStr           string `json:"makerFeeFraction" binding:"required"`          // Fraction of trade value that should be taken from makers
	TakerFeeFractionStr           string `json:"takerFeeFraction" binding:"required"`          // Fraction of trade value that should be taken from takers
	InitialMarginFractionStr      string `json:"initialMarginFraction" binding:"required"`     // Initial margin fraction required to open a position
	MaintenanceMarginFractionStr  string `json:"maintenanceMarginFraction" binding:"required"` // Maintenance margin fraction required to keep a position open

	// Update
	Replace bool `json:"replace"`
}

func (cmb *CreateMarketBody) Validate() error {
	if cutils.QuantumPrecisionCheck(cmb.MinAmountStr, cmb.AmountToQuantumConversionExpo) != nil {
		return fmt.Errorf("minAmount is not precise enough. There is some mismatch with the amountToQuantumConversionExpo")
	}

	if cutils.FloatStrZeroCheck(cmb.MakerFeeFractionStr) || cutils.FloatStrZeroCheck(cmb.TakerFeeFractionStr) || cutils.FloatStrZeroCheck(cmb.InitialMarginFractionStr) || cutils.FloatStrZeroCheck(cmb.MaintenanceMarginFractionStr) {
		return fmt.Errorf("makerFeeFraction, takerFeeFraction, initialMarginFraction, maintenanceMarginFraction cannot be zero")
	}
	return nil
}

func (*MarketController) CreateMarket(ctx *gin.Context) {
	var requestObj CreateMarketBody
	if err := ctx.BindJSON(&requestObj); err != nil {
		xlog.Infof("Market Controller Error: %v\n", err)
		cutils.ApiAbort(ctx, http.StatusBadRequest, "Invalid request parameters")
		return
	}

	if err := requestObj.Validate(); err != nil {
		cutils.ApiAbort(ctx, http.StatusBadRequest, err.Error())
		return
	}

	market := db.MarketTable{
		BaseTable:                    db.BaseTable{ID: requestObj.ID},
		Symbol:                       requestObj.Symbol,
		Type:                         requestObj.Type,
		PriceToQtmConversionExpo:     requestObj.PriceToQuantumConversionExpo,
		AmtToQtmConversionExpo:       requestObj.AmountToQuantumConversionExpo,
		MaxPositionValuex18:          cutils.FloatStrToBigIntX18(requestObj.MaxPositionValueStr),
		MinAmountx18:                 cutils.FloatStrToBigIntX18(requestObj.MinAmountStr),
		BaseAsset:                    requestObj.BaseAsset,
		QuoteAsset:                   requestObj.QuoteAsset,
		IsActive:                     requestObj.IsActive,
		MakerFeeFractionx18:          cutils.FloatStrToBigIntX18(requestObj.MakerFeeFractionStr),
		TakerFeeFractionx18:          cutils.FloatStrToBigIntX18(requestObj.TakerFeeFractionStr),
		InitialMarginFractionx18:     cutils.FloatStrToBigIntX18(requestObj.InitialMarginFractionStr),
		MaintenanceMarginFractionx18: cutils.FloatStrToBigIntX18(requestObj.MaintenanceMarginFractionStr),
	}

	if requestObj.Replace {
		if (&db.MarketDB{}).Update(&market) == nil {
			cutils.ApiAbort(ctx, http.StatusBadRequest, fmt.Sprintf("Market with type: %v and symbol: %v cannot be updated", requestObj.Type, requestObj.Symbol))
			return
		}
	} else {
		if (&db.MarketDB{}).Create(&market) == nil {
			cutils.ApiAbort(ctx, http.StatusBadRequest, fmt.Sprintf("Market with type: %v and symbol: %v already exists. Please add replace key with true value if you really want to update the market", requestObj.Type, requestObj.Symbol))
			return
		}
	}

	cutils.ApiResponse(ctx, gin.H{"market": market}, 200, "")
}

type GetImpactPricesRequest struct {
	Id uint `uri:"id" binding:"required"`
}

func (*MarketController) GetImpactPrices(ctx *gin.Context) {
	var requestObj GetImpactPricesRequest
	if err := ctx.BindUri(&requestObj); err != nil {
		cutils.ApiAbort(ctx, http.StatusBadRequest, "Invalid request parameters")
		return
	}

	impactPrices := client.GlobalEngineClient.GetImpactPrices(requestObj.Id)
	if impactPrices == nil {
		cutils.ApiAbort(ctx, http.StatusNotFound, "Market not found")
		return
	}

	cutils.ApiSuccess(ctx, gin.H{"impact_prices": impactPrices}, "")
}

type GetFundingRateRequest struct {
	Symbol string `uri:"symbol" binding:"required"`
}

func (*MarketController) GetFundingRate(ctx *gin.Context) {
	var requestObj GetFundingRateRequest
	var fundingRate int64
	if err := ctx.BindUri(&requestObj); err != nil {
		xlog.Infof("Market Controller Error: %v\n", err)
		cutils.ApiAbort(ctx, http.StatusBadRequest, "Invalid request parameters")
		return
	}

	fundingRate, err := getFundingRateForSymbol(requestObj.Symbol)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, fmt.Sprintf("Error fetching funding rate for symbol %s", err))
		return
	}
	cutils.ApiSuccessWithoutMessage(ctx, gin.H{"fundingRateX18": fundingRate}, http.StatusOK)
}

type GetAllFundingRatesRequest struct {
	BrokerId *uint `form:"brokerId" binding:"omitempty"`
}

func (*MarketController) GetAllFundingRates(ctx *gin.Context) {
	var requestObj GetAllFundingRatesRequest
	if err := ctx.BindQuery(&requestObj); err != nil {
		cutils.ApiAbort(ctx, http.StatusBadRequest, "Invalid request parameters")
		return
	}

	var rates []ctypes.SymbolFundingRate
	for _, symbol := range ActiveSymbols {
		if requestObj.BrokerId == nil || *requestObj.BrokerId != 2 {
			baseAsset := strings.Split(symbol, "-")[0]
			if strings.HasSuffix(baseAsset, "_OSTRICH") {
				continue
			}
		}

		fundingRate, err := getFundingRateForSymbol(symbol)
		if err != nil {
			xlog.Errorf("Error fetching funding rate for symbol %s: %v", symbol, err)
			continue
		}

		cumulativeFundingRate, err := xredis.GetCumulativeFundingRateForSymbol(xredis.GetRedisClient(), symbol)
		if err != nil {
			xlog.Errorf("Error fetching cumulative funding rate for symbol %s: %v", symbol, err)
			continue
		}

		rates = append(rates, ctypes.SymbolFundingRate{
			Symbol: symbol,
			Funding: ctypes.FundingRates{
				FundingRate:           fundingRate,
				CumulativeFundingRate: cumulativeFundingRate.String(),
			},
		})
	}
	cutils.ApiSuccessWithoutMessage(ctx, gin.H{"rates": rates}, http.StatusOK)
}

func getFundingRateForSymbol(symbol string) (int64, error) {
	fundingRateKey := xredis.GetFundingRateKey(symbol)

	fundingRateStr, err := xredis.GetRedisClient().Get(context.Background(), fundingRateKey).Result()
	if err != nil {
		if err == redis.Nil {
			return 0, fmt.Errorf("funding rate not found for symbol %s", symbol)
		}
		return 0, fmt.Errorf("error fetching funding for symbol %s: %v", symbol, err)
	}

	if fundingRateStr == "" {
		return 0, fmt.Errorf("funding rate not found for symbol %s", symbol)
	}

	var fundingRateData xredis.FundingRateData
	err = json.Unmarshal([]byte(fundingRateStr), &fundingRateData)
	if err != nil {
		return 0, fmt.Errorf("error unmarshalling funding rate JSON for symbol %s: %v", symbol, err)
	}

	fundingRate, err := strconv.ParseInt(fundingRateData.FundingRate, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("error converting funding rate string to int for symbol %s: %v", symbol, err)
	}

	return fundingRate, nil
}

func (*MarketController) GetAllCombinedFundingRates(ctx *gin.Context) {
	// Fetch external funding rates from Redis
	redisKey := xredis.GetExternalFundingRates()
	data, err := xredis.GetRedisClient().Get(context.Background(), redisKey).Result()
	if err != nil {
		xlog.Errorf("Error fetching external funding rates from Redis: %v", err)
		cutils.ApiAbort(ctx, http.StatusBadRequest, "Error fetching external funding")
		return
	}

	var externalFundingData map[string]map[string]string
	if err := json.Unmarshal([]byte(data), &externalFundingData); err != nil {
		xlog.Errorf("Error unmarshalling external funding rates data: %v", err)
		cutils.ApiAbort(ctx, http.StatusBadRequest, "Error fetching external funding")
		return
	}

	// Prepare the final response map
	finalResponse := make(map[uint32]map[string]string)

	// Iterate over ActiveSymbols
	for _, symbol := range ActiveSymbols {
		// Extract the base symbol (e.g., "BOME" from "BOME-USD")
		baseSymbol := strings.Split(symbol, "-")[0]

		// Prepare funding rates map with default values
		fundingRates := map[string]string{
			"binance":     "0",
			"bybit":       "0",
			"bitget":      "0",
			"hyperliquid": "0",
			"logx":        "0",
		}

		// Get 'logx' funding rate
		if fundingRate, err := getFundingRateForSymbol(symbol); err == nil {
			adjustedRate := (float64(fundingRate) / 1e16) * 3600
			fundingRates["logx"] = strconv.FormatFloat(math.Round(adjustedRate*1e8)/1e8, 'f', 8, 64)
		} else {
			continue
		}

		// Add external funding rates
		if externalRates, found := externalFundingData[symbol]; found {
			for exchange, rate := range externalRates {
				fundingRates[strings.ToLower(exchange)] = rate
			}
		}

		// Check if all external funding rates are zero
		allZeros := true
		for exchange, rate := range fundingRates {
			if exchange != "logx" && rate != "0" {
				allZeros = false
				break
			}
		}

		if allZeros {
			continue
		}

		// Lookup productId using the base symbol and add to final response
		if productId, exists := SYMBOL_TO_PRODUCT_ID_MAP[baseSymbol]; exists {
			finalResponse[productId] = fundingRates
		} else {
			xlog.Warnf("Product ID not found for base symbol: %s", baseSymbol)
		}
	}

	// Return the final response
	cutils.ApiSuccessWithoutMessage(ctx, finalResponse, http.StatusOK)
}
