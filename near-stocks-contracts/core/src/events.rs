//! NEP-297 events (Development.md §5.9): `EVENT_JSON:{"standard":"near-stocks","version":"1.0.0",...}`.
//! Integers above 2^53 are strings; byte arrays are 0x-hex.
use near_sdk::env;
use near_sdk::serde_json::{json, Value};

pub const STANDARD: &str = "near-stocks";
pub const VERSION: &str = "1.0.0";

pub fn emit(event: &str, data: Value) {
    let log = json!({ "standard": STANDARD, "version": VERSION, "event": event, "data": [data] });
    env::log_str(&format!("EVENT_JSON:{log}"));
}

pub fn hex(b: &[u8]) -> String {
    const HEX: &[u8; 16] = b"0123456789abcdef";
    let mut s = String::with_capacity(2 + b.len() * 2);
    s.push_str("0x");
    for x in b {
        s.push(HEX[(x >> 4) as usize] as char);
        s.push(HEX[(x & 15) as usize] as char);
    }
    s
}

pub fn unhex(s: &str) -> Vec<u8> {
    let s = s.strip_prefix("0x").unwrap_or(s);
    if s.len() % 2 != 0 {
        env::panic_str("odd-length hex");
    }
    (0..s.len()).step_by(2).map(|i| u8::from_str_radix(&s[i..i + 2], 16).unwrap_or_else(|_| env::panic_str("invalid hex"))).collect()
}

pub fn unhex32(s: &str) -> [u8; 32] {
    unhex(s).try_into().unwrap_or_else(|_| env::panic_str("expected 32 bytes"))
}

pub fn unhex20(s: &str) -> [u8; 20] {
    unhex(s).try_into().unwrap_or_else(|_| env::panic_str("expected 20 bytes"))
}

// ---------------------------------------------------------------------------------------------
// Batch log. NEAR caps a receipt at 100 log entries and 16,384 log bytes in total
// (nearcore max_number_logs / max_total_log_length), so per-fill JSON events would limit a
// batch to ~13 matches. submit_transactions instead emits ONE `batch` event whose high-volume
// records are Borsh vectors, base64-encoded. Everything else (subaccounts, prices, amounts,
// digests, base/quote deltas) is re-derived by the indexer from the batch's own input.

use near_sdk::borsh::{BorshDeserialize, BorshSerialize};

/// One MATCH_ORDERS (104 bytes). Maker first, as applied.
#[derive(BorshSerialize, BorshDeserialize, Clone, Debug, PartialEq, Eq)]
#[borsh(crate = "near_sdk::borsh")]
pub struct FillLog {
    pub tx_idx: u64,
    pub maker_fee: i128,
    pub maker_pnl: i128,
    pub maker_funding: i128,
    pub taker_fee: i128,
    pub taker_pnl: i128,
    pub taker_funding: i128,
}

/// One LIQUIDATE_SUBACCOUNT (104 bytes).
#[derive(BorshSerialize, BorshDeserialize, Clone, Debug, PartialEq, Eq)]
#[borsh(crate = "near_sdk::borsh")]
pub struct LiquidationLog {
    pub tx_idx: u64,
    pub liquidation_fee: i128,
    pub trading_fee: i128,
    pub liquidatee_pnl: i128,
    pub liquidatee_funding: i128,
    pub liquidator_pnl: i128,
    pub liquidator_funding: i128,
}

/// Funding state after a PERPTICK accrual (28 bytes).
#[derive(BorshSerialize, BorshDeserialize, Clone, Debug, PartialEq, Eq)]
#[borsh(crate = "near_sdk::borsh")]
pub struct FundingLog {
    pub product_id: u32,
    pub cum_funding: i128,
    pub time: u64,
}

/// A subaccount whose negative quote was settled (SETTLE_USER_PNL) or socialised (64 bytes).
/// `insurance_paid` is 0 for SETTLE_USER_PNL.
#[derive(BorshSerialize, BorshDeserialize, Clone, Debug, PartialEq, Eq)]
#[borsh(crate = "near_sdk::borsh")]
pub struct SettleLog {
    pub subaccount: [u8; 32],
    pub quote_after: i128,
    pub insurance_paid: i128,
}

#[derive(Default)]
pub struct BatchLog {
    pub fills: Vec<FillLog>,
    pub liquidations: Vec<LiquidationLog>,
    pub funding: Vec<FundingLog>,
    pub settled: Vec<SettleLog>,
    pub options: Vec<OptionCloseLog>,
}

impl BatchLog {
    pub fn emit(self, start_idx: u64, count: usize) {
        let mut data = json!({ "start_idx": start_idx, "count": count });
        let mut put = |k: &str, bytes: Vec<u8>, n: usize| {
            if n > 0 {
                data[k] = near_sdk::serde_json::to_value(near_sdk::json_types::Base64VecU8(bytes)).unwrap();
            }
        };
        put("fills", near_sdk::borsh::to_vec(&self.fills).unwrap(), self.fills.len());
        put("liquidations", near_sdk::borsh::to_vec(&self.liquidations).unwrap(), self.liquidations.len());
        put("funding", near_sdk::borsh::to_vec(&self.funding).unwrap(), self.funding.len());
        put("settled", near_sdk::borsh::to_vec(&self.settled).unwrap(), self.settled.len());
        put("options", near_sdk::borsh::to_vec(&self.options).unwrap(), self.options.len());
        emit("batch", data);
    }
}

/// One CLOSE_OPTIONS_BET that settled a bet (41 bytes). Closing an unknown or closed bet is a
/// no-op and logs nothing.
#[derive(BorshSerialize, BorshDeserialize, Clone, Debug, PartialEq, Eq)]
#[borsh(crate = "near_sdk::borsh")]
pub struct OptionCloseLog {
    pub order_id: u64,
    pub won: bool,
    pub payout: i128,
    pub fee: i128,
}
