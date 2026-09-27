//! Batch transaction envelope: `u8 type ‖ borsh(payload)` (Development.md §5.4, behavior-spec §2).
//! Type IDs are the backend enum in contract/types/transaction.type.go.
use crate::eip712::Order;
use near_sdk::borsh::{BorshDeserialize, BorshSerialize};

pub const PERPTICK: u8 = 0;
pub const LIQUIDATE_SUBACCOUNT: u8 = 1;
pub const WITHDRAW_COLLATERAL: u8 = 3;
pub const MATCH_ORDERS: u8 = 5;
pub const SETTLE_USER_PNL: u8 = 19;
pub const SOCIALISE_SUBACCOUNT: u8 = 20;
pub const SET_NONCE: u8 = 21;
// Phase 4: 24-27. Dropped or reserved: 2, 4, 6-18, 22, 23.
// 13-16, 18, 22 (LogX withdraw/claim/stake/unstake/rewards) were never listed on this deployment
// and are deliberately not implemented here — this contract carries no LogX/reward pool at all.

/// PERPTICK (0). Funding and prices are separate lists so the batcher can send price-only ticks
/// (empty `rates`) as often as it likes. Rates are keyed by product id (H-2).
#[derive(BorshSerialize, BorshDeserialize, Clone, Debug, PartialEq, Eq)]
#[borsh(crate = "near_sdk::borsh")]
pub struct PerpTick {
    /// unix seconds, the funding cron's fundingTimestamp
    pub time: u64,
    pub rates: Vec<(u32, i128)>,
    pub prices: Vec<(u32, i128)>,
}

/// MATCH_ORDERS (5). sigs[i] = taker signature, sigs2[i] = maker signature.
/// Both orders carry signed amounts; `matched_amount` has the maker's sign (endpoint.contract.go).
#[derive(BorshSerialize, BorshDeserialize, Clone, Debug, PartialEq, Eq)]
#[borsh(crate = "near_sdk::borsh")]
pub struct MatchOrders {
    pub product_id: u32,
    pub taker: Order,
    pub maker: Order,
    pub matched_amount: i128,
}

/// LIQUIDATE_SUBACCOUNT (1). sigs[i] = liquidator signature. `amount` has the liquidatee's sign
/// (FinaliseLiquidationRequest.AmountX18); the match price is the liquidator's order price.
#[derive(BorshSerialize, BorshDeserialize, Clone, Debug, PartialEq, Eq)]
#[borsh(crate = "near_sdk::borsh")]
pub struct Liquidate {
    pub product_id: u32,
    /// oracle prices for this check (perp and spot ids), bounded against stored prices
    pub prices: Vec<(u32, i128)>,
    pub liquidator: Order,
    pub liquidatee: [u8; 32],
    pub amount: i128,
}

/// SETTLE_USER_PNL (19)
#[derive(BorshSerialize, BorshDeserialize, Clone, Debug, PartialEq, Eq)]
#[borsh(crate = "near_sdk::borsh")]
pub struct SettleUserPnl {
    pub subaccounts: Vec<[u8; 32]>,
    pub spot_prices: Vec<(u32, i128)>,
}

/// SOCIALISE_SUBACCOUNT (20)
#[derive(BorshSerialize, BorshDeserialize, Clone, Debug, PartialEq, Eq)]
#[borsh(crate = "near_sdk::borsh")]
pub struct Socialise {
    pub subaccount: [u8; 32],
    pub spot_prices: Vec<(u32, i128)>,
}

/// SET_NONCE (21)
#[derive(BorshSerialize, BorshDeserialize, Clone, Debug, PartialEq, Eq)]
#[borsh(crate = "near_sdk::borsh")]
pub struct SetNonce {
    pub subaccount: [u8; 32],
    pub delta: u64,
}

// WITHDRAW_COLLATERAL (3) uses eip712::NearWithdraw as its payload; sigs[i] = session key signature.

pub const PLACE_OPTIONS_BET: u8 = 24;
pub const CLOSE_OPTIONS_BET: u8 = 25;
pub const PRE_MARKET_ORDER_REQUEST: u8 = 26;
pub const SYN_SPOT_ORDER_REQUEST: u8 = 27;

use crate::eip712::{OptionBet, PoolOrder};

/// PLACE_OPTIONS_BET (24): the user-signed bet plus what the backend fixed at placement.
#[derive(BorshSerialize, BorshDeserialize, Clone, Debug, PartialEq, Eq)]
#[borsh(crate = "near_sdk::borsh")]
pub struct PlaceOption {
    pub bet: OptionBet,
    pub order_id: u64,
    pub entry_price_x18: i128,
    pub payout_pct: u32,
    pub fee_pct: u32,
}

/// CLOSE_OPTIONS_BET (25), sequencer only.
#[derive(BorshSerialize, BorshDeserialize, Clone, Debug, PartialEq, Eq)]
#[borsh(crate = "near_sdk::borsh")]
pub struct CloseOption {
    pub order_id: u64,
    pub exit_price_x18: i128,
}

/// PRE_MARKET_ORDER_REQUEST (26) / SYN_SPOT_ORDER_REQUEST (27). `quote_delta` is base tokens out
/// on a buy and net quote out on a sell (behavior-spec §3.10); both are priced off-chain.
#[derive(BorshSerialize, BorshDeserialize, Clone, Debug, PartialEq, Eq)]
#[borsh(crate = "near_sdk::borsh")]
pub struct PoolTrade {
    pub order: PoolOrder,
    pub quote_delta: i128,
    pub fees: i128,
}
