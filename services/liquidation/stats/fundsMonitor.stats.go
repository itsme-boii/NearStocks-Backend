package stats

import (
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"math/big"
	"sync"
	"time"
)

type FundsMonitorStats struct {
	FundsMapx18 map[string]map[uint32]*big.Int
	lock        sync.RWMutex
	Timestamp   int64
}

var GlobalFundsMonitorStats FundsMonitorStats

func InitFundsMonitorStats() {
	GlobalFundsMonitorStats = FundsMonitorStats{
		FundsMapx18: make(map[string]map[uint32]*big.Int),
	}
}

func parseFundsMap(data map[string]map[uint32]*big.Int) map[string]map[uint32]string {
	return cutils.ModifyMapValues(data, func(value map[uint32]*big.Int) map[uint32]string {
		res := make(map[uint32]string)
		for k, v := range value {
			res[k] = cutils.X18ToFloatStr(v)
		}
		return res
	})
}

func (fms *FundsMonitorStats) UpdateFundsMapx18(newFundsWithBalanceLockMapx18, newFundsWithoutBalanceLockMapx18, unsettledFundsMapx18 map[uint32]*big.Int) {
	fms.lock.Lock()
	defer fms.lock.Unlock()
	fms.FundsMapx18 = map[string]map[uint32]*big.Int{
		"withBalanceLock":    newFundsWithBalanceLockMapx18,
		"withoutBalanceLock": newFundsWithoutBalanceLockMapx18,
		"unsettledFunds": unsettledFundsMapx18,
	}
	fms.Timestamp = time.Now().UnixMilli()
}

func (fms *FundsMonitorStats) FundsResults() (map[string]interface{}, error) {
	fms.lock.RLock()
	defer fms.lock.RUnlock()

	return map[string]interface{}{
		"fundsMapx18": parseFundsMap(fms.FundsMapx18),
		"timestamp":   fms.Timestamp,
	}, nil
}
