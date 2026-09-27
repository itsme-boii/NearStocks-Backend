package nearbatch

import (
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/nearchain"
)

// Budget sizes batches so that one submit_transactions call stays inside NEAR's per-receipt limits
// with headroom (Development.md §16.3, G2): 300 Tgas, 100 log entries and 16,384 log bytes.
// Per-transaction costs are the sandbox measurements, rounded up.
type Budget struct {
	MaxTxs      int     // default 80 (the G2 choice)
	MaxTgas     float64 // default 200: leaves a third of the 300 Tgas limit as headroom
	MaxLogBytes int     // default 12,000 of 16,384
	MaxLogs     int     // default 80 of 100
}

func (b Budget) withDefaults() Budget {
	if b.MaxTxs <= 0 {
		b.MaxTxs = 80
	}
	if b.MaxTgas <= 0 {
		b.MaxTgas = 200
	}
	if b.MaxLogBytes <= 0 {
		b.MaxLogBytes = 12_000
	}
	if b.MaxLogs <= 0 {
		b.MaxLogs = 80
	}
	return b
}

// Cost is one transaction's estimated share of a batch.
type Cost struct {
	Tgas     float64
	LogBytes int
	Logs     int // JSON events of their own (withdraw_pending, circuit breakers are rare)
}

// base cost of any batch: the call itself and the `batch` event
const batchTgas, batchLogBytes = 3.0, 200

// Estimate uses the payload type byte and length. Measured (sandbox, G2): match 1.51 Tgas and
// ~139 log bytes; PERPTICK 2 Tgas; withdrawal 3.7 Tgas plus 25 Tgas prepaid to its ft_transfer and
// callback, and one ~300-byte withdraw_pending event.
func Estimate(row db.BatchTable) Cost {
	p := row.Transaction
	if len(p) == 0 {
		return Cost{Tgas: 5, LogBytes: 200}
	}
	n := len(p)
	switch p[0] {
	case nearchain.TxMatchOrders:
		return Cost{Tgas: 1.8, LogBytes: 140}
	case nearchain.TxLiquidateSubaccount:
		return Cost{Tgas: 4, LogBytes: 140}
	case nearchain.TxWithdrawCollateral, nearchain.TxWithdrawLogX:
		return Cost{Tgas: 30, LogBytes: 350, Logs: 1}
	case nearchain.TxPerpTick:
		entries := (n - 9) / 20 // each (u32, i128) is 20 bytes
		return Cost{Tgas: 1 + 0.15*float64(entries), LogBytes: 40 * entries}
	case nearchain.TxSettleUserPnl:
		subs := (n - 9) / 32
		return Cost{Tgas: 1 + 0.3*float64(subs), LogBytes: 90 * subs}
	case nearchain.TxSocialiseSubaccount:
		return Cost{Tgas: 2, LogBytes: 90}
	case nearchain.TxCloseOptionsBet:
		return Cost{Tgas: 1.2, LogBytes: 60}
	case nearchain.TxRewardRateTick:
		return Cost{Tgas: 0.8, LogBytes: 150, Logs: 1}
	default: // signed user requests: options, pools, staking, claims, nonce
		return Cost{Tgas: 2, LogBytes: 0}
	}
}

// Fit returns how many leading rows fit in one batch (at least 1, so a single oversized row is
// still tried and, if refused, parked).
func (b Budget) Fit(rows []db.BatchTable) int {
	b = b.withDefaults()
	tgas, logBytes, logs := batchTgas, batchLogBytes, 1
	for i, r := range rows {
		c := Estimate(r)
		tgas += c.Tgas
		logBytes += c.LogBytes
		logs += c.Logs
		if i > 0 && (i+1 > b.MaxTxs || tgas > b.MaxTgas || logBytes > b.MaxLogBytes || logs > b.MaxLogs) {
			return i
		}
		if i+1 == b.MaxTxs {
			return i + 1
		}
	}
	return len(rows)
}
