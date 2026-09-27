// nsoracle is a TEST-ONLY stand-in for the oracle server (xclient/oracle.xclient.go) when the
// production oracle is unreachable: it serves /allprices and /prices?tokens=... in the same format.
// Crypto mids come from Hyperliquid, refreshed every 2 seconds. US stocks/ETFs listed in
// stockTickers come from Yahoo Finance, refreshed every 30 seconds (extend that map to add more).
// Stablecoins are 1. Any other symbol (forex, commodities, the OSTRICH set) is served at a fixed
// placeholder of 1 and logged: don't trade those markets against this stand-in.
//
//	go run ./nearchain/cmd/nsoracle -addr :8099      (then ORACLE_SERVER_URL=http://localhost:8099)
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
)

type entry struct {
	Price struct {
		Type string `json:"type"`
		Hex  string `json:"hex"`
	} `json:"price"`
	Expo        int64 `json:"expo"`
	PublishTime int64 `json:"publishTime"`
}

// stockTickers are the US stocks/ETFs this stand-in prices for real, via Yahoo Finance: internal
// symbol (as in contractUtils.PRODUCT_ID_SYMBOL_TO_MAP) -> the Yahoo ticker to fetch. They differ
// for the "Ostrich"-suffixed products (e.g. product 165 is "SPY_OSTRICH" internally, ticker "SPY").
// The platform lists many more products than this; they need a real price feed (production oracle
// / "Ostrich") wired up before they can trade for real. Add an entry here to test one with a
// live-ish price meanwhile.
var stockTickers = map[string]string{
	"TSLA": "TSLA", "NVDA": "NVDA", "AAPL": "AAPL", "QQQ": "QQQ", "SPY_OSTRICH": "SPY",
	"COIN_OSTRICH": "COIN", "GOOGL_OSTRICH": "GOOGL", "MSFT_OSTRICH": "MSFT", "AMZN_OSTRICH": "AMZN",
	"META_OSTRICH": "META", "MSTR_OSTRICH": "MSTR", "PLTR_OSTRICH": "PLTR", "HOOD_OSTRICH": "HOOD",
	"CRCL_OSTRICH": "CRCL",
}

var (
	mu          sync.RWMutex
	cryptoMids  = map[string]string{}     // raw Hyperliquid mids
	stockRaw    = map[string]*big.Float{} // raw Yahoo prices
	prices      = map[string]entry{}      // assembled, what /prices serves
	symbols     []string
	isStock     = map[string]bool{}
	stockClient = &http.Client{Timeout: 10 * time.Second}
	// factors scale a symbol's price (POST /override?symbol=ETH&factor=0.93) so a test can move a
	// price, e.g. to make a position liquidatable. factor=1 clears it. Works for stocks too.
	factors = map[string]*big.Float{}
)

const expo = -8

func mk(v *big.Float) entry {
	scaled, _ := new(big.Float).Mul(v, big.NewFloat(1e8)).Int(nil)
	var e entry
	e.Price.Type, e.Price.Hex = "BigNumber", "0x"+scaled.Text(16)
	e.Expo, e.PublishTime = expo, time.Now().Unix()
	return e
}

// rebuild assembles `prices` from the latest raw crypto/stock caches plus test overrides. Called
// after either raw cache refreshes.
func rebuild() {
	mu.Lock()
	defer mu.Unlock()
	next := map[string]entry{}
	for _, s := range symbols {
		var f *big.Float
		switch {
		case s == "USDC" || s == "USDT":
			f = big.NewFloat(1)
		case isStock[s] && stockRaw[s] != nil:
			f = new(big.Float).Copy(stockRaw[s])
		case cryptoMids[s] != "":
			f, _ = new(big.Float).SetString(cryptoMids[s])
		default:
			f = big.NewFloat(1) // placeholder: no live price source configured
		}
		if k := factors[s]; k != nil {
			f = new(big.Float).Mul(f, k)
		}
		next[s] = mk(f)
	}
	prices = next
}

func refreshCrypto() error {
	resp, err := http.Post("https://api.hyperliquid.xyz/info", "application/json", strings.NewReader(`{"type":"allMids"}`))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var mids map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&mids); err != nil {
		return err
	}
	mu.Lock()
	cryptoMids = mids
	mu.Unlock()
	rebuild()
	return nil
}

// yahooChart is the subset of https://query1.finance.yahoo.com/v8/finance/chart/<symbol> used here.
type yahooChart struct {
	Chart struct {
		Result []struct {
			Meta struct {
				RegularMarketPrice float64 `json:"regularMarketPrice"`
			} `json:"meta"`
		} `json:"result"`
		Error any `json:"error"`
	} `json:"chart"`
}

func fetchYahooPrice(symbol string) (*big.Float, error) {
	req, err := http.NewRequest(http.MethodGet, "https://query1.finance.yahoo.com/v8/finance/chart/"+symbol+"?interval=1d&range=1d", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0") // Yahoo 429s requests with no UA
	resp, err := stockClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("yahoo finance %s: %s", symbol, resp.Status)
	}
	var c yahooChart
	if err := json.NewDecoder(resp.Body).Decode(&c); err != nil {
		return nil, err
	}
	if len(c.Chart.Result) == 0 || c.Chart.Result[0].Meta.RegularMarketPrice <= 0 {
		return nil, fmt.Errorf("yahoo finance %s: no price in response (error: %v)", symbol, c.Chart.Error)
	}
	return big.NewFloat(c.Chart.Result[0].Meta.RegularMarketPrice), nil
}

func refreshStocks() {
	for symbol, ticker := range stockTickers {
		p, err := fetchYahooPrice(ticker)
		if err != nil {
			log.Printf("stock price %s (%s): %v (keeping the last known price)", symbol, ticker, err)
			continue
		}
		mu.Lock()
		stockRaw[symbol] = p
		mu.Unlock()
	}
	rebuild()
}

func serve(w http.ResponseWriter, tokens []string) {
	mu.RLock()
	defer mu.RUnlock()
	data := map[string]entry{}
	var list []string
	for _, t := range tokens {
		if e, ok := prices[t]; ok {
			data[t] = e
			list = append(list, t)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"data": data, "tokens": list, "message": "ok"})
}

func main() {
	addr := flag.String("addr", ":8099", "listen address")
	flag.Parse()
	if os.Getenv("ENV") == "" {
		os.Setenv("ENV", "MAINNET")
	}
	contractUtils.Init()
	seen := map[string]bool{}
	for _, s := range contractUtils.PRODUCT_ID_SYMBOL_TO_MAP {
		if s != "" && !seen[s] {
			seen[s] = true
			symbols = append(symbols, s)
		}
	}
	for _, s := range contractUtils.ALL_COLLATERAL_TOKEN_SYMBOLS {
		if !seen[s] {
			seen[s] = true
			symbols = append(symbols, s)
		}
	}
	for s := range stockTickers {
		isStock[s] = true
	}
	if err := refreshCrypto(); err != nil {
		log.Fatal(err)
	}
	refreshStocks()
	missing := []string{}
	for _, s := range symbols {
		if _, ok := prices[s]; !ok || (!isStock[s] && s != "USDC" && s != "USDT" && cryptoMids[s] == "") {
			missing = append(missing, s)
		}
	}
	log.Printf("serving %d symbols (%d live stocks via Yahoo); %d with no live price source (placeholder 1): %v",
		len(symbols), len(stockTickers), len(missing), missing)
	go func() {
		for range time.Tick(2 * time.Second) {
			if err := refreshCrypto(); err != nil {
				log.Printf("refresh crypto: %v", err)
			}
		}
	}()
	go func() {
		for range time.Tick(30 * time.Second) {
			refreshStocks()
		}
	}()
	// alert sink: point DISCORD_*_WEBHOOK here so a test stack never posts to a real channel
	http.HandleFunc("/discord", func(w http.ResponseWriter, r *http.Request) {
		var m map[string]any
		_ = json.NewDecoder(r.Body).Decode(&m)
		log.Printf("ALERT %v", m["content"])
		w.WriteHeader(http.StatusNoContent)
	})
	http.HandleFunc("/override", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		sym := r.URL.Query().Get("symbol")
		k, ok := new(big.Float).SetString(r.URL.Query().Get("factor"))
		if !ok || k.Sign() <= 0 {
			http.Error(w, "factor must be a positive number", http.StatusBadRequest)
			return
		}
		mu.Lock()
		if k.Cmp(big.NewFloat(1)) == 0 {
			delete(factors, sym)
		} else {
			factors[sym] = k
		}
		mu.Unlock()
		log.Printf("OVERRIDE %s x%s", sym, k.Text('f', 4))
		rebuild()
		w.WriteHeader(http.StatusNoContent)
	})
	http.HandleFunc("/allprices", func(w http.ResponseWriter, r *http.Request) { serve(w, symbols) })
	http.HandleFunc("/prices", func(w http.ResponseWriter, r *http.Request) {
		serve(w, strings.Split(r.URL.Query().Get("tokens"), ","))
	})
	fmt.Printf("oracle stand-in on %s\n", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
