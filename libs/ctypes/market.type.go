package ctypes

type MarketType string

const (
	PERPETUAL MarketType = "PERPETUAL"
	SPOT      MarketType = "SPOT"
)

type FundingRates struct {
	FundingRate           int64  `json:"fundingRate"`
	CumulativeFundingRate string `json:"cumulativeFundingRate"`
}
type SymbolFundingRate struct {
	Symbol  string
	Funding FundingRates
}
