package engine

import (
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/services/engine/conditional"
)

type Engine struct {
	orderbooks        map[uint]*Orderbook
	triggerOrderbooks map[uint]*conditional.TriggerOrderbook
}

var GlobalEngine *Engine

func Init() {
	// Initialize global engine
	xlog.Infof("Initializing engine")
	markets := (&db.MarketDB{}).GetAllActiveMarkets()
	obs := map[uint]*Orderbook{}
	tobs := map[uint]*conditional.TriggerOrderbook{}
	for _, market := range *markets {
		obs[market.ID] = NewOrderbook(&market)
		tobs[market.ID] = conditional.NewTriggerOrderbook(&market)
	}
	GlobalEngine = &Engine{orderbooks: obs, triggerOrderbooks: tobs}
	xlog.Infof("Engine initialized")
}

func (e *Engine) GetOrderbook(marketId uint) *Orderbook {
	orderbook, ok := GlobalEngine.orderbooks[marketId]
	if !ok {
		xlog.Infof("Orderbook not found for marketId: %d\n", marketId)
		return nil
	}
	return orderbook
}

func (e *Engine) GetTriggerOrderbook(marketId uint) *conditional.TriggerOrderbook {
	triggerOrderbook, ok := GlobalEngine.triggerOrderbooks[marketId]
	if !ok {
		xlog.Infof("TriggerOrderbook not found for marketId: %d\n", marketId)
		return nil
	}
	return triggerOrderbook
}

func (e *Engine) GetTriggerOrderbooks() map[uint]*conditional.TriggerOrderbook {
	return e.triggerOrderbooks
}
