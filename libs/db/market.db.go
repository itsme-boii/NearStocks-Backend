package db

import (
	"fmt"
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
)

type MarketDB struct{}

const ASSET_USDC = "USDC"

// TODO: Create spot market
// TODO: Add some basic checks
func (*MarketDB) Create(newMarket *MarketTable) *MarketTable {
	tx := db.Create(newMarket)
	if tx.Error != nil {
		fmt.Println("Error creating market: ", tx.Error)
		return nil
	}

	return newMarket
}

func (*MarketDB) Update(market *MarketTable) *MarketTable {
	tx := db.Save(market)
	if tx.Error != nil {
		fmt.Println("Error updating market: ", tx.Error)
		return nil
	}

	return market
}

func (*MarketDB) DeleteById(id uint) {
	db.Delete(&MarketTable{}, id)
}

func (*MarketDB) GetAllMarkets(filter map[string]interface{}) *[]MarketTable {
	markets := []MarketTable{}
	query := db.Select("*")
	if marketType, ok := filter["type"]; ok && marketType != ctypes.MarketType("") {
		query = query.Where("type = ?", marketType)
	}
	if isActive, ok := filter["is_active"]; ok {
		query = query.Where("is_active = ?", isActive)
	}

	return GetDBObjOrEmptyList(query.Find(&markets), &markets)
}

func (*MarketDB) GetById(marketId uint) *MarketTable {
	market := MarketTable{}
	return GetDbObjOrNil(db.First(&market, marketId), &market)
}

// ----------------- NOT TESTED -----------------
// ----------------- NOT TESTED -----------------
// ----------------- NOT TESTED -----------------
func (*MarketDB) GetAllActiveMarkets() *[]MarketTable {
	return (&MarketDB{}).GetAllMarkets(map[string]interface{}{"is_active": true})
}

func (*MarketDB) GetAllActivePerpMarkets() []MarketTable {
	var activePerpMarkets []MarketTable
	query := db.Table("market_tables").
		Where("is_active = ? AND type = ?", true, ctypes.PERPETUAL).
		Order("id ASC").
		Find(&activePerpMarkets)
	if query.Error != nil {
		fmt.Println("Error retrieving active perpetual markets: ", query.Error)
		return nil
	}
	return activePerpMarkets
}

func (*MarketDB) GetAllPerpPairs() *[]MarketTable {
	return (&MarketDB{}).GetAllMarkets(map[string]interface{}{"type": ctypes.PERPETUAL})
}

// GetMarketIDBySymbolAndType retrieves the ID of a market given its symbol and type
func (*MarketDB) GetMarketIDBySymbolAndType(symbol string, marketType ctypes.MarketType) *uint {
	var market MarketTable
	tx := db.Where("symbol = ? AND type = ?", symbol, marketType).First(&market)
	if tx.Error != nil {
		fmt.Println("Error finding market: ", tx.Error)
		return nil
	}
	return &market.ID
}

func (m *MarketDB) GetMaintenanceMarginFraction(symbol string, marketType ctypes.MarketType) (*ctypes.BigInt, error) {
	var market MarketTable
	tx := db.Where("symbol = ? AND type = ?", symbol, marketType).First(&market)
	if tx.Error != nil {
		return nil, fmt.Errorf("error finding market: %v", tx.Error)
	}
	return &market.MaintenanceMarginFractionx18, nil
}

func (m *MarketDB) GetInitialMarginFractionByID(marketID uint32) (*ctypes.BigInt, error) {
	var market MarketTable
	tx := db.Where("id = ?", marketID).First(&market)
	if tx.Error != nil {
		return nil, fmt.Errorf("error finding market: %v", tx.Error)
	}
	return &market.InitialMarginFractionx18, nil
}

func (m *MarketDB) GetMaintenanceMarginFractionByID(marketID uint32) (*ctypes.BigInt, error) {
	var market MarketTable
	tx := db.Where("id = ?", marketID).First(&market)
	if tx.Error != nil {
		return nil, fmt.Errorf("error finding market: %v", tx.Error)
	}
	return &market.MaintenanceMarginFractionx18, nil
}

func (*MarketDB) GetAllActiveMarketIDs() *[]uint {
	var marketIDs []uint
	return GetDBObjOrEmptyList(db.Table("market_tables").Where("is_active = ?", true).Pluck("id", &marketIDs), &marketIDs)
}

// Function to map ids to unit_amt_quantums and unit_price_quantums
func (*MarketDB) GetIDToQtmConversionExpoMaps() (map[uint]int, map[uint]int, error) {
	// Define the maps to hold the results
	unitAmtQuantumsMap := make(map[uint]int)
	unitPriceQuantumsMap := make(map[uint]int)

	// Query to get all the records from market_tables
	var markets []MarketTable
	tx := db.Find(&markets)
	if tx.Error != nil {
		return nil, nil, fmt.Errorf("error retrieving markets: %v", tx.Error)
	}

	// Populate the maps with id to quantum mappings
	for _, market := range markets {
		unitAmtQuantumsMap[market.ID] = int(market.AmtToQtmConversionExpo)
		unitPriceQuantumsMap[market.ID] = int(market.PriceToQtmConversionExpo)
	}

	return unitAmtQuantumsMap, unitPriceQuantumsMap, nil
}

func (*MarketDB) GetAllActivePerpMarketIDs() []uint {
	var activePerpMarketIDs []uint
	query := db.Table("market_tables").
		Where("type = ? AND is_active = ?", ctypes.PERPETUAL, true).
		Pluck("id", &activePerpMarketIDs)
	if query.Error != nil {
		fmt.Println("Error retrieving active perpetual market IDs: ", query.Error)
		return nil
	}
	return activePerpMarketIDs
}
