//! Stored types (Development.md §5.2). Product IDs: even = spot, odd = perp (behavior-spec §1).
use crate::fixed::*;
use crate::perp::PerpBalance;
use near_sdk::near;
use near_sdk::{env, AccountId};

pub const QUOTE_PRODUCT_ID: u32 = 4;

pub fn is_spot(pid: u32) -> bool {
    pid % 2 == 0
}

pub fn is_perp(pid: u32) -> bool {
    pid % 2 == 1
}

/// A subaccount's balances. Both vectors stay sorted by product id so storage is deterministic.
/// There is no `locked` field: open-order locks exist only off-chain, so every on-chain health
/// check treats locked as 0, which is never stricter than the engine (behavior-spec §3.2 step 5).
#[near(serializers = [borsh])]
#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct Subaccount {
    pub spots: Vec<(u32, i128)>,
    pub perps: Vec<(u32, PerpBalance)>,
}

impl Subaccount {
    pub fn spot(&self, pid: u32) -> i128 {
        assert_spot(pid);
        self.spots.iter().find(|(p, _)| *p == pid).map(|(_, b)| *b).unwrap_or(0)
    }

    pub fn set_spot(&mut self, pid: u32, balance: i128) {
        assert_spot(pid);
        match self.spots.binary_search_by_key(&pid, |(p, _)| *p) {
            Ok(i) => self.spots[i].1 = balance,
            Err(i) => self.spots.insert(i, (pid, balance)),
        }
    }

    /// Go deletes a spot entry when it is fully consumed by settlement (SettleLiqPnlUsingSpots).
    pub fn delete_spot(&mut self, pid: u32) {
        self.spots.retain(|(p, _)| *p != pid);
    }

    /// SpotBalance.UpdateBalance
    pub fn add_spot(&mut self, pid: u32, delta: i128) {
        let b = checked_add(self.spot(pid), delta);
        self.set_spot(pid, b);
    }

    pub fn perp(&self, pid: u32) -> PerpBalance {
        assert_perp(pid);
        self.perps.iter().find(|(p, _)| *p == pid).map(|(_, b)| *b).unwrap_or_default()
    }

    pub fn set_perp(&mut self, pid: u32, balance: PerpBalance) {
        assert_perp(pid);
        match self.perps.binary_search_by_key(&pid, |(p, _)| *p) {
            Ok(i) => self.perps[i].1 = balance,
            Err(i) => self.perps.insert(i, (pid, balance)),
        }
    }

    /// UpdatePerpBalance: applies the position change and books realised pnl into the quote.
    pub fn update_perp(&mut self, pid: u32, d_a: i128, d_q: i128, cum_now: i128) -> (i128, i128) {
        let mut pb = self.perp(pid);
        let (pnl, funding) = pb.update(d_a, d_q, cum_now);
        self.set_perp(pid, pb);
        self.add_spot(QUOTE_PRODUCT_ID, pnl);
        (pnl, funding)
    }

    pub fn has_open_perps(&self) -> bool {
        self.perps.iter().any(|(_, b)| b.amount != 0)
    }
}

fn assert_spot(pid: u32) {
    if !is_spot(pid) {
        env::panic_str("invalid spot product id");
    }
}

fn assert_perp(pid: u32) {
    if !is_perp(pid) {
        env::panic_str("invalid perp product id");
    }
}

/// Oracle price as last written by PERPTICK (§5.7).
#[near(serializers = [borsh])]
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub struct Price {
    pub price_x18: i128,
    /// block timestamp (seconds) of the last accepted update; 0 = never set
    pub updated_at: u64,
}

#[near(serializers = [borsh])]
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct SpotProduct {
    /// NEP-141 token that backs this spot; None for ledger-only spots (for example LogX before Phase 4)
    pub token: Option<AccountId>,
    pub decimals: u8,
    /// PRODUCT_MARKET_WEIGHTS != 0: counts toward equity and can settle negative pnl
    pub weighted: bool,
    /// flat fee per withdrawal, x18 (WITHDRAWAL_FEE_MAP)
    pub withdraw_fee_x18: i128,
    pub price: Price,
    /// max move per PERPTICK update in basis points; 0 = unlimited
    pub max_deviation_bps: u32,
}

#[near(serializers = [borsh])]
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct PerpProduct {
    pub imf_x18: i128,
    pub mmf_x18: i128,
    /// liquidation fee fraction, x18 (GetLiquidationFractionx18: 0.015 default)
    pub liq_frac_x18: i128,
    pub cum_funding_x18: i128,
    /// unix seconds of the last funding accrual; 0 = not started
    pub last_funding_time: u64,
    pub price: Price,
    pub max_deviation_bps: u32,
    /// circuit breaker (§5.7): while set, only risk-reducing trades are accepted
    pub halted: bool,
    pub open_interest_long: i128,
    pub open_interest_short: i128,
    /// largest absolute AMM position (base amount, x18); 0 = no cap. Mirrors the markets table's
    /// AmmMaxPositionx18 in the backend.
    pub amm_max_position_x18: i128,
}

impl PerpProduct {
    /// PerpetualMarket.AmmPositionAllowed: the AMM may always shrink; it may grow only to the cap.
    pub fn amm_position_allowed(&self, old: i128, new: i128) -> bool {
        self.amm_max_position_x18 == 0
            || new.unsigned_abs() <= self.amm_max_position_x18.unsigned_abs()
            || new.unsigned_abs() <= old.unsigned_abs()
    }

    /// UpdateLongShortPositions, in closed form.
    pub fn update_open_interest(&mut self, old_amount: i128, d_a: i128) {
        let new_amount = checked_add(old_amount, d_a);
        let long = |a: i128| a.max(0);
        let short = |a: i128| checked_neg(a.min(0));
        self.open_interest_long = checked_add(self.open_interest_long, checked_sub(long(new_amount), long(old_amount)));
        self.open_interest_short = checked_add(self.open_interest_short, checked_sub(short(new_amount), short(old_amount)));
    }
}

/// Per-module pause bits (§5.10).
pub mod pause {
    pub const TRADING: u8 = 1;
    pub const DEPOSITS: u8 = 2;
    pub const WITHDRAWALS: u8 = 4;
    pub const LIQUIDATIONS: u8 = 8;
    pub const ALL: u8 = 15;
}

/// LogX and staked LogX spot products (constants.utils.go LOGX / ST_LOGX).
pub const LOGX_PRODUCT_ID: u32 = 0;
/// The LogX rewards pool (LOGX_REWARDS_SUBACCOUNT_ID in constants.utils.go): reward and airdrop
/// claims are paid from it, so every claimed LogX is backed by tokens the DAO deposited
/// (ft_transfer_call msg {"system":"rewards"}). Broker 1, address 0x...0010, number 1.
pub const LOGX_REWARDS_SUBACCOUNT: [u8; 32] =
    [0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x10, 0, 0, 0, 0, 0, 1];
pub const STAKED_LOGX_PRODUCT_ID: u32 = 2;

/// Side products (behavior-spec §3.9, §3.10) share one config table keyed by (kind, product id).
pub mod side {
    pub const OPTIONS: u8 = 0;
    pub const PRE_MARKET: u8 = 1;
    pub const SYNTHETIC_SPOT: u8 = 2;
}

/// D-5: the sequencer computes payouts, fees and pool prices off-chain; these per-product caps bound
/// what it can submit.
#[near(serializers = [borsh, json])]
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct SideProduct {
    pub enabled: bool,
    /// options: max fee % of the stake; pools: max fee in basis points of the gross quote
    pub max_fee: u32,
    /// options only: max payout % of the stake (payout 180 = 1.8x)
    pub max_payout_pct: u32,
}

/// The pool / house subaccounts that are the counterparty of options, pre-market and synthetic
/// spot (OPTIONS_X_SUBACCOUNT_ID etc. in constants.utils.go).
#[near(serializers = [borsh])]
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub struct ProductAccounts {
    pub options_x: [u8; 32],
    pub options_fees: [u8; 32],
    pub pre_market_x: [u8; 32],
    pub pre_market_fees: [u8; 32],
    pub synthetic_x: [u8; 32],
    pub synthetic_fees: [u8; 32],
}

/// An open options bet (tx 24), removed when closed (tx 25).
#[near(serializers = [borsh])]
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct StoredBet {
    pub subaccount: [u8; 32],
    pub product_id: u32,
    /// signed stake in quote: Divx18(amount * entry price); the sign is the direction
    pub quote_delta: i128,
    pub entry_price_x18: i128,
    pub payout_pct: u32,
    pub fee_pct: u32,
}

/// D-5 / D-6: per-transaction caps on sequencer-computed LogX credits.
#[near(serializers = [borsh, json])]
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub struct ClaimLimits {
    pub max_logx_claim_x18: near_sdk::json_types::I128,
    pub max_reward_claim_x18: near_sdk::json_types::I128,
}
