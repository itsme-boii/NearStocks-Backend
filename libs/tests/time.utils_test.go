package tests

import (
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"testing"
	"time"
)

func TestTimeParsing(t *testing.T) {
	startDate := "2025-07-15 00:00:00"
	startTime, err := time.Parse("2006-01-02 15:04:05", startDate)
	if err != nil {
		t.Fatalf("Error parsing start date: %v", err)
	}

	endDate := "2025-07-16 00:00:00"

	endTime, err := time.Parse("2006-01-02 15:04:05", endDate)
	if err != nil {
		t.Fatalf("Error parsing end date: %v", err)
	}

	xlog.Infof("Start Time: %s, End Time: %s", startTime, endTime)
}
