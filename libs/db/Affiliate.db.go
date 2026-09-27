package db

import (
	"github/eugenix-io/logx-inf-backend/libs/xlog"
)

type AffiliateDB struct{}

func (*AffiliateDB) CheckAffiliateExists(subaccountId string) (bool, error) {
	var count int64
	err := db.Model(&AffiliateTable{}).Where("subaccount_id = ?", subaccountId).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (*AffiliateDB) AddAffiliate(subaccountId, referralCode string) (*AffiliateTable, error) {
	affiliate := AffiliateTable{
		SubaccountId: subaccountId,
		ReferralCode: referralCode,
		IsActive:     true,
	}

	if err := db.Create(&affiliate).Error; err != nil {
		xlog.Errorf("Failed to create affiliate record for subaccount %s: %v", subaccountId, err)
		return nil, err
	}

	xlog.Infof("Successfully created affiliate record for subaccount %s with referral code %s", subaccountId, referralCode)
	return &affiliate, nil
}

func (*AffiliateDB) GetAffiliateBySubaccountId(subaccountId string) (*AffiliateTable, error) {
	var affiliate AffiliateTable
	err := db.Where("subaccount_id = ?", subaccountId).First(&affiliate).Error
	if err != nil {
		return nil, err
	}
	return &affiliate, nil
}

func (*AffiliateDB) GetAllAffiliates() ([]AffiliateTable, error) {
	var affiliates []AffiliateTable
	err := db.Find(&affiliates).Error
	if err != nil {
		return nil, err
	}
	return affiliates, nil
}

// GetReferrersByAffiliateCode gets all referrers (subaccount IDs) that used a specific affiliate code
func (*AffiliateDB) GetReferrersByAffiliateCode(affiliateCode string) ([]string, error) {
	var referrers []string

	err := db.Model(&AffiliateTable{}).
		Where("referral_code = ? AND is_active = ?", affiliateCode, true).
		Pluck("subaccount_id", &referrers).Error

	if err != nil {
		return nil, err
	}

	return referrers, nil
}
