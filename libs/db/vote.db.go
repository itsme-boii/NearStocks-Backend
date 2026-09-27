package db

import (
	"github/eugenix-io/logx-inf-backend/libs/xlog"

	"math/big"

	"gorm.io/gorm"
)

type VoteDB struct{}

func (*VoteDB) Create(vote *VoteTable) *VoteTable {
	if err := db.Create(vote).Error; err != nil {
		xlog.Errorf("Error creating proposal: %v", err)
		return nil
	}
	return vote
}

func (*VoteDB) GetBySubaccountAndProposal(subaccountId string, proposalId uint) *VoteTable {
	var vote VoteTable
	if err := db.Where("subaccount_id = ? AND proposal_id = ?", subaccountId, proposalId).First(&vote).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil
		}
		xlog.Errorf("Error fetching vote: %v", err)
		return nil
	}
	return &vote
}

func (*VoteDB) GetVotesByProposal(proposalId uint) *[]VoteTable {
	var votes []VoteTable
	if err := db.Where("proposal_id = ?", proposalId).Order("created_at DESC").Find(&votes).Error; err != nil {
		xlog.Errorf("Error fetching votes for proposal: %v", err)
		return nil
	}
	return &votes
}

func (*VoteDB) UpdateVotingPower(id uint, buyingPower string) bool {
	if err := db.Model(&VoteTable{}).Where("id = ?", id).Update("voting_power", buyingPower).Error; err != nil {
		xlog.Errorf("Error updating buying power: %v", err)
		return false
	}
	return true
}

// GetVotesBySubaccount retrieves all votes cast by a specific subaccount
func (*VoteDB) GetVotesBySubaccount(subaccountId string) *[]VoteTable {
	var votes []VoteTable
	if err := db.Where("subaccount_id = ?", subaccountId).Find(&votes).Error; err != nil {
		xlog.Errorf("Error fetching votes for subaccount: %v", err)
		return nil
	}
	return &votes
}

// GetVoteCountsByProposals returns a map of proposal ID to vote count
func (*VoteDB) GetVoteCountsByProposals() (map[uint]int, error) {
	type Result struct {
		ProposalID uint
		Count      int
	}

	var results []Result

	// GROUP BY query to count votes per proposal
	if err := db.Model(&VoteTable{}).
		Select("proposal_id, count(*) as count").
		Group("proposal_id").
		Find(&results).Error; err != nil {
		return nil, err
	}

	// Convert to map for easier lookup
	countMap := make(map[uint]int)
	for _, r := range results {
		countMap[r.ProposalID] = r.Count
	}

	return countMap, nil
}

func (*VoteDB) GetVoteDistributionByProposal(proposalId uint) (map[string]*big.Float, map[string]int, *big.Float, error) {
	type Result struct {
		Option      string
		VotingPower string
		Count       int
	}

	var results []Result

	if err := db.Model(&VoteTable{}).
		Select("option, SUM(CAST(voting_power AS NUMERIC)) as voting_power, COUNT(*) as count").
		Where("proposal_id = ?", proposalId).
		Group("option").
		Find(&results).Error; err != nil {
		return nil, nil, nil, err
	}

	distribution := make(map[string]*big.Float)
	counts := make(map[string]int)
	totalVotingPower := new(big.Float).SetInt64(0)

	for _, r := range results {
		votePower, ok := new(big.Float).SetString(r.VotingPower)
		if !ok {
			continue
		}
		distribution[r.Option] = votePower
		counts[r.Option] = r.Count
		totalVotingPower = new(big.Float).Add(totalVotingPower, votePower)
	}

	return distribution, counts, totalVotingPower, nil
}
