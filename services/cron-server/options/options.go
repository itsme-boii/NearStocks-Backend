package options

import (
	"context"
	"encoding/json"
	"fmt"
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"github/eugenix-io/logx-inf-backend/xclient"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/robfig/cron/v3"
)

func StartOptionsProcessingCron() {
	xlog.Infof("Options Processing Cron job started")
	c := cron.New(cron.WithSeconds())

	// Run every second
	_, err := c.AddFunc("*/1 * * * * *", func() {

		_, err := xredis.WithRedisLock(xredis.GetOptionsLockKey(), func() (*xredis.NOOP, error) {
			xlog.Infof("Options Processing Cron running at: %v", time.Now())

			// Fetch current timestamp
			maxTimestamp := time.Now().Unix()

			// Fetch options jobs with closeTimestamp <= maxTimestamp
			jobs, err := xredis.GetRedisClient().ZRangeByScore(context.Background(), xredis.GetOptionsCloseJobsKey(), &redis.ZRangeBy{
				Min: "0",
				Max: fmt.Sprintf("%d", maxTimestamp),
			}).Result()

			if err != nil {
				xlog.Errorf("Failed to fetch options jobs from Redis: %v", err)
				return nil, err
			}

			if len(jobs) == 0 {
				xlog.Infof("No options jobs to process")
				return &xredis.NOOP{}, nil
			}

			// Process each options job and remove it from Redis
			for _, jobJSON := range jobs {
				// Deserialize the job
				var jobData contractUtils.PlaceOptionBetRedis
				if err := json.Unmarshal([]byte(jobJSON), &jobData); err != nil {
					xlog.Errorf("Failed to unmarshal options job JSON: %v", err)
					continue
				}

				// Process the job
				err := processOptionJob(jobData)
				if err != nil {
					xlog.Errorf("Failed to process options job: %v", err)
					continue
				}

				// Remove the processed job from Redis
				_, err = xredis.GetRedisClient().ZRem(context.Background(), xredis.GetOptionsCloseJobsKey(), jobJSON).Result()
				if err != nil {
					xlog.Errorf("Failed to remove options job from Redis: %v", err)
				} else {
					xlog.Infof("Successfully processed and removed options job: %v", jobData)
				}
			}

			return &xredis.NOOP{}, nil
		})

		if err != nil {
			xlog.Errorf("Failed to acquire lock or process options: %v", err)
		}
	})

	if err != nil {
		xlog.Errorf("Error adding cron function: %v", err)
		return
	}

	c.Start()
	xlog.Infof("Options Processing Cron scheduler started")

	select {}
}

func processOptionJob(closeJob contractUtils.PlaceOptionBetRedis) error {
	err := xclient.GlobalApiServerClient.CloseOptionBet(closeJob)
	if err != nil {
		xlog.Errorf("Error hitting api server for closing bet: %v", err)
		return err
	}

	xlog.Infof("Successfully processed options job: %v", closeJob)

	return nil
}
