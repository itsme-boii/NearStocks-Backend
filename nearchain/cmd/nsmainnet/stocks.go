package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"strings"

	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"github/eugenix-io/logx-inf-backend/nearchain"
)

// stockPerp mirrors nstestnet's (product ids/symbols come from contractUtils.PRODUCT_ID_SYMBOL_TO_MAP
// and are environment-independent — the same ids work on testnet and mainnet).
type stockPerp struct {
	id                    uint32
	symbol, display, name string
	// chainlinkFeedEnv, if set, means this market prices off a Chainlink feed on Ethereum mainnet
	// (read via chainlinkPrice) instead of 100exhange-oracle's HTTP /prices — the env var it names
	// holds the feed's proxy contract address. Empty means the normal oraclePrice(symbol) path.
	chainlinkFeedEnv string
}

// usMarkets: the 5 markets we're actually shipping — the final list from the mainnet oracle
// conversation, chosen because each has a real, live, price source CONFIRMED working (not just
// assumed):
//
//   - TSLA, NVDA, AAPL, GOOGL: 100exhange-oracle's Hyperliquid-backed feed (AAPL/TSLA/NVDA/GOOGL are
//     additionally covered by its primary RedStone/Canton PricePill institutional feed). Verified
//     2026-09-27 by running 100exhange-oracle locally against a scratch Redis: /prices returned
//     AAPL $340.08, matching Hyperliquid's live mid at the time.
//   - SPY: Ondo made Chainlink its official oracle, and the SPYon/USD feed is live on Ethereum
//     mainnet — read directly via chainlinkPrice, free (only cost is an Ethereum RPC call). Needs
//     SPY_CHAINLINK_FEED set to the real proxy address (see chainlink.go — never guessed or
//     defaulted, data.chain.link/docs.chain.link both blocked automated lookup when this was written).
//
// `symbol` is the PRICE SOURCE's own token key, which does NOT always match contractUtils'
// PRODUCT_ID_SYMBOL_TO_MAP naming for OSTRICH-suffixed products (e.g. product 127 is "GOOGL_OSTRICH"
// on-chain, but 100exhange-oracle just calls it "GOOGL" — it has no idea what "OSTRICH" means, that
// suffix is nsoracle's/contractUtils' own convention). Query by what the actual source calls it.
//
// Everything else from the original 14-market wishlist (QQQ, AMZN, META, COIN, MSTR, PLTR, HOOD,
// CRCL, plus AMD/MU which never had a backend product id) stays out until it has a real source too.
var usMarkets = []stockPerp{
	{103, "TSLA", "TSLA", "Tesla", ""},
	{105, "NVDA", "NVDA", "Nvidia", ""},
	{107, "AAPL", "AAPL", "Apple", ""},
	{127, "GOOGL", "GOOGL", "Alphabet", ""}, // product id 127 = "GOOGL_OSTRICH" on-chain; oracle calls it plain "GOOGL"
	{165, "SPY", "SPY", "SPDR S&P 500 ETF", "SPY_CHAINLINK_FEED"},
}

// oraclePrice hits ORACLE_SERVER_URL's /prices endpoint — identical protocol to
// xclient/oracle.xclient.go's production client and nstestnet's own helper, so it works against
// whatever real feed ORACLE_SERVER_URL points at. Point this at 100exhange-oracle's real mainnet
// deployment (its GET /prices response shape is exactly what this parses) — NOT at nsoracle, which
// is an unrelated Yahoo Finance test stand-in and explicitly not licensed for real money.
func oraclePrice(ctx context.Context, symbol string) (*big.Int, error) {
	url := os.Getenv("ORACLE_SERVER_URL")
	if url == "" {
		return nil, fmt.Errorf("ORACLE_SERVER_URL is not set — a real production price feed is required before listing stocks on mainnet")
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url+"/prices?tokens="+symbol, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var v struct {
		Data map[string]struct {
			Price struct{ Hex string } `json:"price"`
			Expo  int64                `json:"expo"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return nil, err
	}
	e, ok := v.Data[symbol]
	if !ok {
		return nil, fmt.Errorf("oracle has no price for %s", symbol)
	}
	p, ok := new(big.Int).SetString(strings.TrimPrefix(e.Price.Hex, "0x"), 16)
	if !ok {
		return nil, fmt.Errorf("oracle returned an unparseable price for %s: %q", symbol, e.Price.Hex)
	}
	// Rescale raw*10^-expo to x18 (raw*10^18). The real production oracle (100exhange-oracle) uses
	// expo=-30 always, giving a NEGATIVE shift (18-30=-12) that requires integer DIVISION, not
	// multiplication — big.Int.Exp silently returns 1 for a negative exponent (not an error), which
	// would have made this 10^12x too large and sent a wildly wrong price to upsert_perp. Caught by
	// testing this function against a real, locally-run copy of 100exhange-oracle before mainnet use.
	shift := 18 + e.Expo
	if shift >= 0 {
		return new(big.Int).Mul(p, new(big.Int).Exp(big.NewInt(10), big.NewInt(shift), nil)), nil
	}
	return new(big.Int).Quo(p, new(big.Int).Exp(big.NewInt(10), big.NewInt(-shift), nil)), nil
}

// listStocks lists usMarkets on-chain (owner upsert_perp, same margins as the crypto perps) and
// mirrors each as a backend MarketTable row + OI caps, exactly like nstestnet's flowListStocks.
// Requires ORACLE_SERVER_URL pointed at a real 100exhange-oracle (for TSLA/NVDA/AAPL/GOOGL),
// SPY_CHAINLINK_FEED set to SPY's real Chainlink proxy address (for SPY), and DATABASE_URL/
// REDIS_URL pointed at the real mainnet backend — this writes real rows into whatever database
// this process is configured against, so double check those env vars before running it for real.
func listStocks(ctx context.Context, rpc *nearchain.Client) error {
	x := func(num, den int64) string {
		return new(big.Int).Div(new(big.Int).Mul(big.NewInt(num), e18), big.NewInt(den)).String()
	}
	prices := map[string]*big.Int{}
	var listed []stockPerp
	for _, m := range usMarkets {
		var p *big.Int
		var err error
		if m.chainlinkFeedEnv != "" {
			var feed string
			if feed, err = resolveChainlinkFeed(m.chainlinkFeedEnv); err == nil {
				p, err = chainlinkPrice(ctx, feed, os.Getenv("ETH_RPC_URL"))
			}
		} else {
			p, err = oraclePrice(ctx, m.symbol)
		}
		if err != nil {
			// skip, don't abort the whole batch — e.g. SPY_CHAINLINK_FEED not set yet shouldn't
			// block listing the markets that ARE ready.
			fmt.Printf("  skipping %s: %v\n", m.display, err)
			continue
		}
		prices[m.symbol] = p
		listed = append(listed, m)
	}
	if len(listed) == 0 {
		return fmt.Errorf("no markets had a working price source, nothing to list")
	}
	var actions []nearchain.Action
	for _, m := range listed {
		actions = append(actions, callFn("upsert_perp", map[string]any{"product_id": m.id, "config": map[string]any{
			"imf_x18": x(1, 10), "mmf_x18": x(1, 20), "liq_frac_x18": x(15, 1000), "price_x18": prices[m.symbol].String(),
			"max_deviation_bps": 1000, "amm_max_position_x18": "0"}}, 20, nil))
		fmt.Printf("  %s (%s): listing at $%s\n", m.display, m.name, new(big.Int).Quo(prices[m.symbol], e18))
	}
	res, err := send(ctx, rpc, ownerAccount, coreAccount, actions...)
	if err != nil {
		return err
	}
	fmt.Printf("listed %d stocks on-chain, tx %s\n", len(listed), res.Transaction.Hash)

	db.Init()
	r := xredis.GetRedisClient()
	for _, m := range listed {
		if (&db.MarketDB{}).GetById(uint(m.id)) == nil {
			mkt := (&db.MarketDB{}).Create(&db.MarketTable{
				BaseTable: db.BaseTable{ID: uint(m.id)}, Symbol: m.display + "-USD", Type: ctypes.PERPETUAL,
				AmtToQtmConversionExpo: 9, PriceToQtmConversionExpo: 6,
				MaxPositionValuex18: ctypes.NewBigIntFromString(new(big.Int).Mul(big.NewInt(1_000_000), e18).String()),
				MinAmountx18:        ctypes.NewBigIntFromString("1000000000000000"), // 0.001 shares
				BaseAsset:           m.display, QuoteAsset: "USDC", IsActive: true,
				MakerFeeFractionx18:          ctypes.NewBigIntFromString(x(2, 10000)),
				TakerFeeFractionx18:          ctypes.NewBigIntFromString(x(5, 10000)),
				InitialMarginFractionx18:     ctypes.NewBigIntFromString(x(1, 10)),
				MaintenanceMarginFractionx18: ctypes.NewBigIntFromString(x(1, 20)),
			})
			fmt.Printf("  backend market row created: id %d %s\n", mkt.ID, mkt.Symbol)
		}
		pid := fmt.Sprint(m.id)
		if err := r.Set(ctx, "MARKET_LEVEL_LONG_OI_CAP_"+pid, "1000000", 0).Err(); err != nil {
			return err
		}
		if err := r.Set(ctx, "MARKET_LEVEL_SHORT_OI_CAP_"+pid, "1000000", 0).Err(); err != nil {
			return err
		}
		if err := r.Set(ctx, "SUBACCOUNT_MARKET_CAP_"+pid, "100000", 0).Err(); err != nil {
			return err
		}
	}
	return nil
}
