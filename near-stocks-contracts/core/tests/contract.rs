//! Contract unit tests: every Phase 3 transaction handler, its guards and its failure paths.
//! Each call runs with receipt semantics (a panic reverts it, see common::exec). Numbers are
//! worked by hand in the comments; the Go-parity replay is in tests/parity.rs.
mod common;
use common::*;
use near_sdk::json_types::{I128, U128};
use near_sdk::{NearToken, PromiseOrValue, PromiseResult};
use near_stocks_core::eip712::{NearWithdraw, Order};
use near_stocks_core::events::hex;
use near_stocks_core::{tx, Limits, MigrationEntry, PerpConfig, SpotConfig};

const P65K: i128 = 65_000 * E18;
const P60K: i128 = 60_000 * E18;

fn two_traders(alice_usdc: u128, bob_usdc: u128) -> (Chain, Key, Key) {
    let mut c = setup();
    let (ka, kb) = (Key::new(1), Key::new(2));
    register(&mut c, "alice.near", 0, &ka);
    register(&mut c, "bob.near", 0, &kb);
    deposit(&mut c, "alice.near", alice_usdc);
    deposit(&mut c, "bob.near", bob_usdc);
    (c, ka, kb)
}

fn alice() -> [u8; 32] {
    sub_of("alice.near", 0)
}
fn bob() -> [u8; 32] {
    sub_of("bob.near", 0)
}

/// alice (taker) trades `amt` BTC (positive = buy) against bob (maker) at `price`
fn alice_trades(c: &mut Chain, ka: &Key, kb: &Key, amt: i128, price: i128, taker_reduce: bool) {
    Batch::new().matched(order(alice(), ka, BTC, 0, amt, taker_reduce), ka, order(bob(), kb, BTC, price, -amt, false), kb, -amt).submit(c);
}

fn btc_state(c: &Chain) -> serde_json::Value {
    let p = c.view(|c| c.get_products());
    p["perps"].as_array().unwrap().iter().find(|x| x["product_id"] == BTC).unwrap().clone()
}

// ------------------------------------------------------------------ matching

#[test]
fn match_orders_updates_both_sides_fees_and_open_interest() {
    let (mut c, ka, kb) = two_traders(10_000, 10_000);
    alice_trades(&mut c, &ka, &kb, E18 / 10, P65K, false);
    let (fills, ..) = batch_log();
    assert_eq!(fills.len(), 1);
    assert_eq!((fills[0].maker_fee, fills[0].taker_fee), (39 * E18 / 10, 39 * E18 / 10));
    assert_eq!(fills[0].tx_idx, 0);
    assert_eq!(events("batch")[0]["count"], 1);

    // dQ_taker = Divx18(maker_a * p) = -6500; dQ_maker = +6500; fee = 6500 * 60e13 / 1e18 = 3.9
    assert_eq!(perp(&c, &alice(), BTC), (E18 / 10, -6_500 * E18, 0));
    assert_eq!(perp(&c, &bob(), BTC), (-E18 / 10, 6_500 * E18, 0));
    assert_eq!(spot(&c, &alice(), USDC), 10_000 * E18 - 39 * E18 / 10);
    assert_eq!(spot(&c, &bob(), USDC), 10_000 * E18 - 39 * E18 / 10);
    assert_eq!(spot(&c, &fee_sub(), USDC), 78 * E18 / 10, "D-7: fees land in the fee subaccount");
    let btc = btc_state(&c);
    assert_eq!(btc["open_interest_long"], (E18 / 10).to_string());
    assert_eq!(btc["open_interest_short"], (E18 / 10).to_string());
    assert_eq!(c.view(|c| c.n_submissions()), 1);
}

#[test]
fn partial_fills_accumulate_and_overfill_is_rejected() {
    let (mut c, ka, kb) = two_traders(10_000, 10_000);
    let taker = order(alice(), &ka, BTC, 0, E18 / 10, false);
    let maker = order(bob(), &kb, BTC, P65K, -E18 / 5, false);
    Batch::new().matched(taker.clone(), &ka, maker.clone(), &kb, -E18 / 20).submit(&mut c);
    Batch::new().matched(taker.clone(), &ka, maker.clone(), &kb, -E18 / 20).submit(&mut c);
    assert_eq!(c.view(|c| c.filled_amount(hex(&order_digest(&taker)))).0, E18 / 10);
    assert_eq!(c.view(|c| c.filled_amount(hex(&order_digest(&maker)))).0, -E18 / 10);
    // the taker order is full: replaying it fails, and the failed batch changes nothing
    expect_panic("order overfilled", || Batch::new().matched(taker.clone(), &ka, maker.clone(), &kb, -1).submit(&mut c));
    assert_eq!(c.view(|c| c.filled_amount(hex(&order_digest(&maker)))).0, -E18 / 10);
    assert_eq!(c.view(|c| c.n_submissions()), 2);
}

#[test]
fn a_failing_transaction_reverts_the_whole_batch() {
    let (mut c, ka, kb) = two_traders(10_000, 10_000);
    let good = order(alice(), &ka, BTC, 0, E18 / 10, false);
    let maker = order(bob(), &kb, BTC, P65K, -E18 / 10, false);
    let mut expired = good.clone();
    expired.expiration = NOW_MS - 1;
    let batch = Batch::new().matched(good, &ka, maker.clone(), &kb, -E18 / 10).matched(expired, &ka, maker, &kb, -E18 / 10);
    expect_panic("order expired", || batch.submit(&mut c));
    assert_eq!(perp(&c, &alice(), BTC), (0, 0, 0));
    assert_eq!(spot(&c, &alice(), USDC), 10_000 * E18);
    assert_eq!(c.view(|c| c.n_submissions()), 0);
}

#[test]
fn signature_and_session_key_checks() {
    let (mut c, ka, kb) = two_traders(10_000, 10_000);
    let intruder = Key::new(9);
    let t = order(alice(), &ka, BTC, 0, E18 / 10, false);
    let m = order(bob(), &kb, BTC, P65K, -E18 / 10, false);
    // order names alice's key but is signed by another key
    let mut batch = Batch::new().matched(t.clone(), &ka, m.clone(), &kb, -E18 / 10);
    batch.sigs[0] = intruder.sign(&order_digest(&t));
    expect_panic("signature does not match session key", || batch.submit(&mut c));
    // malformed signature
    let mut batch = Batch::new().matched(t.clone(), &ka, m.clone(), &kb, -E18 / 10);
    batch.sigs2[0].truncate(64);
    expect_panic("bad signature", || batch.submit(&mut c));
    // a key that signs for itself but was never registered for alice
    let t2 = order(alice(), &intruder, BTC, 0, E18 / 10, false);
    // the refusal names the offending subaccount (not the maker), so the batcher pauses only alice
    let needle = format!("session key not registered or expired for {}", hex(&alice()));
    expect_panic(&needle, || Batch::new().matched(t2, &intruder, m.clone(), &kb, -E18 / 10).submit(&mut c));
    // session key past its expiry
    let late = NOW_MS + 25 * 3600 * 1000;
    let mut t3 = t.clone();
    t3.expiration = EXPIRY;
    c.call(sequencer(), |c| {
        let _ = c.n_submissions();
    });
    expect_panic("session key not registered or expired", || {
        Batch::new()
            .tick(late / 1000, vec![], vec![(BTC, P65K), (USDC, E18)])
            .matched(t3, &ka, m.clone(), &kb, -E18 / 10)
            .submit_at(&mut c, late)
    });
    // revoked key
    let kh = ka.hex();
    c.call(acc("alice.near"), |c| c.revoke_session_key(0, kh));
    expect_panic("session key not registered", || Batch::new().matched(t, &ka, m, &kb, -E18 / 10).submit(&mut c));
}

#[test]
fn order_shape_checks() {
    let (mut c, ka, kb) = two_traders(10_000, 10_000);
    let (a, b) = (alice(), bob());
    let m = order(b, &kb, BTC, P65K, -E18 / 10, false);
    expect_panic("prices do not cross", || {
        Batch::new().matched(order(a, &ka, BTC, 64_000 * E18, E18 / 10, false), &ka, m.clone(), &kb, -E18 / 10).submit(&mut c)
    });
    expect_panic("opposite sides", || {
        Batch::new()
            .matched(order(a, &ka, BTC, 0, E18 / 10, false), &ka, order(b, &kb, BTC, P65K, E18 / 10, false), &kb, E18 / 10)
            .submit(&mut c)
    });
    expect_panic("maker's sign", || {
        Batch::new().matched(order(a, &ka, BTC, 0, E18 / 10, false), &ka, m.clone(), &kb, E18 / 10).submit(&mut c)
    });
    expect_panic("self-trade", || {
        Batch::new()
            .matched(order(a, &ka, BTC, 0, E18 / 10, false), &ka, order(a, &ka, BTC, P65K, -E18 / 10, false), &ka, -E18 / 10)
            .submit(&mut c)
    });
    expect_panic("order product mismatch", || {
        let t: Order = order(a, &ka, ETH, 0, E18 / 10, false);
        let mut batch = Batch::new().matched(t, &ka, m.clone(), &kb, -E18 / 10);
        // re-encode with the maker's product so the envelope says BTC while the taker signed ETH
        batch.txs[0] = common::env(
            tx::MATCH_ORDERS,
            &tx::MatchOrders {
                product_id: BTC,
                taker: order(a, &ka, ETH, 0, E18 / 10, false),
                maker: m.clone(),
                matched_amount: -E18 / 10,
            },
        );
        batch.submit(&mut c)
    });
    // orders may live at most 31 days (db.ORDER_EXPIRY_DURATION + 1 day), so fills can be pruned
    let mut far = order(a, &ka, BTC, 0, E18 / 10, false);
    far.expiration = NOW_MS + 32 * 24 * 3600 * 1000;
    expect_panic("too far in the future", || Batch::new().matched(far, &ka, m.clone(), &kb, -E18 / 10).submit(&mut c));
    expect_panic("unknown perp", || {
        Batch::new()
            .matched(order(a, &ka, 7, 0, E18 / 10, false), &ka, order(b, &kb, 7, P65K, -E18 / 10, false), &kb, -E18 / 10)
            .submit(&mut c)
    });
    // a sell taker with a limit above the bid does not cross either
    expect_panic("prices do not cross", || {
        Batch::new()
            .matched(order(a, &ka, BTC, 66_000 * E18, -E18 / 10, false), &ka, order(b, &kb, BTC, P65K, E18 / 10, false), &kb, E18 / 10)
            .submit(&mut c)
    });
}

#[test]
fn reduce_only_cannot_open_or_flip() {
    let (mut c, ka, kb) = two_traders(10_000, 10_000);
    expect_panic("reduce-only", || alice_trades(&mut c, &ka, &kb, E18 / 10, P65K, true));
    alice_trades(&mut c, &ka, &kb, E18 / 10, P65K, false);
    expect_panic("reduce-only", || alice_trades(&mut c, &ka, &kb, -E18 / 5, P65K, true));
    alice_trades(&mut c, &ka, &kb, -E18 / 10, P65K, true);
    assert_eq!(perp(&c, &alice(), BTC), (0, 0, 0));
}

#[test]
fn health_blocks_risk_increase_but_allows_reduction_when_underwater() {
    // 100 USDC cannot carry 0.1 BTC (IM 650)
    let (mut small, ka2, kb2) = two_traders(100, 100_000);
    expect_panic("unhealthy trade", || alice_trades(&mut small, &ka2, &kb2, E18 / 10, P65K, false));

    // 700 USDC can: safety = 696.1 - 0.96*650 = 72.1
    let (mut c, ka, kb) = two_traders(700, 100_000);
    alice_trades(&mut c, &ka, &kb, E18 / 10, P65K, false);
    // BTC -> 60k: equity 196.1, IM 600, safety < 0
    Batch::new().tick(NOW_MS / 1000, vec![], vec![(BTC, P60K)]).submit(&mut c);
    expect_panic("unhealthy trade", || alice_trades(&mut c, &ka, &kb, E18 / 100, P60K, false));
    // selling half raises safety from -379.9 to -93.7, so it is accepted
    alice_trades(&mut c, &ka, &kb, -E18 / 20, P60K, false);
    assert_eq!(perp(&c, &alice(), BTC).0, E18 / 20);
    // realised -250 and fee 1.8: 696.1 - 251.8
    assert_eq!(spot(&c, &alice(), USDC), 4443 * E18 / 10);
}

#[test]
fn amm_pays_no_fee_and_skips_health() {
    let mut c = setup();
    let (ka, kamm) = (Key::new(1), Key::new(3));
    let amm = amm_sub();
    register(&mut c, "alice.near", 0, &ka);
    deposit(&mut c, "alice.near", 10_000);
    let (ammh, kh) = (hex(&amm), kamm.hex());
    expect_panic("only owner", || c.call(acc("alice.near"), |c| c.register_system_session_key(ammh.clone(), kh.clone(), EXPIRY)));
    let ah = hex(&alice());
    expect_panic("not a system subaccount", || c.call(owner(), |c| c.register_system_session_key(ah.clone(), kh.clone(), EXPIRY)));
    c.call(owner(), |c| c.register_system_session_key(ammh.clone(), kh.clone(), EXPIRY));

    Batch::new()
        .matched(order(alice(), &ka, BTC, 0, E18, false), &ka, order(amm, &kamm, BTC, P65K, -E18, false), &kamm, -E18)
        .submit(&mut c);
    // the AMM, with zero collateral, takes a 1 BTC short and pays no fee; alice pays 39
    assert_eq!(perp(&c, &amm, BTC).0, -E18);
    assert_eq!(spot(&c, &amm, USDC), 0);
    assert_eq!(spot(&c, &alice(), USDC), 10_000 * E18 - 39 * E18);
    assert_eq!(spot(&c, &fee_sub(), USDC), 39 * E18);
    let btc = btc_state(&c);
    assert_eq!(btc["open_interest_long"], E18.to_string());
    assert_eq!(btc["open_interest_short"], "0", "open interest ignores the AMM side");
}

// ------------------------------------------------------------------ PERPTICK

#[test]
fn funding_accrues_per_second_and_is_realised_on_trade() {
    let (mut c, ka, kb) = two_traders(10_000, 10_000);
    alice_trades(&mut c, &ka, &kb, E18 / 10, P65K, false);
    let t0 = NOW_MS / 1000;
    let rate = 1_000_000_000_000; // 1e-6 per second
    Batch::new().tick(t0, vec![(BTC, rate)], vec![]).submit(&mut c); // starts the clock, accrues nothing
    assert_eq!(btc_state(&c)["cum_funding_x18"], "0");
    let later = NOW_MS + 3_600_000;
    Batch::new().tick(t0 + 3600, vec![(BTC, rate)], vec![(BTC, P65K), (USDC, E18)]).submit_at(&mut c, later);
    assert_eq!(batch_log().2[0].cum_funding, 3_600_000_000_000_000);
    assert_eq!(btc_state(&c)["cum_funding_x18"], "3600000000000000");
    // closing realises funding: the long pays 6500 * 0.0036 = 23.4, the short receives it
    Batch::new()
        .matched(order(alice(), &ka, BTC, 0, -E18 / 10, false), &ka, order(bob(), &kb, BTC, P65K, E18 / 10, false), &kb, E18 / 10)
        .submit_at(&mut c, later);
    let (fills, _, funding, _) = batch_log();
    assert_eq!(fills[0].taker_funding, 234 * E18 / 10, "alice (taker) pays");
    assert_eq!(fills[0].maker_funding, -234 * E18 / 10, "bob (maker) receives");
    assert!(funding.is_empty());
    assert_eq!(spot(&c, &alice(), USDC), 10_000 * E18 - 78 * E18 / 10 - 234 * E18 / 10);
    assert_eq!(spot(&c, &bob(), USDC), 10_000 * E18 - 78 * E18 / 10 + 234 * E18 / 10);
}

#[test]
fn funding_time_cannot_go_backwards_or_be_in_the_future() {
    let mut c = setup();
    let t0 = NOW_MS / 1000;
    Batch::new().tick(t0, vec![(BTC, 1)], vec![]).submit(&mut c);
    expect_panic("went backwards", || Batch::new().tick(t0 - 1, vec![(BTC, 1)], vec![]).submit(&mut c));
    expect_panic("in the future", || Batch::new().tick(t0 + 61, vec![(BTC, 1)], vec![]).submit(&mut c));
    expect_panic("unknown perp", || Batch::new().tick(t0, vec![(9, 1)], vec![]).submit(&mut c));
    expect_panic("price must be positive", || Batch::new().tick(t0, vec![], vec![(BTC, 0)]).submit(&mut c));
}

#[test]
fn price_jump_trips_circuit_breaker_and_blocks_opening_trades() {
    let (mut c, ka, kb) = two_traders(10_000, 10_000);
    alice_trades(&mut c, &ka, &kb, E18 / 10, P65K, false);
    // +23% is outside the 10% band: the price is kept and the product halts
    Batch::new().tick(NOW_MS / 1000, vec![], vec![(BTC, 80_000 * E18)]).submit(&mut c);
    assert_eq!(events("circuit_breaker").len(), 1);
    let btc = btc_state(&c);
    assert_eq!(btc["price_x18"], P65K.to_string());
    assert_eq!(btc["halted"], true);
    expect_panic("product halted", || alice_trades(&mut c, &ka, &kb, E18 / 10, P65K, false));
    // reducing is still allowed
    alice_trades(&mut c, &ka, &kb, -E18 / 20, P65K, false);
    // guardian may halt but not clear; owner clears by setting a price
    expect_panic("not allowed", || c.call(guardian(), |c| c.set_halted(BTC, false)));
    c.call(guardian(), |c| c.set_halted(ETH, true));
    c.call(owner(), |c| c.set_price(BTC, I128(80_000 * E18)));
    assert_eq!(btc_state(&c)["halted"], false);
    alice_trades(&mut c, &ka, &kb, E18 / 100, 80_000 * E18, false);
}

#[test]
fn stale_prices_stop_trading() {
    let (mut c, ka, kb) = two_traders(10_000, 10_000);
    let batch = Batch::new().matched(
        order(alice(), &ka, BTC, 0, E18 / 10, false),
        &ka,
        order(bob(), &kb, BTC, P65K, -E18 / 10, false),
        &kb,
        -E18 / 10,
    );
    expect_panic("stale price", || batch.submit_at(&mut c, NOW_MS + 301_000));
}

// ------------------------------------------------------------------ liquidation

fn underwater_alice() -> (Chain, Key, Key, Key) {
    let (mut c, ka, kb) = two_traders(700, 100_000);
    let kc = Key::new(3);
    register(&mut c, "carol.near", 0, &kc);
    deposit(&mut c, "carol.near", 100_000);
    alice_trades(&mut c, &ka, &kb, E18 / 10, P65K, false);
    Batch::new().tick(NOW_MS / 1000, vec![], vec![(BTC, P60K)]).submit(&mut c);
    (c, ka, kb, kc)
}

fn liquidation(liquidator: Order, k: &Key, liquidatee: [u8; 32], amount: i128, prices: Vec<(u32, i128)>) -> Batch {
    let sig = k.sign(&order_digest(&liquidator));
    Batch::new().push(
        common::env(tx::LIQUIDATE_SUBACCOUNT, &tx::Liquidate { product_id: BTC, prices, liquidator, liquidatee, amount }),
        sig,
        vec![],
    )
}

fn carol() -> [u8; 32] {
    sub_of("carol.near", 0)
}

#[test]
fn liquidation_closes_position_and_pays_insurance_and_fee() {
    let (mut c, _ka, _kb, kc) = underwater_alice();
    let fee_before = spot(&c, &fee_sub(), USDC);
    liquidation(order(carol(), &kc, BTC, P60K, E18 / 10, false), &kc, alice(), -E18 / 10, vec![(BTC, P60K), (USDC, E18)]).submit(&mut c);
    let (_, liqs, ..) = batch_log();
    assert_eq!((liqs[0].liquidation_fee, liqs[0].trading_fee, liqs[0].liquidatee_pnl), (90 * E18, 36 * E18 / 10, -500 * E18));
    // realised -500, liquidation fee 90, trade fee 3.6: quote 696.1 -> 102.5
    assert_eq!(perp(&c, &alice(), BTC), (0, 0, 0));
    assert_eq!(spot(&c, &alice(), USDC), 1025 * E18 / 10);
    assert_eq!(spot(&c, &insurance_sub(), USDC), 90 * E18, "D-2");
    assert_eq!(spot(&c, &fee_sub(), USDC) - fee_before, 36 * E18 / 10, "D-7");
    // carol takes the long at 60k with no fee
    assert_eq!(perp(&c, &carol(), BTC), (E18 / 10, -6_000 * E18, 0));
    assert_eq!(spot(&c, &carol(), USDC), 100_000 * E18);
}

#[test]
fn liquidation_guards() {
    let (mut c, _ka, _kb, kc) = underwater_alice();
    // bob (short, 100k collateral) is healthy
    expect_panic("not liquidatable", || {
        liquidation(order(carol(), &kc, BTC, P60K, -E18 / 10, false), &kc, bob(), E18 / 10, vec![]).submit(&mut c)
    });
    // cannot increase the liquidatee's position
    expect_panic("must reduce", || {
        liquidation(order(carol(), &kc, BTC, P60K, -E18 / 10, false), &kc, alice(), E18 / 10, vec![]).submit(&mut c)
    });
    // supplied price outside the 10% band of the stored 60k
    expect_panic("outside the allowed band", || {
        liquidation(order(carol(), &kc, BTC, P60K, E18 / 10, false), &kc, alice(), -E18 / 10, vec![(BTC, 40_000 * E18)]).submit(&mut c)
    });
    // the liquidator's signed order must be on the other side
    expect_panic("other side", || {
        liquidation(order(carol(), &kc, BTC, P60K, -E18 / 10, false), &kc, alice(), -E18 / 10, vec![]).submit(&mut c)
    });
    c.call(guardian(), |c| c.pause(8));
    expect_panic("paused", || liquidation(order(carol(), &kc, BTC, P60K, E18 / 10, false), &kc, alice(), -E18 / 10, vec![]).submit(&mut c));
}

// ------------------------------------------------------------------ settlement and insurance

fn seeded(entries: Vec<MigrationEntry>) -> Chain {
    let mut c = Chain::deploy(new_contract);
    c.call(owner(), move |c| {
        c.upsert_spot(USDC, spot_usdc());
        c.upsert_spot(6, SpotConfig { token: Some(acc("usdt.near")), ..spot_usdc() });
        c.upsert_perp(BTC, perp_cfg(P65K));
        c.set_product_order(vec![USDC, 6], vec![USDC, 6]);
        c.migrate_state(entries, vec![]);
        c.finish_migration(0);
    });
    c
}

fn entry(sub: [u8; 32], spots: Vec<(u32, i128)>) -> MigrationEntry {
    MigrationEntry {
        subaccount: hex(&sub),
        owner: None,
        nonce: 0,
        spots: spots.into_iter().map(|(p, v)| (p, I128(v))).collect(),
        perps: vec![],
    }
}

#[test]
fn settle_user_pnl_consumes_other_spots_in_order() {
    let u = sub_of("dave.near", 0);
    let mut c = seeded(vec![entry(u, vec![(USDC, -30 * E18), (6, 50 * E18)])]);
    let settle = tx::SettleUserPnl { subaccounts: vec![u, sub_of("nobody.near", 0)], spot_prices: vec![(6, E18)] };
    Batch::new().push(common::env(tx::SETTLE_USER_PNL, &settle), vec![], vec![]).submit(&mut c);
    let settled = batch_log().3;
    assert_eq!(settled.len(), 1, "non-negative subaccounts are skipped");
    assert_eq!((settled[0].subaccount, settled[0].quote_after), (u, 0));
    // 30 of the 50 USDT pays the negative quote
    assert_eq!(spot(&c, &u, USDC), 0);
    assert_eq!(spot(&c, &u, 6), 20 * E18);
}

#[test]
fn socialise_uses_insurance_and_fails_when_it_is_empty() {
    let u = sub_of("dave.near", 0);
    let mut c = seeded(vec![entry(u, vec![(USDC, -30 * E18), (6, 10 * E18)]), entry(insurance_sub(), vec![(USDC, 25 * E18)])]);
    let soc = tx::Socialise { subaccount: u, spot_prices: vec![(6, E18)] };
    Batch::new().push(common::env(tx::SOCIALISE_SUBACCOUNT, &soc), vec![], vec![]).submit(&mut c);
    assert_eq!(batch_log().3[0].insurance_paid, 20 * E18);
    // 10 USDT consumed, 20 from insurance
    assert_eq!(spot(&c, &u, USDC), 0);
    assert_eq!(spot(&c, &u, 6), 0);
    assert_eq!(spot(&c, &insurance_sub(), USDC), 5 * E18);

    let v = sub_of("erin.near", 0);
    let mut c = seeded(vec![entry(v, vec![(USDC, -30 * E18)]), entry(insurance_sub(), vec![(USDC, 25 * E18)])]);
    let soc = tx::Socialise { subaccount: v, spot_prices: vec![] };
    expect_panic("insurance is out of funds", || {
        Batch::new().push(common::env(tx::SOCIALISE_SUBACCOUNT, &soc), vec![], vec![]).submit(&mut c)
    });
    assert_eq!(spot(&c, &v, USDC), -30 * E18);
    let soc = tx::Socialise { subaccount: insurance_sub(), spot_prices: vec![] };
    expect_panic("insurance fund", || Batch::new().push(common::env(tx::SOCIALISE_SUBACCOUNT, &soc), vec![], vec![]).submit(&mut c));
}

#[test]
fn socialise_requires_closed_positions() {
    let (mut c, ka, kb) = two_traders(10_000, 10_000);
    alice_trades(&mut c, &ka, &kb, E18 / 10, P65K, false);
    let soc = tx::Socialise { subaccount: alice(), spot_prices: vec![] };
    expect_panic("open positions", || Batch::new().push(common::env(tx::SOCIALISE_SUBACCOUNT, &soc), vec![], vec![]).submit(&mut c));
}

// ------------------------------------------------------------------ deposits and withdrawals

fn ft(c: &mut Chain, token: &str, sender: &str, units: u128, msg: &str) -> PromiseOrValue<U128> {
    let msg = msg.to_string();
    c.call(acc(token), |c| c.ft_on_transfer(acc(sender), U128(units), msg))
}

fn refunded(r: PromiseOrValue<U128>) -> u128 {
    match r {
        PromiseOrValue::Value(v) => v.0,
        _ => panic!("unexpected promise"),
    }
}

#[test]
fn deposits_credit_x18_and_respect_msg_pause_and_minimum() {
    let mut c = setup();
    deposit(&mut c, "alice.near", 25);
    assert_eq!(events("deposit")[0]["amount_x18"], (25 * E18).to_string());
    assert_eq!(spot(&c, &alice(), USDC), 25 * E18);

    // 1Click-style: sender is intents.near, the beneficiary is in msg
    let r = ft(&mut c, "usdc.near", "intents.near", 7_000_000, r#"{"account_id":"bob.near","subaccount_number":1,"source":"1click"}"#);
    assert_eq!(refunded(r), 0);
    assert_eq!(events("deposit")[0]["source"], "1click");
    assert_eq!(spot(&c, &sub_of("bob.near", 1), USDC), 7 * E18);

    // below the 1 USDC minimum for a new subaccount: refunded; top-ups of any size are fine
    assert_eq!(refunded(ft(&mut c, "usdc.near", "carol.near", 500_000, "")), 500_000);
    assert_eq!(refunded(ft(&mut c, "usdc.near", "alice.near", 1, r#"{"subaccount_number":0}"#)), 0);
    assert_eq!(spot(&c, &alice(), USDC), 25 * E18 + 1_000_000_000_000);
    // an empty msg credits the sender's subaccount 1, the one the app trades in (`2_<addr>_1`)
    assert_eq!(refunded(ft(&mut c, "usdc.near", "dave.near", 2_000_000, "")), 0);
    assert_eq!(spot(&c, &sub_of("dave.near", 1), USDC), 2 * E18);

    // unknown tokens panic (the FT contract then refunds) and bad msg panics
    expect_panic("token not accepted", || {
        let _ = ft(&mut c, "fake.near", "alice.near", 1, "");
    });
    expect_panic("invalid msg", || {
        let _ = ft(&mut c, "usdc.near", "alice.near", 1, "{not json");
    });
    expect_panic("invalid account_id", || {
        let _ = ft(&mut c, "usdc.near", "alice.near", 1, r#"{"account_id":"BAD ID"}"#);
    });

    c.call(guardian(), |c| c.pause(2));
    assert_eq!(refunded(ft(&mut c, "usdc.near", "alice.near", 9_000_000, "")), 9_000_000, "paused deposits are refunded");
}

#[test]
fn deposit_cannot_hijack_another_owners_subaccount() {
    let mut c = setup();
    let k = Key::new(1);
    register(&mut c, "alice.near", 0, &k);
    // bob.near's subaccount id can only ever bind to bob.near, because the id is derived from it;
    // a deposit naming alice credits alice's own subaccount
    let r = ft(&mut c, "usdc.near", "bob.near", 5_000_000, r#"{"account_id":"alice.near"}"#);
    assert_eq!(refunded(r), 0);
    assert_eq!(c.view(|c| c.get_subaccount(hex(&alice())))["owner"], "alice.near");
}

fn withdraw_req(sub: [u8; 32], key: &Key, amount: u128, nonce: u128, receiver: &str) -> NearWithdraw {
    NearWithdraw { subaccount: sub, session_key: key.addr, product_id: USDC, amount, nonce, receiver: receiver.into() }
}

const X18: u128 = E18 as u128;

#[test]
fn withdrawal_debits_and_callback_settles_fee_or_recredits() {
    let (mut c, ka, _kb) = two_traders(1_000, 10);
    // 100.25 USDC: fee 0.5, payout 99.75 USDC = 99_750_000 units
    Batch::new().withdraw(withdraw_req(alice(), &ka, 100_250_000_000_000_000_000, 0, "alice.near"), &ka).submit(&mut c);
    let ev = events("withdraw_pending");
    assert_eq!(ev[0]["payout"], "99750000");
    assert_eq!(ev[0]["fee"], (E18 / 2).to_string());
    let receipts = near_sdk::test_utils::get_created_receipts();
    assert_eq!(receipts.len(), 2, "ft_transfer and its callback");
    assert_eq!(receipts[0].receiver_id.as_str(), "usdc.near");
    assert_eq!(spot(&c, &alice(), USDC), 1_000 * E18 - 100_250_000_000_000_000_000);
    assert_eq!(c.view(|c| c.get_nonce(hex(&alice()))), 1);

    let ah = hex(&alice());
    let ok = c.callback(PromiseResult::Successful(vec![]), |c| {
        c.on_withdraw_complete(ah.clone(), USDC, U128(100_250_000_000_000_000_000), U128(X18 / 2), U128(99_750_000), acc("alice.near"), 1)
    });
    assert!(ok);
    assert_eq!(spot(&c, &fee_sub(), USDC), E18 / 2);

    // a failed transfer puts the whole debit back and pays no fee
    Batch::new().withdraw(withdraw_req(alice(), &ka, 10 * X18, 1, "alice.near"), &ka).submit(&mut c);
    let before = spot(&c, &alice(), USDC);
    let ok = c.callback(PromiseResult::Failed, |c| {
        c.on_withdraw_complete(ah.clone(), USDC, U128(10 * X18), U128(X18 / 2), U128(9_500_000), acc("alice.near"), 2)
    });
    assert!(!ok);
    assert_eq!(events("withdraw_failed").len(), 1);
    assert_eq!(spot(&c, &alice(), USDC), before + 10 * E18);
    assert_eq!(spot(&c, &fee_sub(), USDC), E18 / 2);
    // #[private] lives in the wasm entry wrapper; tests/sandbox.rs checks it on a real node
}

#[test]
fn withdrawal_guards() {
    let (mut c, ka, kb) = two_traders(1_000, 100_000);
    let a = alice();
    let w = |amount: u128, nonce: u128, receiver: &str| withdraw_req(a, &ka, amount, nonce, receiver);
    expect_panic("bad nonce", || Batch::new().withdraw(w(10 * X18, 5, "alice.near"), &ka).submit(&mut c));
    expect_panic("exceeds withdrawable", || Batch::new().withdraw(w(1_001 * X18, 0, "alice.near"), &ka).submit(&mut c));
    expect_panic("withdrawal fee", || Batch::new().withdraw(w(X18 / 2, 0, "alice.near"), &ka).submit(&mut c));
    expect_panic("invalid receiver", || Batch::new().withdraw(w(10 * X18, 0, "Not An Account"), &ka).submit(&mut c));
    // a replayed signature (same nonce) fails the second time
    Batch::new().withdraw(w(10 * X18, 0, "alice.near"), &ka).submit(&mut c);
    expect_panic("bad nonce", || Batch::new().withdraw(w(10 * X18, 0, "alice.near"), &ka).submit(&mut c));
    // bob's key cannot withdraw from alice
    expect_panic("session key not registered", || Batch::new().withdraw(withdraw_req(a, &kb, 10 * X18, 1, "bob.near"), &kb).submit(&mut c));
    // the signature binds the receiver: changing it after signing breaks the signature
    let mut batch = Batch::new().withdraw(w(10 * X18, 1, "alice.near"), &ka);
    batch.txs[0] = common::env(tx::WITHDRAW_COLLATERAL, &w(10 * X18, 1, "mallory.near"));
    expect_panic("signature does not match", || batch.submit(&mut c));
    // an open position locks initial margin: 0.1 BTC needs 650 of the remaining 986.1
    alice_trades(&mut c, &ka, &kb, E18 / 10, P65K, false);
    expect_panic("exceeds withdrawable", || Batch::new().withdraw(w(337 * X18, 1, "alice.near"), &ka).submit(&mut c));
    Batch::new().withdraw(w(336 * X18, 1, "alice.near"), &ka).submit(&mut c);
    c.call(guardian(), |c| c.pause(4));
    expect_panic("paused", || Batch::new().withdraw(w(X18, 2, "alice.near"), &ka).submit(&mut c));
}

// ------------------------------------------------------------------ endpoint and admin

#[test]
fn batch_envelope_rules() {
    let mut c = setup();
    let tick = || Batch::new().tick(NOW_MS / 1000, vec![], vec![]);
    let b = tick();
    expect_panic("only sequencer", || c.call(acc("mallory.near"), |c| c.submit_transactions(0, b.txs, b.sigs, b.sigs2)));
    let b = tick();
    expect_panic("expected idx 0, got 3", || c.call(sequencer(), |c| c.submit_transactions(3, b.txs, b.sigs, b.sigs2)));
    expect_panic("same non-zero length", || c.call(sequencer(), |c| c.submit_transactions(0, vec![vec![0]], vec![], vec![])));
    expect_panic("same non-zero length", || c.call(sequencer(), |c| c.submit_transactions(0, vec![], vec![], vec![])));
    for dropped in [2u8, 4, 6, 10, 11, 12, 17, 23, 99] {
        expect_panic("unsupported transaction type", || {
            c.call(sequencer(), |c| c.submit_transactions(0, vec![vec![dropped]], vec![vec![]], vec![vec![]]))
        });
    }
    expect_panic("invalid transaction payload", || {
        c.call(sequencer(), |c| c.submit_transactions(0, vec![vec![0, 1, 2]], vec![vec![]], vec![vec![]]))
    });
    expect_panic("empty transaction", || c.call(sequencer(), |c| c.submit_transactions(0, vec![vec![]], vec![vec![]], vec![vec![]])));
    tick().tick(NOW_MS / 1000, vec![], vec![]).submit(&mut c);
    assert_eq!(c.view(|c| c.n_submissions()), 2, "n_submissions advances by the number of transactions");
}

#[test]
fn prune_filled_removes_only_expired_orders() {
    let (mut c, ka, kb) = two_traders(10_000, 10_000);
    let t = order(alice(), &ka, BTC, 0, E18 / 10, false);
    let m = order(bob(), &kb, BTC, P65K, -E18 / 10, false);
    Batch::new().matched(t.clone(), &ka, m.clone(), &kb, -E18 / 10).submit(&mut c);
    let (td, md) = (hex(&order_digest(&t)), hex(&order_digest(&m)));
    expect_panic("only sequencer", || {
        c.call(acc("alice.near"), |c| c.prune_filled(vec![td.clone()]));
    });
    // not yet expired: kept
    assert_eq!(c.call(sequencer(), |c| c.prune_filled(vec![td.clone(), md.clone()])), 0);
    // after expiry both go, and the storage is released
    let after = t.expiration.max(m.expiration) + 1;
    assert_eq!(c.call_with(sequencer(), NearToken::from_yoctonear(0), after, |c| c.prune_filled(vec![td.clone(), md.clone()])), 2);
    assert_eq!(c.view(|c| c.filled_amount(td.clone())).0, 0);
}

#[test]
fn set_nonce_adds_delta() {
    let mut c = setup();
    Batch::new().push(common::env(tx::SET_NONCE, &tx::SetNonce { subaccount: alice(), delta: 3 }), vec![], vec![]).submit(&mut c);
    assert_eq!(c.view(|c| c.get_nonce(hex(&alice()))), 3);
    expect_panic("delta must be positive", || {
        Batch::new().push(common::env(tx::SET_NONCE, &tx::SetNonce { subaccount: alice(), delta: 0 }), vec![], vec![]).submit(&mut c)
    });
}

#[test]
fn session_key_registration_rules() {
    let mut c = setup();
    let k = Key::new(1);
    let kh = k.hex();
    let deposit = NearToken::from_millinear(100);
    expect_panic("maximum session length", || {
        c.call_with(acc("alice.near"), deposit, NOW_MS, |c| c.register_session_key(0, kh.clone(), NOW_MS + 8 * 24 * 3600 * 1000))
    });
    expect_panic("maximum session length", || {
        c.call_with(acc("alice.near"), deposit, NOW_MS, |c| c.register_session_key(0, kh.clone(), NOW_MS))
    });
    expect_panic("for storage", || c.call(acc("alice.near"), |c| c.register_session_key(0, kh.clone(), NOW_MS + 1000)));
    register(&mut c, "alice.near", 0, &k);
    assert_eq!(events("session_key_created")[0]["owner"], "alice.near");
    let refund = near_sdk::test_utils::get_created_receipts();
    assert_eq!(refund.len(), 1, "unused storage deposit is refunded");
    let ah = hex(&alice());
    assert_eq!(c.view(|c| c.session_key_expiry(ah.clone(), kh.clone())), NOW_MS + 24 * 3600 * 1000);
    assert_eq!(c.view(|c| c.get_subaccount(ah.clone()))["owner"], "alice.near");
    assert_eq!(c.view(|c| c.subaccount_id_for(acc("alice.near"), 0)), ah);
}

#[test]
fn admin_permissions_and_latches() {
    let mut c = setup();
    c.call(guardian(), |c| c.pause(1));
    expect_panic("only owner", || c.call(guardian(), |c| c.unpause(1)));
    expect_panic("only owner", || c.call(guardian(), |c| c.set_fee_table(I128(0), vec![])));
    expect_panic("only guardian or owner", || c.call(acc("mallory.near"), |c| c.pause(1)));
    expect_panic("only owner", || c.call(acc("mallory.near"), |c| c.set_roles(Some(acc("mallory.near")), None, None)));
    expect_panic("only owner", || c.call(acc("mallory.near"), |c| c.upgrade().detach()));
    c.call(owner(), |c| c.unpause(1));
    expect_panic("fee factor out of range", || c.call(owner(), |c| c.set_fee_table(I128(0), vec![(2, I128(1_001))])));
    c.call(owner(), |c| c.set_fee_table(I128(10), vec![(2, I128(60))]));
    expect_panic("migration already finished", || c.call(owner(), |c| c.migrate_state(vec![], vec![])));
    expect_panic("migration already finished", || c.call(owner(), |c| c.finish_migration(0)));
    expect_panic("invalid margin config", || c.call(owner(), |c| c.upsert_perp(5, PerpConfig { mmf_x18: I128(2 * E18), ..perp_cfg(E18) })));
    expect_panic("spot ids are even", || c.call(owner(), |c| c.upsert_spot(5, spot_usdc())));
    expect_panic("already mapped", || c.call(owner(), |c| c.upsert_spot(8, spot_usdc())));
    expect_panic("unknown spot in order list", || c.call(owner(), |c| c.set_product_order(vec![USDC, 8], vec![USDC])));
    c.call(owner(), |c| {
        c.set_limits(Limits {
            price_max_age_sec: 60,
            max_session_ttl_ms: 1000,
            min_new_deposit_x18: I128(0),
            max_order_ttl_ms: 31 * 24 * 3600 * 1000,
        })
    });
    assert_eq!(c.view(|c| c.get_config())["price_max_age_sec"], 60);
    // upserting a perp keeps its funding state
    c.call(owner(), |c| c.upsert_perp(BTC, perp_cfg(0)));
    assert_eq!(btc_state(&c)["price_x18"], (65_000 * E18).to_string());

    // a fresh contract refuses batches and deposits until migration is finished
    let mut fresh = Chain::deploy(new_contract);
    fresh.call(owner(), |c| c.upsert_spot(USDC, spot_usdc()));
    expect_panic("migration not finished", || {
        fresh.call(sequencer(), |c| c.submit_transactions(0, vec![vec![0]], vec![vec![]], vec![vec![]]))
    });
    assert_eq!(refunded(ft(&mut fresh, "usdc.near", "a.near", 5_000_000, "")), 5_000_000);
}

#[test]
fn migration_loads_balances_and_perp_state() {
    let u = sub_of("dave.near", 0);
    let mut c = Chain::deploy(new_contract);
    c.call(owner(), |c| {
        c.upsert_spot(USDC, spot_usdc());
        c.upsert_perp(BTC, perp_cfg(P65K));
        c.migrate_state(
            vec![MigrationEntry {
                subaccount: hex(&u),
                owner: Some(acc("dave.near")),
                nonce: 7,
                spots: vec![(USDC, I128(123 * E18))],
                perps: vec![(BTC, I128(E18), I128(-60_000 * E18), I128(42))],
            }],
            vec![near_stocks_core::PerpState {
                product_id: BTC,
                cum_funding_x18: I128(42),
                last_funding_time: 1000,
                open_interest_long: I128(E18),
                open_interest_short: I128(0),
            }],
        );
        c.finish_migration(5_000);
    });
    assert_eq!(c.view(|c| c.n_submissions()), 5_000, "continues the old batch index");
    assert_eq!(spot(&c, &u, USDC), 123 * E18);
    assert_eq!(perp(&c, &u, BTC), (E18, -60_000 * E18, 42));
    assert_eq!(c.view(|c| c.get_nonce(hex(&u))), 7);
    assert_eq!(btc_state(&c)["cum_funding_x18"], "42");
}

// ------------------------------------------------------------------ D-7 sweep, R-5, AMM cap

#[test]
fn dao_sweeps_fees_with_recredit_on_failure() {
    let (mut c, ka, kb) = two_traders(10_000, 10_000);
    alice_trades(&mut c, &ka, &kb, E18 / 10, P65K, false); // 7.8 USDC of fees
    let one = NearToken::from_yoctonear(1);
    expect_panic("only owner", || {
        let _ = c.call_with(acc("mallory.near"), one, NOW_MS, |c| c.sweep_fees(USDC, U128(X18), acc("treasury.near")));
    });
    expect_panic("exactly 1 yoctoNEAR", || {
        let _ = c.call(owner(), |c| c.sweep_fees(USDC, U128(X18), acc("treasury.near")));
    });
    expect_panic("fee balance too low", || {
        let _ = c.call_with(owner(), one, NOW_MS, |c| c.sweep_fees(USDC, U128(8 * X18), acc("treasury.near")));
    });
    // 5.0000001 USDC requested: 5 whole units' worth is sent, the sub-unit remainder stays
    c.call_with(owner(), one, NOW_MS, |c| c.sweep_fees(USDC, U128(5 * X18 + 100_000_000_000), acc("treasury.near")).detach());
    assert_eq!(events("fee_sweep_pending")[0]["payout"], "5000000");
    assert_eq!(spot(&c, &fee_sub(), USDC), 78 * E18 / 10 - 5 * E18);
    let fh = hex(&fee_sub());
    // transfer failed: the fee account gets the 5 back
    c.callback(PromiseResult::Failed, |c| {
        c.on_withdraw_complete(fh.clone(), USDC, U128(5 * X18), U128(0), U128(5_000_000), acc("treasury.near"), 1)
    });
    assert_eq!(spot(&c, &fee_sub(), USDC), 78 * E18 / 10);
    // transfer succeeded: nothing comes back
    c.call_with(owner(), one, NOW_MS, |c| c.sweep_fees(USDC, U128(5 * X18), acc("treasury.near")).detach());
    c.callback(PromiseResult::Successful(vec![]), |c| {
        c.on_withdraw_complete(fh.clone(), USDC, U128(5 * X18), U128(0), U128(5_000_000), acc("treasury.near"), 1)
    });
    assert_eq!(spot(&c, &fee_sub(), USDC), 28 * E18 / 10);
}

fn capped_amm(cap: i128) -> (Chain, Key, Key) {
    let mut c = setup();
    c.call(owner(), |c| c.upsert_perp(BTC, PerpConfig { amm_max_position_x18: I128(cap), ..perp_cfg(0) }));
    let (ka, kamm) = (Key::new(1), Key::new(3));
    register(&mut c, "alice.near", 0, &ka);
    deposit(&mut c, "alice.near", 100_000);
    let (ammh, kh) = (hex(&amm_sub()), kamm.hex());
    c.call(owner(), |c| c.register_system_session_key(ammh, kh, EXPIRY));
    (c, ka, kamm)
}

#[test]
fn amm_position_cap_blocks_growth_but_not_reduction() {
    let (mut c, ka, kamm) = capped_amm(E18 / 2);
    let amm = amm_sub();
    let trade = |c: &mut Chain, amt: i128| {
        Batch::new().matched(order(alice(), &ka, BTC, 0, amt, false), &ka, order(amm, &kamm, BTC, P65K, -amt, false), &kamm, -amt).submit(c)
    };
    trade(&mut c, 4 * E18 / 10); // AMM short 0.4
    expect_panic("AMM position cap", || trade(&mut c, 2 * E18 / 10)); // would be 0.6
    trade(&mut c, E18 / 10); // exactly at the 0.5 cap
    trade(&mut c, -3 * E18 / 10); // reducing is always allowed
    assert_eq!(perp(&c, &amm, BTC).0, -2 * E18 / 10);
    // a cap of 0 means no cap
    let (mut c2, ka2, kamm2) = capped_amm(0);
    Batch::new()
        .matched(order(alice(), &ka2, BTC, 0, E18, false), &ka2, order(amm, &kamm2, BTC, P65K, -E18, false), &kamm2, -E18)
        .submit(&mut c2);
    assert_eq!(perp(&c2, &amm, BTC).0, -E18);
}

#[test]
fn liquidator_must_be_healthy_and_amm_liquidator_respects_cap() {
    let (mut c, _ka, _kb, kc) = underwater_alice();
    // dave, with 100 USDC, cannot take alice's 0.1 BTC (IM 600)
    let kd = Key::new(4);
    register(&mut c, "dave.near", 0, &kd);
    deposit(&mut c, "dave.near", 100);
    let dave = sub_of("dave.near", 0);
    expect_panic("liquidator would be unhealthy", || {
        liquidation(order(dave, &kd, BTC, P60K, E18 / 10, false), &kd, alice(), -E18 / 10, vec![]).submit(&mut c)
    });
    // the AMM as liquidator is bound by its cap (0.05 < 0.1)
    let kamm = Key::new(5);
    let (ammh, kh) = (hex(&amm_sub()), kamm.hex());
    c.call(owner(), |c| c.register_system_session_key(ammh, kh, EXPIRY));
    c.call(owner(), |c| c.upsert_perp(BTC, PerpConfig { amm_max_position_x18: I128(E18 / 20), ..perp_cfg(0) }));
    expect_panic("AMM position cap", || {
        liquidation(order(amm_sub(), &kamm, BTC, P60K, E18 / 10, false), &kamm, alice(), -E18 / 10, vec![]).submit(&mut c)
    });
    // carol, well funded, can
    liquidation(order(carol(), &kc, BTC, P60K, E18 / 10, false), &kc, alice(), -E18 / 10, vec![]).submit(&mut c);
    assert_eq!(perp(&c, &alice(), BTC).0, 0);
}

// ------------------------------------------------------------------ S-6: trusted (1Click) deposits

#[test]
fn trusted_depositor_deposits_are_never_refunded() {
    let mut c = setup();
    c.call(owner(), |c| c.set_trusted_depositors(vec![acc("intents.near")]));
    let one_click = r#"{"account_id":"alice.near","subaccount_number":1,"source":"1click"}"#;

    // below the new-subaccount minimum: an untrusted sender is refunded, 1Click is credited
    assert_eq!(refunded(ft(&mut c, "usdc.near", "carol.near", 100_000, "")), 100_000);
    assert_eq!(refunded(ft(&mut c, "usdc.near", "intents.near", 100_000, one_click)), 0);
    assert_eq!(spot(&c, &sub_of("alice.near", 1), USDC), E18 / 10);

    // deposits paused: 1Click funds are held for alice, then released to her by anyone
    c.call(guardian(), |c| c.pause(2));
    assert_eq!(refunded(ft(&mut c, "usdc.near", "intents.near", 7_000_000, one_click)), 0);
    assert_eq!(events("deposit_held").len(), 1);
    assert_eq!(c.view(|c| c.get_held_deposit(acc("alice.near"), 1, USDC)).0, 7 * E18);
    assert_eq!(spot(&c, &sub_of("alice.near", 1), USDC), E18 / 10, "not credited while paused");
    expect_panic("deposits are closed", || {
        c.call(acc("anyone.near"), |c| c.release_held_deposit(acc("alice.near"), 1, USDC));
    });
    c.call(owner(), |c| c.unpause(2));
    c.call(acc("anyone.near"), |c| c.release_held_deposit(acc("alice.near"), 1, USDC));
    assert_eq!(events("deposit")[0]["source"], "held", "the indexer mirrors the release as a deposit");
    assert_eq!(spot(&c, &sub_of("alice.near", 1), USDC), E18 / 10 + 7 * E18);
    expect_panic("nothing held", || {
        c.call(acc("anyone.near"), |c| c.release_held_deposit(acc("alice.near"), 1, USDC));
    });

    // unreadable msg: parked as unclaimed, the DAO assigns it
    assert_eq!(refunded(ft(&mut c, "usdc.near", "intents.near", 3_000_000, "{broken")), 0);
    assert_eq!(c.view(|c| c.get_unclaimed(USDC)).0, 3_000_000);
    expect_panic("only owner", || c.call(acc("mallory.near"), |c| c.assign_unclaimed(USDC, U128(3_000_000), acc("mallory.near"), 1)));
    expect_panic("more than is unclaimed", || c.call(owner(), |c| c.assign_unclaimed(USDC, U128(3_000_001), acc("bob.near"), 1)));
    c.call(owner(), |c| c.assign_unclaimed(USDC, U128(3_000_000), acc("bob.near"), 1));
    assert_eq!(spot(&c, &sub_of("bob.near", 1), USDC), 3 * E18);
    assert_eq!(c.view(|c| c.get_unclaimed(USDC)).0, 0);

    // an untrusted sender with a bad msg still panics (its FT contract refunds it)
    expect_panic("invalid msg", || {
        let _ = ft(&mut c, "usdc.near", "carol.near", 1_000_000, "{broken");
    });
}

#[test]
fn only_the_owner_funds_system_subaccounts() {
    let mut c = setup();
    let dao = owner().to_string();
    assert_eq!(refunded(ft(&mut c, "usdc.near", &dao, 5_000_000, r#"{"system":"amm"}"#)), 0);
    assert_eq!(refunded(ft(&mut c, "usdc.near", &dao, 2_000_000, r#"{"system":"insurance"}"#)), 0);
    assert_eq!(spot(&c, &common::amm_sub(), USDC), 5 * E18);
    assert_eq!(spot(&c, &common::insurance_sub(), USDC), 2 * E18);
    let d = events("deposit");
    assert_eq!(d[0]["source"], "system", "the indexer credits it without linking the DAO to the subaccount");
    assert_eq!(d[0]["subaccount"], hex(&common::insurance_sub()));
    expect_panic("only the owner funds system subaccounts", || {
        let _ = ft(&mut c, "usdc.near", "mallory.near", 5_000_000, r#"{"system":"amm"}"#);
    });
    expect_panic("unknown system subaccount", || {
        let _ = ft(&mut c, "usdc.near", &dao, 5_000_000, r#"{"system":"fee"}"#);
    });
    c.call(guardian(), |c| c.pause(2));
    expect_panic("deposits are closed", || {
        let _ = ft(&mut c, "usdc.near", &dao, 5_000_000, r#"{"system":"amm"}"#);
    });
}

#[test]
fn prices_for_products_the_contract_does_not_list_are_ignored() {
    // the backend sends every collateral it knows, including legacy product 74
    let u = sub_of("dave.near", 0);
    let mut c = seeded(vec![entry(u, vec![(USDC, -30 * E18), (6, 50 * E18)])]);
    let settle = tx::SettleUserPnl { subaccounts: vec![u], spot_prices: vec![(6, E18), (74, E18), (999, 1)] };
    Batch::new().push(common::env(tx::SETTLE_USER_PNL, &settle), vec![], vec![]).submit(&mut c);
    assert_eq!(spot(&c, &u, 6), 20 * E18);
    // a listed product's price is still band-checked
    let bad = tx::SettleUserPnl { subaccounts: vec![u], spot_prices: vec![(6, 2 * E18), (74, E18)] };
    expect_panic("outside the allowed band", || Batch::new().push(common::env(tx::SETTLE_USER_PNL, &bad), vec![], vec![]).submit(&mut c));
}
