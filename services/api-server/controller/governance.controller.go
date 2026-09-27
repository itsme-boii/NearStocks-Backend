package controller

import (
	"encoding/json"
	"fmt"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"github/eugenix-io/logx-inf-backend/services/api-server/middleware"
	"github/eugenix-io/logx-inf-backend/xclient"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/microcosm-cc/bluemonday"
)

type GovernanceController struct {
	balanceClient *xclient.BalanceClient
	proposalDB    *db.ProposalDB
	voteDB        *db.VoteDB
	stakingDB     *db.StakingDB
}
type ProposalWithVoteCount struct {
	*db.ProposalTable
	VoteCount int `json:"vote_count"`
}
type VoteResult struct {
	Vote    *db.VoteTable
	Status  int
	Message string
	Err     error
}
type TruncatedVote struct {
	SubaccountId string `json:"subaccount_id"`
	ProposalId   uint   `json:"proposal_id"`
	Option       string `json:"option"`
	VotingPower  string `json:"voting_power"`
}
type VotedResponse struct {
	SubaccountId string `json:"subaccount_id"`
	ProposalId   uint   `json:"proposal_id"`
	Option       string `json:"option"`
	VotingPower  string `json:"voting_power"`
}

func RegisterGovernanceController(r *gin.RouterGroup) {
	governanceController := GovernanceController{
		balanceClient: xclient.GlobalBalanceClient,
		proposalDB:    &db.ProposalDB{},
		voteDB:        &db.VoteDB{},
		stakingDB:     &db.StakingDB{},
	}

	rg := r.Group("/governance")

	// Proposal routes
	proposalGroup := rg.Group("/proposal")
	proposalGroup.GET("/all", governanceController.GetAllProposals)
	proposalGroup.GET("/:id", governanceController.GetProposalById)
	proposalGroup.POST("", middleware.RequireAuth, governanceController.CreateProposal)
	proposalGroup.GET("/by-name", governanceController.GetProposalByName)

	// Vote routes
	voteGroup := rg.Group("/vote")
	voteGroup.POST("/proposal/:id", middleware.RequireAuth, governanceController.CastVote)
	voteGroup.GET("/user", middleware.RequireAuth, governanceController.GetUserVotes)
	voteGroup.GET("/proposal/:id", governanceController.GetProposalVotes)
}

type ProposalOption struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Color string `json:"color"`
}

type ProposalRequest struct {
	Name           string           `json:"name" binding:"required"`
	VotingStart    int64            `json:"voting_start" binding:"required"`
	VotingEnd      int64            `json:"voting_end" binding:"required"`
	Description    string           `json:"description" binding:"required"`
	Options        []ProposalOption `json:"options" binding:"required,min=2"`
	Signature      string           `json:"signature" binding:"required"`
	MinVotingPower string           `json:"min_voting_power"`
}

func (gc *GovernanceController) CreateProposal(ctx *gin.Context) {
	var request ProposalRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		cutils.ApiAbort(ctx, http.StatusBadRequest, err.Error())
		return
	}
	currentSubaccount, err := getCurrentSubaccount(ctx)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "Failed to get current subaccount")
		return
	}

	// Check if user is authorized to create a proposal
	isAuthorized, err := gc.isAuthorizedProposer(currentSubaccount.ID)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "Failed to verify authorization status")
		return
	}
	if !isAuthorized {
		cutils.ApiAbort(ctx, http.StatusForbidden, "You need to have minimum required staking amount or be in the allowed proposers list")
		return
	}

	// Validate timestamps
	if cutils.IsTimestampExceeding(uint64(request.VotingStart), 0) {
		cutils.ApiAbort(ctx, http.StatusBadRequest, "Voting start time must be in the future")
		return
	}
	if uint64(request.VotingEnd) <= uint64(request.VotingStart) {
		cutils.ApiAbort(ctx, http.StatusBadRequest, "Voting end time must be after voting start time")
		return
	}

	// Limit description length (e.g., 5000 characters)
	const maxDescriptionLength = 5000
	if len(request.Description) > maxDescriptionLength {
		cutils.ApiAbort(ctx, http.StatusBadRequest, fmt.Sprintf("Description exceeds maximum length of %d characters", maxDescriptionLength))
		return
	}

	// Validate and limit option value lengths
	const maxOptionValueLength = 100
	for i, option := range request.Options {
		if len(option.Value) > maxOptionValueLength {
			cutils.ApiAbort(ctx, http.StatusBadRequest, fmt.Sprintf("Option %d value exceeds maximum length of %d characters", i+1, maxOptionValueLength))
			return
		}
	}

	// Verify signature

	// Convert options to JSON string
	optionsJSON, err := json.Marshal(request.Options)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "Failed to process options")
		return
	}

	// Sanitize HTML to prevent XSS attacks
	p := bluemonday.UGCPolicy()
	// Allow basic HTML
	p.AllowElements("b", "i", "u", "ul", "ol", "li", "p", "br", "h1", "h2", "h3", "h4", "h5", "h6")

	sanitizedDescription := p.Sanitize(request.Description)

	// Create proposal
	proposal := &db.ProposalTable{
		Name:                 request.Name,
		VotingStart:          request.VotingStart,
		VotingEnd:            request.VotingEnd,
		Description:          sanitizedDescription,
		Options:              string(optionsJSON),
		ProposerSubaccountId: currentSubaccount.ID,
		MinVotingPower:       request.MinVotingPower,
		VotingIsActive:       false,
	}

	result := gc.proposalDB.Create(proposal)
	if result == nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "Failed to create proposal")
		return
	}

	cutils.ApiSuccess(ctx, result, "Proposal created successfully")
}

func (gc *GovernanceController) GetAllProposals(ctx *gin.Context) {
	proposals := gc.proposalDB.GetAll()
	if proposals == nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "Failed to fetch proposals")
		return
	}

	// Get vote counts for all proposals in a single query
	voteCounts, err := gc.voteDB.GetVoteCountsByProposals()
	if err != nil {
		xlog.Errorf("Error fetching vote counts: %v", err)
		voteCounts = make(map[uint]int) // Use empty map if there's an error
	}

	result := make([]ProposalWithVoteCount, 0, len(*proposals))
	for _, proposal := range *proposals {
		count := voteCounts[proposal.ID]

		result = append(result, ProposalWithVoteCount{
			ProposalTable: &proposal,
			VoteCount:     count,
		})
	}

	cutils.ApiSuccess(ctx, result, "Proposals fetched successfully")
}

func (gc *GovernanceController) GetProposalById(ctx *gin.Context) {
	idStr := ctx.Param("id")
	proposalId, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusBadRequest, "Invalid proposal ID")
		return
	}

	proposal := gc.proposalDB.GetByID(uint(proposalId))
	if proposal == nil {
		cutils.ApiAbort(ctx, http.StatusNotFound, "Proposal not found")
		return
	}

	// Get vote distribution and counts
	voteDistribution, voteCounts, totalVotingPower, err := gc.voteDB.GetVoteDistributionByProposal(uint(proposalId))

	if err != nil {
		xlog.Errorf("Error fetching vote distribution: %v", err)
		voteDistribution = make(map[string]*big.Float)
		voteCounts = make(map[string]int)
		totalVotingPower = new(big.Float).SetInt64(0)
	}

	var options []ProposalOption
	if err := json.Unmarshal([]byte(proposal.Options), &options); err == nil {
		for _, opt := range options {
			if _, exists := voteDistribution[opt.Key]; !exists {
				voteDistribution[opt.Key] = new(big.Float).SetInt64(0)
			}
			// Initialize vote count for options with no votes
			if _, exists := voteCounts[opt.Key]; !exists {
				voteCounts[opt.Key] = 0
			}
		}
	}

	// Format results to avoid scientific notation
	formattedDistribution := make(map[string]string)
	for option, value := range voteDistribution {
		formattedDistribution[option] = value.String()
	}

	subaccountId := ctx.Query("subaccount_id")
	var userVote *db.VoteTable

	if subaccountId != "" {
		userVote = gc.voteDB.GetBySubaccountAndProposal(subaccountId, uint(proposalId))
	}

	response := gin.H{
		"proposal":           proposal,
		"user_vote":          userVote,
		"vote_distribution":  formattedDistribution,
		"vote_counts":        voteCounts,
		"total_voting_power": totalVotingPower,
	}

	cutils.ApiSuccess(ctx, response, "Proposal details fetched successfully")
}

type VoteRequest struct {
	Option    string `json:"option" binding:"required"`
	Signature string `json:"signature" binding:"required"`
}

func (gc *GovernanceController) CastVote(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusBadRequest, "Invalid proposal ID")
		return
	}

	var request VoteRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		cutils.ApiAbort(ctx, http.StatusBadRequest, err.Error())
		return
	}

	currentSubaccount, err := getCurrentSubaccount(ctx)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "Failed to get current subaccount")
		return
	}

	lockKey := xredis.GetGovernanceVoteLockKey(currentSubaccount.ID, uint(id))

	result, err := xredis.WithRedisLock(lockKey, func() (*VoteResult, error) {

		existingVote := gc.voteDB.GetBySubaccountAndProposal(currentSubaccount.ID, uint(id))
		if existingVote != nil {
			return &VoteResult{Status: http.StatusBadRequest, Message: "You have already voted for this proposal"}, nil
		}

		proposal := gc.proposalDB.GetByID(uint(id))
		if proposal == nil {
			return &VoteResult{Status: http.StatusNotFound, Message: "Proposal not found"}, nil
		}

		// Check if voting period has started
		if cutils.IsTimestampExceeding(uint64(proposal.VotingStart), 0) {
			return &VoteResult{Status: http.StatusBadRequest, Message: "Voting has not started yet"}, nil
		}

		// Check if voting period has ended
		if cutils.IsTimestampExpired(uint64(proposal.VotingEnd)) {
			return &VoteResult{Status: http.StatusBadRequest, Message: "Voting has ended"}, nil
		}

		// If we're within the voting period but status is not active, update it
		gc.proposalDB.UpdateVotingActiveStatus(proposal.ID, true)

		// Validate option
		var options []ProposalOption
		if err := json.Unmarshal([]byte(proposal.Options), &options); err != nil {
			return &VoteResult{Status: http.StatusInternalServerError, Message: "Invalid proposal options format"}, nil
		}

		validOption := false
		for _, opt := range options {
			if opt.Key == request.Option {
				validOption = true
				break
			}
		}
		if !validOption {
			return &VoteResult{Status: http.StatusBadRequest, Message: "Invalid option selected"}, nil
		}

		stakingAmountx18, err := gc.getStakingInfo(currentSubaccount.ID)
		if err != nil {
			return &VoteResult{Status: http.StatusInternalServerError, Message: "Failed to get staking information", Err: err}, nil
		}

		// Calculate voting power (staked tokens / 1000)
		stakingAmountInt, ok := new(big.Int).SetString(stakingAmountx18, 10)
		if !ok {
			return &VoteResult{Status: http.StatusInternalServerError, Message: "Invalid staking amount"}, nil
		}

		minRequiredStake := cutils.FloatStrToX18(os.Getenv("MINIMUM_STAKING_AMOUNT"))
		if stakingAmountInt.Cmp(minRequiredStake) < 0 {
			xlog.Infof("Minimum 1000 staked tokens required to vote for subaccount: %s, staking amount: %v", currentSubaccount.ID, stakingAmountInt)
			return &VoteResult{Status: http.StatusForbidden, Message: "Minimum 1000 staked tokens required to vote"}, nil
		}

		// For voting power calculation, divide by 1000
		divisor := new(big.Int).SetInt64(1000)
		votingPowerInt := new(big.Int).Div(stakingAmountInt, divisor)
		votingPower := votingPowerInt.String()

		// Create vote
		vote := &db.VoteTable{
			SubaccountId: currentSubaccount.ID,
			ProposalId:   uint(id),
			Option:       request.Option,
			VotingPower:  votingPower,
			Signature:    request.Signature,
		}
		xlog.Infof("Vote Casted: %+v", vote)

		createdVote := gc.voteDB.Create(vote)
		if createdVote == nil {
			return &VoteResult{Status: http.StatusInternalServerError, Message: "Failed to cast vote"}, nil
		}

		return &VoteResult{Vote: createdVote}, nil
	})

	// Handle any errors that occurred within the lock itself
	if err != nil {
		xlog.Errorf("Failed to acquire or execute with vote lock: %v", err)
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "Failed to process vote request")
		return
	}

	// Handle application errors
	if result.Status != 0 {
		cutils.ApiAbort(ctx, result.Status, result.Message)
		return
	}

	cutils.ApiSuccess(ctx, result.Vote, "Vote cast successfully")
}

// Helper function to get staking info and calculate voting power
func (gc *GovernanceController) getStakingInfo(subaccountId string) (string, error) {
	subaccountHex, err := cutils.SubaccountIdToHex(subaccountId)
	if err != nil {
		xlog.Errorf("Error converting subaccount ID to hex: %v", err)
		return "0", err
	}

	// Fetch spot balance from balance client
	spotBalance, _, _, err := gc.balanceClient.GetSpotBalance(subaccountHex)
	if err != nil {
		xlog.Errorf("Error fetching spot balance: %v", err)
		return "0", err
	}
	// Find the staked amount (productId 2)
	stakedAmount := "0"
	for _, balance := range spotBalance {
		if balance.ProductId == 2 {
			stakedAmount = balance.TokenBalance
			break
		}
	}

	return stakedAmount, nil
}

// GetUserVotes retrieves all votes cast by the authenticated user
func (gc *GovernanceController) GetUserVotes(ctx *gin.Context) {
	// Get current authenticated subaccount
	currentSubaccount, err := getCurrentSubaccount(ctx)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "Failed to get current subaccount")
		return
	}

	// Get all votes by this subaccount
	votes := gc.voteDB.GetVotesBySubaccount(currentSubaccount.ID)
	if votes == nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "Failed to fetch votes")
		return
	}

	// Calculate user's current voting power (for comparison or future votes)
	stakingAmountx18, err := gc.getStakingInfo(currentSubaccount.ID)
	if err != nil {
		xlog.Errorf("Error fetching staking info: %v", err)
		stakingAmountx18 = "0"
	}

	// Calculate current potential voting power
	stakingAmountInt, ok := new(big.Int).SetString(stakingAmountx18, 10)
	if !ok {
		stakingAmountInt = big.NewInt(0)
	}

	// Voting power is the staking amount divided by 1000
	currentVotingPower := new(big.Int).Div(stakingAmountInt, big.NewInt(1000))
	currentVotingPowerStr := currentVotingPower.String()

	// Format vote responses with voting power
	result := make([]VotedResponse, 0, len(*votes))
	for _, vote := range *votes {
		result = append(result, VotedResponse{
			SubaccountId: vote.SubaccountId,
			ProposalId:   vote.ProposalId,
			Option:       vote.Option,
			VotingPower:  vote.VotingPower,
		})
	}

	response := gin.H{
		"votes":                result,
		"current_voting_power": currentVotingPowerStr,
	}

	cutils.ApiSuccess(ctx, response, "User votes fetched successfully")
}

func (gc *GovernanceController) isAuthorizedProposer(subaccountId string) (bool, error) {
	// Check if user is in the allowed proposers list
	allowedProposers := strings.Split(os.Getenv("ALLOWED_PROPOSERS"), ",")
	for _, id := range allowedProposers {
		if strings.TrimSpace(id) == subaccountId {
			return true, nil
		}
	}

	// Check minimum staking requirement to become proposer
	stakingAmount, err := gc.getStakingInfo(subaccountId)
	if err != nil {
		return false, err
	}

	requiredStake := os.Getenv("MINIMUM_PROPOSER_STAKING_AMOUNT")
	if requiredStake == "" {
		requiredStake = "10000"
	}

	stakeInt, _ := new(big.Int).SetString(stakingAmount, 10)
	minStake := cutils.FloatStrToX18(requiredStake)

	return stakeInt.Cmp(minStake) >= 0, nil
}

func (gc *GovernanceController) GetProposalVotes(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusBadRequest, "Invalid proposal ID")
		return
	}

	votes := gc.voteDB.GetVotesByProposal(uint(id))
	if votes == nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "Failed to fetch votes")
		return
	}

	truncatedVotes := make([]TruncatedVote, 0, len(*votes))
	for _, vote := range *votes {
		addressParts := strings.Split(vote.SubaccountId, "_")
		var hexAddress string
		if len(addressParts) >= 2 {
			hexAddress = addressParts[1]
		} else {
			hexAddress = vote.SubaccountId
		}

		hexAddress = hexAddress[:4] + "..." + hexAddress[len(hexAddress)-4:]

		truncatedVotes = append(truncatedVotes, TruncatedVote{
			SubaccountId: hexAddress,
			ProposalId:   vote.ProposalId,
			Option:       vote.Option,
			VotingPower:  vote.VotingPower,
		})
	}

	cutils.ApiSuccess(ctx, truncatedVotes, "Proposal votes fetched successfully")
}

func (gc *GovernanceController) GetProposalByName(ctx *gin.Context) {
	name := ctx.Query("name")
	if name == "" {
		cutils.ApiAbort(ctx, http.StatusBadRequest, "Proposal name is required")
		return
	}

	// First, we need to add a method to ProposalDB to find by name
	proposal := gc.proposalDB.GetByName(name)
	if proposal == nil {
		cutils.ApiAbort(ctx, http.StatusNotFound, "Proposal not found")
		return
	}


	voteDistribution, voteCounts, totalVotingPower, err := gc.voteDB.GetVoteDistributionByProposal(proposal.ID)
	if err != nil {
		xlog.Errorf("Error fetching vote distribution: %v", err)
		voteDistribution = make(map[string]*big.Float)
		voteCounts = make(map[string]int)
		totalVotingPower = new(big.Float).SetInt64(0)
	}

	var options []ProposalOption
	if err := json.Unmarshal([]byte(proposal.Options), &options); err == nil {
		for _, opt := range options {
			if _, exists := voteDistribution[opt.Key]; !exists {
				voteDistribution[opt.Key] = new(big.Float).SetInt64(0)
			}
			if _, exists := voteCounts[opt.Key]; !exists {
				voteCounts[opt.Key] = 0
			}
		}
	}

	formattedDistribution := make(map[string]string)
	for option, value := range voteDistribution {
		formattedDistribution[option] = value.String()
	}

	subaccountId := ctx.Query("subaccount_id")
	var userVote *db.VoteTable

	if subaccountId != "" {
		userVote = gc.voteDB.GetBySubaccountAndProposal(subaccountId, proposal.ID)
	}

	response := gin.H{
		"proposal":           proposal,
		"user_vote":          userVote,
		"vote_distribution":  formattedDistribution,
		"vote_counts":        voteCounts,
		"total_voting_power": totalVotingPower,
	}

	cutils.ApiSuccess(ctx, response, "Proposal details fetched successfully")
}
