package cutils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

func GetUnrealizedPnL(subaccountID string) (map[uint]string, error) {
	baseURL := os.Getenv("BALANCE_SERVER_URL")
	if baseURL == "" {
		return nil, fmt.Errorf("BALANCE_SERVER_URL environment variable not set")
	}

	url := fmt.Sprintf("%s/balance/getTotalPnL?subaccountID=%s", baseURL, subaccountID)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error making HTTP request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-OK HTTP status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading HTTP response body: %v", err)
	}

	var response struct {
		PnLPerToken map[uint]string `json:"pnlPerToken"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("error unmarshalling HTTP response: %v", err)
	}

	return response.PnLPerToken, nil
}
