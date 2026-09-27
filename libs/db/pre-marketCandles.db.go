package db

import (
	"errors"

	"gorm.io/gorm"
)

type PreMarketCandleDB struct{}

// AddPreMarketCandle adds a new pre-market candle record to the database
func (*PreMarketCandleDB) AddPreMarketCandle(candleData PreMarketCandleTable) (*PreMarketCandleTable, error) {
	if err := db.Create(&candleData).Error; err != nil {
		return nil, err
	}
	return &candleData, nil
}

// GetPreMarketCandles retrieves candles for a specific product within a time range
func (*PreMarketCandleDB) GetPreMarketCandles(productId uint32, startTime, endTime int64, interval string) ([]PreMarketCandleTable, error) {
	var candles []PreMarketCandleTable

	if err := db.Where("product_id = ? AND start_time >= ? AND end_time <= ? AND interval = ?",
		productId, startTime, endTime, interval).
		Order("start_time ASC").
		Find(&candles).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // No candles found
		}
		return nil, err // Other errors
	}

	return candles, nil
}

// GetLatestCandleForInterval retrieves the most recent candle for a specific product and interval
func (*PreMarketCandleDB) GetLatestCandleForInterval(productId uint32, interval string) (*PreMarketCandleTable, error) {
	var candle PreMarketCandleTable

	if err := db.Where("product_id = ? AND interval = ?", productId, interval).
		Order("end_time DESC").
		First(&candle).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // No candle found
		}
		return nil, err
	}

	return &candle, nil
}

// CandleExists checks if a candle with the given product ID, interval, and time range already exists
func (*PreMarketCandleDB) CandleExists(productId uint32, interval string, startTime, endTime int64) (bool, error) {
	var count int64

	err := db.Model(&PreMarketCandleTable{}).
		Where("product_id = ? AND interval = ? AND start_time = ? AND end_time = ?",
			productId, interval, startTime, endTime).
		Count(&count).Error

	if err != nil {
		return false, err
	}

	return count > 0, nil
}
