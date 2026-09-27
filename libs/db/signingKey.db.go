package db

import (
	"errors"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"time"
)

type SigningKeyDB struct{}

func (*SigningKeyDB) Create(signingAddress string, subaccountId string, brokerId uint, expiryTs uint64) error {
	signingKey := SigningKeyTable{Address: signingAddress, SubaccountID: subaccountId, BrokerId: brokerId, ExpiryTs: expiryTs}
	if err := db.Create(&signingKey).Error; err != nil {
		return err
	}
	return nil
}

// NOTE: We don't return the expired signingKey
func (*SigningKeyDB) GetBySigningAddress(signingAddress string) *SigningKeyTable {
	signingKey := SigningKeyTable{Address: signingAddress}
	res := GetDbObjOrNil(db.Where(&signingKey).First(&signingKey), &signingKey)
	if res == nil || cutils.IsTimestampExpired(res.ExpiryTs) {
		return nil
	}
	return res
}

func (*SigningKeyDB) GetTotalUsers(brokerId uint) (int64, error) {
	var count int64

	if err := db.Model(&SigningKeyTable{}).Distinct("subaccount_id").Where("broker_id = ?", brokerId).Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}

// Function to get the count of new users each day for the last 30 days
func (*SigningKeyDB) GetNewUsersDaily(brokerId uint, startDate ...time.Time) ([]DailyData, error) {
	var dailyData []DailyData
	now := time.Now().UTC().Truncate(24 * time.Hour)
	var startOfPeriod time.Time

	if len(startDate) > 0 {
		startOfPeriod = startDate[0]
	} else {
		startOfPeriod = now.AddDate(0, 0, -29)
	}

	endDate := now.Add(24 * time.Hour)

	err := db.Raw(`
	SELECT 
    DATE_TRUNC('day', first_seen_at AT TIME ZONE 'UTC') AS date, 
    COUNT(*) AS count
		FROM 
			(
				SELECT 
					subaccount_id, 
					broker_id,
					MIN(created_at) AS first_seen_at
				FROM 
					signing_key_tables
				WHERE
					broker_id = ?
				GROUP BY 
					subaccount_id, broker_id
			) AS unique_users
		WHERE 
			first_seen_at BETWEEN ? AND ?
		GROUP BY 
			date
		ORDER BY 
			date DESC
    `, brokerId, startOfPeriod, endDate).Scan(&dailyData).Error

	if err != nil {
		return nil, err
	}

	return dailyData, nil
}

// Function to get the latest address for a given subAccountID
func (*SigningKeyDB) GetLatestSessionAddressBySubAccountID(subAccountID string) (string, error) {
	var signingKey SigningKeyTable

	// Query the database for the latest entry for the given subAccountID
	result := db.Where("subaccount_id = ?", subAccountID).Order("updated_at desc").First(&signingKey)
	if result.Error != nil {
		return "", result.Error
	}

	// Check if the retrieved entry is expired
	if cutils.IsTimestampExpired(signingKey.ExpiryTs) {
		return "", errors.New("the signing key is expired")
	}

	// Return the latest valid address
	return signingKey.Address, nil
}

func (*SigningKeyDB) GetCumulativeNewUsers(brokerId uint, endDate time.Time) (int64, error) {
	var cumulativeCount int64
	startDate := time.Date(2024, 9, 1, 0, 0, 0, 0, time.UTC)

	result := db.Model(&SigningKeyTable{}).
		Where("created_at BETWEEN ? AND ? AND broker_id = ?", startDate, endDate, brokerId).
		Distinct("subaccount_id").
		Count(&cumulativeCount)
	if result.Error != nil {
		xlog.Errorf("Error while fetching cumulative new users: %v", result.Error)
		return 0, result.Error
	}

	return cumulativeCount, nil
}

// ------------------------------------------------BELOW CODE IS NOT TESTED THROUGHLY----------------------------------------------------------------
// ------------------------------------------------BELOW CODE IS NOT TESTED THROUGHLY----------------------------------------------------------------
// ------------------------------------------------BELOW CODE IS NOT TESTED THROUGHLY----------------------------------------------------------------// FIXME: Logic might be incorrect here

func (*SigningKeyDB) DeleteByAddress(signingAddress string) {
	db.Where(SigningKeyTable{Address: signingAddress}).Delete(&SigningKeyTable{})
}

// ExpireByAddress expires a session key (any address case), so API requests signed with it are
// refused. The NEAR indexer calls it when a user revokes the key on-chain.
func (*SigningKeyDB) ExpireByAddress(signingAddress string) (int64, error) {
	res := db.Model(&SigningKeyTable{}).Where("lower(address) = lower(?)", signingAddress).Update("expiry_ts", 0)
	return res.RowsAffected, res.Error
}
