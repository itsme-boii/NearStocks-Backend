package perp

import (
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/subaccountTypes"
)

func GetAllPerpMarkets() map[uint]subaccountTypes.PerpetualMarket {
	perpsFromDB := (&db.MarketDB{}).GetAllPerpPairs()
	perpMarketsMap := make(map[uint]subaccountTypes.PerpetualMarket, len(*perpsFromDB))
	// First create a map of perp markets
	// Then iterate over perp specs and populate the perp markets
	for _, perp := range *perpsFromDB {
		perpMarketsMap[perp.ID] = subaccountTypes.PerpetualMarket{
			ProductId:                    perp.ID,
			BaseAsset:                    perp.BaseAsset,
			MaintenanceMarginFractionx18: perp.MaintenanceMarginFractionx18.Val,
			InitialMarginFractionx18:     perp.InitialMarginFractionx18.Val,
		}
		if perp.AmmMaxPositionx18 != nil {
			m := perpMarketsMap[perp.ID]
			m.AmmMaxPositionx18 = perp.AmmMaxPositionx18.Val
			perpMarketsMap[perp.ID] = m
		}
	}

	return perpMarketsMap
}
