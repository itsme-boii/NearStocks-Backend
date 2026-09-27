package stats

import (
	"context"
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"sync"

	"github.com/redis/go-redis/v9"
)

type LiquidationStats struct {
	SubpoolSizes map[string]int
	IsRunning    bool
	lock         sync.RWMutex
}

var GlobalLiquidationStats LiquidationStats

func Init() {
	GlobalLiquidationStats = LiquidationStats{
		SubpoolSizes: make(map[string]int),
		IsRunning:    false,
	}
}

func (ls *LiquidationStats) Results() (map[string]interface{}, error) {
	// Get total liquidation orders stats
	var defaultStats = map[string]string{}
	for _, field := range ctypes.ALL_LIQUIDATION_STATS_FIELDS {
		defaultStats[string(field)] = "0"
	}

	res, err := xredis.GetLiquiRedisClient().HGetAll(context.Background(), xredis.GetLiquidationStatsKey()).Result()
	if err == redis.Nil {
		res = map[string]string{}
	} else if err != nil {
		xlog.Errorf("Error getting liquidation stats from redis: %v", err)
		return nil, err
	}

	res = cutils.MergeMaps(defaultStats, res)

	return map[string]interface{}{
		"subpool":    ls.SubpoolSizes,
		"isRunning":  ls.IsRunning,
		"redisStats": res,
	}, nil
}

func (ls *LiquidationStats) UpdateSubpoolSizes(newSubpoolSizes map[string]int) {
	ls.lock.Lock()
	defer ls.lock.Unlock()
	ls.SubpoolSizes = newSubpoolSizes
}

func (ls *LiquidationStats) UpdateRunningStatus(isRunning bool) {
	ls.lock.Lock()
	defer ls.lock.Unlock()
	ls.IsRunning = isRunning
}
