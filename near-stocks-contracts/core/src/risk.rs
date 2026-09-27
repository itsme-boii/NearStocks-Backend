//! Equity, margin, health and settlement (behavior-spec §3.4–§3.8). Ported from
//! libs/subaccount/subaccount.go, balance.service.go and liquidation.service.go. Pure functions
//! over a `Market` so the same code runs in the contract, in unit tests and in the parity harness.
use crate::fixed::*;
use crate::state::{Subaccount, QUOTE_PRODUCT_ID};
use ethnum::I256;
use near_sdk::env;

pub trait Market {
    /// None means "no oracle price": the spot is skipped, as Go does.
    fn spot_price(&self, pid: u32) -> Option<i128>;
    /// PRODUCT_MARKET_WEIGHTS[pid] != 0
    fn spot_weighted(&self, pid: u32) -> bool;
    /// Must panic when the perp is unknown or has no usable price (Go log.Panicf).
    fn perp_price(&self, pid: u32) -> i128;
    /// (initial, maintenance) margin fractions, x18
    fn perp_margins(&self, pid: u32) -> (i128, i128);
    fn cum_funding(&self, pid: u32) -> i128;
}

/// GetTotalEquityx36 = spot + upnl - unrealised funding.
pub fn equity_x36(sub: &Subaccount, m: &dyn Market) -> I256 {
    let mut total = I256::ZERO;
    for (pid, bal) in &sub.spots {
        if !m.spot_weighted(*pid) {
            continue;
        }
        if let Some(p) = m.spot_price(*pid) {
            total += wide(*bal) * wide(p);
        }
    }
    for (pid, pb) in &sub.perps {
        let price = m.perp_price(*pid);
        total += wide(pb.amount) * wide(price) + wide(pb.v_quote) * wide(E18);
        // CalcFundingFeesx36 = -(vQuote * (cum - last)); equity subtracts it
        let diff = wide(m.cum_funding(*pid)) - wide(pb.last_cum_funding);
        total -= -(wide(pb.v_quote) * diff);
    }
    total
}

/// GetRequiredMarginValues -> (maintenance, initial), both x36.
pub fn margins_x36(sub: &Subaccount, m: &dyn Market) -> (I256, I256) {
    let (mut mm, mut im) = (I256::ZERO, I256::ZERO);
    for (pid, pb) in &sub.perps {
        let notional = (wide(pb.amount) * wide(m.perp_price(*pid))).abs();
        let (imf, mmf) = m.perp_margins(*pid);
        im += divx18(notional * wide(imf));
        mm += divx18(notional * wide(mmf));
    }
    (mm, im)
}

/// getSafetyMarginV2 (x18): Divx18(equity - locked - 0.96 * IM), with locked = 0 on-chain.
pub fn safety_x18(sub: &Subaccount, m: &dyn Market) -> I256 {
    let equity = equity_x36(sub, m);
    let (_, im) = margins_x36(sub, m);
    let im_96 = ediv(im * I256::from(96), I256::from(100));
    divx18(equity - im_96)
}

pub struct Health {
    pub below_initial: bool,
    pub below_maintenance: bool,
    pub requires_insurance: bool,
}

/// GetHealth (locked = 0).
pub fn health(sub: &Subaccount, m: &dyn Market) -> Health {
    let equity = equity_x36(sub, m);
    let (mm, im) = margins_x36(sub, m);
    Health {
        below_initial: im != I256::ZERO && equity < im,
        below_maintenance: mm != I256::ZERO && equity < mm,
        requires_insurance: im == I256::ZERO && equity < I256::ZERO,
    }
}

/// GetWithdrawableBalance for one product. `collateral_order` is ALL_COLLATERAL_SPOTS; Go walks
/// it in reverse and stops at the first product whose price is missing (error).
pub fn withdrawable_x18(sub: &Subaccount, m: &dyn Market, collateral_order: &[u32], pid: u32) -> i128 {
    let equity = equity_x36(sub, m);
    let (_, im) = margins_x36(sub, m);
    let mut avail = equity - im;
    for &p in collateral_order.iter().rev() {
        if avail <= I256::ZERO {
            break;
        }
        let price = m.spot_price(p).unwrap_or_else(|| env::panic_str("price not found for collateral"));
        let bal = sub.spot(p);
        if bal <= 0 {
            continue;
        }
        let value = wide(bal) * wide(price);
        let allowed = if avail >= value {
            avail -= value;
            bal
        } else {
            let free = narrow(ediv(avail, wide(price)));
            avail -= wide(free) * wide(price);
            free
        };
        if p == pid {
            return allowed;
        }
    }
    0
}

/// SettleLiqPnlUsingSpots. `pnl_x36` is usually negative; spots are consumed in `spot_order`
/// (ALL_SPOTS_ON_CONTRACT) and whatever is left lands in the quote. Returns the new quote balance.
pub fn settle_using_spots(
    sub: &mut Subaccount,
    mut pnl_x36: I256,
    spot_order: &[u32],
    weighted: &dyn Fn(u32) -> bool,
    price_of: &dyn Fn(u32) -> Option<i128>,
) -> i128 {
    for &pid in spot_order {
        if pnl_x36 >= I256::ZERO {
            break;
        }
        let bal = sub.spot(pid);
        let price = match price_of(pid) {
            Some(p) if p != 0 => p,
            _ => continue,
        };
        if !weighted(pid) || bal <= 0 {
            continue;
        }
        let value = wide(bal) * wide(price);
        if value.abs() > pnl_x36.abs() {
            let deduct = narrow(ediv(-pnl_x36, wide(price)));
            pnl_x36 = I256::ZERO;
            sub.add_spot(pid, checked_neg(deduct));
        } else {
            pnl_x36 += value;
            sub.delete_spot(pid);
        }
    }
    sub.add_spot(QUOTE_PRODUCT_ID, narrow(divx18(pnl_x36)));
    sub.spot(QUOTE_PRODUCT_ID)
}

/// GetAvailableMargin (x18): Divx18(equity - IM - locked), locked = 0 on-chain. Options, pre-market
/// and synthetic-spot buys require this to cover the quote they spend.
pub fn available_margin_x18(sub: &Subaccount, m: &dyn Market) -> i128 {
    let (_, im) = margins_x36(sub, m);
    narrow(divx18(equity_x36(sub, m) - im))
}
