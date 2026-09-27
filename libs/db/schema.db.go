// TODO: Apply advanced field level permissions
// NOTE: OVERFLOW RISKS: uint64 is mapped with Postgress BigInt which can only support int64 so, we cannot save numbers bigger than 2^63 - 1

package db

import (
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"time"

	"gorm.io/gorm"
)

type BaseTable struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}

type AuthTable struct {
	BaseTable
	Key          string `gorm:"uniqueIndex"`
	SecretHash   string
	SubaccountId string `gorm:"index"` // SubaccountTable.ID
	ExpiryTs     uint64
}

type SubaccountTable struct {
	BaseTable
	GID               uint       `gorm:"primaryKey"`
	BrokerId          uint       `gorm:"index:idx_broker_eth,priority:1"`
	EthAddress        string     `gorm:"index:idx_broker_eth,priority:2"`
	ID                string     `gorm:"uniqueIndex"`                  // This is will used on FE and Smart contract
	UserName          *string    `gorm:"uniqueIndex" json:"user_name"` // Username can be null initially
	UsernameUpdatedAt *time.Time `gorm:"index" json:"username_updated_at"`
}

type BrokerTable struct {
	BaseTable
	Name string
}

type SigningKeyTable struct {
	BaseTable
	BrokerId     uint   `gorm:"index;default:1" json:"broker_id"` // Default value set to 1 for the broker ID
	Address      string `gorm:"index"`
	SubaccountID string `gorm:"index"` // SubaccountTable.ID
	ExpiryTs     uint64
}

// TODO: AUDIT THIS
// Need an index (Type, PairId, IsActive)
// TODO: Add uniqueness constraints -> (type, name)
type MarketTable struct {
	BaseTable
	Symbol                       string            `gorm:"uniqueIndex:idx_type_sym,priority:2"` // "ETH-USD"
	Type                         ctypes.MarketType `gorm:"uniqueIndex:idx_type_sym,priority:1"` // SPOT, PERPETUAL, OPTION
	AmtToQtmConversionExpo       int32             // Exponent to convert amount to quantum
	PriceToQtmConversionExpo     int32             // Exponent to convert price to quantum
	MaxPositionValuex18          ctypes.BigInt     `gorm:"type:text"` // Maximum position value in quote asset x18
	MinAmountx18                 ctypes.BigInt     `gorm:"type:text"` // Minimum amount of base asset that can be traded (x18)
	BaseAsset                    string            // Eg: ETH
	QuoteAsset                   string            // Eg: USDC
	IsActive                     bool              // FIXME: Use this in flows
	MakerFeeFractionx18          ctypes.BigInt     `gorm:"type:text"` // Fraction of trade value that should be taken from makers
	TakerFeeFractionx18          ctypes.BigInt     `gorm:"type:text"` // Fraction of trade value that should be taken from takers
	InitialMarginFractionx18     ctypes.BigInt     `gorm:"type:text"` // Initial margin fraction required to open a position (x18 format)
	MaintenanceMarginFractionx18 ctypes.BigInt     `gorm:"type:text"` // Maintenance margin fraction required to keep a position open (x18 format)
	// Largest absolute AMM position (base amount, x18). NULL or 0 = no cap. Must equal the NEAR
	// contract's amm_max_position_x18 for the market. Pointer so existing rows scan as NULL.
	AmmMaxPositionx18 *ctypes.BigInt `gorm:"type:text"`
}

type OrderTable struct {
	BaseTable
	SubaccountId     string                  `gorm:"index;index:idx_sub_market_status_side,priority:1;index:idx_subaccount_status,priority:1" redis:"subaccount_id" json:"subaccount_id"`
	BrokerId         uint                    `gorm:"index" redis:"broker_id" json:"broker_id"`
	MarketId         uint                    `gorm:"index:idx_sub_market_status_side,priority:2" redis:"market_id" json:"market_id"`
	Side             ctypes.OrderSide        `gorm:"index:idx_sub_market_status_side,priority:4" redis:"side" json:"side"`
	Party            ctypes.Party            `redis:"party" json:"party"`
	Type             ctypes.OrderType        `redis:"type" json:"type"`
	TotalFilledx18   ctypes.BigInt           `gorm:"type:text" redis:"total_filled_x18" json:"total_filled_x18"`
	ExpiryTs         uint64                  `redis:"expiry_ts" json:"expiry_ts"`
	Status           ctypes.OrderStatus      `gorm:"index:idx_sub_market_status_side,priority:3;index:idx_subaccount_status,priority:2" redis:"status,string" json:"status"`
	Timestamp        uint64                  `redis:"timestamp" json:"timestamp"`
	IsReduce         bool                    `redis:"is_reduce" json:"is_reduce"`
	Signature        string                  `gorm:"index" redis:"signature" json:"signature"`
	SessionKey       string                  `redis:"session_key" json:"session_key"`
	Amountx18        ctypes.BigInt           `gorm:"type:text" redis:"amount_x18" json:"amount_x18"`
	Pricex18         ctypes.BigInt           `gorm:"type:text" redis:"price_x18" json:"price_x18"`
	TriggerPricex18  ctypes.BigInt           `gorm:"type:text" redis:"trigger_price_x18" json:"trigger_price_x18"`
	TriggerCondition ctypes.TriggerCondition `redis:"trigger_condition" json:"trigger_condition"`
}
type RewardTable struct {
	gorm.Model
	UserAddress    string `gorm:"type:varchar(100);uniqueIndex"` // NOTE: This is misnomer as it is not user address but subaccount id hex
	CompletedTasks []byte `gorm:"type:json"`
	DailyPoints    int    `gorm:"type:int"`
}

type PointsTable struct {
	BaseTable
	Address   string        `gorm:"index;not null" json:"address"`     // User's Ethereum address
	Week      uint          `gorm:"index;not null" json:"week"`        // Week number
	StartDate time.Time     `gorm:"not null" json:"start_date"`        // Start date and time of the week
	EndDate   time.Time     `gorm:"not null" json:"end_date"`          // End date and time of the week
	Volume    ctypes.BigInt `gorm:"type:text;default:0" json:"volume"` // Trading volume for the week
	Points    ctypes.BigInt `gorm:"type:text;default:0" json:"points"` // Points earned for the week (10^18 format)
}

type FillTable struct {
	BaseTable
	BrokerId       uint   `gorm:"index;default:1" json:"broker_id"`
	SubaccountId   string `gorm:"index:idx_subaccount_market,priority:1"`
	OrderId        uint   `gorm:"index"`
	MarketId       uint   `gorm:"index:idx_subaccount_market,priority:2"`
	Side           ctypes.OrderSide
	Liquidity      ctypes.Liquidity
	Type           ctypes.FillType
	Amountx18      ctypes.BigInt `gorm:"type:text"`
	Pricex18       ctypes.BigInt `gorm:"type:text"`
	Feex18         ctypes.BigInt `gorm:"type:text"`
	RealizedPnlx18 ctypes.BigInt `gorm:"type:text"`
	FundingFeesx18 ctypes.BigInt `gorm:"type:text"` // positive = user pays; negative = user receives
	IsReduce       bool          `gorm:"index"`
}

type UserFeeRewardsTable struct {
	BaseTable
	SubaccountId string        `gorm:"index"`               // Indexed for faster lookup --- 1_0x1238_1 format
	FeeBonusx18  ctypes.BigInt `gorm:"type:text;default:0"` // The fee bonus as a big integer
	Symbol       string        `gorm:"type:text"`           // Symbol, stored as text
	FillTableId  uint          `gorm:"index"`               // Foreign key linking to FillTable,
	IsClaimed    bool          `gorm:"default:false"`
}

type IpTable struct {
	BaseTable
	SubaccountID string `gorm:"index"`
	CountryCode  string
}
type FillOrderTable struct {
	BaseTable
	BrokerID     uint          `gorm:"index;default:1" json:"broker_id"` // Default value set to 1 for BrokerId
	UserAddress  string        `gorm:"index"`                            // Address of the user
	ProductID    uint          `gorm:"index"`                            // Product ID
	SubAccountID string        `gorm:"index"`                            // Subaccount ID
	PriceX18     ctypes.BigInt `gorm:"type:text"`                        // Price with 18 decimals
	Amount       ctypes.BigInt `gorm:"type:text"`                        // Amount of the order
	IsTaker      bool          // Whether the order is taking or making
	FeeAmount    ctypes.BigInt `gorm:"type:text"`                                           // Amount paid in fees
	BaseDelta    ctypes.BigInt `gorm:"type:text"`                                           // Change in base balance
	QuoteDelta   ctypes.BigInt `gorm:"type:text"`                                           // Change in quote balance
	RealisedPnl  ctypes.BigInt `gorm:"type:text"`                                           // Realized profit and loss
	FundingFees  ctypes.BigInt `gorm:"type:text"`                                           // Funding fees
	BlockNumber  uint64        `json:"block_number"`                                        // Block number
	Signature    []byte        `gorm:"type:bytea"`                                          // Signature
	TxnHash      string        `gorm:"index:idx_txn_hash_index,priority:1" json:"txn_hash"` // Transaction hash
	Index        uint          `gorm:"index:idx_txn_hash_index,priority:2" json:"index"`    // Index
}

type ReferralUserTable struct {
	BaseTable
	ReferrerUserID string `gorm:"index;default:''" json:"referrer_user_id"` // Default value set to an empty string for referrerUserID
	UserAddress    string `gorm:"unique;index" json:"user_address"`         // Make UserAddress unique
	CollectedFees  int64  `gorm:"default:0" json:"collected_fees"`          // this column is of no use it was created before but can remove if needed
	ReferralCode   string `gorm:"index" json:"referral_code"`
	IsActive       bool   `gorm:"default:false" json:"is_active"`
}

type ReferralRewardTable struct {
	BaseTable
	UserAddress   string        `gorm:"index" json:"user_address"`
	ProductID     uint          `gorm:"index"`                                     // Product ID
	RewardAmount  ctypes.BigInt `gorm:"type:text;default:0" json:"reward_amount"`  // Default value for RewardAmount is 0
	ClaimedAmount ctypes.BigInt `gorm:"type:text;default:0" json:"claimed_amount"` // Default value for ClaimedAmount is 0
}

type ReferralHistoryTable struct {
	BaseTable
	RefereeUserID     string        `gorm:"index" json:"referee_user_id"`                // Eth Address of the referee (user who was referred)
	ReferrerUserID    string        `gorm:"index" json:"referrer_user_id"`               // Eth Address of the refereeer (user who was referring)
	RefereeFees       ctypes.BigInt `gorm:"type:text;default:0" json:"referee_fees"`     // Fees collected from the referee
	RefereeVolume     ctypes.BigInt `gorm:"type:text;default:0" json:"referee_volume"`   // Trading volume of the referee
	RefereeReward     ctypes.BigInt `gorm:"type:text;default:0" json:"referee_reward"`   // Reward assigned to the referee ---- in usd value
	ReferrerRewards   ctypes.BigInt `gorm:"type:text;default:0" json:"referrer_rewards"` // Reward assigned to the referrer ---- in usd value
	RefereeProductID  uint          `gorm:"index" json:"referee_product_id"`             // Product ID associated with the referee's activity
	ReferrerProductID uint          `gorm:"index" json:"referrer_product_id"`
}

type FundingRateTable struct {
	BaseTable
	MarketId         uint  `gorm:"index" redis:"market_id" json:"market_id"`
	FundingRate      int64 `json:"funding_rate"`
	FundingTimestamp int64 `json:"funding_timestamp"`
}

type PnlTable struct {
	BaseTable
	BrokerId      uint          `gorm:"index;default:1" json:"broker_id"` // Default value set to 1 for BrokerId
	ProductID     uint          `json:"product_id"`
	RealizedPnl   ctypes.BigInt `gorm:"type:text" json:"realized_pnl"`
	UnrealizedPnl string        `json:"unrealized_pnl"`
}

type BatchTable struct {
	BaseTable
	BrokerId           uint   `gorm:"index;default:1" json:"broker_id"` // Default value set to 1 for BrokerId
	SubAccountID1      string `gorm:"index;default:''"`
	SubAccountID2      string `gorm:"index;default:''"`
	Transaction        []byte `gorm:"type:bytea"`          // Postgres type for binary data
	Signature1         []byte `gorm:"type:bytea;not null"` // Signature1 is mandatory
	Signature2         []byte `gorm:"type:bytea"`          // Signature2 is optional
	NSubmissionIdx     uint   `gorm:"default:0"`
	FunctionName       string `gorm:"index;default:''"`
	TransactionCounter uint   `gorm:"default:0"`
	Balance1           string `gorm:"type:text;default:''"`
	Balance2           string `gorm:"type:text;default:''"`
	Nonce              string `gorm:"type:text;default:''"`
	// NEAR batcher state (Development.md §8.3): '' pending, "inflight" (NSubmissionIdx is the
	// planned contract index), "failed" (refused on-chain alone; its subaccount is paused).
	// Landed rows are soft-deleted, as before.
	NearState   string `gorm:"index;default:''"`
	NearTxHash  string `gorm:"default:''"`
	NearFailure string `gorm:"type:text;default:''"`
}

type StakingTable struct {
	BaseTable
	SubaccountId            string `gorm:"index" json:"subaccount_id"`
	Action                  string `json:"action"`
	Amount                  string `json:"amount"`
	CumulativeEarningsRate  string `json:"cumulative_earnings_rate"`
	Earnings                string `json:"earnings"`
	Offsetx18               string `json:"offsetx18"`
	TransientEarningsx18    string `json:"transient_earningsx18"`
	TotalClaimedEarningsx18 string `json:"total_claimed_earningsx18"`
}

type LiquidationTable struct {
	BaseTable
	SubaccountId string `gorm:"uniqueIndex" json:"subaccount_id"`
}

type DepositWithdrawTable struct {
	BaseTable
	TxnHash            *string `json:"txn_hash"`
	TxnIndex           *uint64 `json:"txn_index"`
	BlockNumber        *uint64 `json:"block_number"`
	MessageId          string  `gorm:"uniqueIndex" json:"message_id"`
	SubaccountId       string  `gorm:"index:idx_subaccount_deposit" json:"subaccount_id"`
	Amount             string  `json:"amount"`
	ProductId          uint32  `json:"product_id"`
	SourceChainID      uint64  `json:"source_chain_id"`
	DestinationChainID uint64  `json:"destination_chain_id"`
	Received           *bool   `gorm:"index:idx_final_recd;index:idx_received" json:"received"`
	Finalised          bool    `gorm:"index:idx_final_recd" json:"finalised"`
	Failed             bool    `gorm:"index" json:"failed"`
	IsDeposit          bool    `gorm:"index:idx_subaccount_deposit" json:"is_deposit"`
}

// composite unique index to ensure uniqueness across the combination of SubaccountId and SourceChainID uniq_subaccount_chain
// composite key idx_source_chain_date for optimizing querying based on the pattern source_chain_id = ? AND last_deposit_date < ?
type SubaccountLastDepositTable struct {
	BaseTable
	SubaccountId    string    `gorm:"index:idx_subaccount_chain,uniqueIndex:uniq_subaccount_chain" json:"subaccount_id"`    // Subaccount ID
	SourceChainID   uint64    `gorm:"index:idx_source_chain_date,uniqueIndex:uniq_subaccount_chain" json:"source_chain_id"` // Source chain ID
	LastDepositDate time.Time `gorm:"index:idx_source_chain_date" json:"last_deposit_date"`                                 // Date of the last deposit
}

type LogxTokenUserTable struct {
	BaseTable
	UserAddress string        `gorm:"unique;not null;index" json:"user_address"` // Make UserAddress unique and not null
	TotalAmt    ctypes.BigInt `gorm:"type:text;default:0" json:"claimable_amt"`  // Using ctypes.BigInt for claimable amount with default 0
	ClaimedAmt  ctypes.BigInt `gorm:"type:text;default:0" json:"claimed_amt"`    // Using ctypes.BigInt for claimed amount with default 0
	HasVesting  bool          `gorm:"default:false" json:"has_vesting"`          // Default to false for vesting
	IsBlocked   bool          `gorm:"default:false" json:"is_blocked"`           // Default to false for blocked status
	Reason      string        `gorm:"type:text;default:''" json:"reason"`        // Default to empty string for reason
}

type TokenVestingTable struct {
	BaseTable
	UserAddress          string        `gorm:"index;not null" json:"user_address"`           // UserAddress not unique, can appear in multiple rows
	TotalAmt             ctypes.BigInt `gorm:"type:text;default:0" json:"total_amt"`         // Default to 0 for total amount
	ClaimedAmt           ctypes.BigInt `gorm:"type:text;default:0" json:"claimed_amt"`       // Default to 0 for claimed amount
	AmtUnlockDay1        ctypes.BigInt `gorm:"type:text;default:0" json:"amt_unlock_day1"`   // Default to 0 for amount unlocked on day 1
	UnlockPercentageX100 ctypes.BigInt `gorm:"type:text;default:0" json:"unlock_percentage"` // Now using BigInt for unlock percentage
	Frequency            int64         `gorm:"default:0" json:"frequency"`                   // Default to 0 for frequency (in days)
	VestingType          string        `gorm:"type:text" json:"vesting_type"`                // Vesting Type
}

// LotteryFlowTable represents the table structure for storing lottery flow entries
type LotteryFlowTable struct {
	BaseTable
	SubaccountId string `gorm:"index"`           // 1_0x1238_1 format
	LotteryCode  string `gorm:"type:varchar(4)"` // 4-character alphanumeric lottery code
}

type OptionsTable struct {
	BaseTable
	SubaccountId string         `gorm:"index" json:"subaccount_id"`
	ProductId    uint32         `gorm:"index" json:"product_id"`
	Amount       ctypes.BigInt  `gorm:"type:text" json:"amount"`
	Interval     uint32         `gorm:"index" json:"interval"`
	EntryPrice   ctypes.BigInt  `gorm:"type:text" json:"entry_price"`
	EntryTime    int64          `json:"entry_time"`
	Payout       uint32         `json:"payout"`
	Fees         uint32         `json:"fees"`
	ExitPrice    *ctypes.BigInt `gorm:"type:text;index" json:"exit_price"`
	PayoutAmount *ctypes.BigInt `gorm:"type:text" json:"payout_amount"`
	FeesAmount   *ctypes.BigInt `gorm:"type:text" json:"fees_amount"`
	UserPnl      *ctypes.BigInt `gorm:"type:text" json:"user_pnl"`
	QuoteDelta   *ctypes.BigInt `gorm:"type:text" json:"quote_delta"`
}

type PreMarketTable struct {
	BaseTable
	ProductID         uint32         `gorm:"index" json:"product_id"`
	MaxSupply         ctypes.BigInt  `gorm:"type:text" json:"max_supply"`
	ClosingTimestamp  int64          `json:"closing_timestamp"`
	DeliveryTimestamp *int64         `json:"delivery_timestamp"`
	StartingPrice     ctypes.BigInt  `gorm:"type:text" json:"starting_price"`
	IsEnabled         bool           `json:"is_enabled"`
	ToBeDelivered     *ctypes.BigInt `gorm:"type:text" json:"to_be_delivered"`
	Details           string         `gorm:"type:text;default:'N/A'" json:"details"`
}

type PreMarketUserTable struct {
	BaseTable
	SubaccountId string        `gorm:"index" json:"subaccount_id"`
	ProductID    uint32        `gorm:"index" json:"product_id"`
	Amount       ctypes.BigInt `gorm:"type:text" json:"amount"`
	Fees         ctypes.BigInt `gorm:"type:text" json:"fees"`
	Price        ctypes.BigInt `gorm:"type:text" json:"price"`
	QuoteDelta   ctypes.BigInt `gorm:"type:text" json:"quote_delta"`
	IsBuy        bool          `gorm:"index" json:"is_buy"`
}

type PreMarketCandleTable struct {
	BaseTable
	ProductID  uint32 `gorm:"index:idx_product_time,priority:1" json:"product_id"`
	StartTime  int64  `gorm:"index:idx_product_time,priority:2" json:"start_time"` // Start time of the candle period
	EndTime    int64  `gorm:"index:idx_product_time,priority:3" json:"end_time"`   // End time of the candle period
	Interval   string `json:"interval"`                                            // Candle interval in seconds (e.g., 60 for 1min, 300 for 5min)
	OpenPrice  string `gorm:"type:text" json:"open_price"`                         // Opening price for the period
	HighPrice  string `gorm:"type:text" json:"high_price"`                         // Highest price during the period
	LowPrice   string `gorm:"type:text" json:"low_price"`                          // Lowest price during the period
	ClosePrice string `gorm:"type:text" json:"close_price"`                        // Closing price for the period
}

type SyntheticSpotUserTable struct {
	BaseTable
	SubaccountId string        `gorm:"index" json:"subaccount_id"`
	ProductID    uint32        `gorm:"index" json:"product_id"`
	Amount       ctypes.BigInt `gorm:"type:text" json:"amount"`
	Fees         ctypes.BigInt `gorm:"type:text" json:"fees"`
	Price        ctypes.BigInt `gorm:"type:text" json:"price"`
	QuoteDelta   ctypes.BigInt `gorm:"type:text" json:"quote_delta"`
	IsBuy        bool          `gorm:"index" json:"is_buy"`
}

type ProposalTable struct {
	BaseTable
	Name                 string `json:"name"`
	VotingStart          int64  `json:"voting_start"`
	VotingEnd            int64  `json:"voting_end"`
	Description          string `gorm:"type:text" json:"description"`
	Options              string `gorm:"type:text" json:"options"`
	ProposerSubaccountId string `json:"proposer_subaccount_id"`
	VotingResult         string `gorm:"type:text" json:"voting_result"`
	MinVotingPower       string `gorm:"type:text" json:"min_voting_power"`
	VotingIsActive       bool   `gorm:"index" json:"voting_is_active"`
}

type VoteTable struct {
	BaseTable
	SubaccountId string `gorm:"index;uniqueIndex:idx_subaccount_proposal,priority:1" json:"subaccount_id"`
	ProposalId   uint   `gorm:"index;uniqueIndex:idx_subaccount_proposal,priority:2" json:"proposal_id"`
	Option       string `json:"option"`
	VotingPower  string `gorm:"type:text" json:"voting_power"`
	Signature    string `json:"signature"`
}

type AffiliateTable struct {
	BaseTable
	SubaccountId string `gorm:"unique;index" json:"subaccount_id"`
	ReferralCode string `gorm:"index" json:"referral_code"`     // Referral code used by the user
	IsActive     bool   `gorm:"default:false" json:"is_active"` // Whether the affiliate is active
}

type AirdropAllocationTable struct {
	BaseTable
	Address    string        `gorm:"uniqueIndex;not null" json:"address"`
	Allocation ctypes.BigInt `gorm:"type:text;default:0" json:"allocation"`
	Data       string        `gorm:"type:jsonb" json:"data"` // JSON column containing airdrop data
}

type CantonPartyTable struct {
	BaseTable
	SubAccountId string `gorm:"uniqueIndex"` // SubaccountTable.ID (unique)
	PartyId      string // Canton party id
}

// NearAccountTable maps a NEAR account to its subaccount (Development.md §6.1).
// SubaccountId uses the existing "broker_addr20_n" format with addr20 = keccak256("near:"‖account)[12..].
type NearAccountTable struct {
	BaseTable
	AccountId    string `gorm:"uniqueIndex:uniq_near_account_broker" json:"account_id"`
	BrokerId     uint   `gorm:"uniqueIndex:uniq_near_account_broker" json:"broker_id"`
	SubaccountId string `gorm:"uniqueIndex" json:"subaccount_id"`
	Addr20       string `gorm:"index" json:"addr20"`
}

// IntentsTransferTable tracks NEAR funding flows (Development.md §7): 1Click quotes by deposit
// address, and direct NEAR USDC deposits by transaction hash. Key is unique, so every flow is
// processed at most once.
type IntentsTransferTable struct {
	BaseTable
	Key            string     `gorm:"uniqueIndex" json:"key"` // "1click:<depositAddress>" | "neartx:<hash>" | "withdraw:<id>"
	Direction      string     `gorm:"index" json:"direction"` // deposit | withdraw
	Kind           string     `json:"kind"`                   // 1click | direct
	SubaccountId   string     `gorm:"index" json:"subaccount_id"`
	NearAccountId  string     `gorm:"index" json:"near_account_id"`
	DepositAddress string     `gorm:"index" json:"deposit_address,omitempty"`
	DepositMemo    string     `json:"deposit_memo,omitempty"`
	Confidential   string     `json:"confidentiality,omitempty"`
	QuoteJson      string     `gorm:"type:text" json:"-"`
	Status         string     `gorm:"index" json:"status"`
	AmountUSDC     string     `json:"amount_usdc,omitempty"` // 6-decimal units actually received or sent
	NearTxHash     string     `json:"near_tx_hash,omitempty"`
	CreditedAt     *time.Time `json:"credited_at,omitempty"`
	LastError      string     `gorm:"type:text" json:"last_error,omitempty"`
}
