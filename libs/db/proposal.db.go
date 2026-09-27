package db

import (
	"github/eugenix-io/logx-inf-backend/libs/xlog"

	"encoding/json"
	"math/big"

	"gorm.io/gorm"
)

type ProposalDB struct{}

// VotingResultData represents the outcome of a proposal vote
type VotingResultData struct {
	Results          map[string]*big.Float `json:"results"`
	TotalVotingPower *big.Float            `json:"total_voting_power"`
	MeetsThreshold   bool                  `json:"meets_threshold"`
	FinalOption      string                `json:"final_option"`
}

func (*ProposalDB) Create(proposal *ProposalTable) *ProposalTable {
	if err := db.Create(proposal).Error; err != nil {
		xlog.Errorf("Error creating proposal: %v", err)
		return nil
	}
	return proposal
}

func (*ProposalDB) GetAll() *[]ProposalTable {
	var proposals []ProposalTable
	if err := db.Order("created_at desc").Find(&proposals).Error; err != nil {
		xlog.Errorf("Error fetching all proposals: %v", err)
		return nil
	}
	return &proposals
}

func (*ProposalDB) GetByID(id uint) *ProposalTable {
	var proposal ProposalTable
	if err := db.First(&proposal, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil
		}
		xlog.Errorf("Error fetching proposal by ID: %v", err)
		return nil
	}
	return &proposal
}

// UpdateVotingResult updates the voting result with a structured data object
func (*ProposalDB) UpdateVotingResult(id uint, result *VotingResultData) bool {
	// Convert the result object to JSON
	resultJSON, err := json.Marshal(result)
	if err != nil {
		xlog.Errorf("Error serializing voting results: %v", err)
		return false
	}

	// Update the database with the JSON string
	if err := db.Model(&ProposalTable{}).Where("id = ?", id).Update("voting_result", string(resultJSON)).Error; err != nil {
		xlog.Errorf("Error updating voting result: %v", err)
		return false
	}
	return true
}

func (*ProposalDB) GetActiveProposals() *[]ProposalTable {
	var proposals []ProposalTable
	if err := db.Where("voting_is_active = ?", true).Find(&proposals).Error; err != nil {
		xlog.Errorf("Error fetching active proposals: %v", err)
		return nil
	}
	return &proposals
}

func (*ProposalDB) UpdateVotingActiveStatus(id uint, isActive bool) bool {
	if err := db.Model(&ProposalTable{}).Where("id = ?", id).Update("voting_is_active", isActive).Error; err != nil {
		xlog.Errorf("Error updating voting active status: %v", err)
		return false
	}
	return true
}

func (*ProposalDB) GetByName(name string) *ProposalTable {
	var proposal ProposalTable
	if err := db.Where("name = ?", name).First(&proposal).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil
		}
		xlog.Errorf("Error fetching proposal by name: %v", err)
		return nil
	}
	return &proposal
}
