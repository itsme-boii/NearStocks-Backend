package xredis

import (
	"encoding/json"
	"time"
)

type NOOP struct{}

type LockOptions struct {
	LockExpiry *time.Duration
	MaxRetries *int
	RetryDelay *time.Duration
}

// Sets the lock options to the new options
// If the new options are nil, the old options are retained
func (lo *LockOptions) Set(newOptions LockOptions) {
	if newOptions.LockExpiry != nil {
		lo.LockExpiry = newOptions.LockExpiry
	}
	if newOptions.MaxRetries != nil {
		lo.MaxRetries = newOptions.MaxRetries
	}
	if newOptions.RetryDelay != nil {
		lo.RetryDelay = newOptions.RetryDelay
	}
}

type FundingRateData struct {
	FundingRate      string `json:"fundingRate"`
	FundingTimestamp string `json:"fundingTimestamp"`
}

type CumulativeFundingRateData struct {
	CumulativeFundingRate      string `json:"cumulativeFundingRate"`
	CumulativeFundingTimestamp string `json:"cumulativeFundingTimestamp"`
}

type CumulativeEarningRateData struct {
	CumulativeEarningRate      string `json:"cumulativeEarningRate"`
	CumulativeEarningTimestamp int64  `json:"cumulativeEarningTimestamp"`
}

type EarningData struct {
	ApyEarningFactorx18 string `json:"apyEarningFactor" redis:"apyEarningFactor"` // 1e18 means 100% APY
}

type UnclaimedUserRewards struct {
	UnclaimedUserRewards string    `json:"unclaimed_user_rewards"`
	LastUpdatedTimestamp time.Time `json:"last_updated_timestamp"`
}

type AMMFundingCaps struct {
	MaxFundingCap int64 `json:"maxFundingCap"`
	MinFundingCap int64 `json:"minFundingCap"`
}

type AMMSize struct {
	Size float64 `json:"size"`
}

type AMMSlippage struct {
	Slippage float64 `json:"slippage"`
}

func (crfd CumulativeFundingRateData) MarshalBinary() ([]byte, error) {
	return json.Marshal(crfd)
}

func (crfd *CumulativeFundingRateData) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, crfd)
}

func (erd EarningData) MarshalBinary() ([]byte, error) {
	return json.Marshal(erd)
}

func (erd *EarningData) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, erd)
}
