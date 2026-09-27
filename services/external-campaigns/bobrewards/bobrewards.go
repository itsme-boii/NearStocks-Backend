package bobrewards

import (
	"bytes"
	"encoding/json"
	// "fmt"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"io"
	"math"
	"math/big"
	"net/http"
	"os"
	"time"

	"github.com/robfig/cron/v3"
)

type Logos struct {
	Default string `json:"default,omitempty"`
}

type PartnerPoints struct {
	CurrentPoints       string `json:"current_points"`
	CurrentVotingPoints string `json:"current_voting_points"`
}


type Transfer struct {
	ToAddress string  `json:"toAddress"`
	Points    float64 `json:"points"`
}

type DistributePointsRequest struct {
	Transfers []Transfer `json:"transfers"`
}

// Function to call the /distribute-points API
func distributePoints(transfers []Transfer, votingPoints bool) {
	url := ""
	if votingPoints {
		url = os.Getenv("BOB_BASE_API_URL") + "/distribute-voting-points"
	} else {
		url = os.Getenv("BOB_BASE_API_URL") + "/distribute-points"
	}
	apiKey := os.Getenv("BOB_API_KEY")

	if apiKey == "" {
		xlog.Errorf("API key is not set in the environment variables")
		return
	}

	payload := DistributePointsRequest{
		Transfers: transfers,
	}

	// Encode payload to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		xlog.Errorf("Failed to marshal request body: %v", err)
		return
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		xlog.Errorf("Failed to create HTTP request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		xlog.Errorf("Error sending request to distribute points: %v", err)
		return
	}
	defer resp.Body.Close()

	// Check the response
	if resp.StatusCode != http.StatusOK {
		// Read and log the response body for debugging
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			xlog.Errorf("Failed to read response body: %v", err)
		} else {
			bodyString := string(bodyBytes)
			xlog.Errorf("Failed to distribute points, status code: %d, response: %s", resp.StatusCode, bodyString)
		}
	} else {
		xlog.Infof("Points distributed successfully")
	}
}

func calcSpicePoints(pointsToGive string) []Transfer {
	newDeposits, err := (&db.DepositWithdrawDB{}).GetNetSumForSourceChain(60808)
	if err != nil {
		return nil
	}

	totalNetSum := big.NewInt(0)
	for _, deposit := range newDeposits {
		totalNetSum.Add(totalNetSum, deposit.NetSum)
	}
	if totalNetSum.Cmp(big.NewInt(0)) == 0 {
		xlog.Infof("No net deposits to distribute spice points into")
		return nil
	}

	currentPoints, ok := new(big.Float).SetString(pointsToGive)
	if !ok {
		xlog.Errorf("Failed to convert CurrentPoints to big.Float")
		return nil
	}

	var transfers []Transfer

	for _, deposit := range newDeposits {
		// Calculate the ratio of subaccount's NetSum to total NetSum
		ratio := new(big.Float).Quo(new(big.Float).SetInt(deposit.NetSum), new(big.Float).SetInt(totalNetSum))

		// Calculate points to distribute for this subaccount
		subaccountPoints := new(big.Float).Mul(ratio, currentPoints)

		// Convert points from big.Float to float64 for API
		subaccountPointsFloat, _ := subaccountPoints.Float64()

		// Round down to one decimal place using math.Floor to ensure points are not over-distributed
		subaccountPointsFloat = math.Floor(subaccountPointsFloat*10) / 10

		ethAddress, _ := cutils.ExtractETHAddress(deposit.SubaccountId)

		transfers = append(transfers, Transfer{
			ToAddress: ethAddress,
			Points:    subaccountPointsFloat,
		})
	}

	return transfers
}


func fetchPointsToGive() *PartnerPoints {
	baseURL := os.Getenv("BOB_BASE_API_URL")
	apiKey := os.Getenv("BOB_API_KEY")
	if baseURL == "" || apiKey == "" {
		xlog.Infof("BOB_BASE_API_URL or BOB_API_KEY environment variable is not set")
		return nil
	}

	url := baseURL + "/partner-s3/points"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		xlog.Errorf("Error creating request: %v", err)
		return nil
	}
	req.Header.Set("x-api-key", apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		xlog.Errorf("Error making request to the API: %v", err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		xlog.Errorf("Error: received status code %d", resp.StatusCode)
		return nil
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		xlog.Errorf("Error reading response body: %v", err)
		return nil
	}

	var points PartnerPoints
	err = json.Unmarshal(body, &points)
	if err != nil {
		xlog.Errorf("Error parsing JSON: %v", err)
		return nil
	}

	return &points
}

func distributeRewards() {
	xlog.Infof("Initiating SPICE points distribution")

	// 1 Fetch Accumulated bob points
	logxInfo := fetchPointsToGive()
	if logxInfo != nil {
		xlog.Infof("Current Points to Give out: %+v", logxInfo.CurrentPoints)
	} else {
		xlog.Infof("LogX partner info not found.")
		return
	}
	if logxInfo.CurrentPoints == "0" {
		xlog.Infof("Current Points are zero, nothing to distribute.")
		return
	}

	// 2 Calculate CurrentPoints for each subaccount based on ratio of NetSum Deposit
	transfers := calcSpicePoints(logxInfo.CurrentPoints)
	if transfers == nil {
		xlog.Infof("No subaccount found with net positive deposit, no one to distribute to.")
	} else {

		// fmt.Print(transfers)

		transferSum := new(big.Float).SetFloat64(0)
		for _, transfer := range transfers {
			pointsAsBigFloat := new(big.Float).SetFloat64(transfer.Points)
			transferSum.Add(transferSum, pointsAsBigFloat)
		}
		xlog.Infof("Total SPICE points beings distributed: ", transferSum)

		// 3. Call the /distribute-points API
		distributePoints(transfers, false)
	}

	votingPointsTransfers := calcSpicePoints(logxInfo.CurrentVotingPoints)
	xlog.Infof("Current Voting Points to Give out: %+v", logxInfo.CurrentVotingPoints)
	if logxInfo.CurrentVotingPoints == "0" {
		xlog.Infof("Current Voting Points are zero, nothing to distribute.")
		return
	}
	if votingPointsTransfers == nil {
		xlog.Infof("No subaccount found with net positive deposit, no one to distribute to voting points to")
	} else {
		transferSum := new(big.Float).SetFloat64(0)
		for _, transfer := range votingPointsTransfers {
			pointsAsBigFloat := new(big.Float).SetFloat64(transfer.Points)
			transferSum.Add(transferSum, pointsAsBigFloat)
		}
		xlog.Infof("Total SPICE points beings distributed for voting: ", transferSum)
		distributePoints(votingPointsTransfers, true)
	}

}

func StartRewardDistribution() {
	loc, err := time.LoadLocation("UTC")
	if err != nil {
		panic(err)
	}

	// Create a new cron scheduler with UTC time zone
	c := cron.New(cron.WithSeconds(), cron.WithLocation(loc))

	// cron job to run every hour in the specified time zone (UTC)
	c.AddFunc("0 0 * * * *", func() {
		distributeRewards()
	})

	c.Start()

	// Keep the scheduler running
	select {}
}