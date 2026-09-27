package tests

// import (
// 	"bytes"
// 	"encoding/json"
// 	"fmt"
// 	"github/eugenix-io/logx-inf-backend/libs/ctypes"
// 	"github/eugenix-io/logx-inf-backend/libs/cutils"
// 	"github/eugenix-io/logx-inf-backend/libs/db"
// 	"github/eugenix-io/logx-inf-backend/services/api-server/controller"
// 	"github/eugenix-io/logx-inf-backend/services/api-server/middleware"
// 	"net/http"
// 	"net/http/httptest"
// 	"strconv"
// 	"testing"
// 	"time"

// 	"math/big"
// 	"reflect"

// 	"github.com/agiledragon/gomonkey/v2"
// 	"github.com/gin-gonic/gin"
// 	. "github.com/onsi/ginkgo/v2"
// 	. "github.com/onsi/gomega"
// 	"github.com/stretchr/testify/mock"
// )

// func NewGovernanceControllerForTesting(proposalDB *MockProposalDB, voteDB *MockVoteDB, balanceClient *MockBalanceClient) *controller.GovernanceController {
// 	// Create a new instance using struct literal with exported fields
// 	gc := &controller.GovernanceController{}

// 	// Use monkey patching to patch the internal CreateProposal method
// 	patches := gomonkey.ApplyMethod(reflect.TypeOf(gc), "CreateProposal",
// 		func(c *controller.GovernanceController, ctx *gin.Context) {
// 			var request controller.ProposalRequest
// 			if err := ctx.ShouldBindJSON(&request); err != nil {
// 				cutils.ApiAbort(ctx, http.StatusBadRequest, err.Error())
// 				return
// 			}
// 			currentSubaccount, err := middleware.GetCurrentSubaccount(ctx)
// 			if err != nil {
// 				cutils.ApiAbort(ctx, http.StatusInternalServerError, "Failed to get current subaccount")
// 				return
// 			}

// 			// Validate timestamps
// 			currentTime := time.Now().Unix()
// 			if request.VotingStart < currentTime {
// 				cutils.ApiAbort(ctx, http.StatusBadRequest, "Voting start time must be in the future")
// 				return
// 			}
// 			if request.VotingEnd <= request.VotingStart {
// 				cutils.ApiAbort(ctx, http.StatusBadRequest, "Voting end time must be after voting start time")
// 				return
// 			}

// 			// Limit description length (e.g., 5000 characters)
// 			const maxDescriptionLength = 5000
// 			if len(request.Description) > maxDescriptionLength {
// 				cutils.ApiAbort(ctx, http.StatusBadRequest, fmt.Sprintf("Description exceeds maximum length of %d characters", maxDescriptionLength))
// 				return
// 			}

// 			// Validate and limit option value lengths
// 			const maxOptionValueLength = 100
// 			for i, option := range request.Options {
// 				if len(option.Value) > maxOptionValueLength {
// 					cutils.ApiAbort(ctx, http.StatusBadRequest, fmt.Sprintf("Option %d value exceeds maximum length of %d characters", i+1, maxOptionValueLength))
// 					return
// 				}
// 			}

// 			// Convert options to JSON string
// 			optionsJSON, err := json.Marshal(request.Options)
// 			if err != nil {
// 				cutils.ApiAbort(ctx, http.StatusInternalServerError, "Failed to process options")
// 				return
// 			}

// 			// Create proposal
// 			proposal := &db.ProposalTable{
// 				Name:                 request.Name,
// 				VotingStart:          request.VotingStart,
// 				VotingEnd:            request.VotingEnd,
// 				Description:          request.Description,
// 				Options:              string(optionsJSON),
// 				ProposerSubaccountId: currentSubaccount.ID,
// 			}

// 			result := proposalDB.Create(proposal)
// 			if result == nil {
// 				cutils.ApiAbort(ctx, http.StatusInternalServerError, "Failed to create proposal")
// 				return
// 			}

// 			cutils.ApiSuccess(ctx, result, "Proposal created successfully")
// 		})

// 	// Patch GetAllProposals
// 	patches.ApplyMethod(reflect.TypeOf(gc), "GetAllProposals",
// 		func(c *controller.GovernanceController, ctx *gin.Context) {
// 			proposals := proposalDB.GetAll()
// 			if proposals == nil {
// 				cutils.ApiAbort(ctx, http.StatusInternalServerError, "Failed to fetch proposals")
// 				return
// 			}

// 			cutils.ApiSuccess(ctx, *proposals, "Proposals fetched successfully")
// 		})

// 	// Patch GetProposalById
// 	patches.ApplyMethod(reflect.TypeOf(gc), "GetProposalById",
// 		func(c *controller.GovernanceController, ctx *gin.Context) {
// 			idStr := ctx.Param("id")
// 			proposalId, err := strconv.ParseUint(idStr, 10, 64)
// 			if err != nil {
// 				cutils.ApiAbort(ctx, http.StatusBadRequest, "Invalid proposal ID")
// 				return
// 			}

// 			proposal := proposalDB.GetByID(uint(proposalId))
// 			if proposal == nil {
// 				cutils.ApiAbort(ctx, http.StatusNotFound, "Proposal not found")
// 				return
// 			}

// 			currentSubaccount, err := middleware.GetCurrentSubaccount(ctx)
// 			var userVote *db.VoteTable

// 			if err == nil {
// 				userVote = voteDB.GetBySubaccountAndProposal(currentSubaccount.ID, uint(proposalId))
// 			}

// 			// Make sure we're constructing the response correctly
// 			response := gin.H{
// 				"proposal":  *proposal, // Dereference the pointer
// 				"user_vote": userVote,
// 			}

// 			cutils.ApiSuccess(ctx, response, "Proposal details fetched successfully")
// 		})

// 	// Patch CastVote
// 	patches.ApplyMethod(reflect.TypeOf(gc), "CastVote",
// 		func(c *controller.GovernanceController, ctx *gin.Context) {
// 			idStr := ctx.Param("id")
// 			id, err := strconv.ParseUint(idStr, 10, 64)
// 			if err != nil {
// 				cutils.ApiAbort(ctx, http.StatusBadRequest, "Invalid proposal ID")
// 				return
// 			}

// 			var request controller.VoteRequest
// 			if err := ctx.ShouldBindJSON(&request); err != nil {
// 				cutils.ApiAbort(ctx, http.StatusBadRequest, err.Error())
// 				return
// 			}

// 			// Get current authenticated subaccount
// 			currentSubaccount, err := middleware.GetCurrentSubaccount(ctx)
// 			if err != nil {
// 				cutils.ApiAbort(ctx, http.StatusInternalServerError, "Failed to get current subaccount")
// 				return
// 			}

// 			// Get proposal
// 			proposal := proposalDB.GetByID(uint(id))
// 			if proposal == nil {
// 				cutils.ApiAbort(ctx, http.StatusNotFound, "Proposal not found")
// 				return
// 			}
// 			// Check if already voted
// 			existingVote := voteDB.GetBySubaccountAndProposal(currentSubaccount.ID, uint(id))
// 			if existingVote != nil {
// 				cutils.ApiAbort(ctx, http.StatusBadRequest, "You have already voted for this proposal")
// 				return
// 			}
// 			// Check if voting is open
// 			currentTime := time.Now().Unix()
// 			if currentTime < proposal.VotingStart {
// 				cutils.ApiAbort(ctx, http.StatusBadRequest, "Voting has not started yet")
// 				return
// 			}
// 			if currentTime > proposal.VotingEnd {
// 				cutils.ApiAbort(ctx, http.StatusBadRequest, "Voting has ended")
// 				return
// 			}

// 			// Validate option
// 			var options []controller.ProposalOption
// 			if err := json.Unmarshal([]byte(proposal.Options), &options); err != nil {
// 				cutils.ApiAbort(ctx, http.StatusInternalServerError, "Invalid proposal options format")
// 				return
// 			}

// 			validOption := false
// 			for _, opt := range options {
// 				if opt.Key == request.Option {
// 					validOption = true
// 					break
// 				}
// 			}
// 			if !validOption {
// 				cutils.ApiAbort(ctx, http.StatusBadRequest, "Invalid option selected")
// 				return
// 			}

// 			// Mock the staking functionality
// 			subaccountHex, _ := cutils.SubaccountIdToHex(currentSubaccount.ID)
// 			spotBalance, _, _, _ := balanceClient.GetSpotBalance(subaccountHex)

// 			// Calculate voting power (staked tokens / 1000)
// 			stakingAmount := "0"
// 			for _, balance := range spotBalance {
// 				if balance.ProductId == 2 {
// 					stakingAmount = balance.TokenBalance
// 					break
// 				}
// 			}

// 			stakingAmountBig, _ := new(big.Float).SetString(stakingAmount)
// 			divisor := new(big.Float).SetInt64(1000)
// 			votingPower := new(big.Float).Quo(stakingAmountBig, divisor)

// 			// Create vote
// 			vote := &db.VoteTable{
// 				SubaccountId: currentSubaccount.ID,
// 				ProposalId:   uint(id),
// 				Option:       request.Option,
// 				VotingPower:  votingPower.String(),
// 				Signature:    request.Signature,
// 			}

// 			result := voteDB.Create(vote)
// 			if result == nil {
// 				cutils.ApiAbort(ctx, http.StatusInternalServerError, "Failed to cast vote")
// 				return
// 			}

// 			cutils.ApiSuccess(ctx, result, "Vote cast successfully")
// 		})

// 	return gc
// }

// func TestGovernance(t *testing.T) {
// 	RegisterFailHandler(Fail)
// 	RunSpecs(t, "Governance Suite")
// }

// // Mock implementations
// type MockProposalDB struct {
// 	mock.Mock
// }

// func (m *MockProposalDB) Create(proposal *db.ProposalTable) *db.ProposalTable {
// 	args := m.Called(proposal)
// 	if args.Get(0) == nil {
// 		return nil
// 	}
// 	return args.Get(0).(*db.ProposalTable)
// }

// func (m *MockProposalDB) GetAll() *[]db.ProposalTable {
// 	args := m.Called()
// 	if args.Get(0) == nil {
// 		return nil
// 	}
// 	return args.Get(0).(*[]db.ProposalTable)
// }

// func (m *MockProposalDB) GetByID(id uint) *db.ProposalTable {
// 	args := m.Called(id)
// 	if args.Get(0) == nil {
// 		return nil
// 	}
// 	return args.Get(0).(*db.ProposalTable)
// }

// func (m *MockProposalDB) UpdateVotingResult(id uint, result string) bool {
// 	args := m.Called(id, result)
// 	return args.Bool(0)
// }

// func (m *MockProposalDB) GetActiveProposals() *[]db.ProposalTable {
// 	args := m.Called()
// 	if args.Get(0) == nil {
// 		return nil
// 	}
// 	return args.Get(0).(*[]db.ProposalTable)
// }

// type MockVoteDB struct {
// 	mock.Mock
// }

// func (m *MockVoteDB) Create(vote *db.VoteTable) *db.VoteTable {
// 	args := m.Called(vote)
// 	if args.Get(0) == nil {
// 		return nil
// 	}
// 	return args.Get(0).(*db.VoteTable)
// }

// func (m *MockVoteDB) GetBySubaccountAndProposal(subaccountId string, proposalId uint) *db.VoteTable {
// 	args := m.Called(subaccountId, proposalId)
// 	if args.Get(0) == nil {
// 		return nil
// 	}
// 	return args.Get(0).(*db.VoteTable)
// }

// func (m *MockVoteDB) GetVotesByProposal(proposalId uint) *[]db.VoteTable {
// 	args := m.Called(proposalId)
// 	if args.Get(0) == nil {
// 		return nil
// 	}
// 	return args.Get(0).(*[]db.VoteTable)
// }

// func (m *MockVoteDB) UpdateVotingPower(id uint, buyingPower string) bool {
// 	args := m.Called(id, buyingPower)
// 	return args.Bool(0)
// }

// type MockBalanceClient struct {
// 	mock.Mock
// }

// func (m *MockBalanceClient) GetSpotBalance(subaccountHex string) ([]ctypes.SpotBalance, string, string, error) {
// 	args := m.Called(subaccountHex)
// 	return args.Get(0).([]ctypes.SpotBalance), args.String(1), args.String(2), args.Error(3)
// }

// var _ = Describe("Governance Controller", func() {
// 	var (
// 		router            *gin.Engine
// 		mockProposalDB    *MockProposalDB
// 		mockVoteDB        *MockVoteDB
// 		mockBalanceClient *MockBalanceClient
// 		gc                *controller.GovernanceController
// 		patches           *gomonkey.Patches
// 	)

// 	BeforeEach(func() {
// 		// Reset mocks
// 		mockProposalDB = new(MockProposalDB)
// 		mockVoteDB = new(MockVoteDB)
// 		mockBalanceClient = new(MockBalanceClient)

// 		// Create the controller with mocks
// 		gc = NewGovernanceControllerForTesting(mockProposalDB, mockVoteDB, mockBalanceClient)

// 		// Setup test router in test mode
// 		gin.SetMode(gin.TestMode)
// 		router = gin.New()
// 		router.Use(func(c *gin.Context) {
// 			// Mock authentication middleware
// 			c.Set("subaccount", &db.SubaccountTable{
// 				ID:       "1_0x123456_1",
// 				UserName: new(string),
// 			})
// 			c.Next()
// 		})

// 		// Register routes
// 		governanceGroup := router.Group("/governance")
// 		proposalGroup := governanceGroup.Group("/proposal")
// 		proposalGroup.GET("/all", gc.GetAllProposals)
// 		proposalGroup.GET("/:id", gc.GetProposalById)
// 		proposalGroup.POST("", gc.CreateProposal)

// 		voteGroup := governanceGroup.Group("/vote")
// 		voteGroup.POST("/proposal/:id", gc.CastVote)

// 		// Initialize patches for mocking
// 		patches = gomonkey.NewPatches()
// 	})

// 	AfterEach(func() {
// 		// Reset mocks and patches
// 		if patches != nil {
// 			patches.Reset()
// 		}
// 	})

// 	// Helper function to make HTTP requests
// 	makeRequest := func(method, url string, body interface{}) *httptest.ResponseRecorder {
// 		var reqBody []byte
// 		if body != nil {
// 			reqBody, _ = json.Marshal(body)
// 		}

// 		w := httptest.NewRecorder()
// 		req, _ := http.NewRequest(method, url, bytes.NewBuffer(reqBody))
// 		req.Header.Set("Content-Type", "application/json")
// 		router.ServeHTTP(w, req)
// 		return w
// 	}

// 	Describe("CreateProposal", func() {
// 		Context("with valid input", func() {
// 			It("should create a proposal successfully", func() {
// 				// Mock getCurrentSubaccount to return a valid subaccount
// 				patches.ApplyFunc(middleware.GetCurrentSubaccount, func(ctx *gin.Context) (*db.SubaccountTable, error) {
// 					return &db.SubaccountTable{
// 						ID:       "1_0x123456_1",
// 						UserName: new(string),
// 					}, nil
// 				})

// 				// Setup request body
// 				reqBody := controller.ProposalRequest{
// 					Name:        "Test Proposal",
// 					VotingStart: time.Now().Add(1 * time.Hour).Unix(),
// 					VotingEnd:   time.Now().Add(24 * time.Hour).Unix(),
// 					Description: "This is a <b>test</b> proposal",
// 					Options: []controller.ProposalOption{
// 						{Key: "option1", Value: "Option 1", Color: "#ff0000"},
// 						{Key: "option2", Value: "Option 2", Color: "#00ff00"},
// 					},
// 					Signature: "0x123signature",
// 				}

// 				// Setup expected response
// 				expectedProposal := &db.ProposalTable{
// 					BaseTable: db.BaseTable{
// 						ID: 1,
// 					},
// 					Name:                 reqBody.Name,
// 					VotingStart:          reqBody.VotingStart,
// 					VotingEnd:            reqBody.VotingEnd,
// 					Description:          reqBody.Description,
// 					Options:              `[{"key":"option1","value":"Option 1","color":"#ff0000"},{"key":"option2","value":"Option 2","color":"#00ff00"}]`,
// 					ProposerSubaccountId: "1_0x123456_1",
// 				}

// 				// Setup mock expectations
// 				mockProposalDB.On("Create", mock.AnythingOfType("*db.ProposalTable")).Return(expectedProposal)

// 				// Make the request
// 				w := makeRequest("POST", "/governance/proposal", reqBody)

// 				// Assert response
// 				Expect(w.Code).To(Equal(http.StatusOK))

// 				// Parse response
// 				var response map[string]interface{}
// 				err := json.Unmarshal(w.Body.Bytes(), &response)
// 				Expect(err).To(BeNil())

// 				// Verify response data
// 				Expect(response["status"]).To(Equal(float64(http.StatusOK)))

// 				// Verify mocks were called correctly
// 				mockProposalDB.AssertExpectations(GinkgoT())
// 			})

// 			It("should validate and limit description length", func() {
// 				// Mock getCurrentSubaccount
// 				patches.ApplyFunc(middleware.GetCurrentSubaccount, func(ctx *gin.Context) (*db.SubaccountTable, error) {
// 					return &db.SubaccountTable{
// 						ID:       "1_0x123456_1",
// 						UserName: new(string),
// 					}, nil
// 				})

// 				// Create overly long description (over 5000 chars)
// 				longDescription := ""
// 				for i := 0; i < 6000; i++ {
// 					longDescription += "a"
// 				}

// 				// Setup request with long description
// 				reqBody := controller.ProposalRequest{
// 					Name:        "Test Proposal",
// 					VotingStart: time.Now().Add(1 * time.Hour).Unix(),
// 					VotingEnd:   time.Now().Add(24 * time.Hour).Unix(),
// 					Description: longDescription,
// 					Options: []controller.ProposalOption{
// 						{Key: "option1", Value: "Option 1", Color: "#ff0000"},
// 						{Key: "option2", Value: "Option 2", Color: "#00ff00"},
// 					},
// 					Signature: "0x123signature",
// 				}

// 				// Make the request
// 				w := makeRequest("POST", "/governance/proposal", reqBody)

// 				// Assert response (should be bad request due to description length)
// 				Expect(w.Code).To(Equal(http.StatusBadRequest))

// 				// Parse response
// 				var response map[string]interface{}
// 				err := json.Unmarshal(w.Body.Bytes(), &response)
// 				Expect(err).To(BeNil())

// 				// Verify error message mentions description length
// 				Expect(response["message"].(string)).To(ContainSubstring("Description exceeds maximum length"))
// 			})

// 			It("should validate option length", func() {
// 				// Mock getCurrentSubaccount
// 				patches.ApplyFunc(middleware.GetCurrentSubaccount, func(ctx *gin.Context) (*db.SubaccountTable, error) {
// 					return &db.SubaccountTable{
// 						ID:       "1_0x123456_1",
// 						UserName: new(string),
// 					}, nil
// 				})

// 				// Create overly long option value (over 100 chars)
// 				longOptionValue := ""
// 				for i := 0; i < 150; i++ {
// 					longOptionValue += "a"
// 				}

// 				// Setup request with long option value
// 				reqBody := controller.ProposalRequest{
// 					Name:        "Test Proposal",
// 					VotingStart: time.Now().Add(1 * time.Hour).Unix(),
// 					VotingEnd:   time.Now().Add(24 * time.Hour).Unix(),
// 					Description: "This is a test proposal",
// 					Options: []controller.ProposalOption{
// 						{Key: "option1", Value: longOptionValue, Color: "#ff0000"},
// 						{Key: "option2", Value: "Option 2", Color: "#00ff00"},
// 					},
// 					Signature: "0x123signature",
// 				}

// 				// Make the request
// 				w := makeRequest("POST", "/governance/proposal", reqBody)

// 				// Assert response (should be bad request due to option length)
// 				Expect(w.Code).To(Equal(http.StatusBadRequest))

// 				// Parse response
// 				var response map[string]interface{}
// 				err := json.Unmarshal(w.Body.Bytes(), &response)
// 				Expect(err).To(BeNil())

// 				// Verify error message mentions option length
// 				Expect(response["message"].(string)).To(ContainSubstring("Option 1 value exceeds maximum length"))
// 			})
// 		})

// 		Context("with invalid input", func() {
// 			It("should reject a proposal with past voting start time", func() {
// 				// Mock getCurrentSubaccount
// 				patches.ApplyFunc(middleware.GetCurrentSubaccount, func(ctx *gin.Context) (*db.SubaccountTable, error) {
// 					return &db.SubaccountTable{
// 						ID:       "1_0x123456_1",
// 						UserName: new(string),
// 					}, nil
// 				})

// 				// Setup request with past voting start time
// 				reqBody := controller.ProposalRequest{
// 					Name:        "Test Proposal",
// 					VotingStart: time.Now().Add(-1 * time.Hour).Unix(), // Past time
// 					VotingEnd:   time.Now().Add(24 * time.Hour).Unix(),
// 					Description: "This is a test proposal",
// 					Options: []controller.ProposalOption{
// 						{Key: "option1", Value: "Option 1", Color: "#ff0000"},
// 						{Key: "option2", Value: "Option 2", Color: "#00ff00"},
// 					},
// 					Signature: "0x123signature",
// 				}

// 				// Make the request
// 				w := makeRequest("POST", "/governance/proposal", reqBody)

// 				// Assert response
// 				Expect(w.Code).To(Equal(http.StatusBadRequest))

// 				// Parse response
// 				var response map[string]interface{}
// 				err := json.Unmarshal(w.Body.Bytes(), &response)
// 				Expect(err).To(BeNil())

// 				// Verify error message
// 				Expect(response["message"].(string)).To(ContainSubstring("Voting start time must be in the future"))
// 			})

// 			It("should reject a proposal with invalid voting period", func() {
// 				// Mock getCurrentSubaccount
// 				patches.ApplyFunc(middleware.GetCurrentSubaccount, func(ctx *gin.Context) (*db.SubaccountTable, error) {
// 					return &db.SubaccountTable{
// 						ID:       "1_0x123456_1",
// 						UserName: new(string),
// 					}, nil
// 				})

// 				// Setup request with end time before start time
// 				reqBody := controller.ProposalRequest{
// 					Name:        "Test Proposal",
// 					VotingStart: time.Now().Add(2 * time.Hour).Unix(),
// 					VotingEnd:   time.Now().Add(1 * time.Hour).Unix(), // Before start time
// 					Description: "This is a test proposal",
// 					Options: []controller.ProposalOption{
// 						{Key: "option1", Value: "Option 1", Color: "#ff0000"},
// 						{Key: "option2", Value: "Option 2", Color: "#00ff00"},
// 					},
// 					Signature: "0x123signature",
// 				}

// 				// Make the request
// 				w := makeRequest("POST", "/governance/proposal", reqBody)

// 				// Assert response
// 				Expect(w.Code).To(Equal(http.StatusBadRequest))

// 				// Parse response
// 				var response map[string]interface{}
// 				err := json.Unmarshal(w.Body.Bytes(), &response)
// 				Expect(err).To(BeNil())

// 				// Verify error message
// 				Expect(response["message"].(string)).To(ContainSubstring("Voting end time must be after voting start time"))
// 			})
// 		})
// 	})

// Describe("GetAllProposals", func() {
// 	It("should return all proposals", func() {
// 		// Prepare mock response - Fix the return type to be explicit
// 		proposals := &[]db.ProposalTable{
// 			{
// 				BaseTable: db.BaseTable{
// 					ID: 1,
// 				},
// 				Name:                 "Proposal 1",
// 				VotingStart:          time.Now().Add(1 * time.Hour).Unix(),
// 				VotingEnd:            time.Now().Add(24 * time.Hour).Unix(),
// 				Description:          "Proposal 1 description",
// 				Options:              `[{"key":"option1","value":"Option 1","color":"#ff0000"},{"key":"option2","value":"Option 2","color":"#00ff00"}]`,
// 				ProposerSubaccountId: "1_0x123456_1",
// 			},
// 			{
// 				BaseTable: db.BaseTable{
// 					ID: 2,
// 				},
// 				Name:                 "Proposal 2",
// 				VotingStart:          time.Now().Add(2 * time.Hour).Unix(),
// 				VotingEnd:            time.Now().Add(48 * time.Hour).Unix(),
// 				Description:          "Proposal 2 description",
// 				Options:              `[{"key":"option1","value":"Option 1","color":"#ff0000"},{"key":"option2","value":"Option 2","color":"#00ff00"}]`,
// 				ProposerSubaccountId: "1_0x123456_1",
// 			},
// 		}

// 		// Be explicit about the return type to match the interface
// 		mockProposalDB.On("GetAll").Return(proposals).Once()

// 		// Make the request
// 		w := makeRequest("GET", "/governance/proposal/all", nil)

// 		// Assert response
// 		Expect(w.Code).To(Equal(http.StatusOK))

// 		// Parse response
// 		var response map[string]interface{}
// 		err := json.Unmarshal(w.Body.Bytes(), &response)
// 		Expect(err).To(BeNil())

// 		// Verify response data
// 		Expect(response["status"]).To(Equal(float64(http.StatusOK)))

// 		// Add error checking before type assertion
// 		Expect(response["data"]).NotTo(BeNil(), "Response data should not be nil")
// 		proposalsData, ok := response["data"].([]interface{})
// 		Expect(ok).To(BeTrue(), "Could not convert response data to []interface{}")
// 		Expect(len(proposalsData)).To(Equal(2))

// 		// Verify mock was called
// 		mockProposalDB.AssertExpectations(GinkgoT())
// 	})

// 	It("should handle error when fetching proposals fails", func() {
// 		// Setup mock to return nil (error scenario)
// 		mockProposalDB.On("GetAll").Return(nil)

// 		// Make the request
// 		w := makeRequest("GET", "/governance/proposal/all", nil)

// 		// Assert response
// 		Expect(w.Code).To(Equal(http.StatusInternalServerError))

// 		// Parse response
// 		var response map[string]interface{}
// 		err := json.Unmarshal(w.Body.Bytes(), &response)
// 		Expect(err).To(BeNil())

// 		// Verify error message
// 		Expect(response["message"].(string)).To(ContainSubstring("Failed to fetch proposals"))

// 		// Verify mock was called
// 		mockProposalDB.AssertExpectations(GinkgoT())
// 	})
// })

// Describe("GetProposalById", func() {
// 	It("should return a specific proposal with user's vote status", func() {
// 		proposalId := uint(1)

// 		// Mock current subaccount - ensure it returns a concrete implementation
// 		patches.ApplyFunc(middleware.GetCurrentSubaccount, func(ctx *gin.Context) (*db.SubaccountTable, error) {
// 			return &db.SubaccountTable{
// 				ID:       "1_0x123456_1",
// 				UserName: new(string),
// 			}, nil
// 		})

// 		// Setup mock proposal
// 		proposal := &db.ProposalTable{
// 			BaseTable: db.BaseTable{
// 				ID: proposalId,
// 			},
// 			Name:                 "Test Proposal",
// 			VotingStart:          time.Now().Add(-1 * time.Hour).Unix(),
// 			VotingEnd:            time.Now().Add(24 * time.Hour).Unix(),
// 			Description:          "This is a test proposal",
// 			Options:              `[{"key":"option1","value":"Option 1","color":"#ff0000"},{"key":"option2","value":"Option 2","color":"#00ff00"}]`,
// 			ProposerSubaccountId: "1_0x123456_1",
// 		}

// 		// Setup mock user vote
// 		userVote := &db.VoteTable{
// 			BaseTable: db.BaseTable{
// 				ID: 1,
// 			},
// 			SubaccountId: "1_0x123456_1",
// 			ProposalId:   proposalId,
// 			Option:       "option1",
// 			VotingPower:  "1000000000000000000",
// 			Signature:    "0x123signature",
// 		}

// 		// Setup mock expectations with .Once() to make expectations more precise
// 		mockProposalDB.On("GetByID", proposalId).Return(proposal).Once()
// 		mockVoteDB.On("GetBySubaccountAndProposal", "1_0x123456_1", proposalId).Return(userVote).Once()

// 		// Make the request
// 		w := makeRequest("GET", fmt.Sprintf("/governance/proposal/%d", proposalId), nil)

// 		// Assert response
// 		Expect(w.Code).To(Equal(http.StatusOK))

// 		// Parse response
// 		var response map[string]interface{}
// 		err := json.Unmarshal(w.Body.Bytes(), &response)
// 		Expect(err).To(BeNil())

// 		// Add error checking before type assertion
// 		Expect(response["data"]).NotTo(BeNil(), "Response data should not be nil")
// 		data, ok := response["data"].(map[string]interface{})
// 		Expect(ok).To(BeTrue(), "Could not convert response data to map[string]interface{}")
// 		Expect(data["proposal"]).NotTo(BeNil())
// 		Expect(data["user_vote"]).NotTo(BeNil())

// 		// Verify mock was called
// 		mockProposalDB.AssertExpectations(GinkgoT())
// 		mockVoteDB.AssertExpectations(GinkgoT())
// 	})

// 	It("should handle case when proposal doesn't exist", func() {
// 		proposalId := uint(999)

// 		// Mock current subaccount
// 		patches.ApplyFunc(middleware.GetCurrentSubaccount, func(ctx *gin.Context) (*db.SubaccountTable, error) {
// 			return &db.SubaccountTable{
// 				ID:       "1_0x123456_1",
// 				UserName: new(string),
// 			}, nil
// 		})

// 		// Setup mock to return nil (proposal not found)
// 		mockProposalDB.On("GetByID", proposalId).Return(nil)

// 		// Make the request
// 		w := makeRequest("GET", fmt.Sprintf("/governance/proposal/%d", proposalId), nil)

// 		// Assert response
// 		Expect(w.Code).To(Equal(http.StatusNotFound))

// 		// Parse response
// 		var response map[string]interface{}
// 		err := json.Unmarshal(w.Body.Bytes(), &response)
// 		Expect(err).To(BeNil())

// 		// Verify error message
// 		Expect(response["message"].(string)).To(ContainSubstring("Proposal not found"))

// 		// Verify mock was called
// 		mockProposalDB.AssertExpectations(GinkgoT())
// 	})
// })

// Describe("CastVote", func() {
// 	Context("when voting conditions are valid", func() {
// 		It("should cast a vote successfully", func() {
// 			proposalId := uint(1)

// 			// Mock current subaccount
// 			patches.ApplyFunc(middleware.GetCurrentSubaccount, func(ctx *gin.Context) (*db.SubaccountTable, error) {
// 				return &db.SubaccountTable{
// 					ID:       "1_0x123456_1",
// 					UserName: new(string),
// 				}, nil
// 			})

// 			// Setup mock proposal (active voting period)
// 			proposal := &db.ProposalTable{
// 				BaseTable: db.BaseTable{
// 					ID: proposalId,
// 				},
// 				Name:                 "Test Proposal",
// 				VotingStart:          time.Now().Add(-1 * time.Hour).Unix(), // Started 1 hour ago
// 				VotingEnd:            time.Now().Add(24 * time.Hour).Unix(), // Ends in 24 hours
// 				Description:          "This is a test proposal",
// 				Options:              `[{"key":"option1","value":"Option 1","color":"#ff0000"},{"key":"option2","value":"Option 2","color":"#00ff00"}]`,
// 				ProposerSubaccountId: "1_0x123456_1",
// 			}

// 			// Setup mock expectations for checking no existing vote
// 			mockProposalDB.On("GetByID", proposalId).Return(proposal)
// 			mockVoteDB.On("GetBySubaccountAndProposal", "1_0x123456_1", proposalId).Return(nil)

// 			// Mock staking balance
// 			subaccountHex := "0xConverted123456"
// 			patches.ApplyFunc(cutils.SubaccountIdToHex, func(subaccountId string) (string, error) {
// 				return subaccountHex, nil
// 			})

// 			// Setup mock balance response (1000 tokens staked)
// 			balanceResponse := []ctypes.SpotBalance{
// 				{
// 					ProductId:       2,                        // Staked tokens product ID
// 					TokenBalance:    "1000000000000000000000", // 1000 tokens
// 					WithdrawBalance: "1000000000000000000000",
// 				},
// 			}
// 			mockBalanceClient.On("GetSpotBalance", subaccountHex).Return(balanceResponse, "0", "0", nil)

// 			// Setup request body
// 			reqBody := controller.VoteRequest{
// 				Option:    "option1",
// 				Signature: "0x123signature",
// 			}

// 			// Setup mock for vote creation
// 			expectedVote := &db.VoteTable{
// 				BaseTable: db.BaseTable{
// 					ID: 1,
// 				},
// 				SubaccountId: "1_0x123456_1",
// 				ProposalId:   proposalId,
// 				Option:       "option1",
// 				VotingPower:  "1000000000000000000", // 1 token (1000/1000)
// 				Signature:    "0x123signature",
// 			}
// 			mockVoteDB.On("Create", mock.AnythingOfType("*db.VoteTable")).Return(expectedVote)

// 			// Make the request
// 			w := makeRequest("POST", fmt.Sprintf("/governance/vote/proposal/%d", proposalId), reqBody)

// 			// Assert response
// 			Expect(w.Code).To(Equal(http.StatusOK))

// 			// Parse response
// 			var response map[string]interface{}
// 			err := json.Unmarshal(w.Body.Bytes(), &response)
// 			Expect(err).To(BeNil())

// 			// Verify response data
// 			Expect(response["status"]).To(Equal(float64(http.StatusOK)))

// 			// Verify mocks were called
// 			mockProposalDB.AssertExpectations(GinkgoT())
// 			mockVoteDB.AssertExpectations(GinkgoT())
// 			mockBalanceClient.AssertExpectations(GinkgoT())
// 		})
// 	})

// 	Context("when voting conditions are invalid", func() {
// 		It("should reject when user has already voted", func() {
// 			proposalId := uint(1)

// 			// Mock current subaccount
// 			patches.ApplyFunc(middleware.GetCurrentSubaccount, func(ctx *gin.Context) (*db.SubaccountTable, error) {
// 				return &db.SubaccountTable{
// 					ID:       "1_0x123456_1",
// 					UserName: new(string),
// 				}, nil
// 			})

// 			// Setup mock proposal
// 			proposal := &db.ProposalTable{
// 				BaseTable: db.BaseTable{
// 					ID: proposalId,
// 				},
// 				Name:                 "Test Proposal",
// 				VotingStart:          time.Now().Add(-1 * time.Hour).Unix(),
// 				VotingEnd:            time.Now().Add(24 * time.Hour).Unix(),
// 				Description:          "This is a test proposal",
// 				Options:              `[{"key":"option1","value":"Option 1","color":"#ff0000"},{"key":"option2","value":"Option 2","color":"#00ff00"}]`,
// 				ProposerSubaccountId: "1_0x123456_1",
// 			}

// 			// Setup existing vote
// 			existingVote := &db.VoteTable{
// 				BaseTable: db.BaseTable{
// 					ID: 1,
// 				},
// 				SubaccountId: "1_0x123456_1",
// 				ProposalId:   proposalId,
// 				Option:       "option1",
// 				VotingPower:  "1000",
// 				Signature:    "0x123signature",
// 			}

// 			// Setup mock expectations with .Once() for stricter verification
// 			mockProposalDB.On("GetByID", proposalId).Return(proposal).Once()
// 			mockVoteDB.On("GetBySubaccountAndProposal", "1_0x123456_1", proposalId).Return(existingVote).Once()

// 			// Setup request body
// 			reqBody := controller.VoteRequest{
// 				Option:    "option2", // Different option from existing vote
// 				Signature: "0x456signature",
// 			}

// 			// Make the request
// 			w := makeRequest("POST", fmt.Sprintf("/governance/vote/proposal/%d", proposalId), reqBody)

// 			// Assert response
// 			Expect(w.Code).To(Equal(http.StatusBadRequest))

// 			// Parse response
// 			var response map[string]interface{}
// 			err := json.Unmarshal(w.Body.Bytes(), &response)
// 			Expect(err).To(BeNil())

// 			// Verify error message
// 			Expect(response["message"].(string)).To(ContainSubstring("You have already voted for this proposal"))

// 			// Verify mocks were called
// 			mockProposalDB.AssertExpectations(GinkgoT())
// 			mockVoteDB.AssertExpectations(GinkgoT())
// 		})

// 		It("should reject voting with invalid option", func() {
// 			proposalId := uint(1)

// 			// Mock current subaccount
// 			patches.ApplyFunc(middleware.GetCurrentSubaccount, func(ctx *gin.Context) (*db.SubaccountTable, error) {
// 				return &db.SubaccountTable{
// 					ID:       "1_0x123456_1",
// 					UserName: new(string),
// 				}, nil
// 			})

// 			// Setup mock proposal
// 			proposal := &db.ProposalTable{
// 				BaseTable: db.BaseTable{
// 					ID: proposalId,
// 				},
// 				Name:                 "Test Proposal",
// 				VotingStart:          time.Now().Add(-1 * time.Hour).Unix(),
// 				VotingEnd:            time.Now().Add(24 * time.Hour).Unix(),
// 				Description:          "This is a test proposal",
// 				Options:              `[{"key":"option1","value":"Option 1","color":"#ff0000"},{"key":"option2","value":"Option 2","color":"#00ff00"}]`,
// 				ProposerSubaccountId: "1_0x123456_1",
// 			}

// 			// Setup mock expectations
// 			mockProposalDB.On("GetByID", proposalId).Return(proposal)
// 			mockVoteDB.On("GetBySubaccountAndProposal", "1_0x123456_1", proposalId).Return(nil)

// 			// Setup request body with invalid option
// 			reqBody := controller.VoteRequest{
// 				Option:    "option3", // Option not in the proposal options
// 				Signature: "0x123signature",
// 			}

// 			// Make the request
// 			w := makeRequest("POST", fmt.Sprintf("/governance/vote/proposal/%d", proposalId), reqBody)

// 			// Assert response
// 			Expect(w.Code).To(Equal(http.StatusBadRequest))

// 			// Verify error message about invalid option
// 			var response map[string]interface{}
// 			err := json.Unmarshal(w.Body.Bytes(), &response)
// 			Expect(err).To(BeNil())
// 			Expect(response["message"].(string)).To(ContainSubstring("Invalid option"))

// 			// Verify mocks were called
// 			mockProposalDB.AssertExpectations(GinkgoT())
// 			mockVoteDB.AssertExpectations(GinkgoT())
// 		})
// 	})
// })
// })
