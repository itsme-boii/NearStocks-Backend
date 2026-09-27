//! Perp position math (behavior-spec §3.1, §3.2). Ported line-for-line from
//! libs/subaccountTypes/balance.subaccountTypes.go and services/engine/placeOrder.engine.go.
use crate::fixed::*;
use near_sdk::near;

#[near(serializers = [borsh])]
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub struct PerpBalance {
    pub amount: i128,
    pub v_quote: i128,
    pub last_cum_funding: i128,
}

fn sign(v: i128) -> i32 {
    v.signum() as i32
}

impl PerpBalance {
    /// RealiseFundingFee: fee = Divx18(-(cumNow - last) * vQuote); positive = user pays.
    pub fn realise_funding_fee(&mut self, cum_now: i128) -> i128 {
        let diff = checked_sub(cum_now, self.last_cum_funding);
        let fee = narrow(divx18(-(wide(diff) * wide(self.v_quote))));
        self.last_cum_funding = cum_now;
        fee
    }

    /// UpdateBalance: returns (realised pnl incl. funding, funding fee). The caller adds pnl to quote.
    pub fn update(&mut self, d_a: i128, d_q: i128, cum_now: i128) -> (i128, i128) {
        let funding_fee = self.realise_funding_fee(cum_now);
        let mut pnl = checked_neg(funding_fee);

        if sign(self.amount) * sign(d_a) >= 0 {
            self.amount = checked_add(self.amount, d_a);
            self.v_quote = checked_add(self.v_quote, d_q);
            return (pnl, funding_fee);
        }

        let mut part1 = self.amount.unsigned_abs().min(d_a.unsigned_abs()) as i128;
        if d_a < 0 {
            part1 = -part1;
        }
        let d_q1 = mul_div(d_q, part1, d_a);
        let d_q2 = checked_sub(d_q, d_q1);
        let remove = checked_neg(mul_div(self.v_quote, part1, self.amount));

        pnl = checked_add(pnl, checked_add(d_q1, remove));
        self.amount = checked_add(self.amount, d_a);
        self.v_quote = checked_add(self.v_quote, checked_sub(d_q2, remove));
        (pnl, funding_fee)
    }
}

/// Signed deltas for one match. Each side's vQuote is computed from the *other* side's signed
/// amount (engine placeOrder.engine.go:381-382); price is the maker's price.
pub struct MatchDeltas {
    pub maker_a: i128,
    pub maker_q: i128,
    pub taker_a: i128,
    pub taker_q: i128,
}

pub fn match_deltas(matched: i128, price: i128, maker_is_buy: bool, taker_is_buy: bool) -> MatchDeltas {
    let with_sign = |buy: bool| if buy { matched } else { checked_neg(matched) };
    let maker_a = with_sign(maker_is_buy);
    let taker_a = with_sign(taker_is_buy);
    MatchDeltas { maker_a, maker_q: mul_divx18(taker_a, price), taker_a, taker_q: mul_divx18(maker_a, price) }
}

/// Per-broker trading-fee table (behavior-spec D-1). `factor` is in units of 1e-5 (so 60 = 0.06%),
/// matching GetBrokerFeeFactor; the owner sets it in contract config instead of it being hard-coded.
#[near(serializers = [borsh])]
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct FeeTable {
    pub default_factor: i128,
    pub per_broker: Vec<(u64, i128)>,
}

/// Hard cap so a misconfigured table can never charge more than 1% per trade.
pub const MAX_FEE_FACTOR: i128 = 1_000;

impl FeeTable {
    /// Today's production behavior (libs/cutils/common.utils.go:135): broker 2 pays 0.06%, others 0.
    pub fn legacy() -> Self {
        Self { default_factor: 0, per_broker: vec![(2, 60)] }
    }

    pub fn validate(&self) {
        let ok = |f: i128| (0..=MAX_FEE_FACTOR).contains(&f);
        if !ok(self.default_factor) || !self.per_broker.iter().all(|(_, f)| ok(*f)) {
            near_sdk::env::panic_str("fee factor out of range");
        }
    }

    pub fn factor(&self, broker_id: u64) -> i128 {
        self.per_broker.iter().find(|(b, _)| *b == broker_id).map(|(_, f)| *f).unwrap_or(self.default_factor)
    }
}

/// DeductTradingFee: fee = Divx18(|dQ| * factor * 1e13). Returns the positive fee to remove from quote.
pub fn trading_fee(d_q: i128, factor: i128) -> i128 {
    let fraction = factor * 10_000_000_000_000; // MulxCust(factor, 13)
    mul_divx18(d_q.checked_abs().unwrap_or_else(|| near_sdk::env::panic_str("i128 overflow")), fraction)
}
