package pool

import (
	"context"
	"encoding/json"
	"github/eugenix-io/logx-inf-backend/libs/subaccountTypes"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"sync"

	"github.com/redis/go-redis/v9"
)

type SubaccountSubpool struct {
	category    SubpoolCategory
	subaccounts map[string]subaccountTypes.SubaccountBalances
	lock        *sync.RWMutex
}

type SubaccountSubpoolHelper interface {
	GetAllSubaccounts() map[string]subaccountTypes.SubaccountBalances
	AddSubaccounts(subaccounts []subaccountTypes.SubaccountBalances)
	RemoveSubaccounts(subaccountIds []string)
	ContainsSubaccount(subaccountId string) bool
	FetchFromRedis() error
	WriteToRedis(pipe redis.Pipeliner) error
}

// Ensure that SubaccountSubpool implements SubaccountSubpoolHelper
var _ SubaccountSubpoolHelper = &SubaccountSubpool{}

func (sp *SubaccountSubpool) WriteToRedis(pipe redis.Pipeliner) error {
	sp.lock.RLock()
	defer sp.lock.RUnlock()

	ctx := context.Background()
	hashKey := xredis.GetLiquidationSubpoolKey(string(sp.category))
	// Delete previous state and then write new state
	pipe.Del(ctx, hashKey)

	for _, subaccount := range sp.subaccounts {
		data, err := subaccount.MarshalForRedis()
		if err != nil {
			return err
		}
		// TODO: Change this
		redisData, err := json.Marshal(data)
		if err != nil {
			return err
		}
		pipe.HSet(ctx, hashKey, subaccount.SubaccountId, redisData)
	}
	return nil
}

func (sp *SubaccountSubpool) FetchFromRedis() error {
	ctx := context.Background()
	hashKey := xredis.GetLiquidationSubpoolKey(string(sp.category))

	subaccounts, err := xredis.GetLiquiRedisClient().HGetAll(ctx, hashKey).Result()
	if err != nil {
		return err
	}

	sp.lock.Lock()
	defer sp.lock.Unlock()
	for subaccountId, data := range subaccounts {
		subaccount := subaccountTypes.SubaccountBalances{}
		var subaccountBal map[string]string
		err := json.Unmarshal([]byte(data), &subaccountBal)
		if err != nil {
			return err
		}

		err = subaccount.UnmarshalForRedis(subaccountBal, subaccountId)
		if err != nil {
			return err
		}

		sp.subaccounts[subaccount.SubaccountId] = subaccount
	}
	return nil
}

// Returns copy of subaccounts map
// NOTE: Although we are sending a copy of subaccounts, the subaccounts are not deep copied.
// TODO: Check this once how not deep copying affects the code
func (sp *SubaccountSubpool) GetAllSubaccounts() map[string]subaccountTypes.SubaccountBalances {
	sp.lock.RLock()
	defer sp.lock.RUnlock()

	subaccounts := make(map[string]subaccountTypes.SubaccountBalances)
	for k, v := range sp.subaccounts {
		subaccounts[k] = v
	}
	return subaccounts
}

func (sp *SubaccountSubpool) AddSubaccounts(subaccounts []subaccountTypes.SubaccountBalances) {
	sp.lock.Lock()
	defer sp.lock.Unlock()

	for _, subaccount := range subaccounts {
		sp.subaccounts[subaccount.SubaccountId] = subaccount
	}
	// xlog.Infof("SubaccountSubpool:%s.AddSubaccounts(): %d subaccounts added", sp.category, len(subaccounts))
}

func (sp *SubaccountSubpool) RemoveSubaccounts(subaccountIds []string) {
	sp.lock.Lock()
	defer sp.lock.Unlock()

	for _, subaccountId := range subaccountIds {
		delete(sp.subaccounts, subaccountId)
	}

	// xlog.Infof("SubaccountSubpool:%s.RemoveSubaccounts(): %d subaccounts removed", sp.category, len(subaccountIds))
}

func (sp *SubaccountSubpool) ContainsSubaccount(subaccountId string) bool {
	sp.lock.RLock()
	defer sp.lock.RUnlock()

	_, exists := sp.subaccounts[subaccountId]
	return exists
}
