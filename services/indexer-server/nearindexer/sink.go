package nearindexer

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"
	"time"

	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"github/eugenix-io/logx-inf-backend/xclient"

	"github.com/redis/go-redis/v9"
)

// dbSink is the production Sink: Postgres claims and history, balance-server credits.
type dbSink struct{ brokerId uint }

func (dbSink) RevokeSessionKey(key string) error {
	n, err := (&db.SigningKeyDB{}).ExpireByAddress(key)
	if err == nil && n == 0 {
		xlog.Warnf("NEAR indexer: revoked session key %s is not in the backend", key)
	}
	return err
}

func (dbSink) Claim(e Event) (bool, error) {
	return db.NearChainEventDB{}.Claim(e.Key, e.Name, e.Block, string(e.Data))
}
func (dbSink) Done(e Event) error               { return db.NearChainEventDB{}.MarkDone(e.Key) }
func (dbSink) Failed(e Event, why string) error { return db.NearChainEventDB{}.MarkError(e.Key, why) }

func (dbSink) CreditBalance(sub string, pid uint32, amount *big.Int) error {
	ok, err := xclient.GlobalBalanceClient.UpdateTokenBalance(sub, pid, amount.String())
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("balance-server refused the update")
	}
	return nil
}

func (s dbSink) EnsureNearAccount(accountId, subHex string) error {
	if accountId == "" {
		return nil
	}
	b, err := hex.DecodeString(strings.TrimPrefix(subHex, "0x"))
	if err != nil || len(b) != 32 {
		return fmt.Errorf("bad subaccount %q", subHex)
	}
	addr20, err := cutils.NearAccountToAddr20(accountId)
	if err != nil {
		return err
	}
	broker := uint(new(big.Int).SetBytes(b[:6]).Uint64())
	n := new(big.Int).SetBytes(b[26:]).Uint64()
	_, err = (&db.NearAccountDB{}).GetOrCreate(accountId, broker, cutils.CreateSubaccountId(broker, addr20, int(n)), addr20)
	return err
}

func (dbSink) RecordTransfer(key, txHash, sub string, pid uint32, amount *big.Int, isDeposit bool) error {
	return (&db.DepositWithdrawDB{}).InsertNearTransfer(key, txHash, sub, amount.String(), pid, isDeposit, uint64(contractUtils.NearStocksChainId()))
}

// redisCheckpoint keeps the last processed height, like the appchain trackers (setLastBlockFromRedis).
type redisCheckpoint struct{ key string }

func (c redisCheckpoint) Last() (uint64, bool, error) {
	v, err := xredis.GetRedisClient().Get(context.Background(), c.key).Result()
	if errors.Is(err, redis.Nil) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	h, err := strconv.ParseUint(v, 10, 64)
	return h, err == nil, err
}

func (c redisCheckpoint) Save(h uint64) error {
	return xredis.GetRedisClient().Set(context.Background(), c.key, strconv.FormatUint(h, 10), 0).Err()
}

// Start follows the chain until ctx ends. Env: NEAR_NETWORK, NEAR_STOCKS_ACCOUNT,
// NEARDATA_URL and NEARDATA_API_KEY (optional), NEAR_INDEXER_START_BLOCK (first run only).
func Start(ctx context.Context) {
	base := os.Getenv("NEARDATA_URL")
	if base == "" {
		base = "https://mainnet.neardata.xyz"
		if contractUtils.NearTestnet() {
			base = "https://testnet.neardata.xyz"
		}
	}
	start, _ := strconv.ParseUint(os.Getenv("NEAR_INDEXER_START_BLOCK"), 10, 64)
	alert := func(m string) {
		xlog.Errorf("%s", m)
		if c := xclient.GetGlobalDiscordClient(); c != nil {
			c.SendWebhookMessage(m)
		}
	}
	apiKey := os.Getenv("NEARDATA_API_KEY")
	x := &Indexer{
		Source:      &Neardata{BaseURL: base, APIKey: apiKey},
		Checkpoint:  redisCheckpoint{key: "near-stocks:indexer:" + contractUtils.NearStocksAccount()},
		Handler:     &Handler{Sink: dbSink{}, FeeSubaccount: contractUtils.TRADING_FEES_SUBACCOUNT_ID, Alert: alert},
		Contract:    contractUtils.NearStocksAccount(),
		StartHeight: start,
	}
	if apiKey == "" {
		// no paid FastNEAR subscription: stay comfortably under the free tier's 180 req/min so
		// catch-up after any downtime doesn't burst into constant 429s (verified live — the
		// default Parallel=8 with no pacing exhausted the free limit from a cold start almost
		// immediately, and never let the indexer see a single block).
		x.MinFetchInterval = 500 * time.Millisecond // 2 block fetches/sec = 120/min, headroom for FinalHeight polls too
	}
	for ctx.Err() == nil {
		n, err := x.Step(ctx)
		if err != nil {
			xlog.Errorf("NEAR indexer: %v", err)
		}
		if n == 0 || err != nil {
			time.Sleep(time.Second) // at the tip, or retrying a failed block
		}
	}
}
