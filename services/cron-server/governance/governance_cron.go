package governance

import (
	"encoding/json"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"math/big"
	"os"
	"time"

	"github/eugenix-io/logx-inf-backend/libs/cutils"

	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
)

type VotingResult struct {
	Results          map[string]*big.Float
	TotalVotingPower *big.Float
	FinalOption      string
	MeetsThreshold   bool
}

func RegisterGovernanceRoutes(router *gin.Engine) {
	governanceGroup := router.Group("/api/governance")

	governanceGroup.POST("/check-proposals", ManuallyCheckProposals)
}
func StartGovernanceCron() {
	xlog.Infof("Starting Governance Cron")

	c := cron.New()
	governanceCronInterval := os.Getenv("GOVERNANCE_CRON_INTERVAL")
	if governanceCronInterval == "" {
		governanceCronInterval = "1 * * * *" // Default: run at minute 1 of every hour
		xlog.Infof("Using default governance cron interval: %s", governanceCronInterval)
	} else {
		xlog.Infof("Using configured governance cron interval: %s", governanceCronInterval)
	}

	// Schedule the cron job with the configured interval
	_, err := c.AddFunc(governanceCronInterval, func() {
		checkAndUpdateProposals()
	})

	if err != nil {
		xlog.Errorf("Error setting up governance cron: %v", err)
		return
	}

	c.Start()
	xlog.Infof("Governance Cron started successfully")
}

// TallyVotes calculates voting results for a proposal
func TallyVotes(proposal *db.ProposalTable, proposalId uint) (*VotingResult, bool) {
	// Get all votes for the proposal
	voteDB := &db.VoteDB{}
	votes := voteDB.GetVotesByProposal(proposalId)
	if votes == nil || len(*votes) == 0 {
		return nil, false
	}

	// Parse options from the proposal
	var options []map[string]string
	if err := json.Unmarshal([]byte(proposal.Options), &options); err != nil {
		xlog.Errorf("Invalid proposal options format: %v", err)
		return nil, false
	}

	// Initialize results map for each option
	results := make(map[string]*big.Float)
	for _, opt := range options {
		if key, ok := opt["key"]; ok {
			results[key] = new(big.Float).SetInt64(0)
		}
	}

	// Calculate total voting power
	totalVotingPower := new(big.Float).SetInt64(0)

	// Tally votes
	for _, vote := range *votes {
		votePower, ok := new(big.Float).SetString(vote.VotingPower)
		if !ok {
			continue
		}

		if val, exists := results[vote.Option]; exists {
			results[vote.Option] = new(big.Float).Add(val, votePower)
		}

		totalVotingPower = new(big.Float).Add(totalVotingPower, votePower)
	}

	// Find the option with the maximum votes
	finalOption := ""
	maxVotes := new(big.Float).SetInt64(0)

	for option, votePower := range results {
		if finalOption == "" || votePower.Cmp(maxVotes) > 0 {
			finalOption = option
			maxVotes = votePower
		}
	}

	// Check if total voting power meets minimum threshold
	meetsThreshold := true
	// minVoting power is the quorum threshold
	minRequired, ok := new(big.Float).SetString(proposal.MinVotingPower)
	if ok && minRequired.Cmp(new(big.Float).SetInt64(0)) > 0 && totalVotingPower.Cmp(minRequired) < 0 {
		meetsThreshold = false
	}

	return &VotingResult{
		Results:          results,
		TotalVotingPower: totalVotingPower,
		FinalOption:      finalOption,
		MeetsThreshold:   meetsThreshold,
	}, true
}

func checkAndUpdateProposals() {
	xlog.Infof("Running governance proposal check")
	proposalDB := &db.ProposalDB{}

	// Get all active proposals
	activeProposals := proposalDB.GetActiveProposals()
	if activeProposals == nil {
		xlog.Errorf("Failed to fetch active proposals")
		return
	}

	currentTime := time.Now().Unix() * 1000

	for _, proposal := range *activeProposals {
		// Check if voting period has ended
		if uint64(currentTime) > uint64(proposal.VotingEnd) {
			xlog.Infof("Proposal %d voting has ended, updating status", proposal.ID)

			// Update active status
			proposalDB.UpdateVotingActiveStatus(proposal.ID, false)

			// Tally votes if not already done
			if proposal.VotingResult == "" {
				votingResult, success := TallyVotes(&proposal, proposal.ID)

				if success {
					// Update the voting result
					proposalDB.UpdateVotingResult(proposal.ID, &db.VotingResultData{
						Results:          votingResult.Results,
						TotalVotingPower: votingResult.TotalVotingPower,
						MeetsThreshold:   votingResult.MeetsThreshold,
						FinalOption:      votingResult.FinalOption,
					})

					xlog.Infof("Updated voting result for proposal %d, final option: %s, meets threshold: %v",
						proposal.ID, votingResult.FinalOption, votingResult.MeetsThreshold)
				} else {
					xlog.Errorf("Failed to tally votes for proposal %d", proposal.ID)
				}
			}
		}
	}
}

func ManuallyCheckProposals(ctx *gin.Context) {

	xlog.Infof("Manual trigger for governance proposal check")
	checkAndUpdateProposals()

	cutils.ApiSuccess(ctx, nil, "Proposal check and updates completed successfully")
}
