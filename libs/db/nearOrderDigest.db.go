package db

import (
	"time"

	"gorm.io/gorm/clause"
)

// NearOrderDigestTable lists every order digest sent to near-stocks.near, so the prune job can
// remove the contract's replay-protection entry once the order has expired (Development.md §5.11).
type NearOrderDigestTable struct {
	ID           uint       `gorm:"primarykey"`
	Digest       string     `gorm:"uniqueIndex;size:66"`
	ExpirationMs uint64     `gorm:"index"`
	PrunedAt     *time.Time `gorm:"index"`
	CreatedAt    time.Time
}

type NearOrderDigestDB struct{}

// Record stores digests (the same order can fill many times; duplicates are ignored).
func (NearOrderDigestDB) Record(digests []string, expirationsMs []uint64) error {
	if len(digests) == 0 {
		return nil
	}
	rows := make([]NearOrderDigestTable, len(digests))
	for i := range digests {
		rows[i] = NearOrderDigestTable{Digest: digests[i], ExpirationMs: expirationsMs[i]}
	}
	return db.Clauses(clause.OnConflict{DoNothing: true}).Create(&rows).Error
}

// ExpiredUnpruned returns digests of orders that expired before beforeMs and are still on-chain.
func (NearOrderDigestDB) ExpiredUnpruned(beforeMs uint64, limit int) ([]string, error) {
	var out []string
	err := db.Model(&NearOrderDigestTable{}).Where("pruned_at IS NULL AND expiration_ms < ?", beforeMs).
		Order("expiration_ms ASC").Limit(limit).Pluck("digest", &out).Error
	return out, err
}

func (NearOrderDigestDB) MarkPruned(digests []string) error {
	return db.Model(&NearOrderDigestTable{}).Where("digest IN ?", digests).Update("pruned_at", time.Now()).Error
}
