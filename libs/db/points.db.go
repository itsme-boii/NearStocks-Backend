package db

type PointsDB struct{}

// InsertPoints inserts a new points record for a user
func (*PointsDB) InsertPoints(pointsData PointsTable) (*PointsTable, error) {
	if err := db.Create(&pointsData).Error; err != nil {
		return nil, err
	}
	return &pointsData, nil
}

// GetPointsByAddress retrieves all points records for a specific address
func (*PointsDB) GetPointsByAddress(address string) ([]PointsTable, error) {
	var points []PointsTable

	if err := db.Where("address = ?", address).Order("week DESC").Find(&points).Error; err != nil {
		return nil, err
	}

	return points, nil
}
