package tests

import (
	"errors"
	"testing"

	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/services/cron-server/funding"
)

// behavior-spec H-1: the rate chosen here is both sent in PerpTick and stored in Redis.
func TestSelectFundingRate(t *testing.T) {
	calcErr := errors.New("calc failed")
	latest := &db.FundingRateTable{FundingRate: -42}
	cases := []struct {
		name      string
		calc      int64
		calcErr   error
		latest    *db.FundingRateTable
		latestErr error
		want      int64
	}{
		{"fresh rate wins", 7, nil, latest, nil, 7},
		{"fresh zero rate is kept", 0, nil, latest, nil, 0},
		{"fallback to latest stored", 7, calcErr, latest, nil, -42},
		{"no history -> 0", 7, calcErr, nil, nil, 0},
		{"db error -> 0", 7, calcErr, latest, errors.New("db down"), 0},
	}
	for _, c := range cases {
		if got := funding.SelectFundingRate(c.calc, c.calcErr, c.latest, c.latestErr); got != c.want {
			t.Errorf("%s: got %d want %d", c.name, got, c.want)
		}
	}
}
