package controller

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/xlog"

	"os"

	"github.com/gin-gonic/gin"
)

type IPController struct {
	apiKeys       []string
	currentKey    int
	keyMutex      sync.Mutex
	countryCodeDB *db.CountryCodeDB
}

var notAllowedCountries = []string{
	"CU", "IR", "KP", "US",
}

// List of allowed countries

func RegisterIPController(r *gin.RouterGroup) {
	ipController := IPController{
		currentKey:    0,
		countryCodeDB: &db.CountryCodeDB{},
	}
	rg := r.Group("/ip")

	// Endpoint
	// rg.GET("/check/:subaccountID", ipController.CheckIPBlocked)
	rg.GET("/check", ipController.CheckIPBlocked)
}

// getClientIP retrieves the client's IP address from the request headers
func getClientIP(ctx *gin.Context) string {
	ipAddress := ctx.Request.Header.Get("X-Forwarded-For")
	if ipAddress == "" {
		ipAddress = ctx.Request.Header.Get("X-Real-IP")
	}
	if ipAddress == "" {
		ipAddress = ctx.ClientIP()
	}
	return ipAddress
}

// CheckIPBlocked retrieves the IP address from the request headers, checks if the country code is already in the database,
// if not, gets the country code using the ipgeolocation.io API, and stores the country code and subaccount ID in the database.
func (ic *IPController) CheckIPBlocked(ctx *gin.Context) {

	// Split the API keys by comma if the environment variable is set
	apiKeysEnv := os.Getenv("IP_API_KEYS")
	if apiKeysEnv != "" {
		ic.apiKeys = strings.Split(apiKeysEnv, ",")
	} else {
		// Default API key if environment variable is not set
		ic.apiKeys = []string{""}
	}

	ipAddress := getClientIP(ctx)
	if ipAddress == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Unable to determine client IP"})
		return
	}
	xlog.Infof("IP Controller - IP Address: %s\n", ipAddress)
	// subaccountID := ctx.Param("subaccountID")

	// canBeBlocked, _ := (&db.LogxTokenDB{}).CanBeBlockedCheckvesting(subaccountID)

	// // If the user cannot be blocked (returns false)
	// if !canBeBlocked {
	// 	ctx.JSON(http.StatusOK, gin.H{
	// 		"allowed": true,
	// 		"message": "Access from this country is allowed",
	// 	})
	// 	return
	// }
	// subaccountID = strings.ToLower(subaccountID)

	// Get country code using ipgeolocation.io API
	countryCode, err := ic.getCountryCode(ipAddress)
	if err != nil {
		// ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		// return
		ctx.JSON(http.StatusOK, gin.H{"allowed": true, "message": "Access from this country is allowed"})
		return
	}

	// Check if the country is in the allowed list
	isAllowed := true
	for _, allowedCountry := range notAllowedCountries {
		if countryCode == allowedCountry {
			isAllowed = false
			break
		}
	}

	if !isAllowed {
		ctx.JSON(http.StatusOK, gin.H{"allowed": false, "message": "Access from this country is not allowed"})
		return
	} else {
		ctx.JSON(http.StatusOK, gin.H{"allowed": true, "message": "Access from this country is allowed"})
		return
	}
}

// getCountryCode fetches the country code for a given IP address using the ipgeolocation.io API.
func (ic *IPController) getCountryCode(ipAddress string) (string, error) {

	var url string
	var resp *http.Response
	var err error
	// Store the country code and subaccount ID in the database
	// countryCode := ic.countryCodeDB.GetCountryCodeBySubaccount(subaccountID)
	// if countryCode != nil {
	// 	xlog.Infof("IP Controller - Country code found in database: %s\n", countryCode.CountryCode)
	// 	return countryCode.CountryCode, nil
	// }

	for i := 0; i < len(ic.apiKeys); i++ {
		// Get the current API key using round-robin
		apiKey := ic.getCurrentAPIKey()

		// Construct the URL
		url = fmt.Sprintf("https://api.ipgeolocation.io/ipgeo?apiKey=%s&ip=%s&fields=geo", apiKey, ipAddress)
		// Make the API request
		resp, err = http.Get(url)
		if err != nil {
			return "", fmt.Errorf("error making request to ipgeolocation.io API: %v", err)
		}

		// Check if the status code is 429 (rate limit exceeded)
		if resp.StatusCode == http.StatusTooManyRequests {
			// Move to the next API key
			ic.moveToNextAPIKey()
			resp.Body.Close()
			continue
		}

		// If we get a response other than 429, break the loop
		break
	}

	// If after trying all keys, we still get an error
	if resp == nil {
		return "", fmt.Errorf("all API keys have reached the rate limit")
	} 

	defer resp.Body.Close()
	
	if resp.StatusCode == http.StatusTooManyRequests {
		return "", fmt.Errorf("rate limit exceeded for all API keys")
	}

	// Handle other status codes
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("received non-OK response from ipgeolocation.io API: %v, response: %s", resp.Status, body)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %v", err)
	}

	var result struct {
		CountryCode2 string `json:"country_code2"`
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return "", fmt.Errorf("error unmarshalling response body: %v", err)
	}

	// ic.countryCodeDB.CreateCountryCode(subaccountID, result.CountryCode2)

	return result.CountryCode2, nil
}

// getCurrentAPIKey returns the current API key and updates the index for round-robin.
func (ic *IPController) getCurrentAPIKey() string {
	ic.keyMutex.Lock()
	defer ic.keyMutex.Unlock()
	key := ic.apiKeys[ic.currentKey]
	return key
}

// moveToNextAPIKey moves to the next API key in the array using round-robin.
func (ic *IPController) moveToNextAPIKey() {
	ic.keyMutex.Lock()
	defer ic.keyMutex.Unlock()
	ic.currentKey = (ic.currentKey + 1) % len(ic.apiKeys)
}
