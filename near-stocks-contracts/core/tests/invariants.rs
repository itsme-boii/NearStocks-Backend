//! Property tests for the invariants in Development.md §13.1 (behavior-spec R-3).
mod common;
use common::*;
use near_stocks_core::fixed::E18 as FE18;
use near_stocks_core::perp::{match_deltas, PerpBalance};
use near_stocks_core::risk;
use near_stocks_core::state::Subaccount;
use proptest::prelude::*;

fn amount() -> impl Strategy<Value = i128> {
    prop_oneof![1i128..1_000, 1i128..10i128.pow(18), 10i128.pow(15)..10i128.pow(22)]
}

fn price() -> impl Strategy<Value = i128> {
    prop_oneof![1i128..10i128.pow(18), 10i128.pow(17)..10i128.pow(24)]
}

proptest! {
    #![proptest_config(ProptestConfig::with_cases(2000))]

    /// A match never creates quote: the two vQuote deltas sum to 0 or -1 wei (Euclidean
    /// rounding of the negative side), and base deltas cancel exactly.
    #[test]
    fn match_never_creates_value(m in amount(), p in price(), maker_buys in any::<bool>()) {
        let d = match_deltas(m, p, maker_buys, !maker_buys);
        prop_assert_eq!(d.maker_a + d.taker_a, 0);
        let s = d.maker_q + d.taker_q;
        prop_assert!(s == 0 || s == -1, "sum {}", s);
    }

    /// Closing a position completely leaves no vQuote behind, for any history.
    #[test]
    fn full_close_leaves_no_residue(a in amount(), long in any::<bool>(), vq in -(10i128.pow(24))..10i128.pow(24), dq in -(10i128.pow(24))..10i128.pow(24), cum in -(10i128.pow(16))..10i128.pow(16)) {
        let a = if long { a } else { -a };
        let mut pb = PerpBalance { amount: a, v_quote: vq, last_cum_funding: 0 };
        pb.update(-a, dq, cum);
        prop_assert_eq!(pb.amount, 0);
        prop_assert_eq!(pb.v_quote, 0);
        prop_assert_eq!(pb.last_cum_funding, cum);
    }

    /// Partial updates preserve the average entry price's sign relationship: the pnl realised on a
    /// reduction plus the vQuote change equals the vQuote delta passed in (value moves, nothing leaks).
    #[test]
    fn update_conserves_quote(a in amount(), long in any::<bool>(), vq in -(10i128.pow(24))..10i128.pow(24), da in amount(), buy in any::<bool>(), dq in -(10i128.pow(24))..10i128.pow(24)) {
        let a = if long { a } else { -a };
        let da = if buy { da } else { -da };
        let mut pb = PerpBalance { amount: a, v_quote: vq, last_cum_funding: 0 };
        let (pnl, fee) = pb.update(da, dq, 0);
        prop_assert_eq!(fee, 0);
        prop_assert_eq!(pnl + (pb.v_quote - vq), dq);
    }

    /// Settlement moves value between spots. Go floors the spot deducted and then zeroes the debt
    /// (SettleLiqPnlUsingSpots), so the user can keep at most one wei of the consumed spot
    /// (behavior-spec R-6); it never gains more, and never loses more than one unit of rounding.
    #[test]
    fn settle_using_spots_is_value_preserving(quote in -(10i128.pow(22))..0, other in 0i128..10i128.pow(22), px in 10i128.pow(17)..(2 * 10i128.pow(18))) {
        let mut sub = Subaccount::default();
        sub.set_spot(4, 0);
        sub.set_spot(6, other);
        let before = ethnum::I256::from(quote) * ethnum::I256::from(FE18) + ethnum::I256::from(other) * ethnum::I256::from(px);
        let q = risk::settle_using_spots(&mut sub, ethnum::I256::from(quote) * ethnum::I256::from(FE18), &[4, 6], &|_| true, &|p| if p == 6 { Some(px) } else { Some(FE18) });
        let after = ethnum::I256::from(q) * ethnum::I256::from(FE18) + ethnum::I256::from(sub.spot(6)) * ethnum::I256::from(px);
        prop_assert!(after < before + ethnum::I256::from(px), "gained a full spot wei or more");
        prop_assert!(before - after < ethnum::I256::from(FE18), "lost a quote wei or more");
        prop_assert!(sub.spot(6) >= 0);
    }
}

/// Σ(quote + vQuote) over every subaccount, plus the fee subaccount.
fn system_quote(c: &Chain, subs: &[[u8; 32]]) -> i128 {
    subs.iter().map(|s| spot(c, s, USDC) + perp(c, s, BTC).1 + perp(c, s, ETH).1).sum::<i128>() + spot(c, &fee_sub(), USDC)
}

fn dump(c: &Chain, subs: &[[u8; 32]]) -> Vec<(i128, (i128, i128, i128), (i128, i128, i128))> {
    subs.iter().chain([fee_sub()].iter()).map(|s| (spot(c, s, USDC), perp(c, s, BTC), perp(c, s, ETH))).collect()
}

proptest! {
    #![proptest_config(ProptestConfig::with_cases(24))]

    /// Contract level, with funding off: across any sequence of matches, the system never gains
    /// quote (it loses at most 1 wei per match), n_submissions only moves forward by the number of
    /// accepted transactions, and a rejected batch leaves every balance exactly as it was.
    #[test]
    fn batches_conserve_value_and_rejections_change_nothing(trades in proptest::collection::vec((0usize..3, 1usize..3, any::<bool>(), 1i128..200, 0i128..2000, any::<bool>()), 1..8)) {
        let mut c = setup();
        let names = ["p0.near", "p1.near", "p2.near"];
        let keys: Vec<Key> = (0..3).map(|i| Key::new(40 + i as u8)).collect();
        for (i, n) in names.iter().enumerate() {
            register(&mut c, n, 0, &keys[i]);
            deposit(&mut c, n, 5_000);
        }
        let subs: Vec<[u8; 32]> = names.iter().map(|n| sub_of(n, 0)).collect();
        let mut expected_idx = 0;
        let mut matches = 0i128;
        let start = system_quote(&c, &subs);
        for (t, off, taker_buys, size, px_off, eth) in trades {
            let m = (t + off) % 3;
            let (pid, base) = if eth { (ETH, 2_500 * E18) } else { (BTC, 65_000 * E18) };
            let px = base + px_off * E18 / 7;
            let amt = size * E18 / 1_000 + 12_345;
            let signed = if taker_buys { amt } else { -amt };
            let before = dump(&c, &subs);
            let batch = Batch::new().matched(order(subs[t], &keys[t], pid, 0, signed, false), &keys[t], order(subs[m], &keys[m], pid, px, -signed, false), &keys[m], -signed);
            let ok = std::panic::catch_unwind(std::panic::AssertUnwindSafe(|| batch.submit(&mut c))).is_ok();
            if ok {
                expected_idx += 1;
                matches += 1;
            } else {
                prop_assert_eq!(dump(&c, &subs), before, "rejected batch changed state");
            }
            prop_assert_eq!(c.view(|c| c.n_submissions()), expected_idx);
        }
        let end = system_quote(&c, &subs);
        prop_assert!(end <= start && start - end <= matches, "start {} end {} matches {}", start, end, matches);
    }
}
