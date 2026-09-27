package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gin-gonic/gin"
)

type MutateRedisValue struct {
	Key string `json:"key"`  //when encoding JSON use "key" as the JSON key 
	Value string `json:"value"`
	ProductId string `json:"productId"`
}

type MutateRequestObject struct {
	MutationObject []MutateRedisValue `json:"mutationObject"`
	Password string `json:"password"`
}

func IsAllowedToChangeRedisValue(ctx *gin.Context) {
	bodyBytes, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		ctx.AbortWithError(http.StatusBadRequest, fmt.Errorf("could not read request body"))
		return
	}

	// Restore the io.ReadCloser to the original state so that this object can be passed to the mutation function
	ctx.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	// Unmarshal the body
	var validationObj MutateRequestObject
	if err := json.Unmarshal(bodyBytes, &validationObj); err != nil {
		ctx.AbortWithError(http.StatusBadRequest, fmt.Errorf("unable to extract password from object"))
		return
	}
	if validationObj.Password != os.Getenv("PASSWORD") {
		xlog.Errorf("Validation failed")
		ctx.AbortWithError(http.StatusUnauthorized, fmt.Errorf("validation failed - unauthorized"))
		return
	}
	ctx.Next()
}


type AuthPair struct {
	LogxKey    string
	LogxSecret string
	BrokerId   uint
}

func (ap *AuthPair) isValidAuthPair() bool {
	if ap.LogxKey == "" || ap.LogxSecret == "" {
		fmt.Println("Auth pair key(s) must not be empty")
		return false
	}
	if len(ap.LogxKey) != 32 || len(ap.LogxSecret) != 64 {
		fmt.Println("LogxKey len should be 32 and LogxSecret len should 64")
		return false
	}
	return true
}

// TODO: Move to utils
func (ap *AuthPair) getSecretHash() (string, error) {
	secretHex := ap.LogxSecret
	privateKey, err := crypto.HexToECDSA(secretHex)
	if err != nil {
		return "", err
	}
	publicKeySplitHex := hexutil.Encode(crypto.FromECDSAPub(&privateKey.PublicKey))[2:]
	return publicKeySplitHex, nil
}

// 1.a TODO: Find auth ob in local cache
// 1.b Find auth ob in db
// 2.a If not present return error
// 2.b If present get subaccount info
func (ap *AuthPair) getSubaccountAfterVerification() (*db.SubaccountTable, error) {
	secretHash, err := ap.getSecretHash()
	if err != nil {
		fmt.Println("Error getting secret hash")
		return nil, err
	}

	auth := (&db.AuthDB{}).GetByKey(ap.LogxKey)
	if auth == nil || secretHash != auth.SecretHash {
		return nil, fmt.Errorf("invalid auth keys")
	}

	if subaccount := (&db.SubaccountDB{}).GetByIdBrokerId(ap.BrokerId, auth.SubaccountId); subaccount == nil {
		return nil, fmt.Errorf("subaccount not found")
	} else {
		return subaccount, nil
	}
}

type RequireAuthHeader struct {
	LogxKey      string `header:"Logx-Key"`
	LogxSecret   string `header:"Logx-Secret"`
	BrokerKey    string `header:"Broker-Key"`
	BrokerSecret string `header:"Broker-Secret"`
	BrokerId     uint   `header:"Broker-Id" binding:"required"`
}

// TODO: Utilise broker id as well for validation purposes
func RequireAuth(ctx *gin.Context) {
	// Get headers from both formats
	var requestHeader RequireAuthHeader
	if err := ctx.BindHeader(&requestHeader); err != nil {
		ctx.AbortWithError(http.StatusBadRequest, err)
		return
	}

	// If Broker-Key/Secret are provided, use them; otherwise, fallback to Logx-Key/Secret
	authKey := requestHeader.BrokerKey
	authSecret := requestHeader.BrokerSecret

	if authKey == "" || authSecret == "" {
		// Fallback to Logx headers
		authKey = requestHeader.LogxKey
		authSecret = requestHeader.LogxSecret
	}

	// Use the determined auth credentials
	authPair := AuthPair{
		LogxKey:    authKey,
		LogxSecret: authSecret,
		BrokerId:   requestHeader.BrokerId,
	}

	if !authPair.isValidAuthPair() {
		ctx.AbortWithError(http.StatusUnauthorized, fmt.Errorf("validation failed - unauthorized"))
		return
	}

	if subaccount, err := authPair.getSubaccountAfterVerification(); err != nil {
		xlog.Errorf("could not find subaccount - unauthorized: %v", err)
		cutils.ApiAbort(ctx, http.StatusUnauthorized, "could not find subaccount - unauthorized")
		return
	} else {
		ctx.Set("current-subaccount", subaccount)
	}
	ctx.Next()
}

func RequireSessionKey(ctx *gin.Context) {
	brokerIdStr := ctx.GetHeader("Broker-Id")

	// Try to get session key and user address from both header formats
	sessionKey := ctx.GetHeader("Broker-Signer-Address")
	userAddress := ctx.GetHeader("Broker-User-Address")

	// Fallback to Logx headers if necessary
	if sessionKey == "" {
		sessionKey = ctx.GetHeader("Logx-Signer-Address")
	}
	if userAddress == "" {
		userAddress = ctx.GetHeader("Logx-User-Address")
	}

	// Check if we have the required values after fallback
	if sessionKey == "" || userAddress == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Missing signer address or user address"})
		ctx.Abort()
		return
	}

	// Convert brokerId from string to uint
	brokerId, err := strconv.ParseUint(brokerIdStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid broker-id"})
		ctx.Abort()
		return
	}

	// Get subAccountId from the sessionKey
	signingKey := (&db.SigningKeyDB{}).GetBySigningAddress(sessionKey)
	if signingKey == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Session key not found"})
		ctx.Abort()
		return
	}
	subAccountId := signingKey.SubaccountID

	subAccountDbRow := (&db.SubaccountDB{}).GetByIdBrokerId(uint(brokerId), fmt.Sprint(subAccountId))
	if subAccountDbRow == nil {
		cutils.ApiAbort(ctx, http.StatusNotFound, "Subaccount not found")
		return
	}

	ethAddress := subAccountDbRow.EthAddress

	if ethAddress == userAddress {
		ctx.Next()
	} else {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "user address not mapped to signer address"})
		ctx.Abort()
	}
}

func RequireTimestamp(ctx *gin.Context) {
	// Try both header formats
	timestamp := ctx.GetHeader("Broker-Timestamp")
	if timestamp == "" {
		timestamp = ctx.GetHeader("Logx-Timestamp")
	}

	if timestamp == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Missing timestamp header"})
		ctx.Abort()
		return
	}

	timestampInt64, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid Timestamp: %v", err)})
		ctx.Abort()
		return
	}

	const MINUTE = 60 * 1000
	const TIMEOUT = 2 * MINUTE

	currentTs := time.Now().UnixMilli()

	if timestampInt64 > currentTs {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Timestamp"})
		ctx.Abort()
	}

	if currentTs-timestampInt64 > TIMEOUT {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Timestamp"})
		ctx.Abort()
	}

	ctx.Next()
}

func GetCurrentSubaccount(ctx *gin.Context) (*db.SubaccountTable, error) {
	untypeSubaccount, found := ctx.Get("current-subaccount")
	if !found {
		return nil, fmt.Errorf("subaccount not found")
	}

	curSubaccount, ok := untypeSubaccount.(*db.SubaccountTable)

	if !ok {
		return nil, fmt.Errorf("current subaccount type assertion failed")
	}

	return curSubaccount, nil
}

type RequireSignatureHeader struct {
	Signature           string `header:"Logx-Signature"`        // No longer 'binding:"required"'
	SignerAddress       string `header:"Logx-Signer-Address"`   // No longer 'binding:"required"'
	BrokerSignature     string `header:"Broker-Signature"`      // New header
	BrokerSignerAddress string `header:"Broker-Signer-Address"` // New header
	BrokerId            uint   `header:"Broker-Id" binding:"required"`
}

// NOTE: Must be followed by require auth
func RequireSignature(ctx *gin.Context) {
	var requestHeader RequireSignatureHeader
	if err := ctx.BindHeader(&requestHeader); err != nil {
		fmt.Printf("Error binding header: %v", err)
		cutils.ApiAbort(ctx, http.StatusBadRequest, "Invalid request header")
		return
	}

	// If Broker headers are provided, use them; otherwise, fallback to Logx headers
	signature := requestHeader.BrokerSignature
	signerAddress := requestHeader.BrokerSignerAddress

	if signature == "" || signerAddress == "" {
		// Fallback to Logx headers
		signature = requestHeader.Signature
		signerAddress = requestHeader.SignerAddress
	}

	// Check if we have the required values after fallback
	if signature == "" || signerAddress == "" {
		fmt.Printf("Error: Missing signature or signer address")
		cutils.ApiAbort(ctx, http.StatusBadRequest, "Missing signature or signer address")
		return
	}

	// Soft check for signature
	if len(signature) != 132 {
		fmt.Printf("Error: Invalid signature")
		cutils.ApiAbort(ctx, http.StatusBadRequest, "Invalid signature")
		return
	}

	signingKey := (&db.SigningKeyDB{}).GetBySigningAddress(signerAddress)
	if signingKey == nil {
		fmt.Printf("Error getting signing key")
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "Session key not found for signer address")
		return
	}

	// Check broker id are same
	if signingKey.BrokerId != requestHeader.BrokerId {
		msg := fmt.Sprintf("Error: Signing key broker id %d does not match request broker id %d", signingKey.BrokerId, requestHeader.BrokerId)
		cutils.ApiAbort(ctx, http.StatusForbidden, msg)
		return
	}

	// Get the subaccount from the context
	curSubaccount, err := GetCurrentSubaccount(ctx)
	if err != nil {
		fmt.Printf("Error getting current subaccount: %v. RequireAuth must be used before require RequireSignature", err)
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "Require signature failed as subaccount not found")
		return
	}

	// Verify signingKey belongs to the subaccount
	if signingKey.SubaccountID != curSubaccount.ID {
		fmt.Printf("Error: Signing key does not belong to the subaccount")
		cutils.ApiAbort(ctx, http.StatusForbidden, "Signing key does not belong to the subaccount")
		return
	}

	ctx.Next()
}
