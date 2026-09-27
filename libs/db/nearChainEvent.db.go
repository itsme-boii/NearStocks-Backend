package db

import (
	"time"

	"gorm.io/gorm/clause"
)

// NearChainEventTable makes the NEAR indexer's side effects exactly-once: an event is claimed by
// inserting its key (receipt id + log index) before its balance change is applied (Development.md §8.4).
type NearChainEventTable struct {
	ID          uint   `gorm:"primarykey"`
	Key         string `gorm:"uniqueIndex;size:128"`
	Name        string `gorm:"index"`
	Block       uint64 `gorm:"index"`
	Data        string `gorm:"type:text"`
	ProcessedAt *time.Time
	Error       string `gorm:"type:text;default:''"`
	CreatedAt   time.Time
}

type NearChainEventDB struct{}

// Claim returns true exactly once per key.
func (NearChainEventDB) Claim(key, name string, block uint64, data string) (bool, error) {
	res := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&NearChainEventTable{Key: key, Name: name, Block: block, Data: data})
	return res.RowsAffected == 1, res.Error
}

func (NearChainEventDB) MarkDone(key string) error {
	return db.Model(&NearChainEventTable{}).Where("key = ?", key).Update("processed_at", time.Now()).Error
}

// MarkError records a claimed event whose side effect failed; it is never retried automatically
// (the claim stays), so the alert and this row are what an operator works from.
func (NearChainEventDB) MarkError(key, msg string) error {
	return db.Model(&NearChainEventTable{}).Where("key = ?", key).Update("error", msg).Error
}
