package controller

import (
	"fmt"
	"github/eugenix-io/logx-inf-backend/contract"
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/transaction"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/services/api-server/middleware"
	"github/eugenix-io/logx-inf-backend/services/api-server/services"
	"net/http"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AuthController struct {
	accountService *services.AccountService
}

type AuthData struct {
	LogxKey        string `json:"logx_key,omitempty"`
	LogxSecret     string `json:"logx_secret,omitempty"`
	BrokerKey      string `json:"broker_key"`
	BrokerSecret   string `json:"broker_secret"`
	logxSecretHash string
}

func RegisterAuthController(
	r *gin.RouterGroup,
) {
	authController := AuthController{
		accountService: services.NewAccountService(),
	}
	rg := r.Group("/auth")

	// Endpoints
	rg.POST("", middleware.RequireSequencer, authController.RegisterAuth)
	// TODO: Remove this experimental endpoint later - will be replaced with on-chain transaction
	rg.POST("/canton-experimental", middleware.RequireSequencer, authController.RegisterAuthCantonExperimental)
}

// Here private key is assumed as secret and public key is assumed as hash
func GenerateApiSecretAndHash() (string, string, error) {
	// Create a crypto private key
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		return "", "", err
	}

	// Get private and public key into hex string & remove 0x
	privateKeySplitHex := hexutil.Encode(crypto.FromECDSA(privateKey))[2:]
	publicKeySplitHex := hexutil.Encode(crypto.FromECDSAPub(&privateKey.PublicKey))[2:]

	return privateKeySplitHex, publicKeySplitHex, nil
}

func GenerateApiKey() string {
	return strings.ReplaceAll(uuid.New().String(), "-", "")
}

func GenerateAuthData() (AuthData, error) {
	apiKey := GenerateApiKey()
	apiSecret, apiSecretHash, err := GenerateApiSecretAndHash()
	if err != nil {
		return AuthData{}, err
	}
	return AuthData{
		LogxKey:        apiKey,
		LogxSecret:     apiSecret,
		BrokerKey:      apiKey,
		BrokerSecret:   apiSecret,
		logxSecretHash: apiSecretHash,
	}, nil
}

type RegisterAuthHeader struct {
	BrokerId uint `header:"Broker-Id" binding:"required"`
}
type RegisterAuthBody struct {
	SubaccountId     string `json:"subaccountId" binding:"required"`
	SigningAddress   string `json:"signingKey" binding:"required"`
	SigningSignature string `json:"signingSignature" binding:"required"`
	EthAddress       string `json:"ethAddress" binding:"required"`
	EthSignature     string `json:"ethSignature" binding:"required"`
	ExpiryTs         int64  `json:"expiryTs" binding:"required"`
	Nonce            *int64 `json:"nonce" binding:"required"`
	ChainId          *int64 `json:"chainId" binding:"required"`
}
type RegisterAuthRequest struct {
	RegisterAuthHeader
	RegisterAuthBody
}

const REGISTER_TIMEOUT = 60 * cutils.HOUR_MILLI // FIXME: Change this to 5 minutes
const EXPIRY_DURATION = 7 * cutils.DAY_MILLI

// 0. Verify expiryTs is correct ✅
// 0.01 Validate subaccountId: use broker id and ethAddress
// 0.1 Generate auth key, secret ✅
// 1. Build message from payload, Verify signature of account, Verify EthSignature ✅
// 2.1 Check broker exists ✅
// 2.a Check Subaccount exists, verify eth address ✅
// 2.b Otherwise create subaccount with id ✅
// 3 a. Check if signature already in use, if yes, verify that same subaccount linked ✅
// 3 a1. Also, check that smart contract has signature registered
// 3 b. Save signature on db with expiry ✅
// 3.1 Save secrethash and key in db with expiry ✅
// 4 Register on smart contract ✅
// 5. Generate Logx key and Secret and return ✅
func (ac *AuthController) RegisterAuth(ctx *gin.Context) {
	var _requestHeader RegisterAuthHeader
	var _requestBody RegisterAuthBody

	if err := ctx.BindHeader(&_requestHeader); err != nil {
		ctx.AbortWithError(http.StatusBadRequest, err)
		return
	}
	if err := ctx.BindJSON(&_requestBody); err != nil {
		ctx.AbortWithError(http.StatusBadRequest, err)
		return
	}

	requestObj := RegisterAuthRequest{
		RegisterAuthHeader: _requestHeader,
		RegisterAuthBody:   _requestBody,
	}
	if requestObj.ChainId != nil && *requestObj.ChainId < 0 {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "chain id cannot be negative"})
		return
	}

	subaccountHex, err := cutils.SubaccountIdToHex(requestObj.SubaccountId)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusBadRequest, "Invalid subaccount id")
		return
	}

	subAccountIdHexLower := strings.ToLower(subaccountHex)

	if cutils.IsSubAccountPaused(subAccountIdHexLower) {
		xlog.Errorf("no transactions are currently allowed - SubAccount is paused", subAccountIdHexLower)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "no transactions are currently allowed - please contact dev team"})
		ctx.Abort()
		return
	}

	currentTs := time.Now().UnixMilli()

	// NOTE: We are skipping this check for dev
	if requestObj.ExpiryTs-EXPIRY_DURATION > currentTs {
		xlog.Infof("Auth Controller - Request expiry timestamp is greater than the allowed expiry. Expiry ts: %v, current ts: %v\n", requestObj.ExpiryTs, currentTs)
		cutils.ApiAbort(ctx, http.StatusBadRequest, "Request expiry timestamp is greater than the allowed expiry")
		return
	}

	if !cutils.ValidateSubaccountId(requestObj.SubaccountId, requestObj.BrokerId, requestObj.EthAddress) {
		xlog.Infof("Auth Controller - Invalid subaccount id: %v, expected example: %v\n", requestObj.SubaccountId, cutils.CreateSubaccountId(requestObj.BrokerId, requestObj.EthAddress, 1))
		cutils.ApiAbort(ctx, http.StatusBadRequest, fmt.Sprintf("Invalid subaccount id: %v, expected example: %v", requestObj.SubaccountId, cutils.CreateSubaccountId(requestObj.BrokerId, requestObj.EthAddress, 1)))
		return
	}

	authData, err := GenerateAuthData()
	if err != nil {
		xlog.Errorf("Auth Controller - Unable to generate auth data: %v\n", err)
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "Unable to generate auth data")
		return
	}

	sessionReq := contractUtils.SessionRequest{
		SubaccountId:     requestObj.SubaccountId,
		SigningAddress:   requestObj.SigningAddress,
		EthAddress:       requestObj.EthAddress,
		ExpiryTs:         requestObj.ExpiryTs,
		ChainId:          *requestObj.ChainId,
		SigningSignature: requestObj.SigningSignature,
		EthSignature:     requestObj.EthSignature,
		Nonce:            *requestObj.Nonce,
	}

	xlog.Infof("Auth Controller - Verifying registration signatures on backend")
	if errVerf := contractUtils.VerifySessionkeyCreation(sessionReq); errVerf != nil {
		xlog.Errorf("Auth Contrller - ethAddressSignature verification failed: %v\n", errVerf)
		cutils.ApiAbort(ctx, http.StatusBadRequest, fmt.Sprintf("ethAddressSignature verification failed: %v", errVerf))
		return
	}

	xlog.Infof("Auth Controller - Checking broker in db")
	broker := (&db.BrokerDB{}).GetById(requestObj.BrokerId)
	if broker == nil {
		xlog.Infof("Auth Controller - Broker with id: %v not found\n", requestObj.BrokerId)
		cutils.ApiAbort(ctx, http.StatusBadRequest, "broker not found")
		return
	}

	transactionCounter, err := transaction.IncrementCounter(1)
	if err != nil {
		xlog.Errorf("Failed to increment transaction counter auth: %v", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to increment transaction counter",
		})
		return
	}

	response, err := services.WithNonceRedisLock(subaccountHex, func(currentNonce string) (*Response, error) {
		nonceCheck := cutils.CheckNonce(*requestObj.Nonce, currentNonce)
		if !nonceCheck {
			// ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "nonce check failed"})
			return &Response{status: http.StatusBadRequest, data: gin.H{"error": "nonce check failed"}}, fmt.Errorf("nonce check failed")
		}

		subaccount := (&db.SubaccountDB{}).GetByIdBrokerId(requestObj.BrokerId, requestObj.SubaccountId)
		if subaccount != nil {
			xlog.Infof("Auth Controller - subaccount already present in database")
			if subaccount.EthAddress != requestObj.EthAddress {
				xlog.Infof("Auth Controller - Eth address: %v does not match with subaccount's eth address: %v\n", requestObj.EthAddress, subaccount.EthAddress)
				return &Response{status: http.StatusBadRequest, data: gin.H{"error": "Eth address does not match with subaccount's eth address"}}, fmt.Errorf("eth address does not match with subaccount's eth address")
			}
		} else {
			xlog.Infof("Auth Controller - Subaccount not found. Creating new entry")
			var errSub error
			subaccount, errSub = (&db.SubaccountDB{}).Create(requestObj.EthAddress, requestObj.BrokerId, requestObj.SubaccountId)
			if errSub != nil {
				xlog.Errorf("Auth Controller - Error creating subaccount: %v\n", requestObj.SubaccountId)
				return &Response{status: http.StatusBadRequest, data: gin.H{"error": "Error creating subaccount"}}, errSub
			}
			// create a referral user entry
			referralUserData := db.ReferralUserTable{
				UserAddress: requestObj.EthAddress,
			}

			_, err := (&db.ReferralDB{}).CreateReferralUser(referralUserData)
			if err != nil {
				xlog.Errorf("Auth Controller - Error creating referral user: %v\n", err)
			}
		}

		signingKey := (&db.SigningKeyDB{}).GetBySigningAddress(requestObj.SigningAddress)
		if signingKey != nil {
			if signingKey.SubaccountID != subaccount.ID {
				xlog.Errorf("Auth Controller - Signing key already in use by another subaccount: %v\n", signingKey.SubaccountID)
				return &Response{status: http.StatusBadRequest, data: gin.H{"error: ": "Signing key already in use by another subaccount"}}, fmt.Errorf("signing key already in use by another subaccount")
			}
		} else {
			xlog.Infof("Auth Controller - Registering on smart contract...")
			// Create the signing key in the database
			if err := (&db.SigningKeyDB{}).Create(requestObj.SigningAddress, subaccount.ID, requestObj.BrokerId, uint64(requestObj.ExpiryTs)); err != nil {
				xlog.Errorf("Auth Controller - Error creating signing key in DB: %v, SigningAddress: %s, SubaccountID: %s, ExpiryTs: %d", err, requestObj.SigningAddress, subaccount.ID, requestObj.ExpiryTs)
				return &Response{status: http.StatusInternalServerError, data: gin.H{"message": "something went wrong while creating signing key"}}, err
			}

			err1 := services.IncrementNonce(subaccountHex, currentNonce)
			if err1 != nil {
				xlog.Errorf("Auth Controller - IncrementNonce failed: %v, subAccountIdHex: %s", err1, subaccountHex)
				return &Response{status: http.StatusInternalServerError, data: gin.H{"message": "something went wrong while incrementing nonce"}}, err1
			}

			xlog.Infof("Auth Controller - Incremented nonce successfully for subAccountIdHex: %s", subaccountHex)

			xlog.Infof("Auth Controller - Signing key created successfully in DB for SigningAddress: %s, SubaccountID: %s, ExpiryTs: %d", requestObj.SigningAddress, subaccount.ID, requestObj.ExpiryTs)

			// Register the session key on the contract
			if err := contract.GlobalContracts.EndpointContract.RegisterSessionKey(sessionReq, transactionCounter); err != nil {
				xlog.Errorf("Auth Controller - Error registering session key on contract: %v, subAccountIdHex: %s, SigningAddress: %s, transactionCounter: %d", err, subaccountHex, requestObj.SigningAddress, transactionCounter)
				// Attempt to mark the transaction as deleted in the DB
				services.IncrementNonce(subaccountHex, currentNonce, true)
				return &Response{status: http.StatusInternalServerError, data: gin.H{"message": "something went wrong when registering on contract"}}, err
			}

			xlog.Infof("Auth Controller - Successfully registered session key on contract for subAccountIdHex: %s, SigningAddress: %s, transactionCounter: %d", subaccountHex, requestObj.SigningAddress, transactionCounter)
		}

		if err := (&db.AuthDB{}).Create(authData.LogxKey, authData.logxSecretHash, subaccount.ID, uint64(requestObj.ExpiryTs)); err != nil {
			xlog.Errorf("Auth Controller - Error creating auth key in DB: %v, LogxKey: %s, SubaccountID: %d, ExpiryTs: %d", err, authData.LogxKey, subaccount.ID, requestObj.ExpiryTs)
			return &Response{status: http.StatusInternalServerError, data: gin.H{"message": "something went wrong while creating auth key"}}, err
		}
		// cutils.ApiSuccess(ctx, authData, "Subaccount successfully registered")

		return &Response{status: http.StatusOK, data: gin.H{"message": "Subaccount successfully registered"}}, nil
	})

	if err != nil {
		ctx.AbortWithStatusJSON(response.status, response.data)
		return
	} else {
		if requestObj.BrokerId == 2 {
			authData.LogxKey = ""
			authData.LogxSecret = ""
		}
		cutils.ApiSuccess(ctx, authData, "Subaccount successfully registered")
	}
}

type RegisterAuthCantonExperimentalBody struct {
	SigningAddress string `json:"signingAddress" binding:"required"`
	CantonPartyId  string `json:"cantonPartyId" binding:"required"`
	Signature      string `json:"signature" binding:"required"` // EIP-712 signature proving ownership
	Timestamp      int64  `json:"timestamp" binding:"required"` // Request timestamp for replay protection
	Nonce          int64  `json:"nonce" binding:"required"`     // Nonce for replay protection
}

// TODO: Remove this endpoint later - This is an experimental endpoint
// This endpoint will be replaced with an on-chain transaction-based approach
// RegisterAuthCantonExperimental creates auth key for Canton users by Canton Party ID
// 1. Verify signature to prove ownership of signing address (signing key NOT stored)
// 2. Get subaccount from Canton party ID
// 3. Generate and return auth data with 7-day expiry
func (ac *AuthController) RegisterAuthCantonExperimental(ctx *gin.Context) {
	var requestBody RegisterAuthCantonExperimentalBody

	if err := ctx.BindJSON(&requestBody); err != nil {
		ctx.AbortWithError(http.StatusBadRequest, err)
		return
	}

	xlog.Infof("Auth Controller Canton Experimental - Processing request for Canton Party ID: %s, Signing Address: %s", requestBody.CantonPartyId, requestBody.SigningAddress)

	// Fixed chain ID
	chainId := int64(1)

	// Verify timestamp is recent (within 5 minutes) to prevent replay attacks
	currentTs := time.Now().UnixMilli()
	if !cutils.IsTimestampWithinWindow(requestBody.Timestamp, 5*cutils.MINUTE_MILLI) {
		xlog.Infof("Auth Controller Canton Experimental - Request timestamp is too old or in the future. Timestamp: %d, Current: %d", requestBody.Timestamp, currentTs)
		cutils.ApiAbort(ctx, http.StatusBadRequest, "Request timestamp is too old or in the future")
		return
	}

	// Verify signature to prove ownership of signing address and bind to Canton Party ID
	xlog.Infof("Auth Controller Canton Experimental - Verifying signature for signing address: %s, Canton Party ID: %s", requestBody.SigningAddress, requestBody.CantonPartyId)
	authRequest := contractUtils.CantonAuthRegistration{
		SigningAddress: requestBody.SigningAddress,
		CantonPartyId:  requestBody.CantonPartyId,
		Timestamp:      requestBody.Timestamp,
		Nonce:          requestBody.Nonce,
		ChainId:        chainId,
	}
	xlog.Infof("Auth Controller Canton Experimental - Verifying signature for Canton Party ID: %s, Signing Address: %s", requestBody)
	if err := contractUtils.VerifyCantonAuthRegistration(authRequest, requestBody.Signature); err != nil {
		xlog.Errorf("Auth Controller Canton Experimental - Signature verification failed for %s: %v", requestBody.SigningAddress, err)
		cutils.ApiAbort(ctx, http.StatusUnauthorized, "Invalid signature - signature verification failed")
		return
	}
	xlog.Infof("Auth Controller Canton Experimental - Signature verified successfully for: %s with Canton Party ID: %s", requestBody.SigningAddress, requestBody.CantonPartyId)

	// Get Canton address mapping to find subaccount
	cantonMapping := (&db.CantonAddressDB{}).GetSubAccountIdByPartyId(requestBody.CantonPartyId)
	if cantonMapping == nil {
		xlog.Errorf("Auth Controller Canton Experimental - Canton party ID not found: %s", requestBody.CantonPartyId)
		cutils.ApiAbort(ctx, http.StatusNotFound, "Canton party ID not found")
		return
	}

	// Get subaccount
	subaccount := (&db.SubaccountDB{}).GetById(cantonMapping.SubAccountId)
	if subaccount == nil {
		xlog.Errorf("Auth Controller Canton Experimental - Subaccount not found for Canton party ID: %s, SubaccountID: %s", requestBody.CantonPartyId, cantonMapping.SubAccountId)
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "Subaccount not found")
		return
	}

	// Generate auth data
	authData, err := GenerateAuthData()
	if err != nil {
		xlog.Errorf("Auth Controller Canton Experimental - Unable to generate auth data: %v", err)
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "Unable to generate auth data")
		return
	}

	// Create auth key with 7-day expiry
	expiryTs := currentTs + EXPIRY_DURATION // 7 days

	// Check if signing key already exists, create only if it doesn't
	existingSigningKey := (&db.SigningKeyDB{}).GetBySigningAddress(requestBody.SigningAddress)
	if existingSigningKey == nil {
		// Signing key doesn't exist, create it
		xlog.Infof("Auth Controller Canton Experimental - Creating signing key in DB. Address: %s, SubaccountID: %s, ExpiryTs: %d", requestBody.SigningAddress, subaccount.ID, expiryTs)
		if err := (&db.SigningKeyDB{}).Create(requestBody.SigningAddress, subaccount.ID, 1, uint64(expiryTs)); err != nil {
			xlog.Errorf("Auth Controller Canton Experimental - Error creating signing key in DB: %v", err)
			cutils.ApiAbort(ctx, http.StatusInternalServerError, "Failed to create signing key")
			return
		}
		xlog.Infof("Auth Controller Canton Experimental - Successfully created signing key for address: %s", requestBody.SigningAddress)
	} else {
		xlog.Infof("Auth Controller Canton Experimental - Signing key already exists for address: %s, skipping creation", requestBody.SigningAddress)
	}

	xlog.Infof("Auth Controller Canton Experimental - Creating auth key in DB. LogxKey: %s, SubaccountID: %s, ExpiryTs: %d", authData.LogxKey, subaccount.ID, expiryTs)
	if err := (&db.AuthDB{}).Create(authData.LogxKey, authData.logxSecretHash, subaccount.ID, uint64(expiryTs)); err != nil {
		xlog.Errorf("Auth Controller Canton Experimental - Error creating auth key in DB: %v", err)
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "Failed to create auth key")
		return
	}

	xlog.Infof("Auth Controller Canton Experimental - Successfully created auth key for Canton party ID: %s, Subaccount: %s", requestBody.CantonPartyId, subaccount.ID)
	cutils.ApiSuccess(ctx, authData, "API key and secret generated successfully (experimental)")
}
