package pool

import (
	"fmt"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/subaccountTypes"
	"sync"

	"github.com/redis/go-redis/v9"
)

type SubpoolCategory string

const (
	NewSubaccountsSubpool                    SubpoolCategory = "NewSubaccounts"
	HealthySubaccountsSubpool                SubpoolCategory = "HealthySubaccounts"
	BelowInitialMarginSubaccountsSubpool     SubpoolCategory = "BelowInitialMarginSubaccounts"
	BelowMaintanenceMarginSubaccountsSubpool SubpoolCategory = "BelowMaintanenceMarginSubaccounts" // NOTE: These also have margin below initial margin but we want to maintain these separately
	NegativeBalanceSubaccountsSubpool        SubpoolCategory = "NegativeBalanceSubaccounts"        // NOTE: These also have margin below initial margin but we want to maintain these separately
)

var ALL_SUBPOOL_CATEGORIES = []SubpoolCategory{
	NewSubaccountsSubpool,
	HealthySubaccountsSubpool,
	BelowInitialMarginSubaccountsSubpool,
	BelowMaintanenceMarginSubaccountsSubpool,
	NegativeBalanceSubaccountsSubpool,
}

type SubaccountPool struct {
	pool map[SubpoolCategory]SubaccountSubpool
}

func NewSubaccountPool() SubaccountPool {
	newSubaccountPool := SubaccountPool{
		pool: make(map[SubpoolCategory]SubaccountSubpool),
	}
	for _, category := range ALL_SUBPOOL_CATEGORIES {
		newSubaccountPool.AddSubpool(category)
	}
	return newSubaccountPool
}

type SubaccountPoolHelper interface {
	AddSubpool(category SubpoolCategory)
	AddToNewSubaccountsPool(subaccounts []subaccountTypes.SubaccountBalances) error
	GetAllNewSubaccounts() map[string]subaccountTypes.SubaccountBalances
	GetAllHealthySubaccounts() map[string]subaccountTypes.SubaccountBalances
	GetAllBelowInitialMarginSubaccounts() map[string]subaccountTypes.SubaccountBalances
	GetAllNegativeBalanceSubaccounts() map[string]subaccountTypes.SubaccountBalances
	GetAllBelowMaintanenceMarginSubaccounts() map[string]subaccountTypes.SubaccountBalances
	MoveSubaccountFromHealthyTo(subaccount subaccountTypes.SubaccountBalances, destinationCategory SubpoolCategory)
	MoveSubaccountFromNewTo(subaccount subaccountTypes.SubaccountBalances, destinationCategory SubpoolCategory)
	MoveSubaccountFromBelowInitialMarginTo(subaccount subaccountTypes.SubaccountBalances, destinationCategory SubpoolCategory)
	MoveSubaccountFromNegativeEquityTo(subaccount subaccountTypes.SubaccountBalances, destinationCategory SubpoolCategory)
	WriteToRedis(pipe redis.Pipeliner) error
	FetchFromRedis() error
	GetSubpoolSize(category SubpoolCategory) int
}

// Ensure that SubaccountPool implements SubaccountPoolHelper
var _ SubaccountPoolHelper = &SubaccountPool{}

func (p *SubaccountPool) GetSubpoolSize(category SubpoolCategory) int {
	return len(p.pool[category].subaccounts)
}

func (p *SubaccountPool) AddSubpool(category SubpoolCategory) {
	p.pool[category] = SubaccountSubpool{
		category:    category,
		subaccounts: make(map[string]subaccountTypes.SubaccountBalances),
		lock:        &sync.RWMutex{},
	}
}

func (p *SubaccountPool) AddToNewSubaccountsPool(subaccounts []subaccountTypes.SubaccountBalances) error {
	pool, exists := (p.pool[NewSubaccountsSubpool])
	if !exists {
		return fmt.Errorf("err: NewSubaccountsSubpool not found in SubaccountPool")
	}
	pool.AddSubaccounts(subaccounts)
	return nil
}

func (p *SubaccountPool) GetAllNewSubaccounts() map[string]subaccountTypes.SubaccountBalances {
	pool := (p.pool[NewSubaccountsSubpool])
	return pool.GetAllSubaccounts()
}

func (p *SubaccountPool) GetAllHealthySubaccounts() map[string]subaccountTypes.SubaccountBalances {
	pool := (p.pool[HealthySubaccountsSubpool])
	return pool.GetAllSubaccounts()
}

func (p *SubaccountPool) GetAllBelowInitialMarginSubaccounts() map[string]subaccountTypes.SubaccountBalances {
	pool := (p.pool[BelowInitialMarginSubaccountsSubpool])
	return pool.GetAllSubaccounts()
}

func (p *SubaccountPool) GetAllBelowMaintanenceMarginSubaccounts() map[string]subaccountTypes.SubaccountBalances {
	pool := (p.pool[BelowMaintanenceMarginSubaccountsSubpool])
	return pool.GetAllSubaccounts()
}

func (p *SubaccountPool) GetAllNegativeBalanceSubaccounts() map[string]subaccountTypes.SubaccountBalances {
	pool := (p.pool[NegativeBalanceSubaccountsSubpool])
	return pool.GetAllSubaccounts()
}

func (p *SubaccountPool) MoveSubaccountFromHealthyTo(subaccount subaccountTypes.SubaccountBalances, destinationCategory SubpoolCategory) {
	originCategory := HealthySubaccountsSubpool
	p.moveSubaccountsFromTo([]subaccountTypes.SubaccountBalances{subaccount}, originCategory, destinationCategory)
}

func (p *SubaccountPool) MoveSubaccountFromNewTo(subaccount subaccountTypes.SubaccountBalances, destinationCategory SubpoolCategory) {
	originCategory := NewSubaccountsSubpool
	p.moveSubaccountsFromTo([]subaccountTypes.SubaccountBalances{subaccount}, originCategory, destinationCategory)
}

func (p *SubaccountPool) MoveSubaccountFromBelowInitialMarginTo(subaccount subaccountTypes.SubaccountBalances, destinationCategory SubpoolCategory) {
	originCategory := BelowInitialMarginSubaccountsSubpool
	p.moveSubaccountsFromTo([]subaccountTypes.SubaccountBalances{subaccount}, originCategory, destinationCategory)
}

func (p *SubaccountPool) MoveSubaccountFromBelowMaintenanceMarginTo(subaccount subaccountTypes.SubaccountBalances, destinationCategory SubpoolCategory) {
	originCategory := BelowMaintanenceMarginSubaccountsSubpool
	p.moveSubaccountsFromTo([]subaccountTypes.SubaccountBalances{subaccount}, originCategory, destinationCategory)
}

func (p *SubaccountPool) MoveSubaccountFromNegativeEquityTo(subaccount subaccountTypes.SubaccountBalances, destinationCategory SubpoolCategory) {
	originCategory := NegativeBalanceSubaccountsSubpool
	p.moveSubaccountsFromTo([]subaccountTypes.SubaccountBalances{subaccount}, originCategory, destinationCategory)
}

// Currently only one account is moved at a time but we can move multiple accounts at a time
func (p *SubaccountPool) moveSubaccountsFromTo(subaccounts []subaccountTypes.SubaccountBalances, originCategory SubpoolCategory, destinationCategory SubpoolCategory) {
	if originCategory == destinationCategory {
		return
	}

	originSubpool := p.pool[originCategory]
	destinationSubpool := p.pool[destinationCategory]
	destinationSubpool.AddSubaccounts(subaccounts)

	subaccountIds := cutils.MapSlice(subaccounts, func(subaccount subaccountTypes.SubaccountBalances) string { return subaccount.SubaccountId })
	originSubpool.RemoveSubaccounts(subaccountIds)
}

func (p *SubaccountPool) WriteToRedis(pipe redis.Pipeliner) error {
	for _, category := range ALL_SUBPOOL_CATEGORIES {
		subpool := p.pool[category]
		if err := subpool.WriteToRedis(pipe); err != nil {
			return err
		}
	}
	return nil
}

func (p *SubaccountPool) FetchFromRedis() error {
	for _, category := range ALL_SUBPOOL_CATEGORIES {
		subpool := p.pool[category]
		if err := subpool.FetchFromRedis(); err != nil {
			return err
		}
	}
	return nil
}
