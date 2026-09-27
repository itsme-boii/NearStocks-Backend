package db

import (
	"time"
)

type FundingRateDB struct{}
type FundingRateResponse struct {
	Timestamp   time.Time `json:"timestamp"`
	FundingRate float64   `json:"funding_rate"`
}

// Insert inserts a new record into the FundingRateTable
func (f *FundingRateDB) Insert(marketId uint, fundingRate int64, fundingTimestamp int64) (*FundingRateTable, error) {
	newEntry := &FundingRateTable{
		MarketId:         marketId,
		FundingRate:      fundingRate,
		FundingTimestamp: fundingTimestamp,
	}
	result := db.Create(newEntry)
	if result.Error != nil {
		return nil, result.Error
	}
	return newEntry, nil
}

// GetLatestEntriesByMarketId fetches the latest 'X' entries for a given market ID
func (f *FundingRateDB) GetLatestEntriesByMarketId(marketId uint, lookbackDistance int) ([]FundingRateTable, error) {
	var entries []FundingRateTable
	result := db.Where("market_id = ?", marketId).Order("created_at DESC").Limit(lookbackDistance).Find(&entries)
	if result.Error != nil {
		return nil, result.Error
	}
	return entries, nil
}

// GetLatestEntryByMarketId fetches the latest entry for a given market ID
func (f *FundingRateDB) GetLatestEntryByMarketId(marketId uint) (*FundingRateTable, error) {
	var entry FundingRateTable
	result := db.Where("market_id = ?", marketId).Order("created_at DESC").First(&entry)
	if result.Error != nil {
		return nil, result.Error
	}
	return &entry, nil
}

func (f *FundingRateDB) GetFundingRateForPastDayHourly(marketId int64) ([]FundingRateResponse, error) {
	var results []FundingRateResponse
	pastDay := time.Now().Add(-24 * time.Hour)

	err := db.Model(&FundingRateTable{}).
		Select("DATE_TRUNC('hour', created_at) as timestamp, AVG(funding_rate::float)*3600/1e16 as funding_rate").
		Where("market_id = ? AND created_at > ?", marketId, pastDay).
		Group("DATE_TRUNC('hour', created_at)").
		Order("timestamp ASC").
		Scan(&results).Error

	if err != nil {
		return nil, err
	}
	return results, nil
}

func (f *FundingRateDB) GetFundingRateForPastYearDaily(marketId int64) ([]FundingRateResponse, error) {
	var results []FundingRateResponse
	pastYear := time.Now().Add(-30 * 24 * time.Hour)

	err := db.Model(&FundingRateTable{}).
		Select("DATE_TRUNC('day', created_at) as timestamp, AVG(funding_rate::float)*365*86400/1e16 as funding_rate").
		Where("market_id = ? AND created_at > ?", marketId, pastYear).
		Group("DATE_TRUNC('day', created_at)").
		Order("timestamp ASC").
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	return results, nil
}
