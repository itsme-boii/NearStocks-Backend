package db

import (
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"time"
)

const TTL = 7 * 24 * time.Hour // 7 days

type AuthDB struct{}

func (*AuthDB) Create(key, secretHash string, subaccountId string, expiryTs uint64) error {
	authData := AuthTable{Key: key, SecretHash: secretHash, SubaccountId: subaccountId, ExpiryTs: expiryTs}
	if err := db.Create(&authData).Error; err != nil {
		return err
	}
	return nil
}

// NOTE: We don't return the expired auth
func (*AuthDB) GetByKey(key string) *AuthTable {
	authData := AuthTable{Key: key}
	ret := GetDbObjOrNil(db.Where(&authData).First(&authData), &authData)
	if ret == nil || cutils.IsTimestampExpired(ret.ExpiryTs) {
		return nil
	}
	return ret
}

// ------------------------------------------------BELOW CODE IS NOT TESTED THROUGHLY----------------------------------------------------------------
// ------------------------------------------------BELOW CODE IS NOT TESTED THROUGHLY----------------------------------------------------------------
// ------------------------------------------------BELOW CODE IS NOT TESTED THROUGHLY----------------------------------------------------------------// FIXME: Logic might be incorrect here

func (*AuthDB) DeleteById(id uint) {
	db.Delete(&AuthTable{}, id)
}
