package cutils

import (
	"context"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"runtime/debug"
	"sync"
	"time"
)

// Runs job infinetely with a time interval
// First job starts immediately after function call and then after every interval defined in the parameter
// This also takes an external wait group to wait for the job to finish
func RunInfiniteJobWithTicker(ctx context.Context, wg *sync.WaitGroup, job func() error, interval time.Duration, jobId string) {
	wg.Add(1)
	xlog.Infof("Starting infinite job - %v with interval - %v", jobId, interval)
	go func() {
		defer wg.Done()

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			func() {
				defer func() {
					if r := recover(); r != nil {
						// Add a discord alert here if needed
						xlog.Errorf("Recovered from panic in job - %v: %v \n Stack Trace", jobId, r, string(debug.Stack()))
					}
				}()

				xlog.Infof("Running job - %v", jobId)
				err := job()
				if err != nil {
					xlog.Errorf("Error in job - %v: %v", jobId, err)
				}
			}()

			select {
			case <-ctx.Done():
				xlog.Infof("Exiting job - %v", jobId)
				return
			case <-ticker.C:
				continue
			}
		}
	}()
}
