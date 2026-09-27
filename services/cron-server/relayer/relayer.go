package relayer

import (
	"encoding/json"
	"fmt"
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/xclient"
	"math/big"
	"net/http"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/robfig/cron/v3"
)

func RelayerBalanceCheckCron() {
	c := cron.New(cron.WithSeconds())

	relayerAddress := os.Getenv("RELAYER_ADDRESS")
	coinMarketCapAPIKey := os.Getenv("COINMARKETCAP_API_KEY")
	coinMarketCapURL := os.Getenv("COINMARKETCAP_URL")

	if relayerAddress == "" {
		xlog.Errorf("env for RELAYER_ADDRESS is not set.")
		return
	}
	if coinMarketCapAPIKey == "" {
		xlog.Errorf("env for COINMARKETCAP_API_KEY is not set.")
		return
	}
	if coinMarketCapURL == "" {
		xlog.Errorf("env for COINMARKETCAP_URL is not set.")
		return
	}

	relayerChains := contractUtils.RELAYER_CHAINS

	c.AddFunc("0 */15 * * * *", func() {
		xlog.Infof("Running relayer balance check cron job")

		tokenPrices, err := FetchTokenPrices(coinMarketCapAPIKey, coinMarketCapURL)
		if err != nil {
			xlog.Errorf("Relayer Cron - Error fetching token prices: %v", err)
			return
		}

		err = CheckAndAlertRelayerBalances(relayerChains, relayerAddress, tokenPrices)
		if err != nil {
			xlog.Errorf("Relayer Cron - Error checking relayer balances: %v", err)
		}
	})

	c.Start()

	select {}
}

func FetchTokenPrices(apiKey, apiURL string) (map[string]float64, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("X-CMC_PRO_API_KEY", apiKey)
	req.Header.Set("Accepts", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch token prices: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-200 status code: %d", resp.StatusCode)
	}

	// Decode the response JSON
	var data struct {
		Data map[string]struct {
			Quote struct {
				USD struct {
					Price float64 `json:"price"`
				} `json:"USD"`
			} `json:"quote"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("error decoding response: %v", err)
	}

	prices := make(map[string]float64)
	for symbol, info := range data.Data {
		prices[symbol] = info.Quote.USD.Price
	}

	return prices, nil
}

func CheckAndAlertRelayerBalances(chains []contractUtils.Chain, relayerAddress string, tokenPrices map[string]float64) error {
	for _, chain := range chains {
		balance, err := GetRelayerBalanceForChain(chain, relayerAddress)
		if err != nil {
			xlog.Errorf("Relayer Cron - Error getting balance for chain %s: %v", chain.Name, err)
			continue
		}

		priceInUSD := tokenPrices[chain.NativeId]
		balanceInUSD := balance * priceInUSD

		xlog.Infof("Relayer Cron - Balance for %s: $%.2f", chain.Name, balanceInUSD)

		if balanceInUSD < chain.Threshold {
			message := fmt.Sprintf("Relayer balance for %s is below threshold, Current Balance: $%.2f", chain.Name, balanceInUSD)
			xlog.Infof("Relayer Cron - Sent alert to Discord for chain %s: %s", chain.Name, message)
			xclient.GlobalDiscordClient.SendWebhookMessage(message)
		}
	}

	return nil
}

func GetRelayerBalanceForChain(chain contractUtils.Chain, relayerAddress string) (float64, error) {
	// Connect to the chain's RPC endpoint
	client, err := rpc.Dial(chain.Rpc)
	if err != nil {
		return 0, fmt.Errorf("failed to connect to RPC for chain %s: %v", chain.Name, err)
	}
	defer client.Close()

	// Fetch balance in native token
	var balanceHex string
	err = client.Call(&balanceHex, "eth_getBalance", relayerAddress, "latest")
	if err != nil {
		return 0, fmt.Errorf("failed to fetch balance for chain %s: %v", chain.Name, err)
	}

	// Convert balance from hex to bigInt
	balance := new(big.Int)
	balance.SetString(balanceHex[2:], 16)

	// Convert balance from wei to ether
	balanceInEther := new(big.Float).Quo(new(big.Float).SetInt(balance), big.NewFloat(1e18))

	balanceFloat64, _ := balanceInEther.Float64()

	return balanceFloat64, nil
}
