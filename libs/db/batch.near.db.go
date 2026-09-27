package db

import (
	"time"

	"gorm.io/gorm"
)

// NEAR batcher queries (services/cron-server/nearbatch). Rows keep their transaction_counter order.

const (
	NearStatePending  = ""
	NearStateInflight = "inflight"
	NearStateFailed   = "failed"
)

// NearPending returns queued rows not yet sent, oldest first, skipping paused subaccounts.
func (*BatchDB) NearPending(limit int, exclude []string) ([]BatchTable, error) {
	q := db.Where("near_state = ?", NearStatePending)
	if len(exclude) > 0 {
		q = q.Where("sub_account_id1 NOT IN ? AND sub_account_id2 NOT IN ?", exclude, exclude)
	}
	var rows []BatchTable
	err := q.Order("transaction_counter ASC").Limit(limit).Find(&rows).Error
	return rows, err
}

// NearInflight returns the rows of the batch that was being sent, in contract-index order.
func (*BatchDB) NearInflight() ([]BatchTable, error) {
	var rows []BatchTable
	err := db.Where("near_state = ?", NearStateInflight).Order("n_submission_idx ASC").Find(&rows).Error
	return rows, err
}

// NearMarkInflight records the contract indexes a batch will use, before it is sent.
func (*BatchDB) NearMarkInflight(ids []uint, startIdx uint64) error {
	return db.Transaction(func(tx *gorm.DB) error {
		for i, id := range ids {
			if err := tx.Model(&BatchTable{}).Where("id = ?", id).
				Updates(map[string]any{"near_state": NearStateInflight, "n_submission_idx": startIdx + uint64(i)}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (*BatchDB) NearSetTxHash(ids []uint, hash string) error {
	return db.Model(&BatchTable{}).Where("id IN ?", ids).Update("near_tx_hash", hash).Error
}

// NearMarkLanded soft-deletes rows the contract executed (the same end state as before).
func (*BatchDB) NearMarkLanded(ids []uint) error {
	return db.Model(&BatchTable{}).Where("id IN ?", ids).Updates(map[string]any{"near_state": "", "deleted_at": time.Now()}).Error
}

// NearClearInflight returns rows to the queue (the batch did not land).
func (*BatchDB) NearClearInflight(ids []uint) error {
	return db.Model(&BatchTable{}).Where("id IN ?", ids).Updates(map[string]any{"near_state": NearStatePending, "n_submission_idx": 0, "near_tx_hash": ""}).Error
}

// NearMarkFailed parks a row the contract refused on its own.
func (*BatchDB) NearMarkFailed(id uint, reason string) error {
	return db.Model(&BatchTable{}).Where("id = ?", id).Updates(map[string]any{"near_state": NearStateFailed, "near_failure": reason}).Error
}

// NearRecentlyLanded returns the subaccounts of rows that landed since `since`.
func (*BatchDB) NearRecentlyLanded(since time.Time) ([]string, error) {
	var rows []BatchTable
	err := db.Unscoped().Select("sub_account_id1", "sub_account_id2").
		Where("deleted_at >= ?", since).Find(&rows).Error
	seen := map[string]bool{}
	var out []string
	for _, r := range rows {
		for _, s := range []string{r.SubAccountID1, r.SubAccountID2} {
			if s != "" && !seen[s] {
				seen[s] = true
				out = append(out, s)
			}
		}
	}
	return out, err
}

// NearHasQueued reports which of subs still have rows queued, in flight or parked: for those the
// backend is ahead of (or diverged from) the chain on purpose, so they are not compared.
func (*BatchDB) NearHasQueued(subs []string) (map[string]bool, error) {
	var rows []BatchTable
	err := db.Select("sub_account_id1", "sub_account_id2").
		Where("sub_account_id1 IN ? OR sub_account_id2 IN ?", subs, subs).Find(&rows).Error
	out := map[string]bool{}
	for _, r := range rows {
		out[r.SubAccountID1], out[r.SubAccountID2] = true, true
	}
	return out, err
}

// NearReconcileDiffTable records every difference the reconciler found.
type NearReconcileDiffTable struct {
	ID         uint   `gorm:"primarykey"`
	Subaccount string `gorm:"index"`
	Detail     string `gorm:"type:text"`
	CreatedAt  time.Time
}

func (*BatchDB) NearRecordDiff(sub, detail string) error {
	return db.Create(&NearReconcileDiffTable{Subaccount: sub, Detail: detail}).Error
}

// NearHasUnsettled reports whether sub has rows still to be sent or in flight (parked rows excluded:
// they will never land). nearresync requires none, so the chain is final for the subaccount.
func (*BatchDB) NearHasUnsettled(sub string) (bool, error) {
	var n int64
	err := db.Model(&BatchTable{}).Where("(sub_account_id1 = ? OR sub_account_id2 = ?) AND near_state IN ?", sub, sub,
		[]string{NearStatePending, NearStateInflight}).Count(&n).Error
	return n > 0, err
}
